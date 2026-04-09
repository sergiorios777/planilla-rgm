package handlers

import (
	"html/template"
	"net/http"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
	"strconv"
)

type PuestoHandler struct {
	Repo            *repository.PuestoRepository
	MetaRepo        *repository.MetaRepository
	FuenteRubroRepo *repository.FuenteRubroRepository
}

func (h *PuestoHandler) VistaUI(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)

	// Preparamos listas para los combos
	metas, _ := h.MetaRepo.ObtenerTodos(tenantID)
	fuentes, _ := h.FuenteRubroRepo.ObtenerPorAnio(2026) // Anio fijo por ahora
	regimenes, _ := h.Repo.ObtenerRegimenes()

	datos := map[string]interface{}{
		"Metas":     metas,
		"Fuentes":   fuentes,
		"Regimenes": regimenes,
	}

	tmpl, _ := template.ParseFiles("ui/templates/tenant/puestos_ui.html")
	tmpl.Execute(w, datos)
}

func (h *PuestoHandler) Listar(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	puestos, _ := h.Repo.ObtenerTodos(tenantID)

	tmpl, _ := template.ParseFiles("ui/templates/tenant/puestos_ui.html")
	tmpl.ExecuteTemplate(w, "tabla_puestos", puestos)
}

func (h *PuestoHandler) Crear(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	metaID, _ := strconv.Atoi(r.FormValue("meta_id"))
	fuenteID, _ := strconv.Atoi(r.FormValue("fuente_rubro_id"))
	regimenID, _ := strconv.Atoi(r.FormValue("regimen_id"))
	sueldo, _ := strconv.ParseFloat(r.FormValue("sueldo_presupuestado"), 64)

	nuevoPuesto := models.Puesto{
		TenantID:            obtenerTenantID(r),
		MetaID:              metaID,
		FuenteRubroID:       fuenteID,
		RegimenID:           regimenID,
		Nombre:              r.FormValue("nombre"),
		SueldoPresupuestado: sueldo,
		Activo:              r.FormValue("activo") == "on",
	}

	h.Repo.Crear(&nuevoPuesto)
	h.Listar(w, r)
}
