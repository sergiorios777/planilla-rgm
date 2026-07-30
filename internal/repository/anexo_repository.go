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

// ObtenerTargetCompromisoAjuste busca la combinación (Meta, Clasificador MEF) remunerativa para aplicar el ajuste de redondeo
func (r *AnexoRepository) ObtenerTargetCompromisoAjuste(planillaID int, tenantID int, codigosSunat []string, palabraClave string) (metaCodigo string, clasificadorCodigo string, err error) {
	// 1. Intentar buscar en trabajadores que tengan la retención específica
	queryEspecifico := `
		SELECT 
			COALESCE(m.codigo, '0000') AS meta_codigo,
			COALESCE(cm.codigo, '2.0.0.0.0.0') AS clasificador_codigo
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
			  SELECT_SUB:
			  FROM planilla_detalles pd_sub
			  JOIN planilla_conceptos pc_sub ON pc_sub.planilla_detalle_id = pd_sub.id
			  WHERE pd_sub.planilla_id = $1 
			    AND (pc_sub.codigo_sunat = ANY($3) OR UPPER(pc_sub.nombre_en_boleta) LIKE $4)
		  )
		GROUP BY m.codigo, cm.codigo
		ORDER BY SUM(pc.monto) DESC
		LIMIT 1
	`

	// Limpiamos el subquery ficticio
	queryEspecifico = strings.Replace(queryEspecifico, "SELECT_SUB:", "", 1)

	patron := "%" + strings.ToUpper(palabraClave) + "%"
	err = r.db.QueryRow(queryEspecifico, planillaID, tenantID, pq.Array(codigosSunat), patron).Scan(&metaCodigo, &clasificadorCodigo)
	if err == nil && metaCodigo != "" && clasificadorCodigo != "" {
		return metaCodigo, clasificadorCodigo, nil
	}

	// 2. Fallback: Seleccionar la mayor meta y clasificador remunerativo de la planilla
	queryFallback := `
		SELECT 
			COALESCE(m.codigo, '0000') AS meta_codigo,
			COALESCE(cm.codigo, '2.0.0.0.0.0') AS clasificador_codigo
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
		GROUP BY m.codigo, cm.codigo
		ORDER BY SUM(pc.monto) DESC
		LIMIT 1
	`

	err = r.db.QueryRow(queryFallback, planillaID, tenantID).Scan(&metaCodigo, &clasificadorCodigo)
	if err != nil {
		return "", "", fmt.Errorf("no se encontró meta ni clasificador remunerativo para ajuste: %w", err)
	}

	return metaCodigo, clasificadorCodigo, nil
}
