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
