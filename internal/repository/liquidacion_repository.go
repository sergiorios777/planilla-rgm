package repository

import (
	"database/sql"
	"fmt"
	"planilla-rgm/internal/helpers"
	"planilla-rgm/internal/models"
	"strings"
	"time"
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
			anos_servicios, meses_servicios, dias_servicios, remuneracion_computable, monto_cts,
			monto_vacaciones_truncas, monto_vacaciones_no_gozadas, monto_indemnizacion_vacacional,
			periodos_vencidos_vacaciones, periodos_no_vencidos_vacaciones,
			monto_gratificacion_trunca, total_liquidacion, estado
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRow(query,
		l.TenantID, l.ContratoID, l.FechaInicioComputable, l.FechaCese, l.Motivo,
		l.AnosServicios, l.MesesServicios, l.DiasServicios, l.RemuneracionComputable, l.MontoCts,
		l.MontoVacacionesTruncas, l.MontoVacacionesNoGozadas, l.MontoIndemnizacionVacacional,
		l.PeriodosVencidosVacaciones, l.PeriodosNoVencidosVacaciones,
		l.MontoGratiTrunca, l.TotalLiquidacion, l.Estado,
	).Scan(&l.ID, &l.CreatedAt, &l.UpdatedAt)
}

// ObtenerLiquidacionCesePorID recupera una liquidación de cese por ID
func (r *LiquidacionRepository) ObtenerLiquidacionCesePorID(id int, tenantID int) (*models.LiquidacionCese, error) {
	query := `
		SELECT l.id, l.tenant_id, l.contrato_id, l.fecha_inicio_computable, l.fecha_cese, l.motivo,
		       l.anos_servicios, l.meses_servicios, COALESCE(l.dias_servicios, 0), l.remuneracion_computable, l.monto_cts,
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
		&l.AnosServicios, &l.MesesServicios, &l.DiasServicios, &l.RemuneracionComputable, &l.MontoCts,
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

// ObtenerDatosReporteLiquidacion recupera la estructura completa con metadatos para el PDF
func (r *LiquidacionRepository) ObtenerDatosReporteLiquidacion(id int, tenantID int) (*models.DatosReporteLiquidacion, error) {
	query := `
		SELECT l.id, l.tenant_id, l.contrato_id, l.fecha_inicio_computable, l.fecha_cese, l.motivo,
		       l.anos_servicios, l.meses_servicios, COALESCE(l.dias_servicios, 0), l.remuneracion_computable, l.monto_cts,
		       l.monto_vacaciones_truncas, l.monto_vacaciones_no_gozadas, l.monto_indemnizacion_vacacional,
		       l.periodos_vencidos_vacaciones, l.periodos_no_vencidos_vacaciones,
		       l.monto_gratificacion_trunca, l.total_liquidacion, l.estado,
		       l.created_at, l.updated_at,
		       t.apellido_paterno || ' ' || t.apellido_materno || ', ' || t.nombres AS trabajador_nombre,
		       t.numero_documento AS trabajador_documento,
		       p.nombre AS puesto_nombre,
		       rl.codigo AS regimen,
		       COALESCE(p.sueldo_presupuestado, 0) AS sueldo_basico,
		       c.fecha_inicio AS fecha_ingreso_contrato,
		       tn.nombre AS tenant_nombre,
		       COALESCE(tn.ruc, '') AS tenant_ruc,
		       COALESCE(tn.logo_url, '') AS tenant_logo_url,
		       COALESCE(t.regimen_pensionario, 'ONP') AS regimen_pensionario,
		       COALESCE(afp.nombre, '') AS afp_nombre
		FROM liquidaciones_cese l
		INNER JOIN contratos c ON l.contrato_id = c.id
		INNER JOIN trabajadores t ON c.trabajador_id = t.id
		INNER JOIN puestos p ON c.puesto_id = p.id
		INNER JOIN regimenes_laborales rl ON p.regimen_id = rl.id
		INNER JOIN tenants tn ON l.tenant_id = tn.id
		LEFT JOIN afps afp ON t.afp_id = afp.id
		WHERE l.id = $1 AND l.tenant_id = $2
	`

	datos := &models.DatosReporteLiquidacion{}
	l := &datos.Liquidacion
	var sueldoBasico float64
	var fechaIngresoContrato time.Time
	var regimenPension, afpNombre sql.NullString

	err := r.db.QueryRow(query, id, tenantID).Scan(
		&l.ID, &l.TenantID, &l.ContratoID, &l.FechaInicioComputable, &l.FechaCese, &l.Motivo,
		&l.AnosServicios, &l.MesesServicios, &l.DiasServicios, &l.RemuneracionComputable, &l.MontoCts,
		&l.MontoVacacionesTruncas, &l.MontoVacacionesNoGozadas, &l.MontoIndemnizacionVacacional,
		&l.PeriodosVencidosVacaciones, &l.PeriodosNoVencidosVacaciones,
		&l.MontoGratiTrunca, &l.TotalLiquidacion, &l.Estado,
		&l.CreatedAt, &l.UpdatedAt,
		&l.TrabajadorNombre, &l.TrabajadorDocumento, &l.PuestoNombre, &l.Regimen,
		&sueldoBasico, &fechaIngresoContrato,
		&datos.TenantNombre, &datos.TenantRUC, &datos.TenantLogoURL,
		&regimenPension, &afpNombre,
	)
	if err != nil {
		return nil, err
	}

	datos.FechaIngreso = fechaIngresoContrato
	datos.SueldoBasico = sueldoBasico
	datos.RemuneracionComputable = l.RemuneracionComputable

	// Promedio de gratificación u otros conceptos
	promedioGrati := 0.0
	if l.RemuneracionComputable > sueldoBasico {
		promedioGrati = l.RemuneracionComputable - sueldoBasico
	}
	datos.PromedioGratificacion = promedioGrati

	// Periodo semestral CTS
	mesCese := l.FechaCese.Month()
	anioCese := l.FechaCese.Year()
	if mesCese >= 5 && mesCese <= 10 {
		datos.CtsPeriodoInicio = fmt.Sprintf("01/05/%d", anioCese)
		datos.CtsPeriodoFin = fmt.Sprintf("31/10/%d", anioCese)
	} else {
		anioInicio := anioCese
		if mesCese < 5 {
			anioInicio--
		}
		datos.CtsPeriodoInicio = fmt.Sprintf("01/11/%d", anioInicio)
		datos.CtsPeriodoFin = fmt.Sprintf("30/04/%d", anioInicio+1)
	}

	datos.CtsMeses = l.MesesServicios
	datos.CtsDias = l.DiasServicios
	datos.MontoCtsMeses = (l.RemuneracionComputable / 12.0) * float64(l.MesesServicios)
	datos.MontoCtsDias = (l.RemuneracionComputable / 12.0 / 30.0) * float64(l.DiasServicios)

	// Vacaciones
	remVacacional := sueldoBasico
	datos.VacacionesMeses = l.MesesServicios
	datos.VacacionesDias = l.DiasServicios
	datos.MontoVacacionesMeses = (remVacacional / 12.0) * float64(l.MesesServicios)
	datos.MontoVacacionesDias = (remVacacional / 12.0 / 30.0) * float64(l.DiasServicios)
	datos.VacacionesBrutas = l.MontoVacacionesTruncas + l.MontoVacacionesNoGozadas

	pensionDesc := "ONP"
	if regimenPension.Valid && regimenPension.String != "" {
		pensionDesc = regimenPension.String
	}
	if afpNombre.Valid && afpNombre.String != "" {
		pensionDesc = fmt.Sprintf("AFP (%s)", afpNombre.String)
	}
	datos.DescuentoPensionNombre = pensionDesc

	tasaPension := 0.13
	if strings.Contains(strings.ToUpper(pensionDesc), "AFP") {
		tasaPension = 0.1317
	}
	datos.MontoDescuentoPension = datos.VacacionesBrutas * tasaPension
	datos.VacacionesNetas = datos.VacacionesBrutas - datos.MontoDescuentoPension + l.MontoIndemnizacionVacacional

	// Gratificación Trunca
	if mesCese <= 6 {
		datos.GratiSemestreTipo = "Fiestas Patrias"
		datos.GratiPeriodoInicio = fmt.Sprintf("01/01/%d", anioCese)
		datos.GratiPeriodoFin = fmt.Sprintf("30/06/%d", anioCese)
	} else {
		datos.GratiSemestreTipo = "Navidad"
		datos.GratiPeriodoInicio = fmt.Sprintf("01/07/%d", anioCese)
		datos.GratiPeriodoFin = fmt.Sprintf("31/12/%d", anioCese)
	}
	datos.GratiMeses = l.MesesServicios
	datos.GratiDias = l.DiasServicios
	datos.MontoGratiMeses = (remVacacional / 6.0) * float64(l.MesesServicios)
	datos.MontoGratiDias = (remVacacional / 6.0 / 30.0) * float64(l.DiasServicios)

	// Bonificación Extraordinaria (Ley 29351 / 30334: 9%)
	datos.BonificacionEspecial = l.MontoGratiTrunca * 0.09

	// Total Liquidación
	datos.TotalLiquidacion = l.MontoCts + datos.VacacionesNetas + l.MontoGratiTrunca + datos.BonificacionEspecial
	if l.TotalLiquidacion > 0 {
		datos.TotalLiquidacion = l.TotalLiquidacion
	}
	datos.MontoEnLetras = helpers.NumeroALetras(datos.TotalLiquidacion)
	datos.FechaEmisionTexto = l.FechaCese.Format("02/01/2006")

	return datos, nil
}

// ListarLiquidacionesCese lista todas las liquidaciones registradas en la municipalidad
func (r *LiquidacionRepository) ListarLiquidacionesCese(tenantID int) ([]models.LiquidacionCese, error) {
	query := `
		SELECT l.id, l.tenant_id, l.contrato_id, l.fecha_inicio_computable, l.fecha_cese, l.motivo,
		       l.anos_servicios, l.meses_servicios, COALESCE(l.dias_servicios, 0), l.remuneracion_computable, l.monto_cts,
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
			&l.AnosServicios, &l.MesesServicios, &l.DiasServicios, &l.RemuneracionComputable, &l.MontoCts,
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
