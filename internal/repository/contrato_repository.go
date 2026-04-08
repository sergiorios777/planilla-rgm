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

// ObtenerRegimenes trae el catálogo para el select
func (r *ContratoRepository) ObtenerRegimenes() ([]models.RegimenLaboral, error) {
	query := `SELECT id, codigo, descripcion FROM regimenes_laborales ORDER BY id ASC`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.RegimenLaboral
	for rows.Next() {
		var reg models.RegimenLaboral
		rows.Scan(&reg.ID, &reg.Codigo, &reg.Descripcion)
		lista = append(lista, reg)
	}
	return lista, nil
}

// ObtenerTodos trae los contratos activos e inactivos de la municipalidad actual
func (r *ContratoRepository) ObtenerTodos(tenantID int) ([]models.Contrato, error) {
	query := `
		SELECT c.id, c.trabajador_id, c.regimen_id, c.cargo, c.sueldo_base, 
		       TO_CHAR(c.fecha_inicio, 'YYYY-MM-DD'), TO_CHAR(c.fecha_fin, 'YYYY-MM-DD'), c.activo,
		       t.numero_documento, t.apellido_paterno || ' ' || t.apellido_materno || ', ' || t.nombres AS nombre_completo,
		       r.descripcion
		FROM contratos c
		INNER JOIN trabajadores t ON c.trabajador_id = t.id
		INNER JOIN regimenes_laborales r ON c.regimen_id = r.id
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

		err := rows.Scan(&c.ID, &c.TrabajadorID, &c.RegimenID, &c.Cargo, &c.SueldoBase,
			&c.FechaInicio, &fFin, &c.Activo,
			&c.TrabajadorDoc, &c.TrabajadorNombre, &c.RegimenDesc)
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
	query := `
		INSERT INTO contratos (tenant_id, trabajador_id, regimen_id, cargo, sueldo_base, fecha_inicio, fecha_fin, activo)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, '')::DATE, $8) RETURNING id
	`
	return r.db.QueryRow(query, c.TenantID, c.TrabajadorID, c.RegimenID, c.Cargo, c.SueldoBase, c.FechaInicio, c.FechaFin, c.Activo).Scan(&c.ID)
}
