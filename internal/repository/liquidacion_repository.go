package repository

import (
	"database/sql"
	"planilla-rgm/internal/models"
)

type LiquidacionRepository struct {
	db *sql.DB
}

func NewLiquidacionRepository(db *sql.DB) *LiquidacionRepository {
	return &LiquidacionRepository{db: db}
}

// CrearLiquidacionCese crea un registro de liquidación de beneficios de cese
func (r *LiquidacionRepository) CrearLiquidacionCese(l *models.LiquidacionCese) error {
	query := `
		INSERT INTO liquidaciones_cese (
			tenant_id, contrato_id, fecha_inicio_computable, fecha_cese, motivo,
			anos_servicios, meses_servicios, remuneracion_computable, monto_cts,
			monto_vacaciones_truncas, monto_vacaciones_no_gozadas, monto_indemnizacion_vacacional,
			periodos_vencidos_vacaciones, periodos_no_vencidos_vacaciones,
			monto_gratificacion_trunca, total_liquidacion, estado
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRow(query,
		l.TenantID, l.ContratoID, l.FechaInicioComputable, l.FechaCese, l.Motivo,
		l.AnosServicios, l.MesesServicios, l.RemuneracionComputable, l.MontoCts,
		l.MontoVacacionesTruncas, l.MontoVacacionesNoGozadas, l.MontoIndemnizacionVacacional,
		l.PeriodosVencidosVacaciones, l.PeriodosNoVencidosVacaciones,
		l.MontoGratiTrunca, l.TotalLiquidacion, l.Estado,
	).Scan(&l.ID, &l.CreatedAt, &l.UpdatedAt)
}

// ObtenerLiquidacionCesePorID recupera una liquidación de cese por ID
func (r *LiquidacionRepository) ObtenerLiquidacionCesePorID(id int, tenantID int) (*models.LiquidacionCese, error) {
	query := `
		SELECT l.id, l.tenant_id, l.contrato_id, l.fecha_inicio_computable, l.fecha_cese, l.motivo,
		       l.anos_servicios, l.meses_servicios, l.remuneracion_computable, l.monto_cts,
		       l.monto_vacaciones_truncas, l.monto_vacaciones_no_gozadas, l.monto_indemnizacion_vacacional,
		       l.periodos_vencidos_vacaciones, l.periodos_no_vencidos_vacaciones,
		       l.monto_gratificacion_trunca, l.total_liquidacion, l.estado,
		       l.created_at, l.updated_at,
		       t.apellido_paterno || ' ' || t.apellido_materno || ', ' || t.nombres AS trabajador_nombre,
		       t.numero_documento AS trabajador_documento,
		       p.nombre AS puesto_nombre,
		       rl.codigo AS regimen
		FROM liquidaciones_cese l
		INNER JOIN contratos c ON l.contrato_id = c.id
		INNER JOIN trabajadores t ON c.trabajador_id = t.id
		INNER JOIN puestos p ON c.puesto_id = p.id
		INNER JOIN regimenes_laborales rl ON p.regimen_id = rl.id
		WHERE l.id = $1 AND l.tenant_id = $2
	`
	l := &models.LiquidacionCese{}
	err := r.db.QueryRow(query, id, tenantID).Scan(
		&l.ID, &l.TenantID, &l.ContratoID, &l.FechaInicioComputable, &l.FechaCese, &l.Motivo,
		&l.AnosServicios, &l.MesesServicios, &l.RemuneracionComputable, &l.MontoCts,
		&l.MontoVacacionesTruncas, &l.MontoVacacionesNoGozadas, &l.MontoIndemnizacionVacacional,
		&l.PeriodosVencidosVacaciones, &l.PeriodosNoVencidosVacaciones,
		&l.MontoGratiTrunca, &l.TotalLiquidacion, &l.Estado,
		&l.CreatedAt, &l.UpdatedAt,
		&l.TrabajadorNombre, &l.TrabajadorDocumento, &l.PuestoNombre, &l.Regimen,
	)
	if err != nil {
		return nil, err
	}
	return l, nil
}

// ListarLiquidacionesCese lista todas las liquidaciones registradas en la municipalidad
func (r *LiquidacionRepository) ListarLiquidacionesCese(tenantID int) ([]models.LiquidacionCese, error) {
	query := `
		SELECT l.id, l.tenant_id, l.contrato_id, l.fecha_inicio_computable, l.fecha_cese, l.motivo,
		       l.anos_servicios, l.meses_servicios, l.remuneracion_computable, l.monto_cts,
		       l.monto_vacaciones_truncas, l.monto_vacaciones_no_gozadas, l.monto_indemnizacion_vacacional,
		       l.periodos_vencidos_vacaciones, l.periodos_no_vencidos_vacaciones,
		       l.monto_gratificacion_trunca, l.total_liquidacion, l.estado,
		       l.created_at, l.updated_at,
		       t.apellido_paterno || ' ' || t.apellido_materno || ', ' || t.nombres AS trabajador_nombre,
		       t.numero_documento AS trabajador_documento,
		       p.nombre AS puesto_nombre,
		       rl.codigo AS regimen
		FROM liquidaciones_cese l
		INNER JOIN contratos c ON l.contrato_id = c.id
		INNER JOIN trabajadores t ON c.trabajador_id = t.id
		INNER JOIN puestos p ON c.puesto_id = p.id
		INNER JOIN regimenes_laborales rl ON p.regimen_id = rl.id
		WHERE l.tenant_id = $1
		ORDER BY l.created_at DESC
	`
	rows, err := r.db.Query(query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.LiquidacionCese
	for rows.Next() {
		var l models.LiquidacionCese
		err := rows.Scan(
			&l.ID, &l.TenantID, &l.ContratoID, &l.FechaInicioComputable, &l.FechaCese, &l.Motivo,
			&l.AnosServicios, &l.MesesServicios, &l.RemuneracionComputable, &l.MontoCts,
			&l.MontoVacacionesTruncas, &l.MontoVacacionesNoGozadas, &l.MontoIndemnizacionVacacional,
			&l.PeriodosVencidosVacaciones, &l.PeriodosNoVencidosVacaciones,
			&l.MontoGratiTrunca, &l.TotalLiquidacion, &l.Estado,
			&l.CreatedAt, &l.UpdatedAt,
			&l.TrabajadorNombre, &l.TrabajadorDocumento, &l.PuestoNombre, &l.Regimen,
		)
		if err != nil {
			return nil, err
		}
		lista = append(lista, l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return lista, nil
}

// EliminarLiquidacionCese elimina un registro de liquidación de cese
func (r *LiquidacionRepository) EliminarLiquidacionCese(id int, tenantID int) error {
	query := `DELETE FROM liquidaciones_cese WHERE id = $1 AND tenant_id = $2`
	_, err := r.db.Exec(query, id, tenantID)
	return err
}
