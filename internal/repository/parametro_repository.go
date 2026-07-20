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
func (r *ParametroRepository) ObtenerTodos(busqueda string) ([]models.ParametroGlobal, error) {
	query := `
		SELECT id, clave, valor, TO_CHAR(fecha_desde, 'YYYY-MM-DD'), TO_CHAR(fecha_hasta, 'YYYY-MM-DD'), descripcion 
		FROM parametros_globales 
		WHERE clave ILIKE $1 OR valor::text ILIKE $1 OR descripcion ILIKE $1
		ORDER BY clave ASC, fecha_desde DESC
	`
	busqueda = "%" + busqueda + "%"
	rows, err := r.db.Query(query, busqueda)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.ParametroGlobal
	for rows.Next() {
		var p models.ParametroGlobal
		var fechaHasta sql.NullString // Variable intermedia para manejar nulos de la BD
		var descripcion sql.NullString

		if err := rows.Scan(&p.ID, &p.Clave, &p.Valor, &p.FechaDesde, &fechaHasta, &descripcion); err != nil {
			return nil, err
		}

		if fechaHasta.Valid {
			p.FechaHasta = &fechaHasta.String
		}

		if descripcion.Valid {
			p.Descripcion = descripcion.String
		}

		lista = append(lista, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return lista, nil
}

// ObtenerPorID busca un parámetro específico por su ID
func (r *ParametroRepository) ObtenerPorID(id int) (models.ParametroGlobal, error) {
	var p models.ParametroGlobal
	var fechaHasta sql.NullString
	var descripcion sql.NullString

	query := `
		SELECT id, clave, valor, TO_CHAR(fecha_desde, 'YYYY-MM-DD'), TO_CHAR(fecha_hasta, 'YYYY-MM-DD'), descripcion 
		FROM parametros_globales 
		WHERE id = $1
	`
	err := r.db.QueryRow(query, id).Scan(&p.ID, &p.Clave, &p.Valor, &p.FechaDesde, &fechaHasta, &descripcion)
	if err != nil {
		return p, err
	}

	if fechaHasta.Valid {
		p.FechaHasta = &fechaHasta.String
	}

	if descripcion.Valid {
		p.Descripcion = descripcion.String
	}

	return p, nil
}

// Actualizar actualiza un parámetro global
func (r *ParametroRepository) Actualizar(p *models.ParametroGlobal) error {
	query := `
		UPDATE parametros_globales 
		SET 
			clave = $1, 
			valor = $2, 
			fecha_desde = $3, 
			fecha_hasta = $4, 
			descripcion = $5, 
			updated_at = CURRENT_TIMESTAMP 
		WHERE id = $6
	`
	_, err := r.db.Exec(query, p.Clave, p.Valor, p.FechaDesde, p.FechaHasta, p.Descripcion, p.ID)
	return err
}
