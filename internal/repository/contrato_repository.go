package repository

import (
	"database/sql"
	"planilla-rgm/internal/models"
)

type ContratoRepository struct {
	db *sql.DB
}

func NewContratoRepository(db *sql.DB) *ContratoRepository {
	return &ContratoRepository{db: db}
}

// ObtenerTodos trae los contratos activos e inactivos de la municipalidad actual
// Reemplaza estas dos funciones en contrato_repository.go
func (r *ContratoRepository) ObtenerTodos(tenantID int) ([]models.Contrato, error) {
	query := `
		SELECT c.id, c.trabajador_id, c.puesto_id, 
		       TO_CHAR(c.fecha_inicio, 'YYYY-MM-DD'), TO_CHAR(c.fecha_fin, 'YYYY-MM-DD'), c.activo,
		       t.numero_documento, t.apellido_paterno || ' ' || t.apellido_materno || ', ' || t.nombres,
		       p.nombre, p.sueldo_presupuestado, rl.descripcion
		FROM contratos c
		INNER JOIN trabajadores t ON c.trabajador_id = t.id
		INNER JOIN puestos p ON c.puesto_id = p.id
		INNER JOIN regimenes_laborales rl ON p.regimen_id = rl.id
		WHERE c.tenant_id = $1
		ORDER BY c.activo DESC, t.apellido_paterno ASC
	`
	rows, err := r.db.Query(query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.Contrato
	for rows.Next() {
		var c models.Contrato
		var fFin sql.NullString

		err := rows.Scan(&c.ID, &c.TrabajadorID, &c.PuestoID, &c.FechaInicio, &fFin, &c.Activo,
			&c.TrabajadorDoc, &c.TrabajadorNombre, &c.PuestoNombre, &c.SueldoPresupuestado, &c.RegimenDesc)
		if err == nil {
			if fFin.Valid {
				c.FechaFin = &fFin.String
			}
			lista = append(lista, c)
		}
	}
	return lista, nil
}

func (r *ContratoRepository) Crear(c *models.Contrato) error {
	// Iniciamos una Transacción (Si falla algo, se deshace todo)
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	// 1. Insertamos el contrato
	queryContrato := `
		INSERT INTO contratos (tenant_id, trabajador_id, puesto_id, fecha_inicio, fecha_fin, activo)
		VALUES ($1, $2, $3, $4, NULLIF($5, '')::DATE, $6) RETURNING id
	`
	err = tx.QueryRow(queryContrato, c.TenantID, c.TrabajadorID, c.PuestoID, c.FechaInicio, c.FechaFin, c.Activo).Scan(&c.ID)
	if err != nil {
		tx.Rollback()
		return err
	}

	// 2. Marcamos la Plaza (Puesto) como OCUPADA
	queryPuesto := `UPDATE puestos SET estado = 'OCUPADO' WHERE id = $1 AND tenant_id = $2`
	_, err = tx.Exec(queryPuesto, c.PuestoID, c.TenantID)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Confirmamos los cambios
	return tx.Commit()
}

// TieneContratoActivo verifica si un trabajador ya está ocupando una plaza
func (r *ContratoRepository) TieneContratoActivo(trabajadorID int, tenantID int) (bool, error) {
	var cantidad int
	query := `SELECT COUNT(*) FROM contratos WHERE trabajador_id = $1 AND tenant_id = $2 AND activo = true`

	err := r.db.QueryRow(query, trabajadorID, tenantID).Scan(&cantidad)
	if err != nil {
		return false, err
	}

	// Si la cantidad es mayor a 0, significa que sí tiene un contrato activo
	return cantidad > 0, nil
}
