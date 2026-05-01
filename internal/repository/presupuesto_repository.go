package repository

import (
	"database/sql"
	"planilla-rgm/internal/models"
)

type PresupuestoRepository struct {
	db *sql.DB
}

func NewPresupuestoRepository(db *sql.DB) *PresupuestoRepository {
	return &PresupuestoRepository{db: db}
}

// CrearVersion guarda la cabecera y retorna el ID generado
func (r *PresupuestoRepository) CrearVersion(v *models.PapVersion) error {
	query := `INSERT INTO pap_versiones (tenant_id, anio, tipo, estado) VALUES ($1, $2, $3, 'CERRADA') RETURNING id`
	return r.db.QueryRow(query, v.TenantID, v.Anio, v.Tipo).Scan(&v.ID)
}

// GuardarDetallesMasivo inserta toda la matriz generada en un solo bloque (transacción)
func (r *PresupuestoRepository) GuardarDetallesMasivo(versionID int, detalles []models.PapDetalle) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(`
		INSERT INTO pap_detalles (
			version_id, meta_codigo, meta_descripcion, fuente_rubro_codigo, fuente_rubro_descripcion,
			clasificador_codigo_limpio, clasificador_descripcion,
			mes_01, mes_02, mes_03, mes_04, mes_05, mes_06, mes_07, mes_08, mes_09, mes_10, mes_11, mes_12, total_anual
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
	`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, d := range detalles {
		_, err := stmt.Exec(
			versionID, d.MetaCodigo, d.MetaDescripcion, d.FuenteRubroCodigo, d.FuenteRubroDescripcion,
			d.ClasificadorCodigoLimpio, d.ClasificadorDescripcion,
			d.Meses[0], d.Meses[1], d.Meses[2], d.Meses[3], d.Meses[4], d.Meses[5],
			d.Meses[6], d.Meses[7], d.Meses[8], d.Meses[9], d.Meses[10], d.Meses[11],
			d.TotalAnual,
		)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// ObtenerMatrizPorAnio extrae los detalles del PIA guardado, filtrando SOLO la versión más reciente del año
func (r *PresupuestoRepository) ObtenerMatrizPorAnio(tenantID int, anio int) ([]models.PapDetalle, error) {
	query := `
		SELECT d.id, d.version_id, d.meta_codigo, d.meta_descripcion, 
		       d.fuente_rubro_codigo, d.fuente_rubro_descripcion, 
		       d.clasificador_codigo_limpio, d.clasificador_descripcion,
		       d.mes_01, d.mes_02, d.mes_03, d.mes_04, d.mes_05, d.mes_06, 
		       d.mes_07, d.mes_08, d.mes_09, d.mes_10, d.mes_11, d.mes_12, d.total_anual
		FROM pap_detalles d
		INNER JOIN pap_versiones v ON d.version_id = v.id
		-- AQUI ESTÁ EL CAMBIO CLAVE: Usamos una subconsulta para obtener solo el último ID generado
		WHERE v.id = (
			SELECT id FROM pap_versiones 
			WHERE tenant_id = $1 AND anio = $2 AND tipo = 'PIA' 
			ORDER BY id DESC LIMIT 1
		)
		-- Ordenamos para que la matriz tenga sentido contable
		ORDER BY d.meta_codigo ASC, d.fuente_rubro_codigo ASC, d.clasificador_codigo_limpio ASC
	`

	rows, err := r.db.Query(query, tenantID, anio)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.PapDetalle
	for rows.Next() {
		var d models.PapDetalle
		err := rows.Scan(
			&d.ID, &d.VersionID, &d.MetaCodigo, &d.MetaDescripcion,
			&d.FuenteRubroCodigo, &d.FuenteRubroDescripcion,
			&d.ClasificadorCodigoLimpio, &d.ClasificadorDescripcion,
			&d.Meses[0], &d.Meses[1], &d.Meses[2], &d.Meses[3], &d.Meses[4], &d.Meses[5],
			&d.Meses[6], &d.Meses[7], &d.Meses[8], &d.Meses[9], &d.Meses[10], &d.Meses[11],
			&d.TotalAnual,
		)
		if err == nil {
			lista = append(lista, d)
		}
	}
	return lista, nil
}

// ObtenerGastoRealPorPuesto extrae la suma de ingresos y aportes reales de un puesto
// leyendo las planillas ya cerradas y pagadas en un mes específico.
func (r *PlanillaRepository) ObtenerGastoRealPorPuesto(puestoID int, anio int, mes int) (float64, float64, error) {
	query := `
		SELECT 
			COALESCE(SUM(pd.total_ingresos), 0) AS ingresos_reales,
			COALESCE(SUM(pd.total_aportes), 0) AS aportes_reales
		FROM planilla_detalles pd
		INNER JOIN planillas p ON pd.planilla_id = p.id
		INNER JOIN contratos c ON pd.contrato_id = c.id
		WHERE c.puesto_id = $1
		  AND p.anio = $2
		  AND p.mes = $3
		  AND p.estado = 'CERRADA'
	`
	
	var ingresos, aportes float64
	// Usamos QueryRow porque SUM siempre devuelve una sola fila, 
	// y COALESCE asegura que devuelva 0 si no hay planillas ese mes.
	err := r.db.QueryRow(query, puestoID, anio, mes).Scan(&ingresos, &aportes)
	
	return ingresos, aportes, err
}
