package repository

import (
	"database/sql"
	"planilla-rgm/internal/models"
)

type PuestoConceptoRepository struct {
	db *sql.DB
}

func NewPuestoConceptoRepository(db *sql.DB) *PuestoConceptoRepository {
	return &PuestoConceptoRepository{db: db}
}

// ObtenerAsignados trae los conceptos que ya forman parte de la Plaza
func (r *PuestoConceptoRepository) ObtenerAsignados(puestoID int, tenantID int) ([]models.PuestoConcepto, error) {
	query := `
		SELECT pc.id, pc.puesto_id, pc.concepto_tenant_id, pc.monto, pc.activo,
		       ct.nombre_personalizado, mef.codigo AS clasificador, cm.tipo
		FROM puesto_conceptos pc
		INNER JOIN conceptos_tenant ct ON pc.concepto_tenant_id = ct.id
		INNER JOIN conceptos_maestros cm ON ct.concepto_id = cm.id
		LEFT JOIN clasificadores_mef mef ON ct.clasificador_id = mef.id
		WHERE pc.puesto_id = $1 AND ct.tenant_id = $2
		ORDER BY cm.tipo ASC, ct.nombre_personalizado ASC
	`
	rows, err := r.db.Query(query, puestoID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.PuestoConcepto
	for rows.Next() {
		var pc models.PuestoConcepto
		var monto sql.NullFloat64
		var clasif sql.NullString

		err := rows.Scan(&pc.ID, &pc.PuestoID, &pc.ConceptoTenantID, &monto, &pc.Activo,
			&pc.NombrePersonalizado, &clasif, &pc.ConceptoTipo)
		if err == nil {
			if monto.Valid {
				pc.Monto = &monto.Float64
			}
			if clasif.Valid {
				pc.Clasificador = clasif.String
			}
			lista = append(lista, pc)
		}
	}
	return lista, nil
}

// ObtenerDisponibles trae los conceptos del tenant que AÚN NO están en esta Plaza
func (r *PuestoConceptoRepository) ObtenerDisponibles(puestoID int, tenantID int) ([]models.ConceptoTenant, error) {
	query := `
		SELECT ct.id, ct.nombre_personalizado, cm.tipo
		FROM conceptos_tenant ct
		INNER JOIN conceptos_maestros cm ON ct.concepto_id = cm.id
		WHERE ct.tenant_id = $1 
		  AND ct.activo = true
		  AND ct.id NOT IN (
		      SELECT concepto_tenant_id FROM puesto_conceptos WHERE puesto_id = $2
		  )
		ORDER BY cm.tipo ASC, ct.nombre_personalizado ASC
	`
	rows, err := r.db.Query(query, tenantID, puestoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.ConceptoTenant
	for rows.Next() {
		var ct models.ConceptoTenant
		rows.Scan(&ct.ID, &ct.NombrePersonalizado, &ct.ConceptoTipo)
		lista = append(lista, ct)
	}
	return lista, nil
}

func (r *PuestoConceptoRepository) Crear(pc *models.PuestoConcepto) error {
	query := `
		INSERT INTO puesto_conceptos (puesto_id, concepto_tenant_id, monto, activo)
		VALUES ($1, $2, $3, $4) RETURNING id
	`
	return r.db.QueryRow(query, pc.PuestoID, pc.ConceptoTenantID, pc.Monto, pc.Activo).Scan(&pc.ID)
}

func (r *PuestoConceptoRepository) Eliminar(id int) error {
	_, err := r.db.Exec(`DELETE FROM puesto_conceptos WHERE id = $1`, id)
	return err
}
