package repository

import (
	"database/sql"
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
