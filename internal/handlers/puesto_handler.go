package handlers

import (
	"html/template"
	"log"
	"net/http"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
	"planilla-rgm/internal/services"
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

	servicioPuesto := services.PuestoService{Repo: h.Repo}
	err := servicioPuesto.CrearPuestoConPlantilla(&nuevoPuesto)
	if err != nil {
		log.Println("Error creando puesto con plantilla:", err)
	}
	h.Listar(w, r)
}

// Editar prepara los datos del puesto y las listas para el formulario de edición
func (h *PuestoHandler) Editar(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	tenantID := obtenerTenantID(r)

	// 1. Buscamos el puesto actual
	puesto, _ := h.Repo.ObtenerPorID(id, tenantID)

	// 2. Necesitamos las listas para los combos (Selects)
	metas, _ := h.MetaRepo.ObtenerTodos(tenantID)
	fuentes, _ := h.FuenteRubroRepo.ObtenerPorAnio(2026)
	regimenes, _ := h.Repo.ObtenerRegimenes()

	datos := map[string]interface{}{
		"Puesto":    puesto,
		"Metas":     metas,
		"Fuentes":   fuentes,
		"Regimenes": regimenes,
	}

	// 💡 ENVIAMOS SOLO EL FRAGMENTO: "formulario_editar"
	tmpl, _ := template.ParseFiles("ui/templates/tenant/puestos_ui.html")
	tmpl.ExecuteTemplate(w, "formulario_editar", datos)
}

// Actualizar procesa los cambios y refresca la lista
func (h *PuestoHandler) Actualizar(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id, _ := strconv.Atoi(r.FormValue("id"))

	metaID, _ := strconv.Atoi(r.FormValue("meta_id"))
	fuenteID, _ := strconv.Atoi(r.FormValue("fuente_rubro_id"))
	regimenID, _ := strconv.Atoi(r.FormValue("regimen_id"))
	sueldo, _ := strconv.ParseFloat(r.FormValue("sueldo_presupuestado"), 64)

	puestoActualizado := models.Puesto{
		ID:                  id,
		TenantID:            obtenerTenantID(r),
		MetaID:              metaID,
		FuenteRubroID:       fuenteID,
		RegimenID:           regimenID,
		Nombre:              r.FormValue("nombre"),
		SueldoPresupuestado: sueldo,
		Activo:              r.FormValue("activo") == "on",
	}

	h.Repo.Actualizar(&puestoActualizado)

	// Tras actualizar, mostramos el formulario de "Crear" nuevamente
	h.VistaUI(w, r)
}

// FormularioCrearUI devuelve el formulario limpio
func (h *PuestoHandler) FormularioCrearUI(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)

	metas, _ := h.MetaRepo.ObtenerTodos(tenantID)
	fuentes, _ := h.FuenteRubroRepo.ObtenerPorAnio(2026)
	regimenes, _ := h.Repo.ObtenerRegimenes()

	datos := map[string]interface{}{
		"Metas":     metas,
		"Fuentes":   fuentes,
		"Regimenes": regimenes,
	}

	tmpl, _ := template.ParseFiles("ui/templates/tenant/puestos_ui.html")
	tmpl.ExecuteTemplate(w, "formulario_crear", datos)
}
