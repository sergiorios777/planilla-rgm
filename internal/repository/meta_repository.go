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

// ObtenerPorID trae los datos de una meta específica para el formulario de edición
func (r *MetaRepository) ObtenerPorID(id int, tenantID int) (models.MetaPresupuestal, error) {
	var m models.MetaPresupuestal
	query := `SELECT id, tenant_id, anio, codigo, descripcion, activo FROM metas_presupuestales WHERE id = $1 AND tenant_id = $2`
	err := r.db.QueryRow(query, id, tenantID).Scan(&m.ID, &m.TenantID, &m.Anio, &m.Codigo, &m.Descripcion, &m.Activo)
	return m, err
}

// Actualizar guarda los cambios de la meta
func (r *MetaRepository) Actualizar(m *models.MetaPresupuestal) error {
	// Omitimos actualizar el Año por seguridad contable, solo código, descripción y estado
	query := `UPDATE metas_presupuestales SET codigo = $1, descripcion = $2, activo = $3 WHERE id = $4 AND tenant_id = $5`
	_, err := r.db.Exec(query, m.Codigo, m.Descripcion, m.Activo, m.ID, m.TenantID)
	return err
}
