package repository

import (
	"database/sql"
	"planilla-rgm/internal/models"
	"strings"
)

type ConceptoRepository struct {
	db *sql.DB
}

func NewConceptoRepository(db *sql.DB) *ConceptoRepository {
	return &ConceptoRepository{db: db}
}

// ObtenerTodos trae la lista para la pantalla
func (r *ConceptoRepository) ObtenerTodos() ([]models.ConceptoMaestro, error) {
	query := `SELECT id, parent_id, codigo, descripcion, tipo, activo FROM conceptos_maestros ORDER BY codigo ASC`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.ConceptoMaestro
	for rows.Next() {
		var c models.ConceptoMaestro
		err := rows.Scan(&c.ID, &c.ParentID, &c.Codigo, &c.Descripcion, &c.Tipo, &c.Activo)
		if err != nil {
			return nil, err
		}
		lista = append(lista, c)
	}
	return lista, nil
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
		INSERT INTO conceptos_maestros (codigo, descripcion, tipo, activo) 
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (codigo) DO UPDATE SET descripcion = EXCLUDED.descripcion, tipo = EXCLUDED.tipo
	`)
	if err != nil {
		return err
	}
	for _, c := range conceptos {
		if _, err := stmtInsert.Exec(c.Codigo, c.Descripcion, c.Tipo, c.Activo); err != nil {
			return err
		}
	}
	stmtInsert.Close()

	// --- Cargamos el diccionario de IDs recién creados en memoria ---
	mapaIds := make(map[string]int)
	rows, err := tx.Query(`SELECT id, codigo FROM conceptos_maestros`)
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
