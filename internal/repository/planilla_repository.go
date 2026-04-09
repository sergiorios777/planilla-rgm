package repository

import (
	"database/sql"
	"planilla-rgm/internal/models"
	"strconv"
	"strings"
)

type PlanillaRepository struct {
	db *sql.DB
}

func NewPlanillaRepository(db *sql.DB) *PlanillaRepository {
	return &PlanillaRepository{db: db}
}

// ObtenerTodos trae el historial de planillas de la entidad
func (r *PlanillaRepository) ObtenerTodos(tenantID int) ([]models.Planilla, error) {
	query := `
		SELECT id, tenant_id, anio, mes, descripcion, estado 
		FROM planillas 
		WHERE tenant_id = $1 
		ORDER BY anio DESC, mes DESC, id DESC
	`
	rows, err := r.db.Query(query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.Planilla
	for rows.Next() {
		var p models.Planilla
		err := rows.Scan(&p.ID, &p.TenantID, &p.Anio, &p.Mes, &p.Descripcion, &p.Estado)
		if err == nil {
			lista = append(lista, p)
		}
	}
	return lista, nil
}

// Crear inserta una nueva cabecera mensual
func (r *PlanillaRepository) Crear(p *models.Planilla) error {
	query := `
		INSERT INTO planillas (tenant_id, anio, mes, descripcion, estado)
		VALUES ($1, $2, $3, $4, 'BORRADOR') RETURNING id
	`
	return r.db.QueryRow(query, p.TenantID, p.Anio, p.Mes, p.Descripcion).Scan(&p.ID)
}

// ProcesarPlanilla es el Motor de Cálculo. Genera las boletas de todos los trabajadores activos en ese mes.
func (r *PlanillaRepository) ProcesarPlanilla(planillaID int, tenantID int) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	var anio, mes int
	err = tx.QueryRow(`SELECT anio, mes FROM planillas WHERE id = $1 AND tenant_id = $2`, planillaID, tenantID).Scan(&anio, &mes)
	if err != nil {
		tx.Rollback()
		return err
	}

	// NUEVO: Convertimos el mes actual a texto (Ej: 1 -> "1") para buscarlo luego en la frecuencia
	mesActualStr := strconv.Itoa(mes)

	// Limpiamos reprocesamientos
	_, err = tx.Exec(`DELETE FROM planilla_detalles WHERE planilla_id = $1`, planillaID)
	if err != nil {
		tx.Rollback()
		return err
	}

	// 1. EXTRAER CONTRATOS A MEMORIA
	queryContratos := `
		SELECT c.id, c.puesto_id 
		FROM contratos c
		WHERE c.tenant_id = $1 AND c.activo = true
		  AND c.fecha_inicio <= (make_date($2, $3, 1) + interval '1 month' - interval '1 day')::date
		  AND (c.fecha_fin IS NULL OR c.fecha_fin >= make_date($2, $3, 1)::date)
	`
	rowsContratos, err := tx.Query(queryContratos, tenantID, anio, mes)
	if err != nil {
		tx.Rollback()
		return err
	}

	type ContratoTemp struct {
		ID       int
		PuestoID int
	}
	var contratosActivos []ContratoTemp

	for rowsContratos.Next() {
		var c ContratoTemp
		rowsContratos.Scan(&c.ID, &c.PuestoID)
		contratosActivos = append(contratosActivos, c)
	}
	rowsContratos.Close()

	// 2. RECORRER LA MEMORIA Y PROCESAR
	for _, contrato := range contratosActivos {

		// A. Cabecera de la Boleta
		var detalleID int
		err = tx.QueryRow(`
			INSERT INTO planilla_detalles (planilla_id, contrato_id) 
			VALUES ($1, $2) RETURNING id
		`, planillaID, contrato.ID).Scan(&detalleID)
		if err != nil {
			tx.Rollback()
			return err
		}

		// B. EXTRAER CONCEPTOS A MEMORIA (NUEVO: Traemos ct.frecuencia_meses)
		queryConceptos := `
			SELECT pc.concepto_tenant_id, cm.tipo, pc.monto, ct.frecuencia_meses
			FROM puesto_conceptos pc
			INNER JOIN conceptos_tenant ct ON pc.concepto_tenant_id = ct.id
			INNER JOIN conceptos_maestros cm ON ct.concepto_id = cm.id
			WHERE pc.puesto_id = $1 AND pc.activo = true
		`
		rowsConceptos, err := tx.Query(queryConceptos, contrato.PuestoID)
		if err != nil {
			tx.Rollback()
			return err
		}

		type ConceptoTemp struct {
			TenantID   int
			Tipo       string
			Monto      float64
			Frecuencia string // NUEVO: Para guardar "1,2...12" o "7,12"
		}
		var conceptosPuesto []ConceptoTemp

		for rowsConceptos.Next() {
			var cp ConceptoTemp
			var m sql.NullFloat64
			// NUEVO: Escaneamos también la frecuencia
			rowsConceptos.Scan(&cp.TenantID, &cp.Tipo, &m, &cp.Frecuencia)
			if m.Valid {
				cp.Monto = m.Float64
			}
			conceptosPuesto = append(conceptosPuesto, cp)
		}
		rowsConceptos.Close()

		var tIngresos, tRetenciones, tAportes float64

		// C. PROCESAR CONCEPTOS
		for _, cp := range conceptosPuesto {

			// =================================================================
			// NUEVO: LÓGICA DE FRECUENCIA DE PAGO
			// =================================================================
			mesesPermitidos := strings.Split(cp.Frecuencia, ",")
			aplicaParaEsteMes := false

			for _, mesPermitido := range mesesPermitidos {
				if strings.TrimSpace(mesPermitido) == mesActualStr {
					aplicaParaEsteMes = true
					break
				}
			}

			// Si el mes de la planilla no está en la frecuencia del concepto, lo saltamos
			if !aplicaParaEsteMes {
				continue
			}
			// =================================================================

			tipoUpper := strings.ToUpper(cp.Tipo)

			if tipoUpper == "INGRESO" {
				tIngresos += cp.Monto
			} else if tipoUpper == "RETENCION" {
				tRetenciones += cp.Monto
			} else if tipoUpper == "APORTE" {
				tAportes += cp.Monto
			}

			_, err = tx.Exec(`
				INSERT INTO planilla_conceptos (planilla_detalle_id, concepto_tenant_id, tipo_concepto, monto)
				VALUES ($1, $2, $3, $4)
			`, detalleID, cp.TenantID, tipoUpper, cp.Monto)
			if err != nil {
				tx.Rollback()
				return err
			}
		}

		// D. ACTUALIZAR TOTALES
		neto := tIngresos - tRetenciones
		_, err = tx.Exec(`
			UPDATE planilla_detalles 
			SET total_ingresos = $1, total_retenciones = $2, total_aportes = $3, neto_pagar = $4
			WHERE id = $5
		`, tIngresos, tRetenciones, tAportes, neto, detalleID)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

// ObtenerDetalles trae la lista de boletas generadas en una planilla
func (r *PlanillaRepository) ObtenerDetalles(planillaID int, tenantID int) ([]models.PlanillaDetalle, error) {
	query := `
		SELECT d.id, d.planilla_id, d.contrato_id, d.total_ingresos, d.total_retenciones, d.total_aportes, d.neto_pagar,
		       t.apellido_paterno || ' ' || t.apellido_materno || ', ' || t.nombres AS trabajador_nombre,
		       t.numero_documento AS trabajador_doc,
		       p.nombre AS puesto_nombre
		FROM planilla_detalles d
		INNER JOIN planillas pl ON d.planilla_id = pl.id
		INNER JOIN contratos c ON d.contrato_id = c.id
		INNER JOIN trabajadores t ON c.trabajador_id = t.id
		INNER JOIN puestos p ON c.puesto_id = p.id
		WHERE d.planilla_id = $1 AND pl.tenant_id = $2
		ORDER BY t.apellido_paterno ASC
	`
	rows, err := r.db.Query(query, planillaID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.PlanillaDetalle
	for rows.Next() {
		var d models.PlanillaDetalle
		err := rows.Scan(&d.ID, &d.PlanillaID, &d.ContratoID, &d.TotalIngresos, &d.TotalRetenciones, &d.TotalAportes, &d.NetoPagar,
			&d.TrabajadorNombre, &d.TrabajadorDoc, &d.PuestoNombre)
		if err == nil {
			lista = append(lista, d)
		}
	}
	return lista, nil
}
