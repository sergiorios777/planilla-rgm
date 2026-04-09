package handlers

import (
	"html/template"
	"net/http"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
	"strconv"
	"strings"
)

type PlanillaHandler struct {
	Repo *repository.PlanillaRepository
}

func (h *PlanillaHandler) VistaUI(w http.ResponseWriter, r *http.Request) {
	tmpl, _ := template.ParseFiles("ui/templates/tenant/planillas_ui.html")
	tmpl.Execute(w, nil)
}

func (h *PlanillaHandler) Listar(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	planillas, _ := h.Repo.ObtenerTodos(tenantID)

	tmpl, _ := template.ParseFiles("ui/templates/tenant/planillas_ui.html")
	tmpl.ExecuteTemplate(w, "tabla_planillas", planillas)
}

func (h *PlanillaHandler) Crear(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	tenantID := obtenerTenantID(r)

	anio, _ := strconv.Atoi(r.FormValue("anio"))
	mes, _ := strconv.Atoi(r.FormValue("mes"))

	nuevaPlanilla := models.Planilla{
		TenantID:    tenantID,
		Anio:        anio,
		Mes:         mes,
		Descripcion: r.FormValue("descripcion"),
	}

	err := h.Repo.Crear(&nuevaPlanilla)
	if err != nil {
		// Validamos la restricción UNIQUE que creamos en la migración
		if strings.Contains(err.Error(), "unique_planilla_mes") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`
				<div id="alerta-planilla" hx-swap-oob="true">
					<article style="background-color: #ffcdd2; color: #b71c1c; padding: 1rem; margin-bottom: 1rem; border-radius: 5px;">
						❌ Error: Ya existe una planilla con esa misma descripción para este mes y año.
					</article>
				</div>
			`))
			return
		}
		http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
		return
	}

	// Limpiamos alertas previas y actualizamos la tabla
	w.Write([]byte(`<div id="alerta-planilla" hx-swap-oob="true"></div>`))
	h.Listar(w, r)
}

// Procesar ejecuta el motor de cálculo y redirige a la vista de detalle
func (h *PlanillaHandler) Procesar(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	planillaID, _ := strconv.Atoi(r.URL.Query().Get("id"))

	err := h.Repo.ProcesarPlanilla(planillaID, tenantID)
	if err != nil {
		// Si el motor falla, mostramos una alerta sin salir de la página
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`
			<div id="alerta-planilla" hx-swap-oob="true">
				<article style="background-color: #ffcdd2; color: #b71c1c; padding: 1rem; margin-bottom: 1rem; border-radius: 5px;">
					❌ Error en el Motor de Cálculo: ` + err.Error() + `
				</article>
			</div>
		`))
		return
	}

	// NUEVO: En lugar de redirigir, empujamos la nueva URL al navegador
	// y cargamos la vista directamente en el mismo contenedor
	w.Header().Set("HX-Push-Url", "/tenant/planillas/detalle/ui?id="+strconv.Itoa(planillaID))

	// Limpiamos alertas si las hubo y procesamos la vista
	w.Write([]byte(`<div id="alerta-planilla" hx-swap-oob="true"></div>`))

	// Llamamos a la vista de detalle
	r.URL.RawQuery = "id=" + strconv.Itoa(planillaID)
	h.VistaDetalle(w, r)
}

// VistaDetalle carga la pantalla con las boletas generadas
func (h *PlanillaHandler) VistaDetalle(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	planillaID, _ := strconv.Atoi(r.URL.Query().Get("id"))

	detalles, _ := h.Repo.ObtenerDetalles(planillaID, tenantID)

	datos := map[string]interface{}{
		"PlanillaID": planillaID,
		"Detalles":   detalles,
	}

	tmpl, _ := template.ParseFiles("ui/templates/tenant/planilla_detalle_ui.html")
	tmpl.Execute(w, datos)
}
