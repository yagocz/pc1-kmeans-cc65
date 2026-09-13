# PC1 — Programación Concurrente y Distribuida (CC65)

Repositorio del Trabajo Parcial 2026-20.
Modelo de Machine Learning asignado: **K-Means**.

## Integrantes

| Código | Nombres y apellidos |
|---|---|
| U202215375 | Ricardo Rafael Rivas Carrillo |
| U202412543 | Ernesto Yago Caldas Zapata |
| U202220230 | Andre Angel Chipana Rios |

Curso: Programación Concurrente y Distribuida (CC65) — Carrera de Ciencias de la Computación
Profesor: Herminio Paucar Curasma

## Caso de uso

Segmentación de perfiles de riesgo neonatal en el Perú mediante **K-Means**, a
partir de los registros del Certificado de Nacido Vivo (CNV) del MINSA.

Se agrupan los nacimientos según **peso al nacer, talla, duración de la gestación
y edad de la madre** para identificar perfiles diferenciados (prematuridad, bajo
peso al nacer, embarazo adolescente, gestación a término normal).

**ODS 3 — Salud y bienestar**, metas 3.1 (reducir la mortalidad materna) y
3.2 (poner fin a las muertes evitables de recién nacidos).

## Dataset

| Campo | Valor |
|---|---|
| Nombre | Registros de Nacidos Vivos en el Perú (2015–2025) |
| Fuente | Ministerio de Salud (MINSA) |
| Plataforma | Plataforma Nacional de Datos Abiertos |
| URL | https://www.datosabiertos.gob.pe/dataset/registros-de-nacidos-vivos-en-el-per%C3%BA-2015%E2%80%932025 |
| Archivo | `CNV_MINSA_CORTE_30112025.csv` |
| Licencia | Open Data Commons Attribution License |
| Registros | **4,874,510** (supera el mínimo de 1,000,000) |
| Columnas | 22 |
| Peso | 793.79 MB |

> El CSV original **no está versionado** en este repositorio: GitHub rechaza
> archivos mayores a 100 MB. Se reconstruye con `scripts/descargar_dataset.py`.

## Estructura del repositorio

```
data/
  raw/          CSV original (793.79 MB) — IGNORADO por git
  processed/    dataset limpio partido en 2 .csv.gz — versionado
  sample/       muestra de 5,000 filas — versionada
scripts/
  descargar_dataset.py   descarga el CSV original
  exploracion.py         PASO 2 — EDA, cifras del estado inicial
  limpieza.py            PASO 3 — pipeline de limpieza en 6 pasos
  unir_dataset.py        reensambla el dataset limpio desde sus 2 partes
docs/
  informe_limpieza.md    informe con cifras reales
  salida_exploracion.txt salida cruda del EDA
  salida_limpieza.txt    salida cruda de la limpieza
```

> El dataset limpio pesa 101.5 MB comprimido, por encima del límite de 100 MB
> de GitHub, así que se versiona partido en `dataset_limpio.part1.csv.gz`
> (55.16 MB) y `dataset_limpio.part2.csv.gz` (46.34 MB). Para reconstruirlo:
> `python scripts/unir_dataset.py`

## Cómo reproducir

Requiere Python 3 con `pandas` y `numpy`.

```bash
pip install pandas numpy

python scripts/descargar_dataset.py   # descarga ~794 MB
python scripts/exploracion.py         # genera docs/salida_exploracion.txt
python scripts/limpieza.py            # genera el dataset limpio
```

Si solo querés el dataset limpio sin reprocesar los 794 MB, basta con unir las
partes ya versionadas:

```bash
python scripts/unir_dataset.py
```

## Flujo de trabajo (Git Flow)

- `main` — versión estable de cada entregable
- `develop` — integración
- `feature/setup-repo` — estructura, README, .gitignore
- `feature/limpieza-dataset` — script de limpieza y dataset procesado
- `feature/eda-validacion` — exploración e informe de limpieza

## Siguientes entregables

- **PC2**: modelado en Promela + K-Means secuencial vs concurrente en Go
  (goroutines, `sync.Mutex`, `sync.WaitGroup`, Worker Pool) y cálculo de Speedup.
- **TP**: verificación formal en Spin, informe de GAPs y sustentación.
