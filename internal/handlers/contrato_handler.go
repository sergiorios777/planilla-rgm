package handlers

import (
	"html/template"
	"net/http"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
	"strconv"
	"strings"
)

type ContratoHandler struct {
	Repo           *repository.ContratoRepository
	TrabajadorRepo *repository.TrabajadorRepository // Lo necesitamos para el select
	PuestoRepo     *repository.PuestoRepository
}

func (h *ContratoHandler) VistaUI(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)

	trabajadores, _ := h.TrabajadorRepo.ObtenerTodos(tenantID)
	puestos, _ := h.PuestoRepo.ObtenerVacantes(tenantID) // NUEVO

	datos := map[string]interface{}{
		"Trabajadores": trabajadores,
		"Puestos":      puestos,
	}

	tmpl, _ := template.ParseFiles("ui/templates/tenant/contratos_ui.html")
	tmpl.Execute(w, datos)
}

func (h *ContratoHandler) Listar(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	contratos, _ := h.Repo.ObtenerTodos(tenantID)

	tmpl, _ := template.ParseFiles("ui/templates/tenant/contratos_ui.html")
	tmpl.ExecuteTemplate(w, "tabla_contratos", contratos)
}

func (h *ContratoHandler) Crear(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	tenantID := obtenerTenantID(r)
	tID, _ := strconv.Atoi(r.FormValue("trabajador_id"))
	pID, _ := strconv.Atoi(r.FormValue("puesto_id"))

	// === 1. NUEVA VALIDACIÓN DE NEGOCIO ===
	tieneActivo, err := h.Repo.TieneContratoActivo(tID, tenantID)
	if err != nil {
		http.Error(w, "Error validando el estado del trabajador", http.StatusInternalServerError)
		return
	}

	if tieneActivo {
		// TRUCO HTMX: Devolvemos un fragmento HTML con la etiqueta hx-swap-oob="true".
		// HTMX buscará el div con id="alerta-contrato" y le inyectará este error, sin tocar la tabla.
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`
			<div id="alerta-contrato" hx-swap-oob="true">
				<article style="background-color: #ffcdd2; color: #b71c1c; padding: 1rem; margin-bottom: 1rem; border-radius: 5px; font-weight: bold;">
					❌ Error: El trabajador seleccionado ya posee un contrato activo (Plaza ocupada). Debe finalizarlo antes de asignarle uno nuevo.
				</article>
			</div>
		`))
		return
	}
	// =======================================

	fFinStr := r.FormValue("fecha_fin")
	var fFin *string
	if strings.TrimSpace(fFinStr) != "" {
		fFin = &fFinStr
	}

	nuevoContrato := models.Contrato{
		TenantID:     tenantID,
		TrabajadorID: tID,
		PuestoID:     pID,
		FechaInicio:  r.FormValue("fecha_inicio"),
		FechaFin:     fFin,
		Activo:       r.FormValue("activo") == "on",
	}

	h.Repo.Crear(&nuevoContrato)

	// Si el contrato se crea con éxito, enviamos una orden OOB para "limpiar" cualquier alerta anterior
	w.Write([]byte(`<div id="alerta-contrato" hx-swap-oob="true"></div>`))

	// Finalmente, devolvemos la tabla actualizada como siempre
	h.Listar(w, r)
}
