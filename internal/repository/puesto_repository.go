package repository

import (
	"database/sql"
	"planilla-rgm/internal/models"
)

type PuestoRepository struct {
	db *sql.DB
}

func NewPuestoRepository(db *sql.DB) *PuestoRepository {
	return &PuestoRepository{db: db}
}

// ObtenerVacantes lista solo los puestos que no están ocupados
func (r *PuestoRepository) ObtenerVacantes(tenantID int) ([]models.Puesto, error) {
	query := `
		SELECT p.id, p.nombre, p.sueldo_presupuestado, rl.descripcion
		FROM puestos p
		INNER JOIN regimenes_laborales rl ON p.regimen_id = rl.id
		WHERE p.tenant_id = $1 AND p.estado = 'VACANTE' AND p.activo = true
		ORDER BY p.nombre ASC
	`
	rows, err := r.db.Query(query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.Puesto
	for rows.Next() {
		var p models.Puesto
		rows.Scan(&p.ID, &p.Nombre, &p.SueldoPresupuestado, &p.RegimenDesc)
		lista = append(lista, p)
	}
	return lista, nil
}

// (Añade estas funciones debajo de la que ya tienes "ObtenerVacantes")

func (r *PuestoRepository) ObtenerRegimenes() ([]models.RegimenLaboral, error) {
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

func (r *PuestoRepository) ObtenerTodos(tenantID int) ([]models.Puesto, error) {
	query := `
		SELECT p.id, p.nombre, p.sueldo_presupuestado, p.estado, p.activo,
		       m.codigo, fr.rubro, rl.descripcion
		FROM puestos p
		INNER JOIN metas_presupuestales m ON p.meta_id = m.id
		INNER JOIN fuentes_rubros fr ON p.fuente_rubro_id = fr.id
		INNER JOIN regimenes_laborales rl ON p.regimen_id = rl.id
		WHERE p.tenant_id = $1
		ORDER BY p.id DESC
	`
	rows, err := r.db.Query(query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.Puesto
	for rows.Next() {
		var p models.Puesto
		err := rows.Scan(&p.ID, &p.Nombre, &p.SueldoPresupuestado, &p.Estado, &p.Activo,
			&p.MetaCodigo, &p.FuenteRubroDesc, &p.RegimenDesc)
		if err == nil {
			lista = append(lista, p)
		}
	}
	return lista, nil
}

func (r *PuestoRepository) Crear(p *models.Puesto) error {
	query := `
		INSERT INTO puestos (tenant_id, meta_id, fuente_rubro_id, regimen_id, nombre, sueldo_presupuestado, estado, activo)
		VALUES ($1, $2, $3, $4, $5, $6, 'VACANTE', $7) RETURNING id
	`
	return r.db.QueryRow(query, p.TenantID, p.MetaID, p.FuenteRubroID, p.RegimenID, p.Nombre, p.SueldoPresupuestado, p.Activo).Scan(&p.ID)
}
