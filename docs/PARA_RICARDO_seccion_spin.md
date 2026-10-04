# Sección para el Word: Verificación formal en Spin — Ricardo

> **Ricardo:** esta sección ya está redactada con los resultados de tu propia
> verificación. Revísala, ajústala a tu criterio y pégala en el Word. Vale
> **4 puntos** de la rúbrica del TP.
>
> Lo que falta es **tuyo**: las dos capturas y la explicación de por qué
> elegiste modelar con canales en lugar de un mutex global.

---

## 13. Verificación formal en Spin

### 13.1. Alcance de la verificación

La verificación se ejecutó sobre `promela/kmeans_sync.pml` con SPIN 6.5.2,
aplicando Partial Order Reduction. El script `promela/verificar.sh` reproduce
el procedimiento completo y guarda las salidas en `promela/logs/`.

El procedimiento distingue dos tipos de comprobación, que requieren
compilaciones distintas del verificador:

```bash
spin -a kmeans_sync.pml              # genera pan.c a partir del modelo
gcc -O2 -DNOCLAIM -o pan_seguridad pan.c   # seguridad: aserciones + deadlocks
gcc -O2 -o pan pan.c                        # propiedades LTL
./pan_seguridad
./pan -a -N exclusion_mutua
./pan -a -N escritor_unico
./pan -a -N terminacion
```

La bandera `-DNOCLAIM` es indispensable en la corrida de seguridad. Como el
modelo declara fórmulas LTL, sin esa bandera `pan` carga una de ellas y
**desactiva la detección de estados finales inválidos**, que es justamente la
comprobación que encuentra deadlocks. Omitirla produciría un informe que
afirma ausencia de deadlock sin haberlo verificado.

### 13.2. Ausencia de deadlocks

La corrida de seguridad explora el espacio de estados completo buscando
estados finales inválidos, es decir, configuraciones en las que el sistema no
puede avanzar pero tampoco ha terminado correctamente. La salida reporta:

```
Full statespace search for:
    never claim             - (not selected)
    assertion violations    +
    acceptance   cycles     - (not selected)
    invalid end states      +

State-vector 96 byte, depth reached 325, errors: 0
    20940 states, stored
     9790 states, matched
    30730 transitions (= stored+matched)
```

La línea `invalid end states +` confirma que la comprobación estuvo activa, y
`errors: 0` que no se encontró ninguno. **Los 20,940 estados constituyen el
espacio completo alcanzable del modelo**, no una muestra: la búsqueda es
exhaustiva, de modo que no existe ningún entrelazado de los procesos que
conduzca a un bloqueo.

El mismo resultado descarta además violaciones de las aserciones embebidas en
el modelo, que comprueban que cada worker procesa su chunk exactamente una vez
(`procesado[i] == iter+1`) y que la reducción no pierde contribuciones
(`total == W*PUNTOS`).

### 13.3. Exclusión mutua y demás propiedades LTL

| Propiedad | Fórmula | Estados | Transiciones | Profundidad | Errores |
|---|---|---:|---:|---:|---:|
| `exclusion_mutua` | `[] !(fase == ACTUALIZACION && en_asignacion > 0)` | 20,940 | 30,734 | 639 | **0** |
| `escritor_unico` | `[] (escritores <= 1)` | 20,940 | 30,734 | 639 | **0** |
| `terminacion` | `<> (fase == TERMINADO)` | 20,704 | 81,607 | 594 | **0** |

**`exclusion_mutua`** es la propiedad central del trabajo. Establece que en
ningún instante puede haber un worker dentro de la fase de asignación —leyendo
los centroides— mientras el coordinador está en la fase de actualización
escribiéndolos. Es la condición que garantiza ausencia de condición de carrera
sobre la estructura compartida.

**`escritor_unico`** complementa a la anterior: verifica que nunca haya más de
un proceso escribiendo los centroides globales de forma simultánea.

**`terminacion`** es una propiedad de *liveness*, no de seguridad: no afirma
que nada malo ocurra, sino que algo bueno ocurre eventualmente. Verifica que
el sistema siempre alcanza el estado `TERMINADO`, lo que descarta no solo el
deadlock sino también el *livelock*, una situación en la que los procesos
siguen ejecutándose sin progresar hacia el final.

El consumo del verificador fue de 130.29 MB con 74.39% de compresión del
vector de estados.

### 13.4. Cobertura del modelo

```
unreached in proctype Worker       (0 of 29 states)
unreached in proctype Coordinador  (0 of 60 states)
unreached in init                  (0 of 12 states)
```

Ningún estado del modelo quedó sin alcanzar. Este dato es relevante porque
descarta que alguna propiedad se cumpla de forma trivial: si parte del código
nunca se ejecutara durante la verificación, las propiedades podrían
satisfacerse sin que ello demostrara nada sobre esa región del programa.

### 13.5. Validación del verificador mediante contraejemplo

Una verificación que no reporta errores solo es concluyente si se demuestra
que el verificador es capaz de encontrarlos. Para comprobarlo se construyó
`promela/kmeans_sync_canal_compartido.pml`, idéntica al modelo correcto salvo
que los W workers comparten un único canal `proceed` en lugar de disponer cada
uno del suyo.

SPIN detecta la violación:

```
pan:1: assertion violated (procesado[i]==(iter+1)) (at depth 93)
State-vector 80 byte, depth reached 93, errors: 1
```

La traza muestra que un mismo worker consume los tres chunks de la iteración 0
mientras los otros dos no procesan ninguno, y la barrera se libera igualmente
porque recibe tres mensajes en el canal `done`. Al suprimir esa aserción falla
la siguiente, `total == W*PUNTOS`: la reducción acumula 2 en lugar de 6, es
decir, **se pierden datos al recalcular los centroides**.

El contraejemplo cumple dos funciones. Confirma que los 0 errores del modelo
correcto no son un falso negativo producido por un modelo mal construido, y
justifica formalmente la decisión de diseño de asignar un canal por worker,
tomada **antes** de escribir el código en Go.

### 13.6. Limitación declarada

El modelo verifica una configuración de **W = 3 workers y 3 iteraciones**,
mientras que el benchmark mide configuraciones de hasta 16 workers. La
garantía formal, en sentido estricto, aplica a la configuración verificada.

Esta limitación es inherente al model checking: el espacio de estados crece
exponencialmente con el número de procesos, y una verificación de W = 16 sería
intratable en un equipo de escritorio. Las propiedades verificadas son
estructurales —dependen de la arquitectura de coordinación, no del valor
concreto de W—, por lo que la generalización es razonable, pero **no está
formalmente demostrada** y se declara explícitamente como tal.

---

## Lo que falta que agregues tú, Ricardo

**1. Las dos capturas.** Están en el repo:
- `docs/capturas/06_verificacion_spin.png` → va en §13.3
- `docs/capturas/07_contraejemplo_spin.png` → va en §13.5

**2. Un párrafo tuyo en §13.1** explicando por qué modelaste la coordinación
con canales (`proceed[]` y `done`) en lugar de un mutex global. Es una
decisión de diseño tuya y conviene que la expliques con tus palabras: el
profesor puede preguntarlo en la sustentación.

**3. Revisar que coincida con tu `promela/README.md`**, por si quieres ajustar
alguna cifra o agregar algo del contraejemplo que yo haya resumido de más.

---

## Para commitear

```bash
git checkout develop
git pull origin develop
git checkout -b feature/spin-verificacion

git config user.name "Ricardo Rafael Rivas Carrillo"
git config user.email "tucorreo@upc.edu.pe"

# creás docs/verificacion_spin.md con la sección ya ajustada por vos
git add docs/verificacion_spin.md
git commit -m "Verificacion formal en Spin: ausencia de deadlocks y exclusion mutua"
git push -u origin feature/spin-verificacion
```
