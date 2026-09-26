"""
Convierte el dataset limpio a un archivo binario de float64 con solo las
4 features escaladas, para que Go lo cargue sin parsear texto.

Motivo: leer y parsear 4,597,137 filas de CSV en Go toma un tiempo del
mismo orden que el propio K-Means. Si esa fase entra en la medicion,
contamina el Speedup con trabajo que NINGUNA de las dos versiones
paraleliza. Con un .bin la carga es un solo io.ReadFull.

Formato de salida: float64 little-endian, 4 valores por registro, en el
orden PESO_NACIDO_ESC, TALLA_NACIDO_ESC, DUR_EMB_PARTO_ESC, Edad_Madre_ESC.

Uso:
    python scripts/csv_a_binario.py
"""
import struct
from pathlib import Path

import pandas as pd

RAIZ = Path(__file__).resolve().parents[1]
ENTRADA = RAIZ / "data" / "processed" / "dataset_limpio.csv.gz"
SALIDA = RAIZ / "data" / "processed" / "features.bin"

FEATURES = ["PESO_NACIDO_ESC", "TALLA_NACIDO_ESC",
            "DUR_EMB_PARTO_ESC", "Edad_Madre_ESC"]
CHUNK = 500_000


def main():
    if not ENTRADA.exists():
        raise SystemExit(
            f"No existe {ENTRADA}\nCorre primero: python scripts/unir_dataset.py"
        )

    n = 0
    with open(SALIDA, "wb") as out:
        for chunk in pd.read_csv(ENTRADA, chunksize=CHUNK,
                                 usecols=FEATURES, low_memory=False):
            arr = chunk[FEATURES].to_numpy(dtype="<f8")
            out.write(arr.tobytes())
            n += len(chunk)
            print(f"    ... {n:,} filas", end="\r")

    mb = SALIDA.stat().st_size / 1024 / 1024
    esperado = n * len(FEATURES) * 8

    print(f"\n[ok] {SALIDA.name}")
    print(f"     filas    : {n:,}")
    print(f"     features : {len(FEATURES)}")
    print(f"     peso     : {mb:,.2f} MB")
    print(f"     bytes    : {SALIDA.stat().st_size:,} (esperado {esperado:,})")

    if SALIDA.stat().st_size != esperado:
        raise SystemExit("[ERROR] el tamanio no coincide con lo esperado")
    print("[ok] tamanio verificado")


if __name__ == "__main__":
    main()
