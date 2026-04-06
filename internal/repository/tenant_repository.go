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
	query := `SELECT id, nombre, ruc, activo, created_at FROM tenants ORDER BY id DESC`

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
		if err := rows.Scan(&t.ID, &t.Nombre, &t.Ruc, &t.Activo, &t.CreatedAt); err != nil {
			return nil, err
		}
		lista = append(lista, t)
	}

	return lista, nil
}

// Crear inserta un nuevo inquilino en la base de datos
func (r *TenantRepository) Crear(t *models.Tenant) error {
	// Escribimos la consulta SQL. RETURNING nos devuelve el ID y la fecha generados.
	query := `INSERT INTO tenants (nombre, ruc, activo) 
	          VALUES ($1, $2, $3) 
	          RETURNING id, created_at`

	// Ejecutamos la consulta y guardamos el ID y la fecha devueltos en nuestra estructura
	err := r.db.QueryRow(query, t.Nombre, t.Ruc, t.Activo).Scan(&t.ID, &t.CreatedAt)
	if err != nil {
		return err
	}

	return nil
}
