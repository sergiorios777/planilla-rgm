package repository

import (
	"database/sql"
	"planilla-rgm/internal/models"
)

type MetaRepository struct {
	db *sql.DB
}

func NewMetaRepository(db *sql.DB) *MetaRepository {
	return &MetaRepository{db: db}
}

func (r *MetaRepository) ObtenerTodos(tenantID int) ([]models.MetaPresupuestal, error) {
	query := `SELECT id, tenant_id, anio, codigo, descripcion, activo FROM metas_presupuestales WHERE tenant_id = $1 ORDER BY anio DESC, codigo ASC`
	rows, err := r.db.Query(query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.MetaPresupuestal
	for rows.Next() {
		var m models.MetaPresupuestal
		err := rows.Scan(&m.ID, &m.TenantID, &m.Anio, &m.Codigo, &m.Descripcion, &m.Activo)
		if err == nil {
			lista = append(lista, m)
		}
	}
	return lista, nil
}

func (r *MetaRepository) Crear(m *models.MetaPresupuestal) error {
	query := `INSERT INTO metas_presupuestales (tenant_id, anio, codigo, descripcion, activo) VALUES ($1, $2, $3, $4, $5) RETURNING id`
	return r.db.QueryRow(query, m.TenantID, m.Anio, m.Codigo, m.Descripcion, m.Activo).Scan(&m.ID)
}
