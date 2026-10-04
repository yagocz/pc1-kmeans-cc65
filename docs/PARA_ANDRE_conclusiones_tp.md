# Sección para el Word: Conclusiones y recomendaciones — Andre

> **Andre:** acá está el material y un borrador de las conclusiones del TP.
> Vale **4 puntos** de la rúbrica. Ajustalo a tu criterio y pegalo en el Word.
>
> La rúbrica dice "Análisis con las conclusiones y recomendaciones **del
> alumno**", así que conviene que suene a tu voz y no a un resumen. Donde
> escribas algo propio, mejor.

---

## Insumos (todo ya medido, no hay que correr nada)

| Dato | Valor | Fuente |
|---|---|---|
| Speedup máximo | 5.1071× (W=12) | `docs/resultados_bench.csv` |
| Eficiencia en ese punto | 42.56% | ídem |
| W=12 vs W=16 | t de Welch = −0.30, no significativo | tu análisis de la PC2 |
| Fracción secuencial (Karp-Flatt) | varía 9.08% – 20.13% | tu análisis de la PC2 |
| Costo de memoria del paralelismo | 0.06 MB (0.04%) | `docs/salida_recursos.txt` |
| Verificación SPIN | 20,940 estados, 0 errores | `docs/salida_spin.txt` |
| Contraejemplo | `errors: 1` a profundidad 93 | `docs/salida_contraejemplo.txt` |
| GAPs identificados | 12 (1 crítico, 3 altos, 6 medios, 2 bajos) | `docs/INFORME_GAPS_IA.md` |
| GAPs corregidos | 4 (7, 8, 9, 10) | ídem |
| Método del codo | **no respalda k=5** | `docs/calidad_clustering.md` |

---

## Borrador: Sección 15 — Conclusiones

### 15.1. Sobre el rendimiento concurrente

El Worker Pool alcanzó un Speedup de **5.11× con 12 workers** sobre 4,597,137
registros, reduciendo el tiempo de 1963.53 ms a 384.47 ms. El costo en memoria
fue de 0.06 MB, un 0.04% adicional, porque el dataset se comparte por
referencia y el pool persistente reutiliza los acumuladores privados en lugar
de reservarlos en cada iteración. **La ganancia en tiempo no se pagó con
memoria**, lo que descarta el trade-off que suele acompañar a este tipo de
optimizaciones.

El hallazgo más relevante del análisis de escalabilidad es que **la fracción
secuencial no es constante**: medida con la métrica de Karp-Flatt, varía entre
9.08% (W=6) y 20.13% (W=2). La Ley de Amdahl supone que esa fracción es una
propiedad fija del programa; nuestros datos muestran que en un esquema por
paso de mensajes incorpora un costo de sincronización que crece con W, ya que
cada iteración intercambia 2×W mensajes de canal. El mínimo en W=6 coincide
exactamente con los seis núcleos físicos del procesador utilizado.

### 15.2. Sobre los límites de lo que los datos permiten afirmar

Una conclusión que el equipo tuvo que corregir durante el trabajo es que **el
comportamiento en W=16 es una meseta, no una degradación**. El informe
preliminar afirmaba que el rendimiento empeoraba a partir de W=12, pero la
prueba t de Welch entre ambas configuraciones arroja t = −0.30, un valor no
significativo con intervalos de confianza completamente solapados.

La distinción importa: afirmar "empeora" cuando los datos solo sostienen "deja
de mejorar" es una sobreinterpretación. Para demostrar degradación haría falta
un número de corridas mayor o un test con más potencia estadística.

Algo similar ocurrió con la tabla de recursos, que proviene de una sola corrida
por configuración y no es comparable con las 15 del benchmark. La contradicción
es observable: en la tabla de recursos W=16 aparece más rápido que W=12,
mientras que en la de speedup ocurre lo contrario. Solo la segunda tiene
potencia estadística.

### 15.3. Sobre la verificación formal

La verificación exhaustiva en SPIN exploró 20,940 estados sin encontrar
errores en ninguna de las tres propiedades: exclusión mutua, escritor único y
terminación. El dato que vuelve concluyente ese resultado es el
**contraejemplo deliberado**: una variante del modelo con canal compartido en
la que SPIN detecta la violación a profundidad 93. Sin esa validación, los 0
errores podrían ser un falso negativo de un modelo mal construido.

El orden de trabajo resultó más valioso de lo previsto. El contraejemplo se
construyó **antes** de escribir la versión final en Go, y determinó una
decisión de diseño concreta: usar un canal por worker en lugar de uno
compartido. La verificación formal no validó el código a posteriori; lo
condicionó.

Corresponde declarar también su límite: el modelo verifica W=3 mientras el
benchmark mide hasta W=16. Las propiedades son estructurales y no dependen del
valor concreto de W, pero la generalización no está formalmente demostrada.

### 15.4. Sobre la calidad del clustering

El análisis con IA reveló que **el método del codo no respalda la elección de
k=5** usada en todos los experimentos. La reducción de inercia cae a 5.89% en
k=5 y repunta a 9.74% en k=6, un comportamiento que no corresponde a una curva
de codo bien formada e indica convergencia a un mínimo local, atribuible a
usar inicialización aleatoria simple en lugar de K-means++.

El equipo optó por reportar el hallazgo en lugar de ajustar k retroactivamente.
Cambiar a k=6 habría obligado a repetir las 105 mediciones del benchmark, y el
resultado es información válida en sí misma. El análisis de Speedup no se ve
afectado, porque ambas versiones ejecutan el mismo trabajo con el mismo k.

Pese a esa limitación, los cinco clusters resultaron **clínicamente
interpretables**. La segmentación se estructura según dos ejes —edad materna,
de 22.12 a 35.97 años, y peso al nacer, de 2781 a 3650 g— que son exactamente
los factores que el caso de uso identificó como predictores de riesgo
neonatal. Los clusters 3 y 4 ocupan los extremos opuestos de edad materna y
concentran el 43.5% de los nacimientos, correspondiendo a los rangos que la
literatura obstétrica asocia con mayor riesgo.

### 15.5. Sobre la limitación que atraviesa todo el trabajo

El recorte por IQR aplicado en la limpieza de la PC1 eliminó 271,233 registros,
entre ellos los casos de bajo peso real y prematuridad. La consecuencia se
hizo visible recién al interpretar los clusters: **ningún grupo alcanza el
umbral de bajo peso al nacer**, no porque esos casos no existan en el registro
del MINSA, sino porque fueron excluidos antes del clustering por ser
estadísticamente atípicos.

La tensión se había declarado desde la PC1 y los resultados finales la
confirman. Un análisis orientado a detectar riesgo neonatal no debería
descartar precisamente los casos de riesgo, aunque el criterio IQR sea el
método estándar.

---

## Borrador: Sección 16 — Recomendaciones

### 16.1. Para el código

**Eliminar la duplicación del algoritmo concurrente.** El Worker Pool está
replicado en cuatro archivos (`concurrente`, `bench`, `recursos`, `calidad`).
Es el GAP de mayor riesgo latente: si uno queda desincronizado, el benchmark
mediría una implementación distinta de la verificada formalmente. El riesgo ya
se materializó una vez durante el desarrollo.

**Añadir pruebas automatizadas.** No existe ningún `*_test.go` sobre 1,429
líneas de código Go. Una prueba de equivalencia entre ambas versiones
automatizaría la comprobación que hoy se hace comparando centroides a ojo.

**Condicionar las aserciones a un build tag.** Los cuatro `panic()` de
`go/concurrente/main.go` replican los `assert` del modelo Promela, lo cual es
buena práctica de trazabilidad, pero un panic dentro de una goroutine termina
el proceso sin posibilidad de diagnóstico.

### 16.2. Para la metodología

**Implementar K-means++ y repetir el método del codo.** La irregularidad de la
curva es un síntoma conocido de la inicialización aleatoria simple, y los tres
papers revisados en la PC1 coinciden en señalarlo.

**Repetir el clustering sin el recorte por IQR**, o con k = 3.0 en lugar de
1.5, para capturar los perfiles de riesgo en sus valores reales.

**Verificar el modelo con W=4 y W=5** para mostrar que el espacio de estados
crece sin que aparezcan errores nuevos, y declarar explícitamente el alcance
de la garantía formal.

### 16.3. Para un trabajo futuro

La arquitectura actual está limitada a memoria compartida en una sola máquina.
Ghimire & Amsaad (2024) señalan que ese enfoque topa cuando el dataset supera
la RAM física; en nuestro caso no llegamos a ese límite —140.62 MB sobre 15.9
GB—, pero un dataset diez veces mayor sí lo alcanzaría. La extensión natural
es el esquema de Big-means de Mussabayev et al. (2023), donde cada worker
procesa una muestra independiente y el único estado compartido es la mejor
solución encontrada.

---

## Para commitear

```bash
git checkout develop
git pull origin develop
git checkout -b feature/conclusiones-tp

git config user.name "Andre Angel Chipana Rios"
git config user.email "u202220230@upc.edu.pe"

# creás docs/conclusiones_tp.md con tus secciones ajustadas
git add docs/conclusiones_tp.md
git commit -m "Conclusiones y recomendaciones del Trabajo Parcial"
git push -u origin feature/conclusiones-tp
```

---

## Un consejo

La rúbrica dice "conclusiones **del alumno**". El borrador de arriba es un
punto de partida, pero las partes más valiosas van a ser las que escribas vos:
qué te sorprendió, qué harías distinto, qué no funcionó como esperabas.

Lo del k=5 es buen material para eso: descubrir que el hiperparámetro central
del trabajo no estaba justificado, **a una semana de entregar**, y decidir
reportarlo en vez de taparlo.
