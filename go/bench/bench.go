// Benchmark de K-Means: secuencial vs concurrente.
//
// Ejecuta ambas versiones N veces por configuracion, guarda TODOS los
// tiempos crudos en un CSV y calcula media simple, media recortada,
// Speedup y Eficiencia.
//
// La media recortada descarta el valor maximo y el minimo de cada serie
// antes de promediar, para neutralizar el ruido del sistema operativo
// (una corrida que cayo junto a un pico de CPU de otro proceso no debe
// arrastrar el promedio).
//
// Go PURO: solo biblioteca estandar.
//
// Uso:
//	go run ./bench -datos ../data/processed/features.bin -k 5 -iter 10 -n 10
package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"time"

	"pc2/kmeans"
)

// FIN es la senal de apagado del pool (FIN del modelo Promela).
const FIN = -1

// resultado guarda la serie de tiempos de una configuracion.
type resultado struct {
	version string
	workers int
	tiempos []float64 // ms
}

func mediaSimple(xs []float64) float64 {
	var s float64
	for _, x := range xs {
		s += x
	}
	return s / float64(len(xs))
}

// mediaRecortada descarta el maximo y el minimo antes de promediar.
func mediaRecortada(xs []float64) float64 {
	if len(xs) <= 2 {
		return mediaSimple(xs)
	}
	c := make([]float64, len(xs))
	copy(c, xs)
	sort.Float64s(c)
	return mediaSimple(c[1 : len(c)-1])
}

func desviacion(xs []float64) float64 {
	m := mediaSimple(xs)
	var s float64
	for _, x := range xs {
		s += (x - m) * (x - m)
	}
	v := s / float64(len(xs))
	// raiz cuadrada por Newton (evita importar math solo para esto)
	if v == 0 {
		return 0
	}
	r := v
	for i := 0; i < 40; i++ {
		r = 0.5 * (r + v/r)
	}
	return r
}

func minimo(xs []float64) float64 {
	m := xs[0]
	for _, x := range xs {
		if x < m {
			m = x
		}
	}
	return m
}

func maximo(xs []float64) float64 {
	m := xs[0]
	for _, x := range xs {
		if x > m {
			m = x
		}
	}
	return m
}

// correSecuencial ejecuta el algoritmo secuencial y devuelve el tiempo en ms.
func correSecuencial(ds *kmeans.Dataset, k, iter int, semilla uint64) float64 {
	centroides := kmeans.InicializarCentroides(ds, k, semilla)
	sumas := make([]float64, k*kmeans.D)
	conteos := make([]int, k)

	inicio := time.Now()
	for it := 0; it < iter; it++ {
		for i := range sumas {
			sumas[i] = 0
		}
		for i := range conteos {
			conteos[i] = 0
		}
		for i := 0; i < ds.N; i++ {
			p := ds.Punto(i)
			c := kmeans.CentroideMasCercano(p, centroides, k)
			for j := 0; j < kmeans.D; j++ {
				sumas[c*kmeans.D+j] += p[j]
			}
			conteos[c]++
		}
		kmeans.RecalcularCentroides(centroides, sumas, conteos, k)
	}
	return float64(time.Since(inicio).Microseconds()) / 1000.0
}

// correConcurrente ejecuta el Worker Pool persistente con dispatch por
// canales (la arquitectura verificada en promela/kmeans_sync.pml) y
// devuelve el tiempo en ms.
func correConcurrente(ds *kmeans.Dataset, k, iter, W int, semilla uint64) float64 {
	centroides := kmeans.InicializarCentroides(ds, k, semilla)

	inicios := make([]int, W+1)
	base := ds.N / W
	resto := ds.N % W
	acc := 0
	for w := 0; w < W; w++ {
		inicios[w] = acc
		tam := base
		if w < resto {
			tam++
		}
		acc += tam
	}
	inicios[W] = ds.N

	// Canales del modelo: proceed[W] (dispatch) y done (barrera).
	proceed := make([]chan int, W)
	for w := range proceed {
		proceed[w] = make(chan int, 1)
	}
	done := make(chan int, W)

	// Acumuladores privados por worker (suma_local[id] del modelo).
	sumasLocal := make([][]float64, W)
	conteosLocal := make([][]int, W)
	for w := 0; w < W; w++ {
		sumasLocal[w] = make([]float64, k*kmeans.D)
		conteosLocal[w] = make([]int, k)
	}

	var muCentroides sync.Mutex
	var wg sync.WaitGroup
	wg.Add(W)

	// Pool PERSISTENTE: las goroutines se crean una sola vez.
	for w := 0; w < W; w++ {
		go func(id, desde, hasta int) {
			defer wg.Done()
			sl := sumasLocal[id]
			cl := conteosLocal[id]
			for t := range proceed[id] {
				if t == FIN {
					return
				}
				for i := range sl {
					sl[i] = 0
				}
				for i := range cl {
					cl[i] = 0
				}
				for i := desde; i < hasta; i++ {
					p := ds.Punto(i)
					c := kmeans.CentroideMasCercano(p, centroides, k)
					for j := 0; j < kmeans.D; j++ {
						sl[c*kmeans.D+j] += p[j]
					}
					cl[c]++
				}
				done <- id
			}
		}(w, inicios[w], inicios[w+1])
	}

	sumasGlobal := make([]float64, k*kmeans.D)
	conteosGlobal := make([]int, k)

	inicio := time.Now()
	for it := 0; it < iter; it++ {
		// Dispatch
		for w := 0; w < W; w++ {
			proceed[w] <- it
		}
		// Barrera
		for w := 0; w < W; w++ {
			<-done
		}
		// Reduccion sobre las sumas privadas, bajo el mutex de centroides.
		muCentroides.Lock()
		for i := range sumasGlobal {
			sumasGlobal[i] = 0
		}
		for i := range conteosGlobal {
			conteosGlobal[i] = 0
		}
		for w := 0; w < W; w++ {
			for c := 0; c < k; c++ {
				for j := 0; j < kmeans.D; j++ {
					sumasGlobal[c*kmeans.D+j] += sumasLocal[w][c*kmeans.D+j]
				}
				conteosGlobal[c] += conteosLocal[w][c]
			}
		}
		kmeans.RecalcularCentroides(centroides, sumasGlobal, conteosGlobal, k)
		muCentroides.Unlock()
	}
	transcurrido := time.Since(inicio)

	// Apagado del pool
	for w := 0; w < W; w++ {
		proceed[w] <- FIN
	}
	wg.Wait()
	for w := range proceed {
		close(proceed[w])
	}

	return float64(transcurrido.Microseconds()) / 1000.0
}

func main() {
	datos := flag.String("datos", "../data/processed/features.bin", "dataset binario")
	k := flag.Int("k", 5, "numero de clusters")
	iter := flag.Int("iter", 10, "iteraciones fijas")
	n := flag.Int("n", 10, "ejecuciones por configuracion")
	semilla := flag.Uint64("semilla", 42, "semilla de centroides")
	salida := flag.String("salida", "../docs/resultados_bench.csv", "CSV de tiempos crudos")
	flag.Parse()

	fmt.Println("==============================================================")
	fmt.Println("BENCHMARK K-MEANS  -  secuencial vs concurrente")
	fmt.Println("==============================================================")
	fmt.Println("ESPECIFICACIONES DE LA MAQUINA")
	fmt.Printf("  SO / arquitectura : %s / %s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("  version de Go     : %s\n", runtime.Version())
	fmt.Printf("  NumCPU (logicos)  : %d\n", runtime.NumCPU())
	fmt.Printf("  GOMAXPROCS        : %d\n", runtime.GOMAXPROCS(0))
	fmt.Println("--------------------------------------------------------------")

	ds, err := kmeans.CargarBinario(*datos)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error al cargar:", err)
		os.Exit(1)
	}
	fmt.Printf("  registros         : %d\n", ds.N)
	fmt.Printf("  dimensiones       : %d\n", kmeans.D)
	fmt.Printf("  k                 : %d\n", *k)
	fmt.Printf("  iteraciones fijas : %d\n", *iter)
	fmt.Printf("  corridas por conf : %d\n", *n)
	fmt.Printf("  semilla           : %d\n", *semilla)
	fmt.Println("==============================================================")
	fmt.Println()

	// Grilla de workers: 2, 4, 6, 8, 12, 16.
	// El 6 son los nucleos FISICOS y el 12 los LOGICOS de esta maquina;
	// el 16 sobre-suscribe a proposito, para mostrar la degradacion.
	configs := []int{2, 4, 6, 8, 12, 16}

	var resultados []resultado

	// --- Serie secuencial (linea base) ---
	fmt.Print("secuencial   : ")
	sec := make([]float64, 0, *n)
	for i := 0; i < *n; i++ {
		t := correSecuencial(ds, *k, *iter, *semilla)
		sec = append(sec, t)
		fmt.Printf("%.0f ", t)
	}
	fmt.Println()
	resultados = append(resultados, resultado{"secuencial", 1, sec})

	// --- Series concurrentes ---
	for _, W := range configs {
		fmt.Printf("concurrente W=%-2d: ", W)
		ts := make([]float64, 0, *n)
		for i := 0; i < *n; i++ {
			t := correConcurrente(ds, *k, *iter, W, *semilla)
			ts = append(ts, t)
			fmt.Printf("%.0f ", t)
		}
		fmt.Println()
		resultados = append(resultados, resultado{"concurrente", W, ts})
	}

	// --- Guardar tiempos CRUDOS en CSV ---
	f, err := os.Create(*salida)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error al crear el CSV:", err)
		os.Exit(1)
	}
	w := csv.NewWriter(f)
	w.Write([]string{"version", "workers", "corrida", "tiempo_ms"})
	for _, r := range resultados {
		for i, t := range r.tiempos {
			w.Write([]string{
				r.version,
				strconv.Itoa(r.workers),
				strconv.Itoa(i + 1),
				strconv.FormatFloat(t, 'f', 3, 64),
			})
		}
	}
	w.Flush()
	f.Close()

	// --- Tabla 1: tiempos por configuracion ---
	baseRec := mediaRecortada(sec)

	fmt.Println()
	fmt.Println("==============================================================")
	fmt.Println("TABLA 1 - TIEMPOS POR CONFIGURACION (ms)")
	fmt.Println("==============================================================")
	fmt.Printf("%-14s %10s %10s %10s %12s %12s\n",
		"configuracion", "min", "max", "desv", "media", "m.recortada")
	for _, r := range resultados {
		nombre := r.version
		if r.version == "concurrente" {
			nombre = fmt.Sprintf("concur. W=%d", r.workers)
		}
		fmt.Printf("%-14s %10.2f %10.2f %10.2f %12.2f %12.2f\n",
			nombre, minimo(r.tiempos), maximo(r.tiempos), desviacion(r.tiempos),
			mediaSimple(r.tiempos), mediaRecortada(r.tiempos))
	}

	// --- Tabla 2: Speedup y Eficiencia ---
	fmt.Println()
	fmt.Println("==============================================================")
	fmt.Println("TABLA 2 - SPEEDUP Y EFICIENCIA (sobre media recortada)")
	fmt.Println("==============================================================")
	fmt.Printf("%-10s %14s %14s %10s %12s\n",
		"workers", "T_sec (ms)", "T_conc (ms)", "Speedup", "Eficiencia")
	for _, r := range resultados {
		if r.version != "concurrente" {
			continue
		}
		rec := mediaRecortada(r.tiempos)
		sp := baseRec / rec
		ef := sp / float64(r.workers)
		fmt.Printf("%-10d %14.2f %14.2f %10.4f %11.2f%%\n",
			r.workers, baseRec, rec, sp, ef*100)
	}
	fmt.Println("==============================================================")
	fmt.Printf("\n[ok] tiempos crudos guardados en %s\n", *salida)
}
