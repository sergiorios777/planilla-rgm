package repository

import (
	"database/sql"
	"planilla-rgm/internal/models"
)

// NotificacionRepository gestiona el almacenamiento y consulta de notificaciones en PostgreSQL
type NotificacionRepository struct {
	db *sql.DB
}

// NewNotificacionRepository crea una nueva instancia de NotificacionRepository
func NewNotificacionRepository(db *sql.DB) *NotificacionRepository {
	return &NotificacionRepository{db: db}
}

// ContarNoLeidas cuenta los avisos pendientes de lectura para el usuario actual
func (r *NotificacionRepository) ContarNoLeidas(tenantID *int, usuarioID *int) (int, error) {
	var query string
	var count int
	var err error

	if tenantID == nil {
		// Super Admin (Global)
		query = "SELECT COUNT(*) FROM notificaciones WHERE tenant_id IS NULL AND leido = false"
		err = r.db.QueryRow(query).Scan(&count)
	} else {
		// Tenant User (Filtra por entidad y usuario, incluyendo notificaciones generales del tenant)
		query = `
			SELECT COUNT(*) FROM notificaciones 
			WHERE tenant_id = $1 AND (usuario_id = $2 OR usuario_id IS NULL) AND leido = false`
		err = r.db.QueryRow(query, *tenantID, *usuarioID).Scan(&count)
	}

	return count, err
}

// ObtenerRecientes lista las últimas notificaciones ordenadas cronológicamente
func (r *NotificacionRepository) ObtenerRecientes(tenantID *int, usuarioID *int, limite int) ([]models.Notificacion, error) {
	var query string
	var rows *sql.Rows
	var err error

	if tenantID == nil {
		// Super Admin
		query = `
			SELECT id, tenant_id, usuario_id, titulo, mensaje, tipo, leido, created_at 
			FROM notificaciones 
			WHERE tenant_id IS NULL 
			ORDER BY created_at DESC LIMIT $1`
		rows, err = r.db.Query(query, limite)
	} else {
		// Tenant
		query = `
			SELECT id, tenant_id, usuario_id, titulo, mensaje, tipo, leido, created_at 
			FROM notificaciones 
			WHERE tenant_id = $1 AND (usuario_id = $2 OR usuario_id IS NULL) 
			ORDER BY created_at DESC LIMIT $3`
		rows, err = r.db.Query(query, *tenantID, *usuarioID, limite)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.Notificacion
	for rows.Next() {
		var n models.Notificacion
		var tID, uID sql.NullInt64
		err := rows.Scan(
			&n.ID, &tID, &uID, &n.Titulo, &n.Mensaje, &n.Tipo, &n.Leido, &n.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if tID.Valid {
			v := int(tID.Int64)
			n.TenantID = &v
		}
		if uID.Valid {
			v := int(uID.Int64)
			n.UsuarioID = &v
		}
		lista = append(lista, n)
	}

	return lista, nil
}

// MarcarComoLeidas marca todas las notificaciones pendientes del usuario como leídas
func (r *NotificacionRepository) MarcarComoLeidas(tenantID *int, usuarioID *int) error {
	var query string
	var err error

	if tenantID == nil {
		query = "UPDATE notificaciones SET leido = true WHERE tenant_id IS NULL AND leido = false"
		_, err = r.db.Exec(query)
	} else {
		query = `
			UPDATE notificaciones SET leido = true 
			WHERE tenant_id = $1 AND (usuario_id = $2 OR usuario_id IS NULL) AND leido = false`
		_, err = r.db.Exec(query, *tenantID, *usuarioID)
	}

	return err
}

// Crear inserta una nueva notificación en la base de datos
func (r *NotificacionRepository) Crear(n *models.Notificacion) error {
	query := `
		INSERT INTO notificaciones (tenant_id, usuario_id, titulo, mensaje, tipo, leido) 
		VALUES ($1, $2, $3, $4, $5, $6) 
		RETURNING id, created_at`

	var tIDVal, uIDVal interface{}
	if n.TenantID != nil {
		tIDVal = *n.TenantID
	}
	if n.UsuarioID != nil {
		uIDVal = *n.UsuarioID
	}

	return r.db.QueryRow(query, tIDVal, uIDVal, n.Titulo, n.Mensaje, n.Tipo, n.Leido).Scan(&n.ID, &n.CreatedAt)
}
