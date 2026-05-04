package handlers

import (
	"html/template"
	"log"
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
	busqueda := r.URL.Query().Get("buscar")
	parametros, err := h.Repo.ObtenerTodos(busqueda)
	if err != nil {
		log.Println("Error crítico leyendo la base de datos:", err)
	}

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

// Función EditarUI devuelve el formulario precargado con los datos actuales del parámetro
func (h *ParametroHandler) EditarUI(w http.ResponseWriter, r *http.Request) {
	// 1. Obtenemos el ID de la URL (Ej: /admin/params/edit?id=5)
	idStr := r.URL.Query().Get("id")
	id, _ := strconv.Atoi(idStr)

	// 2. Buscamos el parámetro en la base de datos
	param, err := h.Repo.ObtenerPorID(id)
	if err != nil {
		http.Error(w, "Parámetro no encontrado", http.StatusNotFound)
		return
	}

	// 3. Parseamos la vista y pasamos el objeto 'param' al HTML
	tmpl, _ := template.ParseFiles("ui/templates/admin/parametros_ui.html")

	// Usamos ExecuteTemplate para editar un bloque específico dentro del HTML
	// Asumiendo que tu HTML tiene un template llamado "formulario_editar"
	tmpl.ExecuteTemplate(w, "formulario_editar", param)
}

// Función ActualizarParametro procesa el formulario de edición
func (h *ParametroHandler) ActualizarParametro(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	valor, _ := strconv.ParseFloat(r.FormValue("valor"), 64)
	clave := strings.ToUpper(strings.TrimSpace(r.FormValue("clave")))
	fechaDesde := r.FormValue("fecha_desde")
	fechaHastaStr := r.FormValue("fecha_hasta")

	// ID del parámetro que estamos editando
	id, _ := strconv.Atoi(r.FormValue("id"))

	var fechaHasta *string
	if strings.TrimSpace(fechaHastaStr) != "" {
		fechaHasta = &fechaHastaStr
	}

	param := models.ParametroGlobal{
		ID:          id,
		Clave:       clave,
		Valor:       valor,
		FechaDesde:  fechaDesde,
		FechaHasta:  fechaHasta,
		Descripcion: r.FormValue("descripcion"),
	}

	h.Repo.Actualizar(&param)

	// Recargamos la vista completa para mostrar el valor actualizado
	h.VistaUI(w, r)
}
