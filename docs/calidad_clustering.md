# Calidad del clustering: elección de k y perfiles clínicos

> Cifras obtenidas ejecutando `go run ./calidad -kmin 2 -kmax 10 -kperfil 5
> -iter 10 -w 12` sobre los 4,597,137 registros del dataset limpio.
> Salida cruda en `docs/salida_calidad.txt`.

Este documento cierra los GAP-9 y GAP-10 identificados en
`docs/INFORME_GAPS_IA.md`.

---

## 1. Justificación de k mediante el método del codo

Se ejecutó K-Means para k = 2 … 10 registrando la inercia (función objetivo
MSSC) de cada configuración. El "codo" es el valor de k a partir del cual
agregar clusters deja de reducir significativamente la inercia.

| k | Inercia (MSSC) | Reducción absoluta | Reducción % |
|---:|---:|---:|---:|
| 2 | 12,647,122.05 | — | — |
| 3 | 10,506,074.34 | 2,141,047.71 | 16.93% |
| 4 | 8,958,485.06 | 1,547,589.28 | 14.73% |
| **5** | **8,431,227.57** | **527,257.49** | **5.89%** |
| 6 | 7,610,227.08 | 821,000.49 | **9.74%** |
| 7 | 6,977,524.18 | 632,702.90 | 8.31% |
| 8 | 6,558,768.18 | 418,756.00 | 6.00% |
| 9 | 6,173,619.81 | 385,148.38 | 5.87% |
| 10 | 5,970,483.85 | 203,135.96 | 3.29% |

### Hallazgo: la curva no presenta un codo limpio en k=5

El resultado **no confirma la elección de k=5** que se usó en todos los
experimentos del trabajo. La curva se comporta de forma irregular:

La reducción decae de 16.93% (k=3) a 14.73% (k=4) y cae bruscamente a 5.89%
en k=5, lo que sugeriría un codo en k=4. Sin embargo, **en k=6 la reducción
vuelve a subir a 9.74%**, casi el doble que en k=5. Una curva de codo bien
comportada decrece monótonamente; aquí no lo hace.

La interpretación más razonable es que **k=5 cae en un mínimo local poco
favorable**. La inicialización de centroides usa una semilla fija (42) y no
implementa K-means++, de modo que para k=5 el algoritmo converge a una
solución peor que la que obtendría con una inicialización distinta. El repunte
en k=6 indica que la estructura real de los datos admite al menos seis grupos
diferenciados.

### Consecuencias para el trabajo

**Para el análisis de Speedup: ninguna.** El valor de k afecta el número de
distancias calculadas por iteración, pero tanto la versión secuencial como la
concurrente ejecutan exactamente el mismo trabajo con el mismo k. El Speedup
de 5.11× y todo el análisis de escalabilidad permanecen válidos.

**Para la interpretación de los clusters: sí tiene consecuencias.** Los
perfiles de la sección 2 corresponden a k=5, que según este análisis no es el
número óptimo de grupos. Los perfiles siguen siendo clínicamente coherentes,
pero una segmentación con k=6 o k=7 probablemente separaría mejor los casos de
riesgo.

**Se reporta este hallazgo en lugar de ajustar k retroactivamente.** Cambiar a
k=6 obligaría a repetir las 105 mediciones del benchmark, y el resultado del
método del codo es información válida por sí misma: documenta una limitación
real del trabajo en lugar de ocultarla.

### Recomendación

Para una continuación del trabajo: implementar K-means++ como estrategia de
inicialización y repetir el método del codo. La irregularidad observada es un
síntoma conocido de la inicialización aleatoria simple, y los tres papers
revisados en la PC1 coinciden en señalarla. Mussabayev et al. (2023) usan
K-means++ precisamente para evitarlo.

---

## 2. Perfiles clínicos de los clusters (k = 5)

Los centroides se devolvieron a unidades reales aplicando la transformación
inversa del escalado z-score (`valor_real = valor_escalado × desviación +
media`), usando los parámetros calculados durante la limpieza de la PC1.

| Cluster | Registros | % | Peso (g) | Talla (cm) | Gestación (sem) | Edad madre (años) |
|:---:|---:|---:|---:|---:|---:|---:|
| 0 | 1,069,761 | 23.27% | 3650.06 | 50.92 | 39.70 | 24.17 |
| 1 | 658,842 | 14.33% | 2781.21 | 47.19 | 37.37 | 32.26 |
| 2 | 866,990 | 18.86% | 3324.39 | 49.77 | 38.03 | 28.46 |
| 3 | 991,181 | 21.56% | 3536.39 | 50.29 | 39.39 | 35.97 |
| 4 | 1,010,363 | 21.98% | 2991.25 | 48.18 | 39.01 | 22.12 |

Referencias clínicas aplicadas (Organización Mundial de la Salud): bajo peso
al nacer < 2500 g; peso normal 2500–4000 g; macrosomía > 4000 g; prematuro
< 37 semanas; a término 37–41 semanas; embarazo adolescente < 20 años; edad
materna avanzada ≥ 35 años.

### Interpretación

**Cluster 0 — Gestación a término con peso alto, madres jóvenes adultas**
(23.27%). Peso de 3650 g en el tramo superior del rango normal, 39.7 semanas
de gestación y madres de 24 años en promedio. Es el grupo con mejores
indicadores antropométricos del conjunto.

**Cluster 1 — Peso bajo en el límite y gestación más corta, madres adultas**
(14.33%). El perfil de mayor interés clínico: 2781 g es el peso medio más bajo
de los cinco grupos y se acerca al umbral de bajo peso al nacer (2500 g), con
37.37 semanas de gestación, apenas por encima del límite de prematuridad. Las
madres promedian 32.26 años. Aunque las medias se mantienen dentro del rango
normal, **la cercanía simultánea a dos umbrales de riesgo distingue a este
grupo**, y es el único cuya media de gestación queda por debajo de 38 semanas.

**Cluster 2 — Perfil intermedio a término** (18.86%). Valores próximos a la
media general del dataset en las cuatro variables: 3324 g, 38.03 semanas,
madres de 28.46 años. Es el grupo de referencia contra el cual contrastar los
demás.

**Cluster 3 — Gestación a término, edad materna avanzada** (21.56%). Las
madres promedian **35.97 años**, justo en el umbral de edad materna avanzada
de la OMS. Los indicadores del recién nacido son favorables (3536 g, 39.39
semanas), lo que sugiere que en este grupo el factor de riesgo es la edad
materna y no el resultado neonatal.

**Cluster 4 — Peso moderadamente bajo, madres jóvenes** (21.98%). Madres de
22.12 años, el promedio más bajo del conjunto, con recién nacidos de 2991 g,
unos 650 g por debajo del cluster 0 pese a tener una duración de gestación
similar (39.01 frente a 39.70 semanas). **La diferencia de peso no se explica
por prematuridad**, lo que apunta a otros factores asociados a la maternidad
temprana.

### Lectura conjunta

K-Means separó los nacimientos en grupos con significado clínico reconocible y
no en particiones arbitrarias. Los dos ejes que estructuran la segmentación
son la **edad materna** (de 22.12 años en el cluster 4 a 35.97 en el cluster 3)
y el **peso al nacer** (de 2781 g a 3650 g), que son precisamente los dos
factores que el caso de uso identificó como predictores de riesgo neonatal.

Resulta notable que los clusters 3 y 4 ocupen los extremos opuestos de edad
materna —avanzada y joven— y sumen juntos el 43.5% de los nacimientos. Ambos
corresponden a los rangos etarios que la literatura obstétrica asocia con
mayor riesgo.

### Alineamiento con el ODS 3

El análisis entrega el insumo que el caso de uso prometía: una segmentación
automática de 4.6 millones de nacimientos que identifica qué combinaciones de
peso, talla, gestación y edad materna concentran características de riesgo.

Para las metas 3.1 (mortalidad materna) y 3.2 (mortalidad neonatal evitable),
los clusters 1 y 4 son los de mayor interés para focalizar programas de
control prenatal: el primero por su cercanía simultánea a los umbrales de bajo
peso y prematuridad, el segundo por concentrar maternidad temprana con menor
peso al nacer sin que medie una gestación más corta.

### Limitación que condiciona esta interpretación

El recorte por IQR aplicado en la limpieza de la PC1 eliminó 271,233 registros
(5.57%), entre los que se encuentran los casos de bajo peso real (< 2055 g) y
prematuridad (< 35 semanas). **Los perfiles descritos corresponden al núcleo
central de la distribución, no a los casos extremos de riesgo**, que fueron
excluidos del análisis precisamente por ser estadísticamente atípicos.

Esta tensión ya se había declarado en el informe de la PC1 y los resultados
aquí presentados la confirman: ningún cluster alcanza el umbral de bajo peso
al nacer, no porque esos casos no existan en el registro original, sino porque
fueron removidos antes del clustering. Una continuación del trabajo debería
repetir el análisis sin el recorte por IQR, o con k = 3.0, para capturar los
perfiles de riesgo en sus valores reales.
