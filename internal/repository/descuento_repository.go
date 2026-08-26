package repository

import (
	"database/sql"
	"fmt"
	"log"
	"planilla-rgm/internal/models"
	"strings"
	"time"

	"github.com/lib/pq"
)

type DescuentoRepository struct {
	db *sql.DB
}

func NewDescuentoRepository(db *sql.DB) *DescuentoRepository {
	return &DescuentoRepository{db: db}
}

// ListarPaginado obtiene el listado paginado y filtrado de descuentos de un tenant
func (r *DescuentoRepository) ListarPaginado(tenantID int, filtro models.DescuentoFiltroDTO, limite int, offset int) ([]models.Descuento, int, error) {
	var whereClauses []string
	var args []interface{}
	argIdx := 1

	whereClauses = append(whereClauses, fmt.Sprintf("d.tenant_id = $%d", argIdx))
	args = append(args, tenantID)
	argIdx++

	if filtro.TrabajadorID != nil && *filtro.TrabajadorID > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("d.trabajador_id = $%d", argIdx))
		args = append(args, *filtro.TrabajadorID)
		argIdx++
	}

	if filtro.TipoDescuento != "" && filtro.TipoDescuento != "TODOS" {
		whereClauses = append(whereClauses, fmt.Sprintf("d.tipo_descuento = $%d", argIdx))
		args = append(args, filtro.TipoDescuento)
		argIdx++
	}

	if filtro.Estado == "ACTIVOS" {
		whereClauses = append(whereClauses, "d.activo = true")
	} else if filtro.Estado == "INACTIVOS" {
		whereClauses = append(whereClauses, "d.activo = false")
	}

	if strings.TrimSpace(filtro.Busqueda) != "" {
		patron := "%" + strings.ToLower(strings.TrimSpace(filtro.Busqueda)) + "%"
		whereClauses = append(whereClauses, fmt.Sprintf(`(
			LOWER(t.nombres || ' ' || t.apellido_paterno || ' ' || t.apellido_materno) LIKE $%d OR
			LOWER(t.numero_documento) LIKE $%d OR
			LOWER(d.detalle_documento) LIKE $%d OR
			LOWER(d.descripcion) LIKE $%d OR
			LOWER(d.beneficiario_nombre) LIKE $%d
		)`, argIdx, argIdx, argIdx, argIdx, argIdx))
		args = append(args, patron)
		argIdx++
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	// Conteo total
	countQuery := fmt.Sprintf(`
		SELECT COUNT(d.id)
		FROM descuentos d
		INNER JOIN trabajadores t ON d.trabajador_id = t.id
		WHERE %s
	`, whereSQL)

	var totalRegistros int
	err := r.db.QueryRow(countQuery, args...).Scan(&totalRegistros)
	if err != nil {
		return nil, 0, err
	}

	// Consulta de registros
	dataQuery := fmt.Sprintf(`
		SELECT 
			d.id, d.tenant_id, d.trabajador_id, d.concepto_tenant_id,
			d.tipo_descuento, d.documento_ordenador, COALESCE(d.detalle_documento, ''),
			d.descripcion, d.tipo_calculo, d.base_calculo, d.porcentaje, d.monto_fijo,
			d.monto_total_deuda, d.monto_acumulado, d.cuotas_totales, d.cuota_actual,
			d.inicio_vigencia, d.fin_vigencia, d.activo, COALESCE(d.motivo_baja, ''),
			COALESCE(d.beneficiario_tipo_documento, 'DNI'), COALESCE(d.beneficiario_numero_documento, ''),
			COALESCE(d.beneficiario_nombre, ''), d.entidad_financiera_id,
			COALESCE(d.beneficiario_cuenta, ''), COALESCE(d.beneficiario_cci, ''),
			d.created_at, d.updated_at,
			(t.apellido_paterno || ' ' || t.apellido_materno || ', ' || t.nombres) AS trabajador_nombre,
			t.numero_documento AS trabajador_doc,
			ct.nombre_personalizado AS concepto_nombre,
			cm.codigo AS concepto_sunat,
			COALESCE(ef.nombre, '') AS entidad_financiera_nombre,
			COALESCE(
				(SELECT STRING_AGG(ct_sub.nombre_personalizado, ', ')
				 FROM descuento_conceptos dc_sub
				 INNER JOIN conceptos_tenant ct_sub ON dc_sub.concepto_tenant_id = ct_sub.id
				 WHERE dc_sub.descuento_id = d.id),
				'Todos los haberes afectos'
			) AS conceptos_afectos_nombres
		FROM descuentos d
		INNER JOIN trabajadores t ON d.trabajador_id = t.id
		INNER JOIN conceptos_tenant ct ON d.concepto_tenant_id = ct.id
		INNER JOIN conceptos_maestros cm ON ct.concepto_id = cm.id
		LEFT JOIN entidades_financieras ef ON d.entidad_financiera_id = ef.id
		WHERE %s
		ORDER BY d.activo DESC, d.id DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argIdx, argIdx+1)

	args = append(args, limite, offset)

	rows, err := r.db.Query(dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var lista []models.Descuento
	for rows.Next() {
		var d models.Descuento
		var finVig sql.NullTime
		var entFinID sql.NullInt64

		err := rows.Scan(
			&d.ID, &d.TenantID, &d.TrabajadorID, &d.ConceptoTenantID,
			&d.TipoDescuento, &d.DocumentoOrdenador, &d.DetalleDocumento,
			&d.Descripcion, &d.TipoCalculo, &d.BaseCalculo, &d.Porcentaje, &d.MontoFijo,
			&d.MontoTotalDeuda, &d.MontoAcumulado, &d.CuotasTotales, &d.CuotaActual,
			&d.InicioVigencia, &finVig, &d.Activo, &d.MotivoBaja,
			&d.BeneficiarioTipoDocumento, &d.BeneficiarioNumeroDocumento,
			&d.BeneficiarioNombre, &entFinID,
			&d.BeneficiarioCuenta, &d.BeneficiarioCCI,
			&d.CreatedAt, &d.UpdatedAt,
			&d.TrabajadorNombreCompleto, &d.TrabajadorNumeroDocumento,
			&d.ConceptoNombre, &d.ConceptoCodigoSunat,
			&d.EntidadFinancieraNombre, &d.ConceptosAfectosNombres,
		)
		if err != nil {
			log.Println("Error escaneando descuento:", err)
			return nil, 0, err
		}

		if finVig.Valid {
			tVal := finVig.Time
			d.FinVigencia = &tVal
		}
		if entFinID.Valid {
			idVal := int(entFinID.Int64)
			d.EntidadFinancieraID = &idVal
		}

		lista = append(lista, d)
	}

	return lista, totalRegistros, nil
}

// ObtenerKPIs calcula los contadores clave para las tarjetas Bento Grid
func (r *DescuentoRepository) ObtenerKPIs(tenantID int) (models.DescuentoResumenKPI, error) {
	query := `
		SELECT 
			COUNT(id) FILTER (WHERE activo = true) AS total_activos,
			COUNT(id) FILTER (WHERE activo = true AND tipo_descuento = 'JUDICIAL') AS total_judiciales,
			COUNT(id) FILTER (WHERE activo = true AND tipo_descuento = 'SINDICAL') AS total_sindicales,
			COUNT(id) FILTER (WHERE activo = true AND tipo_descuento IN ('PRESTAMO', 'CONVENIO')) AS total_prestamos,
			COUNT(DISTINCT trabajador_id) FILTER (WHERE activo = true) AS total_trabajadores
		FROM descuentos
		WHERE tenant_id = $1
	`
	var kpi models.DescuentoResumenKPI
	err := r.db.QueryRow(query, tenantID).Scan(
		&kpi.TotalActivos,
		&kpi.TotalJudiciales,
		&kpi.TotalSindicales,
		&kpi.TotalPrestamos,
		&kpi.TotalTrabajadores,
	)
	if err != nil {
		return kpi, err
	}
	return kpi, nil
}

// ObtenerPorID recupera un descuento por ID junto con los IDs de sus conceptos afectos
func (r *DescuentoRepository) ObtenerPorID(id int, tenantID int) (*models.Descuento, error) {
	query := `
		SELECT 
			d.id, d.tenant_id, d.trabajador_id, d.concepto_tenant_id,
			d.tipo_descuento, d.documento_ordenador, COALESCE(d.detalle_documento, ''),
			d.descripcion, d.tipo_calculo, d.base_calculo, d.porcentaje, d.monto_fijo,
			d.monto_total_deuda, d.monto_acumulado, d.cuotas_totales, d.cuota_actual,
			d.inicio_vigencia, d.fin_vigencia, d.activo, COALESCE(d.motivo_baja, ''),
			COALESCE(d.beneficiario_tipo_documento, 'DNI'), COALESCE(d.beneficiario_numero_documento, ''),
			COALESCE(d.beneficiario_nombre, ''), d.entidad_financiera_id,
			COALESCE(d.beneficiario_cuenta, ''), COALESCE(d.beneficiario_cci, ''),
			d.created_at, d.updated_at,
			(t.apellido_paterno || ' ' || t.apellido_materno || ', ' || t.nombres) AS trabajador_nombre,
			t.numero_documento AS trabajador_doc,
			ct.nombre_personalizado AS concepto_nombre,
			cm.codigo AS concepto_sunat,
			COALESCE(ef.nombre, '') AS entidad_financiera_nombre
		FROM descuentos d
		INNER JOIN trabajadores t ON d.trabajador_id = t.id
		INNER JOIN conceptos_tenant ct ON d.concepto_tenant_id = ct.id
		INNER JOIN conceptos_maestros cm ON ct.concepto_id = cm.id
		LEFT JOIN entidades_financieras ef ON d.entidad_financiera_id = ef.id
		WHERE d.id = $1 AND d.tenant_id = $2
	`
	var d models.Descuento
	var finVig sql.NullTime
	var entFinID sql.NullInt64

	err := r.db.QueryRow(query, id, tenantID).Scan(
		&d.ID, &d.TenantID, &d.TrabajadorID, &d.ConceptoTenantID,
		&d.TipoDescuento, &d.DocumentoOrdenador, &d.DetalleDocumento,
		&d.Descripcion, &d.TipoCalculo, &d.BaseCalculo, &d.Porcentaje, &d.MontoFijo,
		&d.MontoTotalDeuda, &d.MontoAcumulado, &d.CuotasTotales, &d.CuotaActual,
		&d.InicioVigencia, &finVig, &d.Activo, &d.MotivoBaja,
		&d.BeneficiarioTipoDocumento, &d.BeneficiarioNumeroDocumento,
		&d.BeneficiarioNombre, &entFinID,
		&d.BeneficiarioCuenta, &d.BeneficiarioCCI,
		&d.CreatedAt, &d.UpdatedAt,
		&d.TrabajadorNombreCompleto, &d.TrabajadorNumeroDocumento,
		&d.ConceptoNombre, &d.ConceptoCodigoSunat,
		&d.EntidadFinancieraNombre,
	)
	if err != nil {
		return nil, err
	}

	if finVig.Valid {
		tVal := finVig.Time
		d.FinVigencia = &tVal
	}
	if entFinID.Valid {
		idVal := int(entFinID.Int64)
		d.EntidadFinancieraID = &idVal
	}

	// Cargar conceptos afectos
	rowsC, err := r.db.Query(`SELECT concepto_tenant_id FROM descuento_conceptos WHERE descuento_id = $1`, d.ID)
	if err == nil {
		for rowsC.Next() {
			var cID int
			if err := rowsC.Scan(&cID); err == nil {
				d.ConceptosAfectosIDs = append(d.ConceptosAfectosIDs, cID)
			}
		}
		rowsC.Close()
	}

	return &d, nil
}

// Crear inserta un nuevo descuento y sus conceptos afectos de forma transaccional
func (r *DescuentoRepository) Crear(d *models.Descuento, conceptosIDs []int) (int, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO descuentos (
			tenant_id, trabajador_id, concepto_tenant_id, tipo_descuento,
			documento_ordenador, detalle_documento, descripcion,
			tipo_calculo, base_calculo, porcentaje, monto_fijo,
			monto_total_deuda, monto_acumulado, cuotas_totales, cuota_actual,
			inicio_vigencia, fin_vigencia, activo, motivo_baja,
			beneficiario_tipo_documento, beneficiario_numero_documento,
			beneficiario_nombre, entidad_financiera_id, beneficiario_cuenta, beneficiario_cci
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25
		) RETURNING id
	`

	var descuentoID int
	err = tx.QueryRow(query,
		d.TenantID, d.TrabajadorID, d.ConceptoTenantID, d.TipoDescuento,
		d.DocumentoOrdenador, d.DetalleDocumento, d.Descripcion,
		d.TipoCalculo, d.BaseCalculo, d.Porcentaje, d.MontoFijo,
		d.MontoTotalDeuda, d.MontoAcumulado, d.CuotasTotales, d.CuotaActual,
		d.InicioVigencia, d.FinVigencia, d.Activo, d.MotivoBaja,
		d.BeneficiarioTipoDocumento, d.BeneficiarioNumeroDocumento,
		d.BeneficiarioNombre, d.EntidadFinancieraID, d.BeneficiarioCuenta, d.BeneficiarioCCI,
	).Scan(&descuentoID)
	if err != nil {
		return 0, fmt.Errorf("error insertando cabecera de descuento: %w", err)
	}

	for _, cid := range conceptosIDs {
		if cid <= 0 {
			continue
		}
		_, err = tx.Exec(`INSERT INTO descuento_conceptos (descuento_id, concepto_tenant_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, descuentoID, cid)
		if err != nil {
			return 0, fmt.Errorf("error asociando concepto afecto: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return descuentoID, nil
}

// Actualizar modifica un descuento existente y recrea sus conceptos afectos
func (r *DescuentoRepository) Actualizar(d *models.Descuento, conceptosIDs []int) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		UPDATE descuentos SET
			concepto_tenant_id = $1,
			tipo_descuento = $2,
			documento_ordenador = $3,
			detalle_documento = $4,
			descripcion = $5,
			tipo_calculo = $6,
			base_calculo = $7,
			porcentaje = $8,
			monto_fijo = $9,
			monto_total_deuda = $10,
			monto_acumulado = $11,
			cuotas_totales = $12,
			cuota_actual = $13,
			inicio_vigencia = $14,
			fin_vigencia = $15,
			activo = $16,
			motivo_baja = $17,
			beneficiario_tipo_documento = $18,
			beneficiario_numero_documento = $19,
			beneficiario_nombre = $20,
			entidad_financiera_id = $21,
			beneficiario_cuenta = $22,
			beneficiario_cci = $23,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $24 AND tenant_id = $25
	`

	res, err := tx.Exec(query,
		d.ConceptoTenantID, d.TipoDescuento, d.DocumentoOrdenador, d.DetalleDocumento,
		d.Descripcion, d.TipoCalculo, d.BaseCalculo, d.Porcentaje, d.MontoFijo,
		d.MontoTotalDeuda, d.MontoAcumulado, d.CuotasTotales, d.CuotaActual,
		d.InicioVigencia, d.FinVigencia, d.Activo, d.MotivoBaja,
		d.BeneficiarioTipoDocumento, d.BeneficiarioNumeroDocumento,
		d.BeneficiarioNombre, d.EntidadFinancieraID, d.BeneficiarioCuenta, d.BeneficiarioCCI,
		d.ID, d.TenantID,
	)
	if err != nil {
		return fmt.Errorf("error actualizando descuento: %w", err)
	}
	filas, _ := res.RowsAffected()
	if filas == 0 {
		return fmt.Errorf("descuento no encontrado o sin permisos")
	}

	// Reemplazar conceptos afectos
	_, err = tx.Exec(`DELETE FROM descuento_conceptos WHERE descuento_id = $1`, d.ID)
	if err != nil {
		return err
	}

	for _, cid := range conceptosIDs {
		if cid <= 0 {
			continue
		}
		_, err = tx.Exec(`INSERT INTO descuento_conceptos (descuento_id, concepto_tenant_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, d.ID, cid)
		if err != nil {
			return fmt.Errorf("error asociando concepto afecto: %w", err)
		}
	}

	return tx.Commit()
}

// ToggleActivo cambia el estado de activo con registro de motivo de baja
func (r *DescuentoRepository) ToggleActivo(id int, tenantID int, activo bool, motivoBaja string) error {
	query := `
		UPDATE descuentos SET
			activo = $1,
			motivo_baja = CASE WHEN $1 = false THEN $2 ELSE '' END,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $3 AND tenant_id = $4
	`
	_, err := r.db.Exec(query, activo, motivoBaja, id, tenantID)
	return err
}

// Eliminar borra un descuento
func (r *DescuentoRepository) Eliminar(id int, tenantID int) error {
	query := `DELETE FROM descuentos WHERE id = $1 AND tenant_id = $2`
	res, err := r.db.Exec(query, id, tenantID)
	if err != nil {
		return err
	}
	filas, _ := res.RowsAffected()
	if filas == 0 {
		return fmt.Errorf("descuento no encontrado o no pertenece a la entidad")
	}
	return nil
}

// ObtenerDescuentosActivosPorTrabajadorMasivo recupera los descuentos activos y vigentes para un lote de trabajadores en un mes específico
func (r *DescuentoRepository) ObtenerDescuentosActivosPorTrabajadorMasivo(tenantID int, trabajadorIDs []int, anio int, mes int) (map[int][]models.DescuentoConConceptos, error) {
	resultado := make(map[int][]models.DescuentoConConceptos)
	if len(trabajadorIDs) == 0 {
		return resultado, nil
	}

	primerDia := time.Date(anio, time.Month(mes), 1, 0, 0, 0, 0, time.UTC)
	ultimoDia := primerDia.AddDate(0, 1, -1)

	query := `
		SELECT 
			d.id, d.tenant_id, d.trabajador_id, d.concepto_tenant_id,
			d.tipo_descuento, d.documento_ordenador, COALESCE(d.detalle_documento, ''),
			d.descripcion, d.tipo_calculo, d.base_calculo, d.porcentaje, d.monto_fijo,
			d.monto_total_deuda, d.monto_acumulado, d.cuotas_totales, d.cuota_actual,
			d.inicio_vigencia, d.fin_vigencia, d.activo,
			ct.nombre_personalizado AS concepto_nombre,
			cm.codigo AS concepto_sunat,
			cm.id AS concepto_maestro_id
		FROM descuentos d
		INNER JOIN conceptos_tenant ct ON d.concepto_tenant_id = ct.id
		INNER JOIN conceptos_maestros cm ON ct.concepto_id = cm.id
		WHERE d.tenant_id = $1 
		  AND d.trabajador_id = ANY($2)
		  AND d.activo = true
		  AND d.inicio_vigencia <= $3
		  AND (d.fin_vigencia IS NULL OR d.fin_vigencia >= $4)
		ORDER BY d.id ASC
	`

	rows, err := r.db.Query(query, tenantID, pq.Array(trabajadorIDs), ultimoDia, primerDia)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var descuentosList []models.Descuento
	var descuentoIDs []int

	for rows.Next() {
		var d models.Descuento
		var finVig sql.NullTime

		err := rows.Scan(
			&d.ID, &d.TenantID, &d.TrabajadorID, &d.ConceptoTenantID,
			&d.TipoDescuento, &d.DocumentoOrdenador, &d.DetalleDocumento,
			&d.Descripcion, &d.TipoCalculo, &d.BaseCalculo, &d.Porcentaje, &d.MontoFijo,
			&d.MontoTotalDeuda, &d.MontoAcumulado, &d.CuotasTotales, &d.CuotaActual,
			&d.InicioVigencia, &finVig, &d.Activo,
			&d.ConceptoNombre, &d.ConceptoCodigoSunat,
			&d.ConceptoMaestroID,
		)
		if err != nil {
			return nil, err
		}
		if finVig.Valid {
			tVal := finVig.Time
			d.FinVigencia = &tVal
		}

		descuentosList = append(descuentosList, d)
		descuentoIDs = append(descuentoIDs, d.ID)
	}

	// Obtener mapa de conceptos afectos para todos los descuentos encontrados
	mapaConceptosAfectos := make(map[int][]int)
	if len(descuentoIDs) > 0 {
		cQuery := `SELECT descuento_id, concepto_tenant_id FROM descuento_conceptos WHERE descuento_id = ANY($1)`
		cRows, err := r.db.Query(cQuery, pq.Array(descuentoIDs))
		if err == nil {
			for cRows.Next() {
				var descID, cTenantID int
				if err := cRows.Scan(&descID, &cTenantID); err == nil {
					mapaConceptosAfectos[descID] = append(mapaConceptosAfectos[descID], cTenantID)
				}
			}
			cRows.Close()
		}
	}

	for _, d := range descuentosList {
		afectos := mapaConceptosAfectos[d.ID]
		resultado[d.TrabajadorID] = append(resultado[d.TrabajadorID], models.DescuentoConConceptos{
			Descuento:          d,
			ConceptosTenantIDs: afectos,
		})
	}

	return resultado, nil
}

// ObtenerInfoTrabajadorPuesto obtiene el puesto activo, régimen y los conceptos de ingreso asignados al puesto del trabajador
func (r *DescuentoRepository) ObtenerInfoTrabajadorPuesto(tenantID int, trabajadorID int) (*models.InfoTrabajadorPuesto, error) {
	info := &models.InfoTrabajadorPuesto{
		TrabajadorID: trabajadorID,
		Conceptos:    make([]models.ConceptoPuestoDTO, 0),
	}

	if trabajadorID == 0 {
		return info, nil
	}

	// 1. Buscar contrato activo del trabajador
	queryContrato := `
		SELECT c.id, c.puesto_id, p.nombre AS puesto_nombre, rl.id, rl.codigo, rl.descripcion
		FROM contratos c
		INNER JOIN puestos p ON c.puesto_id = p.id
		INNER JOIN regimenes_laborales rl ON p.regimen_id = rl.id
		WHERE c.trabajador_id = $1 AND c.tenant_id = $2 AND c.activo = true
		ORDER BY c.fecha_inicio DESC
		LIMIT 1
	`
	var contratoID, puestoID, regimenID int
	var puestoNombre, regimenCodigo, regimenNombre string
	err := r.db.QueryRow(queryContrato, trabajadorID, tenantID).Scan(
		&contratoID, &puestoID, &puestoNombre, &regimenID, &regimenCodigo, &regimenNombre,
	)
	if err != nil {
		log.Printf("ObtenerInfoTrabajadorPuesto: búsqueda de contrato para trabajador %d (tenant %d): %v", trabajadorID, tenantID, err)
	} else {
		info.TieneContrato = true
		info.ContratoID = contratoID
		info.PuestoID = puestoID
		info.PuestoNombre = puestoNombre
		info.RegimenID = regimenID
		info.RegimenCodigo = regimenCodigo
		info.RegimenNombre = regimenNombre

		// 2. Obtener conceptos de ingreso asignados a este puesto
		queryConceptosPuesto := `
			SELECT ct.id, ct.concepto_id, cm.codigo, ct.nombre_personalizado, COALESCE(pc.monto, 0.00)
			FROM puesto_conceptos pc
			INNER JOIN conceptos_tenant ct ON pc.concepto_tenant_id = ct.id AND ct.activo = true
			INNER JOIN conceptos_maestros cm ON ct.concepto_id = cm.id AND UPPER(cm.tipo) = 'INGRESO'
			WHERE pc.puesto_id = $1 AND pc.activo = true
			ORDER BY cm.codigo ASC, ct.nombre_personalizado ASC
		`
		rows, err := r.db.Query(queryConceptosPuesto, puestoID)
		if err == nil {
			for rows.Next() {
				var cp models.ConceptoPuestoDTO
				if err := rows.Scan(&cp.ConceptoTenantID, &cp.ConceptoID, &cp.ConceptoCodigo, &cp.NombrePersonalizado, &cp.Monto); err == nil {
					info.Conceptos = append(info.Conceptos, cp)
				}
			}
			rows.Close()
		}
	}

	// 3. Fallback: Si el puesto no tiene conceptos en puesto_conceptos o el trabajador no tiene contrato activo
	if len(info.Conceptos) == 0 {
		queryFallback := `
			SELECT ct.id, ct.concepto_id, cm.codigo, ct.nombre_personalizado, 0.00
			FROM conceptos_tenant ct
			INNER JOIN conceptos_maestros cm ON ct.concepto_id = cm.id AND UPPER(cm.tipo) = 'INGRESO'
			WHERE ct.tenant_id = $1 AND ct.activo = true
			ORDER BY cm.codigo ASC, ct.nombre_personalizado ASC
		`
		rows, err := r.db.Query(queryFallback, tenantID)
		if err == nil {
			for rows.Next() {
				var cp models.ConceptoPuestoDTO
				if err := rows.Scan(&cp.ConceptoTenantID, &cp.ConceptoID, &cp.ConceptoCodigo, &cp.NombrePersonalizado, &cp.Monto); err == nil {
					info.Conceptos = append(info.Conceptos, cp)
				}
			}
			rows.Close()
		}
	}

	return info, nil
}

// ObtenerConceptosIngresoPorTrabajador lista los conceptos de tipo INGRESO activos para asociar a la base
func (r *DescuentoRepository) ObtenerConceptosIngresoPorTrabajador(tenantID int, trabajadorID int) ([]models.ConceptoTenant, error) {
	query := `
		SELECT DISTINCT ct.id, ct.concepto_id, ct.nombre_personalizado, cm.codigo, cm.tipo
		FROM conceptos_tenant ct
		INNER JOIN conceptos_maestros cm ON ct.concepto_id = cm.id
		WHERE ct.tenant_id = $1 AND ct.activo = true AND UPPER(cm.tipo) = 'INGRESO'
		ORDER BY ct.nombre_personalizado ASC
	`
	rows, err := r.db.Query(query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.ConceptoTenant
	for rows.Next() {
		var c models.ConceptoTenant
		if err := rows.Scan(&c.ID, &c.ConceptoID, &c.NombrePersonalizado, &c.ConceptoCodigo, &c.ConceptoTipo); err != nil {
			return nil, err
		}
		lista = append(lista, c)
	}
	return lista, nil
}

// ObtenerConceptosRetencionTenant lista los conceptos de tipo RETENCION configurados en el tenant para la salida en boleta
func (r *DescuentoRepository) ObtenerConceptosRetencionTenant(tenantID int) ([]models.ConceptoTenant, error) {
	query := `
		SELECT ct.id, ct.concepto_id, ct.nombre_personalizado, cm.codigo, cm.tipo
		FROM conceptos_tenant ct
		INNER JOIN conceptos_maestros cm ON ct.concepto_id = cm.id
		WHERE ct.tenant_id = $1 AND ct.activo = true AND UPPER(cm.tipo) = 'RETENCION'
		ORDER BY ct.nombre_personalizado ASC
	`
	rows, err := r.db.Query(query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.ConceptoTenant
	for rows.Next() {
		var c models.ConceptoTenant
		if err := rows.Scan(&c.ID, &c.ConceptoID, &c.NombrePersonalizado, &c.ConceptoCodigo, &c.ConceptoTipo); err != nil {
			return nil, err
		}
		lista = append(lista, c)
	}
	return lista, nil
}
