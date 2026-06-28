package handlers

import (
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"planilla-rgm/internal/repository"
	"planilla-rgm/internal/services"
)

type LiquidacionHandler struct {
	LiquidacionRepo    *repository.LiquidacionRepository
	LiquidacionService *services.LiquidacionService
	ContratoRepo       *repository.ContratoRepository
}

func NewLiquidacionHandler(repo *repository.LiquidacionRepository, service *services.LiquidacionService, contratoRepo *repository.ContratoRepository) *LiquidacionHandler {
	return &LiquidacionHandler{
		LiquidacionRepo:    repo,
		LiquidacionService: service,
		ContratoRepo:       contratoRepo,
	}
}

func (h *LiquidacionHandler) LiquidacionesVistaUI(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("ui/templates/tenant/liquidaciones_ui.html")
	if err != nil {
		http.Error(w, "Error cargando plantilla: "+err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

func (h *LiquidacionHandler) ListarLiquidacionesCese(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	liquidaciones, err := h.LiquidacionRepo.ListarLiquidacionesCese(tenantID)
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

func (h *LiquidacionHandler) FormularioCrearUI(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	contratos, err := h.ContratoRepo.ObtenerTodos(tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFiles("ui/templates/tenant/liquidaciones_ui.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "formulario_crear_liq", contratos)
}

func (h *LiquidacionHandler) CalcularLiquidacionCese(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	contratoID, _ := strconv.Atoi(r.FormValue("contrato_id"))
	fechaInicio, _ := time.Parse("2006-01-02", r.FormValue("fecha_inicio"))
	fechaCese, _ := time.Parse("2006-01-02", r.FormValue("fecha_cese"))
	motivo := r.FormValue("motivo")
	periodosVencidos, _ := strconv.Atoi(r.FormValue("periodos_vencidos_vacaciones"))
	periodosNoVencidos, _ := strconv.Atoi(r.FormValue("periodos_no_vencidos_vacaciones"))

	liq, err := h.LiquidacionService.CalcularLiquidacion(contratoID, fechaInicio, fechaCese, motivo, periodosVencidos, periodosNoVencidos)
	if err != nil {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<p style="color:red; font-weight: bold; padding: 1rem; background-color: #ffcdd2; border-radius: 5px;">❌ Error de cálculo: ` + err.Error() + `</p>`))
		return
	}

	// Procesar checkboxes
	calcularCts := r.FormValue("calcular_cts") == "on"
	calcularGrati := r.FormValue("calcular_grati_trunca") == "on"
	calcularVac := r.FormValue("calcular_vacaciones") == "on"

	if !calcularCts {
		liq.MontoCts = 0
	}
	if !calcularGrati {
		liq.MontoGratiTrunca = 0
	}
	if !calcularVac {
		liq.MontoVacacionesTruncas = 0
		liq.MontoVacacionesNoGozadas = 0
		liq.MontoIndemnizacionVacacional = 0
	}

	// Permitir overrides manuales si se proveen
	if val := r.FormValue("vacaciones_truncas"); val != "" {
		if v, err := strconv.ParseFloat(strings.TrimSpace(val), 64); err == nil {
			liq.MontoVacacionesTruncas = v
		}
	}
	if val := r.FormValue("vacaciones_no_gozadas"); val != "" {
		if v, err := strconv.ParseFloat(strings.TrimSpace(val), 64); err == nil {
			liq.MontoVacacionesNoGozadas = v
		}
	}
	if val := r.FormValue("indemnizacion_vacaciones"); val != "" {
		if v, err := strconv.ParseFloat(strings.TrimSpace(val), 64); err == nil {
			liq.MontoIndemnizacionVacacional = v
		}
	}
	if val := r.FormValue("gratificacion_trunca"); val != "" {
		if g, err := strconv.ParseFloat(strings.TrimSpace(val), 64); err == nil {
			liq.MontoGratiTrunca = g
		}
	}

	liq.TotalLiquidacion = liq.MontoCts + liq.MontoVacacionesTruncas + liq.MontoVacacionesNoGozadas + liq.MontoIndemnizacionVacacional + liq.MontoGratiTrunca

	tmpl, err := template.ParseFiles("ui/templates/tenant/liquidaciones_ui.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "vista_previa_liquidacion", liq)
}

func (h *LiquidacionHandler) GuardarLiquidacionCese(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	contratoID, _ := strconv.Atoi(r.FormValue("contrato_id"))
	fechaInicio, _ := time.Parse("2006-01-02", r.FormValue("fecha_inicio"))
	fechaCese, _ := time.Parse("2006-01-02", r.FormValue("fecha_cese"))
	motivo := r.FormValue("motivo")
	periodosVencidos, _ := strconv.Atoi(r.FormValue("periodos_vencidos_vacaciones"))
	periodosNoVencidos, _ := strconv.Atoi(r.FormValue("periodos_no_vencidos_vacaciones"))

	liq, err := h.LiquidacionService.CalcularLiquidacion(contratoID, fechaInicio, fechaCese, motivo, periodosVencidos, periodosNoVencidos)
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

	// Procesar checkboxes
	calcularCts := r.FormValue("calcular_cts") == "on"
	calcularGrati := r.FormValue("calcular_grati_trunca") == "on"
	calcularVac := r.FormValue("calcular_vacaciones") == "on"

	if !calcularCts {
		liq.MontoCts = 0
	}
	if !calcularGrati {
		liq.MontoGratiTrunca = 0
	}
	if !calcularVac {
		liq.MontoVacacionesTruncas = 0
		liq.MontoVacacionesNoGozadas = 0
		liq.MontoIndemnizacionVacacional = 0
	}

	// Permitir overrides manuales si se proveen
	if val := r.FormValue("vacaciones_truncas"); val != "" {
		if v, err := strconv.ParseFloat(strings.TrimSpace(val), 64); err == nil {
			liq.MontoVacacionesTruncas = v
		}
	}
	if val := r.FormValue("vacaciones_no_gozadas"); val != "" {
		if v, err := strconv.ParseFloat(strings.TrimSpace(val), 64); err == nil {
			liq.MontoVacacionesNoGozadas = v
		}
	}
	if val := r.FormValue("indemnizacion_vacaciones"); val != "" {
		if v, err := strconv.ParseFloat(strings.TrimSpace(val), 64); err == nil {
			liq.MontoIndemnizacionVacacional = v
		}
	}
	if val := r.FormValue("gratificacion_trunca"); val != "" {
		if g, err := strconv.ParseFloat(strings.TrimSpace(val), 64); err == nil {
			liq.MontoGratiTrunca = g
		}
	}

	liq.TotalLiquidacion = liq.MontoCts + liq.MontoVacacionesTruncas + liq.MontoVacacionesNoGozadas + liq.MontoIndemnizacionVacacional + liq.MontoGratiTrunca
	liq.Estado = "APROBADO"

	err = h.LiquidacionRepo.CrearLiquidacionCese(liq)
	if err != nil {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`
			<div id="alerta-liq" hx-swap-oob="true">
				<article style="background-color: #ffcdd2; color: #b71c1c; padding: 1rem; margin-bottom: 1rem; border-radius: 5px;">
					❌ Error de persistencia: ` + err.Error() + `
				</article>
			</div>
		`))
		return
	}

	w.Header().Set("HX-Trigger", "cerrarModalLiq")
	h.ListarLiquidacionesCese(w, r)
}

func (h *LiquidacionHandler) EliminarLiquidacionCese(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))

	err := h.LiquidacionRepo.EliminarLiquidacionCese(id, tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.ListarLiquidacionesCese(w, r)
}
