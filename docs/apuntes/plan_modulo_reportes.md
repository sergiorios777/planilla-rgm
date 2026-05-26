# Plan de Módulo de Reportes en Tenants

Diseñar un **Módulo de Reportes modular**, limpio y rápido es una excelente adición. En el sector público, los jefes de recursos humanos y los administradores pasan el 40% de su tiempo extrayendo información para enviarla a la OCI (Órgano de Control Institucional), al MEF o para auditorías internas.

Para que este módulo sea altamente escalable (es decir, que en el futuro puedas agregar 50 reportes nuevos sin reescribir código), utilizaremos un **patrón de diseño basado en Catálogo en Memoria en Go**, combinado con la reactividad ultra-ligera de HTMX y la limpieza estética de Pico CSS.

Aquí tienes el plan de diseño y la estructura detallada lista para que la revisemos y se la entreguemos a tu agente en la terminal.

## Análisis del Catálogo de Reportes Iniciales
Para empezar con fuerza, implementaremos los siguientes reportes estratégicos (2 por módulo):

1. **Trabajadores**:

   * *Padrón General de Personal*: Datos completos, DNI, dirección y estado activo.

   * *Cumpleaños del Mes*: Lista cronológica del personal que cumple años en el mes actual (clave para bienestar social).

2. **Organigrama**:

   * *Directorio Estructurado de Dependencias*: Lista jerárquica de oficinas con sus códigos MEF correspondientes.

3. **Puestos (Plazas)**:

   * *Resumen de Plazas (Ocupadas vs. Vacantes)*: Estado actual del CAP/PAP de la municipalidad.

   * *Presupuesto Analítico de Personal (PAP) Resumido*: Costo mensual presupuestado por cada plaza.

4. **Conceptos del Tenant**:

   * *Maestro Local de Conceptos*: Catálogo de ingresos, retenciones y aportes con sus respectivos clasificadores MEF.

5. **Contratos**:

   * *Alertas de Vencimiento*: Contratos a plazo fijo o transitorios CAS que vencen en los próximos 30 o 60 días.

## Plan de Implementación: Módulo de Reportes Generales

### Fase 1: El Catálogo y Estructura en el Backend (Go)
En lugar de crear una tabla en la base de datos para listar los reportes (lo que obligaría a hacer inserts por cada reporte nuevo), definiremos el catálogo mediante estructuras en memoria de Go dentro de un nuevo archivo de configuración.

**Instrucción para el Agente:**

1. Crear `internal/models/reporte.go`:

```Go
package models

type Reporte struct {
	ID          string `json:"id"`          // Ej: "trab_padron"
	Modulo      string `json:"modulo"`      // Ej: "TRABAJADORES", "PUESTOS"
	Nombre      string `json:"nombre"`      // Ej: "Padrón General de Personal"
	Descripcion string `json:"descripcion"` // Ej: "Lista detallada de todo el personal activo..."
}
```

2. Crear `internal/config/catalogo_reportes.go` con la lista estática global:

```Go
package config

import "planilla-rgm/internal/models"

var ListaReportes = []models.Reporte{
	{ID: "trab_padron", Modulo: "TRABAJADORES", Nombre: "👥 Padrón General de Personal", Descripcion: "Listado completo de trabajadores activos con datos de contacto y legajo."},
	{ID: "trab_cumple", Modulo: "TRABAJADORES", Nombre: "🎂 Cumpleaños del Mes", Descripcion: "Personal que celebra su onomástico en el mes en curso para bienestar social."},
	{ID: "org_directorio", Modulo: "ORGANIGRAMA", Nombre: "🏢 Directorio de Dependencias", Descripcion: "Estructura orgánica completa basada en la ordenanza municipal vigente y códigos MEF."},
	{ID: "puesto_resumen", Modulo: "PUESTOS", Nombre: "📊 Ocupabilidad de Plazas (CAP/PAP)", Descripcion: "Cuadro estadístico de puestos en estado VACANTE u OCUPADO."},
	{ID: "puesto_pap", Modulo: "PUESTOS", Nombre: "💰 Presupuesto Analítico (PAP)", Descripcion: "Resumen de costos mensuales presupuestados asignados por plaza."},
	{ID: "concepto_maestro", Modulo: "CONCEPTOS", Nombre: "⚙️ Catálogo Local de Conceptos", Descripcion: "Lista de ingresos y aportes configurados con sus afectaciones de ley y clasificadores."},
	{ID: "contrato_vence", Modulo: "CONTRATOS", Nombre: "⏳ Alertas de Vencimiento", Descripcion: "Contratos de personal a plazo fijo o transitorios CAS próximos a culminar en los siguientes 30 días."},
}
```

### Fase 2: El Handler y las Rutas (`reporte_handler.go`)
El Handler se encargará de dos cosas: filtrar la lista visual usando HTMX y despachar la generación de archivos (PDF/Excel) según el ID del reporte.

**Instrucción para el Agente:**

1. Crear `internal/handlers/reporte_handler.go`:

   * **Método `VistaUI`**: Renderiza la página principal de reportes cargando inicialmente todo el catálogo de `config.ListaReportes`.

   * **Método `FiltrarUI` (HTMX)**: Recibe por parámetro GET `modulo`. Filtra el slice `config.ListaReportes` devolviendo únicamente las tarjetas (*cards*) que coincidan con el módulo seleccionado.

   * **Método `ExportarPDF`**: Recibe `id` del reporte. Ejecuta un `switch id` que llama al servicio correspondiente (ej: `h.TrabajadorRepo.ObtenerTodos`), genera el PDF usando tu servicio actual y escribe los bytes directamente en el navegador con las cabeceras `Content-Type: application/pdf` e Inline para que se abra nativamente en pantalla.

   * **Método `ExportarExcel`**: Recibe `id`. Usa `excelize V2` para armar el libro, rellena las celdas dinámicamente y fuerza la descarga con `Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`.

### Fase 3: La Interfaz de Usuario Visual (`reportes_ui.html`)
Diseñaremos un panel moderno utilizando el sistema de rejillas (`Grid`) de Pico CSS. A la izquierda colocaremos un menú vertical de navegación limpia para filtrar los módulos y a la derecha la lista dinámica de reportes disponibles gobernada por HTMX.

**Instrucción para el Agente:**

1. Crear el archivo `ui/templates/tenant/reportes_ui.html`:

```HTML
<article>
    <header>
        <h3 style="margin: 0;">📊 Centro de Informes y Reportes Institucionales</h3>
        <small>Extrae, visualiza y exporta la información oficial de la municipalidad.</small>
    </header>

    <div class="grid" style="grid-template-columns: 250px 1fr; gap: 2rem;">
        
        <aside>
            <nav>
                <h5 style="margin-top: 0; padding-left: 0.5rem; color: var(--pico-muted-color);">Módulos</h5>
                <ul style="list-style: none; padding-left: 0;">
                    <li><button class="outline secondary style-btn" hx-get="/tenant/reportes/filtrar?modulo=TODOS" hx-target="#panel-lista-reportes" style="width: 100%; text-align: left; margin-bottom: 0.5rem;">📂 Todos los Reportes</button></li>
                    <li><button class="outline secondary style-btn" hx-get="/tenant/reportes/filtrar?modulo=TRABAJADORES" hx-target="#panel-lista-reportes" style="width: 100%; text-align: left; margin-bottom: 0.5rem;">👥 Trabajadores</button></li>
                    <li><button class="outline secondary style-btn" hx-get="/tenant/reportes/filtrar?modulo=ORGANIGRAMA" hx-target="#panel-lista-reportes" style="width: 100%; text-align: left; margin-bottom: 0.5rem;">🏢 Organigrama</button></li>
                    <li><button class="outline secondary style-btn" hx-get="/tenant/reportes/filtrar?modulo=PUESTOS" hx-target="#panel-lista-reportes" style="width: 100%; text-align: left; margin-bottom: 0.5rem;">📊 Puestos (Plazas)</button></li>
                    <li><button class="outline secondary style-btn" hx-get="/tenant/reportes/filtrar?modulo=CONCEPTOS" hx-target="#panel-lista-reportes" style="width: 100%; text-align: left; margin-bottom: 0.5rem;">⚙️ Conceptos</button></li>
                    <li><button class="outline secondary style-btn" hx-get="/tenant/reportes/filtrar?modulo=CONTRATOS" hx-target="#panel-lista-reportes" style="width: 100%; text-align: left; margin-bottom: 0.5rem;">⏳ Contratos</button></li>
                </ul>
            </nav>
        </aside>

        <section>
            <div id="panel-lista-reportes" hx-get="/tenant/reportes/filtrar?modulo=TODOS" hx-trigger="load">
                <div aria-busy="true" style="text-align: center;">Cargando catálogo de informes...</div>
            </div>
        </section>

    </div>
</article>

{{define "lista_reportes"}}
    <div style="display: flex; flex-direction: column; gap: 1rem;">
        {{range .Reportes}}
        <div style="border: 1px solid var(--pico-border-color); padding: 1.25rem; border-radius: var(--pico-border-radius); background: var(--pico-card-background-color);">
            <div style="display: flex; justify-content: space-between; align-items: flex-start; flex-wrap: wrap; gap: 1rem;">
                <div style="flex: 1; min-width: 250px;">
                    <h5 style="margin: 0 0 0.25rem 0;">{{.Nombre}}</h5>
                    <p style="margin: 0; font-size: 0.85rem; color: var(--pico-muted-color);">{{.Descripcion}}</p>
                    <span style="display: inline-block; margin-top: 0.5rem; font-size: 0.7rem; background: var(--pico-secondary-background); padding: 2px 8px; border-radius: 4px; font-weight: bold;">
                        📂 {{notempty .Modulo}}{{.Modulo}}{{end}}
                    </span>
                </div>
                
                <div role="group" style="margin-bottom: 0; width: auto;">
                    <a href="/tenant/reportes/ver-pdf?id={{.ID}}" target="_blank" class="button outline secondary" style="padding: 0.4rem 0.75rem; font-size: 0.85rem; text-decoration: none;">
                        👁️ Ver PDF
                    </a>
                    <a href="/tenant/reportes/descargar-excel?id={{.ID}}" class="button outline contrast" style="padding: 0.4rem 0.75rem; font-size: 0.85rem; text-decoration: none;">
                        📥 Excel
                    </a>
                </div>
            </div>
        </div>
        {{else}}
        <div style="text-align: center; padding: 2rem; font-style: italic; color: var(--pico-muted-color);">
            No se encontraron reportes disponibles para este módulo.
        </div>
        {{end}}
    </div>
{{end}}
```

**¿Por qué este diseño es altamente eficiente y rápido?**
1. **Cero Latencia de Base de Datos para la UI**: Al presionar los botones del menú de la izquierda, HTMX no realiza consultas pesadas a PostgreSQL; simplemente filtra un arreglo en la memoria RAM de Go de forma casi instantánea y devuelve el fragmento HTML limpio.

2. **Ver en Pantalla Instantáneo (`target="_blank"`)**: Al usar un enlace estándar (`<a>`) con `target="_blank"` hacia la ruta del PDF, el navegador web abre una nueva pestaña y activa de forma nativa su propio visor de PDF (como el de Chrome o Firefox) mientras va descargando los bytes de Go en tiempo real. Esto evita cargar librerías pesadas de JavaScript en la interfaz.