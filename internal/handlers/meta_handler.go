package handlers

import (
	"html/template"
	"net/http"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
	"strconv"
)

type MetaHandler struct {
	Repo *repository.MetaRepository
}

func (h *MetaHandler) VistaUI(w http.ResponseWriter, r *http.Request) {
	tmpl, _ := template.ParseFiles("ui/templates/tenant/metas_ui.html")
	tmpl.Execute(w, nil)
}

func (h *MetaHandler) Listar(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r) // Usamos el helper que ya existe en este paquete
	metas, _ := h.Repo.ObtenerTodos(tenantID)

	tmpl, _ := template.ParseFiles("ui/templates/tenant/metas_ui.html")
	tmpl.ExecuteTemplate(w, "tabla_metas", metas)
}

func (h *MetaHandler) Crear(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	anio, _ := strconv.Atoi(r.FormValue("anio"))

	nuevaMeta := models.MetaPresupuestal{
		TenantID:    obtenerTenantID(r),
		Anio:        anio,
		Codigo:      r.FormValue("codigo"),
		Descripcion: r.FormValue("descripcion"),
		Activo:      r.FormValue("activo") == "on",
	}

	h.Repo.Crear(&nuevaMeta)
	h.Listar(w, r)
}

// FormularioCrearUI devuelve el formulario limpio
func (h *MetaHandler) FormularioCrearUI(w http.ResponseWriter, r *http.Request) {
	tmpl, _ := template.ParseFiles("ui/templates/tenant/metas_ui.html") // Ajusta el nombre si tu HTML se llama diferente
	tmpl.ExecuteTemplate(w, "formulario_crear", nil)
}

// EditarUI carga el formulario precargado
func (h *MetaHandler) EditarUI(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	tenantID := obtenerTenantID(r)

	meta, _ := h.Repo.ObtenerPorID(id, tenantID)

	tmpl, _ := template.ParseFiles("ui/templates/tenant/metas_ui.html")
	tmpl.ExecuteTemplate(w, "formulario_editar", meta)
}

// Actualizar guarda cambios y recarga la tabla
func (h *MetaHandler) Actualizar(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id, _ := strconv.Atoi(r.FormValue("id"))

	metaActualizada := models.MetaPresupuestal{
		ID:          id,
		TenantID:    obtenerTenantID(r),
		Codigo:      r.FormValue("codigo"),
		Descripcion: r.FormValue("descripcion"),
		Activo:      r.FormValue("activo") == "on",
	}

	h.Repo.Actualizar(&metaActualizada)

	// Pedimos a HTMX que recargue la tabla
	w.Header().Set("HX-Trigger", "recargarTablaMetas")

	// Volvemos a pintar el formulario de creación
	h.FormularioCrearUI(w, r)
}
