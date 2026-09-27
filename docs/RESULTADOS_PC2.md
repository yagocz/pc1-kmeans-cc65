# Resultados del Entregable 2 (PC2)

> Todas las cifras de este documento provienen de la ejecución real de los
> programas en Go y de la verificación del modelo Promela con SPIN. Las salidas
> crudas están en `docs/salida_bench.txt`, `docs/salida_spin.txt`,
> `docs/salida_race.txt`, `docs/salida_recursos.txt`, `docs/salida_ejecucion.txt`
> y `docs/resultados_bench.csv`, y son reproducibles.

## Entorno de ejecución

Todas las mediciones se tomaron en **una sola máquina**, condición necesaria
para que los tiempos sean comparables entre sí.

| Componente | Valor |
|---|---|
| Procesador | AMD Ryzen 5 5600X |
| Núcleos físicos | 6 |
| Núcleos lógicos (SMT) | 12 |
| Memoria RAM | 15.9 GB |
| Sistema operativo | Windows 11 Pro, build 26200 |
| Versión de Go | go1.27.0 windows/amd64 |
| GOMAXPROCS | 12 |
| Versión de SPIN | 6.5.2 |

## Parámetros del experimento

| Parámetro | Valor | Motivo |
|---|---|---|
| Registros | 4,597,137 | dataset limpio de la PC1 |
| Dimensiones | 4 | features escaladas con z-score |
| k (clusters) | 5 | |
| Iteraciones | 10, **fijas** | sin criterio de convergencia, para que ambas versiones hagan idéntico trabajo |
| Semilla | 42 | mismos centroides iniciales en las dos versiones |
| Corridas por configuración | **15** | la varianza del dispatch por canales exige más muestras que las 10 iniciales |

Dos decisiones metodológicas que condicionan la validez de la medición:

**Se cronometra solo el algoritmo, no la carga de datos.** Leer y parsear
4.6 millones de filas de CSV toma un tiempo del mismo orden que el propio
clustering. Como esa fase no la paraleliza ninguna de las dos versiones, su
inclusión comprimiría artificialmente el Speedup. Por eso el dataset se
convierte previamente a un archivo binario de `float64`
(`scripts/csv_a_binario.py`, 140.29 MB) que Go carga con un solo `io.ReadFull`
antes de arrancar el cronómetro.

**Ambas versiones parten de los mismos centroides y ejecutan el mismo número
de iteraciones.** Sin esto, cada versión convergería por un camino distinto y
la diferencia de tiempo mediría la suerte de la inicialización en lugar del
efecto del paralelismo.

## 1. Correspondencia entre el modelo formal y la implementación

La implementación en Go sigue **exactamente** la arquitectura verificada en
`promela/kmeans_sync.pml`. Cada elemento del modelo tiene su contraparte
directa en el código:

| Promela | Go (`go/concurrente/main.go`) |
|---|---|
| `proctype Coordinador()` | la goroutine principal |
| `proctype Worker(byte id)` | goroutine del Worker Pool |
| `chan proceed[W] = [1] of {byte}` | `proceed []chan int`, un canal por worker |
| `chan done = [W] of {byte}` | `done chan int`, la barrera |
| `suma_local[id]` | acumuladores privados de cada worker |
| `byte FIN = 255` | `const FIN = -1`, señal de apagado |
| `mtype fase` | `fase int32` (atómica) |
| `byte en_asignacion` | `enAsignacion int32` (atómica) |
| `byte escritores` | `escritores int32` (atómica) |
| `byte procesado[id]` | `procesado []int32` (atómica) |

Las aserciones del modelo se replicaron como comprobaciones en tiempo de
ejecución dentro del código Go (`assert(fase == ASIGNACION)`,
`assert(escritores == 0)`, `assert(en_asignacion == 0)`,
`assert(procesado[i] == iter+1)`). Ninguna se disparó en las ejecuciones.

El enunciado exige además `sync.Mutex` y `sync.WaitGroup`: el mutex protege la
escritura de los centroides globales (la región donde el modelo verifica
`escritores <= 1`) y el WaitGroup coordina el cierre ordenado del pool.

**Diseño del pool: persistente.** Las W goroutines se crean **una sola vez** y
viven durante todas las iteraciones, bloqueadas en `<-proceed[id]` entre una y
otra. La alternativa —crear W goroutines nuevas en cada iteración— pagaría el
costo de creación en cada vuelta del algoritmo.

## 2. Comprobación de corrección

Ambas versiones se ejecutaron con la semilla 42:

| | Secuencial | Concurrente (W=12) |
|---|---|---|
| Puntos por cluster | `[1069761 658842 866990 991181 1010363]` | `[1069761 658842 866990 991181 1010363]` |
| Inercia (MSSC) | 8431227.573125 | 8431227.573126 |
| Tiempo | 1792.21 ms | 414.67 ms |

Los centroides finales coinciden en los 20 valores con 6 decimales y los
conteos por cluster son idénticos. La inercia difiere en el último dígito
decimal porque las sumas se acumulan en distinto orden: es error de redondeo
de punto flotante, no un fallo de sincronización.

### Detector de condiciones de carrera

```
CGO_ENABLED=1 go run -race ./concurrente -k 5 -iter 3 -w 6 -semilla 42
```

El detector **no emitió ningún `WARNING: DATA RACE`**. Es evidencia empírica
directa, complementaria a la demostración formal del modelo Promela.

## 3. Tiempos por configuración

15 corridas por configuración. La **media recortada** descarta el valor máximo
y el mínimo antes de promediar.

| Configuración | mín (ms) | máx (ms) | desv. | media (ms) | **media recortada (ms)** |
|---|---:|---:|---:|---:|---:|
| Secuencial | 1693.21 | 2150.80 | 113.34 | 1957.99 | **1963.53** |
| Concurrente W=2 | 1019.80 | 1372.29 | 86.02 | 1181.61 | **1179.39** |
| Concurrente W=4 | 635.42 | 766.51 | 36.68 | 698.27 | **697.86** |
| Concurrente W=6 | 443.34 | 570.22 | 33.10 | 479.99 | **475.87** |
| Concurrente W=8 | 379.83 | 537.13 | 45.07 | 452.29 | **451.33** |
| Concurrente W=12 | 326.10 | 460.14 | 37.85 | 385.62 | **384.47** |
| Concurrente W=16 | 352.82 | 453.43 | 24.80 | 389.23 | **387.10** |

> **Nota metodológica.** Una primera serie de 10 corridas produjo una anomalía
> en W=8, que aparecía más lento que W=6. Una repetición de control de 6
> corridas adicionales por configuración mostró que W=8 es efectivamente más
> rápido, y que la anomalía era ruido de muestreo. Se elevó el número de
> corridas de 10 a 15, con lo que la curva se volvió monótona. El dispatch por
> canales introduce más varianza que una partición estática, y por eso exige
> más muestras para que la media recortada sea estable.

## 4. Speedup y eficiencia

**Speedup = T_secuencial / T_concurrente**, sobre medias recortadas.
**Eficiencia = Speedup / W**, mide qué fracción de cada worker se aprovecha.

| Workers | T secuencial (ms) | T concurrente (ms) | **Speedup** | **Eficiencia** |
|---:|---:|---:|---:|---:|
| 2 | 1963.53 | 1179.39 | **1.6649×** | 83.24% |
| 4 | 1963.53 | 697.86 | **2.8137×** | 70.34% |
| 6 | 1963.53 | 475.87 | **4.1262×** | 68.77% |
| 8 | 1963.53 | 451.33 | **4.3505×** | 54.38% |
| **12** | 1963.53 | 384.47 | **5.1071×** | 42.56% |
| 16 | 1963.53 | 387.10 | **5.0724×** | 31.70% |

Gráficos: `docs/grafico_speedup.png`, `docs/grafico_eficiencia.png`,
`docs/grafico_tiempos.png`.

## 5. Uso de recursos de cómputo

| Configuración | Reloj (ms) | Heap pico | Heap asignado |
|---|---:|---:|---:|
| Secuencial | 1692.78 | 140.57 MB | 0.00 MB |
| Concurrente W=2 | 956.86 | 140.58 MB | 0.00 MB |
| Concurrente W=4 | 652.45 | 140.58 MB | 0.01 MB |
| Concurrente W=6 | 493.27 | 140.59 MB | 0.01 MB |
| Concurrente W=8 | 504.03 | 140.60 MB | 0.01 MB |
| Concurrente W=12 | 365.43 | 140.62 MB | 0.03 MB |
| Concurrente W=16 | 344.60 | 140.63 MB | 0.02 MB |

El dato más relevante es el **costo en memoria del paralelismo: 0.06 MB**
(de 140.57 MB a 140.63 MB entre la versión secuencial y W=16). Cada worker
asigna solo 20 `float64` de sumas y 5 `int` de conteos como acumuladores
locales, y el pool persistente los reutiliza en todas las iteraciones en lugar
de reasignarlos. El dataset de 140 MB se comparte por referencia y nunca se
duplica. El diseño gana 5× en tiempo a cambio de un 0.04% más de memoria.

## 6. Punto de equilibrio

El punto de equilibrio está en **W = 12 workers**, que coincide con el número
de núcleos lógicos de la máquina. En W = 16 el Speedup deja de crecer
(5.0724× frente a 5.1071×): agregar workers más allá de 12 ya no aporta.

La curva tiene tres tramos que se leen en los datos:

**Hasta W = 6** (núcleos físicos) la eficiencia se mantiene alta: 83%, 70%,
69%. Cada worker corre en un núcleo real.

**Entre W = 8 y W = 12** el Speedup sigue creciendo pero la eficiencia cae de
54% a 43%. Los workers 7 a 12 no corren en núcleos físicos sino en los hilos
SMT del Ryzen, que comparten unidades de ejecución con su hilo hermano.
Aportan trabajo, pero cada uno rinde bastante menos que un núcleo real.

**En W = 16** hay sobre-suscripción: 16 goroutines compiten por 12 hilos de
hardware. El planificador de Go las multiplexa sobre los mismos hilos del
sistema operativo, y el costo de cambio de contexto más el de los mensajes de
canal supera la ganancia. La eficiencia cae a 32% y el tiempo se estanca.

**Costo del dispatch por canales.** Cada iteración intercambia `2 × W`
mensajes (W de dispatch + W de barrera). Con W = 16 y 10 iteraciones son 320
operaciones de canal, cada una con su sincronización en el runtime de Go. Este
costo es el precio de una arquitectura que se puede verificar formalmente: el
paso de mensajes es lo que permite que el modelo Promela explore todos los
entrelazados posibles y demuestre ausencia de deadlock.

## 7. Verificación formal en Promela

Archivo: `promela/kmeans_sync.pml`. Configuración: W = 3 workers, 3
iteraciones, 2 puntos abstractos por partición.

El modelo reproduce el esqueleto de sincronización, no el cálculo numérico.
Conserva el `Coordinador`, los W `Worker`, los canales `proceed[]` y `done`, la
barrera de dos fases y la reducción sobre acumuladores privados. Se omiten las
distancias euclidianas y las coordenadas reales porque no influyen en la
corrección de la sincronización.

### Propiedades verificadas

| Propiedad | Fórmula LTL | Qué garantiza |
|---|---|---|
| Exclusión mutua | `[] !(fase == ACTUALIZACION && en_asignacion > 0)` | ningún worker lee centroides mientras se escriben |
| Escritor único | `[] (escritores <= 1)` | nunca hay dos procesos escribiendo los centroides |
| Terminación | `<> (fase == TERMINADO)` | el algoritmo siempre termina, no hay deadlock |

Además se comprueban por aserción que cada worker procesa su chunk exactamente
una vez (`procesado[i] == iter+1`) y que la reducción no pierde datos
(`total == W * PUNTOS`).

### Resultados

Verificación **exhaustiva** con SPIN 6.5.2 y Partial Order Reduction:

| Verificación | Estados almacenados | Transiciones | Profundidad | Errores |
|---|---:|---:|---:|---:|
| Seguridad (aserciones + deadlocks) | 20,940 | 30,730 | 325 | **0** |
| LTL `exclusion_mutua` | 20,940 | 30,734 | 639 | **0** |
| LTL `escritor_unico` | 20,940 | 30,734 | 639 | **0** |
| LTL `terminacion` | 20,704 | 81,607 | 594 | **0** |

Consumo: 130.29 MB, con 74.39% de compresión del vector de estados.

Todo el código de los tres procesos resultó alcanzable: 0 de 29 estados sin
alcanzar en `Worker`, 0 de 60 en `Coordinador`, 0 de 12 en `init`. Esto
descarta código muerto en el modelo.

### Validación del método: contraejemplo deliberado

Una verificación que no reporta errores solo es informativa si el verificador
es capaz de encontrarlos. Para comprobarlo se construyó la variante
`kmeans_sync_canal_compartido.pml`, idéntica salvo que los W workers comparten
un único canal `proceed` en lugar de tener uno cada uno.

SPIN encuentra la violación a profundidad 93:
`assertion violated (procesado[i] == (iter+1))`. La traza muestra que un mismo
worker consume los tres chunks de la iteración 0 mientras los otros dos no
procesan nada, y la barrera se libera igualmente porque recibe tres mensajes en
`done`. Al suprimir esa aserción falla la siguiente, `total == W*PUNTOS`: la
reducción acumula 2 en lugar de 6, es decir, se pierden datos al recalcular los
centroides.

El contraejemplo confirma que la ausencia de errores en el modelo correcto es
un resultado significativo y no un falso negativo del verificador. También
justifica la decisión de diseño de usar **un canal por worker** en la
implementación en Go.

## 8. Reproducibilidad

```bash
# 1. Reconstruir el dataset
python scripts/unir_dataset.py
python scripts/csv_a_binario.py

# 2. Ejecutar ambas versiones
cd go
go run ./secuencial  -k 5 -iter 10 -semilla 42
go run ./concurrente -k 5 -iter 10 -semilla 42 -w 12

# 3. Detector de carreras (requiere gcc)
CGO_ENABLED=1 go run -race ./concurrente -k 5 -iter 3 -w 6

# 4. Benchmark completo (15 corridas por configuración)
go run ./bench -k 5 -iter 10 -n 15

# 5. Uso de recursos
go run ./recursos -k 5 -iter 10

# 6. Gráficos
cd .. && python scripts/graficos_speedup.py

# 7. Verificación formal (Linux/WSL o Docker, ver promela/README.md)
cd promela && bash verificar.sh
```
