package repository

import (
	"database/sql"
	"fmt"
	"planilla-rgm/internal/models"
	"strings"
)

type MefRepository struct {
	db *sql.DB
}

func NewMefRepository(db *sql.DB) *MefRepository {
	return &MefRepository{db: db}
}

// Crear ahora inserta todas las dimensiones del clasificador MEF
func (r *MefRepository) Crear(c *models.ClasificadorMEF) error {
	query := `INSERT INTO clasificadores_mef (anio, codigo, codigo_limpio, descripcion, nivel, tipo_transaccion, activo) 
	          VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`

	return r.db.QueryRow(
		query,
		c.Anio,
		c.CodigoOriginal,
		c.CodigoLimpio,
		c.Descripcion,
		c.Nivel,
		c.TipoTransaccion,
		c.Activo,
	).Scan(&c.ID)
}

// ObtenerTodos devuelve la lista completa sincronizada con los nuevos campos
func (r *MefRepository) ObtenerTodos(busqueda string, anio int, tipo string, limite int, offset int) ([]models.ClasificadorMEF, int, error) {
	query := ` 
	          WHERE 1=1
	`
	// Guardamos los parámetros de forma segura
	var args []interface{}
	contadorArgs := 1

	// 1. Filtro para búsqueda
	if busqueda != "" {
		query += fmt.Sprintf(` AND (codigo_limpio ILIKE '%%' || $%d || '%%' OR descripcion ILIKE '%%' || $%d || '%%')`, contadorArgs, contadorArgs)
		args = append(args, busqueda)
		contadorArgs++
	}

	// 2. Filtro de tipo (Ingreso / Gasto)
	if tipo != "" {
		query += fmt.Sprintf(` AND tipo_transaccion = $%d`, contadorArgs)
		args = append(args, tipo)
		contadorArgs++
	}

	// 3. Filtro por año
	if anio != 0 {
		query += fmt.Sprintf(` AND anio = $%d`, contadorArgs)
		args = append(args, anio)
		contadorArgs++
	}

	// 3.5 Agregar el total de registros
	var totalRegistros int
	countQuery := `SELECT COUNT(*) FROM clasificadores_mef` + query
	err := r.db.QueryRow(countQuery, args...).Scan(&totalRegistros)
	if err != nil {
		return nil, 0, err
	}

	// Cambiamos el query para seleccionar los registros, agregamos al inicio de query el SELECT
	query = `SELECT id, anio, codigo, codigo_limpio, descripcion, nivel, tipo_transaccion, activo, parent_id 
	          FROM clasificadores_mef ` + query

	// 4. Orden y paginación
	query += fmt.Sprintf(` ORDER BY anio DESC, codigo_limpio ASC LIMIT $%d OFFSET $%d`, contadorArgs, contadorArgs+1)
	args = append(args, limite, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var lista []models.ClasificadorMEF
	for rows.Next() {
		var c models.ClasificadorMEF
		err := rows.Scan(
			&c.ID,
			&c.Anio,
			&c.CodigoOriginal,
			&c.CodigoLimpio,
			&c.Descripcion,
			&c.Nivel,
			&c.TipoTransaccion,
			&c.Activo,
			&c.ParentID,
		)
		if err != nil {
			return nil, 0, err
		}
		lista = append(lista, c)
	}
	return lista, totalRegistros, nil
}

// InsertarMasivo recibe una lista grande de clasificadores y los guarda en una sola transacción
func (r *MefRepository) InsertarMasivo(clasificadores []models.ClasificadorMEF) error {
	// Iniciamos la transacción
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	// Si algo sale mal, revertimos todo
	defer tx.Rollback()

	// Preparamos la consulta.
	// ON CONFLICT nos salva: Si subes el mismo Excel dos veces, no dará error, solo actualizará la descripción.
	query := `INSERT INTO clasificadores_mef (anio, codigo, codigo_limpio, descripcion, nivel, tipo_transaccion, activo) 
	          VALUES ($1, $2, $3, $4, $5, $6, $7)
	          ON CONFLICT (anio, codigo_limpio) DO UPDATE SET 
			  descripcion = EXCLUDED.descripcion, 
			  codigo = EXCLUDED.codigo`

	stmt, err := tx.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	// Recorremos la lista y la agregamos a la transacción
	for _, c := range clasificadores {
		_, err := stmt.Exec(c.Anio, c.CodigoOriginal, c.CodigoLimpio, c.Descripcion, c.Nivel, c.TipoTransaccion, c.Activo)
		if err != nil {
			return err
		}
	}

	// Confirmamos y guardamos todo en la base de datos
	return tx.Commit()
}

// VincularJerarquia calcula y asigna los parent_id automáticamente basándose en los puntos del código
func (r *MefRepository) VincularJerarquia(anio int) error {
	// 1. Traemos todos los clasificadores de ese año
	query := `SELECT id, codigo_limpio FROM clasificadores_mef WHERE anio = $1`
	rows, err := r.db.Query(query, anio)
	if err != nil {
		return err
	}
	defer rows.Close()

	// 2. Creamos un "Mapa" (Diccionario) para buscar IDs en microsegundos
	// Ej: mapaIds["1.1.1.1"] = 45
	mapaIds := make(map[string]int)
	for rows.Next() {
		var id int
		var codigo string
		rows.Scan(&id, &codigo)
		mapaIds[codigo] = id
	}

	// 3. Preparamos una transacción para actualizar todos los padres de golpe
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`UPDATE clasificadores_mef SET parent_id = $1 WHERE id = $2`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	// 4. Analizamos cada código para encontrar a su padre
	for codigo, idHijo := range mapaIds {
		// Buscamos la posición del último punto
		ultimoPunto := strings.LastIndex(codigo, ".")

		// Si tiene un punto (ej. "1.1.1"), tiene un padre
		if ultimoPunto != -1 {
			// Cortamos el string hasta el último punto (ej. "1.1")
			codigoPadre := codigo[:ultimoPunto]

			// Si el padre existe en nuestro mapa, hacemos el vínculo
			if idPadre, existe := mapaIds[codigoPadre]; existe {
				_, err := stmt.Exec(idPadre, idHijo)
				if err != nil {
					return err
				}
			}
		}
	}

	// 5. Guardamos los cambios
	return tx.Commit()
}
