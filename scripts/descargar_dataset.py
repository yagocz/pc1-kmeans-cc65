"""
Descarga el dataset original desde la Plataforma Nacional de Datos Abiertos.

El CSV pesa ~794 MB y esta excluido del repositorio via .gitignore
(GitHub rechaza archivos > 100 MB). Cada integrante debe correr este
script una vez para reconstruir data/raw/ en su maquina.

Uso:
    python scripts/descargar_dataset.py
"""
import hashlib
import sys
import urllib.request
from pathlib import Path

RAIZ = Path(__file__).resolve().parents[1]
DESTINO = RAIZ / "data" / "raw"

RECURSOS = [
    (
        "https://www.datosabiertos.gob.pe/sites/default/files/CNV_MINSA_CORTE_30112025.csv",
        "minsa_nacidos_vivos.csv",
    ),
    (
        "https://www.datosabiertos.gob.pe/sites/default/files/Lista_Ubigeos_INEI.csv",
        "Lista_Ubigeos_INEI.csv",
    ),
]


def _barra(bloques, tam_bloque, total):
    if total <= 0:
        return
    pct = min(bloques * tam_bloque / total * 100, 100)
    mb = bloques * tam_bloque / 1024 / 1024
    sys.stdout.write(f"\r    {pct:5.1f}%  ({mb:,.1f} MB)")
    sys.stdout.flush()


def main():
    DESTINO.mkdir(parents=True, exist_ok=True)
    for url, nombre in RECURSOS:
        salida = DESTINO / nombre
        if salida.exists():
            mb = salida.stat().st_size / 1024 / 1024
            print(f"[skip] {nombre} ya existe ({mb:,.2f} MB)")
            continue
        print(f"[descargando] {nombre}")
        urllib.request.urlretrieve(url, salida, reporthook=_barra)
        mb = salida.stat().st_size / 1024 / 1024
        print(f"\n[ok] {nombre}  {mb:,.2f} MB")

    print("\nListo. Ahora podes correr:")
    print("    python scripts/exploracion.py")
    print("    python scripts/limpieza.py")


if __name__ == "__main__":
    main()
