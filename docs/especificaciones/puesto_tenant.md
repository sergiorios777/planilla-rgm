# Especificaciones del Módulo: Puesto (Plazas del Tenant)
Este documento detalla los artefactos, funciones, estructuras de datos y dependencias relacionados con el módulo **Puesto Tenant** (la gestión de plazas o puestos de trabajo que tiene cada municipalidad / tenant).

## 1. Artefactos Involucrados
Los principales archivos y artefactos de código fuente que componen este módulo son:

*   **Modelo de Datos (Struct):** Definidos en `internal/models/core.go` (`Puesto`, `PuestoConcepto`, `ConceptoAsignacion`, entre otros DTOs).
*   **Repositorio (Persistencia):** `internal/repository/puesto_repository.go` (`PuestoRepository`).
*   **Manejador HTTP (Handler):** `internal/handlers/puesto_handler.go` (`PuestoHandler`).
*   **Enrutador (Routes):** Definiciones de rutas protegidas de entorno *Tenant* en `internal/routes/routes.go` (bajo el grupo `/tenant/puestos`).
*   **Interfaz de Usuario (HTML):** Plantillas y fragmentos HTMX en `ui/templates/tenant/puestos_ui.html`.

---

## 2. Funciones y Parámetros

### 2.1. Handler (`PuestoHandler`)
Maneja las peticiones HTTP y renderiza las interfaces usando HTMX.

*   `VistaUI(w http.ResponseWriter, r *http.Request)`: Carga la página base y provee a los formularios las listas de Metas Presupuestales, Fuentes/Rubros y Regímenes Laborales.
*   `Listar(w http.ResponseWriter, r *http.Request)`: Obtiene los parámetros de búsqueda y paginación (`buscar`, `meta_id`, `regimen_id`, `estado`, `limite`, `pagina`). Extrae el ID del tenant de la sesión, obtiene los registros paginados y renderiza el fragmento HTMX `tabla_puestos`.
*   `Crear(w http.ResponseWriter, r *http.Request)`: Procesa el formulario de creación. Recibe los parámetros de meta, fuente, régimen, sueldo, etc. y utiliza `services.PuestoService`  (`CrearPuestoConPlantilla`) para crear el puesto y asignarle los conceptos de forma automática.
*   `Editar(w http.ResponseWriter, r *http.Request)`: Busca un puesto específico por ID (`?id=X`) junto con las opciones de catálogos y renderiza el fragmento `formulario_editar`.
*   `Actualizar(w http.ResponseWriter, r *http.Request)`: Recibe los datos de edición, actualiza el registro en base de datos y vuelve a llamar a la vista UI completa.
*   `FormularioCrearUI(w http.ResponseWriter, r *http.Request)`: Devuelve el fragmento `formulario_crear` cargado con los catálogos vacíos para resetear el panel.
*   `AsignarConceptosUI(w http.ResponseWriter, r *http.Request)`: Carga el modal con la estructura de conceptos del puesto utilizando `ObtenerConceptosParaAsignacion` y renderizando el fragmento `formulario_asignar_conceptos`.
*   `GuardarAsignacion(w http.ResponseWriter, r *http.Request)`: Procesa el formulario enviado por HTMX desde el modal de asignación, leyendo los checks e iterando por los montos especificados, para luego usar el repositorio para almacenar en lote la estructura de pago.

### 2.2. Repository (`PuestoRepository`)
Gestiona la capa de acceso a la base de datos PostgreSQL.

*   **Consultas de Listado:** `ObtenerVacantes(tenantID int)`, `ObtenerTodos(tenantID int)`, `ObtenerTodosPaginacion(...)`. Estas consultas incluyen varios JOINs a tablas de metas, fuentes y regímenes para rellenar los datos descriptivos.
*   **Catálogos Base:** `ObtenerRegimenes()`.
*   **Operaciones CRUD:** `Crear(p *models.Puesto)`, `Actualizar(p *models.Puesto)`, `ObtenerPorID(id int, tenantID int)`.
*   **Gestión de Conceptos del Puesto (Estructura de Gasto):**
    *   `ObtenerConceptosTenantPorCodigosSUNAT(...)`: Para buscar traducciones de códigos PDT.
    *   `AsignarConceptosAPuesto(puestoID int, conceptoTenantIDs []int, sueldoBase float64)`: Realiza una operación de inserción masiva (Bulk Insert) protegida mediante una transacción (`tx.Begin()`). Itera sobre los IDs insertándolos en `puesto_conceptos`. Utiliza la cláusula `ON CONFLICT (puesto_id, concepto_tenant_id) DO NOTHING` para prevenir duplicados.
    *   `ObtenerConceptosParaAsignacion(...)`: Devuelve la matriz combinada de conceptos activos del Tenant indicando cuáles ya están asignados a la plaza (con `LEFT JOIN`).
    *   `GuardarAsignacionConceptos(...)`: Transacción que limpia configuraciones anteriores (`DELETE`) e inserta la nueva estructura de conceptos seleccionada en la vista modal.
    *   `ObtenerConceptosModeloPorRegimen(...)`: Trae los conceptos locales que corresponden al "modelo" del régimen laboral especificado, excluyendo aportaciones previsionales.
    *   `RestaurarPlantillaBase(puestoID int, tenantID int, regimenID int)`: Ejecuta una transacción SQL en tres pasos: (1) Borra los conceptos actuales del puesto protegiendo las configuraciones previsionales mediante la cláusula `cma.codigo NOT IN ('0601', '0606', '0607', '0608')`. (2) Obtiene la lista de conceptos base para el régimen (excluyendo también las pensiones). (3) Inserta la nueva lista obtenida de la base utilizando `ON CONFLICT DO NOTHING` para asegurar integridad sin fallos.
---

## 3. Estructura de Datos

El modelo fundamental para esta característica reside en `internal/models/core.go`:

### 3.1. `Puesto`
Representa una plaza ("silla") dentro de la municipalidad.
```go
type Puesto struct {
	ID                  int     `json:"id"`
	TenantID            int     `json:"tenant_id"`
	MetaID              int     `json:"meta_id"`
	FuenteRubroID       int     `json:"fuente_rubro_id"`
	RegimenID           int     `json:"regimen_id"`
	RegimenCodigo       string  // Código del Régimen Laboral
	Nombre              string  `json:"nombre"`
	SueldoPresupuestado float64 `json:"sueldo_presupuestado"`
	Estado              string  `json:"estado"` // VACANTE u OCUPADO
	Activo              bool    `json:"activo"`
	EsDietario          bool    `json:"es_dietario"`

	// Campos auxiliares (Extraídos mediante JOINs para la UI)
	MetaCodigo       string `json:"meta_codigo,omitempty"`
	FuenteRubroDesc  string `json:"fuente_rubro_desc,omitempty"`
	RegimenDesc      string `json:"regimen_desc,omitempty"`
	RequiereRevision bool
}
```

### 3.2. `PuestoConcepto`
Detalla los conceptos que configuran el costo de una plaza específica.
```go
type PuestoConcepto struct {
	ID               int      `json:"id"`
	PuestoID         int      `json:"puesto_id"`
	ConceptoTenantID int      `json:"concepto_tenant_id"`
	Monto            *float64 `json:"monto"`
	Activo           bool     `json:"activo"`

	// Campos Auxiliares para pintar la UI
	NombrePersonalizado string `json:"nombre_personalizado,omitempty"`
	ConceptoTipo        string `json:"concepto_tipo,omitempty"`
	Clasificador        string `json:"clasificador,omitempty"`
	// ... entre otros
}
```

### 3.3. Estructuras de Ayuda
*   `ConceptoAsignacion`: Estructura DTO para generar dinámicamente la lista de asignaciones manuales en el modal (`AsignarConceptosUI`).

---

## 4. Interacción con Otros Paquetes y Módulos

El flujo del Módulo Puestos interactúa fuertemente con:

1.  **Middleware de Seguridad (`internal/middleware`):**
    *   Usa `middleware.RequireAuth`, garantizando el contexto de ejecución bajo un `TenantID` válido que aísla la data de un inquilino a otro.
2.  **Manejador de Servicios (`internal/services/puesto_service.go`):**
    *   Durante la creación de un nuevo puesto (`Crear`), el Handler delega a `PuestoService.CrearPuestoConPlantilla` para realizar la inserción e inmediatamente configurar los conceptos base que deriven del régimen asociado de la nueva plaza, interactuando con el Repositorio.
3.  **Sistema de Enrutamiento (`internal/routes`):**
    *   Definido bajo rutas protegidas con prefijos de recursos como `/tenant/ui/puestos` y sub-rutas `/tenant/puestos/*` para acciones de estado o carga de UI HTMX parcial.
4.  **Repositorios Secundarios (`MetaRepository` y `FuenteRubroRepository`):**
    *   Al estar el Puesto profundamente integrado en el presupuesto institucional, el Handler y Repositorio consultan listas desde las entidades globales del Tenant (Metas Presupuestales y Fuentes de Financiamiento).
5.  **Sistema de Plantillas y DOM Reactivo (HTMX):**
    *   La plantilla `puestos_ui.html` utiliza modales nativos de HTML (`<dialog>`), eventos como `hx-on::after-settle`, cargas diferidas con `hx-trigger="load"` para los listados.
    *   Se hace uso extensivo de `hx-vals` y eventos `hx-trigger` interactivos (ej. `keyup changed delay:500ms`) en los inputs del formulario de filtros para ofrecer actualizaciones en vivo de la tabla. El manejo de cierres de modal se hace combinando triggers de HTMX (`w.Header().Set("HX-Trigger", "cerrarModalAsignacion")`) y scripts locales.

---

## 5. Acciones de la Tabla de Puestos (`tabla_puestos`)

La tabla que lista las plazas (`tabla_puestos` en `puestos_ui.html`) contiene una columna de "Configuración" con tres botones de acción principales por cada registro:

1.  **Editar:**
    *   **Acción:** Carga el formulario de edición de la plaza.
    *   **Endpoint:** `GET /tenant/puestos/editar?id={ID}`
    *   **Comportamiento:** Usa HTMX para inyectar la respuesta (`formulario_editar`) en el contenedor `#contenedor-formulario`, permitiendo la edición en línea.

2.  **⚙️ Costos:**
    *   **Acción:** Navega a la vista detallada de los conceptos y montos aplicados a la plaza (panel de costos).
    *   **Endpoint:** `GET /tenant/puestos-conceptos/ui?puesto_id={ID}`
    *   **Comportamiento:** Usa HTMX para reemplazar todo el contenedor principal (`#contenido-tenant`) con la interfaz específica de administración de costos del puesto.

3.  **💰 Estructura:**
    *   **Acción:** Abre un modal para habilitar/deshabilitar conceptos específicos para el puesto y definir montos base o fijos.
    *   **Endpoint:** `GET /tenant/puestos/asignar-conceptos-ui?id={ID}`
    *   **Comportamiento:** HTMX inyecta la lista de asignaciones (`formulario_asignar_conceptos`) en `#contenido-modal-asignacion`. El diálogo nativo `<dialog id="modal-asignacion-puesto">` se abre automáticamente (`hx-on::after-settle="this.showModal()"`) mostrando los checkboxes y campos de montos.
