package repository

import (
	"database/sql"
	"fmt"
	"planilla-rgm/internal/models"
	"strings"
)

type AsistenciaRepository struct {
	db *sql.DB
}

func NewAsistenciaRepository(db *sql.DB) *AsistenciaRepository {
	return &AsistenciaRepository{db: db}
}

// ObtenerContratosParaSelect trae la lista de trabajadores activos para el formulario
func (r *AsistenciaRepository) ObtenerContratosParaSelect(tenantID int) ([]models.ContratoSelect, error) {
	query := `
		SELECT c.id, t.numero_documento, t.apellido_paterno || ' ' || t.apellido_materno || ', ' || t.nombres 
		FROM contratos c
		INNER JOIN trabajadores t ON c.trabajador_id = t.id
		WHERE c.tenant_id = $1 AND c.activo = true
		ORDER BY t.apellido_paterno ASC
	`
	rows, err := r.db.Query(query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.ContratoSelect
	for rows.Next() {
		var c models.ContratoSelect
		rows.Scan(&c.ID, &c.NumeroDocumento, &c.TrabajadorNombre)
		lista = append(lista, c)
	}
	return lista, nil
}

// Crear inserta la nueva falta/tardanza en estado "Pendiente" (procesado = false)
func (r *AsistenciaRepository) Crear(contratoID int, tipo string, fecha string, cantidad float64) error {
	query := `
		INSERT INTO ocurrencias_asistencia (contrato_id, tipo, fecha_ocurrencia, cantidad, procesado) 
		VALUES ($1, $2, $3, $4, false)
	`
	_, err := r.db.Exec(query, contratoID, tipo, fecha, cantidad)
	return err
}

// ListarHistorial trae todas las ocurrencias, mostrando primero las pendientes
func (r *AsistenciaRepository) ListarHistorial(tenantID int) ([]models.OcurrenciaVista, error) {
	query := `
		SELECT o.id, o.contrato_id, t.apellido_paterno || ' ' || t.apellido_materno || ', ' || t.nombres,
		       o.tipo, TO_CHAR(o.fecha_ocurrencia, 'DD/MM/YYYY'), o.cantidad, o.procesado
		FROM ocurrencias_asistencia o
		INNER JOIN contratos c ON o.contrato_id = c.id
		INNER JOIN trabajadores t ON c.trabajador_id = t.id
		WHERE c.tenant_id = $1
		ORDER BY o.procesado ASC, o.fecha_ocurrencia DESC
	`
	rows, err := r.db.Query(query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.OcurrenciaVista
	for rows.Next() {
		var o models.OcurrenciaVista
		rows.Scan(&o.ID, &o.ContratoID, &o.TrabajadorNombre, &o.Tipo, &o.FechaOcurrencia, &o.Cantidad, &o.Procesado)
		lista = append(lista, o)
	}
	return lista, nil
}

// ObtenerContratoPorDNI busca el contrato activo usando el DNI del trabajador
func (r *AsistenciaRepository) ObtenerContratoPorDNI(tenantID int, dni string) (int, error) {
	var contratoID int
	query := `
		SELECT c.id 
		FROM contratos c
		INNER JOIN trabajadores t ON c.trabajador_id = t.id
		WHERE t.numero_documento = $1 AND c.tenant_id = $2 AND c.activo = true
		LIMIT 1
	`
	// Usamos strings.TrimSpace por si el Excel trae espacios en blanco ocultos
	err := r.db.QueryRow(query, strings.TrimSpace(dni), tenantID).Scan(&contratoID)
	return contratoID, err
}

func (r *AsistenciaRepository) ObtenerPorID(id int, tenantID int) (models.OcurrenciaVista, error) {
	var o models.OcurrenciaVista
	query := `
		SELECT o.id, o.contrato_id, t.apellido_paterno || ' ' || t.apellido_materno || ', ' || t.nombres,
		       o.tipo, TO_CHAR(o.fecha_ocurrencia, 'YYYY-MM-DD'), o.cantidad, o.procesado
		FROM ocurrencias_asistencia o
		INNER JOIN contratos c ON o.contrato_id = c.id
		INNER JOIN trabajadores t ON c.trabajador_id = t.id
		WHERE o.id = $1 AND c.tenant_id = $2
	`
	err := r.db.QueryRow(query, id, tenantID).Scan(
		&o.ID, &o.ContratoID, &o.TrabajadorNombre, &o.Tipo, &o.FechaOcurrencia, &o.Cantidad, &o.Procesado,
	)
	return o, err
}

func (r *AsistenciaRepository) Actualizar(id int, tipo string, fecha string, cantidad float64, tenantID int) error {
	// Solo permitimos editar si procesado = false
	query := `
		UPDATE ocurrencias_asistencia 
		SET tipo = $1, fecha_ocurrencia = $2, cantidad = $3 
		WHERE id = $4 AND procesado = false 
		AND contrato_id IN (SELECT id FROM contratos WHERE tenant_id = $5)
	`
	_, err := r.db.Exec(query, tipo, fecha, cantidad, id, tenantID)
	return err
}

func (r *AsistenciaRepository) ListarPaginado(tenantID int, buscar string, tipo string, procesado string, limite int, offset int) ([]models.OcurrenciaVista, int, error) {

	whereClause := "WHERE c.tenant_id = $1"
	args := []interface{}{tenantID}
	paramIndex := 2

	if buscar != "" {
		buscaParam := "%" + strings.ToLower(buscar) + "%"
		whereClause += fmt.Sprintf(` AND (LOWER(t.numero_documento) LIKE $%d OR LOWER(t.nombres || ' ' || t.apellido_paterno || ' ' || t.apellido_materno) LIKE $%d)`, paramIndex, paramIndex+1)
		args = append(args, buscaParam, buscaParam)
		paramIndex += 2
	}

	if tipo != "" {
		whereClause += fmt.Sprintf(" AND o.tipo = $%d", paramIndex)
		args = append(args, strings.ToUpper(tipo))
		paramIndex++
	}

	// procesado puede llegar con un valor vacío "", "false", "true". Debemos convertir "false" o "true" en booleanos
	// antes de agregarlos a los argumentos. Por ejemplo, si procesado es "false", lo convertimos a false.
	var procesadoBool bool
	if procesado != "" {
		if procesado == "true" {
			procesadoBool = true
		} else {
			procesadoBool = false
		}

		whereClause += fmt.Sprintf(" AND o.procesado = $%d", paramIndex)
		args = append(args, procesadoBool)
		paramIndex++
	}

	// 1. Contar registros totales
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM ocurrencias_asistencia o
		INNER JOIN contratos c ON o.contrato_id = c.id
		INNER JOIN trabajadores t ON c.trabajador_id = t.id
		%s
	`, whereClause)

	var totalRegistros int
	err := r.db.QueryRow(countQuery, args...).Scan(&totalRegistros)
	if err != nil {
		return nil, 0, err
	}

	// 2. Obtener lista de la página actual
	queryLista := fmt.Sprintf(`
		SELECT o.id, o.contrato_id, t.apellido_paterno || ' ' || t.apellido_materno || ', ' || t.nombres,
		       o.tipo, TO_CHAR(o.fecha_ocurrencia, 'DD/MM/YYYY'), o.cantidad, o.procesado
		FROM ocurrencias_asistencia o
		INNER JOIN contratos c ON o.contrato_id = c.id
		INNER JOIN trabajadores t ON c.trabajador_id = t.id
		%s
		ORDER BY o.procesado ASC, o.fecha_ocurrencia DESC 
		LIMIT $%d OFFSET $%d
	`, whereClause, paramIndex, paramIndex+1)

	argsLista := append(args, limite, offset)

	rows, err := r.db.Query(queryLista, argsLista...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var lista []models.OcurrenciaVista
	for rows.Next() {
		var o models.OcurrenciaVista
		err := rows.Scan(&o.ID, &o.ContratoID, &o.TrabajadorNombre, &o.Tipo, &o.FechaOcurrencia, &o.Cantidad, &o.Procesado)
		if err != nil {
			return nil, 0, err
		}
		lista = append(lista, o)
	}

	for i := range lista {
		lista[i].FechaOcurrencia = strings.TrimSpace(lista[i].FechaOcurrencia)
	}

	return lista, totalRegistros, nil
}
