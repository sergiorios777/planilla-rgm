package repository

import (
	"database/sql"
	"fmt"
	"math"
	"planilla-rgm/internal/models"
	"strings"
)

type PlameRepository struct {
	db *sql.DB
}

func NewPlameRepository(db *sql.DB) *PlameRepository {
	return &PlameRepository{db: db}
}

func (r *PlameRepository) GetDB() *sql.DB {
	return r.db
}

// ObtenerRucTenant obtiene el RUC configurado para la institución (tenant)
func (r *PlameRepository) ObtenerRucTenant(tenantID int) (string, error) {
	var ruc sql.NullString
	err := r.db.QueryRow(`SELECT ruc FROM tenants WHERE id = $1`, tenantID).Scan(&ruc)
	if err != nil {
		return "20000000001", err
	}
	if ruc.Valid && strings.TrimSpace(ruc.String) != "" {
		return strings.TrimSpace(ruc.String), nil
	}
	return "20000000001", nil
}

// ObtenerMaestrosSunat obtiene el catálogo completo de conceptos oficiales de SUNAT (Tabla 22)
func (r *PlameRepository) ObtenerMaestrosSunat() ([]models.ConceptoMaestro, error) {
	query := `
		SELECT 
			id, 
			parent_id, 
			codigo, 
			COALESCE(codigo_interno, '') as codigo_interno, 
			descripcion, 
			tipo, 
			activo, 
			origen
		FROM conceptos_maestros 
		WHERE origen = 'sunat'
		ORDER BY 
			CASE tipo 
				WHEN 'INGRESO' THEN 1 
				WHEN 'DESCUENTO' THEN 2 
				WHEN 'APORTE_TRABAJADOR' THEN 3 
				WHEN 'APORTE_EMPLEADOR' THEN 4 
				ELSE 5 
			END, 
			codigo ASC
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.ConceptoMaestro
	for rows.Next() {
		var m models.ConceptoMaestro
		var parentID sql.NullInt64
		err := rows.Scan(
			&m.ID,
			&parentID,
			&m.Codigo,
			&m.CodigoInterno,
			&m.Descripcion,
			&m.Tipo,
			&m.Activo,
			&m.Origen,
		)
		if err != nil {
			return nil, err
		}
		if parentID.Valid {
			pID := int(parentID.Int64)
			m.ParentID = &pID
		}
		lista = append(lista, m)
	}
	return lista, nil
}

// ObtenerPeriodoPlanillasPlame retorna todas las planillas calculadas del mes con sus métricas de snapshot PLAME
func (r *PlameRepository) ObtenerPeriodoPlanillasPlame(tenantID, anio, mes int) ([]models.PlamePlanillaResumenItem, error) {
	query := `
		SELECT 
			p.id,
			p.anio,
			p.mes,
			COALESCE(p.tipo, 'ORDINARIA') AS tipo_planilla,
			COALESCE(p.descripcion, '') AS descripcion,
			p.estado,
			COUNT(DISTINCT pd.id) AS total_trabajadores,
			COALESCE(SUM(ppc.monto_devengado), 0) AS total_devengado,
			COALESCE(SUM(ppc.monto_pagado), 0) AS total_pagado,
			COUNT(ppc.id) > 0 AS tiene_snapshot,
			COALESCE(BOOL_OR(ppc.es_ajuste_manual), false) AS tiene_ajustes_manuales,
			COALESCE(BOOL_OR(ppc.es_concepto_vacacional OR ppc.codigo_sunat IN ('0118', '2007', '2043', '2049')), false) AS tiene_rem_vacacional,
			COUNT(DISTINCT ppc.codigo_sunat) AS total_conceptos_sunat
		FROM planillas p
		LEFT JOIN planilla_detalles pd ON pd.planilla_id = p.id
		LEFT JOIN planilla_plame_conceptos ppc ON ppc.planilla_id = p.id
		WHERE p.tenant_id = $1 AND p.anio = $2 AND p.mes = $3
		GROUP BY p.id, p.anio, p.mes, p.tipo, p.descripcion, p.estado
		ORDER BY p.id ASC
	`
	rows, err := r.db.Query(query, tenantID, anio, mes)
	if err != nil {
		return nil, fmt.Errorf("error consultando planillas del periodo PLAME: %w", err)
	}
	defer rows.Close()

	var lista []models.PlamePlanillaResumenItem
	for rows.Next() {
		var item models.PlamePlanillaResumenItem
		err := rows.Scan(
			&item.PlanillaID,
			&item.Anio,
			&item.Mes,
			&item.TipoPlanilla,
			&item.Descripcion,
			&item.EstadoPlanilla,
			&item.TotalTrabajadores,
			&item.TotalDevengado,
			&item.TotalPagado,
			&item.TieneSnapshot,
			&item.TieneAjustesManuales,
			&item.TieneRemVacacional,
			&item.TotalConceptosSunat,
		)
		if err != nil {
			return nil, fmt.Errorf("error escaneando planilla de periodo PLAME: %w", err)
		}
		lista = append(lista, item)
	}
	return lista, nil
}

// ObtenerResumenPeriodoPlame calcula las métricas globales y alertas del periodo mensual para el Hub
func (r *PlameRepository) ObtenerResumenPeriodoPlame(tenantID, anio, mes int) (models.PlameHubResumen, error) {
	var res models.PlameHubResumen
	res.Anio = anio
	res.Mes = mes

	// 1. Estadísticas de planillas y totales devengados/pagados
	err := r.db.QueryRow(`
		SELECT 
			COUNT(DISTINCT p.id),
			COUNT(DISTINCT c.trabajador_id),
			COALESCE(SUM(ppc.monto_devengado), 0),
			COALESCE(SUM(ppc.monto_pagado), 0),
			COUNT(DISTINCT CASE WHEN ppc.id IS NOT NULL THEN p.id END),
			COUNT(DISTINCT CASE WHEN ppc.es_ajuste_manual = true THEN p.id END)
		FROM planillas p
		LEFT JOIN planilla_detalles pd ON pd.planilla_id = p.id
		LEFT JOIN contratos c ON pd.contrato_id = c.id
		LEFT JOIN planilla_plame_conceptos ppc ON ppc.planilla_id = p.id
		WHERE p.tenant_id = $1 AND p.anio = $2 AND p.mes = $3
	`, tenantID, anio, mes).Scan(
		&res.TotalPlanillas,
		&res.TotalTrabajadores,
		&res.TotalDevengado,
		&res.TotalPagado,
		&res.PlanillasListas,
		&res.PlanillasConAjustes,
	)
	if err != nil && err != sql.ErrNoRows {
		return res, err
	}

	// 2. Incidencias de descanso vacacional y licencias
	licRepo := NewLicenciaVacacionRepository(r.db)
	mapaIncs, _ := licRepo.ObtenerIncidenciasMes(tenantID, anio, mes)
	for _, incs := range mapaIncs {
		for _, inc := range incs {
			switch inc.Tipo {
			case "VACACION":
				res.TotalVacaciones++
			case "LICENCIA_CON_GOCE":
				res.TotalLicenciasConGoce++
			case "LICENCIA_SIN_GOCE":
				res.TotalLicenciasSinGoce++
			}
		}
	}

	// 3. Verificar si hay vacaciones sin código vacacional en las planillas del mes
	var tieneCodigoVacacional bool
	r.db.QueryRow(`
		SELECT COUNT(1) > 0
		FROM planilla_plame_conceptos ppc
		INNER JOIN planillas p ON ppc.planilla_id = p.id
		WHERE p.tenant_id = $1 AND p.anio = $2 AND p.mes = $3
		  AND (ppc.es_concepto_vacacional = true OR ppc.codigo_sunat IN ('0118', '2007', '2043', '2049'))
	`, tenantID, anio, mes).Scan(&tieneCodigoVacacional)

	res.AlertaVacacionesSin0118 = (res.TotalVacaciones > 0 && !tieneCodigoVacacional)

	return res, nil
}

// ExisteSnapshotPlame verifica si ya existe el snapshot de PLAME para una planilla
func (r *PlameRepository) ExisteSnapshotPlame(planillaID int, tenantID int) (bool, error) {
	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(1)
		FROM planilla_plame_conceptos ppc
		INNER JOIN planillas p ON ppc.planilla_id = p.id
		WHERE ppc.planilla_id = $1 AND p.tenant_id = $2
	`, planillaID, tenantID).Scan(&count)
	return count > 0, err
}

// InicializarSnapshotPlame proyecta los conceptos calculados de planilla_conceptos hacia planilla_plame_conceptos
// aplicando el prorrateo proporcional de descansos vacacionales y consolidando la remuneración vacacional según régimen legal.
func (r *PlameRepository) InicializarSnapshotPlame(planillaID int, tenantID int) error {
	// 1. Obtener datos de la planilla
	var anio, mes int
	var estado string
	err := r.db.QueryRow(`SELECT anio, mes, estado FROM planillas WHERE id = $1 AND tenant_id = $2`, planillaID, tenantID).Scan(&anio, &mes, &estado)
	if err != nil {
		return fmt.Errorf("planilla no encontrada: %w", err)
	}

	// 2. Obtener incidencias de vacaciones en el mes
	licRepo := NewLicenciaVacacionRepository(r.db)
	mapaIncidencias, _ := licRepo.ObtenerIncidenciasMes(tenantID, anio, mes)
	diasVacacionesPorTrabajador := make(map[int]int)
	for trabID, incs := range mapaIncidencias {
		for _, inc := range incs {
			if inc.Tipo == "VACACION" || inc.CodigoSunatSuspension == "23" || inc.CodigoSunatSuspension == "34" {
				diasVacacionesPorTrabajador[trabID] += inc.DiasEnMes
			}
		}
	}

	// 3. Consultar todos los conceptos calculados de la planilla con metadatos
	query := `
		SELECT 
			pd.id AS planilla_detalle_id,
			c.trabajador_id,
			COALESCE(r.codigo, '728') AS regimen_codigo,
			pc.id AS planilla_concepto_id,
			COALESCE(NULLIF(pc.codigo_sunat, ''), cm.codigo, '0121') AS codigo_sunat,
			COALESCE(cm.descripcion, pc.nombre_en_boleta, '') AS descripcion_sunat,
			pc.tipo_concepto,
			pc.monto,
			COALESCE(ct.es_remunerativa, false) AS es_remunerativo
		FROM planilla_conceptos pc
		INNER JOIN planilla_detalles pd ON pc.planilla_detalle_id = pd.id
		INNER JOIN contratos c ON pd.contrato_id = c.id
		INNER JOIN planillas p ON pd.planilla_id = p.id
		LEFT JOIN puestos pst ON c.puesto_id = pst.id
		LEFT JOIN regimenes_laborales r ON pst.regimen_id = r.id
		LEFT JOIN conceptos_tenant ct ON pc.concepto_tenant_id = ct.id
		LEFT JOIN conceptos_maestros cm ON pc.maestro_id = cm.id
		WHERE pd.planilla_id = $1 AND p.tenant_id = $2 AND pc.monto > 0
		ORDER BY c.trabajador_id, pc.id
	`
	rows, err := r.db.Query(query, planillaID, tenantID)
	if err != nil {
		return fmt.Errorf("error consultando conceptos calculados: %w", err)
	}
	defer rows.Close()

	type itemConceptoCalculado struct {
		PlanillaDetalleID  int
		TrabajadorID       int
		RegimenCodigo      string
		PlanillaConceptoID int
		ConceptoID         sql.NullInt64
		CodigoSunat        string
		DescripcionSunat   string
		TipoConcepto       string
		Monto              float64
		EsRemunerativo     bool
	}

	var conceptosOriginales []itemConceptoCalculado
	for rows.Next() {
		var it itemConceptoCalculado
		err := rows.Scan(
			&it.PlanillaDetalleID,
			&it.TrabajadorID,
			&it.RegimenCodigo,
			&it.PlanillaConceptoID,
			&it.CodigoSunat,
			&it.DescripcionSunat,
			&it.TipoConcepto,
			&it.Monto,
			&it.EsRemunerativo,
		)
		if err != nil {
			return fmt.Errorf("error leyendo fila de concepto calculado: %w", err)
		}
		conceptosOriginales = append(conceptosOriginales, it)
	}

	// 4. Iniciar transacción para persistir el snapshot
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Limpiar snapshot previo si existiera
	_, err = tx.Exec(`DELETE FROM planilla_plame_conceptos WHERE planilla_id = $1`, planillaID)
	if err != nil {
		return fmt.Errorf("error limpiando snapshot previo: %w", err)
	}

	// Agrupar por trabajador para procesar prorrateos
	mapaTrabajadores := make(map[int][]itemConceptoCalculado)
	for _, c := range conceptosOriginales {
		mapaTrabajadores[c.PlanillaDetalleID] = append(mapaTrabajadores[c.PlanillaDetalleID], c)
	}

	stmtInsert, err := tx.Prepare(`
		INSERT INTO planilla_plame_conceptos (
			planilla_id,
			planilla_detalle_id,
			trabajador_id,
			planilla_concepto_id,
			codigo_sunat,
			descripcion_sunat,
			tipo_concepto,
			monto_devengado,
			monto_pagado,
			es_concepto_vacacional,
			es_ajuste_manual,
			observacion_ajuste
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`)
	if err != nil {
		return fmt.Errorf("error preparando insert snapshot: %w", err)
	}
	defer stmtInsert.Close()

	for _, lista := range mapaTrabajadores {
		if len(lista) == 0 {
			continue
		}
		detalleID := lista[0].PlanillaDetalleID
		trabID := lista[0].TrabajadorID
		regimen := lista[0].RegimenCodigo

		diasVac := diasVacacionesPorTrabajador[trabID]
		if diasVac > 30 {
			diasVac = 30
		}
		diasOrd := 30 - diasVac
		if diasOrd < 0 {
			diasOrd = 0
		}

		var totalRemVacacional float64
		codSunatVac := mapRegimenCodigoVacacional(regimen)
		descSunatVac := "REMUNERACIÓN VACACIONAL (" + regimen + ")"

		for _, item := range lista {
			if item.Monto <= 0 {
				continue
			}

			if diasVac > 0 && item.TipoConcepto == "INGRESO" && item.EsRemunerativo {
				montoVac := math.Round((item.Monto*(float64(diasVac)/30.0))*100) / 100
				montoOrd := math.Round((item.Monto-montoVac)*100) / 100

				totalRemVacacional += montoVac

				if montoOrd > 0 {
					pConceptoID := item.PlanillaConceptoID
					_, err = stmtInsert.Exec(
						planillaID,
						detalleID,
						trabID,
						pConceptoID,
						item.CodigoSunat,
						item.DescripcionSunat,
						item.TipoConcepto,
						montoOrd,
						montoOrd,
						false,
						false,
						"",
					)
					if err != nil {
						return fmt.Errorf("error insertando concepto ordinario prorrateado: %w", err)
					}
				}
			} else {
				pConceptoID := item.PlanillaConceptoID
				_, err = stmtInsert.Exec(
					planillaID,
					detalleID,
					trabID,
					pConceptoID,
					item.CodigoSunat,
					item.DescripcionSunat,
					item.TipoConcepto,
					item.Monto,
					item.Monto,
					false,
					false,
					"",
				)
				if err != nil {
					return fmt.Errorf("error insertando concepto regular: %w", err)
				}
			}
		}

		// Si acumuló monto vacacional, insertar la línea consolidada
		if totalRemVacacional > 0 {
			totalRemVacacional = math.Round(totalRemVacacional*100) / 100
			_, err = stmtInsert.Exec(
				planillaID,
				detalleID,
				trabID,
				nil, // No tiene concepto único de origen
				codSunatVac,
				descSunatVac,
				"INGRESO",
				totalRemVacacional,
				totalRemVacacional,
				true,
				false,
				fmt.Sprintf("Prorrateo vacacional automático (%d días)", diasVac),
			)
			if err != nil {
				return fmt.Errorf("error insertando concepto vacacional consolidado: %w", err)
			}
		}
	}

	return tx.Commit()
}

// ResetearSnapshotPlame elimina los ajustes manuales y recalcula el snapshot tributario desde planilla_conceptos
func (r *PlameRepository) ResetearSnapshotPlame(planillaID int, tenantID int) error {
	var estado string
	err := r.db.QueryRow(`SELECT estado FROM planillas WHERE id = $1 AND tenant_id = $2`, planillaID, tenantID).Scan(&estado)
	if err != nil {
		return fmt.Errorf("planilla no encontrada: %w", err)
	}
	if estado == "CERRADA" {
		return fmt.Errorf("no se puede restablecer el snapshot de una planilla CERRADA")
	}

	_, err = r.db.Exec(`
		DELETE FROM planilla_plame_conceptos ppc
		USING planillas p
		WHERE ppc.planilla_id = p.id AND p.id = $1 AND p.tenant_id = $2
	`, planillaID, tenantID)
	if err != nil {
		return fmt.Errorf("error limpiando snapshot: %w", err)
	}

	return r.InicializarSnapshotPlame(planillaID, tenantID)
}

// ObtenerPlameConceptosAgrupados obtiene la vista macro agregada de conceptos tributarios para la auditoría
func (r *PlameRepository) ObtenerPlameConceptosAgrupados(planillaID int, tenantID int) ([]models.ConceptoSunatAgrupado, error) {
	query := `
		SELECT 
			ppc.codigo_sunat,
			COALESCE(cm.descripcion, ppc.descripcion_sunat, 'CONCEPTO SUNAT ' || ppc.codigo_sunat) AS descripcion_sunat,
			ppc.tipo_concepto,
			COALESCE(cm.id, 0) AS maestro_id,
			COUNT(DISTINCT ppc.trabajador_id) AS total_trabajadores,
			COALESCE(SUM(ppc.monto_devengado), 0) AS total_devengado,
			COALESCE(SUM(ppc.monto_pagado), 0) AS total_pagado,
			COALESCE(BOOL_OR(ppc.es_ajuste_manual), false) AS tiene_ajustes_manuales,
			COALESCE(BOOL_OR(ppc.es_concepto_vacacional OR ppc.codigo_sunat IN ('0118', '2007', '2043', '2049')), false) AS tiene_vacacional
		FROM planilla_plame_conceptos ppc
		INNER JOIN planillas p ON ppc.planilla_id = p.id
		LEFT JOIN conceptos_maestros cm ON cm.codigo = ppc.codigo_sunat AND cm.origen = 'sunat'
		WHERE ppc.planilla_id = $1 AND p.tenant_id = $2
		GROUP BY ppc.codigo_sunat, COALESCE(cm.descripcion, ppc.descripcion_sunat, 'CONCEPTO SUNAT ' || ppc.codigo_sunat), ppc.tipo_concepto, cm.id
		ORDER BY 
			CASE ppc.tipo_concepto 
				WHEN 'INGRESO' THEN 1 
				WHEN 'RETENCION' THEN 2 
				WHEN 'DESCUENTO' THEN 2
				WHEN 'APORTE' THEN 3 
				ELSE 4 
			END,
			ppc.codigo_sunat ASC
	`
	rows, err := r.db.Query(query, planillaID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo conceptos agrupados de PLAME: %w", err)
	}
	defer rows.Close()

	var lista []models.ConceptoSunatAgrupado
	for rows.Next() {
		var c models.ConceptoSunatAgrupado
		err := rows.Scan(
			&c.CodigoSunatActual,
			&c.DescripcionSunatActual,
			&c.TipoConcepto,
			&c.MaestroID,
			&c.TotalTrabajadores,
			&c.TotalDevengado,
			&c.TotalPagado,
			&c.TieneAjustesManuales,
			&c.TieneVacacional,
		)
		if err != nil {
			return nil, fmt.Errorf("error escaneando concepto agrupado: %w", err)
		}
		lista = append(lista, c)
	}
	return lista, nil
}

// ObtenerPlameTrabajadoresPorConcepto retorna la lista de colaboradores afectados por un código SUNAT específico
func (r *PlameRepository) ObtenerPlameTrabajadoresPorConcepto(planillaID int, tenantID int, codigoSunat string) ([]models.PlameTrabajadorConceptoItem, error) {
	query := `
		SELECT 
			ppc.id AS planilla_plame_concepto_id,
			ppc.planilla_detalle_id,
			ppc.trabajador_id,
			t.tipo_documento,
			t.numero_documento,
			CONCAT(t.apellido_paterno, ' ', t.apellido_materno, ', ', t.nombres) AS nombre_completo,
			COALESCE(r.descripcion, r.codigo, 'SIN RÉGIMEN') AS regimen_nombre,
			ppc.codigo_sunat,
			ppc.descripcion_sunat,
			ppc.tipo_concepto,
			ppc.monto_devengado,
			ppc.monto_pagado,
			ppc.es_concepto_vacacional,
			ppc.es_ajuste_manual,
			COALESCE(ppc.observacion_ajuste, '') AS observacion_ajuste,
			COALESCE(pc.nombre_en_boleta, ppc.descripcion_sunat) AS concepto_laboral_nombre
		FROM planilla_plame_conceptos ppc
		INNER JOIN planillas p ON ppc.planilla_id = p.id
		INNER JOIN trabajadores t ON ppc.trabajador_id = t.id
		INNER JOIN planilla_detalles pd ON ppc.planilla_detalle_id = pd.id
		LEFT JOIN contratos c ON pd.contrato_id = c.id
		LEFT JOIN puestos pst ON c.puesto_id = pst.id
		LEFT JOIN regimenes_laborales r ON pst.regimen_id = r.id
		LEFT JOIN planilla_conceptos pc ON ppc.planilla_concepto_id = pc.id
		WHERE ppc.planilla_id = $1 AND p.tenant_id = $2 AND ppc.codigo_sunat = $3
		ORDER BY t.apellido_paterno ASC, t.apellido_materno ASC, t.nombres ASC
	`
	rows, err := r.db.Query(query, planillaID, tenantID, codigoSunat)
	if err != nil {
		return nil, fmt.Errorf("error consultando colaboradores por código SUNAT: %w", err)
	}
	defer rows.Close()

	var lista []models.PlameTrabajadorConceptoItem
	for rows.Next() {
		var it models.PlameTrabajadorConceptoItem
		err := rows.Scan(
			&it.PlanillaPlameConceptoID,
			&it.PlanillaDetalleID,
			&it.TrabajadorID,
			&it.TipoDocumento,
			&it.NumeroDocumento,
			&it.NombreCompleto,
			&it.RegimenNombre,
			&it.CodigoSunat,
			&it.DescripcionSunat,
			&it.TipoConcepto,
			&it.MontoDevengado,
			&it.MontoPagado,
			&it.EsConceptoVacacional,
			&it.EsAjusteManual,
			&it.ObservacionAjuste,
			&it.ConceptoLaboralNombre,
		)
		if err != nil {
			return nil, fmt.Errorf("error escaneando colaborador por concepto: %w", err)
		}
		lista = append(lista, it)
	}
	return lista, nil
}

// ObtenerPlameConceptosPorDetalle obtiene todos los conceptos tributarios del snapshot para un colaborador
func (r *PlameRepository) ObtenerPlameConceptosPorDetalle(detalleID int, tenantID int) ([]models.PlanillaPlameConcepto, error) {
	query := `
		SELECT 
			ppc.id,
			ppc.planilla_id,
			ppc.planilla_detalle_id,
			ppc.trabajador_id,
			ppc.planilla_concepto_id,
			ppc.codigo_sunat,
			ppc.descripcion_sunat,
			ppc.tipo_concepto,
			ppc.monto_devengado,
			ppc.monto_pagado,
			ppc.es_concepto_vacacional,
			ppc.es_ajuste_manual,
			COALESCE(ppc.observacion_ajuste, '') AS observacion_ajuste,
			ppc.created_at,
			ppc.updated_at,
			CONCAT(t.apellido_paterno, ' ', t.apellido_materno, ', ', t.nombres) AS trabajador_nombre,
			t.numero_documento AS trabajador_documento,
			t.tipo_documento AS trabajador_tipo_doc,
			COALESCE(r.descripcion, r.codigo, 'SIN RÉGIMEN') AS regimen_nombre,
			COALESCE(pc.nombre_en_boleta, ppc.descripcion_sunat) AS concepto_laboral_nombre
		FROM planilla_plame_conceptos ppc
		INNER JOIN planillas p ON ppc.planilla_id = p.id
		INNER JOIN planilla_detalles pd ON ppc.planilla_detalle_id = pd.id
		INNER JOIN trabajadores t ON ppc.trabajador_id = t.id
		LEFT JOIN contratos c ON pd.contrato_id = c.id
		LEFT JOIN puestos pst ON c.puesto_id = pst.id
		LEFT JOIN regimenes_laborales r ON pst.regimen_id = r.id
		LEFT JOIN planilla_conceptos pc ON ppc.planilla_concepto_id = pc.id
		WHERE ppc.planilla_detalle_id = $1 AND p.tenant_id = $2
		ORDER BY ppc.tipo_concepto ASC, ppc.codigo_sunat ASC
	`
	rows, err := r.db.Query(query, detalleID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("error consultando conceptos de detalle: %w", err)
	}
	defer rows.Close()

	var lista []models.PlanillaPlameConcepto
	for rows.Next() {
		var c models.PlanillaPlameConcepto
		var pConceptoID sql.NullInt64
		err := rows.Scan(
			&c.ID,
			&c.PlanillaID,
			&c.PlanillaDetalleID,
			&c.TrabajadorID,
			&pConceptoID,
			&c.CodigoSunat,
			&c.DescripcionSunat,
			&c.TipoConcepto,
			&c.MontoDevengado,
			&c.MontoPagado,
			&c.EsConceptoVacacional,
			&c.EsAjusteManual,
			&c.ObservacionAjuste,
			&c.CreatedAt,
			&c.UpdatedAt,
			&c.TrabajadorNombre,
			&c.TrabajadorDocumento,
			&c.TrabajadorTipoDoc,
			&c.RegimenNombre,
			&c.ConceptoLaboralNombre,
		)
		if err != nil {
			return nil, fmt.Errorf("error escaneando concepto plame de trabajador: %w", err)
		}
		if pConceptoID.Valid {
			idVal := int(pConceptoID.Int64)
			c.PlanillaConceptoID = &idVal
		}
		lista = append(lista, c)
	}
	return lista, nil
}

// GuardarPlameConceptosTrabajador actualiza transaccionalmente las líneas del snapshot para un colaborador
func (r *PlameRepository) GuardarPlameConceptosTrabajador(detalleID int, tenantID int, items []models.PlanillaPlameConcepto) error {
	var planillaID, trabajadorID int
	var estado string
	err := r.db.QueryRow(`
		SELECT pd.planilla_id, c.trabajador_id, p.estado
		FROM planilla_detalles pd
		INNER JOIN contratos c ON pd.contrato_id = c.id
		INNER JOIN planillas p ON pd.planilla_id = p.id
		WHERE pd.id = $1 AND p.tenant_id = $2
	`, detalleID, tenantID).Scan(&planillaID, &trabajadorID, &estado)
	if err != nil {
		return fmt.Errorf("detalle de planilla no encontrado: %w", err)
	}
	if estado == "CERRADA" {
		return fmt.Errorf("la planilla se encuentra CERRADA y no permite modificaciones")
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`DELETE FROM planilla_plame_conceptos WHERE planilla_detalle_id = $1`, detalleID)
	if err != nil {
		return fmt.Errorf("error limpiando conceptos previos del trabajador: %w", err)
	}

	stmtInsert, err := tx.Prepare(`
		INSERT INTO planilla_plame_conceptos (
			planilla_id,
			planilla_detalle_id,
			trabajador_id,
			planilla_concepto_id,
			codigo_sunat,
			descripcion_sunat,
			tipo_concepto,
			monto_devengado,
			monto_pagado,
			es_concepto_vacacional,
			es_ajuste_manual,
			observacion_ajuste,
			updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, CURRENT_TIMESTAMP)
	`)
	if err != nil {
		return fmt.Errorf("error preparando inserción de conceptos editados: %w", err)
	}
	defer stmtInsert.Close()

	for _, item := range items {
		var descSunat string
		err := tx.QueryRow(`SELECT descripcion FROM conceptos_maestros WHERE codigo = $1 AND origen = 'sunat' LIMIT 1`, item.CodigoSunat).Scan(&descSunat)
		if err != nil || descSunat == "" {
			descSunat = "CONCEPTO SUNAT " + item.CodigoSunat
		}

		_, err = stmtInsert.Exec(
			planillaID,
			detalleID,
			trabajadorID,
			item.PlanillaConceptoID,
			item.CodigoSunat,
			descSunat,
			item.TipoConcepto,
			item.MontoDevengado,
			item.MontoPagado,
			item.EsConceptoVacacional,
			true,
			item.ObservacionAjuste,
		)
		if err != nil {
			return fmt.Errorf("error insertando ajuste de concepto para trabajador: %w", err)
		}
	}

	return tx.Commit()
}

// ActualizarCodigoSunatPlameMasivo reasigna un código SUNAT para todas las líneas correspondientes en el snapshot
func (r *PlameRepository) ActualizarCodigoSunatPlameMasivo(planillaID int, tenantID int, codigoActual string, nuevoMaestroID int, actualizarDefault bool) (string, error) {
	var estado string
	err := r.db.QueryRow(`SELECT estado FROM planillas WHERE id = $1 AND tenant_id = $2`, planillaID, tenantID).Scan(&estado)
	if err != nil {
		return "", fmt.Errorf("planilla no encontrada: %w", err)
	}
	if estado == "CERRADA" {
		return "", fmt.Errorf("la planilla se encuentra CERRADA y no permite modificaciones")
	}

	var nuevoCodigoSunat, nuevaDescripcion string
	err = r.db.QueryRow(`SELECT codigo, descripcion FROM conceptos_maestros WHERE id = $1 AND origen = 'sunat'`, nuevoMaestroID).Scan(&nuevoCodigoSunat, &nuevaDescripcion)
	if err != nil {
		return "", fmt.Errorf("código maestro SUNAT no válido: %w", err)
	}

	tx, err := r.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		UPDATE planilla_plame_conceptos
		SET codigo_sunat = $1,
		    descripcion_sunat = $2,
		    es_ajuste_manual = true,
		    updated_at = CURRENT_TIMESTAMP
		WHERE planilla_id = $3 AND codigo_sunat = $4
	`, nuevoCodigoSunat, nuevaDescripcion, planillaID, codigoActual)
	if err != nil {
		return "", fmt.Errorf("error actualizando códigos en snapshot: %w", err)
	}

	if actualizarDefault {
		tx.Exec(`
			UPDATE conceptos_tenant
			SET concepto_id = $1,
			    updated_at = CURRENT_TIMESTAMP
			WHERE tenant_id = $2 
			  AND concepto_id IN (SELECT id FROM conceptos_maestros WHERE codigo = $3 AND origen = 'sunat')
		`, nuevoMaestroID, tenantID, codigoActual)
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}

	return nuevoCodigoSunat, nil
}

// ObtenerDatosPlameJornada obtiene los datos de jornada laboral para el archivo .jor
func (r *PlameRepository) ObtenerDatosPlameJornada(planillaID int, tenantID int) ([]models.PlameJornada, error) {
	var anio, mes int
	err := r.db.QueryRow(`SELECT anio, mes FROM planillas WHERE id = $1 AND tenant_id = $2`, planillaID, tenantID).Scan(&anio, &mes)
	if err != nil {
		return nil, fmt.Errorf("planilla no encontrada: %w", err)
	}

	licRepo := NewLicenciaVacacionRepository(r.db)
	mapaIncs, _ := licRepo.ObtenerIncidenciasMes(tenantID, anio, mes)

	query := `
		SELECT 
			c.trabajador_id,
			t.tipo_documento,
			t.numero_documento,
			pd.dias_trabajados,
			pd.dias_subsidiados,
			pd.dias_no_laborados
		FROM planilla_detalles pd
		INNER JOIN contratos c ON pd.contrato_id = c.id
		INNER JOIN trabajadores t ON c.trabajador_id = t.id
		INNER JOIN planillas p ON pd.planilla_id = p.id
		WHERE pd.planilla_id = $1 AND p.tenant_id = $2
		ORDER BY t.numero_documento ASC
	`
	rows, err := r.db.Query(query, planillaID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.PlameJornada
	for rows.Next() {
		var trabID int
		var j models.PlameJornada
		var diasLaborados, diasSubsidiados float64
		err := rows.Scan(
			&trabID,
			&j.TipoDocumento,
			&j.NumeroDocumento,
			&diasLaborados,
			&diasSubsidiados,
			&j.DiasInasistencia,
		)
		if err != nil {
			return nil, err
		}

		var diasSuspension int
		if incs, ok := mapaIncs[trabID]; ok {
			for _, inc := range incs {
				diasSuspension += inc.DiasEnMes
			}
		}
		if diasSuspension > int(j.DiasInasistencia) {
			j.DiasInasistencia = float64(diasSuspension)
		}

		lista = append(lista, j)
	}
	return lista, nil
}

// ObtenerDatosPlameRemuneracionesDirectas obtiene las remuneraciones consolidadas directamente del snapshot
func (r *PlameRepository) ObtenerDatosPlameRemuneracionesDirectas(planillaID int, tenantID int) ([]models.PlameRemuneracion, error) {
	query := `
		SELECT 
			t.tipo_documento,
			t.numero_documento,
			ppc.codigo_sunat,
			SUM(ppc.monto_devengado) AS monto_devengado,
			SUM(ppc.monto_pagado) AS monto_pagado
		FROM planilla_plame_conceptos ppc
		INNER JOIN planillas p ON ppc.planilla_id = p.id
		INNER JOIN trabajadores t ON ppc.trabajador_id = t.id
		WHERE ppc.planilla_id = $1 AND p.tenant_id = $2
		GROUP BY t.tipo_documento, t.numero_documento, ppc.codigo_sunat
		HAVING SUM(ppc.monto_devengado) > 0 OR SUM(ppc.monto_pagado) > 0
		ORDER BY t.numero_documento ASC, ppc.codigo_sunat ASC
	`
	rows, err := r.db.Query(query, planillaID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo remuneraciones de snapshot: %w", err)
	}
	defer rows.Close()

	var lista []models.PlameRemuneracion
	for rows.Next() {
		var r models.PlameRemuneracion
		err := rows.Scan(
			&r.TipoDocumento,
			&r.NumeroDocumento,
			&r.CodigoConcepto,
			&r.MontoDevengado,
			&r.MontoPagado,
		)
		if err != nil {
			return nil, err
		}
		r.Monto = r.MontoPagado
		lista = append(lista, r)
	}
	return lista, nil
}

// Helper interno para mapear régimen a código SUNAT vacacional
func mapRegimenCodigoVacacional(regimenCodigo string) string {
	reg := strings.TrimSpace(strings.ToUpper(regimenCodigo))
	switch {
	case strings.Contains(reg, "1057"), strings.Contains(reg, "CAS"):
		return "2043"
	case strings.Contains(reg, "30057"), strings.Contains(reg, "SERVIR"):
		return "2049"
	case strings.Contains(reg, "276"):
		return "2007"
	case strings.Contains(reg, "728"):
		return "2007"
	default:
		return "2007"
	}
}

// ObtenerPadronTrabajadoresPlame obtiene la lista consolidada de trabajadores de una planilla para auditoría con búsqueda y paginación
func (r *PlameRepository) ObtenerPadronTrabajadoresPlame(planillaID, tenantID int, q string, limit, offset int) ([]models.PlameTrabajadorPadronItem, int, error) {
	termino := strings.TrimSpace(q)

	countQuery := `
		SELECT COUNT(pd.id)
		FROM planilla_detalles pd
		INNER JOIN contratos c ON pd.contrato_id = c.id
		INNER JOIN trabajadores t ON c.trabajador_id = t.id
		INNER JOIN planillas p ON pd.planilla_id = p.id
		WHERE pd.planilla_id = $1 AND p.tenant_id = $2
		  AND (
			$3 = '' OR 
			t.numero_documento ILIKE '%' || $3 || '%' OR 
			CONCAT(t.apellido_paterno, ' ', t.apellido_materno, ' ', t.nombres) ILIKE '%' || $3 || '%'
		  )
	`
	var total int
	err := r.db.QueryRow(countQuery, planillaID, tenantID, termino).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("error contando trabajadores del padrón: %w", err)
	}

	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT 
			pd.id AS planilla_detalle_id,
			t.id AS trabajador_id,
			t.tipo_documento,
			t.numero_documento,
			TRIM(CONCAT(t.apellido_paterno, ' ', t.apellido_materno, ', ', t.nombres)) AS nombre_completo,
			COALESCE(r.descripcion, r.codigo, 'SIN RÉGIMEN') AS regimen_nombre,
			COALESCE(SUM(ppc.monto_devengado), 0.00) AS total_devengado,
			COALESCE(SUM(ppc.monto_pagado), 0.00) AS total_pagado,
			COUNT(ppc.id) AS total_conceptos,
			COALESCE(BOOL_OR(ppc.es_ajuste_manual), false) AS tiene_ajuste_manual
		FROM planilla_detalles pd
		INNER JOIN contratos c ON pd.contrato_id = c.id
		INNER JOIN trabajadores t ON c.trabajador_id = t.id
		LEFT JOIN puestos pst ON c.puesto_id = pst.id
		LEFT JOIN regimenes_laborales r ON pst.regimen_id = r.id
		INNER JOIN planillas p ON pd.planilla_id = p.id
		LEFT JOIN planilla_plame_conceptos ppc ON pd.id = ppc.planilla_detalle_id
		WHERE pd.planilla_id = $1 AND p.tenant_id = $2
		  AND (
			$3 = '' OR 
			t.numero_documento ILIKE '%' || $3 || '%' OR 
			CONCAT(t.apellido_paterno, ' ', t.apellido_materno, ' ', t.nombres) ILIKE '%' || $3 || '%'
		  )
		GROUP BY pd.id, t.id, t.tipo_documento, t.numero_documento, t.apellido_paterno, t.apellido_materno, t.nombres, r.descripcion, r.codigo
		ORDER BY t.apellido_paterno ASC, t.apellido_materno ASC, t.nombres ASC
		LIMIT $4 OFFSET $5
	`
	rows, err := r.db.Query(query, planillaID, tenantID, termino, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("error consultando padrón de trabajadores: %w", err)
	}
	defer rows.Close()

	var lista []models.PlameTrabajadorPadronItem
	for rows.Next() {
		var it models.PlameTrabajadorPadronItem
		err := rows.Scan(
			&it.PlanillaDetalleID,
			&it.TrabajadorID,
			&it.TipoDocumento,
			&it.NumeroDocumento,
			&it.NombreCompleto,
			&it.RegimenNombre,
			&it.TotalDevengado,
			&it.TotalPagado,
			&it.TotalConceptos,
			&it.TieneAjusteManual,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("error escaneando trabajador del padrón: %w", err)
		}
		lista = append(lista, it)
	}

	return lista, total, nil
}

// ObtenerConceptosNominaPlame obtiene el consolidado de conceptos institucionales de nómina usados en la planilla y su mapeo al snapshot PLAME
func (r *PlameRepository) ObtenerConceptosNominaPlame(planillaID, tenantID int) ([]models.PlameConceptoNominaItem, error) {
	query := `
		SELECT 
			COALESCE(ct.id, 0) AS concepto_tenant_id,
			COALESCE(ct.nombre_personalizado, pc.nombre_en_boleta, 'Remuneración Vacacional') AS concepto_nombre,
			ppc.tipo_concepto,
			ppc.codigo_sunat,
			COALESCE(NULLIF(ppc.descripcion_sunat, ''), cm.descripcion, 'Concepto SUNAT') AS descripcion_sunat,
			COUNT(DISTINCT ppc.trabajador_id) AS total_trabajadores,
			COALESCE(SUM(ppc.monto_devengado), 0.00) AS total_devengado,
			COALESCE(SUM(ppc.monto_pagado), 0.00) AS total_pagado,
			COALESCE(BOOL_OR(ppc.es_ajuste_manual), false) AS tiene_ajuste_manual
		FROM planilla_plame_conceptos ppc
		INNER JOIN planillas p ON ppc.planilla_id = p.id
		LEFT JOIN planilla_conceptos pc ON ppc.planilla_concepto_id = pc.id
		LEFT JOIN conceptos_tenant ct ON pc.concepto_tenant_id = ct.id
		LEFT JOIN conceptos_maestros cm ON cm.codigo = ppc.codigo_sunat AND cm.origen = 'sunat'
		WHERE ppc.planilla_id = $1 AND p.tenant_id = $2
		GROUP BY ct.id, ct.nombre_personalizado, pc.nombre_en_boleta, ppc.tipo_concepto, ppc.codigo_sunat, cm.descripcion, ppc.descripcion_sunat
		ORDER BY 
			CASE ppc.tipo_concepto
				WHEN 'INGRESO' THEN 1
				WHEN 'RETENCION' THEN 2
				WHEN 'APORTE' THEN 3
				ELSE 4
			END,
			concepto_nombre ASC
	`
	rows, err := r.db.Query(query, planillaID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("error consultando conceptos de nómina de la planilla: %w", err)
	}
	defer rows.Close()

	var lista []models.PlameConceptoNominaItem
	for rows.Next() {
		var it models.PlameConceptoNominaItem
		err := rows.Scan(
			&it.ConceptoTenantID,
			&it.ConceptoNombre,
			&it.TipoConcepto,
			&it.CodigoSunat,
			&it.DescripcionSunat,
			&it.TotalTrabajadores,
			&it.TotalDevengado,
			&it.TotalPagado,
			&it.TieneAjusteManual,
		)
		if err != nil {
			return nil, fmt.Errorf("error escaneando concepto de nómina: %w", err)
		}
		lista = append(lista, it)
	}

	return lista, nil
}

// ActualizarCodigoConceptoNominaMasivo actualiza en bloque el código SUNAT para todas las líneas de un concepto institucional en el snapshot
func (r *PlameRepository) ActualizarCodigoConceptoNominaMasivo(planillaID, tenantID int, conceptoTenantID int, nombreEnBoleta string, nuevoCodigoSunat string, actualizarDefault bool) (int64, error) {
	var estado string
	err := r.db.QueryRow(`SELECT estado FROM planillas WHERE id = $1 AND tenant_id = $2`, planillaID, tenantID).Scan(&estado)
	if err != nil {
		return 0, fmt.Errorf("planilla no encontrada: %w", err)
	}
	if estado == "CERRADA" {
		return 0, fmt.Errorf("la planilla se encuentra CERRADA y no permite modificaciones")
	}

	var descSunat string
	err = r.db.QueryRow(`SELECT descripcion FROM conceptos_maestros WHERE codigo = $1 AND origen = 'sunat' LIMIT 1`, nuevoCodigoSunat).Scan(&descSunat)
	if err != nil || descSunat == "" {
		descSunat = "CONCEPTO SUNAT " + nuevoCodigoSunat
	}

	tx, err := r.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var res sql.Result
	if conceptoTenantID > 0 {
		updateQuery := `
			UPDATE planilla_plame_conceptos ppc
			SET codigo_sunat = $1,
			    descripcion_sunat = $2,
			    es_ajuste_manual = true,
			    observacion_ajuste = 'Reasignación masiva desde conceptos de nómina',
			    updated_at = CURRENT_TIMESTAMP
			FROM planilla_conceptos pc
			WHERE ppc.planilla_concepto_id = pc.id
			  AND pc.concepto_tenant_id = $3
			  AND ppc.planilla_id = $4
		`
		res, err = tx.Exec(updateQuery, nuevoCodigoSunat, descSunat, conceptoTenantID, planillaID)
	} else if nombreEnBoleta != "" {
		updateQuery := `
			UPDATE planilla_plame_conceptos ppc
			SET codigo_sunat = $1,
			    descripcion_sunat = $2,
			    es_ajuste_manual = true,
			    observacion_ajuste = 'Reasignación masiva desde conceptos de nómina',
			    updated_at = CURRENT_TIMESTAMP
			FROM planilla_conceptos pc
			WHERE ppc.planilla_concepto_id = pc.id
			  AND pc.nombre_en_boleta = $3
			  AND ppc.planilla_id = $4
		`
		res, err = tx.Exec(updateQuery, nuevoCodigoSunat, descSunat, nombreEnBoleta, planillaID)
	} else {
		return 0, fmt.Errorf("debe especificar conceptoTenantID o nombreEnBoleta")
	}

	if err != nil {
		return 0, fmt.Errorf("error actualizando líneas de concepto en snapshot: %w", err)
	}

	filasAfectadas, _ := res.RowsAffected()

	if actualizarDefault && conceptoTenantID > 0 {
		_, err = tx.Exec(`
			UPDATE conceptos_tenant
			SET concepto_id = COALESCE((SELECT id FROM conceptos_maestros WHERE codigo = $1 AND origen = 'sunat' LIMIT 1), concepto_id)
			WHERE id = $2 AND tenant_id = $3
		`, nuevoCodigoSunat, conceptoTenantID, tenantID)
		if err != nil {
			return 0, fmt.Errorf("error actualizando catálogo predeterminado de concepto: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return filasAfectadas, nil
}

