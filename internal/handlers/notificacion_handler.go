package handlers

import (
	"html/template"
	"net/http"
	"planilla-rgm/internal/middleware"
	"planilla-rgm/internal/repository"
)

// NotificacionHandler maneja el polling y render de la campana
type NotificacionHandler struct {
	Repo *repository.NotificacionRepository
}

// NewNotificacionHandler crea un nuevo handler de notificaciones
func NewNotificacionHandler(repo *repository.NotificacionRepository) *NotificacionHandler {
	return &NotificacionHandler{Repo: repo}
}

// CampanaContadorUI devuelve la campana con el puntito rojo si hay no leídas
func (h *NotificacionHandler) CampanaContadorUI(w http.ResponseWriter, r *http.Request) {
	tID, uID := obtenerUsuarioTenantID(r)

	conteo, err := h.Repo.ContarNoLeidas(tID, uID)
	if err != nil {
		http.Error(w, "Error al contar notificaciones: "+err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFiles("ui/templates/components/notificaciones_ui.html")
	if err != nil {
		http.Error(w, "Error al cargar plantilla: "+err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		Conteo int
	}{
		Conteo: conteo,
	}

	tmpl.ExecuteTemplate(w, "campana", data)
}

// ListaNotificacionesUI devuelve la lista de notificaciones recientes y las marca como leídas
func (h *NotificacionHandler) ListaNotificacionesUI(w http.ResponseWriter, r *http.Request) {
	tID, uID := obtenerUsuarioTenantID(r)

	// Obtener últimas 10 notificaciones
	lista, err := h.Repo.ObtenerRecientes(tID, uID, 10)
	if err != nil {
		http.Error(w, "Error al obtener notificaciones: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Marcar notificaciones como leídas
	err = h.Repo.MarcarComoLeidas(tID, uID)
	if err != nil {
		// Loguear pero no interrumpir flujo para no dejar la UI rota
		http.Error(w, "Error al actualizar leídas: "+err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFiles("ui/templates/components/notificaciones_ui.html")
	if err != nil {
		http.Error(w, "Error al cargar plantilla: "+err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.ExecuteTemplate(w, "lista_notificaciones", lista)
}

// Helper para extraer los IDs del contexto de seguridad de forma robusta
func obtenerUsuarioTenantID(r *http.Request) (*int, *int) {
	var tID *int
	var uID *int

	// Extraer tenant_id
	if val := r.Context().Value("tenant_id"); val != nil {
		if fVal, ok := val.(float64); ok {
			v := int(fVal)
			tID = &v
		} else if iVal, ok := val.(int); ok {
			tID = &iVal
		}
	}

	// Extraer usuario_id
	if val := r.Context().Value(middleware.UsuarioIDKey); val != nil {
		if fVal, ok := val.(float64); ok {
			v := int(fVal)
			uID = &v
		} else if iVal, ok := val.(int); ok {
			uID = &iVal
		}
	}

	return tID, uID
}
