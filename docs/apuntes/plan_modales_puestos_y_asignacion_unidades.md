# Plan de Implementación: Modales de Puestos y Asignación Orgánica

## Fase 1: Capa de Datos (Modelos y Repositorio)
Para dar soporte a los nuevos campos, primero debemos asegurar que las estructuras de Go puedan capturar y enviar valores nulos (ya que un puesto podría crearse inicialmente sin una oficina asignada).

**Instrucciones para el Agente:**

1. Modificar el Struct `Puesto` (`internal/models/core.go` o donde esté definido):

   - Añadir el campo `UnidadOrganicaID *int json:"unidad_organica_id"` (usamos un puntero a entero para mapear correctamente el estado NULL de la base de datos).

   - Añadir el campo `CodigoAirhsp string json:"codigo_airhsp"`.

   - Añadir el campo de lectura `UnidadOrganicaNombre string json:"unidad_organica_nombre,omitempty"` para poder mostrar el nombre de la oficina directamente en las listas y tablas sin alterar la inmutabilidad histórica.

2. Modificar `internal/repository/puesto_repository.go`:

   - Actualizar el método `Crear(p *models.Puesto)`: Modificar la sentencia `INSERT INTO puestos` para incluir las columnas `unidad_organica_id` y `codigo_airhsp`. Asegurar el uso de parámetros posicionales (`$X`) y pasar `p.UnidadOrganicaID` y `p.CodigoAirhsp`.

   - Actualizar el método `Actualizar(p *models.Puesto)` (si existe) o el equivalente: Asegurar que el `UPDATE` guarde las modificaciones de estas dos columnas.

   - Actualizar las consultas de lectura (`Listar`, `ObtenerPorID`): Modificar el `SELECT` principal agregando un `LEFT JOIN unidades_organicas u ON p.unidad_organica_id = u.id` para capturar `COALESCE(u.nombre, 'Sin asignar') AS unidad_organica_nombre e incluir p.codigo_airhsp`.

3. Crear método helper en `internal/repository/organigrama_repository.go`:

   - Implementar la función `ObtenerUnidadesDelOrganigramaActivo(tenantID int) ([]models.UnidadOrganica, error)`.

   - **La Consulta SQL:**
     ```sql
     SELECT id, nombre, tipo
     FROM unidades_organicas
     WHERE tenant_id = $1
     AND organigrama_id = (SELECT id FROM organigramas WHERE tenant_id = $1 AND activo = true LIMIT 1)
     ORDER BY nombre ASC
     ```
     *(Esto garantiza que el formulario de puestos solo liste las oficinas de la Ordenanza Municipal vigente en este milisegundo).*

## Fase 2: Capa de Control (Handlers y Rutas)
Modificaremos el Handler de puestos para inyectarle el repositorio de organigramas. Esto nos permitirá cargar las oficinas disponibles para los selects de los formularios.

**Instrucciones para el Agente:**

1. Modificar `internal/handlers/puesto_handler.go`:

   * Agregar la dependencia `OrganigramaRepo *repository.OrganigramaRepository` al struct `PuestoHandler`.

   * Actualizar `VistaUI`: Llamar a `h.OrganigramaRepo.ObtenerUnidadesDelOrganigramaActivo(tenantID)` e inyectar el slice de unidades orgánicas en el mapa `datos` (ej: `"Unidades": unidades`) para que esté disponible en el formulario de creación.

   * Crear el método `EditarUI(w http.ResponseWriter, r *http.Request)`:
     * Leer el `id` del puesto desde la URL Query.
     * Obtener los datos del puesto (`ObtenerPorID`).
     * Obtener las unidades orgánicas activas, las metas presupuestales, las fuentes de rubro y los regímenes (mismos catálogos que usa `VistaUI`).
     * Renderizar y retornar únicamente el fragmento HTML del contenido del modal de edición (el bloque `formulario_editar_puesto`).

   * Actualizar la lectura de formularios (`Crear` y `Actualizar`):

     * Al procesar `r.FormValue("unidad_organica_id")`, si el valor viene vacío o es `"0"`, asignar `nil` a `Puesto.UnidadOrganicaID`. De lo contrario, parsearlo a entero y pasar su dirección de memoria.
     * Capturar `r.FormValue("codigo_airhsp")`.

2. Modificar `internal/routes/routes.go`:

   * Al inicializar `puestoHandler := handlers.PuestoHandler{...}`, inyectar el `organigramaRepo`.

   * Registrar la nueva ruta GET para la interfaz de edición por HTMX: `/tenant/puestos/editar-ui`.

## Fase 3: Interfaz de Usuario (Templates HTML con Modales)
Esta es la parte visual clave. Reemplazaremos los flujos antiguos por modales que se abren nativamente con JavaScript/HTMX y escuchan eventos de cierre seguros.

**Instrucciones para el Agente:**

1. Modificar `ui/templates/tenant/puestos_ui.html`:

   * Tabla de Puestos: Agregar dos nuevas columnas en el `<thead>` y `<tbody>`: Código AIRHSP y Unidad Orgánica (usando `.UnidadOrganicaNombre`).

   * Botón de Editar en la Tabla: Modificar el botón clásico de edición para que dispare HTMX:

     ```HTML
     <button class="outline"
             hx-get="/tenant/puestos/editar-ui?id={{.ID}}"
             hx-target="#contenido-modal-editar-puesto"
             hx-on::after-request="document.getElementById('modal-editar-puesto').showModal()">
         ✏️ Editar
     </button>
     ```
   * Estructura fija de Modales al final del archivo:

     * Crear `<dialog id="modal-nuevo-puesto">` que contenga el formulario de creación (el que antes estaba expuesto en la página).
     * Crear `<dialog id="modal-editar-puesto"><article id="contenido-modal-editar-puesto"></article></dialog>`.

2. Diseño de los Selects e Inputs dentro de los Formularios:

   * En ambos formularios (Crear y Editar), inyectar los siguientes campos estructurados con Pico CSS:

     ```HTML
     <div class="grid">
         <label for="unidad_organica_id">Unidad Orgánica (Oficina):
             <select id="unidad_organica_id" name="unidad_organica_id">
                 <option value="0">— Sin asignar —</option>
                 {{range .Unidades}}
                     <option value="{{.ID}}" {{if eq .ID $.Puesto.UnidadOrganicaID}}selected{{end}}>[{{.Tipo}}] {{.Nombre}}</option>
                 {{end}}
             </select>
         </label> 

         <label for="codigo_airhsp">Código AIRHSP (MEF):
             <input type="text" id="codigo_airhsp" name="codigo_airhsp" value="{{.Puesto.CodigoAirhsp}}" placeholder="Ej: 001245">
         </label>
     </div>
     ```
   * Cierre automático: Asegurar que los formularios usen `hx-swap="none"` y `hx-on::after-request="document.getElementById('ID_DEL_MODAL').close()"` combinados con las cabeceras de trigger de Go para refrescar la lista de puestos en tiempo real tras guardar.