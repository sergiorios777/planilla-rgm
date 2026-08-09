# SISTEMA DE DISEÑO Y ESPECIFICACIÓN DE UI: PLANILLAS RGM

Este documento contiene la auditoría completa y especificación detallada del sistema de diseño actual del proyecto **Planillas RGM** (Stack: **Go + HTMX + Pico CSS + Alpine.js + Tom Select**). Está diseñado para ser importado directamente en herramientas de prototipado como **Google Stitch** o reutilizado para rediseños visuales sin perder ninguna regla de negocio, marcado HTML semántico ni patrón de comportamiento HTMX existente.

---

## 1. ESPECIFICACIÓN DEL STACK Y ARQUITECTURA

### 1.1 Motor de Plantillas Backend y Estructura de Directorios
El backend desarrollado en Go utiliza el paquete nativo `html/template` para renderizar tanto páginas completas como fragmentos HTML dinámicos.

- **Ubicación de Plantillas (`ui/templates/`)**:
  - `layouts/`: Plantillas base y layouts principales.
    - [`index.html`](file:///C:/server/www/planilla-rgm/ui/templates/layouts/index.html): Layout general del **Panel Admin (SaaS)**.
    - [`tenant_index.html`](file:///C:/server/www/planilla-rgm/ui/templates/layouts/tenant_index.html): Layout del **Panel Municipal (Tenant)**.
    - [`iconos_sprite.html`](file:///C:/server/www/planilla-rgm/ui/templates/layouts/iconos_sprite.html): Sprite centralizado de íconos SVG vectoriales (`#icono-*`).
  - `admin/`: 12 plantillas para la gestión SaaS global (`afps_ui.html`, `clasificadores.html`, `conceptos_calculados_ui.html`, `conceptos_modelo_ui.html`, `conceptos_ui.html`, `fuentes_rubros_ui.html`, `inquilinos_ui.html`, `mef_ui.html`, `parametros_ui.html`, `tareas_ui.html`, `tenants.html`, `usuarios_ui.html`).
  - `tenant/`: 19 plantillas para la gestión operativa municipal (`asistencia_ui.html`, `conceptos_tenant_ui.html`, `contratos_ui.html`, `cts_detalle_ui.html`, `cts_ui.html`, `liquidaciones_ui.html`, `metas_ui.html`, `organigramas_ui.html`, `perfil_ui.html`, `planilla_detalle_ui.html`, `planilla_especial_ui.html`, `planilla_rubros_metas_ui.html`, `planillas_anexos_ui.html`, `planillas_ui.html`, `presupuesto_ui.html`, `puestos_conceptos_ui.html`, `puestos_ui.html`, `reportes_ui.html`, `trabajadores_ui.html`).
  - `components/`: Componentes reutilizables trasversales (`notificaciones_ui.html`).
  - `login.html`: Pantalla independiente de inicio de sesión centrado.

### 1.2 Importación de Librerías y CDNs Activas
Las siguientes dependencias se cargan directamente desde CDN en las cabeceras (`<head>`) de los layouts:

```html
<!-- CSS Principal: Pico CSS v2 (CSS Reset + Estilos Semánticos Nativo) -->
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/@picocss/pico@2/css/pico.min.css">

<!-- Interacciones SPA Dinámicas Server-Driven: HTMX v1.9.10 -->
<script src="https://unpkg.com/htmx.org@1.9.10"></script>

<!-- Estado Dinámico del Lado del Cliente: Alpine.js v3.x -->
<script defer src="https://cdn.jsdelivr.net/npm/alpinejs@3.x.x/dist/cdn.min.js"></script>

<!-- Buscadores Desplegables Avanzados: Tom Select v2.2.2 -->
<link href="https://cdn.jsdelivr.net/npm/tom-select@2.2.2/dist/css/tom-select.css" rel="stylesheet">
<script src="https://cdn.jsdelivr.net/npm/tom-select@2.2.2/dist/js/tom-select.complete.min.js"></script>
```

### 1.3 Estado Actual de los Archivos de Assets Estáticos
- **`ui/static/css/`**: El directorio existe en el sistema de archivos pero **actualmente se encuentra vacío**. Todas las reglas CSS personalizadas y ajustes de componentes se encuentran embebidos directamente en bloques `<style>` dentro de `layouts/index.html` y `layouts/tenant_index.html`, o aplicados como estilos inline `style="..."` dentro de los templates HTML.
- **`ui/static/js/`**: El directorio está **actualmente vacío**. Las funciones JavaScript de soporte (como `toggleMobileMenu()` o escuchadores de eventos HTMX `hx-on::after-request`, `hx-on::after-settle`) están embebidas al final de los archivos HTML o layouts.
- **`ui/static/img/` / `ui/static/uploads/`**: Reservados para recursos gráficos y archivos cargados.

---

## 2. OVERRIDES DE PICO CSS Y VARIABLES ACTUALES

### 2.1 CSS Custom Properties (`--pico-*`) Sobreescritas

El proyecto aplica adaptaciones sobre el sistema de variables de Pico CSS v2 para ajustar selectores dinámicos, menús laterales responsivos y densidad de datos:

```css
/* 1. Reducción global del tamaño base tipográfico para aumentar la densidad informativa */
html {
    font-size: 14px; /* Pico CSS por defecto usa 16px */
}

/* 2. Adaptación visual de TomSelect para integración perfecta con Pico CSS */
.ts-control {
    border-radius: var(--pico-border-radius) !important;
    border: var(--pico-border-width) solid var(--pico-form-element-border-color) !important;
    background-color: var(--pico-form-element-background-color) !important;
    background-image: none !important;
    color: var(--pico-color) !important;
    display: flex !important;
    align-items: center !important;
    height: calc(1.5em + var(--pico-form-element-spacing-vertical) * 2 + var(--pico-border-width) * 2) !important;
    padding: 0 var(--pico-form-element-spacing-horizontal) !important;
    position: relative !important;
    box-shadow: none !important;
}

.ts-wrapper.focus .ts-control {
    border-color: var(--pico-form-element-active-border-color) !important;
    box-shadow: 0 0 0 var(--pico-outline-width) var(--pico-form-element-focus-color) !important;
}

.ts-wrapper.disabled .ts-control {
    background-color: var(--pico-form-element-disabled-background-color) !important;
    opacity: var(--pico-form-element-disabled-opacity) !important;
}

.ts-dropdown {
    background-color: var(--pico-form-element-background-color) !important;
    color: var(--pico-color) !important;
    border: var(--pico-border-width) solid var(--pico-form-element-border-color) !important;
}

.ts-dropdown .active {
    background-color: var(--pico-primary-hover-background) !important;
    color: var(--pico-primary-inverse) !important;
}
```

### 2.2 Paleta de Colores Detectada

#### Colores Base y Neutros:
- **Primary Accordance**: Azul / Índigo por defecto de Pico CSS (`var(--pico-primary)`).
- **Backgrounds**: Modo Claro / Oscuro nativo gestionado por Pico CSS (`var(--pico-background-color)`).
- **Bordes Neutros**: `#e0e0e0` / `var(--pico-muted-border-color)`.
- **Textos Secundarios / Deshabilitados**: `#888888` / `var(--pico-muted-color)`.

#### Estados Semánticos de Planilla, Badges y Alertas:
| Estado / Uso | Color de Fondo | Color de Texto | Código Hex / Variable |
| :--- | :--- | :--- | :--- |
| **Borrador / Pendiente** | `#fff9c4` | `#f57f17` | Fondo Amarillo Ámbar Suave |
| **Cerrada / Activo / Ingreso Exitoso** | `#c8e6c9` | `#1b5e20` | Verde Esmeralda Suave |
| **Inactivo / Error / Retención / Eliminar** | `#ffcdd2` | `#b71c1c` / `#d32f2f` | Rojo Carmesí / Rubí |
| **Ordinaria / Aporte Patronal** | `#e3f2fd` / `#bbdefb` | `#1565c0` | Azul Cobalto |
| **Extraordinaria / Especial** | `#f3e5f5` | `#6a1b9a` | Morado Púrpura |
| **SaaS Admin (Super Usuario)** | `#e1bee7` | `#4a148c` | Violeta Púrpura |
| **Advertencia / Alerta Crítica** | `#fff3cd` | `#856404` / `#e65100` | Amarillo Mostaza / Naranja Alerta |

### 2.3 Tipografía, Espaciados y Radios de Borde
- **Fuente**: Font Stack del sistema gestionado por Pico CSS (`system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif`).
- **Tamaño Base**: `14px` aplicado en el elemento `html`.
- **Radios de Borde (`border-radius`)**:
  - General: `var(--pico-border-radius)` (aprox `0.25rem` / 4px).
  - Badges e Indicadores: `4px` (estándar), `6px` / `50%` (círculos de contador de notificaciones).
- **Espaciados Internos (Padding)**:
  - Botones de acción en tablas: `padding: 0.2rem 0.5rem` / `font-size: 0.8rem`.
  - Botones de acción principales: `padding: 0.25rem 0.75rem` / `font-size: 0.85rem`.

---

## 3. CATÁLOGO DE COMPONENTES Y FRAGMENTOS HTML

### 3.1 Navbars (Barra de Navegación Superior)

#### Navbar del Layout Admin (`ui/templates/layouts/index.html`)
```html
<nav class="container-fluid" style="border-bottom: 1px solid #e0e0e0; margin-bottom: 2rem;">
    <ul>
        <li><strong>💼 Planillas RGM</strong></li>
    </ul>
    <ul style="display: flex; align-items: center; gap: 1rem;">
        <li id="campana-notificaciones-contenedor" 
            hx-get="/notificaciones/campana" 
            hx-trigger="every 30s, load" 
            hx-swap="innerHTML"
            style="margin-bottom: 0;">
        </li>
        <li style="margin-bottom: 0;">Panel Súper Admin</li>
        <li style="margin-bottom: 0;">
            <button class="outline secondary" style="padding: 0.25rem 0.75rem; font-size: 0.85rem; margin-bottom: 0;" hx-post="/logout">
                🚪 Salir
            </button>
        </li>
    </ul>
</nav>
```

#### Navbar del Layout Tenant Municipal con Hamburguesa Responsive (`ui/templates/layouts/tenant_index.html`)
```html
<nav class="container-fluid" style="border-bottom: 1px solid #e0e0e0; margin-bottom: 2rem; padding: 1rem;">
    <ul>
        <li style="margin-right: 0.5rem; display: none;" id="btn-toggle-menu">
            <button class="outline secondary" style="padding: 0.25rem 0.5rem; margin-bottom: 0;" aria-label="Abrir menú" onclick="toggleMobileMenu()">
                ☰
            </button>
        </li>
        <li>
            <strong>
                <svg class="icono-menu"><use href="#icono-edificio-2"></use></svg> 
                {{if .TenantNombre}}{{.TenantNombre}}{{else}}Mi Municipalidad{{end}}
            </strong>
        </li>
    </ul>
    <ul style="display: flex; align-items: center; gap: 1rem;">
        <li id="campana-notificaciones-contenedor" 
            hx-get="/notificaciones/campana" 
            hx-trigger="every 30s, load" 
            hx-swap="innerHTML"
            style="margin-bottom: 0;">
        </li>
        <li style="margin-bottom: 0;">
            <small>{{if .UsuarioNombre}}👤 {{.UsuarioNombre}} ({{.UsuarioRol}}){{else}}Operador de Planillas{{end}}</small>
        </li>
        <li style="margin-bottom: 0;">
            <button class="outline secondary" style="padding: 0.25rem 0.75rem; font-size: 0.85rem; margin-bottom: 0;" hx-post="/logout">
                🚪 Salir
            </button>
        </li>
    </ul>
</nav>
```

### 3.2 Menú Lateral / Drawer Responsive (`aside`)
```html
<aside class="sidebar">
    <nav>
        <ul>
            <li><small><strong>RECURSOS HUMANOS</strong></small></li>
            <li>
                <a href="#" hx-get="/tenant/ui/trabajadores" hx-target="#contenido-tenant">
                    <svg class="icono-menu"><use href="#icono-usuarios"></use></svg>
                    Trabajadores (Legajo)
                </a>
            </li>
            <li>
                <a href="#" hx-get="/tenant/ui/contratos" hx-target="#contenido-tenant">
                    <svg class="icono-menu"><use href="#icono-carpeta-usuario"></use></svg>
                    Contratos
                </a>
            </li>
        </ul>
    </nav>
</aside>
```

### 3.3 Tablas de Datos (`<table class="striped">`)
```html
<div class="overflow-auto">
    <table class="striped">
        <thead>
            <tr>
                <th>Periodo</th>
                <th>Descripción</th>
                <th>Estado</th>
                <th>Acción</th>
            </tr>
        </thead>
        <tbody>
            <tr>
                <td>
                    <strong>2026 - 01</strong><br>
                    <mark style="background-color: #e3f2fd; color: #1565c0; padding: 2px 6px; border-radius: 4px; font-size: 0.7rem; font-weight: bold; border: 1px solid #bbdefb;">
                        📅 ORDINARIA
                    </mark>
                </td>
                <td><small>Planilla Régimen CAS - Enero</small></td>
                <td>
                    <mark style="background-color: #fff9c4; color: #f57f17; padding: 2px 6px; border-radius: 4px; font-size: 0.8rem;">
                        Borrador
                    </mark>
                </td>
                <td>
                    <button class="outline secondary" style="padding: 0.2rem 0.5rem; font-size: 0.8rem;" hx-get="/tenant/planillas/detalle/ui?id=1" hx-target="#contenido-tenant">
                        <svg class="icono-menu"><use href="#icono-eye"></use></svg> Ver Detalles
                    </button>
                </td>
            </tr>
        </tbody>
    </table>
</div>
```

### 3.4 Modales Nativo HTML5 (`<dialog>`)

#### Modal Estático Pre-renderizado (`ui/templates/admin/inquilinos_ui.html`)
```html
<dialog id="modal-crear-inquilino">
    <article style="max-width: 600px; width: 100%;">
        <header>
            <button aria-label="Close" rel="prev" onclick="document.getElementById('modal-crear-inquilino').removeAttribute('open')"></button>
            <h3 style="margin: 0;">➕ Registrar Nueva Entidad</h3>
        </header>
        <form hx-post="/admin/inquilinos/crear" hx-target="#lista-inquilinos" hx-swap="innerHTML" 
              hx-on::after-request="if(event.detail.successful) { this.reset(); document.getElementById('modal-crear-inquilino').removeAttribute('open'); }">
            <label for="nombre">Nombre de la Municipalidad</label>
            <input type="text" id="nombre" name="nombre" placeholder="Ej: Municipalidad Provincial de..." required>

            <label for="ruc">Número de RUC</label>
            <input type="text" id="ruc" name="ruc" maxlength="11" placeholder="Ej: 20123456789" required>

            <label style="margin-bottom: 1.5rem;">
                <input type="checkbox" name="activo" checked> Entidad Activa
            </label>
            
            <footer style="display: flex; justify-content: flex-end; gap: 0.5rem;">
                <button type="button" class="secondary outline" onclick="document.getElementById('modal-crear-inquilino').removeAttribute('open')" style="margin-bottom: 0; width: auto;">Cancelar</button>
                <button type="submit" style="margin-bottom: 0; width: auto;">Guardar Entidad</button>
            </footer>
        </form>
    </article>
</dialog>
```

#### Modal de Carga Dinámica vía HTMX (`ui/templates/tenant/planilla_detalle_ui.html`)
```html
<dialog id="modal-boleta-individual" hx-on::after-settle="if(event.detail.target.id === 'contenido-modal-boleta') this.showModal()">
    <article style="max-width: 750px; width: 95%;">
        <header>
            <a href="#close" aria-label="Close" class="close" onclick="document.getElementById('modal-boleta-individual').close()"></a>
            <h4 style="margin-bottom: 0;">🧾 Detalle de Boleta de Pago</h4>
        </header>
        <div id="contenido-modal-boleta">
            <!-- Carga dinámica devuelta por Go: {{define "modal_boleta_content"}} -->
        </div>
    </article>
</dialog>
```

### 3.5 Badges de Estado y Categorización (`<mark>`)
- **Activo**: `<mark style="background-color: #c8e6c9; color: #1b5e20; padding: 2px 6px; border-radius: 4px; font-size: 0.85rem;">Activo</mark>`
- **Inactivo**: `<mark style="background-color: #ffcdd2; color: #b71c1c; padding: 2px 6px; border-radius: 4px; font-size: 0.85rem;">Inactivo</mark>`
- **Borrador**: `<mark style="background-color: #fff9c4; color: #f57f17; padding: 2px 6px; border-radius: 4px; font-size: 0.8rem;">Borrador</mark>`
- **Planilla Extraordinaria**: `<mark style="background-color: #f3e5f5; color: #6a1b9a; padding: 2px 6px; border-radius: 4px; font-size: 0.7rem; font-weight: bold; border: 1px solid #e1bee7;">⭐ EXTRAORDINARIA</mark>`
- **Concepto Ingreso**: `<mark style="background-color: #c8e6c9; color: #1b5e20; padding: 2px 6px; border-radius: 4px; font-size: 0.7rem; font-weight: bold;">INGRESO</mark>`
- **Concepto Descuento**: `<mark style="background-color: #ffcdd2; color: #b71c1c; padding: 2px 6px; border-radius: 4px; font-size: 0.7rem; font-weight: bold;">DESCUENTO</mark>`
- **Concepto Aporte**: `<mark style="background-color: #bbdefb; color: #1565c0; padding: 2px 6px; border-radius: 4px; font-size: 0.7rem; font-weight: bold;">APORTE</mark>`

### 3.6 Formularios, Buscadores e Inputs Especiales
- **Barra de Búsqueda Reactiva con Debounce**:
  ```html
  <input class="form-control" type="search" name="buscar" placeholder="Buscar por nombre o DNI..."
         hx-get="/tenant/trabajadores/lista" hx-trigger="keyup changed delay:500ms, search"
         hx-target="#lista-trabajadores">
  ```
- **Switch Checkbox de Pico CSS**:
  ```html
  <input type="checkbox" name="es_extraordinaria" role="switch" value="true">
  ```

---

## 4. DICCIONARIO DE PATRONES HTMX

### 4.1 Catálogo de Triggers HTMX
- `load`: Ejecuta una petición al renderizarse el elemento (ej: `hx-trigger="load"` para la carga diferida de tablas).
- `keyup changed delay:500ms, search`: Captura el tipeo en búsquedas, aplicando 500ms de debounce o disparándose al presionar Enter/limpiar el campo `search`.
- `every 30s, load`: Ejecuta polling periódico cada 30 segundos (usado en la campana de notificaciones).
- `intersect, revealed`: Carga perezosa de datos al hacer scroll hasta el contenedor (modal de notificaciones).
- `tasasActualizadas from:body`, `recargarTablaContratos from:body`, `reloadArbol from:body`, `recargarTablaModelos from:body`: Eventos personalizados emitidos desde las cabeceras de respuesta HTTP de Go (`HX-Trigger: tasasActualizadas`) para actualizar tablas en segundo plano tras una mutación.

### 4.2 Atributos de Navegación, Target y Swap
- **`hx-target`**:
  - `#contenido-principal` / `#contenido-tenant`: Navegación SPA que reemplaza el cuerpo central de la aplicación.
  - `#lista-*`: Reemplaza únicamente la tabla o grid de datos.
  - `#contenedor-modal-editar`: Inyecta el modal devuelto por Go.
  - `this`: Reemplaza el elemento actual (ej: celda de edición inline de montos).
- **`hx-swap`**:
  - `innerHTML` (defecto): Reemplaza el contenido interior del contenedor.
  - `none`: Procesa los headers de respuesta (ej. redirigir o cerrar modal) sin alterar el DOM.
  - `hx-swap-oob="true"`: Permite actualizar alertas o contadores globales en cualquier parte del DOM en la misma respuesta HTTP.
- **`hx-indicator`**:
  - `hx-indicator="this"` o `.htmx-indicator`: Activa spinners o estados `aria-busy="true"` durante la transferencia HTTP.

### 4.3 Fragmentos HTML Devueltos por el Servidor Go (`define`)

| Módulo | Nombre del Bloque (`define`) | Descripción del Fragmento Devuelto |
| :--- | :--- | :--- |
| **Admin Inquilinos** | `tabla_inquilinos`, `filas_inquilinos`, `formulario_editar` | Renderiza la tabla completa, filas filtradas o modal de edición de entidades. |
| **Admin Usuarios** | `tabla_usuarios`, `filas_usuarios`, `formulario_editar` | Tabla y formulario dinámico de operadores. |
| **Admin AFPs** | `tabla_afps`, `filas_afps`, `tabla_tasas`, `formulario_editar_afp` | Pestañas de comisiones SBS y administradoras. |
| **Admin Tareas** | `tabla_tareas`, `filas_tareas`, `formulario_editar_tarea` | Monitoreo y ejecución manual de tareas programadas. |
| **Admin Conceptos** | `tabla_modelos`, `modal_reglas_modelo_content`, `tabla_calculados`, `seccion_afectaciones` | Catálogo maestro de conceptos SUNAT y fórmulas calculadas. |
| **Tenant Planillas** | `tabla_planillas`, `formulario_crear`, `modal_boleta_content`, `modal_plame_content` | Listado de periodos, aperturas y visualizador dinámico de boletas individuales. |
| **Tenant Personal** | `tabla_trabajadores`, `formulario_crear`, `formulario_editar`, `formulario_importar_trabajadores` | Legajo de personal e importador Excel. |
| **Tenant Contratos** | `tabla_contratos`, `formulario_crear`, `formulario_editar` | Asignación de puestos y contratos laborales. |
| **Tenant Puestos** | `tabla_puestos`, `formulario_crear`, `formulario_editar`, `tabla_asignados` | CAP (Cuadro de Puestos) y estructura de costos de la plaza. |
| **Tenant Organigrama**| `arbol_parcial`, `modal_unidad_crear`, `modal_unidad_editar` | Árbol jerárquico de unidades orgánicas. |
| **Tenant Reportes** | `lista_reportes` | Catálogo de informes descargables en PDF/Excel con filtros dinámicos. |
| **Componentes** | `campana`, `lista_notificaciones`, `svg_sprite` | Campana con badge en tiempo real, modal de notificaciones y sprite SVG. |

---

## 5. DIFERENCIACIÓN UX: PANEL ADMIN VS. PANEL TENANT MUNICIPAL

### 5.1 Panel Admin (SaaS Super Admin)
- **Propósito**: Administración global del software multitenant (gestión de municipalidades inscritas, asignación de usuarios super_admin, catálogo maestro de conceptos SUNAT, parámetros globales, clasificadores MEF y comisiones mensuales de AFPs).
- **Layout y Estructura**:
  - Grid de 2 columnas estáticas: `260px` (Sidebar fijo) + `1fr` (Área principal `#contenido-principal`).
  - Navegación lateral dividida en dos secciones: `ADMINISTRACIÓN` y `CATÁLOGOS GLOBALES`.
- **Diseño & Densidad**:
  - Visualización utilitaria de alta densidad para tablas masivas de solo lectura o parametrización.
  - Colores distintivos: Identificador `Panel Súper Admin` en el Header, badge de usuario SaaS Admin en color violeta (`#e1bee7` / `#4a148c`).

### 5.2 Panel Tenant Municipal (Operador de Municipalidad)
- **Propósito**: Gestión operativa del día a día de una municipalidad específica (procesamiento de planillas mensuales y especiales, legajos de trabajadores, gestión de contratos, registro de asistencia/tardanzas, cálculo de CTS, liquidaciones, formulación de puestos CAP y reportes institucionales).
- **Layout y Estructura**:
  - Layout Grid responsivo (`250px 1fr`).
  - **Transformación a Drawer Lateral en Móviles (`< 992px`)**: El sidebar se convierte en un panel flotante deslizable (`left: -280px` a `left: 0`) acompañado de un backdrop translúcido con desenfoque (`backdrop-filter: blur(2px)`).
  - Botón de Menú Hamburguesa (`☰`) visible únicamente en pantallas medianas y pequeñas.
- **Secciones del Menú Lateral**:
  1. `MI ENTIDAD`: Perfil Institucional, Estructura Orgánica.
  2. `RECURSOS HUMANOS`: Trabajadores (Legajo), Contratos, Asistencia/Tardanzas.
  3. `PLANILLAS`: Cálculo de Planillas, CTS Semestral, Liquidaciones, Presupuesto, Reportes.
  4. `METAS, PLAZAS Y CONCEPTOS`: Metas Presupuestales, Cuadro de Puestos (CAP), Catálogo de Conceptos.
- **Diseño & Densidad**:
  - Interfaz orientada al operador de nómina, con botones de acción rápida por estado de planilla (`Procesar`, `Ver Boleta`, `Rubros/Metas`, `Anexos/PLAME`, `Cerrar Planilla`).
  - Badges semánticos para diferenciar visualmente regímenes laborales y planillas Ordinarias vs. Extraordinarias.
