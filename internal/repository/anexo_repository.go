package repository

import (
	"database/sql"
	"fmt"
	"planilla-rgm/internal/models"
	"strings"

	"github.com/lib/pq"
)

type AnexoRepository struct {
	db *sql.DB
}

func NewAnexoRepository(db *sql.DB) *AnexoRepository {
	return &AnexoRepository{db: db}
}

// ObtenerCompromisoPresupuestal obtiene el consolidado por Meta y Clasificador MEF para la planilla (excluyendo RETENCIONES)
func (r *AnexoRepository) ObtenerCompromisoPresupuestal(planillaID int, tenantID int) ([]models.ItemCompromisoPresupuestal, error) {
	query := `
		SELECT 
			COALESCE(m.codigo, '0000') AS meta_codigo,
			COALESCE(m.descripcion, 'Sin Meta Presupuestal') AS meta_descripcion,
			COALESCE(cm.codigo, '2.0.0.0.0.0') AS clasificador_codigo,
			COALESCE(cm.descripcion, 'Sin Clasificador Gasto') AS clasificador_descripcion,
			SUM(pc.monto) AS monto_total
		FROM planilla_detalles pd
		JOIN planilla_conceptos pc ON pc.planilla_detalle_id = pd.id
		JOIN contratos c ON pd.contrato_id = c.id
		LEFT JOIN puestos p ON c.puesto_id = p.id
		LEFT JOIN metas_presupuestales m ON p.meta_id = m.id
		LEFT JOIN conceptos_tenant ct ON pc.concepto_tenant_id = ct.id
		LEFT JOIN clasificadores_mef cm ON ct.clasificador_id = cm.id
		WHERE pd.planilla_id = $1 AND c.tenant_id = $2 AND pc.tipo_concepto != 'RETENCION'
		GROUP BY m.codigo, m.descripcion, cm.codigo, cm.descripcion
		ORDER BY m.codigo ASC, cm.codigo ASC
	`

	rows, err := r.db.Query(query, planillaID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("error al consultar compromiso presupuestal: %w", err)
	}
	defer rows.Close()

	var items []models.ItemCompromisoPresupuestal
	for rows.Next() {
		var item models.ItemCompromisoPresupuestal
		if err := rows.Scan(
			&item.MetaCodigo,
			&item.MetaDescripcion,
			&item.ClasificadorCodigo,
			&item.ClasificadorDescripcion,
			&item.MontoTotal,
		); err != nil {
			return nil, fmt.Errorf("error al escanear fila de compromiso presupuestal: %w", err)
		}
		items = append(items, item)
	}

	return items, nil
}

// ConceptoSumatorioSunat contiene la suma de un concepto SUNAT en la planilla
type ConceptoSumatorioSunat struct {
	CodigoSunat    string
	NombreEnBoleta string
	MontoTotal     float64
}

// ObtenerSumatoriasSunat obtiene la sumatoria de todos los conceptos de tipo RETENCION en la planilla
func (r *AnexoRepository) ObtenerSumatoriasSunat(planillaID int, tenantID int) ([]ConceptoSumatorioSunat, error) {
	query := `
		SELECT 
			COALESCE(pc.codigo_sunat, '') AS codigo_sunat,
			COALESCE(pc.nombre_en_boleta, '') AS nombre_en_boleta,
			SUM(pc.monto) AS monto_total
		FROM planilla_detalles pd
		JOIN planilla_conceptos pc ON pc.planilla_detalle_id = pd.id
		JOIN contratos c ON pd.contrato_id = c.id
		WHERE pd.planilla_id = $1 AND c.tenant_id = $2 AND pc.tipo_concepto = 'RETENCION'
		GROUP BY pc.codigo_sunat, pc.nombre_en_boleta
	`

	rows, err := r.db.Query(query, planillaID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("error al consultar sumatorias SUNAT: %w", err)
	}
	defer rows.Close()

	var lista []ConceptoSumatorioSunat
	for rows.Next() {
		var item ConceptoSumatorioSunat
		if err := rows.Scan(&item.CodigoSunat, &item.NombreEnBoleta, &item.MontoTotal); err != nil {
			return nil, err
		}
		lista = append(lista, item)
	}

	return lista, nil
}

// TargetAjusteDetalle contiene toda la información de destino para aplicar ajustes de redondeo (Anexo 1 y Anexo 1A)
type TargetAjusteDetalle struct {
	MetaCodigo         string
	ClasificadorCodigo string
	CodigoSunatIngreso string
	NombreIngreso      string
}

// ObtenerTargetCompromisoAjuste busca la combinación (Meta, Clasificador MEF, Concepto Ingreso SUNAT) remunerativa para aplicar el ajuste de redondeo
func (r *AnexoRepository) ObtenerTargetCompromisoAjuste(planillaID int, tenantID int, codigosSunat []string, palabraClave string) (TargetAjusteDetalle, error) {
	var target TargetAjusteDetalle

	queryEspecifico := `
		SELECT 
			COALESCE(m.codigo, '0000') AS meta_codigo,
			COALESCE(cm.codigo, '2.0.0.0.0.0') AS clasificador_codigo,
			COALESCE(pc.codigo_sunat, '') AS codigo_sunat_ingreso,
			COALESCE(pc.nombre_en_boleta, ct.nombre_personalizado, 'CONCEPTO SIN NOMBRE') AS nombre_ingreso
		FROM planilla_detalles pd
		JOIN planilla_conceptos pc ON pc.planilla_detalle_id = pd.id
		JOIN contratos c ON pd.contrato_id = c.id
		LEFT JOIN puestos p ON c.puesto_id = p.id
		LEFT JOIN metas_presupuestales m ON p.meta_id = m.id
		LEFT JOIN conceptos_tenant ct ON pc.concepto_tenant_id = ct.id
		LEFT JOIN clasificadores_mef cm ON ct.clasificador_id = cm.id
		WHERE pd.planilla_id = $1 AND c.tenant_id = $2
		  AND pc.tipo_concepto = 'INGRESO'
		  AND COALESCE(ct.es_remunerativa, false) = true
		  AND pd.id IN (
			  SELECT pd_sub.id 
			  FROM planilla_detalles pd_sub
			  JOIN planilla_conceptos pc_sub ON pc_sub.planilla_detalle_id = pd_sub.id
			  WHERE pd_sub.planilla_id = $1 
			    AND (pc_sub.codigo_sunat = ANY($3) OR UPPER(pc_sub.nombre_en_boleta) LIKE $4)
		  )
		GROUP BY m.codigo, cm.codigo, pc.codigo_sunat, COALESCE(pc.nombre_en_boleta, ct.nombre_personalizado, 'CONCEPTO SIN NOMBRE')
		ORDER BY SUM(pc.monto) DESC
		LIMIT 1
	`

	patron := "%" + strings.ToUpper(palabraClave) + "%"
	err := r.db.QueryRow(queryEspecifico, planillaID, tenantID, pq.Array(codigosSunat), patron).Scan(
		&target.MetaCodigo,
		&target.ClasificadorCodigo,
		&target.CodigoSunatIngreso,
		&target.NombreIngreso,
	)

	if err == nil && target.MetaCodigo != "" && target.ClasificadorCodigo != "" {
		return target, nil
	}

	queryFallback := `
		SELECT 
			COALESCE(m.codigo, '0000') AS meta_codigo,
			COALESCE(cm.codigo, '2.0.0.0.0.0') AS clasificador_codigo,
			COALESCE(pc.codigo_sunat, '') AS codigo_sunat_ingreso,
			COALESCE(pc.nombre_en_boleta, ct.nombre_personalizado, 'CONCEPTO SIN NOMBRE') AS nombre_ingreso
		FROM planilla_detalles pd
		JOIN planilla_conceptos pc ON pc.planilla_detalle_id = pd.id
		JOIN contratos c ON pd.contrato_id = c.id
		LEFT JOIN puestos p ON c.puesto_id = p.id
		LEFT JOIN metas_presupuestales m ON p.meta_id = m.id
		LEFT JOIN conceptos_tenant ct ON pc.concepto_tenant_id = ct.id
		LEFT JOIN clasificadores_mef cm ON ct.clasificador_id = cm.id
		WHERE pd.planilla_id = $1 AND c.tenant_id = $2
		  AND pc.tipo_concepto = 'INGRESO'
		  AND COALESCE(ct.es_remunerativa, false) = true
		GROUP BY m.codigo, cm.codigo, pc.codigo_sunat, COALESCE(pc.nombre_en_boleta, ct.nombre_personalizado, 'CONCEPTO SIN NOMBRE')
		ORDER BY SUM(pc.monto) DESC
		LIMIT 1
	`

	err = r.db.QueryRow(queryFallback, planillaID, tenantID).Scan(
		&target.MetaCodigo,
		&target.ClasificadorCodigo,
		&target.CodigoSunatIngreso,
		&target.NombreIngreso,
	)

	if err != nil {
		return target, fmt.Errorf("no se encontró meta ni clasificador remunerativo para ajuste: %w", err)
	}

	return target, nil
}

// ObtenerResumenConceptosPlanilla obtiene el consolidado de conceptos por tipo (INGRESO, RETENCION, APORTE) para el Anexo 1A
func (r *AnexoRepository) ObtenerResumenConceptosPlanilla(planillaID int, tenantID int) ([]models.ItemResumenConcepto, error) {
	query := `
		SELECT 
			pc.tipo_concepto,
			COALESCE(pc.codigo_sunat, '') AS codigo_sunat,
			COALESCE(pc.nombre_en_boleta, ct.nombre_personalizado, 'CONCEPTO SIN NOMBRE') AS nombre_concepto,
			SUM(pc.monto) AS monto_total
		FROM planilla_detalles pd
		JOIN planilla_conceptos pc ON pc.planilla_detalle_id = pd.id
		JOIN contratos c ON pd.contrato_id = c.id
		LEFT JOIN conceptos_tenant ct ON pc.concepto_tenant_id = ct.id
		WHERE pd.planilla_id = $1 AND c.tenant_id = $2
		GROUP BY pc.tipo_concepto, pc.codigo_sunat, COALESCE(pc.nombre_en_boleta, ct.nombre_personalizado, 'CONCEPTO SIN NOMBRE')
		ORDER BY 
			CASE pc.tipo_concepto 
				WHEN 'INGRESO' THEN 1 
				WHEN 'RETENCION' THEN 2 
				WHEN 'APORTE' THEN 3 
				ELSE 4 
			END, 
			nombre_concepto ASC
	`

	rows, err := r.db.Query(query, planillaID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("error al consultar resumen por conceptos de planilla: %w", err)
	}
	defer rows.Close()

	var items []models.ItemResumenConcepto
	for rows.Next() {
		var item models.ItemResumenConcepto
		if err := rows.Scan(
			&item.TipoConcepto,
			&item.CodigoSunat,
			&item.NombreConcepto,
			&item.MontoTotal,
		); err != nil {
			return nil, fmt.Errorf("error al escanear fila de resumen de conceptos: %w", err)
		}
		items = append(items, item)
	}

	return items, nil
}

// ObtenerResumenAFP obtiene la agregación por AFP para el Anexo 2
func (r *AnexoRepository) ObtenerResumenAFP(planillaID int, tenantID int) ([]models.ItemResumenAFP, error) {
	query := `
		SELECT 
			COALESCE(a.nombre, 'SIN AFP') AS afp_nombre,
			SUM(CASE WHEN pc.codigo_sunat = '0608' OR UPPER(pc.nombre_en_boleta) LIKE '%APORTE OBLIGATORIO%' THEN pc.monto ELSE 0 END) AS aporte_obligatorio,
			SUM(CASE WHEN pc.codigo_sunat = '0601' OR UPPER(pc.nombre_en_boleta) LIKE '%COMIS%' THEN pc.monto ELSE 0 END) AS comision,
			SUM(CASE WHEN pc.codigo_sunat = '0606' OR UPPER(pc.nombre_en_boleta) LIKE '%PRIMA%' THEN pc.monto ELSE 0 END) AS prima_seguro
		FROM planilla_detalles pd
		JOIN planilla_conceptos pc ON pc.planilla_detalle_id = pd.id
		JOIN contratos c ON pd.contrato_id = c.id
		JOIN trabajadores t ON c.trabajador_id = t.id
		LEFT JOIN afps a ON t.afp_id = a.id
		WHERE pd.planilla_id = $1 AND c.tenant_id = $2
		  AND pc.tipo_concepto = 'RETENCION'
		  AND (
			  pc.codigo_sunat IN ('0608', '0601', '0606')
			  OR UPPER(pc.nombre_en_boleta) LIKE '%AFP%'
			  OR UPPER(pc.nombre_en_boleta) LIKE '%APORTE OBLIGATORIO%'
			  OR UPPER(pc.nombre_en_boleta) LIKE '%PRIMA%SEGURO%'
		  )
		GROUP BY a.nombre
		ORDER BY a.nombre ASC
	`

	rows, err := r.db.Query(query, planillaID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("error al consultar resumen por AFP: %w", err)
	}
	defer rows.Close()

	var items []models.ItemResumenAFP
	for rows.Next() {
		var item models.ItemResumenAFP
		if err := rows.Scan(
			&item.AFPNombre,
			&item.AporteObligatorio,
			&item.Comision,
			&item.PrimaSeguro,
		); err != nil {
			return nil, fmt.Errorf("error al escanear fila de resumen por AFP: %w", err)
		}
		item.TotalAFP = item.AporteObligatorio + item.Comision + item.PrimaSeguro
		items = append(items, item)
	}

	return items, nil
}

// ObtenerDevengadoAFP obtiene el desglose por AFP, Meta y Clasificador MEF para el Anexo 2A
func (r *AnexoRepository) ObtenerDevengadoAFP(planillaID int, tenantID int) ([]models.ItemDevengadoAFP, error) {
	query := `
		SELECT 
			COALESCE(a.nombre, 'SIN AFP') AS afp_nombre,
			COALESCE(m.codigo, '0000') AS meta_codigo,
			COALESCE(cm.codigo, '2.0.0.0.0.0') AS clasificador_codigo,
			COALESCE(cm.descripcion, 'Sin Clasificador Gasto') AS clasificador_descripcion,
			SUM(CASE WHEN pc_ret.codigo_sunat = '0608' OR UPPER(pc_ret.nombre_en_boleta) LIKE '%APORTE OBLIGATORIO%' THEN pc_ret.monto ELSE 0 END) AS aporte_obligatorio,
			SUM(CASE WHEN pc_ret.codigo_sunat = '0601' OR UPPER(pc_ret.nombre_en_boleta) LIKE '%COMIS%' THEN pc_ret.monto ELSE 0 END) AS comision,
			SUM(CASE WHEN pc_ret.codigo_sunat = '0606' OR UPPER(pc_ret.nombre_en_boleta) LIKE '%PRIMA%' THEN pc_ret.monto ELSE 0 END) AS prima_seguro
		FROM planilla_detalles pd
		JOIN planilla_conceptos pc_ret ON pc_ret.planilla_detalle_id = pd.id
		JOIN contratos c ON pd.contrato_id = c.id
		JOIN trabajadores t ON c.trabajador_id = t.id
		LEFT JOIN afps a ON t.afp_id = a.id
		LEFT JOIN puestos p ON c.puesto_id = p.id
		LEFT JOIN metas_presupuestales m ON p.meta_id = m.id
		LEFT JOIN (
			SELECT DISTINCT ON (pd_ing.contrato_id)
				pd_ing.contrato_id,
				ct_ing.clasificador_id
			FROM planilla_detalles pd_ing
			JOIN planilla_conceptos pc_ing ON pc_ing.planilla_detalle_id = pd_ing.id
			JOIN conceptos_tenant ct_ing ON pc_ing.concepto_tenant_id = ct_ing.id
			WHERE pd_ing.planilla_id = $1 AND pc_ing.tipo_concepto = 'INGRESO'
			ORDER BY pd_ing.contrato_id, COALESCE(ct_ing.es_remunerativa, false) DESC, pc_ing.monto DESC
		) clasif_link ON clasif_link.contrato_id = pd.contrato_id
		LEFT JOIN clasificadores_mef cm ON clasif_link.clasificador_id = cm.id
		WHERE pd.planilla_id = $1 AND c.tenant_id = $2
		  AND pc_ret.tipo_concepto = 'RETENCION'
		  AND (
			  pc_ret.codigo_sunat IN ('0608', '0601', '0606')
			  OR UPPER(pc_ret.nombre_en_boleta) LIKE '%AFP%'
			  OR UPPER(pc_ret.nombre_en_boleta) LIKE '%APORTE OBLIGATORIO%'
			  OR UPPER(pc_ret.nombre_en_boleta) LIKE '%PRIMA%SEGURO%'
		  )
		GROUP BY a.nombre, m.codigo, cm.codigo, cm.descripcion
		ORDER BY a.nombre ASC, m.codigo ASC, cm.codigo ASC
	`

	rows, err := r.db.Query(query, planillaID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("error al consultar devengado por AFP: %w", err)
	}
	defer rows.Close()

	var items []models.ItemDevengadoAFP
	for rows.Next() {
		var item models.ItemDevengadoAFP
		if err := rows.Scan(
			&item.AFPNombre,
			&item.MetaCodigo,
			&item.ClasificadorCodigo,
			&item.ClasificadorDescripcion,
			&item.AporteObligatorio,
			&item.Comision,
			&item.PrimaSeguro,
		); err != nil {
			return nil, fmt.Errorf("error al escanear fila de devengado por AFP: %w", err)
		}
		item.TotalFila = item.AporteObligatorio + item.Comision + item.PrimaSeguro
		items = append(items, item)
	}

	return items, nil
}

// ObtenerRetencionesSunat obtiene las retenciones tributarias (ONP, Renta 4ta, Renta 5ta) desglosadas por Meta y Clasificador MEF para el Anexo 3
func (r *AnexoRepository) ObtenerRetencionesSunat(planillaID int, tenantID int) ([]models.ItemRetencionesSunat, error) {
	query := `
		SELECT 
			COALESCE(m.codigo, '0000') AS meta_codigo,
			COALESCE(cm.codigo, '2.0.0.0.0.0') AS clasificador_codigo,
			COALESCE(cm.descripcion, 'Sin Clasificador Gasto') AS clasificador_descripcion,
			SUM(CASE WHEN pc_ret.codigo_sunat = '0607' OR UPPER(pc_ret.nombre_en_boleta) LIKE '%ONP%' OR UPPER(pc_ret.nombre_en_boleta) LIKE '%19990%' THEN pc_ret.monto ELSE 0 END) AS onp,
			SUM(CASE WHEN pc_ret.codigo_sunat = 'S101' OR UPPER(pc_ret.nombre_en_boleta) LIKE '%CUARTA%' OR UPPER(pc_ret.nombre_en_boleta) LIKE '%4TA%' THEN pc_ret.monto ELSE 0 END) AS renta_4ta,
			SUM(CASE WHEN pc_ret.codigo_sunat = '0605' OR UPPER(pc_ret.nombre_en_boleta) LIKE '%QUINTA%' OR UPPER(pc_ret.nombre_en_boleta) LIKE '%5TA%' THEN pc_ret.monto ELSE 0 END) AS renta_5ta
		FROM planilla_detalles pd
		JOIN planilla_conceptos pc_ret ON pc_ret.planilla_detalle_id = pd.id
		JOIN contratos c ON pd.contrato_id = c.id
		LEFT JOIN puestos p ON c.puesto_id = p.id
		LEFT JOIN metas_presupuestales m ON p.meta_id = m.id
		LEFT JOIN (
			SELECT DISTINCT ON (pd_ing.contrato_id)
				pd_ing.contrato_id,
				ct_ing.clasificador_id
			FROM planilla_detalles pd_ing
			JOIN planilla_conceptos pc_ing ON pc_ing.planilla_detalle_id = pd_ing.id
			JOIN conceptos_tenant ct_ing ON pc_ing.concepto_tenant_id = ct_ing.id
			WHERE pd_ing.planilla_id = $1 AND pc_ing.tipo_concepto = 'INGRESO'
			ORDER BY pd_ing.contrato_id, COALESCE(ct_ing.es_remunerativa, false) DESC, pc_ing.monto DESC
		) clasif_link ON clasif_link.contrato_id = pd.contrato_id
		LEFT JOIN clasificadores_mef cm ON clasif_link.clasificador_id = cm.id
		WHERE pd.planilla_id = $1 AND c.tenant_id = $2
		  AND pc_ret.tipo_concepto = 'RETENCION'
		  AND (
			  pc_ret.codigo_sunat IN ('0607', 'S101', '0605')
			  OR UPPER(pc_ret.nombre_en_boleta) LIKE '%ONP%'
			  OR UPPER(pc_ret.nombre_en_boleta) LIKE '%19990%'
			  OR UPPER(pc_ret.nombre_en_boleta) LIKE '%CUARTA%'
			  OR UPPER(pc_ret.nombre_en_boleta) LIKE '%4TA%'
			  OR UPPER(pc_ret.nombre_en_boleta) LIKE '%QUINTA%'
			  OR UPPER(pc_ret.nombre_en_boleta) LIKE '%5TA%'
		  )
		GROUP BY m.codigo, cm.codigo, cm.descripcion
		ORDER BY m.codigo ASC, cm.codigo ASC
	`

	rows, err := r.db.Query(query, planillaID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("error al consultar retenciones SUNAT: %w", err)
	}
	defer rows.Close()

	var items []models.ItemRetencionesSunat
	for rows.Next() {
		var item models.ItemRetencionesSunat
		if err := rows.Scan(
			&item.MetaCodigo,
			&item.ClasificadorCodigo,
			&item.ClasificadorDescripcion,
			&item.ONP,
			&item.Renta4ta,
			&item.Renta5ta,
		); err != nil {
			return nil, fmt.Errorf("error al escanear fila de retenciones SUNAT: %w", err)
		}
		item.TotalFila = item.ONP + item.Renta4ta + item.Renta5ta
		items = append(items, item)
	}

	return items, nil
}
