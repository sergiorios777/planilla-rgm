package repository

import (
	"database/sql"
	"planilla-rgm/internal/models"
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
		       cm.es_pensionable, cm.es_remunerativa, cm.es_base_cts, cm.es_base_beneficios_sociales,
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
			&c.EsPensionable, &c.EsRemunerativa, &c.EsBaseCts, &c.EsBaseBeneficiosSociales,
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
	return lista, nil
}

// ObtenerPorID busca un concepto y además busca qué regímenes tiene marcados
func (r *ConceptoModeloRepository) ObtenerPorID(id int) (*models.ConceptoModelo, error) {
	// 1. Obtenemos los datos base del concepto
	query := `
		SELECT id, concepto_id, nombre_personalizado, frecuencia_meses, clasificador_id, 
		       es_extraordinario, requiere_monto, es_pensionable, es_remunerativa, es_base_cts, es_base_beneficios_sociales
		FROM conceptos_modelo WHERE id = $1`

	var c models.ConceptoModelo
	var clasificadorID sql.NullInt64

	err := r.db.QueryRow(query, id).Scan(
		&c.ID, &c.ConceptoID, &c.NombrePersonalizado, &c.FrecuenciaMeses,
		&clasificadorID, &c.EsExtraordinario, &c.RequiereMonto,
		&c.EsPensionable, &c.EsRemunerativa, &c.EsBaseCts, &c.EsBaseBeneficiosSociales,
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
		 es_extraordinario, requiere_monto, es_pensionable, es_remunerativa, es_base_cts, es_base_beneficios_sociales) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id`

	err = tx.QueryRow(queryConcepto,
		c.ConceptoID, c.NombrePersonalizado, c.FrecuenciaMeses,
		c.ClasificadorID, c.EsExtraordinario, c.RequiereMonto,
		c.EsPensionable, c.EsRemunerativa, c.EsBaseCts, c.EsBaseBeneficiosSociales,
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
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $11`

	_, err = tx.Exec(queryUpdate,
		c.ConceptoID, c.NombrePersonalizado, c.FrecuenciaMeses,
		c.ClasificadorID, c.EsExtraordinario, c.RequiereMonto,
		c.EsPensionable, c.EsRemunerativa, c.EsBaseCts, c.EsBaseBeneficiosSociales,
		c.ID,
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
