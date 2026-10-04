// K-Means SECUENCIAL - algoritmo de Lloyd en una sola goroutine.
//
// Sirve como linea base de la medicion de Speedup y como referencia de
// correccion: la version concurrente debe producir exactamente los mismos
// centroides finales.
//
// Go PURO: solo biblioteca estandar.
//
// Uso:
//
//	go run ./secuencial -datos ../data/processed/features.bin -k 5 -iter 10
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"pc2/kmeans"
)

func main() {
	datos := flag.String("datos", "../data/processed/features.bin", "ruta del dataset binario")
	k := flag.Int("k", 5, "numero de clusters")
	iter := flag.Int("iter", 10, "numero FIJO de iteraciones")
	semilla := flag.Uint64("semilla", 42, "semilla para los centroides iniciales")
	silencio := flag.Bool("q", false, "solo imprimir el tiempo en ms")
	flag.Parse()

	ds, err := kmeans.CargarBinario(*datos)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error al cargar:", err)
		os.Exit(1)
	}

	centroides := kmeans.InicializarCentroides(ds, *k, *semilla)
	asignaciones := make([]int, ds.N)

	// ---------------------------------------------------------------
	// Se cronometra SOLO el bucle del algoritmo, con los datos ya en
	// memoria. La carga del archivo queda fuera a proposito: es una fase
	// que ninguna de las dos versiones paraleliza y su inclusion
	// distorsionaria el Speedup.
	// ---------------------------------------------------------------
	inicio := time.Now()

	sumas := make([]float64, *k*kmeans.D)
	conteos := make([]int, *k)

	for it := 0; it < *iter; it++ {
		// Reiniciar acumuladores de la iteracion
		for i := range sumas {
			sumas[i] = 0
		}
		for i := range conteos {
			conteos[i] = 0
		}

		// FASE 1 - Asignacion: cada punto al centroide mas cercano.
		// FASE 2 - Reduccion: acumular sumas por cluster.
		// En la version secuencial ambas ocurren en el mismo bucle y no
		// hay memoria compartida entre goroutines, por lo que no se
		// necesita ningun mecanismo de sincronizacion.
		for i := 0; i < ds.N; i++ {
			p := ds.Punto(i)
			c := kmeans.CentroideMasCercano(p, centroides, *k)
			asignaciones[i] = c
			for j := 0; j < kmeans.D; j++ {
				sumas[c*kmeans.D+j] += p[j]
			}
			conteos[c]++
		}

		kmeans.RecalcularCentroides(centroides, sumas, conteos, *k)
	}

	transcurrido := time.Since(inicio)

	if *silencio {
		fmt.Printf("%.3f\n", float64(transcurrido.Microseconds())/1000.0)
		return
	}

	fmt.Println("==============================================================")
	fmt.Println("K-MEANS SECUENCIAL")
	fmt.Println("==============================================================")
	fmt.Printf("registros cargados : %d\n", ds.N)
	fmt.Printf("dimensiones        : %d\n", kmeans.D)
	fmt.Printf("k (clusters)       : %d\n", *k)
	fmt.Printf("iteraciones        : %d (fijas, sin criterio de convergencia)\n", *iter)
	fmt.Printf("semilla            : %d\n", *semilla)
	fmt.Printf("GOMAXPROCS         : %d\n", runtime.GOMAXPROCS(0))
	fmt.Println("--------------------------------------------------------------")
	fmt.Println("CENTROIDES FINALES")
	kmeans.ImprimirCentroides(centroides, *k)
	fmt.Println("--------------------------------------------------------------")
	fmt.Printf("puntos por cluster : %v\n", conteos)
	fmt.Printf("inercia (MSSC)     : %.6f\n", kmeans.Inercia(ds, centroides, *k))
	fmt.Println("--------------------------------------------------------------")
	fmt.Printf("TIEMPO DEL ALGORITMO : %.3f ms\n", float64(transcurrido.Microseconds())/1000.0)
	fmt.Println("==============================================================")
}
