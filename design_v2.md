# SISTEMA DE DISEÑO REFINADO: PLANILLAS RGM (V2)

Este manifiesto evoluciona la identidad visual de **Planillas RGM** manteniendo la filosofía de "Zero JS" (preferencia por HTML semántico y CSS nativo) y la integración profunda con **HTMX** y **Pico CSS v2**.

---

## 1. FUNDAMENTOS VISUALES

### 1.1 Tipografía y Escala
- **Base**: `14px` (aplicado en `html`). Optimiza la densidad para aplicaciones de gestión financiera.
- **Font Stack**: `Inter, system-ui, sans-serif`.
- **Pesos**: 
  - `400` (Regular) para lectura general.
  - `600` (Semibold) para encabezados y totales de tablas.
  - `700` (Bold) para estados críticos y marca.

### 1.2 Paleta de Colores Evolucionada

#### Núcleo Semántico (Pico CSS Overrides)
| Variable | Admin (SaaS) | Tenant (Municipal) | Uso |
| :--- | :--- | :--- | :--- |
| `--pico-primary` | `#7c3aed` (Violeta) | `#0284c7` (Azul Cielo) | Acciones principales |
| `--pico-primary-hover` | `#6d28d9` | `#0369a1` | Hover de botones |
| `--pico-background-color` | `#f8fafc` | `#ffffff` | Fondo de página |
| `--pico-card-background` | `#ffffff` | `#ffffff` | Contenedores y Modales |

#### Estados de Planilla y Badges (UX Refinada)
- **Ingresos / Activo**: `bg: #dcfce7`, `text: #166534` (Verde esmeralda profundo).
- **Descuentos / Inactivo**: `bg: #fee2e2`, `text: #991b1b` (Rojo carmesí).
- **Aportes / Ordinaria**: `bg: #e0f2fe`, `text: #075985` (Azul institucional).
- **Borrador / Pendiente**: `bg: #fef9c3`, `text: #854d0e` (Ámbar).
- **Especial / Extraordinaria**: `bg: #f5f3ff`, `text: #5b21b6` (Púrpura).

---

## 2. COMPONENTES Y ESTRUCTURA HTMX

### 2.1 Tablas de Planillas de Alta Jerarquía
Para el resumen de haberes y descuentos, se implementan clases de utilidad semánticas:

```html
<table class="striped condensed">
    <thead> ... </thead>
    <tbody>
        <!-- Fila de ejemplo con jerarquía -->
        <tr>
            <td><strong>HABERES</strong></td>
            <td class="text-right text-success">S/ 4,500.00</td>
        </tr>
        <tr>
            <td>DESCUENTOS</td>
            <td class="text-right text-danger">(S/ 450.00)</td>
        </tr>
    </tbody>
    <tfoot>
        <tr class="total-row">
            <th scope="row">NETO A PAGAR</th>
            <td class="text-right"><strong>S/ 4,050.00</strong></td>
        </tr>
    </tfoot>
</table>
```
*CSS Custom:* `.total-row { border-top: 2px solid var(--pico-primary); font-size: 1.1rem; }`

### 2.2 Estados HTMX (Feedback Visual)
Se utiliza el atributo `aria-busy` nativo de Pico CSS en combinación con los eventos de HTMX:

```css
/* Indicador global de carga HTMX */
.htmx-indicator {
    opacity: 0;
    transition: opacity 200ms ease-in;
}
.htmx-request .htmx-indicator {
    opacity: 1;
}

/* Mejora de feedback en botones */
button[aria-busy="true"] {
    pointer-events: none;
    opacity: 0.8;
}
```

### 2.3 Modales `<dialog>` (Mejora de UX)
Ajuste de espaciado y legibilidad para formularios densos:

```css
dialog article {
    max-width: 700px;
    padding: var(--pico-spacing);
    border-radius: 0.75rem;
    box-shadow: var(--pico-card-sectionning-shadow);
}
dialog header {
    border-bottom: 1px solid var(--pico-muted-border-color);
    margin-bottom: var(--pico-spacing);
}
```

---

## 3. DIFERENCIACIÓN ESTRUCTURAL (LAYOUTS)

### 3.1 Panel Admin (SaaS)
- **Identidad**: Uso de gradientes suaves en el sidebar (`from violet-600 to indigo-700`).
- **Navegación**: Menú lateral persistente con iconos de trazo fino (2px weight).
- **Densidad**: Uso de tablas con `font-size: 0.85rem` para gestión de catálogos masivos.

### 3.2 Panel Tenant (Municipal)
- **Identidad**: Cabecera limpia con el logo de la municipalidad destacado.
- **UX Operativa**: Botones de acción principal (Ej: "Procesar Planilla") siempre visibles con `position: sticky` en la parte superior de la vista de detalle.
- **Mobile First**: Sidebar transformable en Drawer mediante Alpine.js, activado por el botón hamburguesa con transición de `transform 0.3s ease-in-out`.

---

## 4. GUÍA DE IMPLEMENTACIÓN GO/HTMX

- **Trigger de Éxito**: Las mutaciones (POST/PUT) deben devolver el header `HX-Trigger: notifySuccess` para disparar brindis (toasts) suaves.
- **Swap OOB**: Utilizar `hx-swap-oob="true"` para actualizar los totales de la cabecera cuando se edita una celda individual en la tabla de planilla.
- **Validación Inline**: Los inputs deben usar `:invalid` de CSS nativo para feedback inmediato sin necesidad de lógica JS compleja.
