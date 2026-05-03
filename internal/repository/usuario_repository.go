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

// 1. ObtenerPorEmail busca un usuario por su correo electrónico para el inicio de sesión
func (r *UsuarioRepository) ObtenerPorEmail(email string) (*models.Usuario, error) {
	var u models.Usuario

	// Nota: password_hash es como nombramos la columna en la migración original
	query := `SELECT id, tenant_id, nombre, email, password_hash, rol, activo FROM usuarios WHERE email = $1`

	err := r.db.QueryRow(query, email).Scan(&u.ID, &u.TenantID, &u.Nombre, &u.Email, &u.Password, &u.Rol, &u.Activo)
	if err != nil {
		return nil, err // Retornará un error sql.ErrNoRows si no lo encuentra
	}

	return &u, nil
}

// 2. Crear inserta un nuevo usuario (útil para registrar al primer admin y luego a los inquilinos)
func (r *UsuarioRepository) Crear(u *models.Usuario) error {
	query := `INSERT INTO usuarios (tenant_id, nombre, email, password_hash, rol, activo) 
	          VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`

	return r.db.QueryRow(query, u.TenantID, u.Nombre, u.Email, u.Password, u.Rol, u.Activo).Scan(&u.ID)
}

// 3. ObtenerTodos lista los usuarios y el nombre de su municipalidad (si tienen)
func (r *UsuarioRepository) ObtenerTodos(busqueda string) ([]models.Usuario, error) {
	query := `
		SELECT u.id, u.tenant_id, u.nombre, u.email, u.rol, u.activo, COALESCE(t.nombre, 'Súper Admin (SaaS)') as tenant_nombre
		FROM usuarios u
		LEFT JOIN tenants t ON u.tenant_id = t.id
		WHERE u.nombre ILIKE '%' || $1 || '%' OR u.email ILIKE '%' || $1 || '%' OR t.nombre ILIKE '%' || $1 || '%'
		ORDER BY t.id ASC
	`
	rows, err := r.db.Query(query, busqueda)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.Usuario
	for rows.Next() {
		var u models.Usuario
		err := rows.Scan(&u.ID, &u.TenantID, &u.Nombre, &u.Email, &u.Rol, &u.Activo, &u.TenantNombre)
		if err != nil {
			return nil, err
		}
		lista = append(lista, u)
	}
	return lista, nil
}

// 4. ObtenerPorID busca a un usuario específico
func (r *UsuarioRepository) ObtenerPorID(id int) (*models.Usuario, error) {
	var u models.Usuario
	query := `SELECT id, tenant_id, nombre, email, rol, activo FROM usuarios WHERE id = $1`
	err := r.db.QueryRow(query, id).Scan(&u.ID, &u.TenantID, &u.Nombre, &u.Email, &u.Rol, &u.Activo)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// 5. Actualizar guarda los cambios. Si la contraseña viene llena, la actualiza.
func (r *UsuarioRepository) Actualizar(u *models.Usuario) error {
	if u.Password != "" {
		// Actualiza incluyendo la contraseña (ya debe venir encriptada desde el handler)
		query := `UPDATE usuarios SET tenant_id = $1, nombre = $2, email = $3, password_hash = $4, rol = $5, activo = $6 WHERE id = $7`
		_, err := r.db.Exec(query, u.TenantID, u.Nombre, u.Email, u.Password, u.Rol, u.Activo, u.ID)
		return err
	}

	// Actualiza sin tocar la contraseña existente
	query := `UPDATE usuarios SET tenant_id = $1, nombre = $2, email = $3, rol = $4, activo = $5 WHERE id = $6`
	_, err := r.db.Exec(query, u.TenantID, u.Nombre, u.Email, u.Rol, u.Activo, u.ID)
	return err
}
