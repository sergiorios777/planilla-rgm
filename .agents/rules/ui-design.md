# Guía y Reglas de Diseño de Interfaz de Usuario (UI/UX)

Este documento establece los estándares de diseño y desarrollo frontend obligatorios para todas las vistas y plantillas HTML del sistema **Planillas RGM**.

---

## 1. Manifiestos de Diseño de Referencia
Todo desarrollo visual o modificación de vistas HTML debe alinearse estrictamente con los manifiestos del proyecto:
- [DESIGN.md](file:///c:/server/www/planilla-rgm/DESIGN.md): Especificación del sistema de diseño, paleta de colores (Admin Governance vs. Tenant Operations), tipografía (`14px` Inter, JetBrains Mono para datos numéricos), elevación y modales `<dialog>`.
- [design_v2.md](file:///c:/server/www/planilla-rgm/design_v2.md): Evolución de diseño refinado, estados HTMX y componentes de planillas.

---

## 2. Filosofía de Pico.css v2 (HTML Semántico Primero)

El desarrollo frontend prioriza el uso de **HTML semántico nativo** según la filosofía de Pico.css. Se debe evitar la proliferación de clases utilitarias manuales (`.flex-row-between`, `.mb-0`, `.w-full`) cuando existan etiquetas HTML o contenedores nativos de Pico.css.

### Principios Fundamentales:
1. **Estructura Semántica Novedosa**:
   - Usar `<main>`, `<article>`, `<header>`, `<footer>`, `<section>` y `<aside>` para maquetar layout y tarjetas sin agregar clases `div` innecesarias.
   - Para rejillas responsivas de 2 o más columnas, usar el contenedor nativo `<div class="grid">`.
   - Para menús y barras de navegación, usar `<nav>` con listas `<ul>/<li>`.
   - Para estados y etiquetas destacadas, preferir la etiqueta semántica `<mark class="badge-success">` sobre `<span>`.
2. **Modales Nativos**:
   - Usar la etiqueta HTML5 `<dialog>` con `<article>`, `<header>` y `<footer>` nativos en lugar de librerías de modales pesadas.
3. **Formularios Limpios**:
   - Agrupar campos usando `<fieldset>`, `<label>` y `<input>` aprovechando los estilos automáticos de Pico.css.

---

## 3. Excepciones y Componentes Complejos

Se permite el uso de clases personalizadas definidas en [custom.css](file:///c:/server/www/planilla-rgm/ui/static/css/custom.css) **únicamente en los siguientes casos de complejidad**:

1. **Tablas Financieras Densas**:
   - Uso de `table.condensed.striped` y la clase `.total-row` para filas de totales con borde primario de 2px y alineación derecha numérica (`.stat-mono`, `.text-right`).
2. **Componentes con JS o Terceros**:
   - Desplegables avanzadas con **TomSelect** (`.select-con-buscador`, `.ts-control`).
   - Drawers móviles de navegación lateral (`aside`, `#sidebar-backdrop`).
3. **Estados Visuales de HTMX**:
   - Indicadores de carga asíncrona (`.htmx-indicator`, `.htmx-request`, `button[aria-busy="true"]`).
4. **Tarjetas Específicas del Dominio**:
   - Tarjetas de resumen (`.callout-box`, `.banner-info`, `.concepto-box`, `.notif-item`).

---

## 4. Reglas para la Modificación de Estilos (`custom.css`)

- **No agregar micro-utilitarios estilo Tailwind/Bootstrap**: No crear clases atómicas como `.p-2`, `.mt-3`, `.w-60` salvo estricta necesidad de compatibilidad.
- **Sobrescribir CSS Custom Properties**: Todo ajuste de color, tamaño o espaciado global debe realizarse mediante variables `--pico-*` en `:root` o en los selectores de tema (`body.admin-theme`, `body.tenant-theme`).

---

## 5. Catálogo de Componentes Estandarizados Reutilizables

Todas las vistas de la aplicación deben emplear obligatoriamente los siguientes componentes estandarizados desarrollados para mantener la consistencia estética y funcional:

1. **Paginación Numérica HTMX Reutilizable (`componente_paginacion`)**:
   - **Regla**: En cualquier vista con tablas o listados paginados (`trabajadores`, `contratos`, `puestos`, `metas`, `conceptos`, `asistencia`), es **obligatorio** invocar `{{template "componente_paginacion" .Paginacion}}` usando en Go `models.CalcularPaginacion(...)` y `models.PaginacionDTO`. Queda estrictamente prohibido maquetar barras de paginación manuales en 2 filas.

2. **Modal de Confirmación Global de Acciones Irreversibles**:
   - **Regla**: **Prohibido usar `hx-confirm="..."` o `window.confirm()` nativos del navegador**. Toda acción delicada o irreversible (cerrar planilla, eliminar contrato, restaurar conceptos, dar de baja) debe activar el modal global reutilizable mediante los atributos declarativos `data-confirm-title`, `data-confirm-message`, `data-confirm-badge` y `data-confirm-btn`.

3. **Tarjetas de Resumen Financiero y KPIs (Bento Grid)**:
   - **Regla**: Todas las métricas acumuladas (Ingresos, Retenciones, Aportes, Costo Total, Neto a Pagar, Trabajadores Activos) deben presentarse mediante la rejilla responsiva Bento Grid (`grid-auto-fit-200 gap-md mb-lg`) usando la tipografía monoespaciada `.stat-mono` (`JetBrains Mono`) y los badges cromáticos del dominio (`badge-success`, `badge-danger`, `badge-info`, `badge-warning`, `badge-purple`).

4. **Acciones de Tabla (Botones Planos e Iconos SVG)**:
   - **Regla**: En las celdas de acciones de las tablas, usar **exclusivamente** botones planos de icono `<button class="btn-icon">` o menús `details.dropdown-kebab` con iconos del sprite SVG (`#icono-*`). Queda prohibido el uso de emoticonos de texto plano o `role="group"` en tablas.

---

## 6. Ciclo de Vida de Scripts en HTMX y Componentes de Terceros (TomSelect)

Dado que la navegación en la plataforma opera como una Single Page Application (SPA) basada en intercambios parciales de HTMX (`hx-target="#contenido-tenant"`), se deben acatar estrictamente los siguientes estándares para prevenir fugas de memoria, event listeners "zombies" y conflictos de inicialización:

1. **Prohibición de Listeners Globales en Plantillas Secundarias**:
   - **Regla**: Queda **estrictamente prohibido** registrar listeners globales en `document.body`, `document` o `window` (como `document.body.addEventListener('htmx:afterSettle', ...)` o `document.addEventListener('DOMContentLoaded', ...)`) dentro de plantillas HTML inyectadas por HTMX.
   - **Razón**: `document.body` nunca se descarga entre navegaciones HTMX; los listeners añadidos en vistas dinámicas se acumulan, persisten indefinidamente e interceptan de forma descontrolada los eventos de otras pantallas.

2. **Centralización de Inicializadores Estándar en `app.js`**:
   - **Regla**: Toda inicialización automática de componentes de uso masivo (como `.select-con-buscador` estándar) debe residir **exclusivamente en [`ui/static/js/app.js`](file:///c:/server/www/planilla-rgm/ui/static/js/app.js)**, ejecutándose una sola vez a nivel de aplicación.

3. **Selectores con Lógica Interactiva Personalizada (`data-custom-tomselect`)**:
   - **Regla**: Si un elemento `<select>` requiere un comportamiento interactivo propio (como callbacks `onChange`, inserción dinámica de filas, agregación aditiva en tiempo real o filtrado personalizado):
     1. Debe declararse obligatoriamente con el atributo `data-custom-tomselect="true"` para que el auto-inicializador global de `app.js` lo ignore.
     2. En su script local, debe validar y destruir explícitamente cualquier instancia previa antes de inicializar la nueva:
        ```javascript
        const el = document.getElementById('mi-select-personalizado');
        if (el && typeof TomSelect !== 'undefined') {
            if (el.tomselect) {
                el.tomselect.destroy();
            }
            new TomSelect(el, {
                // Configuración y callbacks personalizados
            });
        }
        ```


