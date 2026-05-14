# Especificaciones del Módulo: Puesto Concepto (Estructura de Costos)

Este documento detalla los artefactos, funciones, estructuras de datos y dependencias relacionados con el módulo **Puesto Concepto**. Este módulo gestiona la estructura de costos (ingresos, retenciones y aportes) asignada individualmente a cada puesto (plaza) dentro de un Tenant.

---

## 1. Artefactos Involucrados

Los principales archivos y artefactos de código fuente que componen este módulo son:

*   **Modelo de Datos (Struct):** Estructuras definidas en `internal/models/core.go` (principalmente `PuestoConcepto` y `ConceptoPlanilla`).
*   **Repositorio (Persistencia):** `internal/repository/puesto_concepto_repository.go` (`PuestoConceptoRepository`).
*   **Manejador HTTP (Handler):** `internal/handlers/puesto_concepto_handler.go` (`PuestoConceptoHandler`).
*   **Enrutador (Routes):** Definiciones de rutas protegidas bajo el grupo `/tenant/puestos-conceptos` en `internal/routes/routes.go`.
*   **Interfaz de Usuario (HTML):** Plantillas y fragmentos HTMX en `ui/templates/tenant/puestos_conceptos_ui.html`.

---

## 2. Funciones y Parámetros

### 2.1. Handler (`PuestoConceptoHandler`)
Maneja las peticiones HTTP, implementa la lógica de evaluación de montos manuales y renderiza las interfaces usando HTMX.

*   `VistaUI(w http.ResponseWriter, r *http.Request)`: Carga la pantalla base de configuración. Obtiene los conceptos asignados y disponibles. Aplica una "lógica inteligente" evaluando si los conceptos requieren un ingreso manual basándose en `config.ConceptosQueRequierenMonto`.
*   `Listar(w http.ResponseWriter, r *http.Request)`: Similar a `VistaUI` pero devuelve únicamente el fragmento HTML de la tabla de conceptos asignados (`tabla_asignados`) procesada, ideal para recargas vía HTMX.
*   `Crear(w http.ResponseWriter, r *http.Request)`: Asigna un nuevo concepto a la plaza, procesando los parámetros `puesto_id`, `concepto_tenant_id` y `monto`. Recarga la vista posteriormente.
*   `Eliminar(w http.ResponseWriter, r *http.Request)`: Desasigna (elimina) un concepto de la plaza usando su `id` y devuelve a la vista principal.
*   `RestaurarCostosBase(w http.ResponseWriter, r *http.Request)`: Busca el régimen del puesto y utiliza `PuestoRepository.RestaurarPlantillaBase` para limpiar la plaza actual e inyectar los conceptos por defecto. Desencadena un evento HTMX `refreshCostosBase`.
*   `EditarMontoUI(w http.ResponseWriter, r *http.Request)`: Devuelve un fragmento HTML pequeño con un `<input>` y botones de acción (Guardar/Cancelar) para la edición en línea del monto de un concepto asignado.
*   `ActualizarMonto(w http.ResponseWriter, r *http.Request)`: Recibe el `id` y el nuevo `monto`, actualiza la base de datos y refresca la tabla completa en el cliente.

### 2.2. Repository (`PuestoConceptoRepository`)
Gestiona la persistencia de las relaciones entre puestos y conceptos.

*   `ObtenerAsignados(puestoID int, tenantID int)`: Devuelve un slice de `PuestoConcepto` con los conceptos que **ya forman parte** de la plaza. Incluye joins para traer nombres, códigos y clasificadores.
*   `ObtenerDisponibles(puestoID int, tenantID int)`: Retorna un slice de `ConceptoTenant` con los conceptos que **aún no están asignados** al puesto, facilitando la población del selector de adición.
*   `Crear(pc *models.PuestoConcepto)`: Persiste una nueva asignación de concepto al puesto.
*   `Eliminar(id int)`: Borra el registro de la relación.
*   `ActualizarMonto(id int, monto float64)`: Actualiza de manera atómica el monto fijo establecido.
*   `ObtenerParaCalculo(puestoID int)`: Retorna un arreglo de `ConceptoPlanilla` extrayendo los conceptos activos y formateándolos para ser inyectados directamente en el Motor de Cálculo o Simulador de Planillas.

---

## 3. Estructura de Datos

El modelo subyacente para persistir esta información se encuentra en `internal/models/core.go`:

### 3.1. `PuestoConcepto`
Representa la relación directa y la configuración específica (ej. un monto fijo) de un concepto asociado a un puesto.

```go
type PuestoConcepto struct {
    ID               int      `json:"id"`
    PuestoID         int      `json:"puesto_id"`
    ConceptoTenantID int      `json:"concepto_tenant_id"`
    Monto            *float64 `json:"monto"` // Puntero para diferenciar 0 de NULL
    Activo           bool     `json:"activo"`

    // Campos auxiliares para la UI (No persistidos directamente en la tabla)
    NombrePersonalizado string `json:"nombre_personalizado,omitempty"`
    ConceptoTipo        string `json:"concepto_tipo,omitempty"`
    Clasificador        string `json:"clasificador,omitempty"`
    MaestroCodigo       string `json:"maestro_codigo,omitempty"`
    MontoIngresado      bool   `json:"monto_ingresado,omitempty"`
    RequiereMontoManual bool   `json:"requiere_monto_manual,omitempty"`
}
```

### 3.2. Estructuras de Ayuda
*   `ConceptoPlanilla`: DTO utilizado por la función `ObtenerParaCalculo` para transferir la estructura al Motor de Cálculo de nóminas.

---

## 4. Interacción con Otros Paquetes y Módulos

El submódulo **Puesto Concepto** actúa como un puente entre la gestión de plazas y el motor de nóminas, interactuando con:

1.  **Configuración Global (`internal/config`)**:
    *   Lee el mapa `config.ConceptosQueRequierenMonto` (códigos SUNAT) para inferir visualmente qué conceptos necesitan que el usuario ingrese obligatoriamente un monto para que el motor de cálculo pueda operar correctamente (por ejemplo, Asignación Familiar u otros bonos no calculables).
2.  **Módulo Puestos (`PuestoRepository`)**:
    *   Utiliza este repositorio para obtener detalles del puesto padre (como el Régimen Laboral) requerido durante la acción de `RestaurarCostosBase`.
3.  **Middleware de Seguridad (`internal/middleware`)**:
    *   Todas sus rutas están protegidas bajo `middleware.RequireAuth`, asegurando el entorno multi-tenant.
4.  **Motor de Cálculo / Planillas**:
    *   Provee la data base (lista de `ConceptoPlanilla`) requerida para ejecutar la simulación o el cálculo definitivo de los montos a percibir por el trabajador.

---

## 5. Acciones de la Interfaz de Usuario (`puestos_conceptos_ui.html`)

La pantalla de gestión ofrece controles interactivos gestionados por HTMX para evitar recargas completas de la página:

1.  **Volver al Cuadro de Puestos:**
    *   **Acción:** Retorna a la pantalla principal de listado de plazas.
    *   **Comportamiento HTMX:** `hx-get="/tenant/ui/puestos"` reemplazando el `#contenido-tenant`.
2.  **Restaurar Plantilla:**
    *   **Acción:** Borra la configuración manual actual y restablece los conceptos según el régimen de la plaza.
    *   **Comportamiento HTMX:** Ejecuta un `hx-post` a `/tenant/puestos-conceptos/restaurar`. Dispone de un `hx-confirm` como barrera de seguridad para evitar pérdida accidental de datos.
3.  **Añadir Concepto a la Plaza (Formulario):**
    *   **Acción:** Permite seleccionar un concepto de la lista de *Disponibles* e indicar un monto fijo opcional.
    *   **Comportamiento HTMX:** `hx-post="/tenant/puestos-conceptos/crear"` que actualiza toda la vista para recalcular las listas de asignados y disponibles.
4.  **Ingresar/Editar Monto (Edición Inline):**
    *   **Acción:** Si un concepto requiere un monto manual (identificado por la UI con "⚠️ Ingresar Monto") o si ya tiene uno ("S/ XXX.XX"), el usuario puede hacer clic sobre él.
    *   **Comportamiento HTMX:** Se activa un `hx-get` que reemplaza temporalmente el bloque actual (`hx-target="this"`) por un input numérico (`EditarMontoUI`).
    *   **Sub-acciones:**
        *   **Guardar (✅):** `hx-put` a `/actualizar-monto` para registrar el cambio en la BD.
        *   **Cancelar (❌):** `hx-get` para volver al modo solo lectura.
5.  **Quitar (Eliminar):**
    *   **Acción:** Remueve el concepto de la estructura del puesto.
    *   **Comportamiento HTMX:** `hx-delete="/tenant/puestos-conceptos/eliminar"` con `hx-confirm` para proteger contra clics erróneos.
