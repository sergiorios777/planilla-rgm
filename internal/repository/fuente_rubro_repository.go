package repository

import (
	"database/sql"
	"planilla-rgm/internal/models"
)

type FuenteRubroRepository struct {
	db *sql.DB
}

func NewFuenteRubroRepository(db *sql.DB) *FuenteRubroRepository {
	return &FuenteRubroRepository{db: db}
}

// ObtenerPorAnio lista el catálogo para un año específico
func (r *FuenteRubroRepository) ObtenerPorAnio(anio int, buscar string) ([]models.FuenteRubro, error) {
	query := `
		SELECT id, anio, fuente_financiamiento, rubro, activo 
		FROM fuentes_rubros 
		WHERE anio = $1 AND (fuente_financiamiento ILIKE '%' || $2 || '%' OR rubro ILIKE '%' || $2 || '%')
		ORDER BY fuente_financiamiento, rubro ASC
	`
	rows, err := r.db.Query(query, anio, buscar)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.FuenteRubro
	for rows.Next() {
		var fr models.FuenteRubro
		err := rows.Scan(&fr.ID, &fr.Anio, &fr.FuenteFinanciamiento, &fr.Rubro, &fr.Activo)
		if err == nil {
			lista = append(lista, fr)
		}
	}
	return lista, nil
}
