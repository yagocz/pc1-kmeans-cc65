"""
PASO 2 - Exploracion del dataset original (EDA).

Procesa data/raw/minsa_nacidos_vivos.csv (~794 MB, 4.87M filas) por chunks
para no cargar todo en RAM. Reporta:
    - numero exacto de filas y columnas
    - tipos de dato
    - nulos por columna (incluye centinelas -1 y cadenas vacias)
    - duplicados exactos
    - rango / media / desviacion de cada columna numerica
    - cardinalidad de cada columna categorica

Toda la salida se guarda en docs/salida_exploracion.txt para que las cifras
del informe sean reproducibles y verificables.

Uso:
    python scripts/exploracion.py
"""
import hashlib
from collections import Counter
from pathlib import Path

import numpy as np
import pandas as pd

RAIZ = Path(__file__).resolve().parents[1]
ENTRADA = RAIZ / "data" / "raw" / "minsa_nacidos_vivos.csv"
SALIDA = RAIZ / "docs" / "salida_exploracion.txt"

CHUNK = 250_000
SEP = ";"
ENCODING = "utf-8"

# Columnas que el CNV publica como numericas continuas.
# Ojo: vienen como texto por padding (" 52  ") y centinelas (-1, ">=5").
NUMERICAS = ["PESO_NACIDO", "TALLA_NACIDO", "DUR_EMB_PARTO", "Edad_Madre"]

_lineas = []

# Contador de filas reparadas (388 en el corte 30-11-2025).
FILAS_REPARADAS = {"n": 0}

# Posicion 0-indexada de DESC_OCUPACION segun el HEADER REAL del CSV.
# OJO: el diccionario oficial la lista como item 18, pero en el archivo
# publicado ocupa la posicion 12. Manda el header real.
IDX_OCUPACION = 11


def _reparar_fila(campos):
    """
    Repara filas con un ';' literal dentro de DESC_OCUPACION.

    Ejemplo real (linea 4000 del CSV original):
        ...;VENDEDOR AMBULANTE; COSMETICOS, PERFUMES, LOCION, ETC.;2;...
                              ^ este ';' parte el campo en dos

    Como el ';' sobrante siempre cae dentro de la ocupacion (columna
    categorica que NO se usa como feature de K-Means), fusionamos los
    fragmentos y no se pierde ninguna fila ni ningun valor numerico.
    """
    sobrantes = len(campos) - 22
    if sobrantes <= 0:
        return None  # no sabemos repararla -> pandas la descarta
    fin = IDX_OCUPACION + sobrantes + 1
    fusion = ";".join(campos[IDX_OCUPACION:fin])
    FILAS_REPARADAS["n"] += 1
    return campos[:IDX_OCUPACION] + [fusion] + campos[fin:]


def log(txt=""):
    print(txt)
    _lineas.append(str(txt))


def main():
    if not ENTRADA.exists():
        raise SystemExit(
            f"No existe {ENTRADA}\nCorre primero: python scripts/descargar_dataset.py"
        )

    peso_mb = ENTRADA.stat().st_size / 1024 / 1024
    log("=" * 78)
    log("EXPLORACION - Registros de Nacidos Vivos (CNV) MINSA 2015-2025")
    log("=" * 78)
    log(f"Archivo     : {ENTRADA.name}")
    log(f"Peso        : {peso_mb:,.2f} MB")
    log(f"Separador   : '{SEP}'   Encoding: {ENCODING}")
    log()

    total_filas = 0
    columnas = None
    nulos = Counter()
    vacios = Counter()          # cadenas vacias o solo espacios
    centinela_neg1 = Counter()  # -1 usado como "sin dato"
    cardinalidad = {}           # col -> set de valores (solo categoricas)
    hashes = set()
    duplicados = 0
    # acumuladores para media/desv sin cargar todo en memoria
    acum = {c: {"n": 0, "suma": 0.0, "suma2": 0.0, "min": np.inf, "max": -np.inf}
            for c in NUMERICAS}

    lector = pd.read_csv(
        ENTRADA,
        sep=SEP,
        encoding=ENCODING,
        chunksize=CHUNK,
        dtype=str,          # leemos todo como texto: el casteo es parte de la limpieza
        keep_default_na=False,
        na_values=[],
        engine="python",
        on_bad_lines=_reparar_fila,
    )

    log("Procesando por chunks de {:,} filas...".format(CHUNK))
    for i, chunk in enumerate(lector, 1):
        if columnas is None:
            columnas = list(chunk.columns)
            for c in columnas:
                if c not in NUMERICAS:
                    cardinalidad[c] = set()

        total_filas += len(chunk)

        # Duplicados exactos: hash de la fila completa.
        # Usamos solo 8 bytes del digest (no los 16) para que el set quepa
        # en RAM: con 4.87M filas la diferencia es de ~500 MB a ~250 MB.
        for fila in chunk.itertuples(index=False, name=None):
            h = hashlib.md5("\x1f".join(fila).encode("utf-8")).digest()[:8]
            if h in hashes:
                duplicados += 1
            else:
                hashes.add(h)

        for c in columnas:
            s = chunk[c]
            vacios[c] += int((s.str.strip() == "").sum())
            centinela_neg1[c] += int((s.str.strip() == "-1").sum())

            if c in NUMERICAS:
                v = pd.to_numeric(s.str.strip(), errors="coerce")
                v = v.dropna()
                if len(v):
                    a = acum[c]
                    a["n"] += int(len(v))
                    a["suma"] += float(v.sum())
                    a["suma2"] += float((v.astype("float64") ** 2).sum())
                    a["min"] = min(a["min"], float(v.min()))
                    a["max"] = max(a["max"], float(v.max()))
            else:
                # limitamos la cardinalidad para no explotar en RAM
                if len(cardinalidad[c]) < 50_000:
                    cardinalidad[c].update(s.str.strip().unique().tolist())

        if i % 4 == 0:
            print(f"    ... {total_filas:,} filas")

    log()
    log("-" * 78)
    log("1) DIMENSIONES")
    log("-" * 78)
    log(f"Filas de datos : {total_filas:,}")
    log(f"Columnas       : {len(columnas)}")
    log(f"Supera 1,000,000 de registros: {'SI' if total_filas > 1_000_000 else 'NO'}")
    log()
    log(f"Filas con ';' literal dentro de DESC_OCUPACION (reparadas, no "
        f"descartadas): {FILAS_REPARADAS['n']:,}")
    log()

    log("-" * 78)
    log("2) COLUMNAS Y TIPO DECLARADO")
    log("-" * 78)
    for c in columnas:
        tipo = "numerica continua" if c in NUMERICAS else "categorica / texto"
        log(f"  {c:<28} {tipo}")
    log()

    log("-" * 78)
    log("3) VALORES FALTANTES POR COLUMNA")
    log("-" * 78)
    log(f"{'columna':<28}{'vacios':>12}{'%':>8}{'valor -1':>12}{'%':>8}")
    for c in columnas:
        pv = vacios[c] / total_filas * 100
        pc = centinela_neg1[c] / total_filas * 100
        log(f"{c:<28}{vacios[c]:>12,}{pv:>7.2f}%{centinela_neg1[c]:>12,}{pc:>7.2f}%")
    log()
    log("NOTA: el CNV usa -1 como codigo de 'sin dato', no como valor real.")
    log()

    log("-" * 78)
    log("4) DUPLICADOS EXACTOS (fila completa identica)")
    log("-" * 78)
    log(f"Filas duplicadas : {duplicados:,}  ({duplicados/total_filas*100:.2f}%)")
    log(f"Filas unicas     : {total_filas - duplicados:,}")
    log()

    log("-" * 78)
    log("5) ESTADISTICAS DE COLUMNAS NUMERICAS (sobre valores parseables)")
    log("-" * 78)
    log(f"{'columna':<20}{'n':>12}{'min':>12}{'max':>12}{'media':>12}{'desv':>12}")
    for c in NUMERICAS:
        a = acum[c]
        if a["n"] == 0:
            log(f"{c:<20}{'sin datos':>12}")
            continue
        media = a["suma"] / a["n"]
        var = max(a["suma2"] / a["n"] - media ** 2, 0.0)
        desv = var ** 0.5
        log(f"{c:<20}{a['n']:>12,}{a['min']:>12,.1f}{a['max']:>12,.1f}"
            f"{media:>12,.2f}{desv:>12,.2f}")
    log()
    log("NOTA: los min/max extremos confirman la necesidad de tratar outliers")
    log("      (ej. pesos de 0 g o gestaciones imposibles).")
    log()

    log("-" * 78)
    log("6) CARDINALIDAD DE COLUMNAS CATEGORICAS")
    log("-" * 78)
    log(f"{'columna':<28}{'valores distintos':>20}")
    for c in columnas:
        if c in NUMERICAS:
            continue
        n = len(cardinalidad[c])
        tope = " (tope 50k)" if n >= 50_000 else ""
        log(f"{c:<28}{n:>20,}{tope}")
    log()
    log("=" * 78)
    log("FIN DE LA EXPLORACION")
    log("=" * 78)

    SALIDA.parent.mkdir(parents=True, exist_ok=True)
    SALIDA.write_text("\n".join(_lineas), encoding="utf-8")
    print(f"\n[ok] Salida guardada en {SALIDA}")


if __name__ == "__main__":
    main()
