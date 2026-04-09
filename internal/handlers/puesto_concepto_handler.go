package handlers

import (
	"html/template"
	"net/http"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
	"strconv"
	"strings"
)

type PuestoConceptoHandler struct {
	Repo       *repository.PuestoConceptoRepository
	PuestoRepo *repository.PuestoRepository // Para traer el nombre del puesto
}

// VistaUI carga la pantalla completa de configuración para un puesto específico
func (h *PuestoConceptoHandler) VistaUI(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	puestoID, _ := strconv.Atoi(r.URL.Query().Get("puesto_id"))

	// Traemos la info de los Puestos (reutilizamos ObtenerTodos y filtramos)
	// En un escenario ideal, tendrías un ObtenerPorID en PuestoRepository.
	// Por ahora lo simplificamos enviando solo el ID.

	disponibles, _ := h.Repo.ObtenerDisponibles(puestoID, tenantID)

	datos := map[string]interface{}{
		"PuestoID":    puestoID,
		"Disponibles": disponibles,
	}

	tmpl, _ := template.ParseFiles("ui/templates/tenant/puestos_conceptos_ui.html")
	tmpl.Execute(w, datos)
}

// Listar devuelve solo el fragmento de la tabla actualizada
func (h *PuestoConceptoHandler) Listar(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	puestoID, _ := strconv.Atoi(r.URL.Query().Get("puesto_id"))

	asignados, _ := h.Repo.ObtenerAsignados(puestoID, tenantID)

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
