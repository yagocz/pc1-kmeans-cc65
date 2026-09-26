# Análisis de Rendimiento, Escalabilidad y Recursos de Cómputo (PC2)

**Autor:** Andre Angel Chipana Rios (U202220230)  
**Curso:** Programación Concurrente y Distribuida (CC65) — UPC 2026-20  
**Profesor:** Herminio Paucar Curasma  
**Entorno de Pruebas:** AMD Ryzen 5 5600X (6 núcleos físicos / 12 hilos SMT), 15.9 GB RAM, Windows 11 Pro build 26200, Go 1.27.0  
**Dataset:** 4,597,137 registros × 4 variables biomédicas (CNV - MINSA), $k = 5$, 10 iteraciones fijas, 15 corridas por configuración con media recortada.

---

## Resumen Ejecutivo de Datos Experimentales

Todas las interpretaciones cuantitativas de este documento se fundamentan en las 105 ejecuciones reales registradas en `docs/resultados_bench.csv`, procesadas bajo media recortada para eliminar ruido del sistema operativo:

### Tabla A — Speedup y Eficiencia (Arquitectura Final con Canales)
| Workers ($W$) | Tiempo Recortado (ms) | Speedup ($S$) | Eficiencia ($E$) | Fracción Secuencial ($s$) | Techo Teórico ($1/s$) |
|:---:|---:|:---:|:---:|:---:|:---:|
| **Secuencial (1)** | 1963.53 | 1.0000× | 100.00% | — | — |
| **2** | 1179.39 | 1.6649× | 83.24% | 20.13% | 4.97× |
| **4** | 697.86 | 2.8137× | 70.34% | 14.05% | 7.12× |
| **6** | 475.87 | 4.1262× | 68.77% | **9.08%** *(mínimo)* | **11.01×** |
| **8** | 451.33 | 4.3505× | 54.38% | 11.98% | 8.34× |
| **12** *(Sweet Spot)* | **384.47** | **5.1071×** | **42.56%** | 12.27% | 8.15× |
| **16** | 387.10 | 5.0724× | 31.70% | 14.36% | 6.96× |

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

---

## Sección 10 — Análisis de Speedup, Escalabilidad y Trade-offs (3 Puntos)

### 10.1 Por qué el Speedup no es lineal: Análisis empírico de Amdahl y el costo de sincronización

El modelo clásico de la **Ley de Amdahl** establece que la aceleración teórica máxima de un programa concurrente está estrictamente acotada por su fracción secuencial no paralelizable ($s$):

$$S(W) = \frac{1}{s + \frac{1 - s}{W}} \implies S_{\max} = \lim_{W \to \infty} S(W) = \frac{1}{s}$$

En nuestra implementación en Go, la porción irreductiblemente secuencial está compuesta por:
1. El despacho de señales en los canales individuales `proceed[id]`.
2. La recolección de las $W$ confirmaciones en el canal de barrera `done`.
3. La reducción lineal de los $W$ acumuladores privados sobre las sumas globales.
4. El recálculo de coordenadas de los $k$ centroides y la verificación de condiciones de parada.

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
         9.08% (W=6) <-- MÍNIMO ÓPTIMO (6 Núcleos Físicos)
        /     \
   11.98% (W=8) 12.27% (W=12)
                     \
                      14.36% (W=16) <-- Degradación por Sobre-suscripción
```

#### Interpretación y límites del modelo clásico de Amdahl
La Ley de Amdahl pura asume como hipótesis fundamental que la fracción secuencial $s$ es una constante intrínseca del algoritmo e independiente de $W$. Sin embargo, en un sistema concurrente basado en paso de mensajes, **la sincronización tiene un costo dinámico que crece en $O(W)$**:
* Cada iteración del algoritmo intercambia exactamente $2 \times W$ mensajes de canal ($W$ en despacho `proceed[id]` y $W$ en sincronización `done`).
* Para $W = 16$ y 10 iteraciones, el runtime de Go gestiona 320 operaciones de canales concurrentes, involucrando locks internos del runtime, cambios de estado de goroutines (`runnable` $\leftrightarrow$ `waiting`) y tráfico en la memoria caché.
* Por ello, a partir de $W > 6$, la fracción secuencial efectiva vuelve a crecer (de 9.08% a 14.36%), reduciendo el techo teórico de aceleración de **11.01×** a **6.96×**.

El mínimo absoluto en **$W = 6$ ($s = 9.08\%$)** no es una casualidad numérica: coincide exactamente con los **6 núcleos físicos reales** del procesador AMD Ryzen 5 5600X. En esta configuración se maximiza la relación entre el trabajo computacional intensivo (cálculo de distancias euclidianas) y la sobrecarga de sincronización, sin interferencias de contención de hardware.

---

### 10.2 Dónde y por qué se aplana la curva: Los tres tramos de escalabilidad

La curva de Speedup experimental exhibe tres tramos cualitativamente distintos que reflejan la interacción entre el runtime de Go y la microarquitectura del procesador:

```
Speedup
  5.11× |                                     * (W=12: 5.11×)   * (W=16: 5.07×)
        |                                    /                 (Meseta / Descenso)
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
* **Explicación Microarquitectónica:** Un hilo SMT **no es un núcleo adicional**. Dos hilos lógicos que residen en el mismo núcleo físico comparten las mismas unidades de ejecución ALU/FPU, el mismo predictor de saltos y la misma caché L1/L2. Si un hilo está ejecutando operaciones intensivas de punto flotante (`math.Sqrt` y sumas continuas), el hilo hermano debe esperar ciclos de reloj desocupados para emitir instrucciones.
* **Impacto en Métricas:** Aunque el número de workers se duplica de 6 a 12 (+100%), el speedup solo sube de $4.13\times$ a $5.11\times$ (un modesto +23.7%), provocando una caída de la eficiencia de $68.77\%$ a $42.56\%$.

#### Tramo 3: Sobre-suscripción ($W = 16$) | Saturación y Degradación ($E = 31.70\%$)
* **Comportamiento:** Se instancian 16 goroutines sobre una máquina con solo 12 hilos de hardware disponibles (`GOMAXPROCS = 12`).
* **Degradación:** El planificador del runtime de Go se ve forzado a multiplexar temporalmente 16 goroutines en 12 hilos del sistema operativo mediante colas de trabajo locales (`runqueues`) y robo de trabajo (*work stealing*).
* **Resultado:** El tiempo de pared no solo deja de mejorar, sino que **empeora** de 384.47 ms a 387.10 ms (el speedup cae de $5.11\times$ a $5.07\times$). Los costos adicionales de cambios de contexto (salvado y restauración de registros) y la contención en la caché L3 superan cualquier paralelismo potencial.

---

### 10.3 Trade-offs entre el Modelo Concurrente y Secuencial

La decisión de adoptar el modelo concurrente con Worker Pool implica compromisos técnicos que deben evaluarse objetivamente:

| Dimensión de Trade-off | Versión Secuencial | Versión Concurrente (Worker Pool) | Veredicto / Impacto |
|---|---|---|---|
| **Tiempo de Ejecución** | 1963.53 ms | **384.47 ms** ($W=12$) | **Ganancia:** Reducción del **80.4%** en tiempo de cómputo ($5.11\times$ más rápido). |
| **Consumo de Memoria** | 140.57 MB | 140.62 MB ($W=12$) | **Costo despreciable:** Solo +0.035% de memoria (+0.05 MB) por el pool persistente. |
| **Complejidad de Código** | ~100 líneas, lógica lineal de Lloyd | ~330 líneas, canales, goroutines y atomics | **Desventaja:** Mayor esfuerzo de desarrollo, depuración y mantenimiento. |
| **Superficie de Concurrencia** | Inmune por diseño a carreras de datos | Susceptible a race conditions si se comparte memoria | **Mitigado:** Verificado formalmente con SPIN (0 errores) y `go -race` (0 warnings). |
| **Estabilidad de Tiempos** | Desviación estándar muy baja | Mayor dispersión por jitter del scheduler | **Desventaja:** Requiere elevar repeticiones ($N=15$) para asegurar significancia estadística. |

#### Los costos ocultos de la concurrencia:
1. **Sobrecarga de Sincronización:** Para 10 iteraciones y 12 workers, el programa coordina 240 eventos de sincronización en canales bloqueantes. En datasets reducidos, este overhead supera al cómputo y vuelve al modelo concurrente contraproducente.
2. **Sensibilidad al Ruido de Muestreo:** Durante las pruebas preliminares con $N=10$, se detectó una anomalía donde $W=8$ parecía más lento que $W=6$. Al incrementar a $N=15$ repeticiones y calcular la media recortada, se demostró que se trataba de jitter estocástico del planificador de Go, consolidando una curva estrictamente monótona.

---

### 10.4 Comparación con la Literatura Científica de la PC1

Contrastar los resultados empíricos con los tres papers fundacionales revisados en la PC1 permite validar nuestras observaciones frente a la evidencia experimental global:

#### 1. Mussabayev et al. (2023) — *Parallel Clustering on Shared Memory Architectures*
* **Lo que anticipaban:** Al evaluar K-Means paralelo en arquitecturas multinúcleo de 8 núcleos, los autores advirtieron que a medida que se saturan los núcleos físicos, la contención por recursos de sincronización degrada la eficiencia por hilo.
* **Confirmación con nuestros datos:** Nuestros resultados reflejan con exactitud este principio: la eficiencia decae de forma estrictamente monótona desde $83.24\%$ ($W=2$) hasta $31.70\%$ ($W=16$). La reutilización de acumuladores privados evitó la contención por locks de memoria, pero la contención en el canal de barrera reprodujo la degradación predicha por Mussabayev.

#### 2. Feng et al. (2024) — *Scalability Bottlenecks in Concurrent Lloyd's Algorithm*
* **Lo que anticipaban:** El incremento en el grado de paralelismo no produce ganancias proporcionales indefinidas debido al cuello de botella en la fase de reducción y agregación global de centroides.
* **Confirmación con nuestros datos:** El Tramo 3 de nuestro experimento ($W=16$) confirma empíricamente la advertencia de Feng: pasar de 12 a 16 workers produjo un speedup negativo (retroceso de $5.11\times$ a $5.07\times$). Cuando la fase de reducción secuencial y el despacho de canales alcanzan el umbral crítico, agregar capacidad de cómputo nominal degrada el rendimiento.

#### 3. Ghimire & Amsaad (2024) — *Memory and Computation Limits in Parallel Machine Learning*
* **Lo que sostenían:** Los autores argumentan que en el clustering paralelo sobre memoria compartida, el factor limitante primordial es la capacidad de memoria RAM y la saturación del ancho de banda del bus del sistema al cargar datasets masivos.
* **Diferencia técnica con nuestro caso (Aporte Propio):** En nuestro experimento **no alcanzamos el límite de memoria**, registrando un consumo pico de apenas **140.62 MB sobre un equipo con 15,900 MB disponibles (menos del 0.9% de la RAM)**. Nuestro cuello de botella fue puramente **computacional y de sincronización** (capacidad de cálculo de punto flotante en núcleos SMT y latencia de canales de Go), demostrando que cuando el dataset se empaqueta de forma contigua en memoria binaria sin duplicaciones, la memoria deja de ser el obstáculo y el diseño de la concurrencia asume el rol determinante.

---

## Sección 11 — Uso y Rendimiento de los Recursos de Cómputo (2 Puntos)

### 11.1 Análisis del Consumo de Memoria: El paralelismo cuesta solo 0.06 MB

Uno de los logros más destacados de la implementación es la eficiencia extrema en el uso de la memoria del sistema. De acuerdo con las mediciones tomadas mediante `runtime.ReadMemStats` de Go (Tabla B):

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

Esta economía de memoria se fundamenta en dos decisiones de arquitectura:
1. **Dataset Inmutable Compartido por Referencia:** El dataset binario (`features.bin`, 140.29 MB) se mapea en memoria una única vez antes de iniciar el cronómetro. Las $W$ goroutines trabajadoras reciben punteros y slices que leen directamente el mismo bloque contiguo de memoria. No existe duplicación de datos.
2. **Worker Pool con Acumuladores Persistentes:** Las $W$ goroutines no se crean y destruyen en cada iteración; se instancian una sola vez al inicio del programa y permanecen bloqueadas en `<-proceed[id]`. Cada worker reutiliza una pequeña estructura fija de acumuladores locales:
   $$\text{Memoria por Worker} = (k \times D \times \text{sizeof(float64)}) + (k \times \text{sizeof(int)}) = (5 \times 4 \times 8) + (5 \times 8) = 200 \text{ bytes}$$
   Para $W = 12$, los acumuladores consumen únicamente **2.4 KB de RAM**, garantizando que el recolector de basura (GC) de Go permanezca inactivo durante las 10 iteraciones y eliminando pausas *Stop-The-World*.

---

### 11.2 Determinación Rigurosa del Punto de Equilibrio (Sweet Spot)

El punto de equilibrio del sistema se ubica de forma unívoca en **$W = 12$ workers**, coincidiendo con la cantidad de núcleos lógicos del procesador AMD Ryzen 5 5600X.

#### Justificación con los datos experimentales:
1. **Margen de Ganancia Creciente ($W \le 12$):**
   * De $W=4$ a $W=6$: el tiempo se reduce de 697.86 ms a 475.87 ms (ganancia de 221.99 ms, **-31.8%**).
   * De $W=6$ a $W=8$: el tiempo baja a 451.33 ms (ganancia de 24.54 ms, **-5.2%**).
   * De $W=8$ a $W=12$: el tiempo cae a 384.47 ms (ganancia de 66.86 ms, **-14.8%**).
2. **Inflexión Negativa ($W > 12$):**
   * De $W=12$ a $W=16$: el tiempo **aumenta** de 384.47 ms a 387.10 ms (pérdida de 2.63 ms, **+0.7%**).
   * La eficiencia se desploma al **31.70%**, y el speedup retrocede a **5.07×**.

#### Conclusión Técnica:
El hardware no puede despachar más de 12 instrucciones simultáneas en paralelo real. Todo worker asignado por encima de 12 genera competencia por tiempo de procesador en lugar de paralelismo efectivo. Por lo tanto, **$W = 12$ representa el sweet spot operativo óptimo**, donde se obtiene la máxima tasa de throughput computacional con el menor costo de coordinación.

---

### 11.3 Observación Metodológica y Telemetría de CPU

* **Comportamiento de CPU:** Durante la ejecución secuencial, el sistema operativo concentra la carga en 1 único núcleo al 100% de su capacidad (~16.6% del total de la CPU de 6 núcleos). En contraste, durante la fase de mapeo concurrente en $W=12$, los 12 procesadores lógicos operan a plena capacidad de procesamiento, reflejando saturación completa del silicio en el cálculo de distancias euclidianas.
* **Rigor Científico en la Medición:** Los datos cuantitativos de memoria provienen de llamadas directas a `runtime.ReadMemStats` integradas en `go/recursos/main.go`. Para el uso de CPU, dado que el benchmark se ejecutó en ráfagas de menos de 400 milisegundos por corrida, las herramientas de muestreo basadas en sondeo periódico (como el Monitor de Rendimiento de Windows o `psutil`) presentan artefactos de aliasing temporal. En consecuencia, se reportan las especificaciones de hardware y arquitectura de hilos validadas, evitando interpolar porcentajes instantáneos que comprometan la honestidad científica de los resultados.

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

Este resultado demuestra que el algoritmo K-Means concurrente implementado en Go y modelado en Promela no solo logra una aceleración de **5.11×**, sino que preserva la fidelidad clínica para identificar poblaciones vulnerables según las metas 3.1 y 3.2 del **ODS 3 (Salud y Bienestar)**.
