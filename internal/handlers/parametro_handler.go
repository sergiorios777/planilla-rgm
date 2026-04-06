package handlers

import (
	"html/template"
	"net/http"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
	"strconv"
	"strings"
)

type ParametroHandler struct {
	Repo *repository.ParametroRepository
}

func (h *ParametroHandler) VistaUI(w http.ResponseWriter, r *http.Request) {
	tmpl, _ := template.ParseFiles("ui/templates/admin/parametros_ui.html")
	tmpl.Execute(w, nil)
}

func (h *ParametroHandler) Listar(w http.ResponseWriter, r *http.Request) {
	parametros, _ := h.Repo.ObtenerTodos()
	tmpl, _ := template.ParseFiles("ui/templates/admin/parametros_ui.html")
	tmpl.ExecuteTemplate(w, "tabla_parametros", parametros)
}

func (h *ParametroHandler) Guardar(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	valor, _ := strconv.ParseFloat(r.FormValue("valor"), 64)
	clave := strings.ToUpper(strings.TrimSpace(r.FormValue("clave")))
	fechaDesde := r.FormValue("fecha_desde")
	fechaHastaStr := r.FormValue("fecha_hasta")

	var fechaHasta *string
	// Si el formulario envió una fecha, la usamos. Si está vacío, queda como nil (NULL en BD)
	if strings.TrimSpace(fechaHastaStr) != "" {
		fechaHasta = &fechaHastaStr
	}

	param := models.ParametroGlobal{
		Clave:       clave,
		Valor:       valor,
		FechaDesde:  fechaDesde,
		FechaHasta:  fechaHasta,
		Descripcion: r.FormValue("descripcion"),
	}

	h.Repo.Guardar(&param)
	h.Listar(w, r)
}
