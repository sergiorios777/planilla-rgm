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
	if err := rows.Err(); err != nil {
		return nil, err
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
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return lista, totalRegistros, nil
}

// ResolverCodigoPadre determina el código del concepto padre según las reglas SUNAT,
// incluyendo las excepciones para las series 1000 (Sector Público) y 2000 (Régimen Laboral Público).
func ResolverCodigoPadre(cod string) string {
	if len(cod) != 4 {
		return ""
	}
	// Excepción Serie 1000 (Sector Público)
	if strings.HasPrefix(cod, "1") {
		if cod != "1000" {
			return "1000"
		}
		return "" // "1000" es un concepto padre principal
	}
	// Excepción Serie 2000 (Régimen Laboral Público)
	if strings.HasPrefix(cod, "2") {
		if cod != "2000" {
			return "2000"
		}
		return "" // "2000" es un concepto padre principal
	}
	// Regla estándar para series 0100 a 0900
	if !strings.HasSuffix(cod, "00") {
		return cod[:2] + "00"
	}
	return "" // Termina en "00", es un concepto padre principal
}

// RecalcularJerarquias recalcula y actualiza la relación parent_id de todos los conceptos maestros existentes
func (r *ConceptoRepository) RecalcularJerarquias() error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := recalcularJerarquiasTx(tx); err != nil {
		return err
	}

	return tx.Commit()
}

func recalcularJerarquiasTx(tx *sql.Tx) error {
	// 1. Mapa de conceptos padres (donde ResolverCodigoPadre(codigo) == "")
	mapaPadres := make(map[string]int)
	rowsPadres, err := tx.Query(`SELECT id, codigo FROM conceptos_maestros ORDER BY id ASC`)
	if err != nil {
		return err
	}
	for rowsPadres.Next() {
		var id int
		var cod string
		if err := rowsPadres.Scan(&id, &cod); err != nil {
			rowsPadres.Close()
			return err
		}
		if ResolverCodigoPadre(cod) == "" {
			if _, existe := mapaPadres[cod]; !existe {
				mapaPadres[cod] = id
			}
		}
	}
	rowsPadres.Close()

	// 2. Obtener todos los conceptos para evaluar su jerarquía según su columna `codigo`
	type itemConcepto struct {
		id     int
		codigo string
	}
	var todos []itemConcepto
	rowsAll, err := tx.Query(`SELECT id, codigo FROM conceptos_maestros`)
	if err != nil {
		return err
	}
	for rowsAll.Next() {
		var item itemConcepto
		if err := rowsAll.Scan(&item.id, &item.codigo); err != nil {
			rowsAll.Close()
			return err
		}
		todos = append(todos, item)
	}
	rowsAll.Close()

	stmtJerarquia, err := tx.Prepare(`UPDATE conceptos_maestros SET parent_id = $1 WHERE id = $2`)
	if err != nil {
		return err
	}
	defer stmtJerarquia.Close()

	stmtNull, err := tx.Prepare(`UPDATE conceptos_maestros SET parent_id = NULL WHERE id = $1`)
	if err != nil {
		return err
	}
	defer stmtNull.Close()

	for _, item := range todos {
		codPadre := ResolverCodigoPadre(item.codigo)
		if codPadre != "" {
			if idPadre, existe := mapaPadres[codPadre]; existe && idPadre != item.id {
				stmtJerarquia.Exec(idPadre, item.id)
			} else {
				stmtNull.Exec(item.id)
			}
		} else {
			stmtNull.Exec(item.id)
		}
	}

	return nil
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
			stmtInsert.Close()
			return err
		}
	}
	stmtInsert.Close()

	// --- PASADA 2: Recalcular Jerarquía (Padre/Hijo) ---
	if err := recalcularJerarquiasTx(tx); err != nil {
		return err
	}

	// --- Cargamos el diccionario de IDs en memoria para Afectaciones ---
	mapaIds := make(map[string]int)
	rows, err := tx.Query(`SELECT id, codigo, codigo_interno FROM conceptos_maestros`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id int
		var cod, codInt string
		rows.Scan(&id, &cod, &codInt)
		mapaIds[codInt] = id
		if _, existe := mapaIds[cod]; !existe {
			mapaIds[cod] = id
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	rows.Close()

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


