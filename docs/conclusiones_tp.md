# Conclusiones y recomendaciones

## Conclusiones

### Rendimiento concurrente

En las mediciones del dataset de 4,597,137 registros, el Worker Pool obtuvo su mayor speedup observado con 12 workers: **5.11×**, al reducir el tiempo medio recortado de 1,963.53 ms a 384.47 ms. La eficiencia en ese punto fue de 42.56%, por lo que el aumento de workers no se tradujo en una aceleración lineal. La medición de heap registró 0.06 MB adicionales entre la versión secuencial y W=16 (0.042%); como se tomó una sola corrida por configuración, la considero evidencia del bajo costo de memoria observado, no una estimación estadística del consumo en todas las ejecuciones.

La fracción secuencial estimada con Karp–Flatt cambia entre configuraciones: va de 9.08% en W=6 a 20.13% en W=2, y llega a 14.36% en W=16. Esto muestra que el costo efectivo de coordinación también depende del número de workers. El mínimo observado en W=6 coincide con los seis núcleos físicos de la máquina, aunque una sola máquina y estas configuraciones no bastan para afirmar que seis sea el punto óptimo en otros equipos.

### Alcance de los resultados

W=12 tuvo el menor tiempo medio recortado, pero no se puede afirmar que W=16 lo empeore: la comparación de Welch dio *t* = −0.30 y los intervalos de confianza se solapan. Interpreto el resultado como una meseta entre W=12 y W=16. También separo esa evidencia de la tabla de recursos, que contiene una sola corrida por configuración y cuyos tiempos no son comparables con los del benchmark de 15 corridas.

SPIN exploró exhaustivamente el modelo Promela configurado con tres workers y tres iteraciones: recorrió 20,940 estados para las propiedades de seguridad y no reportó errores en las propiedades verificadas. El modelo defectuoso con un canal compartido sí produjo el contraejemplo esperado a profundidad 93. Esa prueba respalda que el modelo detecta al menos la falla de sincronización deliberada; no amplía por sí sola la garantía formal a los valores de W usados en el benchmark, que llega a 16.

El método del codo ejecutado para k=2…10 no respalda de forma clara la elección de k=5: la reducción de inercia baja a 5.89% en k=5 y sube a 9.74% en k=6. La curva irregular justifica revisar la inicialización y repetir el análisis; por sí sola no demuestra que la causa sea un mínimo local ni determina un k óptimo. Los perfiles calculados con k=5 siguen siendo una descripción de esa ejecución, pero su interpretación depende de una elección de k que no quedó bien sustentada. Esto no invalida la comparación de speedup, porque las versiones secuencial y concurrente ejecutaron el mismo trabajo con el mismo k.

Los centroides de k=5 muestran diferencias descriptivas de edad materna y peso al nacer. Sin embargo, ningún grupo alcanza el umbral de bajo peso al nacer: el recorte por IQR de la PC1 había eliminado 271,233 registros, incluidos casos de bajo peso y prematuridad. Por ello, estos clusters describen el conjunto depurado y no permiten evaluar la detección de los casos extremos que motivan el caso de uso. Tampoco constituyen una validación clínica ni demuestran capacidad para predecir riesgo individual.

### Reflexión sobre el trabajo

Para mí, el resultado más importante no es solo la aceleración, sino haber tenido que acotar las afirmaciones cuando los datos no las respaldaban. La comparación entre W=12 y W=16 cambió de una supuesta degradación a una meseta, y el método del codo mostró que no podíamos presentar k=5 como una elección confirmada. Reportar esas limitaciones deja más claro qué aprendimos y qué queda pendiente.

## Recomendaciones

### Para el código y la verificación

1. **Centralizar el Worker Pool.** La lógica concurrente está repetida en varios programas. Extraerla a una implementación común reduciría el riesgo de que el benchmark, la medición de recursos y el ejecutable principal terminen midiendo versiones distintas.
2. **Agregar pruebas automatizadas.** Incluir pruebas unitarias para las funciones de K-Means y una prueba de equivalencia entre las versiones secuencial y concurrente con un dataset pequeño permitiría detectar regresiones sin comparar centroides manualmente.
3. **Precisar el alcance de SPIN.** Mantener explícito en el informe que la verificación exhaustiva corresponde al modelo con W=3 y tres iteraciones. Si se continúa el trabajo, verificar otros valores pequeños de W aportaría evidencia adicional, aunque no demostraría automáticamente todos los tamaños del benchmark.
4. **Conservar la separación entre mediciones.** Repetir las mediciones de heap con varias corridas y reportar por separado sus resultados evitaría comparar tiempos puntuales con el benchmark de 15 corridas.

### Para la metodología y el caso de uso

1. **Revisar k con inicialización múltiple o K-means++.** Repetir el método del codo con varias semillas y una inicialización más robusta permitiría saber si la irregularidad observada es reproducible antes de escoger otro valor de k.
2. **Evaluar el efecto del recorte por IQR.** Comparar los perfiles obtenidos con los datos sin ese recorte o con un criterio menos restrictivo ayudaría a conservar los casos extremos relevantes. La comparación debe documentar cómo cambia la calidad y la interpretación, sin asumir que una alternativa es mejor de antemano.
3. **Validar los perfiles con criterios adecuados al dominio.** Contrastar los clusters con métricas de calidad y conocimiento clínico independiente sería necesario antes de usarlos para orientar decisiones sobre control prenatal. Las medias de los centroides sirven para describir grupos, no para diagnosticar.
4. **Ampliar el benchmark si se quiere distinguir una meseta de una degradación.** Más corridas y un análisis de potencia permitirían estimar con mayor precisión la diferencia entre W=12 y W=16. Las conclusiones actuales se limitan a la máquina y al protocolo medidos.

### Para una continuación del proyecto

El experimento actual usa un dataset de 140.29 MB en una máquina con 15.9 GB de RAM, de modo que no demuestra cómo respondería la arquitectura ante datos que excedan la memoria disponible. Como siguiente paso se podría evaluar una estrategia distribuida o por particiones, midiendo su tiempo total, memoria y calidad de clustering frente a la implementación actual.
