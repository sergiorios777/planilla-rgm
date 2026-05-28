# Plan de creación del módulo de AFP en admin del saas

## Ideas preliminares

* Crear el módulo de AFP en el panel admin de la aplicación, con formularios modales para crear y editar AFP.
* Con un "botón" para importar los valores de las comisiones y primas de seguros del sistema privado de pensiones por mes, que publica la SBS. La estructura de esta información es:
  Al mes de devengue 2026-05 1/
  
  | AFP  | COMISIÓN SOBRE FLUJO (% Remuneración Bruta Mensual)| COMISIÓN ANUAL SOBRE SALDO| PRIMA DE SEGUROS (%) 3/ (% Remuneración Bruta Mensual)| APORTE OBLIGATORIO AL FONDO DE PENSIONES (% Remuneración Bruta Mensual) | REMUNERACIÓN MÁXIMA ASEGURABLE  |
  |---|---|---|---|---|---|
  | HABITAT| 1,47%| 1,25%| 1,37%| 10,00%| 12 598,91|
  | INTEGRA| 1,55%| 0,78%| 1,37%| 10,00%| 12 598,91|
  | PRIMA| 1,60%| 1,25%| 1,37%| 10,00%| 12 598,91|
  | PROFUTURO| 1,69%| 0,68%| 1,37%| 10,00%| 12 598,91|
  |---|---|---|---|---|---|
  
  1/ Las comisiones sobre la remuneración y las primas retenidas correspondientes a un determinado mes deben pagarse dentro de los 5 primeros dÍas útiles del mes siguiente.
  2/ A partir de Enero de 1997 se eliminó el cobro de Comisión Fija.
  3/ Porcentaje a descontar sobre la Remuneración Bruta hasta el límite determinado por el Reglamentode la Ley del SPP(Remuneración Máxima Asegurable Art. 67° del Título VII del Compendio de Normas reglamentarias del SPP).
  *** A partir de Febrero de 2023, el componente de flujo de la Comisión Mixta es 0%, resultando únicamente en Comisión anulado sobre saldo.
  
  Del cuadro anterior omitimos la columna de remuneración máxima asegurable, que se actualizará en otro módulo.
* La información del cuadro anterior lo va a elaborar el super_admin en formato csv que será utilizado para la importación. Realizar validaciones robustas, también aclarar al usuario cómo deden ser presentados los valor, p.e. como 1.47% o como 0.0147

## Estructura de las tablas involucradas

Las tablas son afps y afp_tasas_mensuales.

```txt
planilla_rgm=# \d afps
                                     Table "public.afps"
   Column   |         Type          | Collation | Nullable |             Default
------------+-----------------------+-----------+----------+----------------------------------
 id         | integer               |           | not null | nextval('afps_id_seq'::regclass)
 codigo_sbs | character varying(10) |           |          |
 nombre     | character varying(50) |           | not null |
 activo     | boolean               |           |          | true
Indexes:
    "afps_pkey" PRIMARY KEY, btree (id)
    "afps_codigo_sbs_key" UNIQUE CONSTRAINT, btree (codigo_sbs)
Referenced by:
    TABLE "afp_tasas_mensuales" CONSTRAINT "afp_tasas_mensuales_afp_id_fkey" FOREIGN KEY (afp_id) REFERENCES afps(id)
    TABLE "trabajadores" CONSTRAINT "trabajadores_afp_id_fkey" FOREIGN KEY (afp_id) REFERENCES afps(id)


planilla_rgm=# \d afp_tasas_mensuales
                                      Table "public.afp_tasas_mensuales"
        Column        |     Type     | Collation | Nullable |                     Default
----------------------+--------------+-----------+----------+-------------------------------------------------
 id                   | integer      |           | not null | nextval('afp_tasas_mensuales_id_seq'::regclass)
 afp_id               | integer      |           |          |
 anio                 | integer      |           | not null |
 mes                  | integer      |           | not null |
 aporte_obligatorio   | numeric(5,4) |           |          | 0.1000
 comision_flujo       | numeric(5,4) |           | not null |
 comision_mixta_flujo | numeric(5,4) |           | not null |
 prima_seguro         | numeric(5,4) |           | not null |
 comision_anual_saldo | numeric(5,4) |           |          | 0
Indexes:
    "afp_tasas_mensuales_pkey" PRIMARY KEY, btree (id)
    "afp_tasas_mensuales_afp_id_anio_mes_key" UNIQUE CONSTRAINT, btree (afp_id, anio, mes)
Foreign-key constraints:
    "afp_tasas_mensuales_afp_id_fkey" FOREIGN KEY (afp_id) REFERENCES afps(id)
```

## Propuesta Final del Plan de Implementación: Módulo de AFPs y Tasas Mensuales (SaaS Admin)
### Fase 1: Capa de Datos (Modelos y Repositorios)
Primero debemos mapear tus tablas en Go y crear las funciones para listar las AFPs y guardar/actualizar la matriz de tasas del mes.

**Instrucciones para el Agente:**

1. Modelos: Crear el archivo `internal/models/afp.go`:

   * Struct `AFP`: `ID int, Nombre string, CodigoSBS string`.

   * Struct `AFPTasaMensual`: `ID int, AfpID int, Anio int, Mes int, AporteObligatorio float64, ComisionFlujo float64, ComisionMixtaFlujo float64, PrimaSeguro float64, ComisionAnualSaldo float64`.

   * Struct auxiliar `AFPTasaVista`: Que combine los datos de la AFP con sus tasas para renderizar la tabla fácilmente en el HTML.

2. Repositorio: Crear `internal/repository/afp_repository.go`:

   * `ObtenerAFPs()`: Retorna la lista de las administradoras.

   * `ObtenerTasasPorMes(anio int, mes int)`: Hace un `LEFT JOIN` entre `afps` y `afp_tasas_mensuales` para devolver la matriz (incluso si hay valores nulos o en cero para ese mes).

   * `GuardarTasasMensuales(tasas []models.AFPTasaMensual)`: Abre una transacción (`tx.Begin()`) y ejecuta un bucle con un `INSERT ... ON CONFLICT (afp_id, anio, mes) DO UPDATE SET` ... para asegurar que no se dupliquen registros si el administrador corrige un dato del mismo mes. (_Asegurarse de crear el índice único compuesto en la BD para_ `afp_id`, `anio`, `mes` si no existe).

### Fase 2: Lógica de Transformación (El Servicio de Importación)
Como los CSV en Perú suelen tener comas para decimales (ej. `1,47%`), necesitamos que Go limpie esos textos y los convierta a números matemáticos puros (`0.0147`).

**Instrucciones para el Agente:**

1. Crear `internal/services/afp_service.go` con el método `ProcesarCSV(file io.Reader, anio int, mes int) error`.

2. El algoritmo del servicio:

   * Leer el catálogo de AFPs desde el repositorio y crear un mapa en memoria (ej. `mapaAFP["HABITAT"] = 1`).

   * Usar `encoding/csv` de Go para leer el archivo subido.

   * Ignorar la primera fila (las cabeceras).

   * Para cada fila siguiente:

     - Leer el nombre de la AFP (Columna 0), pasarlo a mayúsculas y buscar su ID en el mapa.

     - Crear una función `limpiarPorcentaje(string) float64` que quite el símbolo `%`, reemplace la coma , por un punto . y divida entre 100 si es necesario (o lo guarde directo según decidas estructurar tu cálculo, usualmente `1.47` se guarda y se divide en el cálculo final).

     - Mapear las columnas del CSV a los campos: *Flujo, Saldo, Prima, Aporte Obligatorio.*

   * Pasar el slice resultante al repositorio `GuardarTasasMensuales`.

### Fase 3: El Handler y Rutas
El controlador ahora recibirá un formulario `Multipart` en lugar de campos de texto estándar.

**Instrucciones para el Agente:**

1. En `internal/handlers/afp_handler.go`:

   * `VistaUI`: Renderiza la pantalla principal.

   * `ImportarCSV`:

     - Llamar a `r.ParseMultipartForm`.

     - Recuperar los valores `anio` y `mes` seleccionados en el formulario.

     - Recuperar el archivo `r.FormFile("archivo_csv")`.

     - Pasar el archivo al `AfpService`.

     - Retornar un fragmento HTMX de éxito y disparar un evento para recargar la tabla visual.

2. En `internal/routes/routes.go`:

   * Registrar `POST /admin/afps/importar` asociado a este nuevo método.

### Fase 4: La Interfaz de Usuario Visual (Upload & View)
La pantalla tendrá un formulario de carga y una tabla debajo que muestre el estado actual de la base de datos.

**Instrucciones para el Agente:**

1. En `ui/templates/admin/afps_ui.html`, construir un Grid:

   * Izquierda (Formulario de Carga):

```HTML
<form hx-post="/admin/afps/importar" hx-encoding="multipart/form-data" hx-target="#mensaje-importacion" hx-swap="innerHTML">
    <div class="grid">
        <label>Año <input type="number" name="anio" value="2026" required></label>
        <label>Mes <input type="number" name="mes" value="5" required></label>
    </div>
    <label>Archivo CSV de la SBS
        <input type="file" name="archivo_csv" accept=".csv" required>
    </label>
    <button type="submit">Subir e Importar Tasas</button>
    <div id="mensaje-importacion"></div>
</form>
```

   * Derecha (Vista Previa de Datos):

     - Controles simples para seleccionar un año/mes y visualizar en una tabla lo que ya existe en la base de datos.

     - Esta tabla escuchará el evento HTMX (ej. `hx-trigger="tasasActualizadas from:body"`) para recargarse mágicamente justo después de que la importación CSV tenga éxito.

**¿Por qué este enfoque es excelente?**
Tú, como Super Admin, solo abres tu Excel de la SBS una vez al mes, lo guardas como .csv, seleccionas "2026", "Mayo", subes el archivo y Go se encarga de limpiar toda la basura visual (espacios, porcentajes) y hacer los UPSERT en la base de datos en menos de 50 milisegundos.
