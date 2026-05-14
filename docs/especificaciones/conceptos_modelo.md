# Especificaciones del Módulo: Conceptos Modelo

Este documento detalla de manera estructurada los artefactos, funciones, estructuras de datos y dependencias directamente relacionadas con la funcionalidad de **Conceptos Modelo** (plantillas base para regímenes laborales) en el Panel de Administración.

## 1. Artefactos Involucrados

Los principales archivos y artefactos de código fuente que componen este módulo son:

*   **Modelo de Datos (Struct):** Definido en `internal/models/core.go`
*   **Repositorio (Persistencia):** `internal/repository/concepto_modelo_repository.go`
*   **Manejador HTTP (Handler):** `internal/handlers/concepto_modelo_handler.go`
*   **Enrutador (Routes):** Definiciones de rutas en `internal/routes/routes.go`
*   **Interfaz de Usuario (HTML):** Plantilla y fragmentos HTMX en `ui/templates/admin/conceptos_modelo_ui.html`

---

## 2. Funciones y Parámetros

### 2.1. Handler (`ConceptoModeloHandler`)
Controla el flujo de la aplicación, maneja las peticiones HTTP y renderiza las vistas.

*   `VistaUI(w http.ResponseWriter, r *http.Request)`: Carga la página principal del módulo. Utiliza otros repositorios para enviar al frontend la lista de regímenes, conceptos maestros y clasificadores MEF.
*   `Listar(w http.ResponseWriter, r *http.Request)`: Obtiene la lista completa de conceptos modelo y renderiza el fragmento HTMX `tabla_modelos`.
*   `Crear(w http.ResponseWriter, r *http.Request)`: Procesa el envío del formulario de creación. Recibe arreglos (ej. IDs de regímenes), crea el registro en base de datos, envía la cabecera `HX-Trigger: cerrarModal` a HTMX y devuelve la tabla actualizada.
*   `EditarUI(w http.ResponseWriter, r *http.Request)`: Obtiene el ID del concepto por parámetro URL (`?id=X`), busca sus datos actuales (incluyendo regímenes asociados) y renderiza el fragmento `formulario_editar_modelo` pre-llenando los campos.
*   `Actualizar(w http.ResponseWriter, r *http.Request)`: Procesa el envío del formulario de edición. Convierte parámetros, actualiza base de datos, dispara el cierre del modal HTMX y recarga la tabla.
*   `Eliminar(w http.ResponseWriter, r *http.Request)`: Captura el ID a eliminar, ejecuta el borrado en el repositorio y devuelve la tabla actualizada.

### 2.2. Repository (`ConceptoModeloRepository`)
Encapsula la lógica de acceso a la base de datos PostgreSQL.

*   `NewConceptoModeloRepository(db *sql.DB) *ConceptoModeloRepository`: Función constructora.
*   `ObtenerTodos() ([]models.ConceptoModelo, error)`: Realiza un `SELECT` complejo con `JOIN` a tablas maestras (`conceptos_maestros`, `clasificadores_mef`) y utiliza `STRING_AGG` para agrupar en una sola cadena los nombres de los regímenes laborales asociados a cada concepto.
*   `ObtenerPorID(id int) (*models.ConceptoModelo, error)`: Ejecuta una consulta para obtener los datos principales y luego otra consulta secundaria para cargar los IDs de los regímenes en un arreglo (`RegimenesIDs`).
*   `Crear(c *models.ConceptoModelo) error`: Utiliza transacciones SQL (`Begin`, `Commit`, `Rollback`) para insertar primero el registro base en `conceptos_modelo` (obteniendo el ID generado) y luego iterar sobre `RegimenesIDs` insertando en la tabla pivote `regimen_concepto_modelo`.
*   `Actualizar(c *models.ConceptoModelo) error`: Mediante una transacción, actualiza los campos base, borra todas las relaciones previas en `regimen_concepto_modelo` (`DELETE`) y las re-inserta basándose en la nueva selección de regímenes.
*   `Eliminar(id int) error`: Ejecuta un borrado simple en `conceptos_modelo`. Depende de las restricciones de llave foránea (`ON DELETE CASCADE`) en la base de datos para limpiar tablas pivote.

---

## 3. Estructura de Datos

El modelo de datos central se define de la siguiente manera:

```go
// ConceptoModelo representa la plantilla base para cada régimen laboral
type ConceptoModelo struct {
	ID                  int    `json:"id"`
	ConceptoID          int    `json:"concepto_id"`
	NombrePersonalizado string `json:"nombre_personalizado"`
	FrecuenciaMeses     string `json:"frecuencia_meses"`
	ClasificadorID      *int   `json:"clasificador_id"` // Puntero (puede ser nulo en BD)
	EsExtraordinario    bool   `json:"es_extraordinario"`
	RequiereMonto       bool   `json:"requiere_monto"`
	CreatedAt           string `json:"created_at"`

	// Campos "virtuales" obtenidos mediante JOINs para la interfaz de usuario
	RegimenesIDs        []int  `json:"regimenes_ids"` // Arreglo para mapear checkboxes en la UI
	RegimenesNombres    string `json:"regimenes_nombres,omitempty"`
	ConceptoCodigo      string `json:"concepto_codigo,omitempty"`
	ConceptoDescripcion string `json:"concepto_descripcion,omitempty"`
	ClasificadorCodigo  string `json:"clasificador_codigo,omitempty"`
}
```

---

## 4. Interacción con Otros Paquetes y Módulos

El módulo **Conceptos Modelo** interactúa estrechamente con:

1.  **Enrutador Global y Middleware (`internal/routes`, `internal/middleware`):**
    *   Las rutas se agrupan en `registrarRutasAdmin` (ej. `/admin/conceptos-modelo/lista`).
    *   Utilizan obligatoriamente el middleware `RequireRole("super_admin")` para restringir el acceso.
2.  **Otros Repositorios (Inyección de Dependencias):**
    *   El `ConceptoModeloHandler` requiere instancias de `PuestoRepository` (para listar regímenes laborales) y `ConceptoTenantRepository` (para consultar catálogos base y clasificadores), evidenciando que el formulario agrupa datos de múltiples fuentes.
3.  **Sistema de Base de Datos (`database/sql`):**
    *   Uso exhaustivo de transacciones para consistencia y manejo de valores nulos con `sql.NullInt64` o `sql.NullString` (por ejemplo, el `ClasificadorID`).
4.  **Sistema Frontend (HTMX y Plantillas HTML):**
    *   Aprovecha el paquete nativo `html/template`.
    *   Interactúa con HTMX para manipulación dinámica del DOM sin recargar la página:
        *   Devuelve fragmentos HTML pre-renderizados en lugar de JSON en casi todas sus rutas de lectura o mutación.
        *   Se integra en la capa HTTP devolviendo eventos `w.Header().Set("HX-Trigger", "cerrarModal")` para dar instrucciones al lado del cliente.
