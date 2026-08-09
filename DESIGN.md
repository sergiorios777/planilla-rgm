---
name: Planillas RGM
colors:
  surface: '#faf8ff'
  surface-dim: '#d2d9f4'
  surface-bright: '#faf8ff'
  surface-container-lowest: '#ffffff'
  surface-container-low: '#f2f3ff'
  surface-container: '#eaedff'
  surface-container-high: '#e2e7ff'
  surface-container-highest: '#dae2fd'
  on-surface: '#131b2e'
  on-surface-variant: '#3f4850'
  inverse-surface: '#283044'
  inverse-on-surface: '#eef0ff'
  outline: '#707881'
  outline-variant: '#bfc7d2'
  surface-tint: '#006398'
  primary: '#006194'
  on-primary: '#ffffff'
  primary-container: '#007bb9'
  on-primary-container: '#fdfcff'
  inverse-primary: '#93ccff'
  secondary: '#006c4a'
  on-secondary: '#ffffff'
  secondary-container: '#82f5c1'
  on-secondary-container: '#00714e'
  tertiary: '#6a42bd'
  on-tertiary: '#ffffff'
  tertiary-container: '#835dd8'
  on-tertiary-container: '#fffbff'
  error: '#ba1a1a'
  on-error: '#ffffff'
  error-container: '#ffdad6'
  on-error-container: '#93000a'
  primary-fixed: '#cce5ff'
  primary-fixed-dim: '#93ccff'
  on-primary-fixed: '#001d31'
  on-primary-fixed-variant: '#004b73'
  secondary-fixed: '#85f8c4'
  secondary-fixed-dim: '#68dba9'
  on-secondary-fixed: '#002114'
  on-secondary-fixed-variant: '#005137'
  tertiary-fixed: '#eaddff'
  tertiary-fixed-dim: '#d1bcff'
  on-tertiary-fixed: '#24005b'
  on-tertiary-fixed-variant: '#5429a7'
  background: '#faf8ff'
  on-background: '#131b2e'
  surface-variant: '#dae2fd'
typography:
  display-lg:
    fontFamily: Inter
    fontSize: 24.5px
    fontWeight: '700'
    lineHeight: 30px
  headline-md:
    fontFamily: Inter
    fontSize: 18.9px
    fontWeight: '700'
    lineHeight: 24px
  headline-sm:
    fontFamily: Inter
    fontSize: 16.1px
    fontWeight: '600'
    lineHeight: 22px
  body-base:
    fontFamily: Inter
    fontSize: 14px
    fontWeight: '400'
    lineHeight: 20px
  body-bold:
    fontFamily: Inter
    fontSize: 14px
    fontWeight: '600'
    lineHeight: 20px
  label-caps:
    fontFamily: Inter
    fontSize: 11px
    fontWeight: '700'
    lineHeight: 14px
    letterSpacing: 0.05em
  stat-mono:
    fontFamily: JetBrains Mono
    fontSize: 14px
    fontWeight: '600'
    lineHeight: 20px
rounded:
  sm: 0.125rem
  DEFAULT: 0.25rem
  md: 0.375rem
  lg: 0.5rem
  xl: 0.75rem
  full: 9999px
spacing:
  unit: 4px
  xs: 4px
  sm: 8px
  md: 16px
  lg: 24px
  xl: 32px
  gutter: 16px
  sidebar-width: 260px
---

## Brand & Style

The design system is a high-density, institutional SaaS framework designed specifically for the Peruvian public sector. It balances the robustness required for complex payroll management with the lightweight performance of Go and HTMX.

The system employs a **Corporate / Modern** style that prioritizes data integrity and operational speed. It uses a dual-identity approach to distinguish between administrative governance and municipal operations:
- **Governance (Admin):** Evokes authority and stability through Deep Violet and Indigo tones.
- **Operations (Tenant):** Focuses on efficiency and clarity using Ocean Blue and Emerald Green, fostering a productive workspace for HR managers.

The interface is characterized by high information density, crisp hairline borders, and a slate-tinted canvas that reduces eye strain during long periods of data entry. Visual feedback is immediate, utilizing HTMX-driven states to ensure the user always feels in control of the underlying processes.

## Colors

The palette is bifurcated to provide immediate context for the user's current role within the system.

### Panel Tenant (Operational)
- **Primary (Ocean Blue):** `#0284c7` - Main operational actions and primary navigation.
- **Secondary (Emerald Green):** `#059669` - Success states, financial calculations, and positive growth indicators.
- **Canvas (Slate):** `#f8fafc` - A cool-toned background that provides a clean foundation for dense tables.

### Panel Admin (Governance)
- **Tertiary (Deep Violet):** `#5e35b1` - The primary brand color for Super Admin views, signifying system-wide authority.
- **Admin Secondary (Indigo):** `#3f51b5` - Used for sub-navigation and administrative highlights.

### Functional & Status
- **Success:** Background `#d1fae5`, Text `#065f46`, Border `#a7f3d0`.
- **Warning:** Background `#fef3c7`, Text `#92400e`, Border `#fde68a`.
- **Danger:** Background `#fee2e2`, Text `#991b1b`, Border `#fecaca`.
- **Info/Ordinary:** Background `#dbeafe`, Text `#1e40af`, Border `#bfdbfe`.
- **Special/Extraordinary:** Background `#f3e8ff`, Text `#6b21a8`, Border `#e9d5ff`.

## Typography

The typographic system is built on a baseline of **14px** to maximize data visibility in complex spreadsheets and forms.

- **Primary Stack:** **Inter** is used for all UI elements, providing a neutral, systematic, and highly legible experience.
- **Numeric Stack:** **JetBrains Mono** is utilized for all financial figures, currency amounts, and codes to ensure perfect vertical alignment in tables.
- **Hierarchy:** Headers use a bold weight (`700`) with tighter line heights to maintain a professional, news-like authority.
- **Alignment:** Financial data must always be right-aligned using the `stat-mono` style. Labels for metadata or sidebar categories use the `label-caps` style for distinct visual separation.

## Layout & Spacing

The layout utilizes a **Fixed Grid** approach for the sidebar and a **Fluid Grid** for the content area to handle the varying widths of financial reports.

- **Desktop Layout:** A fixed sidebar of `260px` paired with a fluid main container. Gaps between major sections are maintained at `24px` (`lg`).
- **Table Density:** Table cell padding is tightened to `0.4rem 0.75rem` to maximize the number of rows visible above the fold.
- **Mobile Adaption:** At breakpoints below `992px`, the sidebar transitions into an off-canvas drawer. The main container padding reduces to `16px` (`md`).
- **Vertical Rhythm:** A 4px baseline grid ensures consistent alignment between form fields, buttons, and text lines.

## Elevation & Depth

Visual hierarchy is established through **Tonal Layers** and **Low-contrast outlines** rather than heavy shadows, keeping the UI light and fast.

- **Surface Levels:** 
  - **Level 0 (Canvas):** Slate-tinted (`#f8fafc`) for the background.
  - **Level 1 (Cards/Containers):** Pure white (`#ffffff`) with a 1px hairline border in Slate 200 (`#cbd5e1`).
  - **Level 2 (Modals):** Pure white surfaces with a soft ambient shadow (`0 20px 25px -5px rgba(0, 0, 0, 0.1)`) and a `4px` backdrop blur to isolate the task.
- **HTMX Transitions:** Elements being updated via HTMX should use `opacity: 0.6` during the `.htmx-request` state to provide visual feedback of an ongoing process.

## Shapes

The shape language is **Soft** and professional. This ensures the UI feels modern without being overly playful, which is critical for government-oriented software.

- **Standard Elements:** Buttons, inputs, and badges use a `0.25rem` (`rounded-sm`) radius.
- **Containers:** Cards and Modals use a `0.5rem` (`rounded-lg`) radius to provide a slight structural softening.
- **Data Elements:** Table row highlighting and selection states should use sharp corners or very minimal rounding to maintain the "grid" feel of the data.

## Components

### Buttons
- **Primary:** Solid fill using the panel's primary color. Height fixed at `2.25rem` for a compact look.
- **Secondary:** Outline style with a 1px border.
- **HTMX State:** When `aria-busy="true"`, buttons should display a loading spinner and disable pointer events.

### Data Tables
- **Striping:** Use zebra striping with `rgba(241, 245, 249, 0.4)`.
- **Headers:** Background of Slate 100 (`#f1f5f9`), bold caps, with a `2px` bottom border.
- **Totals Row (`tfoot`):** Must be visually distinct with a `#e2e8f0` background and bold monospace text.

### Semantic Badges (`<mark>`)
- High-contrast text on a light tinted background with a matching border.
- Font size must be `11px` (`0.75rem`) to ensure they fit within table rows without increasing row height.

### Modals (`<dialog>`)
- Use native HTML5 `<dialog>` elements.
- Header and Footer should be separated by 1px Slate 200 borders.
- Footer actions must be right-aligned with a `0.5rem` gap.

### Form Fields
- Inputs should have a height of `2.25rem`. 
- Focus states must use a `2px` ring matching the primary color of the specific panel (Admin vs Tenant).