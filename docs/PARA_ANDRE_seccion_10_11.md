# Material para las secciones 10 y 11 del informe — Andre

> Este archivo es el **insumo** para que redactes las secciones 10 y 11.
> Los datos ya están medidos; lo que falta es el análisis escrito, que es
> justamente lo que evalúa la rúbrica (3 puntos la sección 10, 2 puntos la 11).
>
> **Borra este archivo del repo cuando termines** y deja en su lugar las
> secciones redactadas.

---

## Lo que ya está medido (no hay que volver a correr nada)

Máquina única para todas las corridas: **AMD Ryzen 5 5600X**, 6 núcleos
físicos / 12 lógicos, 15.9 GB RAM, Windows 11, Go 1.27.0, GOMAXPROCS=12.
Dataset: 4,597,137 registros × 4 dimensiones, k=5, 10 iteraciones fijas,
semilla 42, 15 corridas por configuración, media recortada.

### Tabla A — Resultados de la arquitectura final (canales)

| Workers | T recortada (ms) | Speedup | Eficiencia |
|---:|---:|---:|---:|
| Secuencial | 1963.53 | — | — |
| 2 | 1179.39 | 1.6649× | 83.24% |
| 4 | 697.86 | 2.8137× | 70.34% |
| 6 | 475.87 | 4.1262× | 68.77% |
| 8 | 451.33 | 4.3505× | 54.38% |
| **12** | **384.47** | **5.1071×** | 42.56% |
| 16 | 387.10 | 5.0724× | 31.70% |

### Tabla B — Uso de recursos

| Configuración | Reloj (ms) | Heap pico | Heap asignado |
|---|---:|---:|---:|
| Secuencial | 1692.78 | 140.57 MB | 0.00 MB |
| W=2 | 956.86 | 140.58 MB | 0.00 MB |
| W=4 | 652.45 | 140.58 MB | 0.01 MB |
| W=6 | 493.27 | 140.59 MB | 0.01 MB |
| W=8 | 504.03 | 140.60 MB | 0.01 MB |
| W=12 | 365.43 | 140.62 MB | 0.03 MB |
| W=16 | 344.60 | 140.63 MB | 0.02 MB |

---

## Sección 10 — Análisis de speedup, escalabilidad y trade-offs

Tres ejes a desarrollar. Los datos están arriba; lo que falta es la
interpretación.

### 10.1 Por qué el speedup no es lineal

**Ley de Amdahl.** Si `s` es la fracción secuencial del programa, el speedup
máximo alcanzable es `1/s` por más workers que se agreguen.

En nuestro K-Means la parte irreductiblemente secuencial es:
- la reducción de los acumuladores privados sobre las sumas globales,
- el recálculo de los centroides (`RecalcularCentroides`),
- el dispatch y la recolección de la barrera (2×W mensajes de canal).

**Cálculo con nuestros datos.** Despejando `s` de `S = 1 / (s + (1-s)/W)`,
es decir `s = (W/S - 1) / (W - 1)`, para cada configuración medida:

| Workers | Speedup medido | Fracción secuencial `s` | Techo teórico `1/s` |
|---:|---:|---:|---:|
| 2 | 1.6649× | 20.13% | 4.97× |
| 4 | 2.8137× | 14.05% | 7.12× |
| 6 | 4.1262× | 9.08% | 11.01× |
| 8 | 4.3505× | 11.98% | 8.34× |
| **12** | **5.1071×** | **12.27%** | **8.15×** |
| 16 | 5.0724× | 14.36% | 6.96× |

Con W=12, alrededor del **12% del tiempo es secuencial**, lo que implica un
techo de **8.15×** aunque tuviéramos infinitos workers. Estamos en 5.11×, o
sea al 63% de ese techo.

**Un detalle que vale la pena señalar en el informe:** la fracción `s` **no
es constante** — va de 9.08% en W=6 hasta 20.13% en W=2 y 14.36% en W=16.
Si Amdahl describiera el sistema perfectamente, `s` sería el mismo valor
para toda configuración. Que varíe significa que hay un costo de
sincronización que **crece con W** (los 2×W mensajes de canal por iteración),
y que el modelo de Amdahl puro, que supone una fracción secuencial fija, no
captura del todo nuestro caso. El mínimo en W=6 coincide con los 6 núcleos
físicos: es la configuración donde la relación entre trabajo útil y costo de
coordinación es más favorable.

### 10.2 Dónde y por qué se aplana la curva

La curva tiene tres tramos claramente diferenciados en la Tabla A:

| Tramo | Workers | Eficiencia | Qué está pasando |
|---|---|---|---|
| 1 | 2 → 6 | 83% → 69% | cada worker en un **núcleo físico** real |
| 2 | 8 → 12 | 54% → 43% | workers 7-12 en **hilos SMT**, comparten unidades de ejecución |
| 3 | 16 | 32% | **sobre-suscripción**: 16 goroutines / 12 hilos de hardware |

El Ryzen 5 5600X tiene 6 núcleos físicos con SMT (2 hilos por núcleo). Un
hilo SMT **no** equivale a un núcleo: comparte unidades de ejecución con su
hermano, así que aporta trabajo pero rinde menos. Eso explica que de W=6 a
W=12 el speedup solo suba de 4.13× a 5.11× (un 24% más) cuando el número de
workers se duplicó.

En W=16 el planificador de Go multiplexa 16 goroutines sobre 12 hilos del
sistema operativo. El costo de cambio de contexto más los mensajes de canal
supera la ganancia, y el tiempo se estanca (387.10 ms frente a 384.47 ms).

### 10.3 Trade-offs entre la versión concurrente y la secuencial

**Lo que se gana:** 5.11× de reducción en tiempo con un costo de memoria
despreciable (ver sección 11).

**Lo que se pierde:**

1. *Complejidad y superficie de error.* La versión secuencial no puede tener
   condiciones de carrera; la concurrente sí, y por eso hizo falta el modelo
   Promela y el detector `-race` para demostrar que no las tiene.

2. *Costo de sincronización.* Cada iteración intercambia 2×W mensajes de
   canal. Con W=16 y 10 iteraciones son 320 operaciones de canal, cada una
   con su sincronización en el runtime de Go.

3. *Varianza en las mediciones.* La desviación estándar de la versión
   concurrente es mayor que la de la secuencial. De hecho, una primera serie
   de 10 corridas produjo una anomalía (W=8 parecía más lento que W=6) que
   resultó ser ruido de muestreo; hubo que subir a 15 corridas para que la
   curva se volviera monótona.

### 10.4 Comparación con lo que anticipaban los papers (dato propio)

Este punto vale porque conecta la parte experimental con la bibliográfica de
la PC1.

**Mussabayev et al. (2023)** reportan que en hardware de 8 núcleos su esquema
competitivo reutilizaba peor la mejor solución previa que el esquema
secuencial-paralelizado, y anticipan que *a más workers, mayor contención*.
Nuestros datos lo confirman: la eficiencia cae monótonamente de 83% a 32%.

**Feng et al. (2024)** advierten que aumentar el paralelismo no garantiza
mejora proporcional por el costo de comunicación, sincronización y agregación
de resultados. Nuestro tramo 3 (W=16) es exactamente ese caso.

**Ghimire & Amsaad (2024)** señalan que el enfoque multinúcleo local topa
cuando el dataset supera la RAM física. En nuestro caso **no** llegamos a ese
límite: 140 MB en una máquina de 15.9 GB. Es una diferencia honesta que vale
la pena señalar: nuestro cuello de botella fue la sincronización, no la
memoria.

---

## Sección 11 — Uso y rendimiento de los recursos de cómputo

### 11.1 Memoria

El dato central: **el paralelismo cuesta 0.06 MB** (de 140.57 MB en la versión
secuencial a 140.63 MB con 16 workers). Es un **0.04% más de memoria** a
cambio de 5× menos tiempo.

Dos decisiones de diseño explican ese número tan bajo:

1. *El dataset se comparte por referencia.* Los 140 MB de `features.bin` se
   cargan una sola vez y todos los workers leen el mismo slice. Nunca se
   duplica.

2. *El pool es persistente.* Las W goroutines se crean una vez y sus
   acumuladores privados (20 `float64` + 5 `int` cada uno) se reutilizan en
   todas las iteraciones, en lugar de reasignarse en cada vuelta.

### 11.2 Punto de equilibrio

**W = 12**, que coincide exactamente con el número de núcleos lógicos.

Justificación con los datos, no por intuición:

- De W=8 a W=12 el tiempo baja de 451.33 a 384.47 ms (mejora del 15%).
- De W=12 a W=16 el tiempo **sube** de 384.47 a 387.10 ms.

Agregar workers más allá de 12 no aporta y empieza a costar. Ese es el punto
donde conviene dejar de sumar goroutines.

### 11.3 Observación sobre la medición de CPU

Los datos de la Tabla B se obtuvieron con `runtime.ReadMemStats` de la
biblioteca estándar. Si querés agregar utilización de CPU por configuración,
se puede medir con el Monitor de Recursos de Windows durante una corrida
larga, o dejarlo planteado como limitación de la medición actual. **Si no lo
medís, no lo inventes**: es preferible decir que no se midió a poner un
número estimado.

---

## Cómo commitear tu parte

```bash
git checkout develop
git pull origin develop
git checkout -b feature/analisis-rendimiento

# (acá editás/creás tus archivos)

git config user.name "Andre Angel Chipana Rios"
git config user.email "u202220230@upc.edu.pe"

git add docs/analisis_rendimiento.md
git commit -m "Analisis de speedup, escalabilidad y recursos de computo"
git push -u origin feature/analisis-rendimiento
```

Sugerencia: creá `docs/analisis_rendimiento.md` con tus secciones 10 y 11
redactadas. Así tu commit tiene contenido propio y no pisa lo que ya está.

---

## Importante

**No vuelvas a correr el benchmark en tu máquina.** Todas las cifras salieron
del mismo equipo (Ryzen 5 5600X). Si remedís en otra máquina los números
dejan de ser comparables entre sí y la tabla del informe pierde validez.

Si necesitás ver las salidas crudas:

| Archivo | Contenido |
|---|---|
| `docs/resultados_bench.csv` | los 105 tiempos individuales |
| `docs/salida_bench.txt` | salida completa del benchmark |
| `docs/salida_recursos.txt` | medición de memoria |
| `docs/RESULTADOS_PC2.md` | el informe técnico completo |
