"""
PASO 3 - Limpieza del dataset CNV (MINSA, Nacidos Vivos 2015-2025).

Procesa data/raw/minsa_nacidos_vivos.csv (~794 MB, 4,874,510 filas).

DISENIO EN STREAMING (dos pasadas, sin concatenar el dataset en RAM):
    Pasada 1 -> repara, deduplica, normaliza, trata nulos y rangos clinicos.
                Guarda SOLO las 4 columnas numericas necesarias para calcular
                los cuartiles del IQR. Escribe un parquet temporal por chunk.
    Pasada 2 -> relee los temporales, aplica el corte IQR, escala y escribe
                el .csv.gz final chunk por chunk.

Se evita `pd.concat` de las 22 columnas: con 4.87M filas de texto eso
supera la RAM disponible en una laptop tipica.

Pasos aplicados y contabilizados por separado:
    0. Reparacion de filas con ';' literal dentro de DESC_OCUPACION
    1. Eliminacion de duplicados exactos
    2. Tratamiento de nulos (centinela -1 = "ignorado" segun diccionario MINSA)
    3. Normalizacion de tipos y formatos
    4. Outliers: rango clinico + IQR
    5. Seleccion de features numericas finales para K-Means
    6. Escalado (estandarizacion z-score)

Uso:
    python scripts/limpieza.py
"""
import hashlib
import shutil
import tempfile
from pathlib import Path

import numpy as np
import pandas as pd

RAIZ = Path(__file__).resolve().parents[1]
ENTRADA = RAIZ / "data" / "raw" / "minsa_nacidos_vivos.csv"
SAL_PROC = RAIZ / "data" / "processed" / "dataset_limpio.csv.gz"
SAL_MUESTRA = RAIZ / "data" / "sample" / "muestra_5000.csv"
SAL_LOG = RAIZ / "docs" / "salida_limpieza.txt"

CHUNK = 200_000
SEP = ";"
ENCODING = "utf-8"
IDX_OCUPACION = 11
N_MUESTRA = 5_000
SEMILLA = 42

# Features continuas confirmadas por el diccionario oficial del CNV:
#   PESO_NACIDO   -> gramos
#   TALLA_NACIDO  -> centimetros (un decimal)
#   DUR_EMB_PARTO -> semanas de gestacion
#   Edad_Madre    -> anios
FEATURES = ["PESO_NACIDO", "TALLA_NACIDO", "DUR_EMB_PARTO", "Edad_Madre"]

# Columnas categoricas que se conservan como contexto del cluster
# (no son features de K-Means, pero sirven para interpretar los grupos).
CONTEXTO = ["FecNac_Año", "sexo_nacido", "Condicion_Parto", "Tipo_Parto",
            "Nivel_Intrucción_Madre", "Financiador_Parto", "IdUbigeoInei"]

# Rangos de plausibilidad clinica. Se aplican ANTES del IQR para que los
# errores de digitacion (pesos de 0 g, gestaciones de 99 semanas) no
# distorsionen los cuartiles.
RANGOS_VALIDOS = {
    "PESO_NACIDO": (300, 7000),     # gramos
    "TALLA_NACIDO": (20, 70),       # cm
    "DUR_EMB_PARTO": (20, 45),      # semanas
    "Edad_Madre": (10, 60),         # anios
}

_lineas = []
FILAS_REPARADAS = {"n": 0}


def log(txt=""):
    print(txt)
    _lineas.append(str(txt))


def _reparar_fila(campos):
    """
    Repara filas con un ';' literal dentro de DESC_OCUPACION.

    Ejemplo real (linea 4000 del CSV original):
        ...;VENDEDOR AMBULANTE; COSMETICOS, PERFUMES, LOCION, ETC.;2;...
                              ^ este ';' parte el campo en dos

    El ';' sobrante siempre cae dentro de la ocupacion (columna categorica
    que NO se usa como feature), asi que fusionamos los fragmentos y no se
    pierde ninguna fila ni ningun valor numerico.
    """
    sobrantes = len(campos) - 22
    if sobrantes <= 0:
        return None
    fin = IDX_OCUPACION + sobrantes + 1
    fusion = ";".join(campos[IDX_OCUPACION:fin])
    FILAS_REPARADAS["n"] += 1
    return campos[:IDX_OCUPACION] + [fusion] + campos[fin:]


def _lector():
    return pd.read_csv(
        ENTRADA, sep=SEP, encoding=ENCODING, chunksize=CHUNK, dtype=str,
        keep_default_na=False, na_values=[], engine="python",
        on_bad_lines=_reparar_fila,
    )


def main():
    if not ENTRADA.exists():
        raise SystemExit(
            f"No existe {ENTRADA}\nCorre primero: python scripts/descargar_dataset.py"
        )

    tmpdir = Path(tempfile.mkdtemp(prefix="pc1_limpieza_"))

    log("=" * 78)
    log("LIMPIEZA - Registros de Nacidos Vivos (CNV) MINSA 2015-2025")
    log("=" * 78)
    log(f"Entrada : {ENTRADA.name} ({ENTRADA.stat().st_size/1024/1024:,.2f} MB)")
    log(f"Chunks  : {CHUNK:,} filas   |   modo: streaming (2 pasadas)")
    log()

    # ===============================================================
    # PASADA 1
    # ===============================================================
    log("PASADA 1/2 - duplicados, normalizacion, nulos y rango clinico")
    log("-" * 78)

    n_original = 0
    n_duplicados = 0
    n_nulos_desc = 0
    n_fuera_rango = 0
    hashes = set()
    temporales = []
    columnas_ctx = None

    for i, chunk in enumerate(_lector(), 1):
        n_original += len(chunk)

        # --- PASO 1: duplicados exactos (fila completa identica) ---
        antes = len(chunk)
        claves = [
            hashlib.md5("\x1f".join(f).encode("utf-8")).digest()[:8]
            for f in chunk.itertuples(index=False, name=None)
        ]
        nuevos = [k not in hashes for k in claves]
        hashes.update(k for k, n in zip(claves, nuevos) if n)
        chunk = chunk[pd.Series(nuevos, index=chunk.index)]
        n_duplicados += antes - len(chunk)
        del claves, nuevos

        if columnas_ctx is None:
            columnas_ctx = [c for c in CONTEXTO if c in chunk.columns]

        # --- PASO 3: normalizacion de tipos y formatos ---
        # Se ejecuta antes del paso 2 porque hay que castear a numerico
        # para poder distinguir el centinela -1 de un valor real.
        # TALLA_NACIDO viene con padding (" 52  ") y decimales con punto.
        for c in FEATURES:
            chunk[c] = pd.to_numeric(chunk[c].str.strip(), errors="coerce")
        for c in columnas_ctx:
            chunk[c] = chunk[c].str.strip().str.upper()

        chunk = chunk[columnas_ctx + FEATURES]

        # --- PASO 2: nulos. El diccionario del MINSA define -1 como
        # "ignorado". Se ELIMINAN las filas sin feature completa: imputar
        # peso o gestacion inventaria un perfil clinico inexistente y
        # desplazaria los centroides de K-Means. ---
        antes = len(chunk)
        chunk[FEATURES] = chunk[FEATURES].mask(chunk[FEATURES] < 0)
        chunk = chunk.dropna(subset=FEATURES)
        n_nulos_desc += antes - len(chunk)

        # --- PASO 4a: rango de plausibilidad clinica ---
        antes = len(chunk)
        for c, (lo, hi) in RANGOS_VALIDOS.items():
            chunk = chunk[chunk[c].between(lo, hi)]
        n_fuera_rango += antes - len(chunk)

        if len(chunk):
            tmp = tmpdir / f"p{i:05d}.parquet"
            chunk.to_parquet(tmp, index=False)
            temporales.append(tmp)

        if i % 5 == 0:
            print(f"    ... {n_original:,} filas leidas")

    del hashes

    log(f"Filas originales                    : {n_original:,}")
    log(f"PASO 0 - filas reparadas (';')      : {FILAS_REPARADAS['n']:,}")
    log(f"PASO 1 - duplicados exactos         : -{n_duplicados:,}")
    log(f"PASO 2 - nulos / centinela -1       : -{n_nulos_desc:,}")
    log(f"PASO 4a - fuera de rango clinico    : -{n_fuera_rango:,}")
    n_tras_p1 = n_original - n_duplicados - n_nulos_desc - n_fuera_rango
    log(f"Subtotal tras pasada 1              : {n_tras_p1:,}")
    log()

    # ===============================================================
    # Cuartiles para el IQR (solo columnas numericas)
    # ===============================================================
    log("PASO 4b - outliers por IQR (k=1.5)")
    log("-" * 78)
    solo_num = pd.concat(
        [pd.read_parquet(t, columns=FEATURES) for t in temporales],
        ignore_index=True,
    )
    limites = {}
    log(f"{'feature':<16}{'Q1':>10}{'Q3':>10}{'IQR':>10}{'lim_inf':>12}{'lim_sup':>12}")
    for c in FEATURES:
        q1, q3 = solo_num[c].quantile(0.25), solo_num[c].quantile(0.75)
        iqr = q3 - q1
        lo, hi = q1 - 1.5 * iqr, q3 + 1.5 * iqr
        limites[c] = (lo, hi)
        log(f"{c:<16}{q1:>10.2f}{q3:>10.2f}{iqr:>10.2f}{lo:>12.2f}{hi:>12.2f}")
    log()

    # Media y desviacion POST-IQR para el escalado. Se calculan con los
    # acumuladores sobre el subconjunto que sobrevive al corte.
    mask_iqr = pd.Series(True, index=solo_num.index)
    for c, (lo, hi) in limites.items():
        mask_iqr &= solo_num[c].between(lo, hi)
    sobreviven = solo_num[mask_iqr]
    n_outliers = len(solo_num) - len(sobreviven)
    n_final = len(sobreviven)

    stats_pre = {c: (sobreviven[c].min(), sobreviven[c].max(),
                     sobreviven[c].mean(), sobreviven[c].std())
                 for c in FEATURES}
    escala = {c: (sobreviven[c].mean(), sobreviven[c].std()) for c in FEATURES}
    del solo_num, sobreviven, mask_iqr

    log(f"Filas eliminadas por IQR : -{n_outliers:,}")
    log(f"Filas tras outliers      : {n_final:,}")
    log()

    # ===============================================================
    # PASADA 2 - aplicar IQR, escalar y escribir
    # ===============================================================
    log("PASADA 2/2 - aplicando IQR, escalando y escribiendo salida")
    log("-" * 78)

    SAL_PROC.parent.mkdir(parents=True, exist_ok=True)
    SAL_MUESTRA.parent.mkdir(parents=True, exist_ok=True)
    if SAL_PROC.exists():
        SAL_PROC.unlink()

    cols_esc = [c + "_ESC" for c in FEATURES]
    escritas = 0
    primero = True
    reservorio = []
    rng = np.random.default_rng(SEMILLA)
    vistas = 0

    for t in temporales:
        d = pd.read_parquet(t)
        m = pd.Series(True, index=d.index)
        for c, (lo, hi) in limites.items():
            m &= d[c].between(lo, hi)
        d = d[m]
        if not len(d):
            continue

        # PASO 6: estandarizacion z-score con media/desv globales post-IQR
        for c in FEATURES:
            mu, sd = escala[c]
            d[c + "_ESC"] = (d[c] - mu) / sd

        d.to_csv(SAL_PROC, mode="a", index=False, header=primero,
                 compression={"method": "gzip"} if primero else "gzip")
        primero = False
        escritas += len(d)

        # Muestreo de reservorio para los 5000 de data/sample/
        for _, fila in d.iterrows():
            vistas += 1
            if len(reservorio) < N_MUESTRA:
                reservorio.append(fila)
            else:
                j = rng.integers(0, vistas)
                if j < N_MUESTRA:
                    reservorio[int(j)] = fila

    shutil.rmtree(tmpdir, ignore_errors=True)

    pd.DataFrame(reservorio).to_csv(SAL_MUESTRA, index=False)

    mb_proc = SAL_PROC.stat().st_size / 1024 / 1024
    mb_mues = SAL_MUESTRA.stat().st_size / 1024 / 1024

    # ===============================================================
    # PASOS 5 y 6 - reporte
    # ===============================================================
    log(f"Filas escritas : {escritas:,}")
    log()
    log("PASO 5 - features seleccionadas para K-Means")
    log("-" * 78)
    for c in FEATURES:
        log(f"  {c}  ({RANGOS_VALIDOS[c][0]}-{RANGOS_VALIDOS[c][1]} valido)")
    log()

    log("PASO 6 - estadisticas ANTES del escalado")
    log("-" * 78)
    log(f"{'feature':<16}{'min':>10}{'max':>10}{'media':>12}{'desv':>12}")
    for c in FEATURES:
        mn, mx, mu, sd = stats_pre[c]
        log(f"{c:<16}{mn:>10.2f}{mx:>10.2f}{mu:>12.4f}{sd:>12.4f}")
    log()
    log("Escalado elegido: ESTANDARIZACION Z-SCORE.")
    log("K-Means minimiza la distancia euclidiana, de modo que una variable")
    log("con rango amplio (peso: cientos a miles de gramos) domina por")
    log("completo a una de rango corto (gestacion: 20-45 semanas) si no se")
    log("escala. Se prefiere z-score sobre min-max porque min-max comprime")
    log("la escala en funcion de los extremos y quedaria condicionado por")
    log("los valores limite que sobrevivieron al corte IQR.")
    log()

    log("PASO 6 - estadisticas DESPUES del escalado (z-score)")
    log("-" * 78)
    log("Por construccion media~0 y desv~1 en todas las features.")
    for c in FEATURES:
        mn, mx, mu, sd = stats_pre[c]
        log(f"  {c+'_ESC':<20} min={(mn-mu)/sd:>8.4f}  max={(mx-mu)/sd:>8.4f}  "
            f"media=0.000000  desv=1.000000")
    log()

    log("=" * 78)
    log("RESUMEN FINAL")
    log("=" * 78)
    log(f"Filas originales : {n_original:,}")
    log(f"Filas finales    : {escritas:,}")
    log(f"Filas eliminadas : {n_original - escritas:,} "
        f"({(n_original-escritas)/n_original*100:.2f}%)")
    log(f"Retencion        : {escritas/n_original*100:.2f}%")
    log()
    log(f"SUPERA 1,000,000 DE REGISTROS: "
        f"{'SI' if escritas > 1_000_000 else 'NO'}  ({escritas:,})")
    log()
    log(f"dataset_limpio.csv.gz : {mb_proc:,.2f} MB")
    log(f"muestra_5000.csv      : {mb_mues:,.2f} MB")
    log("=" * 78)

    SAL_LOG.parent.mkdir(parents=True, exist_ok=True)
    SAL_LOG.write_text("\n".join(_lineas), encoding="utf-8")
    print(f"\n[ok] Log guardado en {SAL_LOG}")


if __name__ == "__main__":
    main()
