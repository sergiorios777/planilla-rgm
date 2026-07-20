package repository

import (
	"database/sql"
	"planilla-rgm/internal/models"
)

type BaseRegimenRepository struct {
	db *sql.DB
}

func NewBaseRegimenRepository(db *sql.DB) *BaseRegimenRepository {
	return &BaseRegimenRepository{db: db}
}

// ObtenerMontoVariable extrae la suma de los conceptos asignados a un puesto que pertenecen a una variable de cálculo específica activa
func (r *BaseRegimenRepository) ObtenerMontoVariable(tenantID, puestoID, regimenID int, codInternoCalculado, variableCalculo string) (float64, error) {
	query := `
		SELECT COALESCE(SUM(pc.monto), 0.0)
		FROM puesto_conceptos pc
		INNER JOIN base_regimen_tenant brt ON pc.concepto_tenant_id = brt.concepto_tenant_id
		INNER JOIN conceptos_calculados cc ON brt.concepto_calculado_id = cc.id
		WHERE pc.puesto_id = $1 
		  AND pc.activo = true 
		  AND brt.tenant_id = $2 
		  AND brt.regimen_id = $3 
		  AND cc.codigo_interno = $4 
		  AND brt.variable_calculo = $5
		  AND brt.activo = true
	`
	var monto float64
	err := r.db.QueryRow(query, puestoID, tenantID, regimenID, codInternoCalculado, variableCalculo).Scan(&monto)
	return monto, err
}

// VerificarConceptoActivo retorna si una variable específica (ej: ASIGNACION_FAMILIAR) está activa para el puesto
func (r *BaseRegimenRepository) VerificarConceptoActivo(tenantID, puestoID, regimenID int, codInternoCalculado, variableCalculo string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM puesto_conceptos pc
			INNER JOIN base_regimen_tenant brt ON pc.concepto_tenant_id = brt.concepto_tenant_id
			INNER JOIN conceptos_calculados cc ON brt.concepto_calculado_id = cc.id
			WHERE pc.puesto_id = $1 
			  AND pc.activo = true 
			  AND brt.tenant_id = $2 
			  AND brt.regimen_id = $3 
			  AND cc.codigo_interno = $4 
			  AND brt.variable_calculo = $5
			  AND brt.activo = true
		)
	`
	var existe bool
	err := r.db.QueryRow(query, puestoID, tenantID, regimenID, codInternoCalculado, variableCalculo).Scan(&existe)
	return existe, err
}

// ListarConceptosCalculados devuelve todos los conceptos calculados globales
func (r *BaseRegimenRepository) ListarConceptosCalculados() ([]models.ConceptoCalculado, error) {
	query := `
		SELECT id, nombre, tipo, codigo_interno
		FROM conceptos_calculados
		ORDER BY tipo, nombre
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.ConceptoCalculado
	for rows.Next() {
		var c models.ConceptoCalculado
		err := rows.Scan(&c.ID, &c.Nombre, &c.Tipo, &c.CodigoInterno)
		if err == nil {
			lista = append(lista, c)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return lista, nil
}

// CrearConceptoCalculado inserta un nuevo concepto calculado global
func (r *BaseRegimenRepository) CrearConceptoCalculado(c *models.ConceptoCalculado) error {
	query := `
		INSERT INTO conceptos_calculados (nombre, tipo, codigo_interno)
		VALUES ($1, $2, $3) RETURNING id
	`
	return r.db.QueryRow(query, c.Nombre, c.Tipo, c.CodigoInterno).Scan(&c.ID)
}

// EliminarConceptoCalculado elimina un concepto calculado global
func (r *BaseRegimenRepository) EliminarConceptoCalculado(id int) error {
	query := `DELETE FROM conceptos_calculados WHERE id = $1`
	_, err := r.db.Exec(query, id)
	return err
}

// ListarAfectacionesDefault devuelve las afectaciones globales (plantilla) para un concepto calculado
func (r *BaseRegimenRepository) ListarAfectacionesDefault(conceptoCalculadoID int) ([]models.BaseRegimenDefaultDTO, error) {
	query := `
		SELECT brd.id, brd.concepto_calculado_id, brd.regimen_id, rl.codigo, rl.descripcion,
		       brd.concepto_modelo_id, cm.nombre_personalizado, brd.variable_calculo
		FROM base_regimen_default brd
		INNER JOIN regimenes_laborales rl ON brd.regimen_id = rl.id
		INNER JOIN conceptos_modelo cm ON brd.concepto_modelo_id = cm.id
		WHERE brd.concepto_calculado_id = $1
		ORDER BY rl.codigo, brd.variable_calculo, cm.nombre_personalizado
	`
	rows, err := r.db.Query(query, conceptoCalculadoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.BaseRegimenDefaultDTO
	for rows.Next() {
		var d models.BaseRegimenDefaultDTO
		err := rows.Scan(
			&d.ID, &d.ConceptoCalculadoID, &d.RegimenID, &d.RegimenCodigo, &d.RegimenDesc,
			&d.ConceptoModeloID, &d.ConceptoModeloDesc, &d.VariableCalculo,
		)
		if err == nil {
			lista = append(lista, d)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return lista, nil
}

// AgregarAfectacionDefault vincula un concepto de modelo a la plantilla de un concepto calculado
func (r *BaseRegimenRepository) AgregarAfectacionDefault(conceptoCalculadoID, regimenID, conceptoModeloID int, variableCalculo string) error {
	query := `
		INSERT INTO base_regimen_default (concepto_calculado_id, regimen_id, concepto_modelo_id, variable_calculo)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (concepto_calculado_id, regimen_id, concepto_modelo_id, variable_calculo) DO NOTHING
	`
	_, err := r.db.Exec(query, conceptoCalculadoID, regimenID, conceptoModeloID, variableCalculo)
	return err
}

// EliminarAfectacionDefault elimina una afectación de la plantilla global
func (r *BaseRegimenRepository) EliminarAfectacionDefault(id int) error {
	query := `DELETE FROM base_regimen_default WHERE id = $1`
	_, err := r.db.Exec(query, id)
	return err
}

// ObtenerConceptosModeloPorRegimen devuelve la lista de conceptos modelo que están asignados a un régimen
func (r *BaseRegimenRepository) ObtenerConceptosModeloPorRegimen(regimenID int) ([]models.ConceptoModeloDTO, error) {
	query := `
		SELECT cm.id, cm.nombre_personalizado
		FROM conceptos_modelo cm
		INNER JOIN regimen_concepto_modelo rcm ON cm.id = rcm.concepto_modelo_id
		WHERE rcm.regimen_id = $1
		ORDER BY cm.nombre_personalizado
	`
	rows, err := r.db.Query(query, regimenID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.ConceptoModeloDTO
	for rows.Next() {
		var c models.ConceptoModeloDTO
		err := rows.Scan(&c.ID, &c.NombrePersonalizado)
		if err == nil {
			lista = append(lista, c)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return lista, nil
}

