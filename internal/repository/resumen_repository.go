package repository

import (
	"database/sql"
)

type ResumenRepository struct {
	db *sql.DB
}

func NewResumenRepository(db *sql.DB) *ResumenRepository {
	return &ResumenRepository{db: db}
}

// ObtenerKPIsTenant obtiene las 4 métricas principales en una sola consulta consolidada
func (r *ResumenRepository) ObtenerKPIsTenant(tenantID int, anio int, mes int) (totalTrab int, planillasAnno int, planillasBorrador int, montoMes float64, err error) {
	query := `
		SELECT 
			(SELECT COUNT(*) FROM trabajadores WHERE tenant_id = $1 AND activo = true) AS total_trabajadores,
			(SELECT COUNT(*) FROM planillas WHERE tenant_id = $1 AND anio = $2 AND estado != 'BORRADOR') AS total_planillas_anno,
			(SELECT COUNT(*) FROM planillas WHERE tenant_id = $1 AND estado = 'BORRADOR') AS planillas_borrador,
			(SELECT COALESCE(SUM(pd.total_ingresos), 0.00) 
			 FROM planilla_detalles pd 
			 JOIN planillas p ON pd.planilla_id = p.id 
			 WHERE p.tenant_id = $1 AND p.anio = $2 AND p.mes = $3) AS monto_total_mes
	`
	err = r.db.QueryRow(query, tenantID, anio, mes).Scan(&totalTrab, &planillasAnno, &planillasBorrador, &montoMes)
	return
}

// ObtenerKPIsAdmin obtiene métricas globales para el panel superadmin
func (r *ResumenRepository) ObtenerKPIsAdmin(anio int, mes int) (tenants int, usuarios int, planillasMes int, tareasPend int, err error) {
	query := `
		SELECT 
			(SELECT COUNT(*) FROM tenants WHERE activo = true) AS total_tenants,
			(SELECT COUNT(*) FROM usuarios) AS total_usuarios,
			(SELECT COUNT(*) FROM planillas WHERE anio = $1 AND mes = $2) AS planillas_mes,
			(SELECT COUNT(*) FROM admin_tareas WHERE activo = true) AS tareas_pendientes
	`
	err = r.db.QueryRow(query, anio, mes).Scan(&tenants, &usuarios, &planillasMes, &tareasPend)
	return
}
