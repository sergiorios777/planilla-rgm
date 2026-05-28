# Plan para motor de Motor de Tareas y Notificaciones Asíncronas

## Ideas iniciales
Diseñar un **Motor de Tareas y Notificaciones Asíncronas** es el siguiente gran paso evolutivo para el backend en Go. Este mismo motor puede y debe ser reutilizado para avisar a los usuarios de las municipalidades cuándo termina un proceso pesado en segundo plano (como el cálculo masivo de una planilla de 1,000 trabajadores).

Para entender perfectamente cómo funciona este engranaje y cómo implementarlo de forma limpia sin ralentizar tu aplicación web, vamos a dividir el diseño en 4 componentes clave.

---

### Componente 1: El Modelo de Datos (La Base del Tiempo)

Para soportar tanto tus tareas operativas (como súper admin) como las notificaciones del pool de trabajo (para los clientes), necesitamos dos tablas en PostgreSQL. Una actúa como el **programador de eventos** y la otra como la **bandeja de entrada en tiempo real**.

```sql
-- 1. Tabla de Tareas Programadas (Uso Interno / Automatizaciones)
CREATE TABLE admin_tareas (
    id SERIAL PRIMARY KEY,
    titulo VARCHAR(150) NOT NULL,
    descripcion TEXT,
    recurrencia VARCHAR(20) NOT NULL, -- 'UNICO', 'MENSUAL', 'TRIMESTRAL'
    fecha_vencimiento TIMESTAMP NOT NULL,
    proximo_aviso TIMESTAMP NOT NULL,
    notificado_email BOOLEAN DEFAULT FALSE,
    activo BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 2. Tabla General de Notificaciones (La Cola de Avisos)
-- Esta es la que lee la UI. Si tenant_id es NULL, es para el Súper Admin.
CREATE TABLE notificaciones (
    id SERIAL PRIMARY KEY,
    tenant_id INT REFERENCES tenants(id) ON DELETE CASCADE, -- NULL = Global SaaS Admin
    usuario_id INT REFERENCES usuarios(id) ON DELETE CASCADE,
    titulo VARCHAR(200) NOT NULL,
    mensaje TEXT NOT NULL,
    tipo VARCHAR(50) NOT NULL, -- 'ALERTA_SISTEMA', 'PROCESO_EXITOSO', 'ERROR'
    leido BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

```

---

### Componente 2: El Observador en Go (El Background Worker)

Para revisar el tiempo sin bloquear las solicitudes HTTP de los usuarios, Go ofrece una herramienta nativa espectacular: las **Goroutines** combinadas con un `time.Ticker`.

Cuando tu servidor web arranca en el `main.go`, lanzaremos un hilo en segundo plano que se ejecutará, por ejemplo, cada 5 minutos. Este hilo consultará la base de datos buscando tareas cuya `fecha_vencimiento` o `proximo_aviso` sea menor o igual al tiempo actual.

#### Estructura Conceptual del Worker en Go:

```go
func IniciarObservadorTareas(db *sql.DB, mailService *MailService) {
	// Un ticker que late cada 5 minutos
	ticker := time.NewTicker(5 * time.Minute)
	
	go func() {
		for range ticker.C {
			ahora := time.Now()
			
			// 1. Buscar tareas vencidas que no han sido notificadas
			query := `SELECT id, titulo, descripcion, recurrencia FROM admin_tareas 
			          WHERE proximo_aviso <= $1 AND activo = true`
			rows, _ := db.Query(query, ahora)
			
			for rows.Next() {
				var id int
				var titulo, desc, recurrencia string
				rows.Scan(&id, &titulo, &desc, &recurrencia)
				
				// A. Escribir el aviso en la cola de notificaciones del Súper Admin (tenant_id = NULL)
				db.Exec(`INSERT INTO notificaciones (tenant_id, usuario_id, titulo, mensaje, tipo) 
				         VALUES (NULL, NULL, $1, $2, 'ALERTA_SISTEMA')`, titulo, desc)
				
				// B. Enviar correo electrónico al súper admin de inmediato
				mailService.Enviar(config.EmailAdmin, "⏰ RECORDATORIO: "+titulo, desc)
				
				// C. Gestionar la Recurrencia (Avanzar el reloj)
				var nuevoAviso time.Time
				if recurrencia == "MENSUAL" {
					nuevoAviso = ahora.AddDate(0, 1, 0) // Suma 1 mes
				} else if recurrencia == "TRIMESTRAL" {
					nuevoAviso = ahora.AddDate(0, 3, 0) // Suma 3 meses
				} else {
					// Si es único, lo desactivamos
					db.Exec("UPDATE admin_tareas SET activo = false WHERE id = $1", id)
					continue
				}
				
				db.Exec("UPDATE admin_tareas SET proximo_aviso = $1 WHERE id = $2", nuevoAviso, id)
			}
		}
	}()
}

```

---

### Componente 3: Integración con el Work Pool (Notificaciones a Municipios)

Cuando una municipalidad ejecuta el "Cierre de Mes" o "Cálculo de Planilla Masiva", tu sistema usa un *Work Pool* (un grupo de Goroutines trabajadoras).

Para notificarles, **no necesitas cambiar el observador del tiempo**. Simplemente, la Goroutine encargada de procesar la planilla, al ejecutar su última línea de código con éxito, inserta un registro en la tabla `notificaciones`.

#### Ejemplo de flujo en tu servicio de cálculos:

```go
func (s *PlanillaService) CalcularPlanillaMasiva(tenantID, usuarioID, planillaID int) {
    go func() {
        // ... Todo tu algoritmo pesado actual con excelize, fórmulas, etc. ...
        err := s.EjecutarAlgoritmoPesado()
        
        if err != nil {
            // Si falló, notificamos el error
            s.Db.Exec(`INSERT INTO notificaciones (tenant_id, usuario_id, titulo, mensaje, tipo) 
                       VALUES ($1, $2, '❌ Falló el cálculo', 'Hubo un error al procesar la planilla: ...', 'ERROR')`, 
                       tenantID, usuarioID)
            return
        }
        
        // 🎉 ¡MÁGIA!: Al terminar con éxito, dejamos el aviso en su bandeja
        s.Db.Exec(`INSERT INTO notificaciones (tenant_id, usuario_id, titulo, mensaje, tipo) 
                   VALUES ($1, $2, '✅ Proceso Terminado', 'La planilla del mes ha sido calculada exitosamente. Ya puede descargar los reportes.', 'PROCESO_EXITOSO')`, 
                   tenantID, usuarioID)
    }()
}

```

---

### Componente 4: ¿Cómo lo mostramos visualmente con HTMX y Pico CSS?

Para que la aplicación se mantenga veloz y no dependa de WebSockets pesados, utilizaremos el patrón **Polling (Consulta periódica)** nativo de HTMX.

En tu barra de navegación principal (tu menú superior o lateral), colocaremos un icono de una "Campana de Notificaciones" que se consultará a sí mismo en silencio cada 30 o 60 segundos.

#### El HTML en tu Layout Principal (`tenant_index.html` o `admin_index.html`):

```html
<div id="campana-notificaciones" 
     hx-get="/tenant/notificaciones/contador" 
     hx-trigger="every 30s, load"
     style="position: relative; cursor: pointer;">
     </div>

```

#### Lo que devuelve Go al consultar la campana:

Si hay notificaciones sin leer, Go devuelve la campana con un puntito rojo de Pico CSS. Si el usuario hace clic, abre un modal (un `<dialog>`) mostrando las últimas filas de la tabla `notificaciones`.

```html
<span class="badge" style="background: red; color: white; position: absolute; top: -5px; right: -5px; border-radius: 50%; padding: 2px 6px; font-size: 0.6rem;">
    {{.CantidadSinLeer}}
</span>
<button class="outline secondary" onclick="document.getElementById('modal-notificaciones').showModal()" style="padding: 0.2rem 0.5rem; margin-bottom: 0;">
    🔔 Notices
</button>

```

## Comentarios adicionales

Vamos a desmitificar el impacto de este diseño con números reales para que estés 100% tranquilo:

### 1. El peso de la Goroutine (El Observador)

En otros lenguajes (como PHP o Python), tener un proceso "infinito" en memoria es costoso. En Go, una **Goroutine** es un hilo hiper-ligero (empieza pesando apenas 2 KB de RAM). Despertar este hilo cada 5 minutos, ejecutar una consulta SQL buscando tareas vencidas (que tomará 1 milisegundo) y volver a dormirlo, **tiene un impacto del 0.0001% en un servidor moderno**. Es literalmente imperceptible.

### 2. El Polling de HTMX (Consultas cada 30s)

Hagamos la matemática del peor escenario:
Imagina que tienes **150 usuarios conectados al mismo tiempo** (es decir, 150 navegadores web abiertos simultáneamente mirando la pantalla).
Si cada uno hace una petición cada 30 segundos, tu servidor recibirá **5 peticiones por segundo** en promedio.
Un servidor Go básico de 10 dólares al mes puede manejar cómodamente **10,000 peticiones por segundo**.

El único "peligro" sería que la base de datos se sature. Para evitarlo, crearemos un **Índice (Index)** en la base de datos. Así, cuando Go pregunte *"¿Cuántas alertas no leídas tiene el usuario 5?"*, PostgreSQL responderá en **menos de 1 milisegundo** sin tener que escanear toda la tabla.

---

## Plan de Implementación: Motor de Tareas y Notificaciones

### Fase 1: Base de Datos (Migraciones e Índices)

Vamos a crear las tablas y, crucialmente, los índices que garantizarán que el *polling* de HTMX consuma cero recursos.

**Instrucciones para el Agente:**

1. Crear un archivo de migración SQL para las tablas `admin_tareas` y `notificaciones` usando la estructura conceptual definida anteriormente.
2. **CRÍTICO:** Incluir en la migración los siguientes índices para un rendimiento extremo:
```sql
CREATE INDEX idx_notificaciones_usuario_leido ON notificaciones(usuario_id, leido);
CREATE INDEX idx_notificaciones_tenant_leido ON notificaciones(tenant_id, leido);
CREATE INDEX idx_admin_tareas_aviso ON admin_tareas(proximo_aviso) WHERE activo = true;

```


3. Crear `internal/models/notificacion.go` y `internal/models/admin_tarea.go` reflejando estas tablas.

### Fase 2: Repositorios (Lectura Ultra-Rápida)

**Instrucciones para el Agente:**

1. Crear `internal/repository/notificacion_repository.go`:
* `ContarNoLeidas(tenantID *int, usuarioID *int) (int, error)`: Un simple `SELECT COUNT(*)` optimizado por nuestro índice.
* `ObtenerRecientes(tenantID *int, usuarioID *int, limite int) ([]models.Notificacion, error)`: Ordenadas por `created_at DESC`.
* `MarcarComoLeidas(tenantID *int, usuarioID *int) error`: Un `UPDATE` masivo a `leido = true`.


2. Crear `internal/repository/admin_tarea_repository.go`:
* `ObtenerTareasVencidas(ahora time.Time) ([]models.AdminTarea, error)`
* `ActualizarProximoAviso(id int, nuevaFecha time.Time, desactivar bool) error`
* `CrearTarea(tarea *models.AdminTarea) error`



### Fase 3: El Servicio y el "Demonio" (Background Worker)

Aquí creamos el hilo infinito seguro que procesará las tareas.

**Instrucciones para el Agente:**

1. Crear `internal/services/tarea_observador_service.go`.
2. Crear un struct `TareaObservador` que reciba ambos repositorios (tareas y notificaciones).
3. Implementar el método `IniciarObservador(db *sql.DB)`:
* Debe lanzar una Goroutine (`go func() { ... }()`).
* Usar `ticker := time.NewTicker(5 * time.Minute)`.
* En el bucle `for range ticker.C`, buscar tareas vencidas.
* Por cada tarea: Escribir una notificación al súper admin y reprogramar la fecha de la tarea (sumando 1 mes, 3 meses, o desactivándola según su recurrencia).



### Fase 4: Handlers (HTMX Endpoints)

El controlador que responderá cada 30 segundos.

**Instrucciones para el Agente:**

1. Crear `internal/handlers/notificacion_handler.go`.
2. `CampanaUI`: Llama a `ContarNoLeidas` y devuelve un micro-fragmento HTML:
`<span class="badge">{{.Conteo}}</span><button onclick="modal.show()">🔔</button>` (Aplicar estilos Pico CSS).
3. `ModalListaUI`: Llama a `ObtenerRecientes`, y marca `MarcarComoLeidas`. Devuelve el HTML de las notificaciones recientes para inyectarlo en el modal.
4. En `routes.go`, registrar estas rutas (ej. `GET /tenant/notificaciones/campana`) e inyectar en el `main.go` la llamada a `IniciarObservador()`.

### Fase 5: Integración en la Interfaz (UI)

Insertar el motor de polling en las vistas generales.

**Instrucciones para el Agente:**

1. En `ui/templates/layouts/tenant_index.html` (y en el de Admin), buscar la barra de navegación superior `<nav>`.
2. Añadir el componente de Polling:
```html
<li id="campana-contenedor" 
    hx-get="/tenant/notificaciones/campana" 
    hx-trigger="every 30s, load" 
    hx-swap="innerHTML">
    </li>

```


3. Al final del layout (antes del `</body>`), agregar el `<dialog id="modal-notificaciones">` que cargará el historial usando HTMX cuando se abra.

---

### Un detalle final para tu rol de Arquitecto

Al tener la capa de repositorios de `notificaciones` lista, tu próxima meta será ir a `planilla_service.go`, buscar el punto donde termina el cálculo pesado, y simplemente inyectar:
`h.NotificacionRepo.Crear(...)`
Al hacer eso, ¡a los 30 segundos (máximo) aparecerá el puntito rojo mágicamente en la pantalla del usuario avisándole que su planilla terminó!