## Pregunta:
Gemini, tengo un problema:

He realizado las consultas con algunos contadores públicos y jefes de recursos humanos de municipalidades acerca de la declaración en PDT/Plame; las conclusiones son: 1) varían los criterios para asignar los códigos sunat en algunos conceptos y 2) para algunos conceptos nuevos pueden variar o no todavía tienen códigos sunat definivos.

Esto tiene consecuencias en el enfoque de nuestra aplicación:
1. Los códigos sunat de los conceptos de ingresos, retenciones y aportes se definen en los conceptos modelo y conceptos tenant (que puede ser personalizado), y los códigos sunat asignados pueden diferir del criterio de los usuarios responsables de la planilla y su declaración ante sunat en las municipalidades.
2. Aunque podemos cambiar los códigos sunat directamente en los conceptos tenant utilizando el módulo correspondiente, su diseño no tiene por finalidad el análisis específico de los códigos sunat que se están utilizando en la planilla específica a declarar, esto vuelve engorroso los cambios requeridos.

Sugerencias de mejora:
1. Crear una vista específica para el análisis y verificación rápida de los códigos sunat utilizados en la planilla y su modificación directa en la tabla `planilla_conceptos`.

Los archivos principales a revisar:
- Estructura de las tablas de la bd: @docs\temporal\estructura_planilla_rgm.sql
- @ui\templates\tenant\planillas_ui.html
- @ui\templates\tenant\conceptos_tenant_ui.html

Ayúdame a plantear esto como una solicitud estructurada para Antigravity 2.0, para generar el análisis detallado de factibilidad de esta propuesta.

## Respuesta:

El planteamiento es acertado y resuelve una necesidad crítica: en la práctica municipal peruana, los criterios de fiscalización o convenios específicos obligan a ajustar el código de la Tabla 22 de SUNAT (ingresos, tributos y descuentos) antes de emitir los archivos de importación para el PDT Planilla Electrónica (PLAME).

Dado que la tabla `planilla_conceptos` ya almacena la columna `codigo_sunat` de forma desacoplada como un *snapshot* del cálculo, la base de datos ya está preparada para soportar estas modificaciones sin alterar el histórico ni forzar cambios permanentes en el catálogo general si no se desea.

---

### Consideraciones técnicas clave para incluir en la solicitud

* **Edición agrupada vs. fila por fila:** Para evitar que el usuario edite cientos de registros individuales, la vista debe presentar un **resumen agrupado por concepto** dentro de la planilla (`concepto_tenant_id`, nombre, clasificador, código SUNAT actual, total de trabajadores afectados y monto acumulado). Al cambiar el código SUNAT a nivel de concepto, se actualiza masivamente en todos los `planilla_conceptos` de esa planilla.


* **Catálogo de validación:** El selector de códigos SUNAT debe alimentarse de `conceptos_maestros` (donde `origen = 'sunat'`) para impedir el ingreso de códigos inexistentes que invaliden la estructura de importación del PLAME.


* **Opción de sincronización inversa (Opcional):** Un control tipo checkbox (*"Actualizar también como predeterminado en el concepto"*) para que, si el cambio de criterio es permanente, actualice `conceptos_tenant` hacia el futuro.



---

### Solicitud estructurada para Antigravity 2.0 CLI

```markdown
Actúa como Arquitecto de Software Go y Desarrollador Full-Stack (Server-Driven UI con HTMX + PostgreSQL).

### Contexto del Problema
En las declaraciones mensuales del PDT/PLAME (Planilla Electrónica - SUNAT), los contadores y gestores de RRHH en las municipalidades requieren verificar y en ocasiones reasignar los códigos SUNAT de los conceptos calculados para una planilla específica (debido a discrepancias de criterio normativo o conceptos extraordinarios/nuevos).

Actualmente los códigos SUNAT se heredan desde `conceptos_maestros` / `conceptos_tenant`[cite: 2], pero no existe una herramienta para auditar y ajustar de forma rápida los códigos SUNAT directamente en los registros ya generados de la planilla (`planilla_conceptos.codigo_sunat`) antes de generar los reportes/interfaces de exportación[cite: 2].

### Archivos de Referencia en el Workspace
1. Estructura BD: `docs/temporal/estructura_planilla_rgm.sql`[cite: 2]
2. Vistas y Templates:
   - `ui/templates/tenant/planillas_ui.html`
   - `ui/templates/tenant/conceptos_tenant_ui.html`
   - `ui/templates/tenant/planilla_rubros_metas_ui.html` (referencia de arquitectura Server-Driven UI)

### Objetivo del Análisis y Propuesta
Evaluar la factibilidad y diseñar la solución técnica para implementar una vista/módulo de **"Auditoría y Reasignación de Códigos SUNAT para PLAME"** asociada a una planilla en estado borrador.

### Tareas Requeridas en el Entregable
1. **Análisis de Impacto en Base de Datos y Consultas SQL:**
   - Diseñar la consulta agregada para agrupar los conceptos de una planilla (`planilla_id`) mostrando: `concepto_tenant_id`, `nombre_en_boleta`, `tipo_concepto`, `codigo_sunat` actual, total de personas afectadas y suma total del monto[cite: 2].
   - Sentencia `UPDATE` optimizada para actualizar en lote `planilla_conceptos.codigo_sunat` para un `concepto_tenant_id` dentro de una `planilla_id` específica[cite: 2].

2. **Diseño de la Arquitectura UI/UX con HTMX:**
   - Proponer la ubicación y flujo de acceso desde `planillas_ui.html` (ej. botón "Auditar SUNAT / PLAME" en la botonera de acciones de la planilla).
   - Estructurar la plantilla HTMX (`planilla_sunat_codigos_ui.html`) que permita:
     * Visualizar la tabla de conceptos agrupados con selector/dropdown de códigos oficiales de SUNAT (provenientes de `conceptos_maestros` donde `origen = 'sunat'`)[cite: 2].
     * Guardado inline / reactivo con `hx-post` o `hx-patch` por cada concepto modificado sin recargar toda la página.
     * Checkbox opcional: *"Guardar como valor predeterminado para futuras planillas"* (actualiza también `conceptos_tenant`).

3. **Plan de Implementación en Go:**
   - Definir Handlers HTTP requeridos en Go (Standard Library).
   - Definir métodos del repositorio/servicio para la obtención de datos agrupados y la actualización en lote de `planilla_conceptos`[cite: 2].
   - Validaciones de consistencia (por ejemplo, impedir cambios si la planilla ya se encuentra en estado `'CERRADA'`)[cite: 2].

```