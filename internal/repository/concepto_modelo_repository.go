package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"planilla-rgm/internal/models"
	"strings"
)

type ConceptoModeloRepository struct {
	db *sql.DB
}

func NewConceptoModeloRepository(db *sql.DB) *ConceptoModeloRepository {
	return &ConceptoModeloRepository{db: db}
}

// ObtenerTodos ahora trae todos los conceptos y agrupa sus regímenes separados por comas
func (r *ConceptoModeloRepository) ObtenerTodos() ([]models.ConceptoModelo, error) {
	query := `
		SELECT cm.id, cm.concepto_id, cm.nombre_personalizado, 
		       cm.frecuencia_meses, cm.clasificador_id, cm.es_extraordinario, cm.requiere_monto,
		       cm.es_pensionable, cm.es_remunerativa, cm.es_base_cts, cm.es_base_beneficios_sociales, cm.es_ocasional, cm.es_afecto_cargas_sociales,
		       cma.codigo, cma.descripcion, cl.codigo_limpio,
		       COALESCE(STRING_AGG(rl.descripcion, ', '), 'Sin régimen') AS regimenes_nombres
		FROM conceptos_modelo cm
		INNER JOIN conceptos_maestros cma ON cm.concepto_id = cma.id
		LEFT JOIN clasificadores_mef cl ON cm.clasificador_id = cl.id
		LEFT JOIN regimen_concepto_modelo rcm ON cm.id = rcm.concepto_modelo_id
		LEFT JOIN regimenes_laborales rl ON rcm.regimen_id = rl.id
		GROUP BY cm.id, cma.codigo, cma.descripcion, cl.codigo_limpio
		ORDER BY cm.id DESC`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.ConceptoModelo
	for rows.Next() {
		var c models.ConceptoModelo
		var clasificadorID sql.NullInt64
		var clasificadorCodigo sql.NullString

		err := rows.Scan(
			&c.ID, &c.ConceptoID, &c.NombrePersonalizado, &c.FrecuenciaMeses,
			&clasificadorID, &c.EsExtraordinario, &c.RequiereMonto,
			&c.EsPensionable, &c.EsRemunerativa, &c.EsBaseCts, &c.EsBaseBeneficiosSociales, &c.EsOcasional, &c.EsAfectoCargasSociales,
			&c.ConceptoCodigo, &c.ConceptoDescripcion, &clasificadorCodigo, &c.RegimenesNombres,
		)
		if err != nil {
			return nil, err
		}

		if clasificadorID.Valid {
			idInt := int(clasificadorID.Int64)
			c.ClasificadorID = &idInt
			c.ClasificadorCodigo = clasificadorCodigo.String
		}

		lista = append(lista, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return lista, nil
}

// ObtenerPorID busca un concepto y además busca qué regímenes tiene marcados
func (r *ConceptoModeloRepository) ObtenerPorID(id int) (*models.ConceptoModelo, error) {
	// 1. Obtenemos los datos base del concepto
	query := `
		SELECT id, concepto_id, nombre_personalizado, frecuencia_meses, clasificador_id, 
		       es_extraordinario, requiere_monto, es_pensionable, es_remunerativa, es_base_cts, es_base_beneficios_sociales, es_ocasional, es_afecto_cargas_sociales
		FROM conceptos_modelo WHERE id = $1`

	var c models.ConceptoModelo
	var clasificadorID sql.NullInt64

	err := r.db.QueryRow(query, id).Scan(
		&c.ID, &c.ConceptoID, &c.NombrePersonalizado, &c.FrecuenciaMeses,
		&clasificadorID, &c.EsExtraordinario, &c.RequiereMonto,
		&c.EsPensionable, &c.EsRemunerativa, &c.EsBaseCts, &c.EsBaseBeneficiosSociales, &c.EsOcasional, &c.EsAfectoCargasSociales,
	)
	if err != nil {
		return nil, err
	}

	if clasificadorID.Valid {
		idInt := int(clasificadorID.Int64)
		c.ClasificadorID = &idInt
	}

	// 2. Obtenemos el arreglo de IDs de los regímenes asociados
	queryRegimenes := `SELECT regimen_id FROM regimen_concepto_modelo WHERE concepto_modelo_id = $1`
	rows, err := r.db.Query(queryRegimenes, id)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var rID int
			rows.Scan(&rID)
			c.RegimenesIDs = append(c.RegimenesIDs, rID)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}

	return &c, nil
}

// Crear utiliza una Transacción para asegurar la integridad de datos
func (r *ConceptoModeloRepository) Crear(c *models.ConceptoModelo) error {
	// Iniciamos la transacción
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	// 1. Insertamos el concepto base
	queryConcepto := `
		INSERT INTO conceptos_modelo 
		(concepto_id, nombre_personalizado, frecuencia_meses, clasificador_id, 
		 es_extraordinario, requiere_monto, es_pensionable, es_remunerativa, es_base_cts, es_base_beneficios_sociales, es_ocasional, es_afecto_cargas_sociales) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) RETURNING id`

	err = tx.QueryRow(queryConcepto,
		c.ConceptoID, c.NombrePersonalizado, c.FrecuenciaMeses,
		c.ClasificadorID, c.EsExtraordinario, c.RequiereMonto,
		c.EsPensionable, c.EsRemunerativa, c.EsBaseCts, c.EsBaseBeneficiosSociales, c.EsOcasional, c.EsAfectoCargasSociales,
	).Scan(&c.ID)

	if err != nil {
		tx.Rollback() // Si falla, cancelamos todo
		return err
	}

	// 2. Insertamos la relación con los regímenes seleccionados
	queryRegimen := `INSERT INTO regimen_concepto_modelo (regimen_id, concepto_modelo_id) VALUES ($1, $2)`
	for _, regID := range c.RegimenesIDs {
		_, err = tx.Exec(queryRegimen, regID, c.ID)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	// Confirmamos la transacción
	return tx.Commit()
}

// Actualizar limpia los regímenes anteriores y guarda los nuevos
func (r *ConceptoModeloRepository) Actualizar(c *models.ConceptoModelo) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	// 1. Actualizamos los datos del concepto
	queryUpdate := `
		UPDATE conceptos_modelo 
		SET concepto_id = $1, nombre_personalizado = $2, frecuencia_meses = $3, 
		    clasificador_id = $4, es_extraordinario = $5, requiere_monto = $6,
		    es_pensionable = $7, es_remunerativa = $8, es_base_cts = $9, es_base_beneficios_sociales = $10,
		    es_ocasional = $11, es_afecto_cargas_sociales = $12, updated_at = CURRENT_TIMESTAMP
		WHERE id = $13`

	_, err = tx.Exec(queryUpdate,
		c.ConceptoID, c.NombrePersonalizado, c.FrecuenciaMeses,
		c.ClasificadorID, c.EsExtraordinario, c.RequiereMonto,
		c.EsPensionable, c.EsRemunerativa, c.EsBaseCts, c.EsBaseBeneficiosSociales,
		c.EsOcasional, c.EsAfectoCargasSociales, c.ID,
	)
	if err != nil {
		tx.Rollback()
		return err
	}

	// 2. Borramos las relaciones antiguas
	_, err = tx.Exec(`DELETE FROM regimen_concepto_modelo WHERE concepto_modelo_id = $1`, c.ID)
	if err != nil {
		tx.Rollback()
		return err
	}

	// 3. Insertamos las nuevas relaciones
	queryRegimen := `INSERT INTO regimen_concepto_modelo (regimen_id, concepto_modelo_id) VALUES ($1, $2)`
	for _, regID := range c.RegimenesIDs {
		_, err = tx.Exec(queryRegimen, regID, c.ID)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

// Eliminar quita el concepto (y gracias al ON DELETE CASCADE de la BD, se borran las relaciones)
func (r *ConceptoModeloRepository) Eliminar(id int) error {
	_, err := r.db.Exec(`DELETE FROM conceptos_modelo WHERE id = $1`, id)
	return err
}

// ObtenerTodosPaginacion obtiene los conceptos modelo filtrados y paginados
func (r *ConceptoModeloRepository) ObtenerTodosPaginacion(busqueda string, atributo string, regimenID int, limite int, offset int) ([]models.ConceptoModelo, int, error) {
	whereClause := "WHERE 1=1"
	var params []interface{}
	paramIndex := 1

	if busqueda != "" {
		whereClause += fmt.Sprintf(" AND (cm.nombre_personalizado ILIKE $%d OR cma.codigo ILIKE $%d OR cl.codigo_limpio ILIKE $%d)", paramIndex, paramIndex+1, paramIndex+2)
		params = append(params, "%"+busqueda+"%", "%"+busqueda+"%", "%"+busqueda+"%")
		paramIndex += 3
	}

	if atributo != "" {
		switch atributo {
		case "es_pensionable":
			whereClause += " AND cm.es_pensionable = true"
		case "es_remunerativa":
			whereClause += " AND cm.es_remunerativa = true"
		case "es_base_cts":
			whereClause += " AND cm.es_base_cts = true"
		case "es_base_beneficios_sociales":
			whereClause += " AND cm.es_base_beneficios_sociales = true"
		case "es_afecto_cargas_sociales":
			whereClause += " AND cm.es_afecto_cargas_sociales = true"
		}
	}

	if regimenID > 0 {
		whereClause += fmt.Sprintf(" AND EXISTS (SELECT 1 FROM regimen_concepto_modelo rcm_f WHERE rcm_f.concepto_modelo_id = cm.id AND rcm_f.regimen_id = $%d)", paramIndex)
		params = append(params, regimenID)
		paramIndex++
	}

	var totalRegistros int
	countQuery := fmt.Sprintf(
		`
			SELECT COUNT(DISTINCT cm.id) FROM conceptos_modelo cm
			INNER JOIN conceptos_maestros cma ON cm.concepto_id = cma.id
			LEFT JOIN clasificadores_mef cl ON cm.clasificador_id = cl.id
			LEFT JOIN regimen_concepto_modelo rcm ON cm.id = rcm.concepto_modelo_id
			LEFT JOIN regimenes_laborales rl ON rcm.regimen_id = rl.id
			%s
		`,
		whereClause,
	)

	err := r.db.QueryRow(countQuery, params...).Scan(&totalRegistros)
	if err != nil {
		log.Println("Error al obtener el total de registros en conceptos_modelo:", err)
		return nil, 0, err
	}

	query := fmt.Sprintf(`
		SELECT cm.id, cm.concepto_id, cm.nombre_personalizado, 
		       cm.frecuencia_meses, cm.clasificador_id, cm.es_extraordinario, cm.requiere_monto,
		       cm.es_pensionable, cm.es_remunerativa, cm.es_base_cts, cm.es_base_beneficios_sociales, cm.es_ocasional, cm.es_afecto_cargas_sociales,
		       cma.codigo, cma.descripcion, cl.codigo_limpio,
		       COALESCE(STRING_AGG(rl.descripcion, ', '), 'Sin régimen') AS regimenes_nombres
		FROM conceptos_modelo cm
		INNER JOIN conceptos_maestros cma ON cm.concepto_id = cma.id
		LEFT JOIN clasificadores_mef cl ON cm.clasificador_id = cl.id
		LEFT JOIN regimen_concepto_modelo rcm ON cm.id = rcm.concepto_modelo_id
		LEFT JOIN regimenes_laborales rl ON rcm.regimen_id = rl.id
		%s
		GROUP BY cm.id, cma.codigo, cma.descripcion, cl.codigo_limpio
		ORDER BY cm.id DESC
		LIMIT $%d OFFSET $%d
	`,
		whereClause,
		paramIndex,
		paramIndex+1,
	)

	params = append(params, limite, offset)

	rows, err := r.db.Query(query, params...)
	if err != nil {
		log.Println("Error al paginar conceptos modelo:", err)
		return nil, 0, err
	}
	defer rows.Close()

	var lista []models.ConceptoModelo
	for rows.Next() {
		var c models.ConceptoModelo
		var clasificadorID sql.NullInt64
		var clasificadorCodigo sql.NullString

		err := rows.Scan(
			&c.ID, &c.ConceptoID, &c.NombrePersonalizado, &c.FrecuenciaMeses,
			&clasificadorID, &c.EsExtraordinario, &c.RequiereMonto,
			&c.EsPensionable, &c.EsRemunerativa, &c.EsBaseCts, &c.EsBaseBeneficiosSociales, &c.EsOcasional, &c.EsAfectoCargasSociales,
			&c.ConceptoCodigo, &c.ConceptoDescripcion, &clasificadorCodigo, &c.RegimenesNombres,
		)
		if err != nil {
			return nil, 0, err
		}

		if clasificadorID.Valid {
			idInt := int(clasificadorID.Int64)
			c.ClasificadorID = &idInt
			c.ClasificadorCodigo = clasificadorCodigo.String
		}

		lista = append(lista, c)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return lista, totalRegistros, nil
}

// ObtenerMapaMaestros retorna un mapa de codigo -> id de conceptos_maestros
func (r *ConceptoModeloRepository) ObtenerMapaMaestros() (map[string]int, error) {
	rows, err := r.db.Query("SELECT id, codigo FROM conceptos_maestros")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	mapa := make(map[string]int)
	for rows.Next() {
		var id int
		var codigo string
		if err := rows.Scan(&id, &codigo); err != nil {
			return nil, err
		}
		mapa[strings.TrimSpace(codigo)] = id
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return mapa, nil
}

// ObtenerMapaClasificadores retorna un mapa de codigo_limpio -> id de clasificadores_mef
func (r *ConceptoModeloRepository) ObtenerMapaClasificadores() (map[string]int, error) {
	rows, err := r.db.Query("SELECT id, codigo_limpio FROM clasificadores_mef")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	mapa := make(map[string]int)
	for rows.Next() {
		var id int
		var codigoLimpio string
		if err := rows.Scan(&id, &codigoLimpio); err != nil {
			return nil, err
		}
		mapa[strings.TrimSpace(codigoLimpio)] = id
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return mapa, nil
}

// ObtenerMapaRegimenes retorna un mapa de nombre estándar -> id basado en regimenes_laborales
func (r *ConceptoModeloRepository) ObtenerMapaRegimenes() (map[string]int, error) {
	rows, err := r.db.Query("SELECT id, codigo FROM regimenes_laborales")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	mapa := make(map[string]int)
	for rows.Next() {
		var id int
		var codigo string
		if err := rows.Scan(&id, &codigo); err != nil {
			return nil, err
		}
		switch codigo {
		case "276":
			mapa["DL 276"] = id
		case "728":
			mapa["DL 728"] = id
		case "1057":
			mapa["DL 1057"] = id
		case "30057":
			mapa["LEY SERVIR"] = id
			mapa["LEY 30057"] = id
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return mapa, nil
}

// GuardarConceptoModeloImportado realiza un UPSERT de concepto_modelo y reinserta relaciones
func (r *ConceptoModeloRepository) GuardarConceptoModeloImportado(tx *sql.Tx, cm *models.ConceptoModelo, regimenesIDs []int) error {
	queryConcepto := `
		INSERT INTO conceptos_modelo 
		(concepto_id, nombre_personalizado, frecuencia_meses, clasificador_id, 
		 es_extraordinario, requiere_monto, es_pensionable, es_remunerativa, es_base_cts, es_base_beneficios_sociales, es_ocasional, es_afecto_cargas_sociales) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (nombre_personalizado) DO UPDATE SET
			concepto_id = EXCLUDED.concepto_id,
			frecuencia_meses = EXCLUDED.frecuencia_meses,
			clasificador_id = EXCLUDED.clasificador_id,
			es_extraordinario = EXCLUDED.es_extraordinario,
			requiere_monto = EXCLUDED.requiere_monto,
			es_pensionable = EXCLUDED.es_pensionable,
			es_remunerativa = EXCLUDED.es_remunerativa,
			es_base_cts = EXCLUDED.es_base_cts,
			es_base_beneficios_sociales = EXCLUDED.es_base_beneficios_sociales,
			es_ocasional = EXCLUDED.es_ocasional,
			es_afecto_cargas_sociales = EXCLUDED.es_afecto_cargas_sociales,
			updated_at = CURRENT_TIMESTAMP
		RETURNING id`

	err := tx.QueryRow(queryConcepto,
		cm.ConceptoID, cm.NombrePersonalizado, cm.FrecuenciaMeses,
		cm.ClasificadorID, cm.EsExtraordinario, cm.RequiereMonto,
		cm.EsPensionable, cm.EsRemunerativa, cm.EsBaseCts, cm.EsBaseBeneficiosSociales, cm.EsOcasional, cm.EsAfectoCargasSociales,
	).Scan(&cm.ID)

	if err != nil {
		return err
	}

	// Limpiar relaciones anteriores
	_, err = tx.Exec(`DELETE FROM regimen_concepto_modelo WHERE concepto_modelo_id = $1`, cm.ID)
	if err != nil {
		return err
	}

	// Insertar nuevas relaciones
	queryRegimen := `INSERT INTO regimen_concepto_modelo (regimen_id, concepto_modelo_id) VALUES ($1, $2)`
	for _, regID := range regimenesIDs {
		_, err = tx.Exec(queryRegimen, regID, cm.ID)
		if err != nil {
			return err
		}
	}

	return nil
}

// CRUD ReglaFinanciamientoModelo (SaaS)

func (r *ConceptoModeloRepository) ObtenerReglasFinanciamientoModelo(ctx context.Context, conceptoModeloID int) ([]models.ReglaFinanciamientoModelo, error) {
	query := `
		SELECT r.id, r.concepto_modelo_id, r.regimen_id, r.meta_id, r.fuente_rubro_id, r.activo, r.created_at, r.updated_at,
		       cm.nombre_personalizado AS concepto_nombre,
		       COALESCE(rl.descripcion, 'Todos') AS regimen_nombre,
		       COALESCE(m.codigo, '') AS meta_codigo,
		       COALESCE(m.descripcion, '') AS meta_descripcion,
		       COALESCE(fr.codigo_fuente_rubro, fr.rubro, '') AS fuente_rubro_codigo,
		       COALESCE(fr.rubro, '') AS fuente_rubro_descripcion
		FROM reglas_financiamiento_modelo r
		INNER JOIN conceptos_modelo cm ON r.concepto_modelo_id = cm.id
		LEFT JOIN regimenes_laborales rl ON r.regimen_id = rl.id
		LEFT JOIN metas_presupuestales m ON r.meta_id = m.id
		LEFT JOIN fuentes_rubros fr ON r.fuente_rubro_id = fr.id
		WHERE r.concepto_modelo_id = $1
		ORDER BY r.id DESC
	`
	rows, err := r.db.QueryContext(ctx, query, conceptoModeloID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.ReglaFinanciamientoModelo
	for rows.Next() {
		var reg models.ReglaFinanciamientoModelo
		var regimenID, metaID, rubroID sql.NullInt64
		err := rows.Scan(
			&reg.ID, &reg.ConceptoModeloID, &regimenID, &metaID, &rubroID, &reg.Activo, &reg.CreatedAt, &reg.UpdatedAt,
			&reg.ConceptoNombre, &reg.RegimenNombre, &reg.MetaCodigo, &reg.MetaDescripcion, &reg.FuenteRubroCodigo, &reg.FuenteRubroDescripcion,
		)
		if err != nil {
			return nil, err
		}
		if regimenID.Valid {
			v := int(regimenID.Int64)
			reg.RegimenID = &v
		}
		if metaID.Valid {
			v := int(metaID.Int64)
			reg.MetaID = &v
		}
		if rubroID.Valid {
			v := int(rubroID.Int64)
			reg.FuenteRubroID = &v
		}
		lista = append(lista, reg)
	}
	return lista, rows.Err()
}

func (r *ConceptoModeloRepository) CrearReglaFinanciamientoModelo(ctx context.Context, regla *models.ReglaFinanciamientoModelo) error {
	query := `
		INSERT INTO reglas_financiamiento_modelo (concepto_modelo_id, regimen_id, meta_id, fuente_rubro_id, activo)
		VALUES ($1, $2, $3, $4, $5) RETURNING id
	`
	return r.db.QueryRowContext(ctx, query, regla.ConceptoModeloID, regla.RegimenID, regla.MetaID, regla.FuenteRubroID, regla.Activo).Scan(&regla.ID)
}

func (r *ConceptoModeloRepository) EliminarReglaFinanciamientoModelo(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM reglas_financiamiento_modelo WHERE id = $1`, id)
	return err
}

