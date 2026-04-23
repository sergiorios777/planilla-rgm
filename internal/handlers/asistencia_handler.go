package handlers

import (
	"html/template"
	"net/http"
	"planilla-rgm/internal/repository"
	"strconv"
)

type AsistenciaHandler struct {
	Repo *repository.AsistenciaRepository
}

func (h *AsistenciaHandler) VistaUI(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r) // Usamos tu helper de sesión

	// Traemos los contratos para llenar el desplegable de búsqueda
	contratos, _ := h.Repo.ObtenerContratosParaSelect(tenantID)

	tmpl, _ := template.ParseFiles("ui/templates/tenant/asistencia_ui.html")
	tmpl.Execute(w, map[string]interface{}{
		"Contratos": contratos,
	})
}

func (h *AsistenciaHandler) Listar(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	ocurrencias, _ := h.Repo.ListarHistorial(tenantID)

	tmpl, _ := template.ParseFiles("ui/templates/tenant/asistencia_ui.html")
	tmpl.ExecuteTemplate(w, "tabla_ocurrencias", ocurrencias)
}

func (h *AsistenciaHandler) Crear(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	contratoID, _ := strconv.Atoi(r.FormValue("contrato_id"))
	cantidad, _ := strconv.ParseFloat(r.FormValue("cantidad"), 64)
	tipo := r.FormValue("tipo")
	fecha := r.FormValue("fecha_ocurrencia")

	// Guardamos en la base de datos
	h.Repo.Crear(contratoID, tipo, fecha, cantidad)

	// Devolvemos la tabla actualizada
	h.Listar(w, r)
}
