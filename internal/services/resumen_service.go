package services

import (
	"fmt"
	"strings"
	"time"

	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
)

type ResumenService struct {
	Repo *repository.ResumenRepository
}

func NewResumenService(repo *repository.ResumenRepository) *ResumenService {
	return &ResumenService{Repo: repo}
}

// ObtenerDashboardTenant genera las tarjetas resumen y métricas del panel municipal
func (s *ResumenService) ObtenerDashboardTenant(tenantID int) (*models.TenantDashboardDTO, error) {
	ahora := time.Now()
	anio := ahora.Year()
	mes := int(ahora.Month())

	totalTrab, planillasAnno, borrador, montoMes, err := s.Repo.ObtenerKPIsTenant(tenantID, anio, mes)
	if err != nil {
		return nil, err
	}

	montoFormateado := formatearMoneda(montoMes)

	tarjetas := []models.KPICard{
		{
			ID:         "kpi-trabajadores",
			Titulo:     "Trabajadores Activos",
			Valor:      fmt.Sprintf("%d", totalTrab),
			Subtitulo:  "Personal registrado en entidad",
			BadgeTexto: "Legajo",
			BadgeClase: "badge-info",
		},
		{
			ID:         "kpi-planillas-anno",
			Titulo:     "Planillas Procesadas",
			Valor:      fmt.Sprintf("%d", planillasAnno),
			Subtitulo:  "Lotes de cálculo finalizados",
			BadgeTexto: fmt.Sprintf("Año %d", anio),
			BadgeClase: "badge-success",
		},
		{
			ID:         "kpi-planillas-borrador",
			Titulo:     "Planilla en Borrador",
			Valor:      fmt.Sprintf("%d", borrador),
			Subtitulo:  "Pendientes de cierre/aprobación",
			BadgeTexto: "En Proceso",
			BadgeClase: "badge-warning",
		},
		{
			ID:         "kpi-monto-total",
			Titulo:     "Monto Total Planilla",
			Valor:      fmt.Sprintf("S/ %s", montoFormateado),
			Subtitulo:  "Presupuesto mensual estimado",
			BadgeTexto: "Bruto Mes",
			BadgeClase: "badge-purple",
		},
	}

	return &models.TenantDashboardDTO{
		TotalTrabajadores:    totalTrab,
		TotalPlanillasAnno:   planillasAnno,
		PlanillasBorrador:    borrador,
		MontoTotalPlanilla:   montoMes,
		MontoTotalFormateado: montoFormateado,
		Tarjetas:             tarjetas,
	}, nil
}

// ObtenerDashboardAdmin genera las tarjetas resumen para el súper admin
func (s *ResumenService) ObtenerDashboardAdmin() (*models.AdminDashboardDTO, error) {
	ahora := time.Now()
	anio := ahora.Year()
	mes := int(ahora.Month())

	tenants, usuarios, planillasMes, tareasPend, err := s.Repo.ObtenerKPIsAdmin(anio, mes)
	if err != nil {
		return nil, err
	}

	tarjetas := []models.KPICard{
		{
			ID:         "kpi-admin-tenants",
			Titulo:     "Tenants Activos",
			Valor:      fmt.Sprintf("%d", tenants),
			Subtitulo:  "Municipalidades registradas",
			BadgeTexto: "Inquilinos",
			BadgeClase: "badge-info",
		},
		{
			ID:         "kpi-admin-usuarios",
			Titulo:     "Usuarios Totales",
			Valor:      fmt.Sprintf("%d", usuarios),
			Subtitulo:  "Cuentas activas en la plataforma",
			BadgeTexto: "Plataforma",
			BadgeClase: "badge-success",
		},
		{
			ID:         "kpi-admin-planillas",
			Titulo:     "Planillas Mes Global",
			Valor:      fmt.Sprintf("%d", planillasMes),
			Subtitulo:  "Planillas procesadas este mes",
			BadgeTexto: fmt.Sprintf("Mes %d/%d", mes, anio),
			BadgeClase: "badge-warning",
		},
		{
			ID:         "kpi-admin-tareas",
			Titulo:     "Tareas Activas",
			Valor:      fmt.Sprintf("%d", tareasPend),
			Subtitulo:  "Automatizaciones y cron jobs",
			BadgeTexto: "Motor Tareas",
			BadgeClase: "badge-purple",
		},
	}

	return &models.AdminDashboardDTO{
		TotalTenantsActivos: tenants,
		TotalUsuarios:       usuarios,
		PlanillasMesGlobal:  planillasMes,
		TareasPendientes:    tareasPend,
		Tarjetas:            tarjetas,
	}, nil
}

// Helper para dar formato monetario con separadores de miles (ej: 12,345.67)
func formatearMoneda(val float64) string {
	raw := fmt.Sprintf("%.2f", val)
	partes := strings.Split(raw, ".")
	entero := partes[0]
	decimal := partes[1]

	var res []string
	l := len(entero)
	for i, c := range entero {
		if i > 0 && (l-i)%3 == 0 {
			res = append(res, ",")
		}
		res = append(res, string(c))
	}

	return strings.Join(res, "") + "." + decimal
}

// GenerarTarjetasDetallePlanilla calcula los 5 KPIs bento grid para la pantalla de detalle de planilla
func GenerarTarjetasDetallePlanilla(detalles []models.PlanillaDetalle) []models.KPICard {
	var totalIngresos, totalRetenciones, totalAportes, netoPagar float64
	for _, d := range detalles {
		totalIngresos += d.TotalIngresos
		totalRetenciones += d.TotalRetenciones
		totalAportes += d.TotalAportes
		netoPagar += d.NetoPagar
	}
	costoTotal := totalIngresos + totalAportes

	return []models.KPICard{
		{
			ID:         "kpi-total-ingresos",
			Titulo:     "Total Ingresos (Bruto)",
			Valor:      fmt.Sprintf("S/ %s", formatearMoneda(totalIngresos)),
			Subtitulo:  "Remuneración bruta calculada",
			BadgeTexto: "Ingresos",
			BadgeClase: "badge-success",
		},
		{
			ID:         "kpi-total-retenciones",
			Titulo:     "Total Retenciones",
			Valor:      fmt.Sprintf("S/ %s", formatearMoneda(totalRetenciones)),
			Subtitulo:  "Descuentos de ley y AFP/ONP",
			BadgeTexto: "Descuentos",
			BadgeClase: "badge-danger",
		},
		{
			ID:         "kpi-total-aportes",
			Titulo:     "Total Aportes Entidad",
			Valor:      fmt.Sprintf("S/ %s", formatearMoneda(totalAportes)),
			Subtitulo:  "Contribución empleador (EsSalud)",
			BadgeTexto: "Aportes",
			BadgeClase: "badge-info",
		},
		{
			ID:         "kpi-costo-total",
			Titulo:     "Costo Total Planilla",
			Valor:      fmt.Sprintf("S/ %s", formatearMoneda(costoTotal)),
			Subtitulo:  "Suma de Ingresos + Aportes",
			BadgeTexto: "Costo Total",
			BadgeClase: "badge-warning",
		},
		{
			ID:         "kpi-neto-pagar",
			Titulo:     "Total Neto a Pagar",
			Valor:      fmt.Sprintf("S/ %s", formatearMoneda(netoPagar)),
			Subtitulo:  "Total líquido para abono",
			BadgeTexto: "Líquido",
			BadgeClase: "badge-purple",
		},
	}
}
