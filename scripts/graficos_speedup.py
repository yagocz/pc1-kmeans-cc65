"""
Genera los graficos de Speedup y Eficiencia a partir de los tiempos crudos
del benchmark.

Lee docs/resultados_bench.csv (columnas: version, workers, corrida, tiempo_ms)
y produce dos PNG para pegar en el informe:

    docs/grafico_speedup.png     Speedup vs workers, con la recta ideal
    docs/grafico_eficiencia.png  Eficiencia (Speedup/W) vs workers

No suaviza ni redondea: los puntos son la media recortada real de cada serie.

Uso:
    python scripts/graficos_speedup.py
"""
from pathlib import Path

import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt
import pandas as pd

RAIZ = Path(__file__).resolve().parents[1]
ENTRADA = RAIZ / "docs" / "resultados_bench.csv"
DOCS = RAIZ / "docs"


def media_recortada(s):
    """Descarta el maximo y el minimo antes de promediar."""
    v = sorted(s.tolist())
    if len(v) <= 2:
        return sum(v) / len(v)
    v = v[1:-1]
    return sum(v) / len(v)


def main():
    if not ENTRADA.exists():
        raise SystemExit(f"No existe {ENTRADA}\nCorre primero el benchmark en Go.")

    df = pd.read_csv(ENTRADA)

    base = media_recortada(df[df.version == "secuencial"].tiempo_ms)

    conc = df[df.version == "concurrente"]
    filas = []
    for w, g in conc.groupby("workers"):
        rec = media_recortada(g.tiempo_ms)
        filas.append({
            "workers": int(w),
            "t_recortada": rec,
            "speedup": base / rec,
            "eficiencia": (base / rec) / int(w),
        })
    r = pd.DataFrame(filas).sort_values("workers")

    print(f"T secuencial (media recortada): {base:,.2f} ms")
    print(r.to_string(index=False))

    # ---------------- GRAFICO 1: SPEEDUP ----------------
    fig, ax = plt.subplots(figsize=(8, 5.5), dpi=150)
    ax.plot(r.workers, r.workers, "--", color="#999999", linewidth=1.5,
            label="Speedup ideal (lineal)")
    ax.plot(r.workers, r.speedup, "o-", color="#1f5f8b", linewidth=2.2,
            markersize=8, label="Speedup medido")

    for _, f in r.iterrows():
        ax.annotate(f"{f.speedup:.2f}x",
                    (f.workers, f.speedup),
                    textcoords="offset points", xytext=(8, -12),
                    fontsize=9, color="#1f5f8b")

    mejor = r.loc[r.speedup.idxmax()]
    ax.axvline(mejor.workers, color="#c94f4f", linestyle=":", linewidth=1.4)
    ax.annotate(f"punto de equilibrio\nW = {int(mejor.workers)}",
                (mejor.workers, mejor.speedup * 0.45),
                fontsize=9, color="#c94f4f", ha="center")

    ax.set_xlabel("Numero de workers (goroutines)", fontsize=11)
    ax.set_ylabel("Speedup  (T secuencial / T concurrente)", fontsize=11)
    ax.set_title("Speedup de K-Means concurrente\n"
                 "4,597,137 registros - 4 dimensiones - AMD Ryzen 5 5600X (6 nucleos / 12 hilos)",
                 fontsize=11.5)
    ax.set_xticks(r.workers.tolist())
    ax.grid(alpha=0.3, linestyle="--")
    ax.legend(fontsize=10)
    fig.tight_layout()
    fig.savefig(DOCS / "grafico_speedup.png")
    print(f"\n[ok] {DOCS / 'grafico_speedup.png'}")

    # ---------------- GRAFICO 2: EFICIENCIA ----------------
    fig, ax = plt.subplots(figsize=(8, 5.5), dpi=150)
    ax.axhline(100, linestyle="--", color="#999999", linewidth=1.5,
               label="Eficiencia ideal (100%)")
    ax.plot(r.workers, r.eficiencia * 100, "s-", color="#2d7a4f",
            linewidth=2.2, markersize=8, label="Eficiencia medida")

    for _, f in r.iterrows():
        ax.annotate(f"{f.eficiencia*100:.0f}%",
                    (f.workers, f.eficiencia * 100),
                    textcoords="offset points", xytext=(8, 6),
                    fontsize=9, color="#2d7a4f")

    ax.axvline(6, color="#888888", linestyle=":", linewidth=1.2)
    ax.annotate("6 nucleos\nfisicos", (6, 20), fontsize=8.5,
                color="#666666", ha="center")
    ax.axvline(12, color="#888888", linestyle=":", linewidth=1.2)
    ax.annotate("12 nucleos\nlogicos", (12, 20), fontsize=8.5,
                color="#666666", ha="center")

    ax.set_xlabel("Numero de workers (goroutines)", fontsize=11)
    ax.set_ylabel("Eficiencia paralela  (Speedup / W)  [%]", fontsize=11)
    ax.set_title("Eficiencia paralela de K-Means concurrente\n"
                 "La caida refleja el costo de sincronizacion y la sobre-suscripcion",
                 fontsize=11.5)
    ax.set_xticks(r.workers.tolist())
    ax.set_ylim(0, 115)
    ax.grid(alpha=0.3, linestyle="--")
    ax.legend(fontsize=10)
    fig.tight_layout()
    fig.savefig(DOCS / "grafico_eficiencia.png")
    print(f"[ok] {DOCS / 'grafico_eficiencia.png'}")

    # ---------------- GRAFICO 3: TIEMPOS ----------------
    fig, ax = plt.subplots(figsize=(8, 5.5), dpi=150)
    etiquetas = ["secuencial"] + [f"W={int(w)}" for w in r.workers]
    valores = [base] + r.t_recortada.tolist()
    colores = ["#c94f4f"] + ["#1f5f8b"] * len(r)
    barras = ax.bar(etiquetas, valores, color=colores, edgecolor="white")
    for b, v in zip(barras, valores):
        ax.annotate(f"{v:,.0f}", (b.get_x() + b.get_width() / 2, v),
                    textcoords="offset points", xytext=(0, 4),
                    ha="center", fontsize=9)
    ax.set_ylabel("Tiempo del algoritmo (ms, media recortada)", fontsize=11)
    ax.set_xlabel("Configuracion", fontsize=11)
    ax.set_title("Tiempo de ejecucion por configuracion\n"
                 "10 corridas por configuracion, media recortada",
                 fontsize=11.5)
    ax.grid(alpha=0.3, linestyle="--", axis="y")
    fig.tight_layout()
    fig.savefig(DOCS / "grafico_tiempos.png")
    print(f"[ok] {DOCS / 'grafico_tiempos.png'}")


if __name__ == "__main__":
    main()
