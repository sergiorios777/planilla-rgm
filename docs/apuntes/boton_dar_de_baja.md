# Función de dar de baja los contratos

## Las ideas iniciales
Los archivos: @[internal\handlers\contrato_handler.go], @[internal\repository\contrato_repository.go] y @[internal\services\contrato_service.go].

Tras analizar detenidamente los archivos indicados, aquí tienes la evaluación detallada de cómo se gestiona la baja de un contrato y su vigencia
  en el sistema:

### 1. contrato_service.go                                                                                                                          

* No contiene ningún procedimiento para dar de baja, actualizar o inactivar un contrato. Solo expone lógica para la creación de contratos (
  CrearContrato ), sincronización de conceptos del puesto ( SincronizarConceptosPuesto ), e inyección de pensiones ( AsignarPensionesAutomaticas ).

### 2. contrato_repository.go                                                                                                                       

* No contiene el método de actualización físicamente.
[!NOTE]
El método  Actualizar(c *models.Contrato) error  de  ContratoRepository  está definido en el archivo concepto_repository.go.

* Este método realiza lo siguiente en una transacción de base de datos:
  1. Actualiza el estado del contrato ( activo = $3 , que recibe  false  al darse de baja) y las fechas correspondientes.
  2. Si el contrato pasa a inactivo ( !c.Activo ), ejecuta inmediatamente un  UPDATE puestos SET estado = 'VACANTE' WHERE id = $1  para liberar la plaza/puesto.

### 3. contrato_handler.go

* El handler expone la función contrato_handler.go ( ContratoHandler.Actualizar ), la cual procesa el formulario enviado desde la interfaz web (donde desactivar el checkbox de "activo" envía un valor que resulta en  cActualizado.Activo = false ).
* Llama a  h.Repo.Actualizar(&cActualizado) .
* Retorna un encabezado HTMX ( HX-Trigger: recargarTablaContratos ) para indicarle a la UI que recargue la lista de contratos de inmediato, reflejando el cambio de estado en pantalla.

---
### ⚠️ Observación Importante sobre la consistencia de datos:

Aunque la baja del contrato y la liberación de la plaza a  VACANTE  se realizan a nivel de base de datos y se visualizan de inmediato en la UI, existe una inconsistencia lógica:

* Cuando un contrato se inactiva, no se invoca la sincronización de conceptos del puesto.
* En el servicio contrato_service.go, se define que cuando un puesto no tiene un contrato activo (es decir, está vacante), se debe restaurar su plantilla base de conceptos de régimen y remover los conceptos propios del ex-trabajador (como pensiones de su AFP o clasificadores de contrato específicos).
* Sin embargo, el handler contrato_handler.go no llama a este servicio de sincronización. Como consecuencia, la plaza quedará liberada, pero mantendrá asignada la estructura de costos y pensiones del trabajador anterior hasta que alguna otra acción fuerce una sincronización manual.

---

## Las preguntas 
¿Desde el punto de vista UX es conveniente agregar un botón exclusivo para dar de baja un contrato, que salga un modal y pida motivo (Libre remoción, fin del contrato, renuncia, destitución) y fecha de fin del contrato?
¿Es necesario agregar una funcionalidad para actualizar automáticamente a la baja del contrato los conceptos propios del régimen y liberar los del usuario como se menciona en el análisis?
¿Se debe aplicar también esto al momento de hacer clic directamente en el campo 'Activo' del formulario de edición o es mejor dejar este botón como solo lectura y evitar accidentes o complicaciones de uso para el usuario?

### Resumen de la Solución

Vamos a transformar la "baja de contrato" de un simple cambio de estado oculto en un formulario general, a un **Evento de Dominio** explícito. Esto significa crear un flujo visual propio (un modal) que exija información crítica (motivo y fecha) y un endpoint en Go dedicado exclusivamente a procesar esta baja, el cual se encargará de limpiar la estructura de costos del puesto automáticamente.

---

### 1. El Botón Exclusivo de "Dar de Baja" (UX y Negocio)

**Es absolutamente conveniente y es la mejor práctica de la industria.**
Dar de baja a un trabajador no es lo mismo que corregir un error tipográfico en su apellido. Es un evento administrativo crítico.

* **Trazabilidad Legal:** En la administración pública, registrar el motivo exacto (Libre remoción, término de contrato, renuncia) y la fecha de cese es fundamental para habilitar el futuro cálculo de su liquidación de beneficios sociales o CTS.
* **Claridad Visual:** Un botón rojo o de advertencia que diga "Terminar Contrato" con un modal dedicado evita que el usuario tenga que adivinar dónde se desactiva a un trabajador.

### 2. Actualización Automática de Conceptos (Consistencia)

**Sí, es estrictamente necesario.**
Como bien notaste en tu análisis, si un trabajador afiliado a "AFP Integra" renuncia, el sistema marca el puesto como `VACANTE`. Si no ejecutamos la limpieza, ese puesto vacío seguirá arrastrando el clasificador de la comisión de AFP Integra del ex-trabajador. Esto corrompería las proyecciones presupuestales y el costeo del Presupuesto Analítico de Personal (PAP).

Al procesar la baja, el backend debe llamar obligatoriamente a tu función de sincronización para "resetear" el puesto a su plantilla de régimen base.

### 3. El Campo "Activo" en el Formulario de Edición

**Debes hacerlo de solo lectura (`disabled`) o retirarlo por completo del formulario de edición general.**
Mantener un simple *checkbox* de "Activo" en el formulario de edición general es peligroso:

* Fomenta accidentes (un clic erróneo desactiva al trabajador sin pedir fecha de fin).
* Permite a los usuarios evadir el flujo correcto, ya que podrían apagar el contrato sin especificar el "Motivo", dejando la base de datos con información incompleta.

El formulario de "Editar Contrato" debe usarse únicamente para corregir datos mal ingresados al momento de la creación (como ajustar un sueldo o cambiar una fecha de inicio mal digitada).

---

## Propuesta del plan

Transformar la "baja de contrato" en un evento explícito con motivo y fecha no solo es la mejor práctica, sino que es un requisito casi indispensable para auditorías y cálculos de liquidaciones futuras. Además, como bien observaste, esto nos permite limpiar la plaza y dejarla lista (vacante y con sus conceptos base) para el siguiente trabajador.

Dado que en el modelo actual `Contrato` no existe un campo para almacenar el "motivo" de la baja, nuestro plan incluirá agregarlo a la base de datos.

---

### Plan de Implementación: Refactorización del Evento "Dar de Baja"

#### Fase 1: Actualización del Modelo y Base de Datos

Necesitamos un lugar para guardar el motivo de la baja (Renuncia, Destitución, etc.) para mantener el historial legal.

**Instrucciones para el Agente:**

1. Crear una migración SQL para agregar el campo a la tabla de contratos:
```sql

```



ALTER TABLE contratos ADD COLUMN motivo_baja VARCHAR(100);

```
2. En el archivo de modelos correspondientes (ej. `internal/models/core.go` o donde esté `Contrato`), actualizar el struct `Contrato` agregando el nuevo campo[cite: 4]:
   ```go
   type Contrato struct {
       // ... campos existentes ...
       FechaFin *string `json:"fecha_fin"` // Puntero para permitir nulos[cite: 4]
       Activo   bool    `json:"activo"`[cite: 4]
       MotivoBaja *string `json:"motivo_baja,omitempty"` // NUEVO
       // ...
   }

```

#### Fase 2: Capa de Repositorios (`contrato_repository.go`)

Crearemos un método exclusivo y seguro para manejar la transacción a nivel de base de datos, separándolo del `Actualizar` genérico.

**Instrucciones para el Agente:**

1. En `internal/repository/contrato_repository.go`, agregar el método `DarDeBaja`:
```go
func (r *ContratoRepository) DarDeBaja(contratoID int, tenantID int, fechaFin string, motivo string) error {
    tx, err := r.db.Begin()
    if err != nil {
        return err
    }

    // 1. Inactivar el contrato, poner fecha fin y motivo
    queryContrato := `UPDATE contratos SET activo = false, fecha_fin = $1, motivo_baja = $2 WHERE id = $3 AND tenant_id = $4`
    _, err = tx.Exec(queryContrato, fechaFin, motivo, contratoID, tenantID)
    if err != nil {
        tx.Rollback()
        return err
    }

    // 2. Liberar el puesto (Pasarlo a VACANTE)
    queryPuesto := `UPDATE puestos SET estado = 'VACANTE' WHERE id = (SELECT puesto_id FROM contratos WHERE id = $1) AND tenant_id = $2`
    _, err = tx.Exec(queryPuesto, contratoID, tenantID)
    if err != nil {
        tx.Rollback()
        return err
    }

    return tx.Commit()
}

```



```

#### Fase 3: Capa de Servicios (`contrato_service.go`)
Aquí conectamos la baja del contrato con tu poderosa función de limpieza de conceptos[cite: 2].

**Instrucciones para el Agente:**
1. En `internal/services/contrato_service.go`, crear el método `TerminarContrato`:
   ```go
   func (s *ContratoService) TerminarContrato(contratoID int, tenantID int, fechaFin string, motivo string) error {
       // 1. Obtener el contrato para saber qué puesto debemos limpiar
       contrato, err := s.Repo.ObtenerPorID(contratoID, tenantID) // Asumiendo que existe este método o usar lógica equivalente
       if err != nil {
           return err
       }

       // 2. Dar de baja en la base de datos
       err = s.Repo.DarDeBaja(contratoID, tenantID, fechaFin, motivo)
       if err != nil {
           return err
       }

       // 3. 🧹 MAGIA: Limpiar los conceptos del puesto inyectando la plantilla base
       // Esto elimina las AFPs del ex-trabajador y deja la plaza en blanco.
       _, err = s.SincronizarConceptosPuesto(tenantID, contrato.PuestoID)[cite: 2]
       
       return err
   }

```

#### Fase 4: Capa de Controladores (`contrato_handler.go`)

Vamos a quitarle la responsabilidad al `Actualizar` general y crear el *endpoint* para nuestro nuevo modal.

**Instrucciones para el Agente:**

1. En `internal/handlers/contrato_handler.go`, modificar el método `Actualizar`. **Eliminar o ignorar** la lectura de `r.FormValue("activo") == "on"`. El contrato siempre mantendrá su estado actual en las ediciones regulares.


2. Crear el nuevo método `ProcesarBaja`:
```go
func (h *ContratoHandler) ProcesarBaja(w http.ResponseWriter, r *http.Request) {
    r.ParseForm()
    tenantID := obtenerTenantID(r)
    contratoID, _ := strconv.Atoi(r.FormValue("contrato_id"))
    fechaFin := r.FormValue("fecha_fin")
    motivo := r.FormValue("motivo")

    servicioContrato := services.ContratoService{
        RepoPuesto: h.PuestoRepo,
        Repo: h.Repo,
        RepoTrabajador: h.TrabajadorRepo,
    }

    err := servicioContrato.TerminarContrato(contratoID, tenantID, fechaFin, motivo)
    if err != nil {
        http.Error(w, "Error al procesar la baja del contrato", http.StatusInternalServerError)
        return
    }

    // Cerrar el modal y recargar la tabla usando eventos HTMX
    w.Header().Set("HX-Trigger", "cerrarModalBaja, recargarTablaContratos")[cite: 5]
    w.WriteHeader(http.StatusOK)
}

```



```

#### Fase 5: Capa de Interfaz (HTML/HTMX)
Aseguramos la UI retirando opciones peligrosas y agregando nuestro evento explícito.

**Instrucciones para el Agente:**
1. En `ui/templates/tenant/contratos_ui.html`, buscar el bloque `formulario_editar` y **eliminar** el checkbox o toggle que permite cambiar el estado a "Activo" o "Inactivo".
2. En la tabla de listado de contratos, agregar un botón rojo junto al de editar (solo visible si el contrato está activo):
   ```html
   {{if .Activo}}
   <button class="outline contrast" style="color: #d32f2f; border-color: #d32f2f; padding: 0.2rem 0.5rem;" 
           onclick="abrirModalBaja('{{.ID}}', '{{.TrabajadorNombre}}')">
       🛑 Dar de Baja
   </button>
   {{end}}

```

3. Agregar el `<dialog>` del modal al final del archivo:
```html
<dialog id="modal-baja">
    <article>
        <header>
            <a href="#close" aria-label="Close" class="close" onclick="document.getElementById('modal-baja').close()"></a>
            <h3>🛑 Finalizar Contrato</h3>
        </header>
        <p>Está a punto de dar de baja a <strong id="baja-trabajador-nombre"></strong>. La plaza quedará vacante y se limpiará su estructura de costos.</p>

        <form hx-post="/tenant/contratos/baja" hx-swap="none">
            <input type="hidden" name="contrato_id" id="baja-contrato-id">

            <label>Motivo de la Baja
                <select name="motivo" required>
                    <option value="">-- Seleccione un motivo --</option>
                    <option value="Término de Contrato">Término de Contrato</option>
                    <option value="Renuncia Voluntaria">Renuncia Voluntaria</option>
                    <option value="Libre Remoción">Libre Remoción (Cargo de Confianza)</option>
                    <option value="Destitución">Destitución / Sanción</option>
                    <option value="Fallecimiento">Fallecimiento</option>
                </select>
            </label>

            <label>Fecha Efectiva de Cese
                <input type="date" name="fecha_fin" required>
            </label>

            <footer>
                <button type="button" class="secondary outline" onclick="document.getElementById('modal-baja').close()">Cancelar</button>
                <button type="submit" style="background-color: #d32f2f; border-color: #d32f2f;">Confirmar Baja</button>
            </footer>
        </form>
    </article>
</dialog>

<script>
    function abrirModalBaja(id, nombre) {
        document.getElementById('baja-contrato-id').value = id;
        document.getElementById('baja-trabajador-nombre').innerText = nombre;
        document.getElementById('modal-baja').showModal();
    }

    document.body.addEventListener("cerrarModalBaja", function() {
        document.getElementById('modal-baja').close();
    });
</script>

```



```
4. Asegurar que las rutas apunten correctamente (`POST /tenant/contratos/baja` hacia `ProcesarBaja`) en `internal/routes/routes.go`.

---

Copia todo este bloque y envíaselo a Antigravity CLI. Tu sistema ahora tendrá una lógica de dominio impecable y a prueba de errores humanos.

```