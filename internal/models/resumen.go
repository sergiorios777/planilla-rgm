package models

// KPICard representa una tarjeta individual para la UI (Bento Grid)
type KPICard struct {
	ID         string `json:"id"`
	Titulo     string `json:"titulo"`
	Valor      string `json:"valor"`
	Subtitulo  string `json:"subtitulo"`
	BadgeTexto string `json:"badge_texto"`
	BadgeClase string `json:"badge_clase"` // badge-info, badge-success, badge-warning, badge-purple
}

// TenantDashboardDTO agrupa la información del panel municipal
type TenantDashboardDTO struct {
	TotalTrabajadores    int       `json:"total_trabajadores"`
	TotalPlanillasAnno   int       `json:"total_planillas_anno"`
	PlanillasBorrador    int       `json:"planillas_borrador"`
	MontoTotalPlanilla   float64   `json:"monto_total_planilla"`
	MontoTotalFormateado string    `json:"monto_total_formateado"`
	Tarjetas             []KPICard `json:"tarjetas"`
}

// AdminDashboardDTO agrupa la información del panel global/superadmin
type AdminDashboardDTO struct {
	TotalTenantsActivos int       `json:"total_tenants_activos"`
	TotalUsuarios       int       `json:"total_usuarios"`
	PlanillasMesGlobal  int       `json:"planillas_mes_global"`
	TareasPendientes    int       `json:"tareas_pendientes"`
	Tarjetas            []KPICard `json:"tarjetas"`
}
