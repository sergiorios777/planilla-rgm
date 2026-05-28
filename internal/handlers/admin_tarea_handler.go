package handlers

import (
	"html/template"
	"net/http"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
	"strconv"
	"strings"
	"time"
)

// AdminTareaHandler maneja las peticiones web para el CRUD de tareas programadas del Super Admin
type AdminTareaHandler struct {
	Repo *repository.AdminTareaRepository
}

// NewAdminTareaHandler crea una nueva instancia de AdminTareaHandler
func NewAdminTareaHandler(repo *repository.AdminTareaRepository) *AdminTareaHandler {
	return &AdminTareaHandler{Repo: repo}
}

// VistaUI renderiza el contenedor principal de la vista de tareas
func (h *AdminTareaHandler) VistaUI(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("ui/templates/admin/tareas_ui.html")
	if err != nil {
		http.Error(w, "Error cargando la vista: "+err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

// Listar devuelve el fragmento HTML con la tabla de tareas según búsqueda
func (h *AdminTareaHandler) Listar(w http.ResponseWriter, r *http.Request) {
	buscar := r.URL.Query().Get("buscar")
	tareas, err := h.Repo.ObtenerTodos(buscar)
	if err != nil {
		http.Error(w, "Error al obtener tareas: "+err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFiles("ui/templates/admin/tareas_ui.html")
	if err != nil {
		http.Error(w, "Error cargando la plantilla: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Si lo solicita HTMX para refrescar el cuerpo de la tabla
	if r.Header.Get("HX-Request") == "true" && r.Header.Get("HX-Target") == "lista-tareas-cuerpo" {
		tmpl.ExecuteTemplate(w, "filas_tareas", tareas)
		return
	}

	tmpl.ExecuteTemplate(w, "tabla_tareas", tareas)
}

// Crear procesa el formulario para guardar una nueva tarea programada
func (h *AdminTareaHandler) Crear(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error al leer formulario", http.StatusBadRequest)
		return
	}

	titulo := strings.TrimSpace(r.FormValue("titulo"))
	descripcion := strings.TrimSpace(r.FormValue("descripcion"))
	recurrencia := r.FormValue("recurrencia")
	fechaVencStr := r.FormValue("fecha_vencimiento")
	proximoAvisoStr := r.FormValue("proximo_aviso")
	activo := r.FormValue("activo") == "on"

	if titulo == "" || recurrencia == "" || fechaVencStr == "" || proximoAvisoStr == "" {
		http.Error(w, "Los campos Título, Recurrencia, Vencimiento y Próximo Aviso son obligatorios", http.StatusBadRequest)
		return
	}

	// Parsear fechas de datetime-local (layout "2006-01-02T15:04")
	fechaVenc, err := time.Parse("2006-01-02T15:04", fechaVencStr)
	if err != nil {
		http.Error(w, "Formato de fecha de vencimiento inválido", http.StatusBadRequest)
		return
	}

	proximoAviso, err := time.Parse("2006-01-02T15:04", proximoAvisoStr)
	if err != nil {
		http.Error(w, "Formato de fecha de próximo aviso inválido", http.StatusBadRequest)
		return
	}

	tarea := models.AdminTarea{
		Titulo:           titulo,
		Descripcion:      descripcion,
		Recurrencia:      recurrencia,
		FechaVencimiento: fechaVenc,
		ProximoAviso:     proximoAviso,
		Activo:           activo,
	}

	err = h.Repo.Crear(&tarea)
	if err != nil {
		http.Error(w, "Error al crear la tarea en BD: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", "tareaCreada")
	h.Listar(w, r)
}

// EditarUI devuelve el modal de edición para una tarea programada
func (h *AdminTareaHandler) EditarUI(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	tarea, err := h.Repo.ObtenerPorID(id)
	if err != nil {
		http.Error(w, "Tarea no encontrada", http.StatusNotFound)
		return
	}

	tmpl, err := template.ParseFiles("ui/templates/admin/tareas_ui.html")
	if err != nil {
		http.Error(w, "Error cargando la plantilla: "+err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.ExecuteTemplate(w, "formulario_editar_tarea", tarea)
}

// Actualizar procesa la modificación de una tarea programada existente
func (h *AdminTareaHandler) Actualizar(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error al leer formulario", http.StatusBadRequest)
		return
	}

	id, _ := strconv.Atoi(r.FormValue("id"))
	titulo := strings.TrimSpace(r.FormValue("titulo"))
	descripcion := strings.TrimSpace(r.FormValue("descripcion"))
	recurrencia := r.FormValue("recurrencia")
	fechaVencStr := r.FormValue("fecha_vencimiento")
	proximoAvisoStr := r.FormValue("proximo_aviso")
	activo := r.FormValue("activo") == "on"

	if titulo == "" || recurrencia == "" || fechaVencStr == "" || proximoAvisoStr == "" {
		http.Error(w, "Todos los campos principales son requeridos", http.StatusBadRequest)
		return
	}

	fechaVenc, err := time.Parse("2006-01-02T15:04", fechaVencStr)
	if err != nil {
		http.Error(w, "Formato de fecha de vencimiento inválido", http.StatusBadRequest)
		return
	}

	proximoAviso, err := time.Parse("2006-01-02T15:04", proximoAvisoStr)
	if err != nil {
		http.Error(w, "Formato de fecha de próximo aviso inválido", http.StatusBadRequest)
		return
	}

	tarea := models.AdminTarea{
		ID:               id,
		Titulo:           titulo,
		Descripcion:      descripcion,
		Recurrencia:      recurrencia,
		FechaVencimiento: fechaVenc,
		ProximoAviso:     proximoAviso,
		Activo:           activo,
	}

	err = h.Repo.Actualizar(&tarea)
	if err != nil {
		http.Error(w, "Error al actualizar la tarea: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.VistaUI(w, r)
}
