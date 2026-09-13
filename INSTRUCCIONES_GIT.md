# Instrucciones de Git para el equipo — PC1 CC65

> Archivo de trabajo interno. **Borrar antes de la entrega final** (o dejarlo,
> no afecta la nota, pero no forma parte del informe).

El repositorio ya está inicializado en `main`, **sin ningún commit todavía**.
Cada integrante debe commitear sus propios archivos desde su cuenta de GitHub
para que el historial muestre la participación de los tres (2 puntos de la
rúbrica del Entregable 1).

---

## Paso 0 — Ya está hecho

El repositorio **ya existe, es público y está vacío**:

**https://github.com/yagocz/pc1-kmeans-cc65**

Ricardo (`20Ricky2010`) y Andre (`Andre-11c`) ya fueron invitados como
colaboradores con permiso de escritura. **Revisen su correo o
https://github.com/notifications y acepten la invitación antes de hacer push**,
o el push será rechazado con un error de permisos.

El `remote origin` ya está configurado en la carpeta local, así que no hace
falta correr `git remote add`.

---

## Paso 1 — Cada uno configura su identidad

Desde `C:\Users\Yago Caldas\pc1-kmeans`, cada integrante en su turno:

```bash
git config user.name "Su Nombre Completo"
git config user.email "sucorreo@upc.edu.pe"
```

Verificar con `git config user.email` antes de commitear. **Si este dato está
mal, el commit aparece a nombre de otra persona y el profesor no puede
acreditar la participación.**

---

## Paso 2 — Reparto de archivos por rama

### Rama `feature/setup-repo` — Integrante A

```
README.md
.gitignore
scripts/descargar_dataset.py
scripts/unir_dataset.py
```

```bash
git checkout -b feature/setup-repo
git add README.md .gitignore scripts/descargar_dataset.py scripts/unir_dataset.py
git commit -m "Estructura del repositorio, gitignore y scripts de dataset"
```

### Rama `feature/limpieza-dataset` — Integrante B

```
scripts/limpieza.py
data/processed/dataset_limpio.part1.csv.gz
data/processed/dataset_limpio.part2.csv.gz
data/sample/muestra_5000.csv
docs/salida_limpieza.txt
```

```bash
git checkout main
git checkout -b feature/limpieza-dataset
git add scripts/limpieza.py data/processed/ data/sample/ docs/salida_limpieza.txt
git commit -m "Pipeline de limpieza en streaming y dataset limpio (4,597,137 filas)"
```

### Rama `feature/eda-validacion` — Integrante C

```
scripts/exploracion.py
docs/informe_limpieza.md
docs/salida_exploracion.txt
docs/Diccionario_Datos_CNV.pdf
```

```bash
git checkout main
git checkout -b feature/eda-validacion
git add scripts/exploracion.py docs/informe_limpieza.md docs/salida_exploracion.txt docs/Diccionario_Datos_CNV.pdf
git commit -m "Exploracion del dataset e informe de limpieza"
```

---

## Paso 3 — Crear `develop` e integrar

`develop` no existe todavía porque una rama de Git necesita al menos un commit
para poder crearse. Después del **primer** commit (el de A):

```bash
git checkout main
git merge feature/setup-repo --no-ff -m "Merge feature/setup-repo"
git checkout -b develop
```

Luego se integran las otras dos:

```bash
git checkout develop
git merge feature/limpieza-dataset --no-ff -m "Merge feature/limpieza-dataset"
git merge feature/eda-validacion   --no-ff -m "Merge feature/eda-validacion"
```

Y finalmente a `main`:

```bash
git checkout main
git merge develop --no-ff -m "Entregable PC1"
```

> El flag `--no-ff` fuerza un commit de merge, de modo que el grafo muestre
> visualmente la estructura de Git Flow. Sin él, Git aplana el historial y se
> pierde la evidencia de las ramas.

---

## Paso 4 — Subir todo

El `remote` ya está configurado. Cada uno sube **su propia rama** apenas
termina su commit (no hace falta esperar a los demás):

```bash
git push -u origin feature/setup-repo        # Ricardo
git push -u origin feature/limpieza-dataset  # Yago
git push -u origin feature/eda-validacion    # Andre
```

Y al final, quien haga los merges sube las ramas integradas:

```bash
git push -u origin main
git push -u origin develop
```

---

## Verificaciones antes de entregar

```bash
# El historial debe mostrar los 3 nombres distintos
git log --pretty=format:"%h  %an  %s"

# El grafo debe mostrar las ramas
git log --graph --oneline --all

# Ningun archivo debe pesar mas de 100 MB
git ls-files | ForEach-Object { "{0,8:N2} MB  {1}" -f ((Get-Item $_).Length/1MB), $_ }
```

**El CSV original de 794 MB NO debe aparecer nunca en `git ls-files`.** Está
excluido por `.gitignore`; si aparece, el push será rechazado por GitHub.
