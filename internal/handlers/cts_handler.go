package handlers

import (
	"html/template"
	"net/http"
	"planilla-rgm/internal/repository"
	"planilla-rgm/internal/services"
	"strconv"
)

type CtsHandler struct {
	CtsRepo      *repository.CtsRepository
	CtsService   *services.CtsService
	ContratoRepo *repository.ContratoRepository
}

func (h *CtsHandler) CtsVistaUI(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("ui/templates/tenant/cts_ui.html")
	if err != nil {
		http.Error(w, "Error cargando plantilla: "+err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

func (h *CtsHandler) ListarPlanillasCts(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	planillas, err := h.CtsRepo.ListarPlanillasCts(tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFiles("ui/templates/tenant/cts_ui.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "tabla_cts_planillas", planillas)
}

func (h *CtsHandler) CrearPlanillaCts(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	tenantID := obtenerTenantID(r)

	anio, _ := strconv.Atoi(r.FormValue("anio"))
	periodo := r.FormValue("periodo")

	_, err := h.CtsService.ProcesarCtsSemestral(tenantID, anio, periodo)
	if err != nil {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`
			<div id="alerta-cts" hx-swap-oob="true">
				<article class="alert-box alert-danger">
					❌ ` + err.Error() + `
				</article>
			</div>
		`))
		h.ListarPlanillasCts(w, r)
		return
	}

	w.Write([]byte(`<div id="alerta-cts" hx-swap-oob="true"></div>`))
	w.Header().Set("HX-Trigger", "cerrarModalCts")
	h.ListarPlanillasCts(w, r)
}

func (h *CtsHandler) VerDetalleCts(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	tenantID := obtenerTenantID(r)

	planilla, err := h.CtsRepo.ObtenerPlanillaCtsPorID(id, tenantID)
	if err != nil {
		http.Error(w, "Planilla no encontrada", http.StatusNotFound)
		return
	}

	detalles, err := h.CtsRepo.ObtenerDetallesCts(id)
	if err != nil {
		http.Error(w, "Detalles no encontrados", http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFiles("ui/templates/tenant/cts_detalle_ui.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, map[string]interface{}{
		"Planilla": planilla,
		"Detalles": detalles,
	})
}

func (h *CtsHandler) SubirExcelGratificaciones(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	planillaCtsID, _ := strconv.Atoi(r.FormValue("planilla_cts_id"))

	renderDetalle := func(exitoMsg, errorMsg string) {
		planilla, err := h.CtsRepo.ObtenerPlanillaCtsPorID(planillaCtsID, tenantID)
		if err != nil {
			http.Error(w, "Planilla no encontrada", http.StatusNotFound)
			return
		}
		detalles, err := h.CtsRepo.ObtenerDetallesCts(planillaCtsID)
		if err != nil {
			http.Error(w, "Detalles no encontrados", http.StatusInternalServerError)
			return
		}
		tmpl, err := template.ParseFiles("ui/templates/tenant/cts_detalle_ui.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, map[string]interface{}{
			"Planilla":    planilla,
			"Detalles":    detalles,
			"AlertaExito": exitoMsg,
			"AlertaError": errorMsg,
		})
	}

	// Recibir archivo multipart
	err := r.ParseMultipartForm(10 << 20) // Máximo 10MB
	if err != nil {
		renderDetalle("", "Error de subida: "+err.Error())
		return
	}

	file, _, err := r.FormFile("excel_file")
	if err != nil {
		renderDetalle("", "Error: Archivo de Excel requerido.")
		return
	}
	defer file.Close()

	procesados, err := h.CtsService.ProcesarExcelGratificaciones(planillaCtsID, file)
	if err != nil {
		renderDetalle("", "Error procesando Excel: "+err.Error())
		return
	}

	renderDetalle("Se actualizaron "+strconv.Itoa(procesados)+" trabajadores con la gratificación del periodo anterior.", "")
}

func (h *CtsHandler) CerrarPlanillaCts(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	tenantID := obtenerTenantID(r)

	err := h.CtsRepo.CambiarEstadoCts(id, tenantID, "CERRADO")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Refrescar el detalle
	r.URL.RawQuery = "id=" + strconv.Itoa(id)
	h.VerDetalleCts(w, r)
}

func (h *CtsHandler) EliminarPlanillaCts(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	tenantID := obtenerTenantID(r)

	err := h.CtsRepo.EliminarPlanillaCts(id, tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.ListarPlanillasCts(w, r)
}


