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

	busqueda := r.URL.Query().Get("buscar")
	datos, err := h.Repo.ObtenerPorAnio(anio, busqueda)
	if err != nil {
		http.Error(w, "Error obteniendo fuentes y rubros", http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFiles("ui/templates/admin/fuentes_rubros_ui.html")
	if err != nil {
		http.Error(w, "Error cargando la vista HTML", http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Target") == "true" {
		tmpl.ExecuteTemplate(w, "filas_fuentes", datos)
		return
	}

	tmpl.ExecuteTemplate(w, "tabla_fuentes", datos)
}
