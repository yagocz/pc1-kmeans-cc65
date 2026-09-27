# Modelo formal de sincronización (Promela / SPIN)

Modelo de la sincronización del K-Means concurrente: un `Coordinador` y `W`
procesos `Worker` que repiten el ciclo de Lloyd con una barrera de dos fases
(asignación y actualización) y reducción sobre acumuladores privados.

| Archivo | Contenido |
|---|---|
| `kmeans_sync.pml` | Modelo verificado (W = 3 workers, 3 iteraciones) |
| `kmeans_sync_canal_compartido.pml` | Variante con un solo canal `proceed` compartido. Es un contraejemplo: SPIN encuentra el error |
| `verificar.sh` | Compila y ejecuta todas las verificaciones (incluido el contraejemplo) y deja las salidas en `logs/` |

## Cómo ejecutar

En Linux (o WSL) con SPIN y gcc instalados (`sudo apt install spin gcc`):

```bash
cd promela
bash verificar.sh
```

Desde Windows con Docker, sin instalar nada:

```powershell
"FROM ubuntu:24.04`nRUN apt-get update -qq && apt-get install -y -qq spin gcc`nWORKDIR /work" | docker build -t spin-local -
docker run --rm -v "${PWD}\promela:/work" spin-local bash verificar.sh
```

El comando equivalente al de la guía, paso a paso:

```bash
spin -a kmeans_sync.pml
gcc -O2 -DNOCLAIM -o pan_seguridad pan.c && ./pan_seguridad   # aserciones y deadlocks
gcc -O2 -o pan pan.c
./pan -a -N exclusion_mutua
./pan -a -N escritor_unico
./pan -a -N terminacion
```

`-DNOCLAIM` hace falta en la corrida de seguridad: como el modelo declara
fórmulas LTL, sin esa bandera `pan` carga una de ellas y desactiva la
detección de estados finales inválidos, que es justamente la que encuentra
deadlocks.

## Resultados (SPIN 6.5.2)

| Verificación | Estados almacenados | Transiciones | Profundidad | Errores |
|---|---:|---:|---:|---:|
| Seguridad (aserciones + deadlocks) | 20,940 | 30,730 | 325 | **0** |
| LTL `exclusion_mutua` | 20,940 | 30,734 | 639 | **0** |
| LTL `escritor_unico` | 20,940 | 30,734 | 639 | **0** |
| LTL `terminacion` | 20,704 | 81,607 | 594 | **0** |

Todo el código de los tres procesos es alcanzable (0 de 29 estados sin
alcanzar en `Worker`, 0 de 60 en `Coordinador`, 0 de 12 en `init`).

## Contraejemplo: canal `proceed` compartido

Con un único canal compartido por todos los workers, SPIN reporta
`assertion violated (procesado[i]==(iter+1))` a profundidad 93: un mismo
worker toma los 3 chunks de la iteración 0, los otros dos no procesan nada y
la barrera se libera igual porque recibe 3 mensajes en `done`. Si se quita esa
aserción, falla `total == W*PUNTOS` (la reducción suma 2 en lugar de 6), es
decir, se pierden datos al recalcular los centroides. `verificar.sh` guarda la
traza completa en `logs/traza_contraejemplo_canal_compartido.txt`.
