package repository

import (
	"database/sql"
	"planilla-rgm/internal/models"
)

type ParametroRepository struct {
	db *sql.DB
}

func NewParametroRepository(db *sql.DB) *ParametroRepository {
	return &ParametroRepository{db: db}
}

// Guardar inserta o actualiza un parámetro global (Upsert)
func (r *ParametroRepository) Guardar(p *models.ParametroGlobal) error {
	query := `
		INSERT INTO parametros_globales (clave, valor, fecha_desde, fecha_hasta, descripcion, updated_at) 
		VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP)
		ON CONFLICT (clave, fecha_desde) DO UPDATE SET 
			valor = EXCLUDED.valor, 
			fecha_hasta = EXCLUDED.fecha_hasta,
			descripcion = EXCLUDED.descripcion,
			updated_at = CURRENT_TIMESTAMP
		RETURNING id`

	return r.db.QueryRow(query, p.Clave, p.Valor, p.FechaDesde, p.FechaHasta, p.Descripcion).Scan(&p.ID)
}

// ObtenerTodos lista los parámetros ordenados por año (más reciente primero) y luego por clave
func (r *ParametroRepository) ObtenerTodos() ([]models.ParametroGlobal, error) {
	query := `
		SELECT id, clave, valor, TO_CHAR(fecha_desde, 'YYYY-MM-DD'), TO_CHAR(fecha_hasta, 'YYYY-MM-DD'), descripcion 
		FROM parametros_globales 
		ORDER BY clave ASC, fecha_desde DESC`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.ParametroGlobal
	for rows.Next() {
		var p models.ParametroGlobal
		var fechaHasta sql.NullString // Variable intermedia para manejar nulos de la BD

		if err := rows.Scan(&p.ID, &p.Clave, &p.Valor, &p.FechaDesde, &fechaHasta, &p.Descripcion); err != nil {
			return nil, err
		}

		if fechaHasta.Valid {
			p.FechaHasta = &fechaHasta.String
		}

		lista = append(lista, p)
	}
	return lista, nil
}
