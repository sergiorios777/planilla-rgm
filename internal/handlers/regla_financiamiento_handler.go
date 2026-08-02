package handlers

import (
	"encoding/json"
	"net/http"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
	"strconv"
	"strings"
)

type ReglaFinanciamientoHandler struct {
	Repo *repository.PlanillaRepository
}

// HandleRoot conmuta según el método HTTP para /api/tenant/reglas-financiamiento
func (h *ReglaFinanciamientoHandler) HandleRoot(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.Listar(w, r)
	case http.MethodPost:
		h.Crear(w, r)
	default:
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
	}
}

// HandleWithID conmuta según el método HTTP para /api/tenant/reglas-financiamiento/{id}
func (h *ReglaFinanciamientoHandler) HandleWithID(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.Obtener(w, r)
	case http.MethodPut:
		h.Actualizar(w, r)
	case http.MethodDelete:
		h.Eliminar(w, r)
	default:
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
	}
}

func (h *ReglaFinanciamientoHandler) Listar(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	reglas, err := h.Repo.ObtenerReglasFinanciamiento(r.Context(), tenantID)
	if err != nil {
		http.Error(w, "Error al obtener reglas de financiamiento: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if reglas == nil {
		reglas = []models.ReglaFinanciamientoConcepto{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reglas)
}

func (h *ReglaFinanciamientoHandler) Obtener(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	idStr := r.PathValue("id")
	if idStr == "" {
		idStr = r.URL.Query().Get("id")
	}
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "ID de regla inválido", http.StatusBadRequest)
		return
	}

	regla, err := h.Repo.ObtenerReglaFinanciamientoPorID(r.Context(), id, tenantID)
	if err != nil {
		http.Error(w, "Regla no encontrada: "+err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(regla)
}

func (h *ReglaFinanciamientoHandler) Crear(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	var regla models.ReglaFinanciamientoConcepto
	if err := json.NewDecoder(r.Body).Decode(&regla); err != nil {
		http.Error(w, "Cuerpo de solicitud inválido: "+err.Error(), http.StatusBadRequest)
		return
	}
	regla.TenantID = tenantID

	if err := h.Repo.CrearReglaFinanciamiento(r.Context(), &regla); err != nil {
		http.Error(w, "Error al crear regla: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Recargar regla completa con JOINs
	reglaCompleta, err := h.Repo.ObtenerReglaFinanciamientoPorID(r.Context(), regla.ID, tenantID)
	if err == nil && reglaCompleta != nil {
		regla = *reglaCompleta
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(regla)
}

func (h *ReglaFinanciamientoHandler) Actualizar(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	idStr := r.PathValue("id")
	if idStr == "" {
		idStr = r.URL.Query().Get("id")
	}
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "ID de regla inválido", http.StatusBadRequest)
		return
	}

	var regla models.ReglaFinanciamientoConcepto
	if err := json.NewDecoder(r.Body).Decode(&regla); err != nil {
		http.Error(w, "Cuerpo de solicitud inválido: "+err.Error(), http.StatusBadRequest)
		return
	}
	regla.ID = id
	regla.TenantID = tenantID

	if err := h.Repo.ActualizarReglaFinanciamiento(r.Context(), &regla); err != nil {
		http.Error(w, "Error al actualizar regla: "+err.Error(), http.StatusInternalServerError)
		return
	}

	reglaCompleta, err := h.Repo.ObtenerReglaFinanciamientoPorID(r.Context(), id, tenantID)
	if err == nil && reglaCompleta != nil {
		regla = *reglaCompleta
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(regla)
}

func (h *ReglaFinanciamientoHandler) Eliminar(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	idStr := r.PathValue("id")
	if idStr == "" {
		idStr = r.URL.Query().Get("id")
	}
	if idStr == "" {
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) > 0 {
			idStr = parts[len(parts)-1]
		}
	}
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "ID de regla inválido", http.StatusBadRequest)
		return
	}

	if err := h.Repo.EliminarReglaFinanciamiento(r.Context(), id, tenantID); err != nil {
		http.Error(w, "Error al eliminar regla: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Regla eliminada correctamente"})
}
