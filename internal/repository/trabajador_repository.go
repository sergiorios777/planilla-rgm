package repository

import (
	"database/sql"
	"fmt"
	"log"
	"planilla-rgm/internal/models"
)

type TrabajadorRepository struct {
	db *sql.DB
}

func NewTrabajadorRepository(db *sql.DB) *TrabajadorRepository {
	return &TrabajadorRepository{db: db}
}

// ObtenerAFPsActivas trae el catálogo para llenar los <select> del formulario
func (r *TrabajadorRepository) ObtenerAFPsActivas() (map[int]string, error) {
	query := `SELECT id, nombre FROM afps WHERE activo = true ORDER BY nombre`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	afps := make(map[int]string)
	for rows.Next() {
		var id int
		var nombre string
		rows.Scan(&id, &nombre)
		afps[id] = nombre
	}
	return afps, nil
}

// Obtener todos los trabajadores de un tenant (sin paginación)
func (r *TrabajadorRepository) ObtenerTodos(tenantID int) ([]models.Trabajador, error) {
	query := `
		SELECT id, tenant_id, tipo_documento, numero_documento, nombres, apellido_paterno, apellido_materno, 
		       TO_CHAR(fecha_nacimiento, 'YYYY-MM-DD'), sexo, activo,
		       COALESCE(regimen_pensionario, 'ONP'), COALESCE(afp_id, 0), COALESCE(afp_tipo_comision, ''), COALESCE(cuspp, '')
		FROM trabajadores 
		WHERE tenant_id = $1
		ORDER BY apellido_paterno, apellido_materno ASC
	`

	rows, err := r.db.Query(query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.Trabajador
	for rows.Next() {
		var t models.Trabajador
		var fecha sql.NullString
		err := rows.Scan(&t.ID, &t.TenantID, &t.TipoDocumento, &t.NumeroDocumento, &t.Nombres, &t.ApellidoPaterno, &t.ApellidoMaterno, &fecha, &t.Sexo, &t.Activo, &t.RegimenPensionario, &t.AfpID, &t.AfpTipoComision, &t.Cuspp)
		if err != nil {
			return nil, err
		}
		if fecha.Valid {
			t.FechaNacimiento = fecha.String
		}
		lista = append(lista, t)
	}
	return lista, nil
}

// Obtener todos los trabajadores de un tenant paginado
func (r *TrabajadorRepository) ObtenerTodosPaginacion(tenantID int, busqueda string, limite int, offset int) ([]models.Trabajador, int, error) {
	whereClause := `WHERE tenant_id = $1`

	args := []interface{}{tenantID}
	contadorArgs := 2

	if busqueda != "" {
		whereClause += fmt.Sprintf(` AND (numero_documento ILIKE $%d OR nombres || ' ' || apellido_paterno || ' ' || apellido_materno ILIKE $%d)`, contadorArgs, contadorArgs)
		args = append(args, "%"+busqueda+"%")
		contadorArgs++
	}

	var totalRegistros int
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM trabajadores %s`, whereClause)
	err := r.db.QueryRow(countQuery, args...).Scan(&totalRegistros)
	if err != nil {
		return nil, 0, err
	}
	log.Println("Total registros trabajadores: ", totalRegistros)

	query := fmt.Sprintf(`
		SELECT id, tenant_id, tipo_documento, numero_documento, nombres, apellido_paterno, apellido_materno, 
		       TO_CHAR(fecha_nacimiento, 'YYYY-MM-DD'), sexo, activo,
		       COALESCE(regimen_pensionario, 'ONP'), COALESCE(afp_id, 0), COALESCE(afp_tipo_comision, ''), COALESCE(cuspp, '')
		FROM trabajadores 
		%s
		ORDER BY apellido_paterno, apellido_materno ASC
		LIMIT $%d OFFSET $%d
	`, whereClause, contadorArgs, contadorArgs+1)

	args = append(args, limite, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var lista []models.Trabajador
	for rows.Next() {
		var t models.Trabajador
		var fecha sql.NullString
		err := rows.Scan(&t.ID, &t.TenantID, &t.TipoDocumento, &t.NumeroDocumento, &t.Nombres, &t.ApellidoPaterno, &t.ApellidoMaterno, &fecha, &t.Sexo, &t.Activo, &t.RegimenPensionario, &t.AfpID, &t.AfpTipoComision, &t.Cuspp)
		if err != nil {
			return nil, 0, err
		}
		if fecha.Valid {
			t.FechaNacimiento = fecha.String
		}
		lista = append(lista, t)
	}
	return lista, totalRegistros, nil
}

// Obtener trabajador por ID y tenantID
func (r *TrabajadorRepository) ObtenerPorID(id int, tenantID int) (*models.Trabajador, error) {
	var t models.Trabajador
	var fecha sql.NullString

	query := `
		SELECT id, tenant_id, tipo_documento, numero_documento, nombres, apellido_paterno, apellido_materno, 
		       TO_CHAR(fecha_nacimiento, 'YYYY-MM-DD'), sexo, activo,
		       COALESCE(regimen_pensionario, 'ONP'), COALESCE(afp_id, 0), COALESCE(afp_tipo_comision, ''), COALESCE(cuspp, '')
		FROM trabajadores 
		WHERE id = $1 AND tenant_id = $2
	`
	err := r.db.QueryRow(query, id, tenantID).Scan(&t.ID, &t.TenantID, &t.TipoDocumento, &t.NumeroDocumento, &t.Nombres, &t.ApellidoPaterno, &t.ApellidoMaterno, &fecha, &t.Sexo, &t.Activo, &t.RegimenPensionario, &t.AfpID, &t.AfpTipoComision, &t.Cuspp)
	if err != nil {
		return nil, err
	}
	if fecha.Valid {
		t.FechaNacimiento = fecha.String
	}
	return &t, nil
}

// Crear inserta un trabajador forzando el tenant_id
func (r *TrabajadorRepository) Crear(t *models.Trabajador) error {
	query := `
		INSERT INTO trabajadores (tenant_id, tipo_documento, numero_documento, nombres, apellido_paterno, apellido_materno, fecha_nacimiento, sexo, activo, regimen_pensionario, afp_id, afp_tipo_comision, cuspp) 
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, '')::DATE, $8, $9, $10, NULLIF($11, 0), $12, $13) RETURNING id
	`
	return r.db.QueryRow(query, t.TenantID, t.TipoDocumento, t.NumeroDocumento, t.Nombres, t.ApellidoPaterno, t.ApellidoMaterno, t.FechaNacimiento, t.Sexo, t.Activo, t.RegimenPensionario, t.AfpID, t.AfpTipoComision, t.Cuspp).Scan(&t.ID)
}

// Actualizar guarda los cambios asegurando la propiedad (tenant_id)
func (r *TrabajadorRepository) Actualizar(t *models.Trabajador) error {
	query := `
		UPDATE trabajadores 
		SET tipo_documento = $1, numero_documento = $2, nombres = $3, apellido_paterno = $4, 
		    apellido_materno = $5, fecha_nacimiento = NULLIF($6, '')::DATE, sexo = $7, activo = $8, 
		    regimen_pensionario = $9, afp_id = NULLIF($10, 0), afp_tipo_comision = $11, cuspp = $12, updated_at = CURRENT_TIMESTAMP 
		WHERE id = $13 AND tenant_id = $14
	`
	_, err := r.db.Exec(query, t.TipoDocumento, t.NumeroDocumento, t.Nombres, t.ApellidoPaterno, t.ApellidoMaterno, t.FechaNacimiento, t.Sexo, t.Activo, t.RegimenPensionario, t.AfpID, t.AfpTipoComision, t.Cuspp, t.ID, t.TenantID)
	return err
}
