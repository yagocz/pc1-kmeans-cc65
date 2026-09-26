/*
 * kmeans_sync_canal_compartido.pml  (CONTRAEJEMPLO: se espera que SPIN reporte error)
 * Modelo formal de la sincronizacion del K-Means concurrente (algoritmo de Lloyd)
 * con Worker Pool, barrera de dos fases y reduccion en memoria privada.
 *
 * Correspondencia con la implementacion en Go:
 *   Coordinador        -> goroutine principal (dispatch, reduccion, test de convergencia)
 *   Worker(id)         -> goroutine del Worker Pool
 *   proceed[id]        -> canal por el que cada worker recibe su chunk de la iteracion t
 *   done               -> barrera (equivale a wg.Done() / wg.Wait())
 *   suma_local[id]     -> acumuladores privados local_sums / local_counts
 *   centroide          -> coordenadas globales de los K centroides
 *
 * Los millones de registros no se cargan aqui: cada worker procesa PUNTOS
 * puntos abstractos, suficientes para exponer todos los entrelazamientos.
 */

#define W        3      /* workers concurrentes */
#define MAX_ITER 3      /* iteraciones maximas de Lloyd */
#define PUNTOS   2      /* puntos abstractos por particion */
#define FIN      255    /* senal de apagado del pool */

mtype = { ASIGNACION, ACTUALIZACION, TERMINADO };
mtype fase = ASIGNACION;

byte iter = 0;            /* iteracion cuyos centroides estan publicados */
byte centroide = 0;
byte suma_local[W];
byte procesado[W];        /* iteraciones completadas por cada worker */
byte en_asignacion = 0;   /* workers dentro de la fase de asignacion */
byte escritores = 0;      /* procesos escribiendo los centroides globales */
bool convergio = false;

chan proceed = [W] of { byte };   /* canal UNICO compartido (diseno de la guia) */
chan done = [W] of { byte };

proctype Worker(byte id) {
    byte t, p;
end:
    do
    :: proceed ? t ->
        if
        :: t == FIN -> break
        :: else -> skip
        fi;

        atomic {
            en_asignacion++;
            assert(fase == ASIGNACION);
            assert(t == iter)            /* R2: trabaja sobre la iteracion vigente */
        };

        suma_local[id] = 0;
        p = 0;
        do
        :: p < PUNTOS ->
            /* lectura de centroides: nadie puede estar escribiendolos (R1) */
            assert(escritores == 0 && t == iter);
            suma_local[id] = suma_local[id] + 1;
            p++
        :: else -> break
        od;

        procesado[id]++;
        en_asignacion--;
        done ! id
    od
}

proctype Coordinador() {
    byte i, id, total;

    do
    :: iter < MAX_ITER && !convergio ->
        fase = ASIGNACION;

        /* Master dispatch: un chunk por worker */
        for (i : 0 .. W - 1) {
            proceed ! iter
        }

        /* Barrera: espera las W notificaciones */
        for (i : 0 .. W - 1) {
            done ? id
        }

        atomic {
            fase = ACTUALIZACION;
            escritores++;
            assert(en_asignacion == 0)
        };

        /* Reduccion global sobre las sumas privadas */
        total = 0;
        for (i : 0 .. W - 1) {
            assert(procesado[i] == iter + 1);   /* cada worker proceso su chunk exactamente una vez */
            total = total + suma_local[i]
        }
        assert(total == W * PUNTOS);
        centroide = total;

        atomic {
            escritores--;
            iter++
        };

        /* Test de convergencia abstraido: SPIN explora ambos resultados */
        if
        :: convergio = true
        :: skip
        fi
    :: else -> break
    od;

    fase = TERMINADO;
    for (i : 0 .. W - 1) {
        proceed ! FIN
    }
}

init {
    byte i;
    atomic {
        for (i : 0 .. W - 1) {
            run Worker(i)
        }
        run Coordinador()
    }
}

/* Propiedades LTL */
ltl exclusion_mutua  { [] !(fase == ACTUALIZACION && en_asignacion > 0) }
ltl escritor_unico   { [] (escritores <= 1) }
ltl terminacion      { <> (fase == TERMINADO) }
