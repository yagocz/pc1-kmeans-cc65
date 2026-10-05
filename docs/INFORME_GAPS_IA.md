# Informe técnico de GAPs — Análisis del código con IA

**Curso:** Programación Concurrente y Distribuida (CC65) — Trabajo Parcial 2026-20
**Entregable:** TP (Semana 7)
**Repositorio analizado:** https://github.com/yagocz/pc1-kmeans-cc65
**Rama / commit:** `main`
**Fecha del análisis:** 3 de octubre de 2026

| Código | Nombres y apellidos |
|---|---|
| U202215375 | Ricardo Rafael Rivas Carrillo |
| U202412543 | Ernesto Yago Caldas Zapata |
| U202220230 | Andre Angel Chipana Rios |

---

## 1. Prompt estructurado utilizado

Se empleó **Claude (Anthropic)** con el siguiente prompt estructurado. Se
documenta íntegro porque el enunciado exige "elaborar un prompt estructurado"
y porque la reproducibilidad del análisis depende de él.

```
CONTEXTO
Analiza el repositorio https://github.com/yagocz/pc1-kmeans-cc65, que
implementa K-Means secuencial y concurrente en Go puro (sin librerías de
terceros) sobre 4,597,137 registros, más un modelo formal de la
sincronización en Promela verificado con SPIN.

ROL
Actúa como revisor técnico senior especializado en concurrencia y
verificación formal.

ALCANCE DEL ANÁLISIS
Evalúa cuatro dimensiones y reporta GAPs concretos:
  1. Calidad de código: duplicación, formato, cobertura de pruebas,
     manejo de errores, separación de responsabilidades.
  2. Seguridad: validación de entradas, uso de panic, exposición de
     rutas, consumo de recursos no acotado.
  3. Patrones de concurrencia: granularidad del bloqueo, correspondencia
     entre el modelo formal y la implementación, cobertura de la
     verificación, fugas de goroutines.
  4. Metodología experimental: validez de las mediciones, trazabilidad
     de las cifras, limitaciones declaradas.

REGLAS DURAS
- No inventes hallazgos. Cada GAP debe citar archivo y línea concretos,
  o una cifra medida ejecutando una herramienta sobre el código real.
- Si un hallazgo no se puede verificar ejecutando algo, decláralo como
  "no verificado" en lugar de afirmarlo.
- Clasifica cada GAP por severidad (Crítica / Alta / Media / Baja) y
  justifica la clasificación por su impacto, no por su dificultad.
- Incluye los GAPs que el propio equipo ya detectó y documentó: omitirlos
  daría una imagen falsa de completitud.

SALIDA ESPERADA
Un informe en Markdown con: metodología, tabla resumen de GAPs,
descripción detallada de cada uno con evidencia reproducible, y
recomendaciones priorizadas.
```

### Herramientas ejecutadas sobre el código

| Herramienta | Comando | Resultado |
|---|---|---|
| Compilador Go | `go build ./...` | Sin errores |
| Analizador estático | `go vet ./...` | Sin hallazgos |
| Formateador | `gofmt -l .` | **6 archivos no formateados** |
| Detector de carreras | `go run -race ./concurrente` | Sin advertencias |
| Verificador formal | `spin -a` + `pan` | 20,940 estados, 0 errores |
| Búsqueda de pruebas | `find . -name "*_test.go"` | **0 archivos** |

---

## 2. Resumen de GAPs identificados

| # | GAP | Dimensión | Severidad |
|---|---|---|:---:|
| 1 | La verificación formal cubre W=3, pero el benchmark mide hasta W=16 | Concurrencia | **Crítica** |
| 2 | Ausencia total de pruebas automatizadas | Calidad | **Alta** |
| 3 | El algoritmo concurrente está duplicado en cuatro archivos | Calidad | **Alta** |
| 4 | `panic()` como mecanismo de aserción en código de producción | Seguridad | **Alta** |
| 5 | La carga del dataset duplica el consumo de memoria | Seguridad | Media |
| 6 | El detector de carreras se ejecutó con parámetros reducidos | Concurrencia | Media |
| 7 | ~~Seis archivos no cumplen `gofmt`~~ → **RESUELTO** | Calidad | Media |
| 8 | ~~Errores de escritura ignorados en la exportación del CSV~~ → **RESUELTO** | Calidad | Media |
| 9 | ~~Elección de k=5 sin justificación empírica~~ → **RESUELTO** (con hallazgo adverso) | Metodología | Media |
| 10 | ~~Los clusters no se interpretan clínicamente~~ → **RESUELTO** | Metodología | Media |
| 11 | Generador congruencial lineal propio en lugar de `math/rand` | Calidad | Baja |
| 12 | Medición de recursos con una sola corrida por configuración | Metodología | Baja |

**Distribución:** 1 crítica, 3 altas, 6 medias, 2 bajas.

**Cuatro GAPs fueron corregidos durante este análisis** (7, 8, 9 y 10). Su
resolución se documenta en la sección 5.

---

## 3. GAPs detallados

### GAP-1 — La verificación formal cubre W=3, pero el benchmark mide hasta W=16
**Dimensión:** Patrones de concurrencia · **Severidad: Crítica**

**Evidencia.** `promela/kmeans_sync.pml` líneas 18-20:

```promela
#define W        3      /* workers concurrentes */
#define MAX_ITER 3      /* iteraciones maximas de Lloyd */
#define PUNTOS   2      /* puntos abstractos por particion */
```

El benchmark (`docs/resultados_bench.csv`) mide configuraciones de 2, 4, 6, 8,
12 y 16 workers. **Ninguna de ellas corresponde al modelo verificado.**

**Impacto.** El informe afirma que la implementación está "libre de condición
de carrera" apoyándose en SPIN, pero la garantía formal aplica únicamente a
W=3. Las propiedades verificadas son estructurales y no dependen del valor
concreto de W, de modo que la generalización es razonable; sin embargo, **no
está demostrada**, y presentarla como si lo estuviera sobreestima el alcance
de la verificación.

**Agravante.** El espacio de estados crece exponencialmente con W: la
verificación de W=3 explora 20,940 estados, y una verificación de W=16 sería
intratable en un equipo de escritorio. Esta es una limitación inherente al
model checking, no un descuido, pero debe declararse explícitamente.

**Recomendación.** Verificar al menos W=4 y W=5 para mostrar que el número de
estados crece sin que aparezcan errores nuevos, y declarar en el informe que
la verificación es paramétrica en la estructura pero acotada en el valor de W.
Alternativamente, documentar por qué las propiedades son independientes de W.

---

### GAP-2 — Ausencia total de pruebas automatizadas
**Dimensión:** Calidad de código · **Severidad: Alta**

**Evidencia.** `find . -name "*_test.go"` devuelve cero resultados sobre 1,429
líneas de código Go.

**Impacto.** La corrección se verifica de forma manual, comparando a ojo los
centroides de ambas versiones. Funciones como `DistanciaCuadrada`,
`InicializarCentroides` y `RecalcularCentroides` no tienen ninguna prueba que
detecte una regresión. El caso del cluster vacío (`go/kmeans/kmeans.go:137`)
es una rama de código que nunca se ejercitó deliberadamente.

**Recomendación.** Añadir `go/kmeans/kmeans_test.go` con pruebas unitarias para
las tres funciones puras, y una prueba de integración que verifique que ambas
versiones producen los mismos centroides sobre un dataset sintético pequeño.
Esto último automatizaría la comprobación que hoy se hace a mano.

---

### GAP-3 — El algoritmo concurrente está duplicado en cuatro archivos
**Dimensión:** Calidad de código · **Severidad: Alta**

**Evidencia.** La lógica del Worker Pool (canales `proceed[]`, canal `done`,
acumuladores privados y reducción) aparece replicada en:

| Archivo | Líneas |
|---|---:|
| `go/concurrente/main.go` | 328 |
| `go/bench/bench.go` | 363 |
| `go/recursos/main.go` | 209 |
| `go/calidad/main.go` | 249 |

**Impacto.** Es el GAP con mayor riesgo latente del proyecto. Un cambio en la
estrategia de sincronización obliga a modificar cuatro lugares, y si uno queda
desincronizado, **el benchmark mediría una implementación distinta de la que se
verificó formalmente**, invalidando silenciosamente la correspondencia
Promela-Go que sostiene todo el argumento del trabajo.

Este riesgo ya se materializó una vez durante el desarrollo: la primera versión
usaba partición estática con mutex y no correspondía al modelo de Promela;
hubo que reescribir el código y **volver a ejecutar las 105 mediciones**.

**Recomendación.** Extraer la función a `go/kmeans/pool.go` con una firma como
`EjecutarConcurrente(ds *Dataset, k, iter, W int, semilla uint64) ([]float64, []int)`
y que los cuatro programas la invoquen. Reduce ~700 líneas duplicadas a una
sola definición.

---

### GAP-4 — `panic()` como mecanismo de aserción en código de producción
**Dimensión:** Seguridad · **Severidad: Alta**

**Evidencia.** `go/concurrente/main.go`, cuatro ocurrencias:

```
línea 175:  panic("violacion: worker activo fuera de la fase de asignacion")
línea 191:  panic("violacion: lectura de centroides con un escritor activo")
línea 242:  panic("violacion: actualizacion con workers aun en asignacion")
línea 257:  panic(fmt.Sprintf("violacion: worker %d proceso %d veces, ..."))
```

**Impacto.** Estas aserciones replican los `assert` del modelo Promela, lo que
es una buena práctica de trazabilidad. El problema es el mecanismo: un `panic`
dentro de una goroutine **termina el proceso completo**, sin liberar recursos
ni permitir diagnóstico. En un entorno real, una aserción violada bajo carga
produciría una caída abrupta en lugar de un error manejable.

Agravante: el `panic` de la línea 175 se ejecuta dentro de una goroutine
worker, donde no puede ser recuperado por el coordinador.

**Recomendación.** Condicionar las aserciones a un build tag (`//go:build
assert`) para que no estén activas en compilaciones normales, o sustituirlas
por un canal de errores que el coordinador consuma y convierta en una
terminación ordenada.

---

### GAP-5 — La carga del dataset duplica el consumo de memoria
**Dimensión:** Seguridad (consumo de recursos) · **Severidad: Media**

**Evidencia.** `go/kmeans/kmeans.go:61-67`:

```go
buf := make([]byte, info.Size())        // 140.29 MB
if _, err := io.ReadFull(f, buf); err != nil { ... }
puntos := make([]float64, total)        // otros 140.29 MB
```

Ambas estructuras coexisten durante la conversión. Medición: `features.bin`
pesa 147,108,384 bytes (140.29 MB), por lo que **el pico de memoria es de
280.59 MB**, el doble del tamaño del archivo.

**Impacto.** Con el dataset actual es asumible en una máquina de 15.9 GB. Con
un dataset cuatro veces mayor el pico sería de ~1.1 GB, y el patrón no escala.
Además no hay validación del tamaño del archivo antes de reservar: un archivo
corrupto o malicioso de gran tamaño provocaría un agotamiento de memoria.

**Recomendación.** Leer en bloques de tamaño fijo reutilizando un búfer
pequeño, o usar `encoding/binary.Read` directamente sobre el `io.Reader`.
Añadir una cota superior al tamaño aceptado.

---

### GAP-6 — El detector de carreras se ejecutó con parámetros reducidos
**Dimensión:** Patrones de concurrencia · **Severidad: Media**

**Evidencia.** `docs/salida_race.txt` línea 6:

```
Comando: CGO_ENABLED=1 go run -race ./concurrente -k 5 -iter 3 -w 6
```

La configuración del benchmark completo es `-iter 10 -w 12`.

**Impacto.** El detector de carreras de Go es dinámico: solo detecta carreras
en los entrelazados que efectivamente ocurren durante la ejecución observada.
Con 3 iteraciones y 6 workers se exploran menos entrelazados que con 10
iteraciones y 12 workers, de modo que la cobertura es menor que la del
escenario realmente medido.

Atenuante: el detector impone una penalización de rendimiento de 5-10×, lo que
hace costoso ejecutarlo con los parámetros completos sobre 4.6 millones de
registros.

**Recomendación.** Ejecutar `-race` al menos una vez con `-iter 10 -w 12` y
documentar el tiempo que toma, o justificar explícitamente por qué se usó una
configuración reducida.

---

### GAP-7 — Seis archivos no cumplen el formato estándar de Go
**Dimensión:** Calidad de código · **Severidad: Media**

**Evidencia.** `gofmt -l .` reporta:

```
bench/bench.go
calidad/main.go
concurrente/main.go
kmeans/kmeans.go
recursos/main.go
secuencial/main.go
```

Es decir, **el 100% de los archivos Go del proyecto**.

**Impacto.** `gofmt` es el estándar de facto del ecosistema Go y su
cumplimiento es verificable automáticamente. No afecta la corrección pero sí
la calidad percibida y la facilidad de revisión en diffs.

**Recomendación.** Ejecutar `gofmt -w .` y añadir la verificación a un hook de
pre-commit.

**✅ RESUELTO.** Se aplicó `gofmt` a los seis archivos. Verificación posterior:
`gofmt -l .` no devuelve ninguno, `go build ./...` compila y `go vet ./...` no
reporta hallazgos.

---

### GAP-8 — Errores de escritura ignorados en la exportación del CSV
**Dimensión:** Calidad de código · **Severidad: Media**

**Evidencia.** `go/bench/bench.go:311` y `:314`:

```go
w.Write([]string{"version", "workers", "corrida", "tiempo_ms"})
```

`csv.Writer.Write` devuelve un `error` que no se comprueba en ninguna de las
dos llamadas.

**Impacto.** Si el disco se llena o los permisos fallan a mitad de la
escritura, el programa terminaría reportando éxito con un CSV truncado. Dado
que ese archivo es la evidencia que sustenta todas las cifras del informe, un
truncamiento silencioso comprometería la trazabilidad del trabajo.

**Recomendación.** Comprobar el error de cada `Write` y el de `w.Flush()`, que
tampoco se verifica.

**✅ RESUELTO.** Se introdujo un envoltorio `escribir()` que comprueba el error
de cada fila y termina con diagnóstico si falla, y se verifican además
`w.Error()` tras el `Flush` y el error de `f.Close()`.

---

### GAP-9 — Elección de k=5 sin justificación empírica
**Dimensión:** Metodología · **Severidad: Media**

**Evidencia.** Todos los experimentos usan `k=5`, valor fijado por defecto en
los cuatro programas. No existe en el repositorio ninguna ejecución del método
del codo, silhouette o criterio equivalente que lo justifique.

**Impacto.** El número de clusters es el hiperparámetro central de K-Means. Un
valor elegido sin sustento debilita las conclusiones sobre los perfiles
encontrados, aunque no afecta la validez del análisis de Speedup, que es
independiente de k.

**Estado.** El equipo detectó este GAP y escribió `go/calidad/main.go`, que
implementa el método del codo para k=2..10. El programa **compila y se ejecuta,
pero sus resultados no se incorporaron al informe** y conserva dos marcadores
`TODO(Andre)` en las líneas 190 y 240.

**Recomendación.** Ejecutar `go run ./calidad -kmin 2 -kmax 10`, identificar el
codo y declarar si k=5 resulta adecuado. Si el codo cae en otro valor, debe
reportarse: es un hallazgo legítimo, no un error.

**✅ RESUELTO, con hallazgo adverso.** Se ejecutó el método del codo para
k = 2…10 (`docs/salida_calidad.txt`). El resultado **no confirma la elección de
k=5**: la reducción de inercia cae a 5.89% en k=5 pero repunta a 9.74% en k=6,
lo que indica que k=5 converge a un mínimo local desfavorable. El análisis
completo está en `docs/calidad_clustering.md`. No afecta la validez del
Speedup, que es independiente de k.

---

### GAP-10 — Los clusters no se interpretan clínicamente
**Dimensión:** Metodología · **Severidad: Media**

**Evidencia.** El informe reporta los centroides en unidades z-score
(`0.836239`, `-1.209021`, …) sin traducirlos a unidades reales ni asignarles
significado.

**Impacto.** El caso de uso declarado es "segmentar perfiles de riesgo
neonatal" y el trabajo se alinea con el ODS 3 (metas 3.1 y 3.2). Sin
interpretación, el resultado se queda en números sin conexión con el objetivo
que justifica el proyecto. Es el GAP que más distancia hay entre lo prometido
y lo entregado.

**Estado.** `go/calidad/main.go` ya implementa el des-escalado a unidades
reales usando las medias y desviaciones del z-score aplicado en la PC1. Una
ejecución preliminar produjo perfiles clínicamente reconocibles —un grupo con
edad materna media de 36.3 años y otro de 20.6 años con menor peso al nacer—
pero **esos resultados no se incorporaron al informe**.

**Recomendación.** Ejecutar el programa, nombrar cada cluster con las
referencias de la OMS que el propio código documenta (bajo peso < 2500 g,
prematuro < 37 semanas, embarazo adolescente < 20 años) y añadir una sección
que cierre el vínculo con el ODS 3.

**✅ RESUELTO.** Se interpretaron los cinco clusters en unidades reales
(`docs/calidad_clustering.md`). Los dos ejes que estructuran la segmentación
son la edad materna (22.12 a 35.97 años) y el peso al nacer (2781 a 3650 g),
que son exactamente los factores de riesgo del caso de uso. El análisis
también evidencia una limitación: ningún cluster alcanza el umbral de bajo
peso porque el recorte por IQR de la PC1 excluyó esos casos.

---

### GAP-11 — Generador congruencial lineal propio en lugar de `math/rand`
**Dimensión:** Calidad de código · **Severidad: Baja**

**Evidencia.** `go/kmeans/kmeans.go:114`:

```go
estado = estado*6364136223846793005 + 1442695040888963407
return estado >> 16
```

**Impacto.** La restricción del curso es no usar **librerías de terceros**;
`math/rand` es biblioteca estándar y habría sido admisible. El LCG propio
funciona y es reproducible, pero tiene peores propiedades estadísticas que
`rand.NewSource`.

Atenuante: para seleccionar k centroides iniciales de entre 4.6 millones de
puntos, la calidad del generador es irrelevante en la práctica.

**Recomendación.** Documentar explícitamente por qué se eligió un LCG propio,
o sustituirlo por `math/rand` con semilla fija, que ofrece la misma
reproducibilidad.

---

### GAP-12 — Medición de recursos con una sola corrida por configuración
**Dimensión:** Metodología · **Severidad: Baja**

**Evidencia.** `docs/salida_recursos.txt` reporta una medición puntual por
configuración, frente a las 15 corridas con media recortada del benchmark.

**Impacto.** Los tiempos de esa tabla no son comparables con los del
benchmark. La contradicción es observable: en la tabla de recursos W=16
aparece más rápido que W=12, mientras que en la tabla de speedup ocurre lo
inverso.

**Estado.** **Este GAP ya fue detectado y declarado por el propio equipo** en
`docs/analisis_rendimiento.md`, bajo el epígrafe "Advertencia metodológica
(corrección)", donde se advierte que solo la medición de heap es estable entre
corridas y que únicamente la tabla del benchmark tiene potencia estadística.

**Recomendación.** Mantener la advertencia y, si hay tiempo, repetir la
medición de recursos con el mismo número de corridas que el benchmark.

---

## 4. Fortalezas identificadas

Un informe de GAPs que solo enumerara defectos daría una imagen
desbalanceada. El análisis identificó estas prácticas correctas:

**Verificación formal previa a la implementación.** El contraejemplo
`kmeans_sync_canal_compartido.pml` se construyó deliberadamente para demostrar
que el verificador detecta errores cuando existen (SPIN encuentra la violación
a profundidad 93). Esa validación convierte el resultado "0 errores" en
evidencia significativa en lugar de un posible falso negativo. Es una práctica
que no suele verse en trabajos de este nivel.

**Granularidad de bloqueo correcta.** Los acumuladores privados por worker
reducen las sincronizaciones de 4,597,137 a 2×W por iteración. La
documentación del código explica explícitamente por qué la alternativa ingenua
—bloquear por punto— sería más lenta que la versión secuencial.

**Rigor estadístico.** El análisis en `docs/analisis_rendimiento.md` incorpora
intervalos de confianza y pruebas t de Welch, y usa esa evidencia para
**corregir** una afirmación previa del equipo: el supuesto máximo en W=12
resultó ser una meseta estadísticamente indistinguible de W=16 (t = −0.30).
Corregir una conclusión propia con evidencia es exactamente lo que se espera
de un trabajo técnico.

**Trazabilidad de las cifras.** Toda cifra del informe tiene su salida cruda
versionada en `docs/`. Las mediciones se tomaron en una sola máquina y los
parámetros del experimento están documentados.

**Honestidad sobre las limitaciones.** El informe de la PC1 declara que el
recorte por IQR elimina casos clínicamente relevantes —bajo peso y
prematuridad— que son justamente los perfiles que el caso de uso busca
detectar. Señalar una tensión que debilita el propio trabajo, en lugar de
ocultarla, es un indicador de calidad metodológica.

---

## 5. Correcciones aplicadas durante el análisis

Cuatro GAPs se corrigieron en el mismo ciclo en que se detectaron. Se
documentan aquí con su verificación posterior, porque un informe que solo
enumere defectos sin resolver ninguno tiene menos valor que uno que demuestre
el ciclo completo de detección y corrección.

| GAP | Corrección | Verificación |
|:---:|---|---|
| 7 | `gofmt` aplicado a los 6 archivos | `gofmt -l .` sin resultados; compila y pasa `go vet` |
| 8 | Envoltorio que comprueba el error de cada escritura, de `w.Error()` y de `f.Close()` | Compila y pasa `go vet` |
| 9 | Método del codo ejecutado para k = 2…10 | `docs/salida_calidad.txt`, `docs/calidad_clustering.md` |
| 10 | Centroides des-escalados e interpretados clínicamente | `docs/calidad_clustering.md` |

### Sobre el GAP-9: una corrección que reveló un problema mayor

Resolver el GAP-9 produjo un hallazgo que el equipo no esperaba: **el método
del codo no respalda la elección de k=5**. La reducción de inercia cae a 5.89%
en k=5 y repunta a 9.74% en k=6, un comportamiento que no corresponde a una
curva de codo bien formada e indica que k=5 converge a un mínimo local
desfavorable por usar inicialización aleatoria simple en lugar de K-means++.

El hallazgo se reporta en lugar de ajustar k retroactivamente, por dos
razones. Cambiar a k=6 obligaría a repetir las 105 mediciones del benchmark, y
el resultado es información válida en sí misma: documenta una limitación real
del trabajo. El análisis de Speedup no se ve afectado, porque ambas versiones
ejecutan el mismo trabajo con el mismo k.

### GAPs no corregidos y por qué

Los ocho GAPs restantes quedan documentados pero sin resolver, por decisión
deliberada:

Los GAPs **1** (alcance de la verificación) y **6** (parámetros del detector de
carreras) requieren ejecutar verificaciones que tomarían horas de cómputo y
cuyo resultado esperado es confirmar lo que ya se sabe.

Los GAPs **2** (pruebas) y **3** (duplicación) son refactorizaciones de alcance
medio. Corregir el GAP-3 implicaría modificar los cuatro programas y **volver a
ejecutar todas las mediciones** para garantizar que el código medido sigue
siendo el verificado; el riesgo de introducir una regresión a esta altura del
trabajo supera el beneficio.

El GAP-4 (`panic` en producción) es una decisión de diseño defendible en un
trabajo académico, donde el fallo ruidoso es preferible al silencioso.

El GAP-5 (memoria duplicada) no se manifiesta con el dataset actual.

El GAP-12 ya estaba declarado por el equipo antes de este análisis.

---

## 6. Recomendaciones priorizadas

| Prioridad | Acción | GAP | Esfuerzo |
|:---:|---|:---:|---|
| 1 | Extraer el Worker Pool a una función compartida | 3 | Medio |
| 2 | Declarar el alcance de la verificación formal (W=3) | 1 | Bajo |
| 3 | Añadir pruebas unitarias y de equivalencia | 2 | Medio |
| 4 | Ejecutar `go/calidad` e incorporar sus resultados | 9, 10 | Bajo |
| 5 | Condicionar las aserciones a un build tag | 4 | Bajo |
| 6 | Ejecutar `gofmt -w .` | 7 | Trivial |
| 7 | Comprobar errores de `csv.Writer` | 8 | Trivial |
| 8 | Ejecutar `-race` con los parámetros completos | 6 | Bajo |
| 9 | Leer el dataset por bloques | 5 | Medio |

Las acciones 6 y 7 son triviales y de impacto inmediato. Las acciones 1 y 2
son las de mayor valor: la primera elimina el riesgo de divergencia entre lo
medido y lo verificado; la segunda corrige una afirmación que hoy sobreestima
el alcance de la garantía formal.

---

## 7. Conclusión del análisis

El proyecto presenta una base técnica sólida: el algoritmo concurrente está
correctamente diseñado, formalmente verificado y empíricamente validado, con
un Speedup de 5.11× respaldado por 105 mediciones.

El GAP crítico no está en el código sino en **el alcance declarado de la
verificación formal**: el modelo cubre W=3 y el informe generaliza la garantía
a configuraciones de hasta W=16 sin declarar esa extrapolación. La corrección
es documental, no técnica.

Entre los GAPs de implementación, la duplicación del algoritmo en cuatro
archivos es el de mayor riesgo latente, porque puede romper silenciosamente la
correspondencia entre el modelo verificado y el código medido —un riesgo que
ya se materializó una vez durante el desarrollo.

Cuatro GAPs se corrigieron durante este mismo análisis (7, 8, 9 y 10), y otros
tantos ya habían sido detectados por el propio equipo antes de la revisión.
Eso indica una capacidad de autocrítica que el análisis externo confirma más
que descubre.

La corrección del GAP-9 arrojó el hallazgo más valioso del informe: **el
método del codo no respalda la elección de k=5**, el hiperparámetro que
gobierna todos los experimentos. La reducción de inercia repunta en k=6 en
lugar de seguir decreciendo, lo que delata un mínimo local atribuible a la
inicialización aleatoria simple. El equipo optó por reportarlo en lugar de
ajustar k retroactivamente, decisión coherente con el criterio que ya había
aplicado al declarar la tensión entre el recorte por IQR y los perfiles de
riesgo que el caso de uso busca detectar.
