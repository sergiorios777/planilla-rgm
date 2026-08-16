package handlers

import (
	"html/template"
	"net/http"

	"planilla-rgm/internal/helpers"
	"planilla-rgm/internal/middleware"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
	"planilla-rgm/internal/services"
)

type ResumenHandler struct {
	Service     *services.ResumenService
	TenantRepo  *repository.TenantRepository
	UsuarioRepo *repository.UsuarioRepository
}

func NewResumenHandler(service *services.ResumenService, tenantRepo *repository.TenantRepository, usuarioRepo *repository.UsuarioRepository) *ResumenHandler {
	return &ResumenHandler{
		Service:     service,
		TenantRepo:  tenantRepo,
		UsuarioRepo: usuarioRepo,
	}
}

// IndexTenant renderiza tenant_index.html con los KPIs procesados por el servicio
func (h *ResumenHandler) IndexTenant(w http.ResponseWriter, r *http.Request) {
	var uID, tID int

	// Extraer usuario_id (soporta contextKey y string)
	if val := r.Context().Value(middleware.UsuarioIDKey); val != nil {
		if fVal, ok := val.(float64); ok {
			uID = int(fVal)
		} else if iVal, ok := val.(int); ok {
			uID = iVal
		}
	}
	if uID == 0 {
		if val := r.Context().Value("usuario_id"); val != nil {
			if fVal, ok := val.(float64); ok {
				uID = int(fVal)
			} else if iVal, ok := val.(int); ok {
				uID = iVal
			}
		}
	}

	// Extraer tenant_id
	if val := r.Context().Value("tenant_id"); val != nil {
		if fVal, ok := val.(float64); ok {
			tID = int(fVal)
		} else if iVal, ok := val.(int); ok {
			tID = iVal
		}
	}

	var tenantNombre string
	if tID > 0 && h.TenantRepo != nil {
		t, err := h.TenantRepo.ObtenerPorID(tID)
		if err == nil && t != nil {
			tenantNombre = t.Nombre
		}
	}

	var usuarioNombre, usuarioRol string
	if uID > 0 && h.UsuarioRepo != nil {
		u, err := h.UsuarioRepo.ObtenerPorID(uID)
		if err == nil && u != nil {
			usuarioNombre = u.Nombre
			usuarioRol = helpers.FormatearRol(u.Rol)
		}
	}

	if usuarioRol == "" {
		if rVal := r.Context().Value(middleware.RolKey); rVal != nil {
			if rStr, ok := rVal.(string); ok && rStr != "" {
				usuarioRol = helpers.FormatearRol(rStr)
			}
		}
	}

	dashboardData, err := h.Service.ObtenerDashboardTenant(tID)
	if err != nil {
		dashboardData = &models.TenantDashboardDTO{
			MontoTotalFormateado: "0.00",
		}
	}

	tmpl, err := template.ParseFiles("ui/templates/layouts/tenant_index.html", "ui/templates/layouts/iconos_sprite.html")
	if err != nil {
		http.Error(w, "Error cargando la vista principal del inquilino", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"TenantNombre":         tenantNombre,
		"UsuarioNombre":        usuarioNombre,
		"UsuarioRol":           usuarioRol,
		"TotalTrabajadores":    dashboardData.TotalTrabajadores,
		"TotalPlanillasAnno":   dashboardData.TotalPlanillasAnno,
		"PlanillasBorrador":    dashboardData.PlanillasBorrador,
		"MontoTotalPlanilla":   dashboardData.MontoTotalFormateado,
		"Tarjetas":             dashboardData.Tarjetas,
	}

	tmpl.Execute(w, data)
}

// RefrescarKPITenantParcial renderiza sólo el componente parcial Bento Grid para HTMX
func (h *ResumenHandler) RefrescarKPITenantParcial(w http.ResponseWriter, r *http.Request) {
	var tID int
	if val, ok := r.Context().Value("tenant_id").(float64); ok {
		tID = int(val)
	}

	dashboardData, err := h.Service.ObtenerDashboardTenant(tID)
	if err != nil {
		dashboardData = &models.TenantDashboardDTO{
			MontoTotalFormateado: "0.00",
		}
	}

	tmpl, err := template.ParseFiles("ui/templates/components/kpi_bento_grid.html")
	if err != nil {
		http.Error(w, "Error cargando fragmento de tarjetas KPI", http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, dashboardData)
}

// RefrescarKPIAdminParcial renderiza sólo el componente parcial para el Panel Admin
func (h *ResumenHandler) RefrescarKPIAdminParcial(w http.ResponseWriter, r *http.Request) {
	adminData, err := h.Service.ObtenerDashboardAdmin()
	if err != nil {
		adminData = &models.AdminDashboardDTO{}
	}

	tmpl, err := template.ParseFiles("ui/templates/components/kpi_bento_grid.html")
	if err != nil {
		http.Error(w, "Error cargando fragmento de tarjetas KPI admin", http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, adminData)
}
