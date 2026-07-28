package repository

import (
	"database/sql"
	"fmt"
	"log"
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return lista, nil
}

// ObtenerTodos trae el catálogo configurado por la municipalidad
func (r *ConceptoTenantRepository) ObtenerTodos(tenantID int) ([]models.ConceptoTenant, error) {
	query := `
		SELECT ct.id, ct.concepto_id, ct.modelo_id, ct.nombre_personalizado, ct.frecuencia_meses, ct.clasificador_id, ct.activo,
		       ct.es_extraordinario, ct.es_pensionable, ct.es_remunerativa, ct.es_base_cts, ct.es_base_beneficios_sociales, ct.es_ocasional, ct.es_afecto_cargas_sociales,
		       cm.codigo, cm.tipo, 
			   mef.codigo AS clasificador_codigo,
			   COALESCE(STRING_AGG(rl.codigo, ', '), 'Sin régimen') AS regimenes_codigos
		FROM conceptos_tenant ct
		INNER JOIN conceptos_maestros cm ON ct.concepto_id = cm.id
		LEFT JOIN clasificadores_mef mef ON ct.clasificador_id = mef.id
		LEFT JOIN regimen_concepto_tenant rct ON ct.id = rct.concepto_tenant_id AND ct.tenant_id = rct.tenant_id
		LEFT JOIN regimenes_laborales rl ON rct.regimen_id = rl.id
		WHERE ct.tenant_id = $1
		GROUP BY ct.id, cm.codigo, cm.tipo, mef.codigo
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
		var modeloID sql.NullInt64
		var regimenesCodigos sql.NullString

		err := rows.Scan(&ct.ID, &ct.ConceptoID, &modeloID, &ct.NombrePersonalizado, &ct.FrecuenciaMeses, &clasifID, &ct.Activo,
			&ct.EsExtraordinario, &ct.EsPensionable, &ct.EsRemunerativa, &ct.EsBaseCts, &ct.EsBaseBeneficiosSociales, &ct.EsOcasional, &ct.EsAfectoCargasSociales,
			&ct.ConceptoCodigo, &ct.ConceptoTipo, &clasifCod, &regimenesCodigos)
		if err == nil {
			if clasifID.Valid {
				id := int(clasifID.Int64)
				ct.ClasificadorID = &id
				ct.ClasificadorCodigo = clasifCod.String
			}
			if modeloID.Valid {
				mID := int(modeloID.Int64)
				ct.ModeloID = &mID
			}
			if regimenesCodigos.Valid {
				ct.RegimenesCodigos = regimenesCodigos.String
			}
			lista = append(lista, ct)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return lista, nil
}

// ObtenerTodosPaginacion trae el catálogo configurado por la municipalidad con paginación
func (r *ConceptoTenantRepository) ObtenerTodosPaginacion(tenantID int, busqueda string, regimenID int, limite int, offset int) ([]models.ConceptoTenant, int, error) {
	whereClause := "WHERE ct.tenant_id = $1"
	params := []interface{}{tenantID}
	paramIndex := 2

	if busqueda != "" {
		whereClause += fmt.Sprintf(" AND (ct.nombre_personalizado ILIKE $%d OR cm.codigo ILIKE $%d OR mef.codigo ILIKE $%d)", paramIndex, paramIndex+1, paramIndex+2)
		params = append(params, "%"+busqueda+"%", "%"+busqueda+"%", "%"+busqueda+"%")
		paramIndex += 3
	}

	if regimenID > 0 {
		whereClause += fmt.Sprintf(" AND EXISTS (SELECT 1 FROM regimen_concepto_tenant rct_f WHERE rct_f.concepto_tenant_id = ct.id AND rct_f.tenant_id = ct.tenant_id AND rct_f.regimen_id = $%d)", paramIndex)
		params = append(params, regimenID)
		paramIndex++
	}

	var totalRegistros int
	countQuery := fmt.Sprintf(
		`
			SELECT COUNT(DISTINCT ct.id) FROM conceptos_tenant ct 
			INNER JOIN conceptos_maestros cm ON ct.concepto_id = cm.id
			LEFT JOIN clasificadores_mef mef ON ct.clasificador_id = mef.id
			LEFT JOIN regimen_concepto_tenant rct ON ct.id = rct.concepto_tenant_id AND ct.tenant_id = rct.tenant_id
			LEFT JOIN regimenes_laborales rl ON rct.regimen_id = rl.id
			%s
		`,
		whereClause,
	)

	err := r.db.QueryRow(countQuery, params...).Scan(&totalRegistros)
	if err != nil {
		log.Println("Error al obtener el total de registros (en concepto_tenant_repository):", err)
		return nil, 0, err
	}

	query := fmt.Sprintf(`
		SELECT ct.id, ct.concepto_id, ct.modelo_id, ct.nombre_personalizado, ct.frecuencia_meses, ct.clasificador_id, ct.activo,
		       ct.es_extraordinario, ct.es_pensionable, ct.es_remunerativa, ct.es_base_cts, ct.es_base_beneficios_sociales, ct.es_ocasional, ct.es_afecto_cargas_sociales,
		       cm.codigo, cm.tipo, 
			   mef.codigo AS clasificador_codigo,
			   COALESCE(STRING_AGG(rl.codigo, ', '), 'Sin régimen') AS regimenes_codigos
		FROM conceptos_tenant ct
		INNER JOIN conceptos_maestros cm ON ct.concepto_id = cm.id
		LEFT JOIN clasificadores_mef mef ON ct.clasificador_id = mef.id
		LEFT JOIN regimen_concepto_tenant rct ON ct.id = rct.concepto_tenant_id AND ct.tenant_id = rct.tenant_id
		LEFT JOIN regimenes_laborales rl ON rct.regimen_id = rl.id
		%s
		GROUP BY ct.id, cm.codigo, cm.tipo, mef.codigo
		ORDER BY cm.tipo ASC, ct.nombre_personalizado ASC
		LIMIT $%d OFFSET $%d
	`,
		whereClause,
		paramIndex,
		paramIndex+1,
	)

	params = append(params, limite, offset)

	rows, err := r.db.Query(query, params...)
	if err != nil {
		log.Println("Error al paginar conceptos locales:", err)
		return nil, 0, err
	}
	defer rows.Close()

	var lista []models.ConceptoTenant
	for rows.Next() {
		var ct models.ConceptoTenant
		var clasifID sql.NullInt64
		var clasifCod sql.NullString
		var modeloID sql.NullInt64
		var regimenesCodigos sql.NullString

		err := rows.Scan(&ct.ID, &ct.ConceptoID, &modeloID, &ct.NombrePersonalizado, &ct.FrecuenciaMeses, &clasifID, &ct.Activo,
			&ct.EsExtraordinario, &ct.EsPensionable, &ct.EsRemunerativa, &ct.EsBaseCts, &ct.EsBaseBeneficiosSociales, &ct.EsOcasional, &ct.EsAfectoCargasSociales,
			&ct.ConceptoCodigo, &ct.ConceptoTipo, &clasifCod, &regimenesCodigos)
		if err == nil {
			if clasifID.Valid {
				id := int(clasifID.Int64)
				ct.ClasificadorID = &id
				ct.ClasificadorCodigo = clasifCod.String
			}
			if modeloID.Valid {
				mID := int(modeloID.Int64)
				ct.ModeloID = &mID
			}
			if regimenesCodigos.Valid {
				ct.RegimenesCodigos = regimenesCodigos.String
			}
			lista = append(lista, ct)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return lista, totalRegistros, nil
}

// Crear inserta la configuración local y sus relaciones con los regímenes laborales en una transacción
func (r *ConceptoTenantRepository) Crear(ct *models.ConceptoTenant) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	query := `
		INSERT INTO conceptos_tenant (tenant_id, concepto_id, modelo_id, nombre_personalizado, frecuencia_meses, clasificador_id, activo, es_extraordinario, es_pensionable, es_remunerativa, es_base_cts, es_base_beneficios_sociales, es_ocasional, es_afecto_cargas_sociales)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14) RETURNING id
	`
	err = tx.QueryRow(query, ct.TenantID, ct.ConceptoID, ct.ModeloID, ct.NombrePersonalizado, ct.FrecuenciaMeses, ct.ClasificadorID, ct.Activo, ct.EsExtraordinario, ct.EsPensionable, ct.EsRemunerativa, ct.EsBaseCts, ct.EsBaseBeneficiosSociales, ct.EsOcasional, ct.EsAfectoCargasSociales).Scan(&ct.ID)
	if err != nil {
		tx.Rollback()
		return err
	}

	queryRegimen := `
		INSERT INTO regimen_concepto_tenant (tenant_id, regimen_id, concepto_tenant_id)
		VALUES ($1, $2, $3)
	`
	for _, regimenID := range ct.RegimenesIDs {
		_, err = tx.Exec(queryRegimen, ct.TenantID, regimenID, ct.ID)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

// Actualizar actualiza la configuración local y sus relaciones de regímenes en una transacción (Limpiar e Insertar)
func (r *ConceptoTenantRepository) Actualizar(ct *models.ConceptoTenant) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	query := `
		UPDATE conceptos_tenant 
		SET concepto_id = $2, nombre_personalizado = $3, frecuencia_meses = $4, clasificador_id = $5, activo = $6, es_extraordinario = $7,
		    es_pensionable = $8, es_remunerativa = $9, es_base_cts = $10, es_base_beneficios_sociales = $11, es_ocasional = $12, es_afecto_cargas_sociales = $13, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND tenant_id = $14
	`
	_, err = tx.Exec(query, ct.ID, ct.ConceptoID, ct.NombrePersonalizado, ct.FrecuenciaMeses, ct.ClasificadorID, ct.Activo, ct.EsExtraordinario,
		ct.EsPensionable, ct.EsRemunerativa, ct.EsBaseCts, ct.EsBaseBeneficiosSociales, ct.EsOcasional, ct.EsAfectoCargasSociales, ct.TenantID)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Limpiar relaciones previas
	queryDelete := `DELETE FROM regimen_concepto_tenant WHERE concepto_tenant_id = $1 AND tenant_id = $2`
	_, err = tx.Exec(queryDelete, ct.ID, ct.TenantID)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Insertar nuevas relaciones
	queryInsert := `
		INSERT INTO regimen_concepto_tenant (tenant_id, regimen_id, concepto_tenant_id)
		VALUES ($1, $2, $3)
	`
	for _, regimenID := range ct.RegimenesIDs {
		_, err = tx.Exec(queryInsert, ct.TenantID, regimenID, ct.ID)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return lista, nil
}

// ObtenerPorID trae un concepto específico para rellenar el formulario de edición
func (r *ConceptoTenantRepository) ObtenerPorID(id int, tenantID int) (models.ConceptoTenant, error) {
	var c models.ConceptoTenant
	var modeloID sql.NullInt64
	query := `
		SELECT ct.id, ct.concepto_id, ct.modelo_id, ct.nombre_personalizado, ct.frecuencia_meses, ct.clasificador_id, ct.activo,
		       ct.es_extraordinario, ct.es_pensionable, ct.es_remunerativa, ct.es_base_cts, ct.es_base_beneficios_sociales, ct.es_ocasional, ct.es_afecto_cargas_sociales,
		       cm.codigo, cm.tipo, 
			   mef.codigo AS clasificador_codigo
		FROM conceptos_tenant ct
		INNER JOIN conceptos_maestros cm ON ct.concepto_id = cm.id
		LEFT JOIN clasificadores_mef mef ON ct.clasificador_id = mef.id
		WHERE ct.id = $1 AND ct.tenant_id = $2
	`
	err := r.db.QueryRow(query, id, tenantID).Scan(
		&c.ID, &c.ConceptoID, &modeloID, &c.NombrePersonalizado, &c.FrecuenciaMeses, &c.ClasificadorID, &c.Activo,
		&c.EsExtraordinario, &c.EsPensionable, &c.EsRemunerativa, &c.EsBaseCts, &c.EsBaseBeneficiosSociales, &c.EsOcasional, &c.EsAfectoCargasSociales,
		&c.ConceptoCodigo, &c.ConceptoTipo, &c.ClasificadorCodigo)
	if err == nil && modeloID.Valid {
		mID := int(modeloID.Int64)
		c.ModeloID = &mID
	}
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

// ClonarDesdeModelo copia todos los conceptos base a un nuevo tenant.
// También sirve como función "Restaurar", ya que ignora los conceptos que ya existen.
func (r *ConceptoTenantRepository) ClonarDesdeModelo(tenantID int) error {
	query := `
		INSERT INTO conceptos_tenant 
		(tenant_id, concepto_id, modelo_id, nombre_personalizado, frecuencia_meses, clasificador_id, es_extraordinario, requiere_monto, activo,
		 es_pensionable, es_remunerativa, es_base_cts, es_base_beneficios_sociales, es_ocasional, es_afecto_cargas_sociales)
		SELECT 
			$1, concepto_id, id, nombre_personalizado, frecuencia_meses, clasificador_id, es_extraordinario, requiere_monto, true,
			es_pensionable, es_remunerativa, es_base_cts, es_base_beneficios_sociales, es_ocasional, es_afecto_cargas_sociales
		FROM conceptos_modelo
		ON CONFLICT (tenant_id, modelo_id) DO NOTHING;
	`
	_, err := r.db.Exec(query, tenantID)
	return err
}

// ClonarRelacionesRegimen copia las relaciones régimen <-> concepto del catálogo modelo al tenant local
func (r *ConceptoTenantRepository) ClonarRelacionesRegimen(tenantID int) error {
	query := `
		INSERT INTO regimen_concepto_tenant (tenant_id, regimen_id, concepto_tenant_id)
		SELECT 
			ct.tenant_id, 
			rcm.regimen_id, 
			ct.id 
		FROM conceptos_tenant ct
		INNER JOIN regimen_concepto_modelo rcm ON ct.modelo_id = rcm.concepto_modelo_id
		WHERE ct.tenant_id = $1
		ON CONFLICT (tenant_id, regimen_id, concepto_tenant_id) DO NOTHING;
	`
	_, err := r.db.Exec(query, tenantID)
	return err
}

// ObtenerRegimenesPorConcepto obtiene los IDs de los regímenes asociados a un concepto del tenant
func (r *ConceptoTenantRepository) ObtenerRegimenesPorConcepto(id int, tenantID int) ([]int, error) {
	query := `SELECT regimen_id FROM regimen_concepto_tenant WHERE concepto_tenant_id = $1 AND tenant_id = $2`
	rows, err := r.db.Query(query, id, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var rID int
		if err := rows.Scan(&rID); err == nil {
			ids = append(ids, rID)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}

// SincronizarDesdeModeloAvanzado sincroniza conceptos del modelo al tenant de forma atómica y bajo filtros específicos
func (r *ConceptoTenantRepository) SincronizarDesdeModeloAvanzado(tenantID int, modo string, fechaInicio string, fechaFin string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("error al iniciar transaccion: %w", err)
	}
	defer tx.Rollback()

	// 1. Construir query dinámico para insertar conceptos
	queryInsert := `
		INSERT INTO conceptos_tenant 
		(tenant_id, concepto_id, modelo_id, nombre_personalizado, frecuencia_meses, clasificador_id, es_extraordinario, requiere_monto, activo,
		 es_pensionable, es_remunerativa, es_base_cts, es_base_beneficios_sociales, es_ocasional, es_afecto_cargas_sociales)
		SELECT 
			$1, concepto_id, id, nombre_personalizado, frecuencia_meses, clasificador_id, es_extraordinario, requiere_monto, true,
			es_pensionable, es_remunerativa, es_base_cts, es_base_beneficios_sociales, es_ocasional, es_afecto_cargas_sociales
		FROM conceptos_modelo
		WHERE 1=1
	`
	var args []interface{}
	args = append(args, tenantID)

	switch modo {
	case "FECHAS":
		queryInsert += " AND created_at::date BETWEEN $2::date AND $3::date"
		args = append(args, fechaInicio, fechaFin)
	case "EXTRAORDINARIOS":
		queryInsert += " AND es_extraordinario = true"
	}

	queryInsert += " ON CONFLICT (tenant_id, modelo_id) DO NOTHING"

	_, err = tx.Exec(queryInsert, args...)
	if err != nil {
		return fmt.Errorf("error al insertar conceptos tenant: %w", err)
	}

	// 2. Sincronizar relaciones de regímenes
	queryRegimenes := `
		INSERT INTO regimen_concepto_tenant (tenant_id, regimen_id, concepto_tenant_id)
		SELECT 
			ct.tenant_id, 
			rcm.regimen_id, 
			ct.id 
		FROM conceptos_tenant ct
		INNER JOIN regimen_concepto_modelo rcm ON ct.modelo_id = rcm.concepto_modelo_id
		WHERE ct.tenant_id = $1
		ON CONFLICT (tenant_id, regimen_id, concepto_tenant_id) DO NOTHING
	`
	_, err = tx.Exec(queryRegimenes, tenantID)
	if err != nil {
		return fmt.Errorf("error al clonar relaciones de regimenes: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error al confirmar transaccion: %w", err)
	}

	return nil
}

// ObtenerModelosDisponibles retorna los conceptos modelo que aún no existen en conceptos_tenant para el tenantID especificado
func (r *ConceptoTenantRepository) ObtenerModelosDisponibles(tenantID int) ([]models.ConceptoModelo, error) {
	query := `
		SELECT 
			cm.id, cm.concepto_id, cm.nombre_personalizado, cm.frecuencia_meses, cm.clasificador_id,
			cm.es_extraordinario, cm.requiere_monto, cm.es_pensionable, cm.es_remunerativa, 
			cm.es_base_cts, cm.es_base_beneficios_sociales, cm.es_ocasional, cm.es_afecto_cargas_sociales,
			cma.codigo AS concepto_codigo, cma.tipo AS concepto_tipo, cma.descripcion AS concepto_descripcion,
			mef.codigo AS clasificador_codigo,
			COALESCE(STRING_AGG(rl.codigo, ', '), 'Sin régimen') AS regimenes_nombres
		FROM conceptos_modelo cm
		INNER JOIN conceptos_maestros cma ON cm.concepto_id = cma.id
		LEFT JOIN clasificadores_mef mef ON cm.clasificador_id = mef.id
		LEFT JOIN regimen_concepto_modelo rcm ON cm.id = rcm.concepto_modelo_id
		LEFT JOIN regimenes_laborales rl ON rcm.regimen_id = rl.id
		WHERE cm.id NOT IN (
			SELECT ct.modelo_id 
			FROM conceptos_tenant ct 
			WHERE ct.tenant_id = $1 AND ct.modelo_id IS NOT NULL
		)
		GROUP BY cm.id, cm.concepto_id, cm.nombre_personalizado, cm.frecuencia_meses, cm.clasificador_id,
		         cm.es_extraordinario, cm.requiere_monto, cm.es_pensionable, cm.es_remunerativa, 
		         cm.es_base_cts, cm.es_base_beneficios_sociales, cm.es_ocasional, cm.es_afecto_cargas_sociales,
		         cma.codigo, cma.tipo, cma.descripcion, mef.codigo
		ORDER BY cma.tipo ASC, cm.nombre_personalizado ASC
	`
	rows, err := r.db.Query(query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.ConceptoModelo
	for rows.Next() {
		var cm models.ConceptoModelo
		var clasifID sql.NullInt64
		var clasifCod sql.NullString
		var regimenesNombres sql.NullString

		err := rows.Scan(
			&cm.ID, &cm.ConceptoID, &cm.NombrePersonalizado, &cm.FrecuenciaMeses, &clasifID,
			&cm.EsExtraordinario, &cm.RequiereMonto, &cm.EsPensionable, &cm.EsRemunerativa,
			&cm.EsBaseCts, &cm.EsBaseBeneficiosSociales, &cm.EsOcasional, &cm.EsAfectoCargasSociales,
			&cm.ConceptoCodigo, &cm.ConceptoTipo, &cm.ConceptoDescripcion,
			&clasifCod, &regimenesNombres,
		)
		if err != nil {
			return nil, err
		}
		if clasifID.Valid {
			id := int(clasifID.Int64)
			cm.ClasificadorID = &id
			cm.ClasificadorCodigo = clasifCod.String
		}
		if regimenesNombres.Valid {
			cm.RegimenesNombres = regimenesNombres.String
		}
		lista = append(lista, cm)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return lista, nil
}

// AgregarDesdeModelo copia un concepto modelo específico al tenant local junto con sus relaciones y reglas
func (r *ConceptoTenantRepository) AgregarDesdeModelo(tenantID int, modeloID int) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("error al iniciar transacción: %w", err)
	}
	defer tx.Rollback()

	// 1. Insertar el concepto modelo en conceptos_tenant
	queryInsertConcepto := `
		INSERT INTO conceptos_tenant 
		(tenant_id, concepto_id, modelo_id, nombre_personalizado, frecuencia_meses, clasificador_id, es_extraordinario, requiere_monto, activo,
		 es_pensionable, es_remunerativa, es_base_cts, es_base_beneficios_sociales, es_ocasional, es_afecto_cargas_sociales)
		SELECT 
			$1, concepto_id, id, nombre_personalizado, frecuencia_meses, clasificador_id, es_extraordinario, requiere_monto, true,
			es_pensionable, es_remunerativa, es_base_cts, es_base_beneficios_sociales, es_ocasional, es_afecto_cargas_sociales
		FROM conceptos_modelo
		WHERE id = $2
		ON CONFLICT (tenant_id, modelo_id) DO NOTHING;
	`
	_, err = tx.Exec(queryInsertConcepto, tenantID, modeloID)
	if err != nil {
		return fmt.Errorf("error al clonar concepto modelo a tenant: %w", err)
	}

	// 2. Clonar relaciones con regímenes laborales
	queryRegimenes := `
		INSERT INTO regimen_concepto_tenant (tenant_id, regimen_id, concepto_tenant_id)
		SELECT 
			ct.tenant_id, 
			rcm.regimen_id, 
			ct.id 
		FROM conceptos_tenant ct
		INNER JOIN regimen_concepto_modelo rcm ON ct.modelo_id = rcm.concepto_modelo_id
		WHERE ct.tenant_id = $1 AND ct.modelo_id = $2
		ON CONFLICT (tenant_id, regimen_id, concepto_tenant_id) DO NOTHING;
	`
	_, err = tx.Exec(queryRegimenes, tenantID, modeloID)
	if err != nil {
		return fmt.Errorf("error al clonar relaciones de regímenes: %w", err)
	}

	// 3. Sembrar reglas de cálculo por régimen en base_regimen_tenant
	queryBaseRegimen := `
		INSERT INTO base_regimen_tenant (tenant_id, concepto_calculado_id, regimen_id, concepto_tenant_id, variable_calculo)
		SELECT $1, brd.concepto_calculado_id, brd.regimen_id, ct.id, brd.variable_calculo
		FROM base_regimen_default brd
		INNER JOIN conceptos_tenant ct ON brd.concepto_modelo_id = ct.modelo_id
		WHERE ct.tenant_id = $1 AND ct.modelo_id = $2
		ON CONFLICT (tenant_id, concepto_calculado_id, regimen_id, concepto_tenant_id, variable_calculo) DO NOTHING;
	`
	_, err = tx.Exec(queryBaseRegimen, tenantID, modeloID)
	if err != nil {
		return fmt.Errorf("error al sembrar base_regimen_tenant: %w", err)
	}

	return tx.Commit()
}

// SincronizarConceptoModelo restablece los valores de un concepto tenant a los de su concepto modelo asociado, excepto nombre_personalizado.
func (r *ConceptoTenantRepository) SincronizarConceptoModelo(tenantID int, id int) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("error al iniciar transacción: %w", err)
	}
	defer tx.Rollback()

	// 1. Actualizar campos en conceptos_tenant conservando nombre_personalizado
	queryUpdate := `
		UPDATE conceptos_tenant ct
		SET concepto_id = cm.concepto_id,
		    frecuencia_meses = cm.frecuencia_meses,
		    clasificador_id = cm.clasificador_id,
		    es_extraordinario = cm.es_extraordinario,
		    requiere_monto = cm.requiere_monto,
		    es_pensionable = cm.es_pensionable,
		    es_remunerativa = cm.es_remunerativa,
		    es_base_cts = cm.es_base_cts,
		    es_base_beneficios_sociales = cm.es_base_beneficios_sociales,
		    es_ocasional = cm.es_ocasional,
		    es_afecto_cargas_sociales = cm.es_afecto_cargas_sociales,
		    updated_at = CURRENT_TIMESTAMP
		FROM conceptos_modelo cm
		WHERE ct.modelo_id = cm.id AND ct.id = $1 AND ct.tenant_id = $2;
	`
	res, err := tx.Exec(queryUpdate, id, tenantID)
	if err != nil {
		return fmt.Errorf("error al actualizar concepto_tenant desde modelo: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("no se encontró concepto tenant con modelo asociado para sincronizar")
	}

	// 2. Re-sincronizar relaciones de regímenes laborales
	_, err = tx.Exec(`DELETE FROM regimen_concepto_tenant WHERE concepto_tenant_id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		return fmt.Errorf("error al borrar relaciones de regímenes previas: %w", err)
	}

	queryRegimenes := `
		INSERT INTO regimen_concepto_tenant (tenant_id, regimen_id, concepto_tenant_id)
		SELECT $2, rcm.regimen_id, $1
		FROM regimen_concepto_modelo rcm
		INNER JOIN conceptos_tenant ct ON rcm.concepto_modelo_id = ct.modelo_id
		WHERE ct.id = $1 AND ct.tenant_id = $2
		ON CONFLICT (tenant_id, regimen_id, concepto_tenant_id) DO NOTHING;
	`
	_, err = tx.Exec(queryRegimenes, id, tenantID)
	if err != nil {
		return fmt.Errorf("error al reinsertar relaciones de regímenes: %w", err)
	}

	// 3. Re-sincronizar base_regimen_tenant
	_, err = tx.Exec(`DELETE FROM base_regimen_tenant WHERE concepto_tenant_id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		return fmt.Errorf("error al borrar base_regimen_tenant previo: %w", err)
	}

	queryBaseRegimen := `
		INSERT INTO base_regimen_tenant (tenant_id, concepto_calculado_id, regimen_id, concepto_tenant_id, variable_calculo)
		SELECT $2, brd.concepto_calculado_id, brd.regimen_id, $1, brd.variable_calculo
		FROM base_regimen_default brd
		INNER JOIN conceptos_tenant ct ON brd.concepto_modelo_id = ct.modelo_id
		WHERE ct.id = $1 AND ct.tenant_id = $2
		ON CONFLICT (tenant_id, concepto_calculado_id, regimen_id, concepto_tenant_id, variable_calculo) DO NOTHING;
	`
	_, err = tx.Exec(queryBaseRegimen, id, tenantID)
	if err != nil {
		return fmt.Errorf("error al reinsertar base_regimen_tenant: %w", err)
	}

	return tx.Commit()
}


