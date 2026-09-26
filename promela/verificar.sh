#!/usr/bin/env bash
# Verificacion formal de kmeans_sync.pml con SPIN.
# Uso: bash verificar.sh   (desde la carpeta promela/)
set -e
cd "$(dirname "$0")"
mkdir -p logs

spin -a kmeans_sync.pml
gcc -O2 -o pan pan.c
gcc -O2 -DNOCLAIM -o pan_seguridad pan.c

echo "=== Seguridad: aserciones y estados finales invalidos (deadlocks) ==="
./pan_seguridad | tee logs/spin_seguridad.txt

for prop in exclusion_mutua escritor_unico terminacion; do
    echo
    echo "=== LTL: $prop ==="
    ./pan -a -N "$prop" | tee "logs/spin_ltl_${prop}.txt"
done

echo
echo "=== Contraejemplo: canal proceed compartido (se espera errors: 1) ==="
spin -a kmeans_sync_canal_compartido.pml
gcc -O2 -DNOCLAIM -o pan_contraejemplo pan.c
./pan_contraejemplo | tee logs/spin_contraejemplo_canal_compartido.txt || true
spin -t -p kmeans_sync_canal_compartido.pml > logs/traza_contraejemplo_canal_compartido.txt

rm -f pan pan_seguridad pan_contraejemplo pan.* _spin_nvr.tmp *.trail
