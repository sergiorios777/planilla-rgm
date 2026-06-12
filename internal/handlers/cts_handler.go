package handlers

import (
	"html/template"
	"net/http"
	"planilla-rgm/internal/repository"
	"planilla-rgm/internal/services"
	"strconv"
	"strings"
	"time"
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
				<article style="background-color: #ffcdd2; color: #b71c1c; padding: 1rem; margin-bottom: 1rem; border-radius: 5px;">
					❌ Error al procesar CTS: ` + err.Error() + `
				</article>
			</div>
		`))
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

func (h *CtsHandler) LiquidacionesVistaUI(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("ui/templates/tenant/liquidaciones_ui.html")
	if err != nil {
		http.Error(w, "Error cargando plantilla: "+err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

func (h *CtsHandler) ListarLiquidacionesCese(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	liquidaciones, err := h.CtsRepo.ListarLiquidacionesCese(tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFiles("ui/templates/tenant/liquidaciones_ui.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "tabla_liquidaciones", liquidaciones)
}

func (h *CtsHandler) FormularioCrearUI(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	contratos, err := h.ContratoRepo.ObtenerTodos(tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Filtrar contratos para que no envíe contratos que no tengan ID válido o no activos si deseamos
	// Para mantenerlo amplio, enviaremos todo.
	tmpl, err := template.ParseFiles("ui/templates/tenant/liquidaciones_ui.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "formulario_crear_liq", contratos)
}

func (h *CtsHandler) CalcularLiquidacionCese(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	contratoID, _ := strconv.Atoi(r.FormValue("contrato_id"))
	fechaInicio, _ := time.Parse("2006-01-02", r.FormValue("fecha_inicio"))
	fechaCese, _ := time.Parse("2006-01-02", r.FormValue("fecha_cese"))
	motivo := r.FormValue("motivo")

	liq, err := h.CtsService.CalcularLiquidacion(contratoID, fechaInicio, fechaCese, motivo)
	if err != nil {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<p style="color:red; font-weight: bold; padding: 1rem; background-color: #ffcdd2; border-radius: 5px;">❌ Error de cálculo: ` + err.Error() + `</p>`))
		return
	}

	vacTruncas, _ := strconv.ParseFloat(strings.TrimSpace(r.FormValue("vacaciones_truncas")), 64)
	gratiTrunca, _ := strconv.ParseFloat(strings.TrimSpace(r.FormValue("gratificacion_trunca")), 64)
	liq.MontoVacacionesTruncas = vacTruncas
	liq.MontoGratiTrunca = gratiTrunca
	liq.TotalLiquidacion = liq.MontoCts + vacTruncas + gratiTrunca

	tmpl, err := template.ParseFiles("ui/templates/tenant/liquidaciones_ui.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "vista_previa_liquidacion", liq)
}

func (h *CtsHandler) GuardarLiquidacionCese(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	contratoID, _ := strconv.Atoi(r.FormValue("contrato_id"))
	fechaInicio, _ := time.Parse("2006-01-02", r.FormValue("fecha_inicio"))
	fechaCese, _ := time.Parse("2006-01-02", r.FormValue("fecha_cese"))
	motivo := r.FormValue("motivo")

	liq, err := h.CtsService.CalcularLiquidacion(contratoID, fechaInicio, fechaCese, motivo)
	if err != nil {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`
			<div id="alerta-liq" hx-swap-oob="true">
				<article style="background-color: #ffcdd2; color: #b71c1c; padding: 1rem; margin-bottom: 1rem; border-radius: 5px;">
					❌ Error de cálculo: ` + err.Error() + `
				</article>
			</div>
		`))
		return
	}

	vacTruncas, _ := strconv.ParseFloat(strings.TrimSpace(r.FormValue("vacaciones_truncas")), 64)
	gratiTrunca, _ := strconv.ParseFloat(strings.TrimSpace(r.FormValue("gratificacion_trunca")), 64)
	liq.MontoVacacionesTruncas = vacTruncas
	liq.MontoGratiTrunca = gratiTrunca
	liq.TotalLiquidacion = liq.MontoCts + vacTruncas + gratiTrunca
	liq.Estado = "APROBADO"

	err = h.CtsRepo.CrearLiquidacionCese(liq)
	if err != nil {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`
			<div id="alerta-liq" hx-swap-oob="true">
				<article style="background-color: #ffcdd2; color: #b71c1c; padding: 1rem; margin-bottom: 1rem; border-radius: 5px;">
					❌ Error al guardar liquidación: ` + err.Error() + `
				</article>
			</div>
		`))
		return
	}

	w.Write([]byte(`<div id="alerta-liq" hx-swap-oob="true"></div>`))
	w.Header().Set("HX-Trigger", "cerrarModalLiq")
	h.ListarLiquidacionesCese(w, r)
}

func (h *CtsHandler) EliminarLiquidacionCese(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	tenantID := obtenerTenantID(r)

	err := h.CtsRepo.EliminarLiquidacionCese(id, tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.ListarLiquidacionesCese(w, r)
}
