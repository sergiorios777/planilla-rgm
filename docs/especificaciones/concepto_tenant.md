# Especificaciones del Módulo: Concepto Tenant (Conceptos Locales)

Este documento detalla de manera estructurada los artefactos, funciones, estructuras de datos y dependencias relacionados con el módulo **Conceptos Locales** (la configuración específica de conceptos remunerativos que tiene cada municipalidad / tenant).

## 1. Artefactos Involucrados

Los principales archivos y artefactos de código fuente que componen este módulo son:

*   **Modelo de Datos (Struct):** Definido en `internal/models/core.go` (`ConceptoTenant`).
*   **Repositorio (Persistencia):** `internal/repository/concepto_tenant_repository.go` (`ConceptoTenantRepository`).
*   **Manejador HTTP (Handler):** `internal/handlers/concepto_tenant_handler.go` (`ConceptoTenantHandler`).
*   **Enrutador (Routes):** Definiciones de rutas de entorno *Tenant* en `internal/routes/routes.go`.
*   **Interfaz de Usuario (HTML):** Plantilla y fragmentos HTMX en `ui/templates/tenant/conceptos_tenant_ui.html`.

---

## 2. Funciones y Parámetros

### 2.1. Handler (`ConceptoTenantHandler`)
Maneja las peticiones HTTP y la interacción con la vista (HTMX).

*   `VistaUI(w http.ResponseWriter, r *http.Request)`: Carga la página base y provee a los formularios las listas de Conceptos Maestros y Clasificadores MEF.
*   `Listar(w http.ResponseWriter, r *http.Request)`: Obtiene los parámetros de búsqueda y paginación (`buscar`, `pagina`, `limite`), extrae el ID del tenant de la sesión, obtiene los registros paginados y renderiza el fragmento HTMX `tabla_conceptos_tenant`.
*   `Crear(w http.ResponseWriter, r *http.Request)`: Procesa el formulario de creación. Valida errores (como restricciones UNIQUE en base de datos si ya existe el "Nombre Personalizado") e inyecta alertas HTML en caso de error, o recarga la tabla en caso de éxito.
*   `EditarUI(w http.ResponseWriter, r *http.Request)`: Busca un concepto específico por ID (`?id=X`) junto con las opciones de catálogos y renderiza el formulario de edición.
*   `Actualizar(w http.ResponseWriter, r *http.Request)`: Recibe los datos de edición, actualiza el registro en base de datos, dispara el header `HX-Trigger: recargarTablaConceptos` para recargar la vista, y devuelve el formulario limpio de creación.
*   `FormularioCrearUI(w http.ResponseWriter, r *http.Request)`: Devuelve explícitamente el fragmento `formulario_crear` vacío para resetear el panel.
*   `FilaUI(w http.ResponseWriter, r *http.Request)`: Devuelve únicamente un nodo HTML `<tr>` formateado con `badges` y colores para representar de manera asíncrona un solo registro en la tabla tras una actualización parcial.
*   `Restaurar(w http.ResponseWriter, r *http.Request)`: Invoca la clonación de los `conceptos_modelo` predeterminados hacia el tenant actual (evitando duplicados gracias a restricciones de BD).

### 2.2. Repository (`ConceptoTenantRepository`)
Gestiona la capa de acceso a base de datos PostgreSQL.

*   `NewConceptoTenantRepository(db *sql.DB) *ConceptoTenantRepository`: Función constructora.
*   `ObtenerMaestros()`: Trae el catálogo SUNAT base (`conceptos_maestros`) para cargar selects.
*   `ObtenerTodos(tenantID int)` / `ObtenerTodosPaginacion(...)`: Ejecuta sentencias `SELECT` con `JOIN` hacia `conceptos_maestros` y `clasificadores_mef`, filtrando estrictamente por `tenant_id`. La versión paginada retorna la lista, el total de registros y permite filtros por nombre/código.
*   `Crear(ct *models.ConceptoTenant)`: Inserta una nueva configuración local, retornando el ID generado.
*   `Actualizar(ct *models.ConceptoTenant)` / `ActualizarCompleto(...)`: Ejecuta `UPDATE` para cambiar los datos permitidos del concepto local.
*   `ObtenerClasificadores()`: Trae del MEF solo clasificadores con `nivel = 6` (detalle) y de transacciones específicas (`2.1.%`, `2.3.%`, `2.6.%`) que están activos.
*   `ObtenerPorID(id int, tenantID int)`: Devuelve el detalle del concepto y sus metadatos (SIAF y Maestros) asegurando que pertenezca al inquilino correcto.
*   `ClonarDesdeModelo(tenantID int)`: Ejecuta un volcado masivo (`INSERT INTO ... SELECT ...`) desde la tabla `conceptos_modelo` hacia la tabla `conceptos_tenant`, aprovechando `ON CONFLICT DO NOTHING` para restaurar o inicializar catálogos sin corromper la BD.

---

## 3. Estructura de Datos

El modelo fundamental para esta característica reside en `internal/models/core.go`:

```go
// ConceptoTenant es la configuración local (por municipalidad) de un concepto maestro
type ConceptoTenant struct {
	ID                  int    `json:"id"`
	TenantID            int    `json:"tenant_id"`   // Clave foránea para aislar datos por municipalidad
	ConceptoID          int    `json:"concepto_id"` // Apunta a conceptos_maestros (Catálogo SUNAT/PDT)
	NombrePersonalizado string `json:"nombre_personalizado"`
	FrecuenciaMeses     string `json:"frecuencia_meses"`
	ClasificadorID      *int   `json:"clasificador_id"` // Opcional (Puntero a ClasificadorMEF)
	Activo              bool   `json:"activo"`
	EsExtraordinario    bool   `json:"es_extraordinario"`

	// Campos Auxiliares (Extraídos mediante JOINs para la UI)
	ConceptoCodigo     string `json:"concepto_codigo,omitempty"`
	ConceptoNombre     string `json:"concepto_nombre,omitempty"`
	ConceptoTipo       string `json:"concepto_tipo,omitempty"`
	ClasificadorCodigo string `json:"clasificador_codigo,omitempty"`
}
```

---

## 4. Interacción con Otros Paquetes y Módulos

El flujo del Módulo de Conceptos Locales interactúa fuertemente con:

1.  **Middleware de Seguridad (`internal/middleware`):**
    *   Usa `middleware.RequireAuth`, por lo tanto requiere que el usuario haya iniciado sesión bajo el contexto de un `Tenant` válido (Municipalidad).
2.  **Sistema de Enrutamiento (`internal/routes`):**
    *   Definido bajo la sección `registrarRutasTenant` con el prefijo `/tenant/conceptos-locales/*`.
3.  **Otros Catálogos / Repositorios:**
    *   No depende de otros repositorios de manera inyectada en el *Handler* (a diferencia de *ConceptoModelo*), sino que el mismo `ConceptoTenantRepository` hace las consultas simples a las tablas globales `conceptos_maestros` y `clasificadores_mef` porque no requiere lógicas cruzadas complejas fuera de sus fronteras.
4.  **Sistema de Plantillas y DOM Reactivo (HTMX):**
    *   `Conceptos_tenant_ui.html` maneja eventos avanzados.
    *   El manejador puede devolver vistas completas, tablas pre-renderizadas o simplemente *nodos HTML sueltos* (ej. función `FilaUI` devuelve un `<tr id="concepto-X">`) para inyecciones quirúrgicas y mejoras drásticas en rendimiento.
    *   Uso constante del OOB Swap (`hx-swap-oob="true"`) para actualizar alertas de error sin perder el estado del formulario actual.
5.  **Módulo de Administración (`internal/handlers/admin_handler.go`):**
    *   **Creación de Inquilinos:** La función `CrearInquilino` del `AdminHandler` interactúa directamente con `ConceptoTenantRepository`. Una vez que se crea exitosamente un nuevo `Tenant` (Municipalidad) en la base de datos, el manejador de administración invoca automáticamente el método `ClonarDesdeModelo(nuevoTenant.ID)`.
    *   **Inicialización Automática:** Esta interacción garantiza que, en el instante en que se registra un nuevo inquilino, su catálogo local de conceptos se inicialice clonando los datos predeterminados de la tabla `conceptos_modelo`. Esto permite a los nuevos inquilinos contar con una base operativa inmediata sin requerir configuración manual posterior.
