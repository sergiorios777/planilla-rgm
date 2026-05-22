# Plan de Implementación: Asignación Dinámica MEF por Tipo de Contrato

**Objetivo:**
Asignar correctamente el concepto remunerativo principal a los puestos creados, basado en el tipo de contrato y régimen laboral.

## Fase 1: Base de Datos y Modelos (El nuevo campo)
Vamos a preparar la tabla de contratos para almacenar la temporalidad/tipo del vínculo.

**Instrucciones para el Agente:**

1. **Migración SQL**: Crear un archivo de migración para alterar la tabla:
   `ALTER TABLE contratos ADD COLUMN tipo_contrato VARCHAR(100);`

2. **Modelo Go**: En `internal/models/core.go` (o donde esté definido Contrato), agregar:
   `TipoContrato string json:"tipo_contrato" a la estructura Contrato.`

## Fase 2: Configuración Dura (Harcodeo de Clasificadores)
Tal como sugeriste, usaremos tu archivo de configuración para mantener la matriz limpia y fuera de la lógica del servicio.

**Instrucciones para el Agente:**

1. En el archivo `internal/config/plantillas_conceptos.go`, agregar el siguiente mapa de constantes:

```Go
// ClasificadorMefPorContrato mapea [Régimen][Tipo Contrato] -> Código Limpio MEF
var ClasificadorMefPorContrato = map[string]map[string]string{
	"DL 276": {
		"Nombrado":     "2.1.1 1.1 2",
		"A plazo fijo": "2.1.1 1.1 3",
	},
	"Ley 30057": {
		"Alta dirección - Libre designación y remoción": "2.1.1 1.1 7",
		"Alcalde": "2.1.1 1.1 1",
	},
	"DL 1057": {
		"Indeterminado": "2.1.1 13.1 1",
		"Transitorio":   "2.1.1 13.1 2",
	},
	"DL 728": {
		"Permanentes":  "2.1.1 8.1 1",
		"A plazo fijo": "2.1.1 8.2 1",
	},
}
```

## Fase 3: Capa de Repositorios (Nuevas Consultas)
Necesitamos consultar el ID local del concepto basándonos en su clasificador MEF.

**Instrucciones para el Agente:**

1. En `internal/repository/puesto_repository.go` (o `concepto_tenant_repository.go`), crear un método para encontrar el concepto remunerativo exacto:
   `ObtenerConceptoRemunerativoPorClasificador(tenantID int, regimenID int, codigoMefLimpio string) (int, error)`
   *Consulta sugerida*: Un `SELECT ct.id cruzando conceptos_tenant, clasificadores_mef` (donde `codigo_limpio = $3`) y `regimen_concepto_tenant` (para el `regimenID`).

2. En `internal/repository/contrato_repository.go`: Actualizar el método `Crear(c *models.Contrato)` para que en el `INSERT` incluya la columna `tipo_contrato`.

## Fase 4: Refactorización de Servicios (El núcleo de la lógica)
Vamos a quitarle a PuestoService la responsabilidad de asignar conceptos y dársela a ContratoService.

**Instrucciones para el Agente:**

1. Limpiar `PuestoService`: En `internal/services/puesto_service.go`, modificar `CrearPuestoConPlantilla` para que solo llame a `s.Repo.Crear(nuevoPuesto)`. Eliminar las llamadas a `ObtenerConceptosModeloPorRegimen` y `AsignarConceptosAPuesto`.

2. Potenciar `ContratoService`: En `internal/services/contrato_service.go`:

   * Modificar (o envolver) la lógica de creación del contrato para que, inmediatamente después de hacer `Repo.Crear(c)`:

     1. Obtenga el RegimenID, RegimenDesc y SueldoPresupuestado del Puesto asignado.

     2. Llame a la plantilla base del régimen: `idsLocales := s.RepoPuesto.ObtenerConceptosModeloPorRegimen(...)`.

     3. Busque el clasificador MEF correspondiente leyendo nuestro mapa: `codigoMef := config.ClasificadorMefPorContrato[RegimenDesc][c.TipoContrato]`.

     4. Busque el ID de ese concepto remunerativo exacto usando el método del paso 3.1.

     5. Añada ese ID remunerativo exacto a la lista de `idsLocales`.

     6. Llame a la asignación de pensiones (como ya lo hace).

     7. Finalmente, guarde toda esa canasta de conceptos en el puesto: `s.RepoPuesto.AsignarConceptosAPuesto(c.PuestoID, idsFinales, SueldoPresupuestado)`.

## Fase 5: UI y Handlers (La Selección Dinámica)
Si el usuario elige un puesto "CAS (1057)", el selector de Tipo de Contrato solo debe mostrar "Indeterminado" y "Transitorio". Lo haremos con HTMX.

**Instrucciones para el Agente:**

1. Modificar el `<select name="puesto_id">` dentro del `{{define "formulario_crear"}}`:
   Le añadiremos atributos para que, al cambiar el puesto, recargue el contenedor del formulario por completo:

```HTML
<select name="puesto_id" 
        hx-get="/tenant/contratos/formulario-dinamico" 
        hx-target="#contenedor-formulario" 
        hx-swap="innerHTML">
    <option value="">— Seleccione una Plaza Vacante —</option>
    {{range .Puestos}}
        <option value="{{.ID}}" {{if eq .ID $.PuestoSeleccionadoID}}selected{{end}}>
            {{.Nombre}} (S/. {{.SueldoPresupuestado}}) - {{.RegimenDesc}}
        </option>
    {{end}}
</select>
```

2. El select de "Tipo de Contrato" en el mismo bloque:
   El select se dibujará dinámicamente según las opciones que Go envíe en una variable $.OpcionesContrato:

```HTML
<label for="tipo_contrato">Tipo de Contrato:
    <select id="tipo_contrato" name="tipo_contrato" required {{if not $.OpcionesContrato}}disabled{{end}}>
        <option value="">{{if $.OpcionesContrato}}— Seleccione Duración/Naturaleza —{{else}}Primero seleccione un puesto...{{end}}</option>
        {{range $.OpcionesContrato}}
            <option value="{{.}}">{{.}}</option>
        {{end}}
    </select>
</label>
```

3. En el Backend (`contrato_handler.go`):
   El agente creará la ruta `/tenant/contratos/formulario-dinamico`. Esta ruta leerá el `puesto_id`, buscará a qué régimen pertenece ese puesto (ej: "DL 1057"), extraerá las llaves del mapa de `config.ClasificadorMefPorContrato["DL 1057"]` (que serían `["Indeterminado", "Transitorio"]`) y renderizará únicamente el bloque `"formulario_crear"` pasándole esas opciones.

Este enfoque es excelente porque evita tener JavaScript en el cliente o selects deshabilitados "misteriosos"; es el servidor el que conduce el estado de la interfaz de forma 100% dinámica.