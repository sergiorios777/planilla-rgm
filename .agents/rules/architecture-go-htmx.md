# Reglas de Arquitectura: Go + Server-Driven UI (HTMX)

Este documento establece las directivas y estándares arquitectónicos obligatorios para el desarrollo y mantenimiento del backend en Go y su integración con HTMX en **Planillas RGM**.

---

## 1. Flujo Arquitectónico y Responsabilidades de Capas

El sistema opera bajo una arquitectura en capas desacoplada con flujo unidireccional estricto:

```text
Browser (HTMX) ──► HTTP Handler ──► Service (Lógica / Tx) ──► Repository / DB
      ▲                                                              │
      └─────────── HTML Fragment / Binario (Template Go / Stream) ───┘
```

### Handlers (Capa de Transporte y Control HTTP - `internal/handlers`)
- **Aislamiento de Tenant**: Extraer y validar obligatoriamente la sesión y el identificador de inquilino (`tenant_id`) del contexto de la petición (`r.Context()`).
- **Validación de Entrada**: Parsear, sanitizar y validar parámetros HTTP (Query, Form, Path).
- **Delegación**: Invocar al servicio correspondiente para mutaciones, cálculos o flujos orquestados.
- **Presentación**: Renderizar fragmentos o vistas completas mediante plantillas Go (`html/template`).
- **Control HTMX**: Controlar el comportamiento del cliente mediante cabeceras de respuesta estándar de HTMX (`HX-Trigger`, `HX-Redirect`, `HX-Reswap`) o bloques fuera de banda (`hx-swap-oob="true"`).
- **Prohibición**: Queda prohibido escribir consultas SQL directamente dentro de los Handlers.

### Services (Capa de Dominio y Negocio - `internal/services`)
- **Lógica Pura de Negocio**: Concentrar el 100% de las fórmulas y reglas laborales/tributarias peruanas (cálculo de planillas, CTS, gratificaciones, vacaciones, asignación familiar, renta de 5ta categoría, retenciones judiciales y aportes AFP/ONP).
- **Control Transaccional**: Administrar transacciones en base de datos (`sql.Tx`) cuando una operación involucre múltiples mutaciones atómicas.
- **Desacoplamiento HTTP**: Los servicios deben ser completamente agnósticos al protocolo HTTP. Queda prohibido recibir `*http.Request` o escribir en `http.ResponseWriter` dentro de los servicios. Deben recibir y retornar modelos o tipos de Go puros y `error`.

### Repositories (Capa de Persistencia - `internal/repository`)
- **Consultas Aisladas**: Ejecutar sentencias SQL nativas y parametrizadas contra PostgreSQL.
- **Seguridad Multi-Tenancy**: Toda consulta de lectura, actualización o eliminación debe incluir estrictamente el filtro de inquilino (`WHERE tenant_id = $1` o equivalente) para impedir la fuga o acceso no autorizado entre tenants.
- **Mapeo a Modelos**: Transformar las filas resultantes (`*sql.Rows`, `*sql.Row`) en structs del dominio (`internal/models`).
- **Aislamiento HTTP**: Los repositorios no deben conocer conceptos de transporte HTTP, cookies, sesiones ni cabeceras.

---

## 2. Capa Cliente (HTML / HTMX / JavaScript Progresivo)

### Principio Server-Driven UI (HDA)
- **HTML como Transferencia de Estado**: Prohibido crear endpoints JSON tipo REST exclusivamente para que JavaScript construya interfaces dinámicas o manipule el DOM en operaciones CRUD. Toda mutación o consulta de vista debe solicitar fragmentos HTML renderizados por el servidor en Go.
- **Excepciones Legítimas de Respuesta**: Solo se admiten respuestas que no sean fragmentos HTML en:
  1. **Descargas de Archivos Binarios**: Boletas en PDF (`pdf_service`), reportes en Excel (`excelize`) y archivos de exportación en texto plano (ej. PDT/PLAME).
  2. **APIs Máquina a Máquina**: Endpoints de integración explícitamente autorizados y protegidos por tokens de servicio.

### Alcance Estricto de JavaScript
- **Mejora Progresiva**: JavaScript debe reservarse únicamente para interacciones visuales en el cliente:
  - Inicialización y control de selectores avanzados (**TomSelect**).
  - Apertura, cierre y accesibilidad de modales nativos `<dialog>`.
  - Máscaras de campos de texto (ej. montos en soles, DNI/RUC).
  - Escucha de eventos del ciclo de vida de HTMX (`htmx:confirm`, `htmx:afterSwap`, `htmx:afterSettle`, `htmx:responseError`).
- **Cero Lógica de Negocio en el Cliente**: Queda terminantemente prohibido calcular montos de planillas, tributos o validaciones complejas de dominio en JavaScript.
- **Prevención de Fugas de Memoria**: Seguir los estándares de ciclo de vida especificados en [.agents/rules/ui-design.md](file:///c:/server/www/planilla-rgm/.agents/rules/ui-design.md) (no registrar listeners globales duplicados en plantillas inyectadas dinámicamente).

---

## 3. Prácticas Idiomáticas en Comunicación HTMX

1. **Feedback y Errores**:
   - En lugar de respuestas JSON con campos de error, devolver fragmentos HTML con alertas/banners semánticos o usar cabeceras `HX-Trigger: {"mostrarNotificacion": {"tipo": "error", "mensaje": "..."}}`.
   - Para errores de validación de formulario, retornar el fragmento del formulario con código HTTP `422 Unprocessable Entity` para que HTMX pueda inyectar los mensajes de validación sin recargar la página.

2. **Actualizaciones en Múltiples Zonas (OOB Swaps)**:
   - Cuando una acción en una tabla deba refrescar métricas o un contador de resumen en la barra lateral o cabecera, incluir el fragmento auxiliar con el atributo `hx-swap-oob="true"` en la misma respuesta HTML.
