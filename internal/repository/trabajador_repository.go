package repository

import (
	"database/sql"
	"fmt"
	"planilla-rgm/internal/models"
	"strings"
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return afps, nil
}

// ObtenerMapaAFPsParaImportar carga todas las AFPs activas y mapea sus nombres y códigos SBS (en mayúsculas) a su ID de BD
func (r *TrabajadorRepository) ObtenerMapaAFPsParaImportar() (map[string]int, error) {
	query := `SELECT id, codigo_sbs, nombre FROM afps WHERE activo = true`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	mapa := make(map[string]int)
	for rows.Next() {
		var id int
		var codigoSBS sql.NullString
		var nombre string
		if err := rows.Scan(&id, &codigoSBS, &nombre); err != nil {
			return nil, err
		}
		mapa[strings.ToUpper(strings.TrimSpace(nombre))] = id
		if codigoSBS.Valid && codigoSBS.String != "" {
			mapa[strings.ToUpper(strings.TrimSpace(codigoSBS.String))] = id
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return mapa, nil
}

// Obtener todos los trabajadores de un tenant (sin paginación)
func (r *TrabajadorRepository) ObtenerTodos(tenantID int) ([]models.Trabajador, error) {
	query := `
		SELECT id, tenant_id, tipo_documento, numero_documento, nombres, apellido_paterno, apellido_materno, 
		       TO_CHAR(fecha_nacimiento, 'YYYY-MM-DD'), TO_CHAR(fecha_ingreso, 'YYYY-MM-DD'), sexo, activo,
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
		var fechaNac, fechaIng sql.NullString
		err := rows.Scan(&t.ID, &t.TenantID, &t.TipoDocumento, &t.NumeroDocumento, &t.Nombres, &t.ApellidoPaterno, &t.ApellidoMaterno, &fechaNac, &fechaIng, &t.Sexo, &t.Activo, &t.RegimenPensionario, &t.AfpID, &t.AfpTipoComision, &t.Cuspp)
		if err != nil {
			return nil, err
		}
		if fechaNac.Valid {
			t.FechaNacimiento = fechaNac.String
		}
		if fechaIng.Valid {
			t.FechaIngreso = fechaIng.String
		}
		lista = append(lista, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
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

	query := fmt.Sprintf(`
		SELECT id, tenant_id, tipo_documento, numero_documento, nombres, apellido_paterno, apellido_materno, 
		       TO_CHAR(fecha_nacimiento, 'YYYY-MM-DD'), TO_CHAR(fecha_ingreso, 'YYYY-MM-DD'), sexo, activo,
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
		var fechaNac, fechaIng sql.NullString
		err := rows.Scan(&t.ID, &t.TenantID, &t.TipoDocumento, &t.NumeroDocumento, &t.Nombres, &t.ApellidoPaterno, &t.ApellidoMaterno, &fechaNac, &fechaIng, &t.Sexo, &t.Activo, &t.RegimenPensionario, &t.AfpID, &t.AfpTipoComision, &t.Cuspp)
		if err != nil {
			return nil, 0, err
		}
		if fechaNac.Valid {
			t.FechaNacimiento = fechaNac.String
		}
		if fechaIng.Valid {
			t.FechaIngreso = fechaIng.String
		}
		lista = append(lista, t)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return lista, totalRegistros, nil
}

// Obtener trabajador por ID y tenantID
func (r *TrabajadorRepository) ObtenerPorID(id int, tenantID int) (*models.Trabajador, error) {
	var t models.Trabajador
	var fechaNac, fechaIng sql.NullString

	query := `
		SELECT id, tenant_id, tipo_documento, numero_documento, nombres, apellido_paterno, apellido_materno, 
		       TO_CHAR(fecha_nacimiento, 'YYYY-MM-DD'), TO_CHAR(fecha_ingreso, 'YYYY-MM-DD'), sexo, activo,
		       COALESCE(regimen_pensionario, 'ONP'), COALESCE(afp_id, 0), COALESCE(afp_tipo_comision, ''), COALESCE(cuspp, '')
		FROM trabajadores 
		WHERE id = $1 AND tenant_id = $2
	`
	err := r.db.QueryRow(query, id, tenantID).Scan(&t.ID, &t.TenantID, &t.TipoDocumento, &t.NumeroDocumento, &t.Nombres, &t.ApellidoPaterno, &t.ApellidoMaterno, &fechaNac, &fechaIng, &t.Sexo, &t.Activo, &t.RegimenPensionario, &t.AfpID, &t.AfpTipoComision, &t.Cuspp)
	if err != nil {
		return nil, err
	}
	if fechaNac.Valid {
		t.FechaNacimiento = fechaNac.String
	}
	if fechaIng.Valid {
		t.FechaIngreso = fechaIng.String
	}
	return &t, nil
}

// Crear inserta un trabajador forzando el tenant_id
func (r *TrabajadorRepository) Crear(t *models.Trabajador) error {
	query := `
		INSERT INTO trabajadores (tenant_id, tipo_documento, numero_documento, nombres, apellido_paterno, apellido_materno, fecha_nacimiento, fecha_ingreso, sexo, activo, regimen_pensionario, afp_id, afp_tipo_comision, cuspp) 
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, '')::DATE, NULLIF($8, '')::DATE, $9, $10, $11, NULLIF($12, 0), $13, $14) RETURNING id
	`
	return r.db.QueryRow(query, t.TenantID, t.TipoDocumento, t.NumeroDocumento, t.Nombres, t.ApellidoPaterno, t.ApellidoMaterno, t.FechaNacimiento, t.FechaIngreso, t.Sexo, t.Activo, t.RegimenPensionario, t.AfpID, t.AfpTipoComision, t.Cuspp).Scan(&t.ID)
}

// Actualizar guarda los cambios asegurando la propiedad (tenant_id)
func (r *TrabajadorRepository) Actualizar(t *models.Trabajador) error {
	query := `
		UPDATE trabajadores 
		SET tipo_documento = $1, numero_documento = $2, nombres = $3, apellido_paterno = $4, 
		    apellido_materno = $5, fecha_nacimiento = NULLIF($6, '')::DATE, fecha_ingreso = NULLIF($7, '')::DATE, 
		    sexo = $8, activo = $9, regimen_pensionario = $10, afp_id = NULLIF($11, 0), 
		    afp_tipo_comision = $12, cuspp = $13, updated_at = CURRENT_TIMESTAMP 
		WHERE id = $14 AND tenant_id = $15
	`
	_, err := r.db.Exec(query, t.TipoDocumento, t.NumeroDocumento, t.Nombres, t.ApellidoPaterno, t.ApellidoMaterno, t.FechaNacimiento, t.FechaIngreso, t.Sexo, t.Activo, t.RegimenPensionario, t.AfpID, t.AfpTipoComision, t.Cuspp, t.ID, t.TenantID)
	return err
}

// ExisteDocumento verifica si ya existe un trabajador con el mismo tipo y número de documento para el tenant
func (r *TrabajadorRepository) ExisteDocumento(tenantID int, tipoDoc string, numDoc string) (bool, error) {
	var existe bool
	query := `SELECT EXISTS(SELECT 1 FROM trabajadores WHERE tenant_id = $1 AND tipo_documento = $2 AND numero_documento = $3)`
	err := r.db.QueryRow(query, tenantID, tipoDoc, numDoc).Scan(&existe)
	return existe, err
}

// ImportarTrabajadores inserta de manera atómica (transaccional) una lista de trabajadores
func (r *TrabajadorRepository) ImportarTrabajadores(tenantID int, trabajadores []models.Trabajador) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO trabajadores (
			tenant_id, tipo_documento, numero_documento, nombres, 
			apellido_paterno, apellido_materno, fecha_nacimiento, fecha_ingreso, sexo, 
			activo, regimen_pensionario, afp_id, afp_tipo_comision, cuspp
		) VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, '')::DATE, NULLIF($8, '')::DATE, $9, $10, $11, NULLIF($12, 0), $13, $14)
	`
	stmt, err := tx.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, t := range trabajadores {
		_, err = stmt.Exec(
			tenantID, t.TipoDocumento, t.NumeroDocumento, t.Nombres,
			t.ApellidoPaterno, t.ApellidoMaterno, t.FechaNacimiento, t.FechaIngreso, t.Sexo,
			t.Activo, t.RegimenPensionario, t.AfpID, t.AfpTipoComision, t.Cuspp,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// ObtenerCumpleaniosMes obtiene los trabajadores que cumplen años en el mes indicado
func (r *TrabajadorRepository) ObtenerCumpleaniosMes(tenantID int, mes int) ([]models.Trabajador, error) {
	query := `
		SELECT id, tenant_id, tipo_documento, numero_documento, nombres, apellido_paterno, apellido_materno, 
		       TO_CHAR(fecha_nacimiento, 'YYYY-MM-DD'), TO_CHAR(fecha_ingreso, 'YYYY-MM-DD'), sexo, activo,
		       COALESCE(regimen_pensionario, 'ONP'), COALESCE(afp_id, 0), COALESCE(afp_tipo_comision, ''), COALESCE(cuspp, '')
		FROM trabajadores 
		WHERE tenant_id = $1 AND EXTRACT(MONTH FROM fecha_nacimiento) = $2 AND activo = true
		ORDER BY EXTRACT(DAY FROM fecha_nacimiento) ASC, apellido_paterno ASC
	`
	rows, err := r.db.Query(query, tenantID, mes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.Trabajador
	for rows.Next() {
		var t models.Trabajador
		var fechaNac, fechaIng sql.NullString
		err := rows.Scan(&t.ID, &t.TenantID, &t.TipoDocumento, &t.NumeroDocumento, &t.Nombres, &t.ApellidoPaterno, &t.ApellidoMaterno, &fechaNac, &fechaIng, &t.Sexo, &t.Activo, &t.RegimenPensionario, &t.AfpID, &t.AfpTipoComision, &t.Cuspp)
		if err != nil {
			return nil, err
		}
		if fechaNac.Valid {
			t.FechaNacimiento = fechaNac.String
		}
		if fechaIng.Valid {
			t.FechaIngreso = fechaIng.String
		}
		lista = append(lista, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return lista, nil
}
