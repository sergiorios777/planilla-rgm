package repository

import (
	"database/sql"
	"planilla-rgm/internal/models"
)

// TenantRepository gestiona la comunicación con la tabla 'tenants'
type TenantRepository struct {
	db *sql.DB
}

// NewTenantRepository es el constructor que recibe la conexión de la base de datos
func NewTenantRepository(db *sql.DB) *TenantRepository {
	return &TenantRepository{db: db}
}

// ObtenerTodos ejecuta un SELECT para traer la lista de inquilinos
func (r *TenantRepository) ObtenerTodos() ([]models.Tenant, error) {
	// Escribimos la consulta SQL
	query := `SELECT id, nombre, ruc, direccion, frase_gestion, logo_url, slug, activo, created_at FROM tenants ORDER BY id DESC`

	// Ejecutamos la consulta
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close() // Cerramos las filas al terminar para liberar memoria

	var lista []models.Tenant

	// Recorremos los resultados fila por fila
	for rows.Next() {
		var t models.Tenant
		// Escaneamos los datos de la fila de PostgreSQL hacia nuestra estructura de Go
		if err := rows.Scan(&t.ID, &t.Nombre, &t.Ruc, &t.Direccion, &t.FraseGestion, &t.LogoURL, &t.Slug, &t.Activo, &t.CreatedAt); err != nil {
			return nil, err
		}
		lista = append(lista, t)
	}

	return lista, nil
}

// Crear inserta un nuevo inquilino en la base de datos
func (r *TenantRepository) Crear(t *models.Tenant) error {
	// Escribimos la consulta SQL. RETURNING nos devuelve el ID y la fecha generados.
	query := `INSERT INTO tenants (nombre, ruc, direccion, frase_gestion, logo_url, slug, activo) 
	          VALUES ($1, $2, $3, $4, $5, $6, $7) 
	          RETURNING id, created_at`

	// Ejecutamos la consulta y guardamos el ID y la fecha devueltos en nuestra estructura
	err := r.db.QueryRow(query, t.Nombre, t.Ruc, t.Direccion, t.FraseGestion, t.LogoURL, t.Slug, t.Activo).Scan(&t.ID, &t.CreatedAt)
	if err != nil {
		return err
	}

	return nil
}

// ObtenerPorID busca un inquilino específico
func (r *TenantRepository) ObtenerPorID(id int) (*models.Tenant, error) {
	var t models.Tenant
	query := `SELECT id, nombre, ruc, direccion, frase_gestion, logo_url, slug, activo, created_at FROM tenants WHERE id = $1`
	err := r.db.QueryRow(query, id).Scan(&t.ID, &t.Nombre, &t.Ruc, &t.Direccion, &t.FraseGestion, &t.LogoURL, &t.Slug, &t.Activo, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// Actualizar guarda los cambios de un inquilino existente
func (r *TenantRepository) Actualizar(t *models.Tenant) error {
	query := `UPDATE tenants SET nombre = $1, ruc = $2, direccion = $3, frase_gestion = $4, logo_url = $5, slug = $6, activo = $7, updated_at = CURRENT_TIMESTAMP WHERE id = $8`
	_, err := r.db.Exec(query, t.Nombre, t.Ruc, t.Direccion, t.FraseGestion, t.LogoURL, t.Slug, t.Activo, t.ID)
	return err
}
