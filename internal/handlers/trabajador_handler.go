package handlers

import (
	"html/template"
	"net/http"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
	"strconv"
)

type TrabajadorHandler struct {
	Repo *repository.TrabajadorRepository
}

// obtenerTenantID es un helper para sacar el ID de forma segura de la sesión
func obtenerTenantID(r *http.Request) int {
	// El JWT parsea los números como float64, lo convertimos a int
	if val, ok := r.Context().Value("tenant_id").(float64); ok {
		return int(val)
	}
	return 0 // En un caso real, si es 0 deberíamos bloquear la petición
}

func (h *TrabajadorHandler) VistaUI(w http.ResponseWriter, r *http.Request) {
	tmpl, _ := template.ParseFiles("ui/templates/tenant/trabajadores_ui.html")
	tmpl.Execute(w, nil)
}

func (h *TrabajadorHandler) Listar(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	trabajadores, _ := h.Repo.ObtenerTodos(tenantID)

	tmpl, _ := template.ParseFiles("ui/templates/tenant/trabajadores_ui.html")
	tmpl.ExecuteTemplate(w, "tabla_trabajadores", trabajadores)
}

func (h *TrabajadorHandler) Crear(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	nuevoTrabajador := models.Trabajador{
		TenantID:        obtenerTenantID(r), // INYECCIÓN SEGURA DEL BACKEND
		TipoDocumento:   r.FormValue("tipo_documento"),
		NumeroDocumento: r.FormValue("numero_documento"),
		Nombres:         r.FormValue("nombres"),
		ApellidoPaterno: r.FormValue("apellido_paterno"),
		ApellidoMaterno: r.FormValue("apellido_materno"),
		FechaNacimiento: r.FormValue("fecha_nacimiento"),
		Sexo:            r.FormValue("sexo"),
		Activo:          r.FormValue("activo") == "on",
	}

	h.Repo.Crear(&nuevoTrabajador)
	h.Listar(w, r)
}

func (h *TrabajadorHandler) EditarUI(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	tenantID := obtenerTenantID(r)

	trabajador, err := h.Repo.ObtenerPorID(id, tenantID)
	if err != nil {
		http.Error(w, "Trabajador no encontrado", http.StatusNotFound)
		return
	}

	tmpl, _ := template.ParseFiles("ui/templates/tenant/trabajadores_ui.html")
	tmpl.ExecuteTemplate(w, "formulario_editar", trabajador)
}

func (h *TrabajadorHandler) Actualizar(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	id, _ := strconv.Atoi(r.FormValue("id"))

	trabajadorEditado := models.Trabajador{
		ID:              id,
		TenantID:        obtenerTenantID(r), // INYECCIÓN SEGURA DEL BACKEND
		TipoDocumento:   r.FormValue("tipo_documento"),
		NumeroDocumento: r.FormValue("numero_documento"),
		Nombres:         r.FormValue("nombres"),
		ApellidoPaterno: r.FormValue("apellido_paterno"),
		ApellidoMaterno: r.FormValue("apellido_materno"),
		FechaNacimiento: r.FormValue("fecha_nacimiento"),
		Sexo:            r.FormValue("sexo"),
		Activo:          r.FormValue("activo") == "on",
	}

	h.Repo.Actualizar(&trabajadorEditado)
	h.VistaUI(w, r)
}
