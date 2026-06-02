# Refactorización de asignación de conceptos

## Idea inicial

Hola, Gemini. Soy yo nuevamente.

Contexto y objetivo:
Ayúdame, a analizar el proceso de asignación de conceptos cuando se crea un contrato, hay conceptos que se asignan por el régimen laboral asignado al puesto y otros conceptos que se asignan por régimen pensionario asignado al trabajador.
Puedes identificar las funciones relacionadas para este propósito, y evaluar si tiene la suficiente independencia para que puedan ser llamadas por otros módulo, como por ejemplo el módulo puestos conceptos con el botón "restaurar conceptos base".

Importante: Actualmente el botón "restaurar conceptos base" solo restaura los conceptos relacionados al régimen laboral. Quizá sea bueno implementar un diseño que utilice estrategias de selección: 1 ) si puede identificar al trabajador a a través de un contrato vigente (el contrato vincula al puesto con el trabajador) puede agregar los conceptos por régimen laboral (del puesto) y por régimen pensionario (del trabajador) y 2) si no hay contrato vigente solo asigna lanza una advertencia y solo asigna conceptos relacionados al régimen laboral.

Alcance de la asignación de conceptos:
1. A nivel general para todos los puestos que se pueda ejecutar en segundo plano. Colocar un botón restarurar conceptos base para todos. (Advertir que van a eliminar los existentes)
2. Dentro de cada puesto (puesto_concepto) con el botón indicado inialmente "Restaurar conceptos base"

Petición:
1. Analiza si es recomendable una refactorización para lograr nuestro objetivo. 
2. Luego prepararemos el plan detallado para pasarle al agente Antigravity CLI.

¿Requieres alguna otra parte del código para tu análisis? 
Te subí los archivos relevantes.

## Análisis y propuesta

Lo que propones no es solo una excelente idea de usabilidad, sino una necesidad imperativa para evitar la **desincronización de datos** en un sistema de planillas.

Cuando un operador de recursos humanos manipula manualmente los conceptos de un puesto (añadiendo bonificaciones temporales o judiciales) y luego desea regresar al estado "de fábrica", el sistema debe ser lo suficientemente inteligente como para saber si esa plaza está vacía o si está ocupada, ya que un trabajador bajo la ONP no puede recibir el impacto de las tasas de una AFP por un error de restauración.

A continuación, presento el análisis arquitectónico y el plan detallado para que ejecutes la refactorización con **Antigravity CLI**.

---

### Part 1: Análisis Arquitectónico y Factibilidad

**Sí, la refactorización es totalmente recomendada y necesaria.** 

#### El Diagnóstico Actual

Si observamos las líneas 25 a 63 de tu archivo `contrato_service.go`, toda la lógica matemática y legal para mezclar la plantilla base del régimen, excluir ciertos clasificadores, buscar el código MEF correcto según el `TipoContrato` e inyectar el régimen pensionario del trabajador, está **atrapada** dentro de la función `CrearContrato`.

Debido a este acoplamiento, tu handler de conceptos (`puesto_concepto_handler.go`) no puede reutilizar esa inteligencia y se ve obligado a llamar únicamente a `h.PuestoRepo.RestaurarPlantillaBase`, dejando fuera por completo al trabajador y a su pensión.

#### La Solución Ideada: El Motor de Sincronización Único

Moveremos todo ese bloque lógico a una nueva función independiente dentro de `ContratoService` llamada `SincronizarConceptosPuesto`. Esta función no creará contratos; solo leerá el estado actual del puesto y aplicará las dos estrategias que definiste:

| Estado del Puesto | Estrategia de Asignación | Acción de la Interfaz (UI) |
| --- | --- | --- |
| **Con Contrato Activo** | Plantilla Base Régimen + Clasificador MEF del Contrato + Pensión del Trabajador. | Restaura con éxito total. |
| **Vacante (Sin Contrato)** | Únicamente Plantilla Base del Régimen Laboral. | Restaura y lanza una advertencia en pantalla. |

---

### Part 2: Plan de Implementación para Antigravity CLI

Dale estas instrucciones quirúrgicas a tu agente a través de la CLI para automatizar la refactorización en las 3 capas del software.

#### Fase 1: Capa de Servicios (Extracción del Motor Lógico)

**Instrucciones para el Agente:**

1. En `internal/services/contrato_service.go`, crear el método `SincronizarConceptosPuesto(tenantID, puestoID int) (bool, error)`.
2. **Implementar la lógica de selección:**
* **Paso A:** Consultar si existe un contrato activo para este puesto: `SELECT id, trabajador_id, tipo_contrato FROM contratos WHERE puesto_id = $1 AND tenant_id = $2 AND activo = true LIMIT 1`.
* **Paso B (Estrategia 2 - Vacante):** Si no hay contrato activo, obtener el régimen del puesto, ejecutar `s.RepoPuesto.RestaurarPlantillaBase(puestoID, tenantID, puesto.RegimenID)` y retornar `false, nil`.
* **Paso C (Estrategia 1 - Ocupado):** Si hay un contrato activo, ejecutar exactamente las mismas reglas de extracción (plantilla base del régimen, búsqueda del clasificador MEF por el tipo de contrato, inyección de códigos de pensión SUNAT del trabajador) y guardar el resultado final mediante `s.RepoPuesto.AsignarConceptosAPuesto`. Retornar `true, nil`.


3. **Refactorizar `CrearContrato`:** Reemplazar las líneas 20 a 64 de `CrearContrato` para que, después de guardar el contrato, simplemente llame internamente a `s.SincronizarConceptosPuesto(c.TenantID, c.PuestoID)`.

#### Fase 2: Capa de Controladores (Soporte para los dos alcances)

Vamos a adaptar el handler individual y a crear el procesador en segundo plano para la restauración masiva.

**Instrucciones para el Agente:**

1. En `internal/handlers/puesto_concepto_handler.go`, inyectar la dependencia de `ContratoService *services.ContratoService` dentro del struct `PuestoConceptoHandler`.
2. **Modificar `RestaurarCostosBase` (Alcance Individual):**
* Reemplazar la llamada directa al repositorio por: `tieneContrato, err := h.ContratoService.SincronizarConceptosPuesto(tenantID, puestoID)`.
* Si `tieneContrato` es `false`, inyectar una cabecera de respuesta HTMX personalizada: `w.Header().Set("HX-Trigger", "refreshCostosBaseWarning")`. Si es `true`, lanzar `refreshCostosBase`.


3. **Crear `RestaurarTodosCostosBase` (Alcance General - Segundo Plano):**
* Crear un método en el handler que reciba una petición POST.
* Ejecutar una Goroutine en segundo plano (Work Pool) para no congelar la pantalla:
```go
go func(tID int) {
    // 1. Obtener todos los IDs de puestos del tenant
    puestosIDs, _ := h.PuestoRepo.ObtenerTodosIDs(tID)
    for _, id := range puestosIDs {
        h.ContratoService.SincronizarConceptosPuesto(tID, id)
    }
    // 2. Al terminar, escribir una alerta usando nuestro nuevo sistema de notificaciones
    h.NotificacionRepo.Crear(tID, "🧹 Restauración Masiva Terminada", "Se han restablecido los costos de todas las plazas operativas.", "PROCESO_EXITOSO")
}(tenantID)

```

#### Fase 3: Capa de Interfaz de Usuario (UI con Alertas Dinámicas)

**Instrucciones para el Agente:**

1. En `ui/templates/tenant/puestos_conceptos_ui.html`:
* Agregar un escuchador de eventos Javascript para capturar la advertencia de HTMX:
```javascript
document.body.addEventListener("refreshCostosBaseWarning", function() {
    alert("⚠️ El puesto está VACANTE. Se han restaurado únicamente los conceptos del régimen laboral base, excluyendo pensiones.");
});

```

2. En la vista principal del listado de puestos (`ui/templates/tenant/puestos_ui.html`):
* Colocar un botón general en la cabecera:
```html
<button class="outline contrast" 
        hx-post="/tenant/puestos-conceptos/restaurar-todos" 
        hx-confirm="🚨 ¡ATENCIÓN! Esta acción eliminará todas las modificaciones manuales de conceptos en TODOS los puestos del sistema para restablecer sus valores de fábrica. ¿Desea continuar?"
        style="border-color: red; color: red;">
    🔄 Restaurar Todos los Puestos
</button>

```

---

### ¿Requiero alguna otra parte del código?

Para la ejecución de este plan específico, **no es necesario que subas más archivos**. El agente de la CLI tendrá suficiente información analizando la estructura expuesta en tu `contrato_service.go` y la consulta del JOIN que ya conoce de los puestos.

Con este diseño, lograste que tu aplicación sea modular, que no duplique código y que aproveche el motor asíncrono para tareas masivas. ¡Pásale el plan a Antigravity CLI para ver cómo unifica ambos mundos!