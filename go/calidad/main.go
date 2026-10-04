// Analisis de calidad del clustering y perfiles clinicos.
//
// ============================================================================
// PARA ANDRE - este programa responde dos preguntas que el informe todavia
// no contesta y que el caso de uso EXIGE:
//
//  1. Por que k = 5 y no otro valor?  -> metodo del codo
//  2. Que significa clinicamente cada cluster?  -> des-escalado + perfiles
//
// El esqueleto y el des-escalado ya estan resueltos. Lo que falta hacer
// esta marcado con  // TODO(Andre)
// ============================================================================
//
// Uso:
//
//	go run ./calidad -datos ../data/processed/features.bin -kmin 2 -kmax 10
package main

import (
	"flag"
	"fmt"
	"os"
	"sync"

	"pc2/kmeans"
)

// Parametros del escalado z-score aplicado en la PC1 (scripts/limpieza.py).
// Se usan para devolver los centroides a sus unidades reales:
//
//	valor_real = valor_escalado * desviacion + media
var (
	medias = [kmeans.D]float64{3294.8199, 49.4305, 38.8311, 28.2344}
	desvs  = [kmeans.D]float64{424.8124, 1.7781, 1.1894, 6.8913}
	unidad = [kmeans.D]string{"g", "cm", "sem", "anios"}
	nombre = [kmeans.D]string{"peso", "talla", "gestacion", "edad madre"}
)

// desescalar convierte un centroide de unidades z-score a unidades reales.
func desescalar(c []float64) [kmeans.D]float64 {
	var r [kmeans.D]float64
	for j := 0; j < kmeans.D; j++ {
		r[j] = c[j]*desvs[j] + medias[j]
	}
	return r
}

// correKMeans ejecuta el K-Means concurrente y devuelve los centroides
// finales, los conteos por cluster y la inercia (funcion objetivo MSSC).
// Reutiliza la arquitectura de canales verificada en Promela.
func correKMeans(ds *kmeans.Dataset, k, iter, W int, semilla uint64) ([]float64, []int, float64) {
	centroides := kmeans.InicializarCentroides(ds, k, semilla)

	inicios := make([]int, W+1)
	base, resto, acc := ds.N/W, ds.N%W, 0
	for w := 0; w < W; w++ {
		inicios[w] = acc
		t := base
		if w < resto {
			t++
		}
		acc += t
	}
	inicios[W] = ds.N

	proceed := make([]chan int, W)
	for i := range proceed {
		proceed[i] = make(chan int, 1)
	}
	done := make(chan int, W)

	sl := make([][]float64, W)
	cl := make([][]int, W)
	for w := 0; w < W; w++ {
		sl[w] = make([]float64, k*kmeans.D)
		cl[w] = make([]int, k)
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(W)

	for w := 0; w < W; w++ {
		go func(id, desde, hasta int) {
			defer wg.Done()
			for t := range proceed[id] {
				if t == -1 {
					return
				}
				for i := range sl[id] {
					sl[id][i] = 0
				}
				for i := range cl[id] {
					cl[id][i] = 0
				}
				for i := desde; i < hasta; i++ {
					p := ds.Punto(i)
					c := kmeans.CentroideMasCercano(p, centroides, k)
					for j := 0; j < kmeans.D; j++ {
						sl[id][c*kmeans.D+j] += p[j]
					}
					cl[id][c]++
				}
				done <- id
			}
		}(w, inicios[w], inicios[w+1])
	}

	sumas := make([]float64, k*kmeans.D)
	conteos := make([]int, k)

	for it := 0; it < iter; it++ {
		for w := 0; w < W; w++ {
			proceed[w] <- it
		}
		for w := 0; w < W; w++ {
			<-done
		}
		mu.Lock()
		for i := range sumas {
			sumas[i] = 0
		}
		for i := range conteos {
			conteos[i] = 0
		}
		for w := 0; w < W; w++ {
			for c := 0; c < k; c++ {
				for j := 0; j < kmeans.D; j++ {
					sumas[c*kmeans.D+j] += sl[w][c*kmeans.D+j]
				}
				conteos[c] += cl[w][c]
			}
		}
		kmeans.RecalcularCentroides(centroides, sumas, conteos, k)
		mu.Unlock()
	}

	for w := 0; w < W; w++ {
		proceed[w] <- -1
	}
	wg.Wait()
	for i := range proceed {
		close(proceed[i])
	}

	return centroides, conteos, kmeans.Inercia(ds, centroides, k)
}

func main() {
	datos := flag.String("datos", "../data/processed/features.bin", "dataset binario")
	kmin := flag.Int("kmin", 2, "k minimo para el metodo del codo")
	kmax := flag.Int("kmax", 10, "k maximo para el metodo del codo")
	kperfil := flag.Int("kperfil", 5, "k para el analisis de perfiles clinicos")
	iter := flag.Int("iter", 10, "iteraciones fijas")
	workers := flag.Int("w", 12, "workers")
	semilla := flag.Uint64("semilla", 42, "semilla")
	flag.Parse()

	ds, err := kmeans.CargarBinario(*datos)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error al cargar:", err)
		os.Exit(1)
	}

	// =====================================================================
	// PARTE 1 - METODO DEL CODO
	// Justifica la eleccion de k. Se ejecuta K-Means para cada k del rango
	// y se registra la inercia; el "codo" de la curva es el k a partir del
	// cual agregar clusters ya no reduce significativamente la inercia.
	// =====================================================================
	fmt.Println("==============================================================")
	fmt.Println("PARTE 1 - METODO DEL CODO (justificacion de k)")
	fmt.Println("==============================================================")
	fmt.Printf("registros: %d   |   iteraciones: %d   |   workers: %d\n\n",
		ds.N, *iter, *workers)
	fmt.Printf("%-5s %18s %14s %12s\n", "k", "inercia (MSSC)", "reduccion", "red. %")

	var inerciaAnterior float64
	for k := *kmin; k <= *kmax; k++ {
		_, _, inercia := correKMeans(ds, k, *iter, *workers, *semilla)
		if k == *kmin {
			fmt.Printf("%-5d %18.2f %14s %12s\n", k, inercia, "-", "-")
		} else {
			red := inerciaAnterior - inercia
			fmt.Printf("%-5d %18.2f %14.2f %11.2f%%\n",
				k, inercia, red, red/inerciaAnterior*100)
		}
		inerciaAnterior = inercia
	}

	fmt.Println()
	fmt.Println("TODO(Andre): identifica el codo de la curva y justifica en el")
	fmt.Println("informe si k=5 fue una buena eleccion o si otro k la mejora.")
	fmt.Println("Si el codo NO cae en 5, dilo: es un hallazgo valido, no un error.")
	fmt.Println()

	// =====================================================================
	// PARTE 2 - PERFILES CLINICOS
	// Devuelve los centroides a unidades reales para poder interpretarlos.
	// Sin esto los centroides en z-score no dicen nada a un lector clinico.
	// =====================================================================
	fmt.Println("==============================================================")
	fmt.Printf("PARTE 2 - PERFILES CLINICOS (k = %d)\n", *kperfil)
	fmt.Println("==============================================================")

	centroides, conteos, inercia := correKMeans(ds, *kperfil, *iter, *workers, *semilla)

	fmt.Printf("inercia final: %.2f\n\n", inercia)
	fmt.Printf("%-9s %12s %9s", "cluster", "registros", "%")
	for j := 0; j < kmeans.D; j++ {
		fmt.Printf("%14s", nombre[j])
	}
	fmt.Println()
	fmt.Printf("%-9s %12s %9s", "", "", "")
	for j := 0; j < kmeans.D; j++ {
		fmt.Printf("%14s", "("+unidad[j]+")")
	}
	fmt.Println()
	fmt.Println("--------------------------------------------------------------------------------")

	for c := 0; c < *kperfil; c++ {
		real := desescalar(centroides[c*kmeans.D : (c+1)*kmeans.D])
		pct := float64(conteos[c]) / float64(ds.N) * 100
		fmt.Printf("%-9d %12d %8.2f%%", c, conteos[c], pct)
		for j := 0; j < kmeans.D; j++ {
			fmt.Printf("%14.2f", real[j])
		}
		fmt.Println()
	}

	fmt.Println()
	fmt.Println("REFERENCIAS CLINICAS para interpretar (fuente: OMS):")
	fmt.Println("  - bajo peso al nacer      : < 2500 g")
	fmt.Println("  - peso normal             : 2500 - 4000 g")
	fmt.Println("  - macrosomia              : > 4000 g")
	fmt.Println("  - prematuro               : < 37 semanas de gestacion")
	fmt.Println("  - a termino               : 37 - 41 semanas")
	fmt.Println("  - postermino              : > 41 semanas")
	fmt.Println("  - embarazo adolescente    : madre < 20 anios")
	fmt.Println("  - edad materna avanzada   : madre >= 35 anios")
	fmt.Println()
	fmt.Println("TODO(Andre): ponle un NOMBRE CLINICO a cada cluster usando estas")
	fmt.Println("referencias y las cifras de la tabla. Ejemplo del formato:")
	fmt.Println("  \"Cluster 2 (18.9% de los nacimientos): recien nacidos a termino")
	fmt.Println("   de peso normal, madres jovenes\"")
	fmt.Println()
	fmt.Println("Esta es la seccion que CIERRA el caso de uso: demuestra que el")
	fmt.Println("clustering encontro perfiles con sentido clinico y no grupos")
	fmt.Println("arbitrarios. Es lo que conecta el ODS 3 con el resultado real.")
	fmt.Println("==============================================================")
}
