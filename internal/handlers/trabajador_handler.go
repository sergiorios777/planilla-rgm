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
	// Extraemos la lista de AFPs de la BD para poblar el <select> de Crear
	afps, _ := h.Repo.ObtenerAFPsActivas()
	tmpl, _ := template.ParseFiles("ui/templates/tenant/trabajadores_ui.html")

	// Pasamos la lista a la vista principal
	tmpl.Execute(w, map[string]interface{}{
		"ListaAFPs": afps,
	})
}

func (h *TrabajadorHandler) Listar(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	trabajadores, _ := h.Repo.ObtenerTodos(tenantID)

	tmpl, _ := template.ParseFiles("ui/templates/tenant/trabajadores_ui.html")
	tmpl.ExecuteTemplate(w, "tabla_trabajadores", trabajadores)
}

func (h *TrabajadorHandler) Crear(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	afpID, _ := strconv.Atoi(r.FormValue("afp_id"))

	nuevoTrabajador := models.Trabajador{
		TenantID:           obtenerTenantID(r),
		TipoDocumento:      r.FormValue("tipo_documento"),
		NumeroDocumento:    r.FormValue("numero_documento"),
		Nombres:            r.FormValue("nombres"),
		ApellidoPaterno:    r.FormValue("apellido_paterno"),
		ApellidoMaterno:    r.FormValue("apellido_materno"),
		FechaNacimiento:    r.FormValue("fecha_nacimiento"),
		Sexo:               r.FormValue("sexo"),
		Activo:             r.FormValue("activo") == "on",
		RegimenPensionario: r.FormValue("regimen_pensionario"),
		AfpID:              afpID,
		AfpTipoComision:    r.FormValue("afp_tipo_comision"),
		Cuspp:              r.FormValue("cuspp"),
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

	// Extraemos las AFPs para el desplegable de edición
	afps, _ := h.Repo.ObtenerAFPsActivas()

	// Creamos una estructura al vuelo para pasar el trabajador Y la lista al template
	data := struct {
		*models.Trabajador
		ListaAFPs map[int]string
	}{
		Trabajador: trabajador,
		ListaAFPs:  afps,
	}

	tmpl, _ := template.ParseFiles("ui/templates/tenant/trabajadores_ui.html")
	tmpl.ExecuteTemplate(w, "formulario_editar", data)
}

func (h *TrabajadorHandler) Actualizar(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	id, _ := strconv.Atoi(r.FormValue("id"))
	afpID, _ := strconv.Atoi(r.FormValue("afp_id"))

	trabajadorEditado := models.Trabajador{
		ID:                 id,
		TenantID:           obtenerTenantID(r),
		TipoDocumento:      r.FormValue("tipo_documento"),
		NumeroDocumento:    r.FormValue("numero_documento"),
		Nombres:            r.FormValue("nombres"),
		ApellidoPaterno:    r.FormValue("apellido_paterno"),
		ApellidoMaterno:    r.FormValue("apellido_materno"),
		FechaNacimiento:    r.FormValue("fecha_nacimiento"),
		Sexo:               r.FormValue("sexo"),
		Activo:             r.FormValue("activo") == "on",
		RegimenPensionario: r.FormValue("regimen_pensionario"),
		AfpID:              afpID,
		AfpTipoComision:    r.FormValue("afp_tipo_comision"),
		Cuspp:              r.FormValue("cuspp"),
	}

	h.Repo.Actualizar(&trabajadorEditado)
	h.VistaUI(w, r)
}
