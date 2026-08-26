package repository

import (
	"database/sql"
	"planilla-rgm/internal/models"
)

type EntidadFinancieraRepository struct {
	db *sql.DB
}

func NewEntidadFinancieraRepository(db *sql.DB) *EntidadFinancieraRepository {
	return &EntidadFinancieraRepository{db: db}
}

// ListarTodas obtiene el catálogo de entidades financieras activas
func (r *EntidadFinancieraRepository) ListarTodas() ([]models.EntidadFinanciera, error) {
	query := `SELECT id, codigo, nombre, activo, created_at FROM entidades_financieras WHERE activo = true ORDER BY codigo ASC`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.EntidadFinanciera
	for rows.Next() {
		var ef models.EntidadFinanciera
		if err := rows.Scan(&ef.ID, &ef.Codigo, &ef.Nombre, &ef.Activo, &ef.CreatedAt); err != nil {
			return nil, err
		}
		lista = append(lista, ef)
	}
	return lista, nil
}

// ObtenerPorID obtiene una entidad financiera específica
func (r *EntidadFinancieraRepository) ObtenerPorID(id int) (*models.EntidadFinanciera, error) {
	query := `SELECT id, codigo, nombre, activo, created_at FROM entidades_financieras WHERE id = $1`
	var ef models.EntidadFinanciera
	err := r.db.QueryRow(query, id).Scan(&ef.ID, &ef.Codigo, &ef.Nombre, &ef.Activo, &ef.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &ef, nil
}
