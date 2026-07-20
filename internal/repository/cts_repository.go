package repository

import (
	"database/sql"
	"math"
	"planilla-rgm/internal/models"
	"time"

	"github.com/lib/pq"
)

type CtsRepository struct {
	db *sql.DB
}

func NewCtsRepository(db *sql.DB) *CtsRepository {
	return &CtsRepository{db: db}
}

// CrearPlanillaCts inserta una nueva cabecera de planilla CTS semestral
func (r *CtsRepository) CrearPlanillaCts(p *models.PlanillaCts) error {
	query := `
		INSERT INTO planillas_cts (tenant_id, anio, periodo, estado)
		VALUES ($1, $2, $3, $4) RETURNING id, fecha_calculo, created_at, updated_at
	`
	return r.db.QueryRow(query, p.TenantID, p.Anio, p.Periodo, p.Estado).
		Scan(&p.ID, &p.FechaCalculo, &p.CreatedAt, &p.UpdatedAt)
}

// ObtenerPlanillaCtsPorID recupera una planilla CTS por ID y TenantID
func (r *CtsRepository) ObtenerPlanillaCtsPorID(id int, tenantID int) (*models.PlanillaCts, error) {
	query := `
		SELECT id, tenant_id, anio, periodo, estado, fecha_calculo, created_at, updated_at
		FROM planillas_cts
		WHERE id = $1 AND tenant_id = $2
	`
	p := &models.PlanillaCts{}
	err := r.db.QueryRow(query, id, tenantID).
		Scan(&p.ID, &p.TenantID, &p.Anio, &p.Periodo, &p.Estado, &p.FechaCalculo, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// ListarPlanillasCts lista todas las planillas CTS de un tenant
func (r *CtsRepository) ListarPlanillasCts(tenantID int) ([]models.PlanillaCts, error) {
	query := `
		SELECT id, tenant_id, anio, periodo, estado, fecha_calculo, created_at, updated_at
		FROM planillas_cts
		WHERE tenant_id = $1
		ORDER BY anio DESC, periodo DESC
	`
	rows, err := r.db.Query(query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.PlanillaCts
	for rows.Next() {
		var p models.PlanillaCts
		err := rows.Scan(&p.ID, &p.TenantID, &p.Anio, &p.Periodo, &p.Estado, &p.FechaCalculo, &p.CreatedAt, &p.UpdatedAt)
		if err == nil {
			lista = append(lista, p)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return lista, nil
}

// GuardarDetallesCts inserta en bloque los detalles pre-calculados de la CTS
func (r *CtsRepository) GuardarDetallesCts(detalles []models.PlanillaCtsDetalle) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO planilla_cts_detalles (
			planilla_cts_id, contrato_id, sueldo_basico, asignacion_familiar,
			sexto_gratificacion, promedio_variables, remuneracion_computable,
			meses_computables, dias_faltas, monto_descuento_faltas, monto_cts
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, d := range detalles {
		_, err := stmt.Exec(
			d.PlanillaCtsID, d.ContratoID, d.SueldoBasico, d.AsignacionFamilia,
			d.SextoGratificacion, d.PromedioVariables, d.RemuneracionComputable,
			d.MesesComputables, d.DiasFaltas, d.MontoDescuentoFaltas, d.MontoCts,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// ObtenerDetallesCts extrae los detalles calculados de una planilla de CTS
func (r *CtsRepository) ObtenerDetallesCts(planillaCtsID int) ([]models.PlanillaCtsDetalle, error) {
	query := `
		SELECT d.id, d.planilla_cts_id, d.contrato_id, d.sueldo_basico, d.asignacion_familiar,
		       d.sexto_gratificacion, d.promedio_variables, d.remuneracion_computable,
		       d.meses_computables, d.dias_faltas, d.monto_descuento_faltas, d.monto_cts, d.created_at,
		       t.apellido_paterno || ' ' || t.apellido_materno || ', ' || t.nombres AS trabajador_nombre,
		       t.numero_documento AS trabajador_documento
		FROM planilla_cts_detalles d
		INNER JOIN contratos c ON d.contrato_id = c.id
		INNER JOIN trabajadores t ON c.trabajador_id = t.id
		WHERE d.planilla_cts_id = $1
		ORDER BY t.apellido_paterno ASC
	`
	rows, err := r.db.Query(query, planillaCtsID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.PlanillaCtsDetalle
	for rows.Next() {
		var d models.PlanillaCtsDetalle
		err := rows.Scan(
			&d.ID, &d.PlanillaCtsID, &d.ContratoID, &d.SueldoBasico, &d.AsignacionFamilia,
			&d.SextoGratificacion, &d.PromedioVariables, &d.RemuneracionComputable,
			&d.MesesComputables, &d.DiasFaltas, &d.MontoDescuentoFaltas, &d.MontoCts, &d.CreatedAt,
			&d.TrabajadorNombre, &d.TrabajadorDocumento,
		)
		if err == nil {
			lista = append(lista, d)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return lista, nil
}

// ActualizarDetalleCts calcula e introduce ajustes individuales en caliente
func (r *CtsRepository) ActualizarDetalleCts(d *models.PlanillaCtsDetalle) error {
	query := `
		UPDATE planilla_cts_detalles
		SET sueldo_basico = $1, asignacion_familiar = $2, sexto_gratificacion = $3,
		    promedio_variables = $4, remuneracion_computable = $5, meses_computables = $6,
		    dias_faltas = $7, monto_descuento_faltas = $8, monto_cts = $9
		WHERE id = $10
	`
	_, err := r.db.Exec(query,
		d.SueldoBasico, d.AsignacionFamilia, d.SextoGratificacion,
		d.PromedioVariables, d.RemuneracionComputable, d.MesesComputables,
		d.DiasFaltas, d.MontoDescuentoFaltas, d.MontoCts, d.ID,
	)
	return err
}

// CambiarEstadoCts actualiza el estado de la planilla de CTS
func (r *CtsRepository) CambiarEstadoCts(id int, tenantID int, estado string) error {
	query := `UPDATE planillas_cts SET estado = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2 AND tenant_id = $3`
	_, err := r.db.Exec(query, estado, id, tenantID)
	return err
}

// EliminarPlanillaCts elimina un registro de CTS
func (r *CtsRepository) EliminarPlanillaCts(id int, tenantID int) error {
	query := `DELETE FROM planillas_cts WHERE id = $1 AND tenant_id = $2`
	_, err := r.db.Exec(query, id, tenantID)
	return err
}



// ObtenerContratosCtsEligibles obtiene todos los contratos activos del régimen DL 728 en un periodo dado
func (r *CtsRepository) ObtenerContratosCtsEligibles(tenantID int, desde time.Time, hasta time.Time) ([]models.ContratoPlanilla, error) {
	query := `
		SELECT c.id, c.puesto_id, rl.codigo AS regimen, COALESCE(t.regimen_pensionario, 'ONP'), COALESCE(t.afp_id, 0), COALESCE(t.afp_tipo_comision, ''),
		       c.fecha_inicio, c.fecha_fin,
		       t.apellido_paterno || ' ' || t.apellido_materno || ', ' || t.nombres AS trabajador_nombre,
		       t.numero_documento, p.nombre AS puesto_nombre, COALESCE(p.codigo_airhsp, ''),
		       COALESCE(org.documento_aprobacion, ''), COALESCE(uo.nombre, ''), COALESCE(uo.tipo, ''),
		       COALESCE(p.sueldo_presupuestado, 0)
		FROM contratos c
		INNER JOIN trabajadores t ON c.trabajador_id = t.id
		INNER JOIN puestos p ON c.puesto_id = p.id
		INNER JOIN regimenes_laborales rl ON p.regimen_id = rl.id
		LEFT JOIN unidades_organicas uo ON p.unidad_organica_id = uo.id
		LEFT JOIN organigramas org ON uo.organigrama_id = org.id
		WHERE c.tenant_id = $1
		  AND rl.codigo = '728'
		  AND c.fecha_inicio <= $2
		  AND (c.fecha_fin IS NULL OR c.fecha_fin >= $3)
		  AND c.activo = true
	`
	rows, err := r.db.Query(query, tenantID, hasta, desde)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.ContratoPlanilla
	for rows.Next() {
		var c models.ContratoPlanilla
		var fFin sql.NullTime
		err := rows.Scan(
			&c.ID, &c.PuestoID, &c.Regimen, &c.RegimenPensionario, &c.AfpID, &c.AfpTipoComision,
			&c.FechaInicio, &fFin,
			&c.TrabajadorNombreCompleto, &c.TrabajadorNumeroDocumento, &c.PuestoNombre, &c.PuestoCodigoAirhsp,
			&c.OrganigramaDocumentoAprobacion, &c.UnidadOrganicaNombre, &c.UnidadOrganicaTipo, &c.SueldoBasicoHistorico,
		)
		if err == nil {
			if fFin.Valid {
				c.FechaFin = &fFin.Time
			}
			lista = append(lista, c)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return lista, nil
}

// ObtenerSueldoBasicoActivo obtiene el sueldo básico oficial configurado para el puesto en puesto_conceptos
func (r *CtsRepository) ObtenerSueldoBasicoActivo(puestoID int, codigosSueldo []string) (float64, error) {
	query := `
		SELECT COALESCE(pc.monto, 0)
		FROM puesto_conceptos pc
		INNER JOIN conceptos_tenant ct ON pc.concepto_tenant_id = ct.id
		INNER JOIN conceptos_maestros cm ON ct.concepto_id = cm.id
		WHERE pc.puesto_id = $1 
		  AND pc.activo = true 
		  AND ct.activo = true 
		  AND (cm.codigo_interno = ANY($2) OR cm.codigo = ANY($2))
	`
	var monto float64
	err := r.db.QueryRow(query, puestoID, pq.Array(codigosSueldo)).Scan(&monto)
	if err == sql.ErrNoRows {
		return 0.0, nil
	}
	return monto, err
}

// ObtenerRemuneracionFamiliarActiva consulta si la asignación familiar está activa en la Plaza y calcula su valor dinámico (10% de la RMV)
func (r *CtsRepository) ObtenerRemuneracionFamiliarActiva(puestoID int, codigosAsigFam []string) (float64, error) {
	// 1. Verificar si está activo para el puesto
	queryActivo := `
		SELECT pc.id
		FROM puesto_conceptos pc
		INNER JOIN conceptos_tenant ct ON pc.concepto_tenant_id = ct.id
		INNER JOIN conceptos_maestros cm ON ct.concepto_id = cm.id
		WHERE pc.puesto_id = $1 
		  AND pc.activo = true 
		  AND ct.activo = true 
		  AND (cm.codigo_interno = ANY($2) OR cm.codigo = ANY($2))
	`
	var id int
	err := r.db.QueryRow(queryActivo, puestoID, pq.Array(codigosAsigFam)).Scan(&id)
	if err == sql.ErrNoRows {
		return 0.0, nil
	}
	if err != nil {
		return 0.0, err
	}

	// 2. Si está activo, obtener el RMV vigente (el último valor parametrizado)
	queryRMV := `
		SELECT valor 
		FROM parametros_globales 
		WHERE clave = 'RMV' 
		ORDER BY fecha_desde DESC 
		LIMIT 1
	`
	var rmv float64
	err = r.db.QueryRow(queryRMV).Scan(&rmv)
	if err != nil {
		rmv = 1025.00 // Fallback
	}

	return math.Round((rmv*0.10)*100) / 100, nil
}

// ObtenerGratificacionHistorica obtiene la gratificación pagada en un año/mes específico
func (r *CtsRepository) ObtenerGratificacionHistorica(contratoID int, anio int, mes int, codigosGrati []string) (float64, error) {
	query := `
		SELECT COALESCE(SUM(pc.monto), 0)
		FROM planilla_conceptos pc
		INNER JOIN planilla_detalles pd ON pc.planilla_detalle_id = pd.id
		INNER JOIN planillas p ON pd.planilla_id = p.id
		INNER JOIN conceptos_tenant ct ON pc.concepto_tenant_id = ct.id
		INNER JOIN conceptos_maestros cm ON ct.concepto_id = cm.id
		WHERE pd.contrato_id = $1
		  AND p.anio = $2
		  AND p.mes = $3
		  AND (cm.codigo_interno = ANY($4) OR pc.codigo_sunat = ANY($4))
	`
	var suma float64
	err := r.db.QueryRow(query, contratoID, anio, mes, pq.Array(codigosGrati)).Scan(&suma)
	return suma, err
}

type ConceptoVariable struct {
	MaestroID int
	Monto     float64
	Mes       int
}

// ObtenerVariablesSemestre recupera los montos de variables para el cálculo de CTS
func (r *CtsRepository) ObtenerVariablesSemestre(contratoID int, anio1 int, meses1 []int, anio2 int, meses2 []int, codigosExcluidos []string) ([]ConceptoVariable, error) {
	query := `
		SELECT pc.maestro_id, pc.monto, p.mes
		FROM planilla_conceptos pc
		INNER JOIN planilla_detalles pd ON pc.planilla_detalle_id = pd.id
		INNER JOIN planillas p ON pd.planilla_id = p.id
		INNER JOIN conceptos_tenant ct ON pc.concepto_tenant_id = ct.id
		INNER JOIN conceptos_maestros cm ON ct.concepto_id = cm.id
		WHERE pd.contrato_id = $1
		  AND ct.es_base_cts = true
		  AND pc.tipo_concepto = 'INGRESO'
		  AND NOT (cm.codigo_interno = ANY($6) OR cm.codigo = ANY($6) OR pc.codigo_sunat = ANY($6))
		  AND (
		      (p.anio = $2 AND p.mes = ANY($3))
		   OR (p.anio = $4 AND p.mes = ANY($5))
		  )
	`
	rows, err := r.db.Query(query, contratoID, anio1, pq.Array(meses1), anio2, pq.Array(meses2), pq.Array(codigosExcluidos))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []ConceptoVariable
	for rows.Next() {
		var v ConceptoVariable
		err := rows.Scan(&v.MaestroID, &v.Monto, &v.Mes)
		if err == nil {
			lista = append(lista, v)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return lista, nil
}

// ObtenerInasistenciasSemestre cuenta los días de faltas injustificadas en el semestre
func (r *CtsRepository) ObtenerInasistenciasSemestre(contratoID int, desde time.Time, hasta time.Time) (int, error) {
	query := `
		SELECT COALESCE(SUM(cantidad), 0)
		FROM ocurrencias_asistencia
		WHERE contrato_id = $1
		  AND fecha_ocurrencia >= $2
		  AND fecha_ocurrencia <= $3
		  AND tipo = 'INASISTENCIA'
	`
	var total int
	err := r.db.QueryRow(query, contratoID, desde, hasta).Scan(&total)
	return total, err
}
