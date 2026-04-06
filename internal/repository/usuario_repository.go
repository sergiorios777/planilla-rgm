package repository

import (
	"database/sql"
	"planilla-rgm/internal/models"
)

type UsuarioRepository struct {
	db *sql.DB
}

func NewUsuarioRepository(db *sql.DB) *UsuarioRepository {
	return &UsuarioRepository{db: db}
}

// ObtenerPorEmail busca un usuario por su correo electrónico para el inicio de sesión
func (r *UsuarioRepository) ObtenerPorEmail(email string) (*models.Usuario, error) {
	var u models.Usuario

	// Nota: password_hash es como nombramos la columna en la migración original
	query := `SELECT id, tenant_id, nombre, email, password_hash, rol FROM usuarios WHERE email = $1`

	err := r.db.QueryRow(query, email).Scan(&u.ID, &u.TenantID, &u.Nombre, &u.Email, &u.Password, &u.Rol)
	if err != nil {
		return nil, err // Retornará un error sql.ErrNoRows si no lo encuentra
	}

	return &u, nil
}

// Crear inserta un nuevo usuario (útil para registrar al primer admin y luego a los inquilinos)
func (r *UsuarioRepository) Crear(u *models.Usuario) error {
	query := `INSERT INTO usuarios (tenant_id, nombre, email, password_hash, rol) 
	          VALUES ($1, $2, $3, $4, $5) RETURNING id`

	return r.db.QueryRow(query, u.TenantID, u.Nombre, u.Email, u.Password, u.Rol).Scan(&u.ID)
}
