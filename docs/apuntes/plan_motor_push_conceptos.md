# El Diseño del Motor de Propagación (Push Sync)
El objetivo es iterar sobre todas las municipalidades activas y ejecutar una transacción atómica que inserte los conceptos filtrados y, acto seguido, copie sus relaciones laborales. El plan de implementación detallado tiene 3 fases.

## Fase 1: El Repositorio (El Motor SQL)
*Archivo: internal/repository/concepto_tenant_repository.go*

Vamos a crear una nueva función "avanzada" que agrupe la clonación de conceptos y la clonación de regímenes dentro de una sola transacción, aceptando los filtros que mencionaste.

**Instrucción para la implementación:**
Añadir la función `SincronizarDesdeModeloAvanzado(tenantID int, modo string, fechaInicio string, fechaFin string) error`.

1. Abrir Transacción: tx, err := r.db.Begin().
2. Construir Query Dinámico:
   * La consulta base será un INSERT INTO conceptos_tenant ... SELECT ... FROM conceptos_modelo WHERE 1=1.
   * Si modo == "FECHAS", concatenar AND created_at BETWEEN $2 AND $3 y pasar las fechas.
   * Si modo == "EXTRAORDINARIOS", concatenar AND es_extraordinario = true.
   * Mantener el ON CONFLICT (tenant_id, modelo_id) DO NOTHING.
3. Ejecutar Inserción de Conceptos: Usar tx.Exec() con el query construido.
4. Sincronizar Regímenes (Reutilizar Lógica):
   * Usar exactamente la misma consulta SQL que existe en ClonarRelacionesRegimen, pero ejecutada dentro de esta transacción (tx.Exec).
5. Confirmar: tx.Commit().

## Fase 2: El Handler (El Director de Orquesta)
*Archivos: internal/handlers/concepto_modelo_handler.go y internal/routes/routes.go*

El Handler recibirá la petición del Super Admin, buscará a todos los inquilinos y ejecutará el motor para cada uno.

**Instrucción para la implementación:**
1. Inyección de Dependencias: En ConceptoModeloHandler, necesitamos agregar TenantRepo *repository.TenantRepository a la estructura para poder consultar la lista de municipalidades. Actualizar routes.go para pasarle esta nueva dependencia.
2. Nueva Función Sincronizar(w http.ResponseWriter, r *http.Request):
   * Leer los valores del formulario HTMX: modo, fecha_inicio, fecha_fin.
   * Obtener todas las municipalidades usando h.TenantRepo.ObtenerTodos("").
   * Crear un bucle for sobre las municipalidades filtrando solo las que estén activas (if !tenant.Activo { continue }).
   * Por cada inquilino activo, llamar a h.ConceptoTenantRepo.SincronizarDesdeModeloAvanzado(tenant.ID, modo, fecha_inicio, fecha_fin).
   * Enviar una respuesta de éxito a HTMX cerrando el modal.

## Fase 3: La Interfaz de Usuario (El Panel de Control)
*Archivo: ui/templates/admin/conceptos_modelo_ui.html*

Necesitamos un botón y un modal elegante donde el Super Admin pueda elegir qué y cómo sincronizar.

**Instrucción para la implementación:**
1. Botón de Acción: Al lado del botón "Nuevo Concepto Modelo", agregar un botón con un icono de nube/sincronización que abra un nuevo <dialog id="modal-sincronizar">.
2. Modal de Sincronización:
   * Crear un formulario con hx-post="/admin/conceptos-modelo/sincronizar".
   * Un <select name="modo"> con 3 opciones:
     - FALTANTES (Opción A: Todos los conceptos nuevos que falten).
     - EXTRAORDINARIOS (Opción C: Solo conceptos extraordinarios).
     - FECHAS (Opción B: Por rango de fechas).
   * Dos campos <input type="date"> (Inicio y Fin) que se muestren o habiliten visualmente cuando se elija la opción por fechas.
   * Un botón de "Iniciar Sincronización Masiva".
