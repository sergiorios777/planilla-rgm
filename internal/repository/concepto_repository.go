package repository

import (
	"database/sql"
	"fmt"
	"planilla-rgm/internal/models"
	"strconv"
	"strings"
)

type ConceptoRepository struct {
	db *sql.DB
}

func NewConceptoRepository(db *sql.DB) *ConceptoRepository {
	return &ConceptoRepository{db: db}
}

// ObtenerPadres obtiene todos los conceptos maestros padre (parent_id IS NULL)
func (r *ConceptoRepository) ObtenerPadres() ([]models.ConceptoMaestro, error) {
	query := `SELECT id, parent_id, codigo, codigo_interno, descripcion, tipo, activo, origen FROM conceptos_maestros WHERE parent_id IS NULL ORDER BY codigo ASC`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.ConceptoMaestro
	for rows.Next() {
		var c models.ConceptoMaestro
		err := rows.Scan(&c.ID, &c.ParentID, &c.Codigo, &c.CodigoInterno, &c.Descripcion, &c.Tipo, &c.Activo, &c.Origen)
		if err != nil {
			return nil, err
		}
		lista = append(lista, c)
	}
	return lista, nil
}

// ObtenerTodos trae la lista para la pantalla con filtros
func (r *ConceptoRepository) ObtenerTodos(busqueda string, tipo string, parentIDStr string, limite int, offset int) ([]models.ConceptoMaestro, int, error) {
	whereClause := "WHERE 1=1"
	
	var args []interface{}
	contadorArgs := 1

	if busqueda != "" {
		whereClause += fmt.Sprintf(` AND (codigo ILIKE '%%' || $%d || '%%' OR descripcion ILIKE '%%' || $%d || '%%')`, contadorArgs, contadorArgs)
		args = append(args, busqueda)
		contadorArgs++
	}

	if tipo != "" {
		whereClause += fmt.Sprintf(` AND tipo = $%d`, contadorArgs)
		args = append(args, tipo)
		contadorArgs++
	}

	if parentIDStr != "" {
		if parentID, err := strconv.Atoi(parentIDStr); err == nil {
			whereClause += fmt.Sprintf(` AND parent_id = $%d`, contadorArgs)
			args = append(args, parentID)
			contadorArgs++
		}
	}

	var totalRegistros int
	countQuery := `SELECT COUNT(*) FROM conceptos_maestros ` + whereClause
	err := r.db.QueryRow(countQuery, args...).Scan(&totalRegistros)
	if err != nil {
		return nil, 0, err
	}

	query := `SELECT id, parent_id, codigo, codigo_interno, descripcion, tipo, activo, origen FROM conceptos_maestros ` + whereClause
	query += fmt.Sprintf(` ORDER BY codigo ASC LIMIT $%d OFFSET $%d`, contadorArgs, contadorArgs+1)
	args = append(args, limite, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var lista []models.ConceptoMaestro
	for rows.Next() {
		var c models.ConceptoMaestro
		err := rows.Scan(&c.ID, &c.ParentID, &c.Codigo, &c.CodigoInterno, &c.Descripcion, &c.Tipo, &c.Activo, &c.Origen)
		if err != nil {
			return nil, 0, err
		}
		lista = append(lista, c)
	}
	return lista, totalRegistros, nil
}

// ProcesarImportacion ejecuta el algoritmo de 3 pasadas en una sola transacción segura
func (r *ConceptoRepository) ProcesarImportacion(conceptos []models.ConceptoMaestro, afectaciones map[string][]string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// --- PASADA 1: Insertar Conceptos ---
	stmtInsert, err := tx.Prepare(`
		INSERT INTO conceptos_maestros (codigo, codigo_interno, descripcion, tipo, activo, origen) 
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (codigo_interno) DO UPDATE SET codigo = EXCLUDED.codigo, descripcion = EXCLUDED.descripcion, tipo = EXCLUDED.tipo, origen = EXCLUDED.origen
	`)
	if err != nil {
		return err
	}
	for _, c := range conceptos {
		if _, err := stmtInsert.Exec(c.Codigo, c.CodigoInterno, c.Descripcion, c.Tipo, c.Activo, c.Origen); err != nil {
			return err
		}
	}
	stmtInsert.Close()

	// --- Cargamos el diccionario de IDs recién creados en memoria ---
	mapaIds := make(map[string]int)
	rows, err := tx.Query(`SELECT id, codigo_interno FROM conceptos_maestros`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id int
		var cod string
		rows.Scan(&id, &cod)
		mapaIds[cod] = id
	}
	rows.Close()

	// --- PASADA 2: Vincular Jerarquía (Padre/Hijo) ---
	stmtJerarquia, err := tx.Prepare(`UPDATE conceptos_maestros SET parent_id = $1 WHERE id = $2`)
	if err != nil {
		return err
	}
	for cod, idHijo := range mapaIds {
		// Regla SUNAT: Si tiene 4 caracteres y NO termina en "00" (ej. "0121"), su padre es "0100"
		if len(cod) == 4 && !strings.HasSuffix(cod, "00") {
			codigoPadre := cod[:2] + "00"
			if idPadre, existe := mapaIds[codigoPadre]; existe {
				stmtJerarquia.Exec(idPadre, idHijo)
			}
		}
	}
	stmtJerarquia.Close()

	// --- PASADA 3: Vincular Afectaciones ---
	stmtAfecta, err := tx.Prepare(`
		INSERT INTO conceptos_afectaciones (concepto_base_id, concepto_derivado_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`)
	if err != nil {
		return err
	}
	for codBase, derivados := range afectaciones {
		idBase, existeBase := mapaIds[codBase]
		if !existeBase {
			continue
		}
		for _, codDerivado := range derivados {
			codDerivado = strings.TrimSpace(codDerivado) // Limpiamos espacios
			if idDerivado, existeDer := mapaIds[codDerivado]; existeDer {
				stmtAfecta.Exec(idBase, idDerivado)
			}
		}
	}
	stmtAfecta.Close()

	// Confirmamos todo
	return tx.Commit()
}

func (r *ContratoRepository) ObtenerPorID(id int, tenantID int) (models.Contrato, error) {
	var c models.Contrato
	var fFin sql.NullString
	var tipoContrato sql.NullString
	var nivel sql.NullString
	var motivoBaja sql.NullString
	query := `
		SELECT c.id, c.trabajador_id, c.puesto_id, 
		       TO_CHAR(c.fecha_inicio, 'YYYY-MM-DD'), TO_CHAR(c.fecha_fin, 'YYYY-MM-DD'), c.activo,
		       t.apellido_paterno || ' ' || t.apellido_materno || ', ' || t.nombres AS trabajador_nombre,
		       p.nombre AS puesto_nombre, COALESCE(c.tipo_contrato, '') AS tipo_contrato, COALESCE(c.nivel, '') AS nivel,
		       c.motivo_baja
		FROM contratos c
		INNER JOIN trabajadores t ON c.trabajador_id = t.id
		INNER JOIN puestos p ON c.puesto_id = p.id
		WHERE c.id = $1 AND c.tenant_id = $2
	`
	err := r.db.QueryRow(query, id, tenantID).Scan(
		&c.ID, &c.TrabajadorID, &c.PuestoID, &c.FechaInicio, &fFin, &c.Activo,
		&c.TrabajadorNombre, &c.PuestoNombre, &tipoContrato, &nivel, &motivoBaja,
	)
	if fFin.Valid {
		c.FechaFin = &fFin.String
	}
	if tipoContrato.Valid {
		c.TipoContrato = tipoContrato.String
	}
	if nivel.Valid {
		c.Nivel = nivel.String
	}
	if motivoBaja.Valid {
		c.MotivoBaja = &motivoBaja.String
	}
	return c, err
}

func (r *ContratoRepository) Actualizar(c *models.Contrato) error {
	// Solo actualizamos fechas, estado y nivel. Si el contrato pasa a inactivo, la plaza se libera.
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	query := `UPDATE contratos SET fecha_inicio = $1, fecha_fin = NULLIF($2, '')::DATE, activo = $3, nivel = $4 WHERE id = $5 AND tenant_id = $6`
	_, err = tx.Exec(query, c.FechaInicio, c.FechaFin, c.Activo, c.Nivel, c.ID, c.TenantID)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Si se inactiva el contrato, liberamos la plaza a VACANTE
	if !c.Activo {
		_, err = tx.Exec(`UPDATE puestos SET estado = 'VACANTE' WHERE id = $1`, c.PuestoID)
		if err != nil {
			tx.Rollback()
			return err
		}
	} else {
		// Si se reactiva (por si acaso), lo marcamos OCUPADO
		_, err = tx.Exec(`UPDATE puestos SET estado = 'OCUPADO' WHERE id = $1`, c.PuestoID)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}
