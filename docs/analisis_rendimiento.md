# Análisis de Rendimiento, Escalabilidad y Recursos de Cómputo (PC2)

**Autor:** Andre Angel Chipana Rios (U202220230)  
**Curso:** Programación Concurrente y Distribuida (CC65) — UPC 2026-20  
**Profesor:** Herminio Paucar Curasma  
**Entorno de Pruebas:** AMD Ryzen 5 5600X (6 núcleos físicos / 12 hilos SMT), 15.9 GB RAM, Windows 11 Pro build 26200, Go 1.27.0  
**Dataset:** 4,597,137 registros × 4 variables biomédicas (CNV - MINSA), $k = 5$, 10 iteraciones fijas, 15 corridas por configuración con media recortada.

**Procedencia de los datos (trazabilidad entre ramas):** este análisis no regenera
mediciones en otra máquina. Reutiliza las 105 ejecuciones del commit
`2fb0345` / `3751312` / `ef5fce1` de la rama `origin/feature/go-concurrente`
(`go/bench/bench.go`, `go/recursos/main.go`, `go/concurrente/main.go`,
`go/secuencial/main.go`, `go/kmeans/kmeans.go`) y el modelo formal del commit
`477de46` de la rama `origin/feature/promela-model`
(`promela/kmeans_sync.pml`, `promela/verificar.sh`, SPIN 6.5.2). Los archivos
`docs/resultados_bench.csv`, `docs/salida_bench.txt` y
`docs/salida_recursos.txt` se copian a esta rama para que el análisis sea
reproducible sin depender de otra rama.

---

## Resumen Ejecutivo de Datos Experimentales

Todas las interpretaciones cuantitativas de este documento se fundamentan en las 105 ejecuciones reales registradas en `docs/resultados_bench.csv`, procesadas bajo media recortada para eliminar ruido del sistema operativo.

> **Definición operativa usada en `go/bench/bench.go`:** media recortada =
> promedio tras descartar el mínimo y el máximo de cada serie de 15
> (recorte 1+1, 6.7% por cola). Desviación reportada en `salida_bench.txt` es
> **poblacional** (divide por $N$, no por $N-1$); subestima ~3.5% frente a la
> muestral. Para inferencia se usa además el error estándar
> $SE = s_{m}/\sqrt{N}$ con $s_{m}$ muestral.

### Tabla A — Speedup y Eficiencia (Arquitectura Final con Canales)

| Workers ($W$) | Tiempo Recortado (ms) | Speedup ($S$) | Eficiencia ($E$) | Fracción Secuencial ($s$) | Techo Local ($1/s$) |
|:---:|---:|:---:|:---:|:---:|:---:|
| **Secuencial (1)** | 1963.53 | 1.0000× | 100.00% | — | — |
| **2** | 1179.39 | 1.6649× | 83.24% | 20.13% | 4.97× |
| **4** | 697.86 | 2.8137× | 70.34% | 14.05% | 7.12× |
| **6** | 475.87 | 4.1262× | 68.77% | **9.08%** *(mínimo)* | **11.01×** |
| **8** | 451.33 | 4.3505× | 54.38% | 11.98% | 8.34× |
| **12** *(máximo recortado)* | **384.47** | **5.1071×** | **42.56%** | 12.27% | 8.15× |
| **16** | 387.10 | 5.0724× | 31.70% | 14.36% | 6.96× |

> $S = \bar{T}_{sec}/\bar{T}_{conc}$, $E = S/W$,
> $s = (W/S-1)/(W-1)$. El "techo" es **extrapolación local** bajo Amdahl con
> $s$ de esa fila; **no** es un límite alcanzable porque $s$ no es constante
> (ver 10.1). Se cambió la etiqueta "Sweet Spot unívoco" por "máximo
> recortado": $W=12$ vs $W=16$ es empate técnico (ver 10.2 y 11.2).

### Tabla A1 — Rigor estadístico (verificación propia desde el CSV)

| Config | media (ms) | recortada (ms) | desv. pobl. | desv. muestral | SE | IC95% media |
|:---|---:|---:|---:|---:|---:|:---|
| Sec | 1957.99 | 1963.53 | 113.34 | 117.32 | 30.29 | [1898.6, 2017.4] |
| W=2 | 1181.61 | 1179.39 | 86.02 | 89.04 | 22.99 | [1136.6, 1226.7] |
| W=4 | 698.27 | 697.86 | 36.68 | 37.97 | 9.80 | [679.1, 717.5] |
| W=6 | 479.99 | 475.87 | 33.10 | 34.27 | 8.85 | [462.6, 497.3] |
| W=8 | 452.29 | 451.33 | 45.07 | 46.65 | 12.05 | [428.7, 475.9] |
| W=12 | 385.62 | 384.47 | 37.85 | 39.18 | 10.12 | [365.8, 405.5] |
| W=16 | 389.23 | 387.10 | 24.80 | 25.67 | 6.63 | [376.2, 402.2] |

$CV$ 5–10%, rango 19–35% de la media: hay jitter real del scheduler/runtime.
Tests Welch (dos colas, $N=15$): $W12$ vs $W16$ $t=-0.30$ (no significativo,
IC totalmente solapados); $W6$ vs $W8$ $t=1.85$ ($p\approx0.07$, borderline);
$W8$ vs $W12$ $t=4.24$ (significativo). Ver 11.2.

### Tabla B — Uso de Memoria Heap del Runtime de Go

| Configuración | Tiempo Reloj (ms) | Heap Pico (MB) | Heap Asignado Extra (MB) | Sobrecarga de Memoria |
|:---|---:|---:|---:|:---:|
| **Secuencial** | 1692.78 | 140.57 MB | 0.00 MB | Línea base |
| **W = 2** | 956.86 | 140.58 MB | 0.00 MB | +0.007% |
| **W = 4** | 652.45 | 140.58 MB | 0.01 MB | +0.007% |
| **W = 6** | 493.27 | 140.59 MB | 0.01 MB | +0.014% |
| **W = 8** | 504.03 | 140.60 MB | 0.01 MB | +0.021% |
| **W = 12** | 365.43 | 140.62 MB | 0.03 MB | +0.035% |
| **W = 16** | 344.60 | 140.63 MB | 0.02 MB | **+0.042%** |

> **Advertencia metodológica (corrección):** Tabla B proviene de
> `go/recursos/main.go` (`runtime.ReadMemStats`), **una sola corrida por
> configuración**, no de las 15 del benchmark. Su columna "Tiempo Reloj" **no
> es comparable** a la media recortada de Tabla A. Ejemplo: en Tabla B $W16$
> (344.60 ms) aparece más rápido que $W12$ (365.43 ms), mientras en Tabla A
> ocurre lo inverso (387.10 vs 384.47 ms). Ambas son mediciones puntuales
> afectadas por ruido; solo Tabla A tiene potencia estadística. La parte
> válida de Tabla B es el heap, estable entre corridas.

---

## Sección 10 — Análisis de Speedup, Escalabilidad y Trade-offs (3 Puntos)

### 10.0 Correspondencia verificada Promela ↔ Go (base de todo el análisis)

El benchmark mide exactamente la arquitectura verificada en
`promela/kmeans_sync.pml` (`477de46`) e implementada en
`go/concurrente/main.go` (`2fb0345`):

| Promela | Go (`go/concurrente/main.go`) |
|---|---|
| `proctype Coordinador()` | goroutine principal (dispatch, reducción) |
| `proctype Worker(byte id)` | goroutine del Worker Pool persistente |
| `chan proceed[W]` (uno por worker) | `proceed []chan int` (cap 1) |
| `chan done` (barrera) | `done chan int` (cap W) |
| `suma_local[id]` | `acumulador{sumas []float64, conteos []int}` privado |
| `FIN = 255` | `const FIN = -1` |
| `fase / en_asignacion / escritores / procesado` | `int32` atómicos + `sync.Mutex` + `sync.WaitGroup` |

Propiedades SPIN: `exclusion_mutua []!(ACTUALIZACION && en_asignacion>0)`,
`escritor_unico [](escritores<=1)`, `terminacion <>(TERMINADO)` —
20,940 estados, 0 errores, 0 estados inalcanzables en `Worker/Coordinador/init`.
Contraejemplo `kmeans_sync_canal_compartido.pml`: un solo canal compartido
falla `procesado[i]==iter+1` a prof. 93 (un worker roba 3 chunks, reducción
suma 2 en vez de 6). Justifica un canal por worker en Go. En Go las mismas
aserciones corren en tiempo de ejecución y `go run -race` da 0 warnings.
Sin esta correspondencia, el Speedup no sería atribuible al diseño verificado.

### 10.1 Por qué el Speedup no es lineal: Análisis empírico de Amdahl y el costo de sincronización

El modelo clásico de la **Ley de Amdahl** establece que la aceleración teórica máxima de un programa concurrente está estrictamente acotada por su fracción secuencial no paralelizable ($s$):

$$S(W) = \frac{1}{s + \frac{1 - s}{W}} \implies S_{\max} = \lim_{W \to \infty} S(W) = \frac{1}{s}$$

En nuestra implementación en Go, la porción irreductiblemente secuencial está compuesta por:
1. El despacho de señales en los canales individuales `proceed[id]`.
2. La recolección de las $W$ confirmaciones en el canal de barrera `done`.
3. La reducción lineal de los $W$ acumuladores privados sobre las sumas globales.
4. El recálculo de coordenadas de los $k$ centroides (`RecalcularCentroides`) y la verificación de condiciones de parada.

#### Despeje de la fracción secuencial observada
Despejando analíticamente la fracción secuencial $s$ en función del speedup experimental medido $S$ y el número de workers $W$:

$$s = \frac{\frac{W}{S} - 1}{W - 1}$$

Al evaluar cada configuración con nuestros datos empíricos (Tabla A), emerge el **hallazgo técnico más relevante del estudio**:

> **Hallazgo Central:** La fracción secuencial $s$ **no es constante**. Varía desde un mínimo de **9.08%** ($W=6$) hasta un **20.13%** ($W=2$) y sube a **14.36%** ($W=16$).

```
Fracción Secuencial Observada s(W) vs. Número de Workers:
   20.13% (W=2)
     \
      14.05% (W=4)
        \
         9.08% (W=6) <-- MÍNIMO (6 Núcleos Físicos)
        /     \
   11.98% (W=8) 12.27% (W=12)
                     \
                      14.36% (W=16) <-- Meseta con mayor costo de coordinación
```

#### Interpretación y límites del modelo clásico de Amdahl
La Ley de Amdahl pura asume como hipótesis fundamental que la fracción secuencial $s$ es una constante intrínseca del algoritmo e independiente de $W$. Sin embargo, en un sistema concurrente basado en paso de mensajes, **la sincronización tiene un costo dinámico que crece en $O(W)$**:
* Cada iteración del algoritmo intercambia exactamente $2 \times W$ mensajes de canal ($W$ en despacho `proceed[id]` y $W$ en sincronización `done`).
* Para $W = 16$ y 10 iteraciones, el runtime de Go gestiona 320 operaciones de canales concurrentes, involucrando locks internos del runtime, cambios de estado de goroutines (`runnable` $\leftrightarrow$ `waiting`) y tráfico en la memoria caché.
* Por ello, a partir de $W > 6$, la fracción secuencial efectiva vuelve a crecer (de 9.08% a 14.36%). El "techo" $1/s$ de Tabla A debe leerse como **extrapolación local** de esa fila, no como límite global alcanzable: con $s$ variable no hay un único $S_{\max}$.

El mínimo absoluto en **$W = 6$ ($s = 9.08\%$)** no es una casualidad numérica: coincide exactamente con los **6 núcleos físicos reales** del procesador AMD Ryzen 5 5600X. En esta configuración se maximiza la relación entre el trabajo computacional intensivo (cálculo de distancias euclidianas **al cuadrado**, sin `sqrt`, en `kmeans.DistanciaCuadrada`) y la sobrecarga de sincronización, sin interferencias de contención de hardware.

---

### 10.2 Dónde y por qué se aplana la curva: Los tres tramos de escalabilidad

La curva de Speedup experimental exhibe tres tramos cualitativamente distintos que reflejan la interacción entre el runtime de Go y la microarquitectura del procesador:

```
Speedup
  5.11× |                                     * (W=12: 5.11×)   * (W=16: 5.07×)
        |                                    /                 (Meseta / empate técnico)
  4.13× |                         * (W=6) --' (W=8: 4.35×)
        |                        /
  2.81× |             * (W=4)  /
        |            /        /   [Tramo 2: Hilos SMT 8-12]
  1.66× |   * (W=2) /        /    Eficiencia: 54% -> 43%
        |  /       /        /
  1.00× | / [Tramo 1: Cores Físicos 2-6]
        |/  Eficiencia: 83% -> 69%
        +------------------------------------------------------------- Workers (W)
        0   2         4         6         8         12        16
```

#### Tramo 1: Núcleos Físicos Reales ($W = 2 \to 6$) | Alta Eficiencia ($83\% \to 69\%$)
* **Comportamiento:** Cada goroutine se asigna a un núcleo de cómputo físico independiente. Cada núcleo cuenta con su propia unidad de coma flotante (FPU), pipelines de ejecución dedicados y memorias caché privadas L1 (32 KB instrucción + 32 KB datos) y L2 (512 KB).
* **Rendimiento:** El speedup crece de forma casi lineal ($1.66\times \to 4.13\times$). La eficiencia se mantiene en niveles sobresalientes ($83.24\%$ en $W=2$ y $68.77\%$ en $W=6$). En este tramo, el cómputo puro domina abrumadoramente sobre la coordinación.

#### Tramo 2: Hilos Lógicos SMT ($W = 8 \to 12$) | Rendimientos Decrecientes ($54\% \to 43\%$)
* **Comportamiento:** Los workers 7 al 12 ya no disponen de núcleos físicos libres; se ejecutan sobre la tecnología **SMT (Simultaneous Multithreading)** de AMD.
* **Explicación Microarquitectónica:** Un hilo SMT **no es un núcleo adicional**. Dos hilos lógicos que residen en el mismo núcleo físico comparten las mismas unidades de ejecución ALU/FPU, el mismo predictor de saltos y la misma caché L1/L2. El código evita `math.Sqrt` (usa distancia cuadrada, mismo `argmin`) pero sigue siendo intensivo en FPU y sumas; el hilo hermano debe esperar ciclos desocupados para emitir instrucciones.
* **Impacto en Métricas:** Aunque el número de workers se duplica de 6 a 12 (+100%), el speedup solo sube de $4.13\times$ a $5.11\times$ (un modesto +23.7%), provocando una caída de la eficiencia de $68.77\%$ a $42.56\%$.

#### Tramo 3: Sobre-suscripción ($W = 16$) | Meseta, no degradación demostrable ($E = 31.70\%$)
* **Comportamiento:** Se instancian 16 goroutines sobre una máquina con solo 12 hilos de hardware disponibles (`GOMAXPROCS = 12`).
* **Mecanismo esperado:** El planificador del runtime de Go multiplexa 16 goroutines en 12 hilos OS mediante `runqueues` y *work stealing*, con costo extra de cambio de contexto y contención L3.
* **Resultado corregido:** El tiempo pasa de 384.47 ms a 387.10 ms (+2.63 ms, +0.7%). **Esta diferencia no es estadísticamente significativa**: $t$ Welch $=-0.30$, IC95 $[365.8,405.5]$ vs $[376.2,402.2]$ totalmente solapados. Lo correcto es hablar de **meseta / empate técnico**, no de "empeora" ni "degradación". La eficiencia cae a 31.70% por definición ($S/W$), pero el tiempo de pared se estanca. Para afirmar degradación haría falta $N$ mayor o test con potencia.

---

### 10.3 Trade-offs entre el Modelo Concurrente y Secuencial

La decisión de adoptar el modelo concurrente con Worker Pool implica compromisos técnicos que deben evaluarse objetivamente:

| Dimensión de Trade-off | Versión Secuencial | Versión Concurrente (Worker Pool) | Veredicto / Impacto |
|---|---|---|---|
| **Tiempo de Ejecución** | 1963.53 ms | **384.47 ms** ($W=12$, máximo recortado) | **Ganancia:** Reducción del **80.4%** en tiempo de cómputo ($5.11\times$ más rápido). Con IC: $W12$ significativamente mejor que $W8$ ($t=4.24$), pero indistinguible de $W16$. |
| **Consumo de Memoria** | 140.57 MB | 140.62 MB ($W=12$) | **Costo despreciable:** Solo +0.035% de memoria (+0.05 MB) por el pool persistente. Medición puntual estable. |
| **Complejidad de Código** | ~100 líneas, lógica lineal de Lloyd | ~330 líneas, canales, goroutines y atomics | **Desventaja:** Mayor esfuerzo de desarrollo, depuración y mantenimiento. |
| **Superficie de Concurrencia** | Inmune por diseño a carreras de datos | Susceptible a race conditions si se comparte memoria | **Mitigado:** Verificado formalmente con SPIN (0 errores, `477de46`) y `go -race` (0 warnings, `2fb0345`). |
| **Estabilidad de Tiempos** | CV 5.8%, rango 23% de media | CV 5–10%, rango hasta 35% | **Desventaja:** Requiere $N=15$ y media recortada; aun así $W6$ vs $W8$ queda borderline ($t=1.85$). |

#### Los costos ocultos de la concurrencia:
1. **Sobrecarga de Sincronización:** Para 10 iteraciones y 12 workers, el programa coordina 240 eventos de sincronización en canales bloqueantes. En datasets reducidos, este overhead supera al cómputo y vuelve al modelo concurrente contraproducente.
2. **Sensibilidad al Ruido de Muestreo:** Durante las pruebas preliminares con $N=10$, se detectó una anomalía donde $W=8$ parecía más lento que $W=6$. Al incrementar a $N=15$ repeticiones y calcular la media recortada, se demostró que se trataba de jitter estocástico del planificador de Go, consolidando una curva monótona en media recortada (aunque $W6$ vs $W8$ sigue con solape parcial de IC).

---

### 10.4 Comparación con la Literatura Científica de la PC1

Contrastar los resultados empíricos con los tres papers fundacionales revisados en la PC1 permite validar nuestras observaciones frente a la evidencia experimental global:

#### 1. Mussabayev et al. (2023) — *Parallel Clustering on Shared Memory Architectures*
* **Lo que anticipaban:** Al evaluar K-Means paralelo en arquitecturas multinúcleo de 8 núcleos, los autores advirtieron que a medida que se saturan los núcleos físicos, la contención por recursos de sincronización degrada la eficiencia por hilo.
* **Confirmación con nuestros datos:** Nuestros resultados reflejan con exactitud este principio: la eficiencia decae de forma estrictamente monótona desde $83.24\%$ ($W=2$) hasta $31.70\%$ ($W=16$). La reutilización de acumuladores privados evitó la contención por locks de memoria, pero la contención en el canal de barrera reprodujo la degradación predicha por Mussabayev.

#### 2. Feng et al. (2024) — *Scalability Bottlenecks in Concurrent Lloyd's Algorithm*
* **Lo que anticipaban:** El incremento en el grado de paralelismo no produce ganancias proporcionales indefinidas debido al cuello de botella en la fase de reducción y agregación global de centroides.
* **Confirmación matizada con nuestros datos:** El Tramo 3 ($W=16$) es compatible con Feng (estancamiento al crecer la reducción secuencial y el dispatch $O(W)$), pero **no** demuestra "speedup negativo": el retroceso $5.11\times\to5.07\times$ está dentro del ruido ($t=-0.30$). La lección de Feng se confirma como meseta, no como caída probada.

#### 3. Ghimire & Amsaad (2024) — *Memory and Computation Limits in Parallel Machine Learning*
* **Lo que sostenían:** Los autores argumentan que en el clustering paralelo sobre memoria compartida, el factor limitante primordial es la capacidad de memoria RAM y la saturación del ancho de banda del bus del sistema al cargar datasets masivos.
* **Diferencia técnica con nuestro caso (Aporte Propio):** En nuestro experimento **no alcanzamos el límite de memoria**, registrando un consumo pico de apenas **140.62 MB sobre un equipo con 15,900 MB disponibles (menos del 0.9% de la RAM)**. Nuestro cuello de botella fue puramente **computacional y de sincronización** (capacidad de cálculo de punto flotante en núcleos SMT y latencia de canales de Go), demostrando que cuando el dataset se empaqueta de forma contigua en memoria binaria sin duplicaciones, la memoria deja de ser el obstáculo y el diseño de la concurrencia asume el rol determinante.

---

## Sección 11 — Uso y Rendimiento de los Recursos de Cómputo (2 Puntos)

### 11.1 Análisis del Consumo de Memoria: El paralelismo cuesta solo 0.06 MB

Uno de los logros más destacados de la implementación es la eficiencia extrema en el uso de la memoria del sistema. De acuerdo con las mediciones puntuales tomadas mediante `runtime.ReadMemStats` de Go (`go/recursos/main.go`, Tabla B):

* Memoria Heap en versión secuencial: **140.57 MB**
* Memoria Heap en versión concurrente ($W=16$): **140.63 MB**
* **Sobrecarga de memoria neta atribuible al paralelismo:** **+0.06 MB (+60 KB, o +0.042%)**.

```
Distribución del Heap en Memoria (140.62 MB en W=12):
+-----------------------------------------------------------+---------+
| Dataset Compartido por Referencia (Slice float64 continuo) | Workers |
| 4,597,137 registros x 4 dimensiones = 140.29 MB (99.76%)   | 0.33 MB |
+-----------------------------------------------------------+---------+
```

> **Precisión:** los 0.33 MB "Workers" no son solo los acumuladores. Cada
> worker aporta 200 B de acumuladores
> $(5\times4\times8+5\times8)$, es decir 2.4 KB en $W=12$, más stacks de
> goroutine (~8 KB iniciales c/u), canales `proceed/done` y overhead del
> runtime. Por eso el total es ~0.33 MB y no 2.4 KB. Aun así es despreciable.

Esta economía de memoria se fundamenta en dos decisiones de arquitectura:
1. **Dataset Inmutable Compartido por Referencia:** El dataset binario (`features.bin`, 140.29 MB) se mapea en memoria una única vez antes de iniciar el cronómetro. Las $W$ goroutines trabajadoras reciben punteros y slices que leen directamente el mismo bloque contiguo de memoria. No existe duplicación de datos.
2. **Worker Pool con Acumuladores Persistentes:** Las $W$ goroutines no se crean y destruyen en cada iteración; se instancian una sola vez al inicio del programa y permanecen bloqueadas en `<-proceed[id]`. Cada worker reutiliza su estructura fija de acumuladores locales. `GC ciclos = 0` en todas las configs confirma que no hay presión sobre el recolector ni pausas *Stop-The-World*.

> **Nota de consistencia Go:** `go/concurrente/main.go` usa `struct{sumas,conteos + pad [64]byte}` anti-*false sharing*; `go/bench/bench.go` y `go/recursos/main.go` usan `[][]` sin padding. La diferencia es de ~1 línea de caché por worker y no invalida el benchmark, pero para rigor total el bench debería usar el mismo `struct` con padding.

---

### 11.2 Determinación del Punto de Equilibrio (Sweet Spot) — versión corregida

El máximo en media recortada está en **$W = 12$ workers** (384.47 ms, $5.11\times$), coincidiendo con los núcleos lógicos del Ryzen 5 5600X. **Corrección frente a la versión anterior:** no es "unívoco" ni hay "inflexión negativa demostrada".

#### Justificación con los datos experimentales:
1. **Margen de Ganancia ($W \le 12$):**
   * De $W=4$ a $W=6$: 697.86 ms → 475.87 ms (-221.99 ms, **-31.8%**, $t$ muy significativo).
   * De $W=6$ a $W=8$: 475.87 ms → 451.33 ms (-24.54 ms, **-5.2%**, $t=1.85$ borderline, IC con solape parcial).
   * De $W=8$ a $W=12$: 451.33 ms → 384.47 ms (-66.86 ms, **-14.8%**, $t=4.24$ significativo).
2. **Meseta ($W > 12$):**
   * De $W=12$ a $W=16$: 384.47 ms → 387.10 ms (+2.63 ms, **+0.7%**, $t=-0.30$, $p\approx0.76$, IC solapados).
   * La eficiencia cae por definición a **31.70%**, pero el tiempo no empeora de forma probada.

#### Conclusión Técnica corregida:
El hardware no puede despachar más de 12 hilos en paralelo real; por encima de 12 hay multiplexación (*work stealing*) y costo $O(W)$ en canales. **$W = 12$ es el óptimo operativo** (máximo recortado con menor costo de coordinación), pero debe reportarse como **meseta $W12\approx W16$** y no como pico aislado. Recomendación: fijar $W=GOMAXPROCS=12$ en producción y, si se exige afirmar degradación, ampliar a $N\ge30$ con test de potencia.

---

### 11.3 Observación Metodológica y Telemetría de CPU

* **Comportamiento de CPU (inferencia, no medición directa):** Durante la ejecución secuencial, la carga se concentra en 1 núcleo al 100% (~8.3% de 12 hilos lógicos, ~16.6% de 6 físicos). En $W=12$ la fase de mapeo satura los 12 hilos lógicos en cálculo de distancias cuadradas. Son valores inferidos de `GOMAXPROCS=12` y la arquitectura, no lecturas de `htop/psutil`.
* **Rigor Científico en la Medición:** Los datos de memoria provienen de `runtime.ReadMemStats` en `go/recursos/main.go`. Para CPU, ráfagas $<400$ ms sufren aliasing con muestreadores periódicos (Monitor de Windows, `psutil`); por eso no se inventan porcentajes instantáneos. Mejora futura: medir `tiempo CPU / tiempo wall` con `time` o contadores del runtime en corridas largas ($iter\ge50`) para estabilizar la ventana.

---

## Limitaciones y validez (adenda de corrección)

1. Sin $W=1$ concurrente no se aísla el overhead puro del pool/canales.
2. Recorte 1+1 de 15 es débil ante 2+ outliers por cola; alternativa: mediana o recorte 20%.
3. $k=5$ sin codo en este doc; ver `go/calidad/main.go` (`ef5fce1`, con `TODO(Andre)` para codo y perfiles).
4. 10 iteraciones fijas sin criterio de convergencia: válido para Speedup (igual trabajo), no para calidad final.
5. Una sola máquina: no generalizar el $W^*$ a otro hardware.

---

## Anexo: Interpretación Clínica de Clusters (Cierre del Caso de Uso ODS 3)

Para cerrar el ciclo del problema planteado en la PC1 (Segmentación de Riesgo Neonatal en el Perú mediante registros del CNV - MINSA), se des-escalaron los centroides finales obtenidos en la versión concurrente a sus unidades clínicas reales (utilizando las medias y desviaciones estándar del z-score):

$$\text{Valor Real} = \text{Valor Escalado} \times \sigma + \mu$$

| Cluster | Registros | Proporción | Peso (g) | Talla (cm) | Gestación (sem) | Edad Madre (años) | Perfil Clínico Asignado |
|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---|
| **0** | 1,069,761 | 23.27% | 3650.07 g | 50.91 cm | 39.70 sem | 24.17 años | **Bajo Riesgo / Óptimo:** Recién nacidos a término con excelente peso y talla, madres jóvenes. |
| **1** | 658,842 | 14.33% | 2781.21 g | 47.19 cm | 37.37 sem | 32.26 años | **Vulnerabilidad Neonatal:** Talla baja y peso limítrofe, gestación temprana en madres adultas. |
| **2** | 866,990 | 18.86% | 3324.39 g | 49.77 cm | 38.03 sem | 28.46 años | **Término Estándar:** Antropometría promedio normal en madres de edad fértil estándar. |
| **3** | 991,181 | 21.56% | 3536.39 g | 50.29 cm | 39.39 sem | 35.97 años | **Riesgo Materno Avanzado:** Neonatos a término con peso adecuado, madres $\ge 35$ años. |
| **4** | 1,010,363 | 21.98% | 2991.24 g | 48.18 cm | 39.01 sem | 22.12 años | **Riesgo Nutricional Juvenil:** Peso inferior a 3000g en madres muy jóvenes (embarazo juvenil). |

> Los perfiles son etiquetas interpretativas (medias por cluster des-escaladas
> con $\mu=[3294.82,49.43,38.83,28.23]$,
> $\sigma=[424.81,1.78,1.19,6.89]$ de `go/calidad/main.go`); no sustituyen
> validación con silueta/codo. Demuestran que el K-Means concurrente
> ($5.11\times$ en $W12$) preserva la fidelidad clínica para ODS 3 (metas 3.1 y 3.2).

## Reproducibilidad (en esta rama)

```bash
# CSV ya versionado aquí (excepción en .gitignore):
#   docs/resultados_bench.csv (105 filas, de origin/feature/go-concurrente)
# Salidas crudas copiadas: docs/salida_bench.txt, docs/salida_recursos.txt
# Para regenerar en la rama Go (misma máquina Ryzen 5600X, no otra):
#   git checkout origin/feature/go-concurrente
#   python scripts/csv_a_binario.py
#   cd go && go run ./bench -k 5 -iter 10 -n 15
#   go run ./recursos -k 5 -iter 10
# Verificación Promela (rama promela-model 477de46):
#   cd promela && bash verificar.sh  # SPIN 6.5.2, 0 errores
```
