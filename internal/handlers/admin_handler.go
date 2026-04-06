package handlers

import (
	"html/template"
	"net/http"

	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
)

// AdminHandler agrupa las funciones web del panel del Super Admin
type AdminHandler struct {
	Repo *repository.TenantRepository
}

// ListarInquilinos obtiene los datos y los inyecta en una plantilla HTML
func (h *AdminHandler) ListarInquilinos(w http.ResponseWriter, r *http.Request) {
	// 1. Pedimos los datos al repositorio
	tenants, err := h.Repo.ObtenerTodos()
	if err != nil {
		http.Error(w, "Error al obtener la lista de inquilinos de la base de datos", http.StatusInternalServerError)
		return
	}

	// 2. Cargamos el archivo HTML que diseñaremos en el siguiente paso
	tmpl, err := template.ParseFiles("ui/templates/admin/tenants.html")
	if err != nil {
		http.Error(w, "Error cargando la vista HTML", http.StatusInternalServerError)
		return
	}

	// 3. Renderizamos el HTML enviándole la lista de inquilinos
	tmpl.Execute(w, tenants)
}

// CrearInquilino recibe los datos del formulario HTMX y guarda la entidad
func (h *AdminHandler) CrearInquilino(w http.ResponseWriter, r *http.Request) {
	// 1. Leemos los datos que vienen del formulario HTML
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error al leer el formulario", http.StatusBadRequest)
		return
	}

	nombre := r.FormValue("nombre")
	ruc := r.FormValue("ruc")
	// En HTML, un checkbox marcado envía el valor "on"
	activo := r.FormValue("activo") == "on"

	// 2. Preparamos nuestro modelo de Go
	nuevoTenant := models.Tenant{
		Nombre: nombre,
		Ruc:    ruc,
		Activo: activo,
	}

	// 3. Le pedimos al repositorio que lo guarde en PostgreSQL
	err = h.Repo.Crear(&nuevoTenant)
	if err != nil {
		http.Error(w, "Error al guardar en la base de datos", http.StatusInternalServerError)
		return
	}

	// 4. TRUCO DE HTMX: En lugar de redirigir, simplemente volvemos a llamar
	// a la función ListarInquilinos para que devuelva la tabla HTML actualizada.
	h.ListarInquilinos(w, r)
}

// VistaUI devuelve la estructura HTML de la página de inquilinos
func (h *AdminHandler) VistaUI(w http.ResponseWriter, r *http.Request) {
	tmpl, _ := template.ParseFiles("ui/templates/admin/inquilinos_ui.html")
	tmpl.Execute(w, nil)
}
