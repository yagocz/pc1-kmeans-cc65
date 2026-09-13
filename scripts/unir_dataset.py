"""
Reensambla el dataset limpio a partir de sus dos partes versionadas.

El dataset limpio pesa ~101.5 MB comprimido y GitHub rechaza archivos
mayores a 100 MB, por lo que se versiona partido en dos mitades:

    data/processed/dataset_limpio.part1.csv.gz
    data/processed/dataset_limpio.part2.csv.gz

Este script las une en un unico archivo local:

    data/processed/dataset_limpio.csv.gz   (4,597,137 filas)

Uso:
    python scripts/unir_dataset.py
"""
import gzip
from pathlib import Path

import pandas as pd

RAIZ = Path(__file__).resolve().parents[1]
PROC = RAIZ / "data" / "processed"
PARTES = [PROC / "dataset_limpio.part1.csv.gz", PROC / "dataset_limpio.part2.csv.gz"]
SALIDA = PROC / "dataset_limpio.csv.gz"

FILAS_ESPERADAS = 4_597_137


def main():
    faltan = [p.name for p in PARTES if not p.exists()]
    if faltan:
        raise SystemExit(f"Faltan las partes: {', '.join(faltan)}")

    n = 0
    with gzip.open(SALIDA, "wt", encoding="utf-8", newline="", compresslevel=9) as out:
        primero = True
        for parte in PARTES:
            print(f"[leyendo] {parte.name}")
            for chunk in pd.read_csv(parte, chunksize=250_000, low_memory=False):
                chunk.to_csv(out, index=False, header=primero)
                primero = False
                n += len(chunk)

    mb = SALIDA.stat().st_size / 1024 / 1024
    print(f"\n[ok] {SALIDA.name}  {mb:,.2f} MB  |  {n:,} filas")

    if n != FILAS_ESPERADAS:
        print(f"[ADVERTENCIA] se esperaban {FILAS_ESPERADAS:,} filas")
    else:
        print("[ok] conteo de filas verificado")


if __name__ == "__main__":
    main()
