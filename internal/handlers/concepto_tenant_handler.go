package handlers

import (
	"html/template"
	"net/http"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
	"strconv"
	"strings"
)

type ConceptoTenantHandler struct {
	Repo *repository.ConceptoTenantRepository
}

func (h *ConceptoTenantHandler) VistaUI(w http.ResponseWriter, r *http.Request) {
	maestros, _ := h.Repo.ObtenerMaestros()
	clasificadores, _ := h.Repo.ObtenerClasificadores() // NUEVO

	datos := map[string]interface{}{
		"Maestros":       maestros,
		"Clasificadores": clasificadores,
	}

	tmpl, _ := template.ParseFiles("ui/templates/tenant/conceptos_tenant_ui.html")
	tmpl.Execute(w, datos)
}

func (h *ConceptoTenantHandler) Listar(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	conceptos, _ := h.Repo.ObtenerTodos(tenantID)

	tmpl, _ := template.ParseFiles("ui/templates/tenant/conceptos_tenant_ui.html")
	tmpl.ExecuteTemplate(w, "tabla_conceptos_tenant", conceptos)
}

func (h *ConceptoTenantHandler) Crear(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	tenantID := obtenerTenantID(r)
	cID, _ := strconv.Atoi(r.FormValue("concepto_id"))

	clasifStr := r.FormValue("clasificador_id")
	var clasifID *int
	if strings.TrimSpace(clasifStr) != "" {
		idParsed, err := strconv.Atoi(clasifStr)
		if err == nil {
			clasifID = &idParsed
		}
	}

	nuevoConcepto := models.ConceptoTenant{
		TenantID:            tenantID,
		ConceptoID:          cID,
		NombrePersonalizado: r.FormValue("nombre_personalizado"),
		FrecuenciaMeses:     r.FormValue("frecuencia_meses"),
		ClasificadorID:      clasifID,
		Activo:              r.FormValue("activo") == "on",
		EsExtraordinario:    r.FormValue("es_extraordinario") == "on",
	}

	err := h.Repo.Crear(&nuevoConcepto)
	if err != nil {
		// Validamos si el error es por la restricción UNIQUE de la base de datos
		if strings.Contains(err.Error(), "unique_nombre_concepto_tenant") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`
				<div id="alerta-concepto" hx-swap-oob="true">
					<article style="background-color: #ffcdd2; color: #b71c1c; padding: 1rem; margin-bottom: 1rem; border-radius: 5px;">
						❌ Error: Ya existe un concepto con ese "Nombre Personalizado". Por favor, usa un nombre distinto (Ej. "Sueldo Base CAS" en lugar de "Sueldo Base").
					</article>
				</div>
			`))
			return
		}
		http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
		return
	}

	// Limpiamos alertas previas y actualizamos la tabla
	w.Write([]byte(`<div id="alerta-concepto" hx-swap-oob="true"></div>`))
	h.Listar(w, r)
}
