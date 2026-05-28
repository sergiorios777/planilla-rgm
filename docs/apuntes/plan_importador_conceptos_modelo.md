# Plan para implementar un Importador de Conceptos Modelo

## Ideas inciales
Agregar la característica de importador conceptos modelo al módulo de conceptos modelo del panel `admin`. Las tablas clave son: `conceptos_modelo` y `regimen_conceptos_modelo`.
Algunas consideraciones de la estructura de los datos que serán importados son:

1. El usuario elabora el archivo CSV que se subirá al importador.
2. Para la columna `conceptos_modelo.concepto_id` debe utilizar el campo `conceptos_maestros.codigo` en el csv.
3. Para la columna `conceptos_modelo.clasificador_id` debe utilizar el campo `clasificadores_mef.codigo_limpio` en el csv. Por ejemplo: en lugar de utilizar el código normal "2.1.1 1.1 1" debe utilizar "2.1.1.1.1.1"; el lugar de utilizar "2.1.1 1.2 99" debe utilizar "2.1.1.1.2.99".
4. Para la identificación de los regímenes a los cuáles afecta el concepto modelo utilizar cuatro columnas en el CSV cada uno referido a un régimen laboral (DL 276, DL 728, DL 1057, LEY 30057)
5. El archivo CSV debe tener encabezados.

Utilizar códigos estandarizados (`conceptos_maestros.codigo` y `clasificadores_mef.codigo_limpio`) en lugar de IDs numéricos internos, garantizas que el archivo CSV sea legible para un ser humano y completamente independiente del estado de las secuencias autoincrementales de la base de datos.
---

## ¿Cómo funcionará la propuesta?

Cuando subimos un archivo con cientos de filas que se relacionan con otras tablas, aplicamos un patrón de arquitectura llamado **ETL (Extract, Transform, Load)**. Si lo hiciéramos de forma ingenua (fila por fila consultando a la base de datos), el sistema sería sumamente lento. Para que sea ultra-eficiente y rápido, implementaremos tres pilares:

1. **Diccionarios en Memoria RAM (Evitamos el "Cuello de Botella"):** Antes de leer la primera fila del CSV, el servicio de Go hará tres consultas rápidas a la base de datos para traerse todos los clasificadores, conceptos maestros y regímenes. Con esto armará tres mapas en la memoria RAM (ej: `mapaClasificadores["2.1.1.1.1.1"] = 14`). Así, cuando Go procese las 500 filas del CSV, resolverá los IDs de inmediato buscando en la memoria de la computadora en lugar de hacer miles de viajes lentos de consulta a PostgreSQL.
2. **Transaccionalidad Atómica (Todo o Nada):**
La importación se ejecutará envuelta en una transacción de SQL (`tx, err := db.Begin()`). Si el archivo tiene 100 filas, y la fila 99 tiene un error de digitación (un clasificador que no existe), el sistema ejecutará un `ROLLBACK`. Esto significa que se borrará todo lo que se intentó guardar de las 98 filas anteriores. Así evitamos el dolor de cabeza de dejar la base de datos "a medias" con información corrupta.
3. **Mapeo Dinámico de Regímenes (Muchos a Muchos):**
Tu CSV tendrá 4 columnas al final (`DL 276`, `DL 728`, `DL 1057`, `LEY SERVIR`). Si la fila actual tiene un "1", "SI" o "true" en las columnas `DL 276` y `DL 1057`, el sistema guardará el concepto principal en `conceptos_modelo` y luego insertará dos filas en la tabla pivot `regimen_concepto_modelo` vinculando ese concepto con los IDs de esos dos regímenes en específico.

---

## Propuesta del Plan de Implementación: Importador de Conceptos Modelo (SaaS Admin)

Sigue este plano estructurado paso a paso para que **Antigravity 2.0** realice una refactorización limpia y modular, manteniendo la lógica de negocio fuera del handler.

### Fase 1: Capa de Datos (Ampliación del Repositorio)

Debemos darle herramientas al repositorio para alimentar nuestros diccionarios en memoria y ejecutar la inserción masiva transaccional.

**Instrucciones para el Agente:**

1. En `internal/repository/concepto_modelo_repository.go`, agregar los siguientes métodos de soporte:
* `ObtenerMapaMaestros() (map[string]int, error)`: Retorna un mapa donde la llave es `codigo` (string) y el valor es `id` (int) de la tabla `conceptos_maestros`.
* `ObtenerMapaClasificadores() (map[string]int, error)`: Retorna un mapa donde la llave es `codigo_limpio` (string) y el valor es `id` (int) de la tabla `clasificadores_mef`.
* `ObtenerMapaRegimenes() (map[string]int, error)`: Retorna un mapa mapeando nombres estándar a sus IDs correspondientes (ej: `"DL 276" -> 1`).


2. Implementar el método maestro transaccional:
* `GuardarConceptoModeloImportado(tx *sql.Tx, cm *models.ConceptoModelo, regimenesIDs []int) error`: Inserta el concepto en `conceptos_modelo`, obtiene el `id` generado (`RETURNING id`) y luego corre un bucle para insertar en `regimen_concepto_modelo` las relaciones del concepto con cada `regimenID`.



### Fase 2: Capa de Negocio (El Servicio de Importación)

Crearemos un servicio dedicado para aislar las reglas de negocio y el procesamiento del CSV.

**Instrucciones para el Agente:**

1. Crear el archivo `internal/services/concepto_modelo_service.go` con el struct `ConceptoModeloService` (inyectándole el `ConceptoModeloRepository` y la conexión `*sql.DB`).
2. Implementar la función `ImportarDesdeCSV(reader io.Reader) (exitosos int, err error)`:
* **Paso A:** Cargar los tres mapas en memoria RAM usando los métodos del repositorio creados en la Fase 1.
* **Paso B:** Iniciar la transacción con `tx, err := s.Db.Begin()`. Asegurar un `defer tx.Rollback()` por seguridad.
* **Paso C:** Inicializar `csv.NewReader` configurando `FieldsPerRecord = 11` (o el número exacto de columnas). Leer la primera línea para descartar las cabeceras.
* **Paso D:** Iterar por cada fila del CSV:
* Buscar el `concepto_id` en el mapa de maestros usando el código del CSV. Si no existe, retornar error deteniendo la transacción.
* Buscar el `clasificador_id` en el mapa de clasificadores usando el `codigo_limpio` extraído del CSV.
* Mapear los campos booleanos (`requiere_monto`, `es_pensionable`, `es_remunerativa`, etc.).
* Evaluar las 4 columnas de regímenes. Si contienen `"1"`, `"true"` o `"SI"`, añadir el ID del régimen correspondiente desde el mapa a un *slice* de enteros (`regimenesAAfectar`).
* Llamar a `s.Repo.GuardarConceptoModeloImportado(tx, &modelo, regimenesAAfectar)`.


* **Paso E:** Si el bucle termina sin errores, ejecutar `tx.Commit()`.



### Fase 3: Capa de Control (Handler y Rutas)

El controlador procesará el archivo binario subido por el formulario multipart de HTMX.

**Instrucciones para el Agente:**

1. En `internal/handlers/concepto_modelo_handler.go`:
* Inyectar el nuevo `ConceptoModeloService` dentro del struct `ConceptoModeloHandler`.
* Crear el método `ImportarCSV(w http.ResponseWriter, r *http.Request)`:
* Validar método POST y procesar `r.ParseMultipartForm(10 << 20)`.
* Recuperar el archivo mediante `r.FormFile("archivo_csv")`.
* Llamar al servicio: `exitosos, err := h.Service.ImportarDesdeCSV(file)`.
* Si hay error, responder con un fragmento HTML de alerta estilizado con Pico CSS (color anaranjado/rojo) mostrando el error detallado.
* Si es exitoso, enviar la cabecera `HX-Trigger: cerrarModalImportar, refrescarListaModelos` y responder con un mensaje de éxito.




2. En `internal/routes/routes.go`, registrar la ruta:
* `POST /admin/conceptos-modelo/importar` apuntando a este nuevo método.



### Fase 4: Interfaz de Usuario (HTML5 + Pico CSS + HTMX)

Modernizaremos la vista agregando un modal nativo que controle la carga.

**Instrucciones para el Agente:**

1. En `ui/templates/admin/conceptos_modelo_ui.html`, en la sección de la cabecera (junto al botón "Nuevo Concepto Modelo"), agregar un nuevo botón:
```html
<button class="outline secondary" onclick="document.getElementById('modal-importar-csv').showModal()">
    📥 Importar desde CSV
</button>

```


2. Al final del archivo, implementar el nuevo `<dialog>` de carga masiva:
```html
<dialog id="modal-importar-csv">
    <article style="max-width: 600px; width: 95%;">
        <header>
            <a href="#close" aria-label="Close" class="close" onclick="document.getElementById('modal-importar-csv').close()"></a>
            <h3>📥 Importar Catálogo de Conceptos Modelo</h3>
        </header>

        <form hx-post="/admin/conceptos-modelo/importar" 
              hx-encoding="multipart/form-data" 
              hx-target="#feedback-importacion"
              hx-swap="innerHTML">

            <label for="archivo_csv">Seleccione el archivo CSV estructurado:
                <input type="file" id="archivo_csv" name="archivo_csv" accept=".csv" required>
            </label>

            <div style="background: var(--pico-muted-background); padding: 0.75rem; border-radius: var(--pico-border-radius); margin-bottom: 1rem;">
                <small style="display: block; color: var(--pico-muted-color); font-size: 0.8rem;">
                    💡 <strong>Estructura del archivo:</strong> Debe contener encabezados obligatorios. Utilice el código maestro SUNAT para el concepto, el código limpio (ej: 2.1.1.1.1.1) para el clasificador MEF, y marque con '1' o 'true' las columnas de afectación de los regímenes (DL 276, DL 728, DL 1057, LEY SERVIR).
                </small>
            </div>

            <div id="feedback-importacion"></div>

            <footer>
                <div style="display: flex; justify-content: flex-end; gap: 10px;">
                    <button type="button" class="secondary outline" onclick="document.getElementById('modal-importar-csv').close()" style="margin-bottom: 0;">Cancelar</button>
                    <button type="submit" style="margin-bottom: 0;">Procesar Archivo Masivo</button>
                </div>
            </footer>
        </form>
    </article>
</dialog>

```


3. En el bloque de scripts inferior, agregar los escuchadores de eventos para que HTMX cierre el diálogo y refresque la grilla automáticamente:
```javascript
document.body.addEventListener("cerrarModalImportar", function () {
    document.getElementById('modal-importar-csv').close();
    document.getElementById('archivo_csv').value = ''; // Limpiamos el input file
    document.getElementById('feedback-importacion').innerHTML = '';
});

```


*(Asegurarse de que tu contenedor principal de listado de modelos escuche el trigger `refrescarListaModelos` en su atributo `hx-trigger` para actualizar la tabla de inmediato).*

---

## Información detallada de la estructura de la base de datos en sql.
Busca información detallada de la base de datos (tablas y relaciones) en sql en "docs\temporal\planilla_rgm.sql".