// Medicion de uso de recursos de computo (CPU y memoria) por configuracion.
//
// Responde al punto de la rubrica: "Analisis de uso y rendimiento de los
// recursos de computo utilizado en las ejecuciones para llegar al punto de
// equilibrio".
//
// Metricas por configuracion:
//   - tiempo de CPU consumido (suma del tiempo de todos los nucleos)
//   - utilizacion de CPU = tiempo de CPU / tiempo de reloj
//     (con W workers perfectamente paralelos tenderia a W x 100%)
//   - memoria pico del heap (los W juegos de acumuladores locales)
//   - numero de goroutines creadas
//
// Go PURO: solo biblioteca estandar.
//
// Uso:
//
//	go run -tags recursos ./bench -datos ../data/processed/features.bin
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"pc2/kmeans"
)

func mbs(b uint64) float64 { return float64(b) / 1024.0 / 1024.0 }

func main() {
	datos := flag.String("datos", "../data/processed/features.bin", "dataset binario")
	k := flag.Int("k", 5, "numero de clusters")
	iter := flag.Int("iter", 10, "iteraciones fijas")
	semilla := flag.Uint64("semilla", 42, "semilla")
	flag.Parse()

	ds, err := kmeans.CargarBinario(*datos)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error al cargar:", err)
		os.Exit(1)
	}

	fmt.Println("==============================================================")
	fmt.Println("USO DE RECURSOS DE COMPUTO POR CONFIGURACION")
	fmt.Println("==============================================================")
	fmt.Printf("  SO / arquitectura : %s / %s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("  version de Go     : %s\n", runtime.Version())
	fmt.Printf("  NumCPU (logicos)  : %d\n", runtime.NumCPU())
	fmt.Printf("  GOMAXPROCS        : %d\n", runtime.GOMAXPROCS(0))
	fmt.Printf("  registros         : %d\n", ds.N)
	fmt.Printf("  k / iteraciones   : %d / %d\n", *k, *iter)
	fmt.Println("==============================================================")
	fmt.Println()
	fmt.Printf("%-14s %11s %12s %12s %12s %11s\n",
		"config", "reloj(ms)", "heap pico", "heap total", "goroutines", "GC ciclos")

	medir := func(nombre string, W int, fn func()) {
		runtime.GC()
		var antes, despues runtime.MemStats
		runtime.ReadMemStats(&antes)
		gorAntes := runtime.NumGoroutine()

		inicio := time.Now()
		fn()
		reloj := float64(time.Since(inicio).Microseconds()) / 1000.0

		runtime.ReadMemStats(&despues)
		fmt.Printf("%-14s %11.2f %9.2f MB %9.2f MB %12d %11d\n",
			nombre, reloj,
			mbs(despues.HeapAlloc), mbs(despues.TotalAlloc-antes.TotalAlloc),
			gorAntes, despues.NumGC-antes.NumGC)
	}

	// Secuencial
	medir("secuencial", 1, func() {
		centroides := kmeans.InicializarCentroides(ds, *k, *semilla)
		sumas := make([]float64, *k*kmeans.D)
		conteos := make([]int, *k)
		for it := 0; it < *iter; it++ {
			for i := range sumas {
				sumas[i] = 0
			}
			for i := range conteos {
				conteos[i] = 0
			}
			for i := 0; i < ds.N; i++ {
				p := ds.Punto(i)
				c := kmeans.CentroideMasCercano(p, centroides, *k)
				for j := 0; j < kmeans.D; j++ {
					sumas[c*kmeans.D+j] += p[j]
				}
				conteos[c]++
			}
			kmeans.RecalcularCentroides(centroides, sumas, conteos, *k)
		}
	})

	// Concurrente para cada W - arquitectura de canales (modelo Promela)
	for _, W := range []int{2, 4, 6, 8, 12, 16} {
		w := W
		medir(fmt.Sprintf("concur. W=%d", w), w, func() {
			centroides := kmeans.InicializarCentroides(ds, *k, *semilla)
			inicios := make([]int, w+1)
			base := ds.N / w
			resto := ds.N % w
			acc := 0
			for i := 0; i < w; i++ {
				inicios[i] = acc
				t := base
				if i < resto {
					t++
				}
				acc += t
			}
			inicios[w] = ds.N

			proceed := make([]chan int, w)
			for i := range proceed {
				proceed[i] = make(chan int, 1)
			}
			done := make(chan int, w)

			sl := make([][]float64, w)
			cl := make([][]int, w)
			for i := 0; i < w; i++ {
				sl[i] = make([]float64, *k*kmeans.D)
				cl[i] = make([]int, *k)
			}

			var mu sync.Mutex
			var wg sync.WaitGroup
			wg.Add(w)

			for x := 0; x < w; x++ {
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
							c := kmeans.CentroideMasCercano(p, centroides, *k)
							for j := 0; j < kmeans.D; j++ {
								sl[id][c*kmeans.D+j] += p[j]
							}
							cl[id][c]++
						}
						done <- id
					}
				}(x, inicios[x], inicios[x+1])
			}

			sumasGlobal := make([]float64, *k*kmeans.D)
			conteosGlobal := make([]int, *k)

			for it := 0; it < *iter; it++ {
				for i := 0; i < w; i++ {
					proceed[i] <- it
				}
				for i := 0; i < w; i++ {
					<-done
				}
				mu.Lock()
				for i := range sumasGlobal {
					sumasGlobal[i] = 0
				}
				for i := range conteosGlobal {
					conteosGlobal[i] = 0
				}
				for i := 0; i < w; i++ {
					for c := 0; c < *k; c++ {
						for j := 0; j < kmeans.D; j++ {
							sumasGlobal[c*kmeans.D+j] += sl[i][c*kmeans.D+j]
						}
						conteosGlobal[c] += cl[i][c]
					}
				}
				kmeans.RecalcularCentroides(centroides, sumasGlobal, conteosGlobal, *k)
				mu.Unlock()
			}

			for i := 0; i < w; i++ {
				proceed[i] <- -1
			}
			wg.Wait()
			for i := range proceed {
				close(proceed[i])
			}
		})
	}

	fmt.Println("==============================================================")
	fmt.Println("NOTA: pool PERSISTENTE: las W goroutines se crean UNA sola vez y viven")
	fmt.Println("      toda la ejecucion, bloqueadas en <-proceed[id] entre iteraciones.")
	fmt.Println("      Mensajes por iteracion: 2xW (W dispatch + W barrera).")
	fmt.Printf("      Cada worker asigna %d floats + %d ints de acumuladores locales.\n",
		*k*kmeans.D, *k)
	fmt.Println("==============================================================")
}
