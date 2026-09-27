// Package kmeans contiene el nucleo compartido entre la version secuencial
// y la concurrente de K-Means: carga de datos, distancia, inicializacion de
// centroides y actualizacion.
//
// Go PURO: solo biblioteca estandar, sin dependencias de terceros.
//
// Curso CC65 - Programacion Concurrente y Distribuida - PC2
package kmeans

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
)

// D es el numero de dimensiones (features) del dataset:
// PESO_NACIDO_ESC, TALLA_NACIDO_ESC, DUR_EMB_PARTO_ESC, Edad_Madre_ESC
const D = 4

// Dataset guarda los puntos en un slice PLANO de tamanio N*D en lugar de
// []struct o [][]float64. Esto mejora la localidad de cache: los D valores
// de un punto quedan contiguos en memoria y el recorrido secuencial
// aprovecha las lineas de cache completas. Con 4.6 millones de puntos la
// diferencia es medible.
type Dataset struct {
	N      int       // numero de puntos
	Puntos []float64 // tamanio N*D, plano
}

// Punto devuelve el offset del punto i dentro del slice plano.
func (d *Dataset) Punto(i int) []float64 {
	off := i * D
	return d.Puntos[off : off+D]
}

// CargarBinario lee el dataset desde un archivo binario de float64
// little-endian. Se usa formato binario en lugar de CSV a proposito: parsear
// 4.6 millones de filas de texto toma un tiempo del mismo orden que el
// propio clustering, y contaminaria la medicion del Speedup con una fase
// que ninguna de las dos versiones paraleliza.
func CargarBinario(ruta string) (*Dataset, error) {
	f, err := os.Open(ruta)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}

	total := info.Size() / 8 // 8 bytes por float64
	if total%D != 0 {
		return nil, fmt.Errorf("el archivo no es multiplo de %d dimensiones", D)
	}
	n := int(total) / D

	buf := make([]byte, info.Size())
	if _, err := io.ReadFull(f, buf); err != nil {
		return nil, err
	}

	puntos := make([]float64, total)
	for i := range puntos {
		bits := binary.LittleEndian.Uint64(buf[i*8:])
		puntos[i] = math.Float64frombits(bits)
	}

	return &Dataset{N: n, Puntos: puntos}, nil
}

// DistanciaCuadrada calcula la distancia euclidiana AL CUADRADO entre un
// punto y un centroide. No se aplica la raiz cuadrada porque el argmin es
// el mismo con o sin ella, y ahorrar una raiz por punto y por centroide en
// 4.6 millones de puntos es una optimizacion que importa.
func DistanciaCuadrada(punto, centroide []float64) float64 {
	var suma float64
	for j := 0; j < D; j++ {
		dif := punto[j] - centroide[j]
		suma += dif * dif
	}
	return suma
}

// CentroideMasCercano devuelve el indice del centroide mas proximo al punto.
func CentroideMasCercano(punto []float64, centroides []float64, k int) int {
	mejor := 0
	mejorDist := math.MaxFloat64
	for c := 0; c < k; c++ {
		d := DistanciaCuadrada(punto, centroides[c*D:(c+1)*D])
		if d < mejorDist {
			mejorDist = d
			mejor = c
		}
	}
	return mejor
}

// InicializarCentroides elige k centroides con una semilla FIJA usando un
// generador lineal congruencial propio (no math/rand, para que el resultado
// sea identico y reproducible entre ejecuciones y entre versiones).
//
// Es indispensable que ambas versiones partan de los MISMOS centroides: si
// no, cada una converge por un camino distinto y la diferencia de tiempo
// mediria la suerte de la inicializacion en lugar del efecto del paralelismo.
func InicializarCentroides(d *Dataset, k int, semilla uint64) []float64 {
	centroides := make([]float64, k*D)
	estado := semilla
	siguiente := func() uint64 {
		// LCG de Numerical Recipes: reproducible en cualquier maquina.
		estado = estado*6364136223846793005 + 1442695040888963407
		return estado >> 16
	}
	usados := make(map[int]bool, k)
	for c := 0; c < k; c++ {
		var idx int
		for {
			idx = int(siguiente() % uint64(d.N))
			if !usados[idx] {
				usados[idx] = true
				break
			}
		}
		copy(centroides[c*D:(c+1)*D], d.Punto(idx))
	}
	return centroides
}

// RecalcularCentroides divide las sumas acumuladas por los conteos para
// obtener los nuevos centroides. Si un cluster queda vacio se conserva su
// centroide anterior, para evitar NaN por division entre cero.
func RecalcularCentroides(centroides, sumas []float64, conteos []int, k int) {
	for c := 0; c < k; c++ {
		if conteos[c] == 0 {
			continue // cluster vacio: se mantiene el centroide previo
		}
		inv := 1.0 / float64(conteos[c])
		for j := 0; j < D; j++ {
			centroides[c*D+j] = sumas[c*D+j] * inv
		}
	}
}

// ImprimirCentroides muestra los centroides finales con 6 decimales.
// Sirve como prueba de correccion: la version secuencial y la concurrente
// deben imprimir exactamente lo mismo.
func ImprimirCentroides(centroides []float64, k int) {
	nombres := [D]string{"PESO_ESC", "TALLA_ESC", "GESTA_ESC", "EDAD_ESC"}
	fmt.Printf("%-10s", "cluster")
	for j := 0; j < D; j++ {
		fmt.Printf("%14s", nombres[j])
	}
	fmt.Println()
	for c := 0; c < k; c++ {
		fmt.Printf("%-10d", c)
		for j := 0; j < D; j++ {
			fmt.Printf("%14.6f", centroides[c*D+j])
		}
		fmt.Println()
	}
}

// Inercia calcula la suma de distancias cuadradas de cada punto a su
// centroide asignado (la funcion objetivo MSSC que K-Means minimiza).
// Se usa como segunda comprobacion de que ambas versiones convergen igual.
func Inercia(d *Dataset, centroides []float64, k int) float64 {
	var total float64
	for i := 0; i < d.N; i++ {
		p := d.Punto(i)
		c := CentroideMasCercano(p, centroides, k)
		total += DistanciaCuadrada(p, centroides[c*D:(c+1)*D])
	}
	return total
}
