package repository

import (
	"database/sql"
	"planilla-rgm/internal/models"
	"time"
)

// AdminTareaRepository gestiona las consultas SQL para las tareas programadas en PostgreSQL
type AdminTareaRepository struct {
	db *sql.DB
}

// NewAdminTareaRepository crea una nueva instancia de AdminTareaRepository
func NewAdminTareaRepository(db *sql.DB) *AdminTareaRepository {
	return &AdminTareaRepository{db: db}
}

// ObtenerTodos lista todas las tareas de la administración del SaaS
func (r *AdminTareaRepository) ObtenerTodos(buscar string) ([]models.AdminTarea, error) {
	query := `
		SELECT id, titulo, COALESCE(descripcion, '') as descripcion, recurrencia, 
		       fecha_vencimiento, proximo_aviso, notificado_email, activo, created_at 
		FROM admin_tareas 
		WHERE titulo ILIKE '%' || $1 || '%' OR descripcion ILIKE '%' || $1 || '%' 
		ORDER BY fecha_vencimiento ASC`

	rows, err := r.db.Query(query, buscar)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.AdminTarea
	for rows.Next() {
		var t models.AdminTarea
		err := rows.Scan(
			&t.ID, &t.Titulo, &t.Descripcion, &t.Recurrencia,
			&t.FechaVencimiento, &t.ProximoAviso, &t.NotificadoEmail, &t.Activo, &t.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		lista = append(lista, t)
	}

	return lista, nil
}

// ObtenerPorID obtiene una tarea programada específica
func (r *AdminTareaRepository) ObtenerPorID(id int) (*models.AdminTarea, error) {
	var t models.AdminTarea
	query := `
		SELECT id, titulo, COALESCE(descripcion, '') as descripcion, recurrencia, 
		       fecha_vencimiento, proximo_aviso, notificado_email, activo, created_at 
		FROM admin_tareas 
		WHERE id = $1`

	err := r.db.QueryRow(query, id).Scan(
		&t.ID, &t.Titulo, &t.Descripcion, &t.Recurrencia,
		&t.FechaVencimiento, &t.ProximoAviso, &t.NotificadoEmail, &t.Activo, &t.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &t, nil
}

// Crear registra una nueva tarea programada
func (r *AdminTareaRepository) Crear(t *models.AdminTarea) error {
	query := `
		INSERT INTO admin_tareas (titulo, descripcion, recurrencia, fecha_vencimiento, proximo_aviso, activo) 
		VALUES ($1, $2, $3, $4, $5, $6) 
		RETURNING id, created_at`

	return r.db.QueryRow(
		query, t.Titulo, t.Descripcion, t.Recurrencia, t.FechaVencimiento, t.ProximoAviso, t.Activo,
	).Scan(&t.ID, &t.CreatedAt)
}

// Actualizar guarda los cambios en una tarea existente
func (r *AdminTareaRepository) Actualizar(t *models.AdminTarea) error {
	query := `
		UPDATE admin_tareas 
		SET titulo = $1, descripcion = $2, recurrencia = $3, fecha_vencimiento = $4, proximo_aviso = $5, activo = $6 
		WHERE id = $7`

	_, err := r.db.Exec(
		query, t.Titulo, t.Descripcion, t.Recurrencia, t.FechaVencimiento, t.ProximoAviso, t.Activo, t.ID,
	)
	return err
}

// ObtenerTareasVencidas busca todas las tareas pendientes de notificar para el Daemon
func (r *AdminTareaRepository) ObtenerTareasVencidas(ahora time.Time) ([]models.AdminTarea, error) {
	query := `
		SELECT id, titulo, COALESCE(descripcion, '') as descripcion, recurrencia, 
		       fecha_vencimiento, proximo_aviso, notificado_email, activo, created_at 
		FROM admin_tareas 
		WHERE proximo_aviso <= $1 AND activo = true`

	rows, err := r.db.Query(query, ahora)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.AdminTarea
	for rows.Next() {
		var t models.AdminTarea
		err := rows.Scan(
			&t.ID, &t.Titulo, &t.Descripcion, &t.Recurrencia,
			&t.FechaVencimiento, &t.ProximoAviso, &t.NotificadoEmail, &t.Activo, &t.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		lista = append(lista, t)
	}

	return lista, nil
}

// ActualizarProximoAviso avanza el aviso de una tarea tras haber sido procesada
func (r *AdminTareaRepository) ActualizarProximoAviso(id int, nuevoAviso time.Time, desactivar bool) error {
	query := `UPDATE admin_tareas SET proximo_aviso = $1, activo = $2 WHERE id = $3`
	activo := !desactivar
	_, err := r.db.Exec(query, nuevoAviso, activo, id)
	return err
}
