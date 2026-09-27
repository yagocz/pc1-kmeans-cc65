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

## Entregable 2 (PC2) — K-Means secuencial vs concurrente en Go

Implementación del algoritmo de Lloyd en **Go puro** (solo biblioteca
estándar), en dos versiones, más el modelado de la sincronización en Promela.

```
go/
  kmeans/       nucleo compartido: carga, distancia, inicializacion
  secuencial/   K-Means de Lloyd en una sola goroutine (linea base)
  concurrente/  Worker Pool con sync.Mutex y sync.WaitGroup
  bench/        benchmark: 10 corridas por configuracion, media recortada
  recursos/     medicion de CPU y memoria por configuracion
promela/
  kmeans_sync.pml                    modelo verificado en SPIN
  kmeans_sync_canal_compartido.pml   contraejemplo deliberado
  verificar.sh                       ejecuta todas las verificaciones
docs/
  RESULTADOS_PC2.md      resultados completos con las cifras reales
  resultados_bench.csv   tiempos crudos de las 105 corridas
  grafico_speedup.png    Speedup vs workers, con la recta ideal
  grafico_eficiencia.png Eficiencia paralela
  grafico_tiempos.png    Tiempo por configuracion
  salida_bench.txt       salida cruda del benchmark
  salida_race.txt        salida del detector de carreras (-race)
  salida_spin.txt        salida de la simulacion en Spin
  salida_recursos.txt    uso de CPU y memoria
  salida_ejecucion.txt   evidencia de ejecucion de ambas versiones
```

### Resultados principales

| Workers | Tiempo (ms) | Speedup | Eficiencia |
|---:|---:|---:|---:|
| Secuencial | 1963.53 | — | — |
| 2 | 1179.39 | 1.66x | 83% |
| 4 | 697.86 | 2.81x | 70% |
| 6 | 475.87 | 4.13x | 69% |
| 8 | 451.33 | 4.35x | 54% |
| **12** | **384.47** | **5.11x** | 43% |
| 16 | 387.10 | 5.07x | 32% |

**Punto de equilibrio: W = 12**, que coincide con los nucleos logicos. En
W = 16 el Speedup se estanca por sobre-suscripcion (16 goroutines compitiendo
por 12 hilos de hardware) y por el costo de los 2xW mensajes de canal por
iteracion.

La arquitectura es la **verificada formalmente** en `promela/kmeans_sync.pml`:
Worker Pool persistente con dispatch por canales `proceed[id]` y barrera por
canal `done`, mas `sync.Mutex` sobre los centroides y `sync.WaitGroup` para el
cierre del pool. SPIN la verifica de forma exhaustiva: 20,940 estados,
0 errores en las tres propiedades LTL.

Maquina de referencia: AMD Ryzen 5 5600X (6 nucleos / 12 hilos), 15.9 GB RAM,
Windows 11, Go 1.27.0. Todas las corridas en el mismo equipo.

### Como ejecutar

```bash
python scripts/unir_dataset.py     # reensambla el dataset limpio
python scripts/csv_a_binario.py    # genera features.bin para Go

cd go
go run ./secuencial  -k 5 -iter 10 -semilla 42
go run ./concurrente -k 5 -iter 10 -semilla 42 -w 12
go run ./bench -k 5 -iter 10 -n 15
```

Detalle completo en [docs/RESULTADOS_PC2.md](docs/RESULTADOS_PC2.md).

## Siguientes entregables

- **PC2**: ✅ completado (ver sección anterior).
- **TP**: verificación formal en Spin, informe de GAPs y sustentación.
