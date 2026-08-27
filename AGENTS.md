# Directivas de Desarrollo - Planillas RGM

Este documento contiene los estándares principales para el mantenimiento y desarrollo del proyecto **Planillas RGM**.

## Estándares de Arquitectura Backend y Server-Driven UI

- **Reglas de Arquitectura Go + HTMX**: Consulta la regla [.agents/rules/architecture-go-htmx.md](file:///c:/server/www/planilla-rgm/.agents/rules/architecture-go-htmx.md) para el flujo unidireccional estricto (`Handler ──► Service ──► Repository`), aislamiento multi-tenant obligatorio (`tenant_id`), renderizado de fragmentos con `html/template`, alcance progresivo de JavaScript y prohibición de endpoints REST/JSON para manipulación del DOM.

## Estándares de Diseño y Frontend (UI/UX)

- **Manifiestos de Diseño**: Toda vista HTML y hoja de estilos debe estar alineada a los documentos [DESIGN.md](file:///c:/server/www/planilla-rgm/DESIGN.md) y [design_v2.md](file:///c:/server/www/planilla-rgm/design_v2.md).
- **Regla de Diseño UI**: Consulta la regla [.agents/rules/ui-design.md](file:///c:/server/www/planilla-rgm/.agents/rules/ui-design.md) para conocer las pautas de HTML semántico (Pico.css v2), modales `<dialog>`, tablas financieras, excepciones en [custom.css](file:///c:/server/www/planilla-rgm/ui/static/css/custom.css) y el ciclo de vida de scripts/TomSelect en intercambios HTMX.
