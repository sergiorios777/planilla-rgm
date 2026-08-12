package repository

import (
	"database/sql"
	"fmt"
	"planilla-rgm/internal/models"
)

type ContratoRepository struct {
	db *sql.DB
}

func NewContratoRepository(db *sql.DB) *ContratoRepository {
	return &ContratoRepository{db: db}
}

// ObtenerTodos trae los contratos activos e inactivos de la municipalidad actual
func (r *ContratoRepository) ObtenerTodos(tenantID int) ([]models.Contrato, error) {
	query := `
		SELECT c.id, c.trabajador_id, c.puesto_id, 
		       TO_CHAR(c.fecha_inicio, 'YYYY-MM-DD'), TO_CHAR(c.fecha_fin, 'YYYY-MM-DD'), c.activo,
		       t.numero_documento, t.apellido_paterno || ' ' || t.apellido_materno || ', ' || t.nombres,
		       p.nombre, p.sueldo_presupuestado, rl.descripcion, COALESCE(c.tipo_contrato, ''), COALESCE(c.nivel, ''),
		       c.motivo_baja
		FROM contratos c
		INNER JOIN trabajadores t ON c.trabajador_id = t.id
		INNER JOIN puestos p ON c.puesto_id = p.id
		INNER JOIN regimenes_laborales rl ON p.regimen_id = rl.id
		WHERE c.tenant_id = $1
		ORDER BY c.activo DESC, t.apellido_paterno ASC
	`
	rows, err := r.db.Query(query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.Contrato
	for rows.Next() {
		var c models.Contrato
		var fFin sql.NullString
		var motivoBaja sql.NullString

		err := rows.Scan(&c.ID, &c.TrabajadorID, &c.PuestoID, &c.FechaInicio, &fFin, &c.Activo,
			&c.TrabajadorDoc, &c.TrabajadorNombre, &c.PuestoNombre, &c.SueldoPresupuestado, &c.RegimenDesc, &c.TipoContrato, &c.Nivel, &motivoBaja)
		if err != nil {
			return nil, err
		}
		if fFin.Valid {
			c.FechaFin = &fFin.String
		}
		if motivoBaja.Valid {
			c.MotivoBaja = &motivoBaja.String
		}
		lista = append(lista, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return lista, nil
}

// ObtenerTodosPaginado paginado
func (r *ContratoRepository) ObtenerTodosPaginado(tenantID int, busqueda string, regimenID int, limite int, offset int) ([]models.Contrato, int, error) {
	// 1. Definimos la consulta base con un WHERE inicial para el inquilino
	whereClause := `WHERE c.tenant_id = $1`

	// 2. Inicializamos nuestros argumentos seguros con el tenantID
	args := []interface{}{tenantID}
	contadorArgs := 2 // El siguiente parámetro será $2

	// 3. Lógica de búsqueda dinámica
	if busqueda != "" {
		// Buscamos en DNI, Nombres completos o Nombre del Puesto
		whereClause += fmt.Sprintf(` AND (
			t.numero_documento ILIKE $%d OR 
			t.nombres || ' ' || t.apellido_paterno || ' ' || t.apellido_materno ILIKE $%d OR 
			p.nombre ILIKE $%d
		)`, contadorArgs, contadorArgs, contadorArgs)

		// Añadimos el valor de búsqueda al arreglo (con los comodines % para coincidencia parcial)
		args = append(args, "%"+busqueda+"%")
		contadorArgs++
	}

	// 4. Lógica del filtro por Régimen Laboral
	if regimenID > 0 {
		whereClause += fmt.Sprintf(` AND p.regimen_id = $%d`, contadorArgs)
		args = append(args, regimenID)
		contadorArgs++
	}

	var totalRegistros int
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) FROM contratos c 
		INNER JOIN trabajadores t ON c.trabajador_id = t.id 
		INNER JOIN puestos p ON c.puesto_id = p.id 
		INNER JOIN regimenes_laborales rl ON p.regimen_id = rl.id %s`, whereClause)
	err := r.db.QueryRow(countQuery, args...).Scan(&totalRegistros)
	if err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`
		SELECT c.id, c.trabajador_id, c.puesto_id, 
		       TO_CHAR(c.fecha_inicio, 'YYYY-MM-DD'), TO_CHAR(c.fecha_fin, 'YYYY-MM-DD'), c.activo,
		       t.numero_documento, t.apellido_paterno || ' ' || t.apellido_materno || ', ' || t.nombres AS trabajador_nombre,
		       p.nombre AS puesto_nombre, p.sueldo_presupuestado, rl.descripcion AS regimen_descripcion,
		       COALESCE(c.tipo_contrato, '') AS tipo_contrato, COALESCE(c.nivel, '') AS nivel,
		       c.motivo_baja
		FROM contratos c
		INNER JOIN trabajadores t ON c.trabajador_id = t.id
		INNER JOIN puestos p ON c.puesto_id = p.id
		INNER JOIN regimenes_laborales rl ON p.regimen_id = rl.id
		%s
		ORDER BY c.activo DESC, t.apellido_paterno ASC
		LIMIT $%d OFFSET $%d
	`, whereClause, contadorArgs, contadorArgs+1)

	args = append(args, limite, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var lista []models.Contrato
	for rows.Next() {
		var c models.Contrato
		var fechaFin sql.NullString
		var motivoBaja sql.NullString
		err := rows.Scan(&c.ID, &c.TrabajadorID, &c.PuestoID, &c.FechaInicio, &fechaFin, &c.Activo,
			&c.TrabajadorDoc, &c.TrabajadorNombre, &c.PuestoNombre, &c.SueldoPresupuestado, &c.RegimenDesc, &c.TipoContrato, &c.Nivel, &motivoBaja)
		if err != nil {
			return nil, 0, err
		}
		if fechaFin.Valid {
			c.FechaFin = &fechaFin.String
		}
		if motivoBaja.Valid {
			c.MotivoBaja = &motivoBaja.String
		}
		lista = append(lista, c)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return lista, totalRegistros, nil
}

func (r *ContratoRepository) Crear(c *models.Contrato) error {
	// Iniciamos una Transacción (Si falla algo, se deshace todo)
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	// 1. Insertamos el contrato
	queryContrato := `
		INSERT INTO contratos (tenant_id, trabajador_id, puesto_id, fecha_inicio, fecha_fin, activo, tipo_contrato, nivel)
		VALUES ($1, $2, $3, $4, NULLIF($5, '')::DATE, $6, $7, $8) RETURNING id
	`
	err = tx.QueryRow(queryContrato, c.TenantID, c.TrabajadorID, c.PuestoID, c.FechaInicio, c.FechaFin, c.Activo, c.TipoContrato, c.Nivel).Scan(&c.ID)
	if err != nil {
		tx.Rollback()
		return err
	}

	// 2. Marcamos la Plaza (Puesto) como OCUPADA
	queryPuesto := `UPDATE puestos SET estado = 'OCUPADO' WHERE id = $1 AND tenant_id = $2`
	_, err = tx.Exec(queryPuesto, c.PuestoID, c.TenantID)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Confirmamos los cambios
	return tx.Commit()
}

// TieneContratoActivo verifica si un trabajador ya está ocupando una plaza
func (r *ContratoRepository) TieneContratoActivo(trabajadorID int, tenantID int) (bool, error) {
	var cantidad int
	query := `SELECT COUNT(*) FROM contratos WHERE trabajador_id = $1 AND tenant_id = $2 AND activo = true`

	err := r.db.QueryRow(query, trabajadorID, tenantID).Scan(&cantidad)
	if err != nil {
		return false, err
	}

	// Si la cantidad es mayor a 0, significa que sí tiene un contrato activo
	return cantidad > 0, nil
}

// ObtenerContratosVencimiento trae los contratos que están próximos a vencer en un rango de días determinado (ej. 30 días)
func (r *ContratoRepository) ObtenerContratosVencimiento(tenantID int, dias int) ([]models.Contrato, error) {
	query := `
		SELECT c.id, c.trabajador_id, c.puesto_id, 
		       TO_CHAR(c.fecha_inicio, 'YYYY-MM-DD'), TO_CHAR(c.fecha_fin, 'YYYY-MM-DD'), c.activo,
		       t.numero_documento, t.apellido_paterno || ' ' || t.apellido_materno || ', ' || t.nombres,
		       p.nombre, p.sueldo_presupuestado, rl.descripcion, COALESCE(c.tipo_contrato, ''), COALESCE(c.nivel, '')
		FROM contratos c
		INNER JOIN trabajadores t ON c.trabajador_id = t.id
		INNER JOIN puestos p ON c.puesto_id = p.id
		INNER JOIN regimenes_laborales rl ON p.regimen_id = rl.id
		WHERE c.tenant_id = $1 AND c.activo = true 
		  AND c.fecha_fin IS NOT NULL 
		  AND c.fecha_fin >= CURRENT_DATE 
		  AND c.fecha_fin <= CURRENT_DATE + ($2 * INTERVAL '1 day')
		ORDER BY c.fecha_fin ASC, t.apellido_paterno ASC
	`
	rows, err := r.db.Query(query, tenantID, dias)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.Contrato
	for rows.Next() {
		var c models.Contrato
		var fFin sql.NullString

		err := rows.Scan(&c.ID, &c.TrabajadorID, &c.PuestoID, &c.FechaInicio, &fFin, &c.Activo,
			&c.TrabajadorDoc, &c.TrabajadorNombre, &c.PuestoNombre, &c.SueldoPresupuestado, &c.RegimenDesc, &c.TipoContrato, &c.Nivel)
		if err != nil {
			return nil, err
		}
		if fFin.Valid {
			c.FechaFin = &fFin.String
		}
		lista = append(lista, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return lista, nil
}

// ObtenerActivoPorPuesto busca el contrato activo vigente para un puesto específico
func (r *ContratoRepository) ObtenerActivoPorPuesto(puestoID int, tenantID int) (*models.Contrato, error) {
	var c models.Contrato
	var fFin sql.NullString
	var tipoContrato sql.NullString
	var nivel sql.NullString
	query := `
		SELECT id, trabajador_id, puesto_id, TO_CHAR(fecha_inicio, 'YYYY-MM-DD'), TO_CHAR(fecha_fin, 'YYYY-MM-DD'), activo, COALESCE(tipo_contrato, ''), COALESCE(nivel, '')
		FROM contratos 
		WHERE puesto_id = $1 AND tenant_id = $2 AND activo = true 
		LIMIT 1`
	err := r.db.QueryRow(query, puestoID, tenantID).Scan(&c.ID, &c.TrabajadorID, &c.PuestoID, &c.FechaInicio, &fFin, &c.Activo, &tipoContrato, &nivel)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No hay contrato activo
		}
		return nil, err
	}
	if fFin.Valid {
		c.FechaFin = &fFin.String
	}
	c.TipoContrato = tipoContrato.String
	c.Nivel = nivel.String
	return &c, nil
}

// DarDeBaja desactiva un contrato, registra fecha fin y motivo, y libera la plaza correspondiente de forma transaccional
func (r *ContratoRepository) DarDeBaja(contratoID int, tenantID int, fechaFin string, motivo string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	// 1. Crear snapshot de los conceptos del puesto actuales para el contrato
	querySnapshot := `
		INSERT INTO contrato_conceptos_snapshot (tenant_id, contrato_id, concepto_tenant_id, monto)
		SELECT $1, $2, pc.concepto_tenant_id, COALESCE(pc.monto, 0.00)
		FROM puesto_conceptos pc
		WHERE pc.puesto_id = (SELECT puesto_id FROM contratos WHERE id = $2 AND tenant_id = $1)
		  AND pc.activo = true
		ON CONFLICT (contrato_id, concepto_tenant_id) DO UPDATE SET monto = EXCLUDED.monto
	`
	_, err = tx.Exec(querySnapshot, tenantID, contratoID)
	if err != nil {
		tx.Rollback()
		return err
	}

	// 2. Inactivar el contrato, poner fecha fin y motivo
	queryContrato := `UPDATE contratos SET activo = false, fecha_fin = NULLIF($1, '')::DATE, motivo_baja = $2 WHERE id = $3 AND tenant_id = $4`
	_, err = tx.Exec(queryContrato, fechaFin, motivo, contratoID, tenantID)
	if err != nil {
		tx.Rollback()
		return err
	}

	// 3. Liberar el puesto (Pasarlo a VACANTE)
	queryPuesto := `UPDATE puestos SET estado = 'VACANTE' WHERE id = (SELECT puesto_id FROM contratos WHERE id = $1 AND tenant_id = $2) AND tenant_id = $2`
	_, err = tx.Exec(queryPuesto, contratoID, tenantID)
	if err != nil {
		tx.Rollback()
		return err
	}

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

