package handlers

import (
	"html/template"
	"log"
	"net/http"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
	"planilla-rgm/internal/services"
	"strconv"
	"strings"
)

type PuestoConceptoHandler struct {
	Repo             *repository.PuestoConceptoRepository
	PuestoRepo       *repository.PuestoRepository // Para traer el nombre del puesto
	ContratoService  *services.ContratoService
	NotificacionRepo *repository.NotificacionRepository
}

// VistaUI carga la pantalla completa de configuración para un puesto específico
func (h *PuestoConceptoHandler) VistaUI(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	puestoID, _ := strconv.Atoi(r.URL.Query().Get("puesto_id"))

	// 1. Obtenemos los conceptos asignados
	asignados, _ := h.Repo.ObtenerAsignados(puestoID, tenantID)
	// log.Println("asignados:", asignados)

	// 2. 💡 LÓGICA INTELIGENTE EN EL SERVIDOR
	for i, cp := range asignados {

		// A. Evaluamos el puntero: Si no es nulo y es mayor a 0
		if cp.Monto != nil && *cp.Monto > 0 {
			asignados[i].MontoIngresado = true
		}
	}
	// log.Println("asignados con RequiereMontoManual:", asignados)

	disponibles, _ := h.Repo.ObtenerDisponibles(puestoID, tenantID)
	// log.Println("disponibles:", disponibles)

	// Obtener datos del puesto para mostrar el nombre
	puestoNombre := ""
	if puesto, err := h.PuestoRepo.ObtenerPorID(puestoID, tenantID); err == nil {
		puestoNombre = puesto.Nombre
	}

	datos := map[string]interface{}{
		"PuestoID":     puestoID,
		"PuestoNombre": puestoNombre,
		"Asignados":    asignados, // 💡 Mandamos la lista procesada a la vista
		"Disponibles":  disponibles,
	}

	tmpl, _ := template.ParseFiles("ui/templates/tenant/puestos_conceptos_ui.html")
	tmpl.Execute(w, datos)
}

// Listar devuelve únicamente el fragmento HTML de la tabla para HTMX
func (h *PuestoConceptoHandler) Listar(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	puestoID, _ := strconv.Atoi(r.URL.Query().Get("puesto_id"))

	// 1. Obtenemos los conceptos asignados crudos desde la BD
	asignados, _ := h.Repo.ObtenerAsignados(puestoID, tenantID)

	// 2. 💡 EL SECRETO: Aplicamos la lógica inteligente AQUÍ,
	// porque esta es la función que HTMX usa para pintar los datos reales
	for i, cp := range asignados {
		// A. Evaluamos el puntero: Si no es nulo y es mayor a 0
		if cp.Monto != nil && *cp.Monto > 0 {
			asignados[i].MontoIngresado = true
		}
	}

	// 3. Renderizamos SOLO el fragmento de la tabla con los datos ya procesados
	tmpl, _ := template.ParseFiles("ui/templates/tenant/puestos_conceptos_ui.html")
	tmpl.ExecuteTemplate(w, "tabla_asignados", asignados)
}

// Crear asigna el concepto a la plaza
func (h *PuestoConceptoHandler) Crear(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	puestoID, _ := strconv.Atoi(r.FormValue("puesto_id"))
	conceptoID, _ := strconv.Atoi(r.FormValue("concepto_tenant_id"))

	montoStr := r.FormValue("monto")
	var monto *float64
	if strings.TrimSpace(montoStr) != "" {
		m, _ := strconv.ParseFloat(montoStr, 64)
		monto = &m
	}

	nuevoPC := models.PuestoConcepto{
		PuestoID:         puestoID,
		ConceptoTenantID: conceptoID,
		Monto:            monto,
		Activo:           true,
	}

	h.Repo.Crear(&nuevoPC)

	// Refrescamos la página completa para actualizar el select de disponibles
	// w.Header().Set("HX-Redirect", "/tenant/puestos-conceptos/ui?puesto_id="+strconv.Itoa(puestoID))
	r.URL.RawQuery = "puesto_id=" + strconv.Itoa(puestoID)
	// w.WriteHeader(http.StatusOK)
	h.VistaUI(w, r)
}

// Eliminar quita un concepto de la plaza
func (h *PuestoConceptoHandler) Eliminar(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	puestoID := r.URL.Query().Get("puesto_id")

	h.Repo.Eliminar(id)

	// w.Header().Set("HX-Redirect", "/tenant/puestos-conceptos/ui?puesto_id="+puestoID)
	r.URL.RawQuery = "puesto_id=" + puestoID
	// w.WriteHeader(http.StatusOK)
	h.VistaUI(w, r)
}

// RestaurarCostosBase limpia la plaza e inyecta los conceptos por defecto
func (h *PuestoConceptoHandler) RestaurarCostosBase(w http.ResponseWriter, r *http.Request) {
	pID := r.URL.Query().Get("puesto_id")
	if pID == "" {
		pID = r.FormValue("puesto_id")
	}
	puestoID, _ := strconv.Atoi(pID)
	tenantID := obtenerTenantID(r)

	// Ejecutar restauración unificada usando el servicio
	tieneContrato, err := h.ContratoService.SincronizarConceptosPuesto(tenantID, puestoID)
	if err != nil {
		http.Error(w, "Error al restaurar conceptos: "+err.Error(), 500)
		return
	}

	// Forzamos un refresco de HTMX disparando un evento custom
	if tieneContrato {
		w.Header().Set("HX-Trigger", "refreshCostosBase")
	} else {
		w.Header().Set("HX-Trigger", "refreshCostosBaseWarning")
	}

	// Enviamos de vuelta a la función VistaUI para que recargue todo.
	r.URL.RawQuery = "puesto_id=" + strconv.Itoa(puestoID)
	h.VistaUI(w, r)
}

// RestaurarTodosCostosBase restablece los conceptos de todos los puestos del tenant de forma asíncrona
func (h *PuestoConceptoHandler) RestaurarTodosCostosBase(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)

	// Lanzamos Goroutine en segundo plano
	go func(tID int) {
		puestos, err := h.PuestoRepo.ObtenerTodos(tID)
		if err != nil {
			log.Printf("Error al obtener puestos para restauración masiva: %v", err)
			return
		}

		for _, p := range puestos {
			_, err := h.ContratoService.SincronizarConceptosPuesto(tID, p.ID)
			if err != nil {
				log.Printf("Error al restaurar conceptos para puesto %d: %v", p.ID, err)
			}
		}

		// Al terminar, registrar una notificación en la base de datos
		titulo := "🧹 Restauración Masiva Terminada"
		mensaje := "Se han restablecido los costos de todas las plazas operativas de la entidad."
		tipo := "PROCESO_EXITOSO"
		n := &models.Notificacion{
			TenantID: &tID,
			Titulo:   titulo,
			Mensaje:  mensaje,
			Tipo:     tipo,
			Leido:    false,
		}
		err = h.NotificacionRepo.Crear(n)
		if err != nil {
			log.Printf("Error al crear notificación de restauración masiva: %v", err)
		}
	}(tenantID)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`<p style="color: green; font-weight: bold; margin-bottom: 0;">🔄 El proceso de restauración masiva ha iniciado en segundo plano. Se te notificará al finalizar.</p>`))
}

// EditarMontoUI devuelve un pequeño input para el monto
func (h *PuestoConceptoHandler) EditarMontoUI(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	puestoID := r.URL.Query().Get("puesto_id")

	// Retornamos un fragmento HTML con un input que se guarda al perder el foco o presionar Enter
	html := `
		<form hx-put="/tenant/puestos-conceptos/actualizar-monto" hx-target="#contenido-tenant" style="margin-bottom:0;">
			<input type="hidden" name="id" value="` + id + `">
			<div style="display: flex; gap: 5px; align-items: center;">
				<input type="number" step="0.01" name="monto" autofocus 
					   style="margin-bottom: 0; padding: 2px 5px; width: 100px;">
				<button class="outline" style="margin-bottom: 0; padding: 2px 10px;">✅</button>
				
				<button type="button" class="outline secondary" style="margin-bottom: 0; padding: 2px 10px;"
				        hx-get="/tenant/puestos-conceptos/ui?puesto_id=` + puestoID + `" hx-target="#contenido-tenant">
				    ❌
				</button>
			</div>
		</form>
	`
	w.Write([]byte(html))
}

// ActualizarMonto procesa el cambio y refresca la tabla
func (h *PuestoConceptoHandler) ActualizarMonto(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id, _ := strconv.Atoi(r.FormValue("id"))
	monto, _ := strconv.ParseFloat(r.FormValue("monto"), 64)

	// 1. Actualizar monto
	h.Repo.ActualizarMonto(id, monto)

	// 2. Obtener el puesto_id para saber qué página recargar
	var puestoID int
	// Refrescamos toda la tabla para que los booleanos se recalculen
	// Necesitamos el puesto_id para llamar a Listar
	err := h.PuestoRepo.DB().QueryRow("SELECT puesto_id FROM puesto_conceptos WHERE id = $1", id).Scan(&puestoID)
	if err != nil {
		http.Error(w, "Error al obtener puesto", 500)
		return
	}

	// 3. Refrescamos la vista completa en el target correcto (#contenido-tenant)
	r.URL.RawQuery = "puesto_id=" + strconv.Itoa(puestoID)
	h.VistaUI(w, r)
}
