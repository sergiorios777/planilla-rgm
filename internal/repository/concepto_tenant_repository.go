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
		INSERT INTO conceptos_tenant (tenant_id, concepto_id, nombre_personalizado, frecuencia_meses, clasificador_id, activo)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id
	`
	return r.db.QueryRow(query, ct.TenantID, ct.ConceptoID, ct.NombrePersonalizado, ct.FrecuenciaMeses, ct.ClasificadorID, ct.Activo).Scan(&ct.ID)
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
