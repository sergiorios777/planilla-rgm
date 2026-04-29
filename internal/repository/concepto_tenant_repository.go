package repository

import (
	"database/sql"
	"planilla-rgm/internal/models"
)

type ConceptoTenantRepository struct {
	db *sql.DB
}

func NewConceptoTenantRepository(db *sql.DB) *ConceptoTenantRepository {
	return &ConceptoTenantRepository{db: db}
}

// ObtenerMaestros trae el catálogo SUNAT base para el select
func (r *ConceptoTenantRepository) ObtenerMaestros() ([]map[string]interface{}, error) {
	query := `SELECT id, codigo, descripcion, tipo FROM conceptos_maestros ORDER BY tipo, codigo ASC`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []map[string]interface{}
	for rows.Next() {
		var id int
		var cod, desc, tipo string
		rows.Scan(&id, &cod, &desc, &tipo)
		lista = append(lista, map[string]interface{}{
			"ID":          id,
			"Codigo":      cod,
			"Descripcion": desc,
			"Tipo":        tipo,
		})
	}
	return lista, nil
}

// ObtenerTodos trae el catálogo configurado por la municipalidad
func (r *ConceptoTenantRepository) ObtenerTodos(tenantID int) ([]models.ConceptoTenant, error) {
	query := `
		SELECT ct.id, ct.concepto_id, ct.nombre_personalizado, ct.frecuencia_meses, ct.clasificador_id, ct.activo,
		       cm.codigo, cm.tipo, 
			   mef.codigo AS clasificador_codigo
		FROM conceptos_tenant ct
		INNER JOIN conceptos_maestros cm ON ct.concepto_id = cm.id
		LEFT JOIN clasificadores_mef mef ON ct.clasificador_id = mef.id
		WHERE ct.tenant_id = $1
		ORDER BY cm.tipo ASC, ct.nombre_personalizado ASC
	`
	rows, err := r.db.Query(query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.ConceptoTenant
	for rows.Next() {
		var ct models.ConceptoTenant
		var clasifID sql.NullInt64
		var clasifCod sql.NullString

		err := rows.Scan(&ct.ID, &ct.ConceptoID, &ct.NombrePersonalizado, &ct.FrecuenciaMeses, &clasifID, &ct.Activo,
			&ct.ConceptoCodigo, &ct.ConceptoTipo, &clasifCod)
		if err == nil {
			if clasifID.Valid {
				id := int(clasifID.Int64)
				ct.ClasificadorID = &id
				ct.ClasificadorCodigo = clasifCod.String
			}
			lista = append(lista, ct)
		}
	}
	return lista, nil
}

// Crear inserta la configuración local
func (r *ConceptoTenantRepository) Crear(ct *models.ConceptoTenant) error {
	query := `
		INSERT INTO conceptos_tenant (tenant_id, concepto_id, nombre_personalizado, frecuencia_meses, clasificador_id, activo, es_extraordinario)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id
	`
	return r.db.QueryRow(query, ct.TenantID, ct.ConceptoID, ct.NombrePersonalizado, ct.FrecuenciaMeses, ct.ClasificadorID, ct.Activo, ct.EsExtraordinario).Scan(&ct.ID)
}

// Actualizar actualiza la configuración local
func (r *ConceptoTenantRepository) Actualizar(ct *models.ConceptoTenant) error {
	query := `
		UPDATE conceptos_tenant 
		SET concepto_id = $2, nombre_personalizado = $3, frecuencia_meses = $4, clasificador_id = $5, activo = $6, es_extraordinario = $7
		WHERE id = $1 AND tenant_id = $8
	`
	_, err := r.db.Exec(query, ct.ID, ct.ConceptoID, ct.NombrePersonalizado, ct.FrecuenciaMeses, ct.ClasificadorID, ct.Activo, ct.EsExtraordinario, ct.TenantID)
	return err
}

// 1. NUEVA FUNCIÓN: Para llenar el menú desplegable en el formulario
func (r *ConceptoTenantRepository) ObtenerClasificadores() ([]models.ClasificadorMEF, error) {
	// Filtramos solo los que son nivel detalle (los que no tienen hijos) o por transacción
	// Ajusta la consulta según tu necesidad, aquí traemos todos los activos
	query := `
	        SELECT id, codigo_limpio, descripcion 
	        FROM clasificadores_mef 
	        WHERE activo = true 
			    AND nivel = 6 
			    AND (
				    codigo LIKE '2.1.%' OR 
				    codigo LIKE '2.3.%' OR 
				    codigo LIKE '2.6.%'
			    )
	        ORDER BY codigo ASC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.ClasificadorMEF
	for rows.Next() {
		var c models.ClasificadorMEF
		rows.Scan(&c.ID, &c.CodigoLimpio, &c.Descripcion)
		lista = append(lista, c)
	}
	return lista, nil
}

// ObtenerPorID trae un concepto específico para rellenar el formulario de edición
func (r *ConceptoTenantRepository) ObtenerPorID(id int, tenantID int) (models.ConceptoTenant, error) {
	var c models.ConceptoTenant
	query := `
		SELECT ct.id, ct.concepto_id, ct.nombre_personalizado, ct.frecuencia_meses, ct.clasificador_id, ct.activo,
		       cm.codigo, cm.tipo, 
			   mef.codigo AS clasificador_codigo
		FROM conceptos_tenant ct
		INNER JOIN conceptos_maestros cm ON ct.concepto_id = cm.id
		LEFT JOIN clasificadores_mef mef ON ct.clasificador_id = mef.id
		WHERE ct.id = $1 AND ct.tenant_id = $2
	`
	err := r.db.QueryRow(query, id, tenantID).Scan(
		&c.ID, &c.ConceptoID, &c.NombrePersonalizado, &c.FrecuenciaMeses, &c.ClasificadorID, &c.Activo,
		&c.ConceptoCodigo, &c.ConceptoTipo, &c.ClasificadorCodigo)
	return c, err
}

// ActualizarPersonalizado guarda el nuevo nombre personalizado y el estado
func (r *ConceptoTenantRepository) ActualizarPersonalizado(id int, tenantID int, nombre string, activo bool) error {
	query := `
		UPDATE conceptos_tenant 
		SET nombre_personalizado = $1, activo = $2 
		WHERE id = $3 AND tenant_id = $4
	`
	_, err := r.db.Exec(query, nombre, activo, id, tenantID)
	return err
}

func (r *ConceptoTenantRepository) ActualizarCompleto(id int, tenantID int, conceptoID int, clasificadorID *int, nombre string, frecuencia string, activo bool) error {
	query := `
		UPDATE conceptos_tenant 
		SET concepto_id = $1, clasificador_id = $2, nombre_personalizado = $3, 
		    frecuencia_meses = $4, activo = $5, updated_at = CURRENT_TIMESTAMP
		WHERE id = $6 AND tenant_id = $7
	`
	_, err := r.db.Exec(query, conceptoID, clasificadorID, nombre, frecuencia, activo, id, tenantID)
	return err
}
