## Pregunta:

El punto 1 de tu respuesta anterior, me parece más relevante para su depuración, antes de analizar vista por vista.

Nuestra aplicación pretende ser muy ligera en elementos de su diseño, por eso está basada en el framework pico.css que prioriza la sintaxis semántica. Esta consideración debe primar sobre todas las ideas de diseño.

Nuestro archivo de estilos es: `@ui/static/css/custom.css` y debe ser evaluado para ajustarlo a la filosofia de Pico CSS v2 y a nuestro manifiesto de diseño.

Debemos evaluar lo siguiente (punto 1 de la respuesta anterior):

> "Lo que se podría refactorizar por sobreescribir Pico CSS innecesariamente:
> Utilidades "Utility-First" Atómicas: Tienes decenas de clases estilo Tailwind (.mb-sm, .mt-md, .p-0, .w-full, .font-bold, .d-block). > Pico CSS promueve el uso de HTML semántico nativo y su grilla integrada (<grid>, <article>, <main>), por lo que depender en exceso de estas micro-clases atómicas en cada etiqueta recarga el archivo CSS.  
> Uso repetido de !important: Hay más de 60 declaraciones !important en reglas como .btn-sm, .ts-control, .badge. Esto significa que tu CSS está peleando continuamente contra la especificidad por defecto de Pico CSS. "

Las **variables de pico.css 2** estan en:

* `@docs\temporal\styles_variables_pico.css`
* `@docs\temporal\color_variables_pico.css`

El manifiesto de diseño en: `@DESIGN.md`
El archivo complementario para el manifiesto de diseño es: `@design_v2.md`

¿Cómo podemos evaluar nuestro `@ui/static/css/custom.css` para mejorarlo?

---

## Respuesta (Gemini Flash 3.6 Extendido):

Es una excelente decisión estratégica depurar primero el archivo `@ui/static/css/custom.css`. Intentar auditar 35 vistas HTML contra un archivo de estilos que contiene redundancias y peleas de especificidad (`!important`) habría generado confusión en los agentes y soluciones parches.

Al revisar detalladamente el archivo `@ui/static/css/custom.css` en conjunto con las variables de **Pico CSS v2** y el manifiesto **`@DESIGN.md`**, el diagnóstico es muy claro: el archivo creció tanto porque está intentando forzar estilos mediante CSS utilitario en lugar de configurar el motor de Pico CSS.

---

## Diagnóstico Técnico de `@ui/static/css/custom.css`

### 1. La Causa de los +60 `!important`

Pico CSS v2 aplica estilos semánticos con selectores de especificidad moderada. En tu `@ui/static/css/custom.css`, encuentras líneas como:

```css
input, select, textarea, button {
    border-radius: 0.25rem !important; /*[cite: 2] */
}

```

Esto ocurre porque se está intentando sobreescribir el elemento HTML directamente, en lugar de redefinir la variable nativa de Pico CSS en la raíz:

```css
:root {
    --pico-border-radius: 0.25rem; /* Solución limpia nativa de Pico[cite: 6] */
}

```

### 2. "Invasión" de Clases Utilitarias (Sección 13)

Toda la Sección 13 de tu archivo `custom.css` (que ocupa casi la mitad del archivo) está llena de clases estilo Tailwind:

* Margenes/Paddings: `.m-0`, `.mb-sm`, `.mb-md`, `.p-0`, `.pt-md`.


* Anchos/Altos: `.w-full`, `.w-50`, `.max-w-400`, `.h-full`.


* Tipografía y Display: `.font-bold`, `.d-block`, `.text-uppercase`, `.flex-center`.



Si tu proyecto promueve la **sintaxis semántica sobre Pico CSS**, depender de estas clases atómicas rompe la filosofía. Pico CSS resuelve esto de forma nativa con etiquetas HTML (`<header>`, `<article>`, `<grid>`, `<main>`, `<mark>`) y la variable `--pico-spacing`.

### 3. Duplicación de Variables Existentes

En la Sección 8 y 9 de `custom.css`, se definen alturas y bordes para inputs y componentes de formulario. Sin embargo, Pico CSS v2 ya ofrece variables explícitas para esto:

* `--pico-form-element-spacing-vertical` y `--pico-form-element-spacing-horizontal`.


* `--pico-form-element-border-color` y `--pico-form-element-background-color`.



---

## Plan de Depuración para `custom.css`

Para reducir `custom.css` de ~1000 líneas a unas **250-300 líneas limpias y mantenibles**, el trabajo consiste en:

1. **Trasladar valores globales a `:root` / `[data-theme]`:** Configurar los border-radius, font-sizes y paddings globales a través de las variables `--pico-*`.


2. **Eliminar casi todos los `!important`:** Al usar las CSS Custom Properties de Pico, las reglas sobreescritas heredan de forma natural sin forzar la cascada.


3. **Consolidar Componentes Complejos:** Mantener únicamente las reglas estructurales necesarias que Pico CSS no cubre por sí solo (ej: el Sidebar Admin con gradiente, el Drawer responsive del Tenant, los HTMX indicators y la adaptación de TomSelect).


4. **Purgar Clases Utilitarias Redundantes:** Agrupar los comportamientos frecuentes en clases semánticas o eliminarlos para que Pico controle el layout.



---

## Prompt Estructurado para Antigravity CLI

Puedes ejecutar el siguiente prompt en tu terminal para que el agente refactorice y depure `custom.css` directamente basándose en `DESIGN.md` y las variables de Pico CSS:

```text
[OBJETIVO DE AUDITORÍA Y REFACTORIZACIÓN CSS]
Refactoriza y depura el archivo `@ui/static/css/custom.css` para alinearlo estrictamente a la filosofía de Pico CSS v2 (orientado a HTML semántico y variables CSS nativas) y al manifiesto de diseño `@DESIGN.md`. 

[REGLAS Y PRIORIDADES DE DISEÑO]
1. PRIORIDAD SEMÁNTICA: Pico CSS debe gobernar la tipografía, formularios, tablas y espaciados mediante variables nativas. Se debe minimizar el uso de clases utilitarias estilo "utility-first".
2. ELIMINACIÓN DE !IMPORTANT: Reemplaza las sobreescrituras que usan `!important` por redefinición de variables nativas de Pico CSS (`--pico-*`) en `:root`, `body.admin-theme` y `body.tenant-theme`.
3. CONSERVACIÓN DE COMPONENTES CLAVE: No elimines la lógica funcional de:
   - Temas Admin (SaaS Governance) y Tenant (Operaciones Municipales) con sus variables `--pico-primary`.
   - Modulo Responsive Drawer para el Aside en Mobile/Tenant.
   - Transiciones y estados HTMX (`.htmx-indicator`, `aria-busy`).
   - Adaptación visual de TomSelect.

[PASOS DE EJECUCIÓN]

Paso 1: Mapeo a Variables de Pico CSS v2
- Configura en el bloque inicial de `custom.css` o en `:root` las variables globales de Pico CSS:
  * `--pico-font-family` (Inter)
  * `--pico-font-size` (14px)
  * `--pico-border-radius` (0.25rem / 4px)
  * Espaciados de formularios (`--pico-form-element-spacing-*`)
- Elimina declaraciones tipo `input, select { border-radius: 0.25rem !important; }` y usa las variables nativas de Pico.

Paso 2: Limpieza de Badges y Tablas
- Refactoriza `.badge` y `<mark>` para usar variables o herencia limpia.
- Simplifica las reglas de tablas (.condensed, .striped, .total-row) eliminando `!important` en rellenos y bordes.

Paso 3: Depuración Masiva de la Sección 13 (Clases Utilitarias)
- Revisa las clases de la Sección 13 (`.mb-sm`, `.p-0`, `.w-full`, `.font-bold`, `.d-block`, `.flex-row-between`, etc.).
- Conserva únicamente aquellas clases compuestas que representen layouts de componentes reales (ej: `.sticky-toolbar`, `.layout-admin`, `.card-section`).
- Elimina las micro-clases atómicas redundantes que sobreescriben margenes/padding/display individuales de un solo atributo con `!important`.

[OUTPUT REQUERIDO]
Reescribe `ui/static/css/custom.css` aplicando estas optimizaciones. Agrega comentarios estructurados por sección. Al finalizar, genera un resumen de cuántas líneas y declaraciones `!important` fueron eliminadas.

```