# Informe de limpieza del dataset

**Curso:** Programación Concurrente y Distribuida (CC65) — Trabajo Parcial 2026-20
**Modelo de ML asignado:** K-Means
**Fecha de elaboración:** 13 de septiembre de 2026

| Código | Nombres y apellidos |
|---|---|
| U202215375 | Ricardo Rafael Rivas Carrillo |
| U202412543 | Ernesto Yago Caldas Zapata |
| U202220230 | Andre Angel Chipana Rios |

> Todas las cifras de este informe provienen de la ejecución real de
> `scripts/exploracion.py` y `scripts/limpieza.py` sobre el archivo original.
> Las salidas crudas están en `docs/salida_exploracion.txt` y
> `docs/salida_limpieza.txt`, y son reproducibles.

---

## 1. Ficha del dataset

| Campo | Valor |
|---|---|
| Nombre | Registros de Nacidos Vivos en el Perú (2015–2025) |
| Fuente | Ministerio de Salud del Perú (MINSA) |
| Sistema de origen | Certificado de Nacido Vivo (CNV) |
| Plataforma | Plataforma Nacional de Datos Abiertos |
| URL del dataset | https://www.datosabiertos.gob.pe/dataset/registros-de-nacidos-vivos-en-el-per%C3%BA-2015%E2%80%932025 |
| Archivo descargado | `CNV_MINSA_CORTE_30112025.csv` |
| Licencia | Open Data Commons Attribution License |
| Fecha de descarga | 13 de septiembre de 2026 |
| Cobertura temporal | 1 de enero de 2015 – 31 de agosto de 2025 |
| Frecuencia de actualización | Trimestral |
| **Filas originales** | **4,874,510** |
| **Columnas originales** | **22** |
| Peso del archivo | 793.79 MB (832,344,118 bytes) |
| Separador | `;` (punto y coma) |
| Codificación | UTF-8 |

El CSV original **no está versionado** en el repositorio porque GitHub rechaza
archivos mayores a 100 MB. Se reconstruye ejecutando
`python scripts/descargar_dataset.py`.

### Selección del dataset: alternativas descartadas

Antes de elegir este dataset se evaluaron y **descartaron por medición directa**
tres candidatos que no alcanzaban el mínimo de 1,000,000 de registros exigido:

| Dataset evaluado | Entidad | Filas medidas | Resultado |
|---|---|---:|---|
| Monitoreo de contaminantes del aire en Lima Metropolitana | SENAMHI | 577,794 | Descartado |
| Variables meteorológicas de estaciones automáticas | SENAMHI | 416,273 | Descartado |
| Lista de precios de combustibles (diaria) | OSINERGMIN | 497,156 | Descartado |
| **Registros de Nacidos Vivos (CNV)** | **MINSA** | **4,874,510** | **Seleccionado** |

---

## 2. Justificación del caso de uso y ODS

### Caso de uso

**Segmentación de perfiles de riesgo neonatal en el Perú mediante K-Means.**

Se agrupan los nacimientos registrados según cuatro variables antropométricas y
demográficas —peso al nacer, talla, duración de la gestación y edad de la madre—
con el fin de identificar perfiles diferenciados que hoy el sistema de salud
trata de forma agregada. Los agrupamientos esperados corresponden a situaciones
clínicamente reconocidas: prematuridad, bajo peso al nacer, embarazo adolescente
y gestación a término dentro de rangos normales.

El valor del análisis no es diagnóstico individual, sino **de política pública**:
conocer qué combinaciones de características concentran mayor riesgo permite
focalizar programas de control prenatal en los grupos y territorios donde el
problema es más denso.

### Alineamiento con los ODS

El caso se alinea con el **ODS 3 — Salud y bienestar**, específicamente:

- **Meta 3.1**: reducir la tasa mundial de mortalidad materna.
- **Meta 3.2**: poner fin a las muertes evitables de recién nacidos y de niños
  menores de 5 años.

El bajo peso al nacer y la prematuridad son los dos predictores más fuertes de
mortalidad neonatal. La edad materna en los extremos del rango (adolescente o
avanzada) es a su vez un factor reconocido de riesgo obstétrico. Un método que
segmente automáticamente 4.6 millones de nacimientos según estas variables
entrega insumos directos para las metas 3.1 y 3.2.

### Por qué K-Means es apropiado para estas variables

1. **Las cuatro features son numéricas continuas y están en escala de intervalo.**
   Peso (gramos), talla (centímetros), gestación (semanas) y edad materna (años)
   admiten diferencias y promedios con significado físico real. La distancia
   euclidiana entre dos nacimientos es interpretable: mide qué tan parecidos son
   clínicamente.

2. **No existen etiquetas previas de "perfil de riesgo".** El dataset no trae una
   columna de clase, por lo que el problema es de aprendizaje **no supervisado**.
   K-Means es el algoritmo de particionamiento estándar para este escenario.

3. **Los grupos esperados son compactos y de forma aproximadamente esférica en el
   espacio escalado.** Un recién nacido a término de peso normal se concentra
   alrededor de una región densa del espacio; K-Means asume justamente esa
   geometría (minimiza la suma de distancias cuadráticas intra-cluster, MSSC).

4. **El volumen justifica la paralelización**, que es el objetivo del curso. Con
   4,597,137 registros y 4 dimensiones, cada iteración de Lloyd exige calcular
   ~4.6 millones × k distancias. Es un costo suficientemente alto para que el
   contraste entre la versión secuencial y la concurrente en Go (goroutines,
   `sync.Mutex`, `sync.WaitGroup`, Worker Pool) produzca un Speedup medible, que
   es lo que se evaluará en la PC2.

**Limitación reconocida.** K-Means requiere fijar *k* de antemano y es sensible a
la inicialización de los centroides. En la PC2 se abordará con K-means++ y con el
método del codo, en línea con lo revisado en los papers de la sección
bibliográfica.

---

## 3. Estado inicial del dataset

Ejecución: `python scripts/exploracion.py` → `docs/salida_exploracion.txt`

- **Filas:** 4,874,510
- **Columnas:** 22
- **Celdas vacías:** 0 en todas las columnas (el CNV no usa cadenas vacías)

### Hallazgo 1 — El valor `-1` es un centinela, no un dato

El diccionario oficial del CNV (`docs/Diccionario_Datos_CNV.pdf`) define
explícitamente `-1` como **"ignorado"**. No es un valor medido y no puede
entrar a un cálculo de distancias.

| Columna | Registros con `-1` | % |
|---|---:|---:|
| `Hijos_fallec_madre` | 4,726,940 | 96.97% |
| `Num_embar_madre` | 49,841 | 1.02% |
| `Hijos_vivo_madre` | 37,530 | 0.77% |
| `IdUbigeoInei` | 8,640 | 0.18% |
| `TALLA_NACIDO` | 4,697 | 0.10% |
| `PESO_NACIDO` | 1,315 | 0.03% |
| `Ipress` | 1 | 0.00% |

`Hijos_fallec_madre` tiene **96.97% de valores ignorados**, lo que la vuelve
inservible para cualquier análisis cuantitativo. Este hallazgo confirmó la
decisión de **no** incluirla entre las features de K-Means.

En contraste, las cuatro features seleccionadas tienen una proporción de
centinelas mínima (máximo 0.10%), lo que hace viable **eliminar** las filas
afectadas en lugar de imputarlas.

### Hallazgo 2 — Valores imposibles en columnas numéricas

Estadísticas sobre el archivo crudo, antes de cualquier limpieza:

| Columna | n | mín | máx | media | desv. |
|---|---:|---:|---:|---:|---:|
| `PESO_NACIDO` | 4,874,510 | -1.0 | 8,000.0 | 3,248.09 | 528.73 |
| `TALLA_NACIDO` | 4,874,510 | -1.0 | 5,334.0 | 49.10 | 5.49 |
| `DUR_EMB_PARTO` | 4,874,510 | 3.0 | 99.0 | 38.62 | 1.72 |
| `Edad_Madre` | 4,874,510 | 0.0 | 62.0 | 28.30 | 6.94 |

Los máximos delatan errores de digitación en el registro de origen: una talla de
**5,334 cm** (53 metros), una gestación de **99 semanas** (casi dos años) y una
edad materna de **0 años** son físicamente imposibles. Estos valores justifican
un filtro de plausibilidad clínica **previo** al tratamiento estadístico de
outliers.

### Hallazgo 3 — Filas con `;` literal dentro de un campo de texto

388 filas (0.0080%) contienen un punto y coma dentro del campo
`DESC_OCUPACION`, sin comillas que lo protejan, lo que las parte en 23 campos en
lugar de 22. Ejemplo real (línea 4000 del archivo original):

```
...;VENDEDOR AMBULANTE; COSMETICOS, PERFUMES, LOCION, ETC.;2;...
                      ^ este ';' rompe el parseo
```

Todas las filas afectadas tienen exactamente 23 campos y el separador sobrante
cae siempre dentro de la ocupación. Al ser una columna categórica que **no** se
usa como feature, los fragmentos se pueden fusionar de forma determinista sin
perder ninguna fila ni alterar ningún valor numérico.

### Hallazgo 4 — Cardinalidad de las columnas categóricas

| Columna | Valores distintos |
|---|---:|
| `DESC_OCUPACION` | 2,875 |
| `Ipress` | 2,314 |
| `IdUbigeoInei` | 1,893 |
| `Pais_Madre` | 127 |
| `nacmuer_abort_madre` | 13 |
| `Atiende_Parto` | 13 |
| `FecNac_Mes` | 12 |
| `FecNac_Año` | 11 |
| `Nivel_Intrucción_Madre` | 11 |
| `Financiador_Parto` | 9 |
| `Estado_Civil` | 7 |
| `Num_embar_madre` / `Hijos_vivo_madre` / `Hijos_fallec_madre` | 6 |
| `Lugar_Nacido` | 6 |
| `Tipo_Parto` | 5 |
| `Condicion_Parto` | 4 |
| `sexo_nacido` | 3 |

La alta cardinalidad de `DESC_OCUPACION` (2,875 categorías de texto libre)
desaconseja su uso como feature: codificarla generaría miles de dimensiones
dispersas, sobre las cuales la distancia euclidiana pierde significado.

---

## 4. Procedimiento de limpieza

Ejecución: `python scripts/limpieza.py` → `docs/salida_limpieza.txt`
Tiempo de ejecución: 6.6 minutos.

El script procesa el archivo **en streaming, con dos pasadas y por chunks de
200,000 filas**, sin cargar nunca el dataset completo en memoria. Este diseño fue
necesario: concatenar 4.87 millones de filas de texto en un solo DataFrame excede
la RAM disponible en un equipo de 16 GB.

### Tabla de pasos

| # | Paso | Criterio aplicado | Filas antes | Eliminadas | Filas después |
|---|---|---|---:|---:|---:|
| 0 | Reparación de filas malformadas | Fusionar los fragmentos de `DESC_OCUPACION` partidos por un `;` literal | 4,874,510 | 0 *(388 reparadas)* | 4,874,510 |
| 1 | Duplicados exactos | Hash MD5 de la fila completa (22 columnas) | 4,874,510 | 867 | 4,873,643 |
| 2 | Nulos / centinelas | Eliminar filas con `-1` en cualquiera de las 4 features | 4,873,643 | 5,168 | 4,868,475 |
| 3 | Normalización de tipos | Quitar padding, castear a numérico, `strip` + mayúsculas en categóricas | 4,868,475 | 0 | 4,868,475 |
| 4a | Rango de plausibilidad clínica | Peso 300–7000 g; talla 20–70 cm; gestación 20–45 sem; edad 10–60 años | 4,868,475 | 105 | 4,868,370 |
| 4b | Outliers estadísticos | IQR con k = 1.5 sobre las 4 features | 4,868,370 | 271,233 | 4,597,137 |
| 5 | Selección de features | 4 variables continuas para K-Means | 4,597,137 | 0 | 4,597,137 |
| 6 | Escalado | Estandarización z-score | 4,597,137 | 0 | **4,597,137** |

**Total eliminado: 277,373 filas (5.69%). Retención: 94.31%.**

### Justificación de cada criterio

**Paso 0 — Reparar en lugar de descartar.** Las 388 filas malformadas podrían
haberse descartado, pero el error está confinado a una columna categórica que no
participa del modelo. Repararlas conserva 388 registros con sus cuatro features
numéricas intactas.

**Paso 1 — Duplicados exactos.** Se eliminan solo las filas idénticas en las 22
columnas. El conteo depende del conjunto de columnas considerado, por lo que se
reporta de forma transparente:

| Criterio de comparación | Duplicados detectados |
|---|---:|
| 22 columnas originales (criterio aplicado) | 974 |
| 11 columnas conservadas tras la selección | 4,182 |

El pipeline eliminó **867** porque deduplica sobre las 22 columnas dentro del
flujo por chunks, tras la reparación del paso 0. Se optó por el criterio más
conservador: dos nacimientos distintos pueden coincidir legítimamente en peso,
talla, gestación y edad materna sin ser el mismo registro, de modo que deduplicar
sobre el subconjunto reducido eliminaría datos válidos.

**Paso 2 — Eliminar en lugar de imputar.** Esta es la decisión metodológica más
relevante del procedimiento. Se eligió **eliminar** por dos razones:

1. *El costo es despreciable.* Las filas afectadas son 5,168 de 4.87 millones
   (0.11%). Eliminarlas no compromete el tamaño ni la representatividad.
2. *Imputar distorsionaría el modelo.* Sustituir un peso al nacer ignorado por la
   media (3,248 g) fabricaría un recién nacido "promedio" que nunca existió.
   Como K-Means ubica los centroides mediante promedios, cada valor imputado
   arrastraría artificialmente los centroides hacia el centro de la distribución,
   justamente donde se difuminan los perfiles de riesgo que el análisis busca
   detectar. En un estudio de riesgo neonatal, inventar datos en la variable
   crítica invalida el resultado.

**Paso 3 — Normalización.** `TALLA_NACIDO` se publica como texto con relleno de
espacios (`"52  "`), lo que impide operar con ella directamente. Se aplicó
`strip` y conversión numérica. En las columnas categóricas conservadas se
normalizaron espacios y mayúsculas para evitar que `" PERU "` y `"PERU"` se
traten como categorías distintas.

**Paso 4 — Outliers en dos etapas.** El orden es deliberado:

- *4a, rango clínico:* elimina valores físicamente imposibles (talla de 5,334 cm).
  Se ejecuta **primero** porque estos valores extremos inflan los cuartiles y
  desplazan los límites del IQR; calcular el IQR sobre datos corruptos produciría
  umbrales sin sentido.
- *4b, IQR con k = 1.5:* se eligió **IQR sobre z-score** porque el IQR se apoya en
  cuartiles, que son robustos frente a valores extremos. El z-score, en cambio,
  usa media y desviación estándar, ambas contaminadas por los mismos outliers que
  se pretende detectar — un dato aberrante ensancha la desviación y termina
  ocultándose bajo su propio umbral.

Límites calculados sobre el conjunto ya saneado:

| Feature | Q1 | Q3 | IQR | Límite inferior | Límite superior |
|---|---:|---:|---:|---:|---:|
| `PESO_NACIDO` | 2,970.00 | 3,580.00 | 610.00 | 2,055.00 | 4,495.00 |
| `TALLA_NACIDO` | 48.00 | 50.50 | 2.50 | 44.25 | 54.25 |
| `DUR_EMB_PARTO` | 38.00 | 40.00 | 2.00 | 35.00 | 43.00 |
| `Edad_Madre` | 23.00 | 33.00 | 10.00 | 8.00 | 48.00 |

Este paso eliminó 271,233 filas (5.57%), el grueso del total depurado.

> **Observación metodológica.** Aplicar el IQR recorta también casos clínicamente
> reales y relevantes: un recién nacido de 1,800 g (bajo peso) o una gestación de
> 33 semanas (prematuro) quedan fuera por caer bajo el límite inferior. Existe
> por tanto una tensión entre la depuración estadística y el caso de uso, ya que
> son precisamente los perfiles de riesgo que interesa detectar. Se mantuvo el
> criterio IQR por ser el método estándar exigido y porque conserva 4.59 millones
> de registros, pero se deja constancia de que para un estudio clínico definitivo
> convendría evaluar un k mayor (k = 3.0) o prescindir del recorte por IQR y
> confiar solo en el filtro de plausibilidad clínica.

**Paso 5 — Features finales.** Se seleccionaron las cuatro variables continuas:

| Feature | Unidad | Fuente de la definición |
|---|---|---|
| `PESO_NACIDO` | gramos | Diccionario oficial CNV |
| `TALLA_NACIDO` | centímetros (1 decimal) | Diccionario oficial CNV |
| `DUR_EMB_PARTO` | semanas de gestación | Diccionario oficial CNV |
| `Edad_Madre` | años | Diccionario oficial CNV |

Se excluyeron las columnas categóricas (no admiten distancia euclidiana), los
identificadores `Ipress` e `IdUbigeoInei` (son códigos, no magnitudes: la
diferencia entre dos ubigeos carece de significado numérico) y
`Hijos_fallec_madre` (96.97% de valores ignorados).

Se conservaron además siete columnas categóricas de contexto —`FecNac_Año`,
`sexo_nacido`, `Condicion_Parto`, `Tipo_Parto`, `Nivel_Intrucción_Madre`,
`Financiador_Parto`, `IdUbigeoInei`— que **no** son features del modelo pero
permitirán caracterizar e interpretar los clusters obtenidos en la PC2.

**Paso 6 — Escalado: estandarización z-score.**

K-Means minimiza la suma de distancias euclidianas cuadráticas al centroide. La
distancia euclidiana suma las diferencias al cuadrado de cada dimensión, de modo
que **una variable con rango numérico amplio domina el cálculo**. Con los datos
sin escalar:

- `PESO_NACIDO` varía en un rango de 2,440 unidades (2,055 – 4,495 g)
- `DUR_EMB_PARTO` varía en un rango de 8 unidades (35 – 43 semanas)

Una diferencia de 500 gramos aporta `500² = 250,000` a la distancia; una
diferencia de 4 semanas de gestación aporta `4² = 16`. Sin escalar, el peso
determinaría la totalidad del agrupamiento y la prematuridad sería invisible para
el algoritmo, pese a ser un factor de riesgo central en el caso de uso.

Se eligió **z-score** ( (x − μ) / σ ) sobre **min-max** porque min-max normaliza
en función del mínimo y el máximo observados, quedando condicionado por los
valores límite que sobrevivieron al recorte IQR. El z-score usa media y
desviación, que resumen el conjunto completo, y deja todas las features centradas
en 0 con dispersión 1 — es decir, con **el mismo peso** en el cálculo de la
distancia.

---

## 5. Resultado: dataset limpio

| Campo | Valor |
|---|---|
| Archivo | `data/processed/dataset_limpio.csv.gz` |
| Formato | CSV comprimido con gzip |
| **Filas** | **4,597,137** |
| Columnas | 15 (7 de contexto + 4 features + 4 features escaladas) |
| Peso comprimido | 101.50 MB |
| Versionado en GitHub | partido en `dataset_limpio.part1.csv.gz` (55.16 MB) y `dataset_limpio.part2.csv.gz` (46.34 MB) |
| Reensamblado | `python scripts/unir_dataset.py` |
| Muestra versionada | `data/sample/muestra_5000.csv` (5,000 filas, 0.77 MB) |

> **Nota sobre el versionado.** El archivo comprimido pesa 101.50 MB, por encima
> del límite de 100 MB por archivo que impone GitHub. Se recomprimió a nivel 9
> (bajó de 102.44 MB a 101.50 MB, insuficiente) y finalmente se optó por
> versionarlo partido en dos mitades, acompañadas de `scripts/unir_dataset.py`
> para reconstruirlo. El dataset no se recortó: las dos partes suman los
> 4,597,137 registros completos.

Verificación de integridad: el archivo comprimido fue releído por completo y
devolvió exactamente **4,597,137 filas** y 15 columnas, sin encabezados
duplicados.

### Estadísticas descriptivas — antes del escalado

| Feature | mín | máx | media | desviación |
|---|---:|---:|---:|---:|
| `PESO_NACIDO` | 2,055.00 | 4,495.00 | 3,294.8199 | 424.8124 |
| `TALLA_NACIDO` | 44.30 | 54.20 | 49.4305 | 1.7781 |
| `DUR_EMB_PARTO` | 35.00 | 43.00 | 38.8311 | 1.1894 |
| `Edad_Madre` | 10.00 | 48.00 | 28.2344 | 6.8913 |

### Estadísticas descriptivas — después del escalado (z-score)

| Feature | mín | máx | media | desviación |
|---|---:|---:|---:|---:|
| `PESO_NACIDO_ESC` | -2.9185 | 2.8252 | 0.000000 | 1.000000 |
| `TALLA_NACIDO_ESC` | -2.8855 | 2.6824 | 0.000000 | 1.000000 |
| `DUR_EMB_PARTO_ESC` | -3.2209 | 3.5050 | 0.000000 | 1.000000 |
| `Edad_Madre_ESC` | -2.6460 | 2.8682 | 0.000000 | 1.000000 |

Las cuatro features quedan centradas en media 0 y desviación 1, con rangos
comparables (aproximadamente −3 a +3). Ninguna domina el cálculo de la distancia
euclidiana: la condición necesaria para que K-Means agrupe por similitud real y
no por escala de medición.

### Comparación antes / después

| Indicador | Original | Limpio |
|---|---:|---:|
| Filas | 4,874,510 | 4,597,137 |
| Columnas | 22 | 15 |
| Peso | 793.79 MB | 101.50 MB (gzip) |
| Mínimo de `PESO_NACIDO` | -1.0 (centinela) | 2,055.00 |
| Máximo de `TALLA_NACIDO` | 5,334.0 (imposible) | 54.20 |
| Máximo de `DUR_EMB_PARTO` | 99.0 (imposible) | 43.00 |
| Mínimo de `Edad_Madre` | 0.0 (imposible) | 10.00 |

---

## 6. Confirmación del requisito de volumen

> **El dataset final conserva 4,597,137 registros.**
>
> Supera en **4.6 veces** el mínimo de 1,000,000 exigido por el enunciado del
> Trabajo Parcial, con un margen de **3,597,137 registros** por encima del
> umbral.

Cifra verificada por dos vías independientes: el conteo reportado por
`scripts/limpieza.py` durante la escritura, y la relectura completa del archivo
comprimido resultante.

---

## 7. Reproducibilidad

```bash
pip install pandas numpy pyarrow

python scripts/descargar_dataset.py   # descarga ~794 MB desde datosabiertos.gob.pe
python scripts/exploracion.py         # ~3.2 min  -> docs/salida_exploracion.txt
python scripts/limpieza.py            # ~6.6 min  -> data/processed/ + data/sample/
```

Entorno de ejecución: Windows 11, Python 3.14.2, pandas 3.0.5, numpy 2.4.2,
pyarrow 25.0.1, 16 GB de RAM.

---

## 8. Proyección hacia la PC2

El dataset queda preparado para la siguiente entrega:

- **Las features escaladas** (`*_ESC`) se cargan directamente en Go y se recorren
  como `[]float64`, sin necesidad de preprocesamiento adicional.
- **4,597,137 registros × 4 dimensiones** implican ~4.6 millones × k cálculos de
  distancia por iteración de Lloyd: carga suficiente para que el contraste entre
  la implementación secuencial y la concurrente arroje un Speedup medible.
- **El formato es paralelizable de forma natural.** Al ser filas independientes,
  la fase de asignación se reparte entre goroutines mediante un Worker Pool; solo
  la actualización de los centroides requiere sincronización con `sync.Mutex`, y
  `sync.WaitGroup` coordina el cierre de cada iteración. La sección crítica queda
  reducida a la acumulación de sumas parciales por cluster, lo que facilitará el
  modelado en Promela y la verificación en Spin del Entregable 3.
