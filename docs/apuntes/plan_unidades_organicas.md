# Plan de Implementación
**Objetivo:** Control de Versiones de Estructura Orgánica y vinculación con los puestos.

## Fase 1: Migración de Base de Datos (SQL)
Necesitamos crear la base de nuestra historia organizativa y enlazarla con los puestos.

**Instrucciones para el Agente:**
Crear un archivo de migración SQL con las siguientes sentencias:

* Tabla `organigramas`:
  `id (PK), tenant_id (FK), documento_aprobacion, descripcion, fecha_vigencia (DATE), activo (BOOLEAN DEFAULT false), created_at.`

* Tabla `unidades_organicas`:
  `id (PK), tenant_id (FK), organigrama_id (FK), parent_id (FK a unidades_organicas.id), codigo_mef, nombre, tipo. Añadir restricción UNIQUE (organigrama_id, nombre).`

* Alterar `puestos`:
  `ALTER TABLE puestos ADD COLUMN unidad_organica_id INT REFERENCES unidades_organicas(id) ON DELETE SET NULL;`

## Fase 2: Modelos (Go)

**Instrucciones para el Agente:**

1. En `internal/models/`, crear los structs `Organigrama` y `UnidadOrganica` que reflejen las tablas creados en la Fase 1.

## Fase 3: El Repositorio (El Motor de Transición)
Aquí ocurre la magia. Cuando se aprueba una nueva ordenanza (V2), nadie quiere volver a tipear 50 oficinas. El sistema debe clonar la estructura de la V1 a la V2 manteniendo la jerarquía (quién es jefe de quién) y luego trasladar los puestos físicos a sus nuevas "casas".

**Instrucciones para el Agente:**

1. Crear `internal/repository/organigrama_repository.go`.

2. Implementar CRUD básico: `CrearOrganigrama, ObtenerOrganigramasPorTenant, CrearUnidad, ObtenerUnidades`.

3. Implementar la función maestra `ClonarEstructuraYTrasladarPuestos(tenantID, origenID, destinoID int) error`:

   * Iniciar Transacción: `tx, err := r.db.Begin().`

   * Paso A (Leer Origen): Hacer un `SELECT` de todas las unidades orgánicas del `origenID`, ordenadas por `parent_id NULLS FIRST`. (Esto es vital para crear los "padres" antes que los "hijos").

   * Paso B (El Mapa de Memoria): En Go, crear un mapa `mapaIDs := make(map[int]int)`. Este mapa guardará `ViejoID -> NuevoID`.

   * Paso C (Clonar Unidades): Hacer un bucle `for` sobre las unidades leídas.

     - Si la unidad tenía un `parent_id`, buscar su nuevo equivalente en `mapaIDs`.

     - Insertar la unidad en la base de datos apuntando al `destinoID` (la nueva versión).

     - Guardar el `ID` retornado (`RETURNING id`) en el mapa: `mapaIDs[viejoID] = nuevoID`.

   * Paso D (Trasladar Puestos): Hacer un bucle sobre el `mapaIDs` y ejecutar: `UPDATE puestos SET unidad_organica_id = nuevoID WHERE unidad_organica_id = viejoID AND tenant_id = tenantID`.

   * Paso E (Cambio de Mando): `UPDATE organigramas SET activo = false WHERE id = origenID`
     `UPDATE organigramas SET activo = true WHERE id = destinoID`

   * Confirmar: tx.Commit().

## Fase 4: Handlers y UI (Frontend)

**Instrucciones para el Agente:**

1. Handler: Crear `organigrama_handler.go` para servir las vistas y procesar el botón de "Clonar Versión".

2. UI (Vistas):

   * Crear `organigramas_ui.html` para listar las versiones (V1, V2, etc.) con sus fechas y documentos de aprobación.

   * Añadir un botón "**Nueva Versión (Clonar de Actual)**" que dispare un modal pidiendo el número de la nueva ordenanza y la fecha de vigencia.

   * Al enviar ese modal, el Handler crea el nuevo organigrama vacío, llama a la función `ClonarEstructuraYTrasladarPuestos`, y recarga la página.

## ¿Cómo lo experimentará el usuario (Super Admin del Tenant)?
1. La municipalidad inicia con la "Ordenanza 001-2023" (Activa). Crea sus gerencias y asigna sus puestos a esas gerencias.

2. Hoy, el alcalde firma la "Ordenanza 050-2026" que cambia la estructura.

3. El usuario entra al sistema, hace clic en "Nueva Versión", escribe "Ord. 050-2026" y confirma.

4. En menos de 1 segundo, el sistema crea la V2, copia las 50 gerencias, mueve a los 200 trabajadores/puestos a las nuevas gerencias, apaga la V1 y enciende la V2.

5. El usuario solo tiene que entrar a la V2, buscar si alguna gerencia cambió de nombre (ej: de "Informática" a "Tecnologías") y editarla. ¡Los puestos ya estarán ahí adentro!


## Información Adicional

### Part 1: Diseño Visual del Organigrama en la UI (con HTMX)
Representar un organigrama jerárquico (Árbol Padre-Hijo) en HTML tradicional a veces puede ser un dolor de cabeza. Para mantener la interfaz limpia, rápida y "SaaS", utilizaremos un diseño de árbol colapsable basado en listas indentadas y detalles (`<details>`) nativos de HTML5, estilizados con tu framework CSS (Pico CSS).

La clave es que la vista del organigrama esté dividida en dos componentes:

1. El selector de versión (Organigrama Activo): Un dropdown o pestañas para cambiar entre la Ordenanza del 2024, 2025, etc.

2. El árbol dinámico: Una macro o función recursiva en los templates de Go que se dibuja a sí misma para pintar los infinitos niveles de subgerencias.

#### El Código de la UI (`ui/templates/tenant/organigramas_ui.html`)
Así es como estructuraremos la vista visual utilizando HTMX para que la recarga de versiones sea instantánea:

``` HTML
<article>
    <header style="display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 1rem;">
        <div>
            <h3 style="margin: 0;">🏢 Estructura Orgánica (Organigrama)</h3>
            <p style="margin: 0;"><small>Gestión de oficinas y dependencias según convenios institucionales.</small></p>
        </div>
        
        <div role="group" style="margin-bottom: 0;">
            <select name="organigrama_id" 
                    hx-get="/tenant/organigrama/arbol" 
                    hx-target="#contenedor-arbol-organico"
                    style="margin-bottom: 0; max-width: 250px;">
                {{range .Organigramas}}
                    <option value="{{.ID}}" {{if .Activo}}selected{{end}}>
                        {{.DocumentoAprobacion}} {{if .Activo}}(Activo){{end}}
                    </option>
                {{end}}
            </select>
            <button class="outline" onclick="document.getElementById('modal-clonar-organigrama').showModal()">
                ✨ Nueva Versión (Clonar)
            </button>
        </div>
    </header>

    <div id="contenedor-arbol-organico" hx-get="/tenant/organigrama/arbol" hx-trigger="load">
        <div aria-busy="true" style="text-align: center;">Cargando estructura orgánica...</div>
    </div>
</article>

{{define "nodo_organigrama"}}
    <ul>
        {{range .Nodos}}
        <li>
            <details open style="margin-bottom: 0.5rem; padding-bottom: 0.5rem; border-bottom: 1px dashed #ccc;">
                <summary style="font-weight: bold; color: var(--pico-primary);">
                    📁 [{{.Tipo}}] {{.Nombre}} 
                    <small style="color: #666; font-weight: normal; margin-left: 10px;">
                        (ID: {{.ID}} | Código MEF: {{if .CodigoMef}}{{.CodigoMef}}{{else}}N/A{{end}})
                    </small>
                    <span style="float: right; font-weight: normal;">
                        <a href="#" hx-get="/tenant/organigrama/unidad/editar_ui?id={{.ID}}" hx-target="#modal-unidad" style="font-size: 0.8rem; margin-right: 10px;">✏️ Editar</a>
                        <a href="#" hx-get="/tenant/organigrama/unidad/agregar_hijo_ui?parent_id={{.ID}}" hx-target="#modal-unidad" style="font-size: 0.8rem; color: green;">➕ Añadir Subunidad</a>
                    </span>
                </summary>
                
                {{if .Hijos}}
                    {{template "nodo_organigrama" dict "Nodos" .Hijos}}
                {{else}}
                    <p style="margin-left: 2rem; margin-bottom: 0; font-style: italic; color: #888;">
                        <small>📍 No tiene subunidades. Plazas asignadas: {{.TotalPuestos}}</small>
                    </p>
                {{end}}
            </details>
        </li>
        {{end}}
    </ul>
{{end}}
```

Nota pedagógica de Go: Para que la recursividad funcione (`{{template "nodo_organigrama" ...}}`), el Handler de Go debe estructurar las unidades en memoria como un árbol de structs donde cada nodo tiene un slice de hijos: `Hijos []UnidadNodo`.

### Part 2: ¿Cómo implementar la "Inmutabilidad Transaccional"?
La Inmutabilidad Transaccional establece que la tabla `boletas` (y sus detalles de conceptos) no debe depender de relaciones (`FK`) dinámicas que puedan cambiar de nombre en el futuro.

Para lograr esto, modificaremos o diseñaremos las tablas históricas de planillas para que actúen como "cajas negras" o capturas fotográficas (snapshots).

#### El Diseño de Base de Datos Inmutable
Cuando corras el motor de cálculo a fin de mes, en lugar de guardar únicamente el unidad_organica_id en la boleta, duplicaremos la información crítica como campos de texto plano.

```SQL
-- 1. Tabla de Boletas de Pago (Historial Inmutable)
CREATE TABLE boletas (
    id SERIAL PRIMARY KEY,
    tenant_id INT NOT NULL REFERENCES tenants(id),
    periodo_mes INT NOT NULL,  -- Ej: 5 (Mayo)
    periodo_anio INT NOT NULL, -- Ej: 2026
    trabajador_id INT NOT NULL, -- Mantenemos FK al trabajador por legajo
    puesto_id INT NOT NULL,     -- Mantenemos FK al puesto físico
    
    -- FOTOGRAFÍA INMUTABLE EN EL MOMENTO DEL CÁLCULO:
    trabajador_nombre_completo VARCHAR(250) NOT NULL, -- "Juan Perez"
    trabajador_numero_documento VARCHAR(20) NOT NULL, -- DNI
    puesto_codigo_airhsp VARCHAR(20),                -- Código MEF del puesto
    puesto_nombre VARCHAR(200) NOT NULL,             -- "Especialista TI II"
    
    -- NUESTRO CABO SUELTO RESUELTO:
    organigrama_documento_aprobacion VARCHAR(200) NOT NULL, -- "Ordenanza Municipal N° 045-2024"
    unidad_organica_nombre VARCHAR(200) NOT NULL,           -- "Subgerencia de Tecnologías"
    unidad_organica_tipo VARCHAR(50) NOT NULL,              -- "Subgerencia"
    
    sueldo_basico_historico NUMERIC(10,2) NOT NULL,
    total_ingresos NUMERIC(10,2) NOT NULL,
    total_descuentos NUMERIC(10,2) NOT NULL,
    total_aportes NUMERIC(10,2) NOT NULL,
    neto_a_pagar NUMERIC(10,2) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 2. Detalles de la boleta (Conceptos calculados inmutables)
CREATE TABLE boleta_conceptos (
    id SERIAL PRIMARY KEY,
    boleta_id INT NOT NULL REFERENCES boletas(id) ON DELETE CASCADE,
    concepto_tenant_id INT, -- Opcional, por si se borra el concepto original
    
    -- FOTOGRAFÍA DEL CONCEPTO:
    codigo_sunat VARCHAR(10) NOT NULL,         -- Ej: "0121"
    nombre_en_boleta VARCHAR(150) NOT NULL,     -- Ej: "Remuneración Básica" (Aunque el tenant lo cambie luego, aquí se queda)
    tipo_concepto VARCHAR(20) NOT NULL,        -- INGRESO, DESCUENTO, APORTE
    monto_calculado NUMERIC(10,2) NOT NULL
);
```

#### ¿Cómo interactúan las Jerarquías Versionadas con esta Inmutabilidad?
El flujo de trabajo unificado que ejecutará tu sistema en Go será el siguiente:

1. Fase de Operación Diaria (Estructura Viva): El usuario navega por la UI jerárquica del Organigrama V2. Mueve un puesto de la oficina "A" a la oficina "B". Al hacer esto, el sistema actualiza internamente el campo `puestos.unidad_organica_id`.

2. Fase de Cierre de Mes (Cálculo de Planilla):
El motor de Go inicia el proceso de cálculo. Hace un `SELECT` de los puestos y sus uniones (`JOIN`) actuales:

```SQL
SELECT p.nombre, p.codigo_airhsp, u.nombre, u.tipo, o.documento_aprobacion
FROM puestos p
INNER JOIN unidades_organicas u ON p.unidad_organica_id = u.id
INNER JOIN organigramas o ON u.organigrama_id = o.id
WHERE p.tenant_id = $1
```

3. El Acto de Inmutabilidad:
Al ejecutar el `INSERT INTO boletas`, Go toma los strings devueltos por el `JOIN` ("Subgerencia de Tecnologías", "Ordenanza 045-2024") y los escribe directamente en las columnas `unidad_organica_nombre` y `organigrama_documento_aprobacion`.

## ¿Por qué este diseño mixto es perfecto?
Si el próximo año el alcalde deroga la estructura anterior y tú corres nuestro proceso de "Mover Puestos de V2 a V3", todas las unidades de la V2 quedarán desactivadas o archivadas.

Sin embargo, cuando el usuario vaya al módulo "Historial de Boletas" y busque la boleta de Mayo de 2026, la consulta SQL será un simple `SELECT * FROM boletas WHERE id = $1`. No requerirá ningún `JOIN` hacia la tabla unidades_organicas. El reporte se pintará instantáneamente mostrando que en ese mes, el puesto pertenecía a la "Subgerencia de Tecnologías" bajo la "Ordenanza 045-2024", intacta y protegida legalmente.