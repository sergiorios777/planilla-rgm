package repository

import (
	"database/sql"
	"fmt"
	"planilla-rgm/internal/models"
)

type MetaRepository struct {
	db *sql.DB
}

func NewMetaRepository(db *sql.DB) *MetaRepository {
	return &MetaRepository{db: db}
}

func (r *MetaRepository) ObtenerTodos(tenantID int) ([]models.MetaPresupuestal, error) {
	query := `SELECT id, tenant_id, anio, codigo, descripcion, activo FROM metas_presupuestales WHERE tenant_id = $1 ORDER BY anio DESC, codigo ASC`
	rows, err := r.db.Query(query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.MetaPresupuestal
	for rows.Next() {
		var m models.MetaPresupuestal
		err := rows.Scan(&m.ID, &m.TenantID, &m.Anio, &m.Codigo, &m.Descripcion, &m.Activo)
		if err == nil {
			lista = append(lista, m)
		}
	}
	return lista, nil
}

// ObtenerTodosPaginación
func (r *MetaRepository) ObtenerTodosPaginacion(tenantID int, busqueda string, estado string, limite int, offset int) ([]models.MetaPresupuestal, int, error) {
	// 1. Definimos la consulta base con un WHERE inicial para el inquilino
	whereClause := `WHERE tenant_id = $1`

	// 2. Inicializamos nuestros argumentos seguros con el tenantID
	args := []interface{}{tenantID}
	contadorArgs := 2 // El siguiente parámetro será $2

	// 3. Lógica de búsqueda dinámica
	if busqueda != "" {
		// Buscamos en código o descripción
		whereClause += fmt.Sprintf(` AND (
			codigo ILIKE $%d OR 
			descripcion ILIKE $%d
		)`, contadorArgs, contadorArgs)

		// Añadimos el valor de búsqueda al arreglo (con los comodines % para coincidencia parcial)
		args = append(args, "%"+busqueda+"%")
		contadorArgs++
	}

	// 4. Lógica del filtro por Estado (activo = true o false)
	if estado != "" {
		whereClause += fmt.Sprintf(` AND activo = $%d`, contadorArgs)
		if estado == "Activo" {
			args = append(args, true)
		} else {
			args = append(args, false)
		}
		contadorArgs++
	}

	var totalRegistros int
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) FROM metas_presupuestales %s`, whereClause)
	err := r.db.QueryRow(countQuery, args...).Scan(&totalRegistros)
	if err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, anio, codigo, descripcion, activo
		FROM metas_presupuestales
		%s
		ORDER BY anio DESC, codigo ASC
		LIMIT $%d OFFSET $%d
	`, whereClause, contadorArgs, contadorArgs+1)

	args = append(args, limite, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var lista []models.MetaPresupuestal
	for rows.Next() {
		var m models.MetaPresupuestal
		err := rows.Scan(&m.ID, &m.TenantID, &m.Anio, &m.Codigo, &m.Descripcion, &m.Activo)
		if err != nil {
			return nil, 0, err
		}
		lista = append(lista, m)
	}
	return lista, totalRegistros, nil
}

func (r *MetaRepository) Crear(m *models.MetaPresupuestal) error {
	query := `INSERT INTO metas_presupuestales (tenant_id, anio, codigo, descripcion, activo) VALUES ($1, $2, $3, $4, $5) RETURNING id`
	return r.db.QueryRow(query, m.TenantID, m.Anio, m.Codigo, m.Descripcion, m.Activo).Scan(&m.ID)
}

// ObtenerPorID trae los datos de una meta específica para el formulario de edición
func (r *MetaRepository) ObtenerPorID(id int, tenantID int) (models.MetaPresupuestal, error) {
	var m models.MetaPresupuestal
	query := `SELECT id, tenant_id, anio, codigo, descripcion, activo FROM metas_presupuestales WHERE id = $1 AND tenant_id = $2`
	err := r.db.QueryRow(query, id, tenantID).Scan(&m.ID, &m.TenantID, &m.Anio, &m.Codigo, &m.Descripcion, &m.Activo)
	return m, err
}

// Actualizar guarda los cambios de la meta
func (r *MetaRepository) Actualizar(m *models.MetaPresupuestal) error {
	// Omitimos actualizar el Año por seguridad contable, solo código, descripción y estado
	query := `UPDATE metas_presupuestales SET codigo = $1, descripcion = $2, activo = $3 WHERE id = $4 AND tenant_id = $5`
	_, err := r.db.Exec(query, m.Codigo, m.Descripcion, m.Activo, m.ID, m.TenantID)
	return err
}
