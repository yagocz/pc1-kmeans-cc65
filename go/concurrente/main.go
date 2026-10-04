// K-Means CONCURRENTE - Worker Pool persistente con dispatch por canales.
//
// ============================================================================
// CORRESPONDENCIA CON EL MODELO FORMAL promela/kmeans_sync.pml
// ----------------------------------------------------------------------------
// Esta implementacion sigue EXACTAMENTE la arquitectura verificada en SPIN.
// Cada elemento del modelo tiene su contraparte directa en este archivo:
//
//	Promela                     Go
//	------------------------    ------------------------------------------
//	proctype Coordinador()      coordinador() (la goroutine principal)
//	proctype Worker(byte id)    goroutine lanzada en el Worker Pool
//	chan proceed[W]             canal proceed[id]: dispatch de la iteracion
//	chan done                   canal done: barrera (W notificaciones)
//	suma_local[id]              acumuladores privados de cada worker
//	byte FIN = 255              senal de apagado del pool
//	mtype fase                  variable fase (ASIGNACION/ACTUALIZACION)
//	byte en_asignacion          contador de workers en fase de asignacion
//	byte escritores             contador de escritores de centroides
//	byte procesado[id]          iteraciones completadas por cada worker
//
// Propiedades verificadas exhaustivamente en SPIN (20,940 estados, 0 errores):
//
//	[] !(fase == ACTUALIZACION && en_asignacion > 0)   exclusion mutua
//	[] (escritores <= 1)                               escritor unico
//	<> (fase == TERMINADO)                             terminacion
//
// ============================================================================
//
// MECANISMOS DE SINCRONIZACION (exigidos por el enunciado del curso):
//   - canales      : dispatch del trabajo y barrera de fin de iteracion
//   - sync.Mutex   : protege la escritura de los centroides globales
//   - sync.WaitGroup: cierre ordenado del pool al terminar
//
// DECISION DE DISENIO - pool PERSISTENTE:
// Los workers se crean UNA sola vez y viven durante todas las iteraciones,
// bloqueados en <-proceed[id] entre una y otra. La alternativa (crear W
// goroutines nuevas en cada iteracion) pagaria el costo de creacion en cada
// vuelta del algoritmo. Con el pool persistente ese costo se paga una vez.
//
// Go PURO: solo biblioteca estandar.
//
// Uso:
//
//	go run ./concurrente -datos ../data/processed/features.bin -k 5 -iter 10 -w 6
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"pc2/kmeans"
)

// FIN es la senal de apagado del pool. Corresponde a #define FIN 255
// del modelo Promela.
const FIN = -1

// Fases del algoritmo. Corresponden a mtype = { ASIGNACION, ACTUALIZACION,
// TERMINADO } del modelo.
const (
	faseAsignacion int32 = iota
	faseActualizacion
	faseTerminado
)

// acumulador es la memoria PRIVADA de cada worker: corresponde a
// suma_local[id] del modelo. El coordinador solo lo lee DESPUES de la
// barrera, cuando ningun worker esta escribiendo.
type acumulador struct {
	sumas   []float64
	conteos []int
	// padding para evitar false sharing: cada acumulador ocupa su propia
	// linea de cache y los workers no invalidan la cache de sus vecinos.
	_ [64]byte
}

func main() {
	datos := flag.String("datos", "../data/processed/features.bin", "ruta del dataset binario")
	k := flag.Int("k", 5, "numero de clusters")
	iter := flag.Int("iter", 10, "numero FIJO de iteraciones")
	semilla := flag.Uint64("semilla", 42, "semilla para los centroides iniciales")
	workers := flag.Int("w", runtime.NumCPU(), "numero de workers (goroutines)")
	silencio := flag.Bool("q", false, "solo imprimir el tiempo en ms")
	flag.Parse()

	if *workers < 1 {
		fmt.Fprintln(os.Stderr, "el numero de workers debe ser >= 1")
		os.Exit(1)
	}

	ds, err := kmeans.CargarBinario(*datos)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error al cargar:", err)
		os.Exit(1)
	}

	// Misma semilla que la version secuencial => mismos centroides iniciales.
	centroides := kmeans.InicializarCentroides(ds, *k, *semilla)

	W := *workers

	// Particion del dataset en W bloques contiguos. Corresponde al "chunk"
	// que el Coordinador envia por proceed[i] en el modelo.
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

	// ---------------------------------------------------------------
	// Canales del modelo:
	//   chan proceed[W] = [1] of { byte }   -> dispatch por worker
	//   chan done = [W] of { byte }         -> barrera
	// ---------------------------------------------------------------
	proceed := make([]chan int, W)
	for w := range proceed {
		proceed[w] = make(chan int, 1)
	}
	done := make(chan int, W)

	// Acumuladores privados: suma_local[id] del modelo.
	acums := make([]acumulador, W)
	for w := range acums {
		acums[w].sumas = make([]float64, *k*kmeans.D)
		acums[w].conteos = make([]int, *k)
	}

	// Estado observable, contraparte de las variables del modelo que las
	// propiedades LTL vigilan. Se mantienen con operaciones atomicas para
	// que las aserciones en tiempo de ejecucion sean fiables.
	var fase int32 = faseAsignacion
	var enAsignacion int32 // en_asignacion del modelo
	var escritores int32   // escritores del modelo
	procesado := make([]int32, W)

	// sync.Mutex protege la escritura de los centroides globales.
	// Corresponde a la region donde el modelo incrementa 'escritores' y
	// verifica assert(escritores <= 1).
	var muCentroides sync.Mutex

	// sync.WaitGroup coordina el cierre ordenado del pool.
	var wg sync.WaitGroup
	wg.Add(W)

	// ---------------------------------------------------------------
	// WORKER POOL PERSISTENTE
	// Replica proctype Worker(byte id) del modelo.
	// ---------------------------------------------------------------
	for w := 0; w < W; w++ {
		go func(id, desde, hasta int) {
			defer wg.Done()
			ac := &acums[id]

			// do :: proceed[id] ? t -> ...
			for t := range proceed[id] {
				// if :: t == FIN -> break
				if t == FIN {
					return
				}

				// atomic { en_asignacion++; assert(fase == ASIGNACION) }
				atomic.AddInt32(&enAsignacion, 1)
				if atomic.LoadInt32(&fase) != faseAsignacion {
					panic("violacion: worker activo fuera de la fase de asignacion")
				}

				// Reiniciar acumuladores privados (suma_local[id] = 0)
				for i := range ac.sumas {
					ac.sumas[i] = 0
				}
				for i := range ac.conteos {
					ac.conteos[i] = 0
				}

				// FASE DE ASIGNACION - sin locks.
				// Los centroides se LEEN de forma concurrente. El modelo
				// garantiza con assert(escritores == 0) que nadie los
				// escribe en este momento.
				if atomic.LoadInt32(&escritores) != 0 {
					panic("violacion: lectura de centroides con un escritor activo")
				}
				for i := desde; i < hasta; i++ {
					p := ds.Punto(i)
					c := kmeans.CentroideMasCercano(p, centroides, *k)
					for j := 0; j < kmeans.D; j++ {
						ac.sumas[c*kmeans.D+j] += p[j]
					}
					ac.conteos[c]++
				}

				// procesado[id]++ ; en_asignacion-- ; done ! id
				atomic.AddInt32(&procesado[id], 1)
				atomic.AddInt32(&enAsignacion, -1)
				done <- id
			}
		}(w, inicios[w], inicios[w+1])
	}

	// ---------------------------------------------------------------
	// Se cronometra SOLO el bucle del algoritmo, con los datos ya en
	// memoria (igual que en la version secuencial).
	// ---------------------------------------------------------------
	inicio := time.Now()

	sumasGlobal := make([]float64, *k*kmeans.D)
	conteosGlobal := make([]int, *k)

	// ---------------------------------------------------------------
	// COORDINADOR - replica proctype Coordinador() del modelo.
	// ---------------------------------------------------------------
	for it := 0; it < *iter; it++ {
		atomic.StoreInt32(&fase, faseAsignacion)

		// Master dispatch: un chunk por worker.
		// for (i : 0 .. W-1) { proceed[i] ! iter }
		for w := 0; w < W; w++ {
			proceed[w] <- it
		}

		// BARRERA: espera las W notificaciones.
		// for (i : 0 .. W-1) { done ? id }
		for w := 0; w < W; w++ {
			<-done
		}

		// atomic { fase = ACTUALIZACION; escritores++; assert(en_asignacion == 0) }
		atomic.StoreInt32(&fase, faseActualizacion)
		muCentroides.Lock()
		atomic.AddInt32(&escritores, 1)
		if atomic.LoadInt32(&enAsignacion) != 0 {
			panic("violacion: actualizacion con workers aun en asignacion")
		}

		// Reduccion global sobre las sumas privadas.
		// El coordinador lee suma_local[i] con la garantia de que ningun
		// worker esta escribiendo: todos pasaron la barrera.
		for i := range sumasGlobal {
			sumasGlobal[i] = 0
		}
		for i := range conteosGlobal {
			conteosGlobal[i] = 0
		}
		for w := 0; w < W; w++ {
			// assert(procesado[i] == iter + 1)
			if got := atomic.LoadInt32(&procesado[w]); got != int32(it+1) {
				panic(fmt.Sprintf("violacion: worker %d proceso %d veces, esperado %d",
					w, got, it+1))
			}
			ac := &acums[w]
			for c := 0; c < *k; c++ {
				for j := 0; j < kmeans.D; j++ {
					sumasGlobal[c*kmeans.D+j] += ac.sumas[c*kmeans.D+j]
				}
				conteosGlobal[c] += ac.conteos[c]
			}
		}

		kmeans.RecalcularCentroides(centroides, sumasGlobal, conteosGlobal, *k)

		// atomic { escritores--; iter++ }
		atomic.AddInt32(&escritores, -1)
		muCentroides.Unlock()
	}

	transcurrido := time.Since(inicio)

	// fase = TERMINADO; for (i : 0..W-1) { proceed[i] ! FIN }
	atomic.StoreInt32(&fase, faseTerminado)
	for w := 0; w < W; w++ {
		proceed[w] <- FIN
	}
	wg.Wait()
	for w := range proceed {
		close(proceed[w])
	}

	if *silencio {
		fmt.Printf("%.3f\n", float64(transcurrido.Microseconds())/1000.0)
		return
	}

	fmt.Println("==============================================================")
	fmt.Println("K-MEANS CONCURRENTE - Worker Pool con dispatch por canales")
	fmt.Println("==============================================================")
	fmt.Printf("registros cargados : %d\n", ds.N)
	fmt.Printf("dimensiones        : %d\n", kmeans.D)
	fmt.Printf("k (clusters)       : %d\n", *k)
	fmt.Printf("iteraciones        : %d (fijas, sin criterio de convergencia)\n", *iter)
	fmt.Printf("semilla            : %d\n", *semilla)
	fmt.Printf("WORKERS            : %d\n", W)
	fmt.Printf("GOMAXPROCS         : %d\n", runtime.GOMAXPROCS(0))
	fmt.Printf("NumCPU             : %d\n", runtime.NumCPU())
	fmt.Println("--------------------------------------------------------------")
	fmt.Println("ARQUITECTURA (verificada en promela/kmeans_sync.pml)")
	fmt.Printf("  pool persistente  : %d goroutines creadas UNA vez\n", W)
	fmt.Printf("  canales proceed[] : %d (uno por worker, dispatch)\n", W)
	fmt.Printf("  canal done        : capacidad %d (barrera)\n", W)
	fmt.Printf("  sync.Mutex        : protege escritura de centroides\n")
	fmt.Printf("  sync.WaitGroup    : cierre ordenado del pool\n")
	fmt.Println("--------------------------------------------------------------")
	fmt.Println("PARTICION DEL DATASET")
	for w := 0; w < W && w < 16; w++ {
		fmt.Printf("  worker %2d : [%9d , %9d)  =>  %9d puntos  (procesado %d veces)\n",
			w, inicios[w], inicios[w+1], inicios[w+1]-inicios[w],
			atomic.LoadInt32(&procesado[w]))
	}
	fmt.Println("--------------------------------------------------------------")
	fmt.Println("CENTROIDES FINALES")
	kmeans.ImprimirCentroides(centroides, *k)
	fmt.Println("--------------------------------------------------------------")
	fmt.Printf("puntos por cluster : %v\n", conteosGlobal)
	fmt.Printf("inercia (MSSC)     : %.6f\n", kmeans.Inercia(ds, centroides, *k))
	fmt.Println("--------------------------------------------------------------")
	fmt.Printf("mensajes por iteracion : %d dispatch + %d barrera = %d\n", W, W, 2*W)
	fmt.Printf("TIEMPO DEL ALGORITMO   : %.3f ms\n", float64(transcurrido.Microseconds())/1000.0)
	fmt.Println("==============================================================")
}
