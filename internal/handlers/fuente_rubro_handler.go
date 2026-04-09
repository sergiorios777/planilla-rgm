package handlers

import (
	"html/template"
	"net/http"
	"planilla-rgm/internal/repository"
	"strconv"
)

type FuenteRubroHandler struct {
	Repo *repository.FuenteRubroRepository
}

func (h *FuenteRubroHandler) VistaUI(w http.ResponseWriter, r *http.Request) {
	tmpl, _ := template.ParseFiles("ui/templates/admin/fuentes_rubros_ui.html")
	tmpl.Execute(w, nil)
}

func (h *FuenteRubroHandler) Listar(w http.ResponseWriter, r *http.Request) {
	// Por defecto usamos 2026, pero podríamos leerlo de un select en el futuro
	anio := 2026

	// Si el usuario envía un año en el query (?anio=2027), lo usamos
	if a, err := strconv.Atoi(r.URL.Query().Get("anio")); err == nil {
		anio = a
	}

	datos, _ := h.Repo.ObtenerPorAnio(anio)

	tmpl, _ := template.ParseFiles("ui/templates/admin/fuentes_rubros_ui.html")
	tmpl.ExecuteTemplate(w, "tabla_fuentes", datos)
}
