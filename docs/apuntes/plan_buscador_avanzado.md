# Plan de Implementación: Buscador Avanzado en Selects (TomSelect + HTMX)
**Objetivo:** Transformar los desplegables de "Trabajadores" y "Puestos Vacantes" en contratos_ui.html en componentes interactivos con buscador en tiempo real, garantizando que sigan funcionando tras las recargas asíncronas de HTMX.

## Fase 1: Inclusión de Recursos (CDN)
Necesitamos añadir la hoja de estilos y el motor de TomSelect al archivo. Para mantener el módulo auto-contenido, los agregaremos al inicio de contratos_ui.html.

**Instrucciones para el Agente:**

1. Al principio del archivo `ui/templates/tenant/contratos_ui.html` (antes del primer `<article>`), inyectar las siguientes etiquetas CDN:

```HTML
<link href="https://cdn.jsdelivr.net/npm/tom-select@2.2.2/dist/css/tom-select.css" rel="stylesheet">
<script src="https://cdn.jsdelivr.net/npm/tom-select@2.2.2/dist/js/tom-select.complete.min.js"></script>

<style>
    /* Ajuste estético para que se integre perfectamente con Pico CSS */
    .ts-control {
        border-radius: var(--pico-border-radius) !important;
        border: 1px solid var(--pico-border-color) !important;
        background-color: var(--pico-background-color) !important;
        color: var(--pico-color) !important;
        padding: 0.5rem 0.75rem !important;
    }
    .ts-wrapper.disabled .ts-control {
        background-color: var(--pico-form-element-disabled-background-color) !important;
        opacity: var(--pico-form-element-disabled-opacity) !important;
    }
</style>
```

## Fase 2: Identificación de los Selects (Capa HTML)
Debemos marcar los selects que queremos que adquieran el buscador agregando una clase identificadora común: `select-con-buscador`.

**Instrucciones para el Agente:**

1. Buscar el `<select name="trabajador_id">` (dentro del modal o formulario de creación) y añadirle la clase: `class="select-con-buscador"`.

2. Buscar el `<select name="puesto_id">` y añadirle la clase: `class="select-con-buscador"`.

3. (Opcional) Si existen estos mismos selects en el bloque de formulario_editar, añadirles también la clase `class="select-con-buscador"`, si se utilizan.

## Fase 3: Script de Inicialización Reactivo (Capa Javascript)
Este es el componente crucial. El script destruirá instancias antiguas y creará nuevas cada vez que la página cambie o HTMX inyecte código fresco.

**Instrucciones para el Agente:**

1. Al final del archivo `ui/templates/tenant/contratos_ui.html`, agregar el siguiente bloque de código `<script>`:

```HTML
<script>
    // Función maestra para inicializar TomSelect en un contenedor específico
    function inicializarBuscadores(contenedor) {
        contenedor.querySelectorAll('.select-con-buscador').forEach((el) => {
            // Evitamos duplicar la inicialización si el elemento ya tiene TomSelect activo
            if (!el.tomselect) {
                new TomSelect(el, {
                    create: false,
                    maxItems: 1,
                    placeholder: "-- Seleccione una opción --",
                    sortField: { field: "text", direction: "asc" },
                    plugins: ['dropdown_input'] // Asegura una barra de entrada limpia
                });
            }
        });
    }

    // 1. Ejecutar en la carga inicial de la página
    document.addEventListener('DOMContentLoaded', () => {
        inicializarBuscadores(document.body);
    });

    // 2. ¡MAGIA HTMX!: Ejecutar automáticamente cada vez que HTMX inyecte o reemplace HTML
    document.body.addEventListener('htmx:afterSettle', (event) => {
        // Inicializamos únicamente los elementos nuevos que acaban de caer en el target de HTMX
        inicializarBuscadores(event.detail.target);
    });
</script>
```

## **Nota adicional para el desarrollador:**

Sin embargo, como arquitecto, debo darte una advertencia técnica clave para que se la pasemos al agente: HTMX y las librerías de Javascript tradicionales a veces chocan. Si solo inicializamos TomSelect al cargar la página (DOMContentLoaded), en el momento en que HTMX reemplace el formulario dinámicamente (como el flujo del formulario dinámico que diseñamos antes), el nuevo `<select>` vendrá limpio y "perderá" el buscador.