# Instrucciones de Git para la PC2 — CC65

> Archivo de trabajo interno del equipo.

El código de la PC2 ya está listo y **probado** en la carpeta local, pero
**sin commitear**. Cada integrante commitea sus propios archivos desde su
cuenta para que el historial acredite la participación de los tres (2 puntos
de la rúbrica del Entregable 2).

Repositorio: **https://github.com/yagocz/pc1-kmeans-cc65**

---

## Paso 0 — Partir de `develop`

Las tres ramas de la PC2 salen de `develop`, no de `main`:

```bash
git checkout develop
git pull origin develop
```

Cada uno configura su identidad antes de commitear (si no, el commit sale a
nombre de otro):

```bash
git config user.name "Su Nombre Completo"
git config user.email "sucorreo@upc.edu.pe"
```

---

## Reparto por rama

### `feature/go-secuencial` — Ricardo

Núcleo compartido y versión secuencial.

```
go/go.mod
go/kmeans/kmeans.go
go/secuencial/main.go
scripts/csv_a_binario.py
```

```bash
git checkout develop
git checkout -b feature/go-secuencial
git add go/go.mod go/kmeans/ go/secuencial/ scripts/csv_a_binario.py
git commit -m "K-Means secuencial en Go puro y nucleo compartido"
git push -u origin feature/go-secuencial
```

### `feature/go-concurrente` — Yago

Worker Pool, sincronización y evidencia de ausencia de carreras.

```
go/concurrente/main.go
docs/salida_ejecucion.txt
docs/salida_race.txt
.gitignore
```

```bash
git checkout develop
git checkout -b feature/go-concurrente
git add go/concurrente/ docs/salida_ejecucion.txt docs/salida_race.txt .gitignore
git commit -m "K-Means concurrente con Worker Pool, sync.Mutex y sync.WaitGroup"
git push -u origin feature/go-concurrente
```

### `feature/promela-benchmark` — Andre

Modelo formal, benchmark, medición de recursos y gráficos.

```
promela/kmeans_sync.pml
go/bench/bench.go
go/recursos/main.go
scripts/graficos_speedup.py
docs/RESULTADOS_PC2.md
docs/resultados_bench.csv
docs/salida_bench.txt
docs/salida_spin.txt
docs/salida_recursos.txt
docs/grafico_speedup.png
docs/grafico_eficiencia.png
docs/grafico_tiempos.png
README.md
```

```bash
git checkout develop
git checkout -b feature/promela-benchmark
git add promela/ go/bench/ go/recursos/ scripts/graficos_speedup.py \
        docs/RESULTADOS_PC2.md docs/resultados_bench.csv docs/salida_bench.txt \
        docs/salida_spin.txt docs/salida_recursos.txt docs/*.png README.md
git commit -m "Modelo Promela, benchmark de Speedup y analisis de recursos"
git push -u origin feature/promela-benchmark
```

---

## Merges (Ricardo)

```bash
git checkout develop
git merge feature/go-secuencial     --no-ff -m "Merge feature/go-secuencial"
git merge feature/go-concurrente    --no-ff -m "Merge feature/go-concurrente"
git merge feature/promela-benchmark --no-ff -m "Merge feature/promela-benchmark"
git push origin develop

git checkout main
git merge develop --no-ff -m "Entregable PC2: Go secuencial vs concurrente, Promela y Speedup"
git push origin main
```

El flag `--no-ff` fuerza un commit de merge para que el grafo muestre la
estructura de Git Flow. Sin él Git aplana el historial y se pierde la
evidencia de las ramas.

---

## Captura para el informe

```bash
git log --graph --pretty=format:"%h %d %an: %s" --all
```

---

## Qué NO se commitea

Ya está cubierto por el `.gitignore`, pero para que nadie lo fuerce:

| Archivo | Peso | Por qué |
|---|---|---|
| `data/raw/minsa_nacidos_vivos.csv` | 794 MB | supera el límite de GitHub |
| `data/processed/dataset_limpio.csv.gz` | 101 MB | supera el límite de GitHub |
| `data/processed/features.bin` | 140 MB | se genera con `scripts/csv_a_binario.py` |
| `tools/spin.exe` | — | binario compilado |
| `promela/pan.*` | — | verificador generado por Spin |

---

## Si alguien quiere correr el código en su máquina

```bash
python scripts/unir_dataset.py     # reensambla el dataset limpio
python scripts/csv_a_binario.py    # genera features.bin (140 MB)

cd go
go run ./secuencial  -k 5 -iter 10 -semilla 42
go run ./concurrente -k 5 -iter 10 -semilla 42 -w 12
```

**Importante para el informe:** los benchmarks del informe salieron todos de
la misma máquina (Ryzen 5 5600X, 6 núcleos / 12 hilos). Si alguien corre el
benchmark en su laptop obtendrá otros números y **no** son comparables con la
tabla. Si se vuelve a medir, se remide todo en un solo equipo y se anotan sus
especificaciones.
