package repository

import (
	"database/sql"
	"fmt"
	"planilla-rgm/internal/models"
	"strings"
)

type LicenciaVacacionRepository struct {
	db *sql.DB
}

func NewLicenciaVacacionRepository(db *sql.DB) *LicenciaVacacionRepository {
	return &LicenciaVacacionRepository{db: db}
}

// ObtenerTiposSuspensionSunat retorna el catálogo oficial de la Tabla 21 de SUNAT
func (r *LicenciaVacacionRepository) ObtenerTiposSuspensionSunat() ([]models.SunatTipoSuspension, error) {
	query := `
		SELECT id, codigo, descripcion, descripcion_abreviada, tipo_suspension, activo, created_at
		FROM sunat_tipos_suspension
		WHERE activo = true
		ORDER BY 
			CASE tipo_suspension WHEN 'IMPERFECTA' THEN 1 ELSE 2 END,
			codigo ASC
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error consultando catálogo SUNAT Tabla 21: %w", err)
	}
	defer rows.Close()

	var lista []models.SunatTipoSuspension
	for rows.Next() {
		var s models.SunatTipoSuspension
		err := rows.Scan(
			&s.ID,
			&s.Codigo,
			&s.Descripcion,
			&s.DescripcionAbreviada,
			&s.TipoSuspension,
			&s.Activo,
			&s.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		lista = append(lista, s)
	}
	return lista, nil
}

// Listar obtiene las vacaciones y licencias registradas aplicando filtros
func (r *LicenciaVacacionRepository) Listar(tenantID int, buscar string, tipo string, estado string, anio int, mes int) ([]models.LicenciaVacacionVista, error) {
	var condiciones []string
	var args []interface{}
	idx := 1

	condiciones = append(condiciones, fmt.Sprintf("lv.tenant_id = $%d", idx))
	args = append(args, tenantID)
	idx++

	if buscar != "" {
		patron := "%" + strings.ToLower(buscar) + "%"
		condiciones = append(condiciones, fmt.Sprintf(`(
			LOWER(t.apellido_paterno || ' ' || t.apellido_materno || ', ' || t.nombres) LIKE $%d OR
			t.numero_documento LIKE $%d OR
			LOWER(lv.documento_aprobacion) LIKE $%d
		)`, idx, idx, idx))
		args = append(args, patron)
		idx++
	}

	if tipo != "" && tipo != "TODOS" {
		condiciones = append(condiciones, fmt.Sprintf("lv.tipo = $%d", idx))
		args = append(args, tipo)
		idx++
	}

	if estado != "" && estado != "TODOS" {
		condiciones = append(condiciones, fmt.Sprintf("lv.estado = $%d", idx))
		args = append(args, estado)
		idx++
	}

	if anio > 0 && mes > 0 {
		condiciones = append(condiciones, fmt.Sprintf(`(
			lv.fecha_inicio <= (make_date($%d, $%d, 1) + interval '1 month' - interval '1 day')::date AND
			lv.fecha_fin >= make_date($%d, $%d, 1)
		)`, idx, idx+1, idx, idx+1))
		args = append(args, anio, mes)
		idx += 2
	}

	whereClause := strings.Join(condiciones, " AND ")

	query := fmt.Sprintf(`
		SELECT 
			lv.id, lv.tenant_id, lv.trabajador_id, lv.contrato_id,
			TRIM(t.apellido_paterno || ' ' || t.apellido_materno || ', ' || t.nombres) AS trabajador_nombre,
			t.numero_documento AS trabajador_doc,
			COALESCE(p.nombre, 'Sin Plaza Asignada') AS puesto_nombre,
			COALESCE(rl.codigo, '-') AS regimen_codigo,
			lv.tipo, COALESCE(lv.subtipo, '') AS subtipo,
			lv.codigo_sunat_suspension,
			COALESCE(st.descripcion_abreviada, '') AS sunat_descripcion_abrev,
			COALESCE(st.tipo_suspension, '') AS sunat_tipo_suspension,
			TO_CHAR(lv.fecha_inicio, 'DD/MM/YYYY') AS fecha_inicio,
			TO_CHAR(lv.fecha_fin, 'DD/MM/YYYY') AS fecha_fin,
			lv.dias_calendario,
			lv.documento_aprobacion,
			TO_CHAR(lv.fecha_aprobacion, 'DD/MM/YYYY') AS fecha_aprobacion,
			COALESCE(lv.observaciones, '') AS observaciones,
			lv.estado
		FROM personal_licencias_vacaciones lv
		INNER JOIN trabajadores t ON lv.trabajador_id = t.id
		LEFT JOIN contratos c ON lv.contrato_id = c.id
		LEFT JOIN puestos p ON c.puesto_id = p.id
		LEFT JOIN regimenes_laborales rl ON p.regimen_id = rl.id
		LEFT JOIN sunat_tipos_suspension st ON lv.codigo_sunat_suspension = st.codigo
		WHERE %s
		ORDER BY lv.fecha_inicio DESC, lv.id DESC
	`, whereClause)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("error listando vacaciones y licencias: %w", err)
	}
	defer rows.Close()

	var lista []models.LicenciaVacacionVista
	for rows.Next() {
		var item models.LicenciaVacacionVista
		var cID sql.NullInt64
		var fAprob sql.NullString
		err := rows.Scan(
			&item.ID, &item.TenantID, &item.TrabajadorID, &cID,
			&item.TrabajadorNombre, &item.TrabajadorDoc, &item.PuestoNombre, &item.RegimenCodigo,
			&item.Tipo, &item.Subtipo,
			&item.CodigoSunatSuspension, &item.SunatDescripcionAbrev, &item.SunatTipoSuspension,
			&item.FechaInicio, &item.FechaFin, &item.DiasCalendario,
			&item.DocumentoAprobacion, &fAprob, &item.Observaciones, &item.Estado,
		)
		if err != nil {
			return nil, err
		}
		if cID.Valid {
			v := int(cID.Int64)
			item.ContratoID = &v
		}
		if fAprob.Valid && fAprob.String != "" {
			item.FechaAprobacion = &fAprob.String
		}
		lista = append(lista, item)
	}
	return lista, nil
}

// ObtenerPorID retorna un registro específico
func (r *LicenciaVacacionRepository) ObtenerPorID(id int, tenantID int) (*models.LicenciaVacacion, error) {
	query := `
		SELECT 
			id, tenant_id, trabajador_id, contrato_id, tipo, COALESCE(subtipo, ''),
			codigo_sunat_suspension, 
			TO_CHAR(fecha_inicio, 'YYYY-MM-DD'), 
			TO_CHAR(fecha_fin, 'YYYY-MM-DD'),
			dias_calendario, documento_aprobacion,
			TO_CHAR(fecha_aprobacion, 'YYYY-MM-DD'),
			COALESCE(observaciones, ''), estado, created_at, updated_at
		FROM personal_licencias_vacaciones
		WHERE id = $1 AND tenant_id = $2
	`
	var lv models.LicenciaVacacion
	var cID sql.NullInt64
	var fAprob sql.NullString

	err := r.db.QueryRow(query, id, tenantID).Scan(
		&lv.ID, &lv.TenantID, &lv.TrabajadorID, &cID, &lv.Tipo, &lv.Subtipo,
		&lv.CodigoSunatSuspension, &lv.FechaInicio, &lv.FechaFin,
		&lv.DiasCalendario, &lv.DocumentoAprobacion, &fAprob,
		&lv.Observaciones, &lv.Estado, &lv.CreatedAt, &lv.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if cID.Valid {
		v := int(cID.Int64)
		lv.ContratoID = &v
	}
	if fAprob.Valid && fAprob.String != "" {
		lv.FechaAprobacion = &fAprob.String
	}
	return &lv, nil
}

// ValidarSolapamiento verifica si el trabajador ya tiene una vacación o licencia activa que se solape con el rango propuesto
func (r *LicenciaVacacionRepository) ValidarSolapamiento(tenantID int, trabajadorID int, fechaInicio, fechaFin string, excluirID int) (bool, error) {
	query := `
		SELECT COUNT(*) 
		FROM personal_licencias_vacaciones
		WHERE tenant_id = $1 
		  AND trabajador_id = $2 
		  AND estado != 'CANCELADO'
		  AND id != $3
		  AND fecha_inicio <= $4::date
		  AND fecha_fin >= $5::date
	`
	var count int
	err := r.db.QueryRow(query, tenantID, trabajadorID, excluirID, fechaFin, fechaInicio).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Crear inserta una nueva vacación o licencia
func (r *LicenciaVacacionRepository) Crear(item *models.LicenciaVacacion) error {
	query := `
		INSERT INTO personal_licencias_vacaciones (
			tenant_id, trabajador_id, contrato_id, tipo, subtipo,
			codigo_sunat_suspension, fecha_inicio, fecha_fin,
			documento_aprobacion, fecha_aprobacion, observaciones, estado
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7::date, $8::date,
			$9, NULLIF($10, '')::date, $11, $12
		) RETURNING id, dias_calendario
	`
	var fAprobVal interface{}
	if item.FechaAprobacion != nil && *item.FechaAprobacion != "" {
		fAprobVal = *item.FechaAprobacion
	} else {
		fAprobVal = ""
	}

	return r.db.QueryRow(
		query,
		item.TenantID, item.TrabajadorID, item.ContratoID, item.Tipo, item.Subtipo,
		item.CodigoSunatSuspension, item.FechaInicio, item.FechaFin,
		item.DocumentoAprobacion, fAprobVal, item.Observaciones, item.Estado,
	).Scan(&item.ID, &item.DiasCalendario)
}

// Actualizar modifica un registro existente
func (r *LicenciaVacacionRepository) Actualizar(item *models.LicenciaVacacion) error {
	query := `
		UPDATE personal_licencias_vacaciones
		SET tipo = $1,
		    subtipo = $2,
		    codigo_sunat_suspension = $3,
		    fecha_inicio = $4::date,
		    fecha_fin = $5::date,
		    documento_aprobacion = $6,
		    fecha_aprobacion = NULLIF($7, '')::date,
		    observaciones = $8,
		    estado = $9,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $10 AND tenant_id = $11
		RETURNING dias_calendario
	`
	var fAprobVal interface{}
	if item.FechaAprobacion != nil && *item.FechaAprobacion != "" {
		fAprobVal = *item.FechaAprobacion
	} else {
		fAprobVal = ""
	}

	return r.db.QueryRow(
		query,
		item.Tipo, item.Subtipo, item.CodigoSunatSuspension,
		item.FechaInicio, item.FechaFin,
		item.DocumentoAprobacion, fAprobVal, item.Observaciones,
		item.Estado, item.ID, item.TenantID,
	).Scan(&item.DiasCalendario)
}

// Eliminar elimina lógicamente o físicamente un registro
func (r *LicenciaVacacionRepository) Eliminar(id int, tenantID int) error {
	query := `DELETE FROM personal_licencias_vacaciones WHERE id = $1 AND tenant_id = $2`
	res, err := r.db.Exec(query, id, tenantID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("registro no encontrado")
	}
	return nil
}

// ObtenerKPIs calcula las tarjetas Bento para el encabezado de la vista
func (r *LicenciaVacacionRepository) ObtenerKPIs(tenantID int, anio int, mes int) (*models.KpisLicenciaVacacion, error) {
	query := `
		SELECT 
			COALESCE(COUNT(*) FILTER (WHERE lv.tipo = 'VACACION' AND lv.fecha_inicio <= (make_date($2, $3, 1) + interval '1 month' - interval '1 day')::date AND lv.fecha_fin >= make_date($2, $3, 1) AND lv.estado != 'CANCELADO'), 0) AS vac_mes,
			COALESCE(COUNT(*) FILTER (WHERE lv.tipo = 'LICENCIA_CON_GOCE' AND lv.fecha_inicio <= (make_date($2, $3, 1) + interval '1 month' - interval '1 day')::date AND lv.fecha_fin >= make_date($2, $3, 1) AND lv.estado != 'CANCELADO'), 0) AS lic_cg_mes,
			COALESCE(COUNT(*) FILTER (WHERE lv.tipo = 'LICENCIA_SIN_GOCE' AND lv.fecha_inicio <= (make_date($2, $3, 1) + interval '1 month' - interval '1 day')::date AND lv.fecha_fin >= make_date($2, $3, 1) AND lv.estado != 'CANCELADO'), 0) AS lic_sg_mes,
			COALESCE(COUNT(*), 0) AS total_historico
		FROM personal_licencias_vacaciones lv
		WHERE lv.tenant_id = $1
	`
	var k models.KpisLicenciaVacacion
	err := r.db.QueryRow(query, tenantID, anio, mes).Scan(
		&k.TotalEnVacacionesMes,
		&k.TotalLicenciasConGoceMes,
		&k.TotalLicenciasSinGoceMes,
		&k.TotalHistorico,
	)
	if err != nil {
		return nil, fmt.Errorf("error calculando KPIs de licencias y vacaciones: %w", err)
	}
	return &k, nil
}

// ObtenerIncidenciasMes obtiene todas las incidencias de vacaciones/licencias que solapan con un mes específico
// Retorna un mapa indexado por trabajador_id con su lista de incidencias calculadas
func (r *LicenciaVacacionRepository) ObtenerIncidenciasMes(tenantID int, anio int, mes int) (map[int][]models.PersonalIncidenciaMes, error) {
	query := `
		SELECT 
			lv.id, lv.trabajador_id, COALESCE(c.id, 0) AS contrato_id,
			TRIM(t.apellido_paterno || ' ' || t.apellido_materno || ', ' || t.nombres) AS trabajador_nombre,
			t.numero_documento AS trabajador_doc,
			COALESCE(p.nombre, 'Sin Plaza') AS puesto_nombre,
			COALESCE(rl.codigo, '-') AS regimen_codigo,
			lv.tipo, COALESCE(lv.subtipo, '') AS subtipo,
			lv.codigo_sunat_suspension,
			TO_CHAR(lv.fecha_inicio, 'DD/MM/YYYY') AS fecha_inicio,
			TO_CHAR(lv.fecha_fin, 'DD/MM/YYYY') AS fecha_fin,
			(LEAST(lv.fecha_fin, (make_date($2, $3, 1) + interval '1 month' - interval '1 day')::date) - GREATEST(lv.fecha_inicio, make_date($2, $3, 1)) + 1) AS dias_en_mes,
			lv.documento_aprobacion,
			COALESCE(lv.observaciones, '') AS observaciones
		FROM personal_licencias_vacaciones lv
		INNER JOIN trabajadores t ON lv.trabajador_id = t.id
		LEFT JOIN contratos c ON lv.contrato_id = c.id
		LEFT JOIN puestos p ON c.puesto_id = p.id
		LEFT JOIN regimenes_laborales rl ON p.regimen_id = rl.id
		WHERE lv.tenant_id = $1
		  AND lv.estado != 'CANCELADO'
		  AND lv.fecha_inicio <= (make_date($2, $3, 1) + interval '1 month' - interval '1 day')::date
		  AND lv.fecha_fin >= make_date($2, $3, 1)
		ORDER BY lv.fecha_inicio ASC
	`
	rows, err := r.db.Query(query, tenantID, anio, mes)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo incidencias de personal para el mes %d/%d: %w", mes, anio, err)
	}
	defer rows.Close()

	mapa := make(map[int][]models.PersonalIncidenciaMes)
	for rows.Next() {
		var inc models.PersonalIncidenciaMes
		err := rows.Scan(
			&inc.IncidenciaID, &inc.TrabajadorID, &inc.ContratoID,
			&inc.TrabajadorNombre, &inc.TrabajadorDoc, &inc.PuestoNombre, &inc.RegimenCodigo,
			&inc.Tipo, &inc.Subtipo, &inc.CodigoSunatSuspension,
			&inc.FechaInicio, &inc.FechaFin, &inc.DiasEnMes,
			&inc.DocumentoAprobacion, &inc.Observaciones,
		)
		if err != nil {
			return nil, err
		}
		mapa[inc.TrabajadorID] = append(mapa[inc.TrabajadorID], inc)
	}
	return mapa, nil
}

// ObtenerContratosActivosSelect trae la lista de trabajadores con contrato activo para el selector TomSelect
func (r *LicenciaVacacionRepository) ObtenerContratosActivosSelect(tenantID int) ([]models.ContratoSelect, error) {
	query := `
		SELECT c.id, t.numero_documento, 
		       TRIM(t.apellido_paterno || ' ' || t.apellido_materno || ', ' || t.nombres) || ' - ' || COALESCE(p.nombre, 'Sin Plaza') || ' (' || COALESCE(rl.codigo, '-') || ')'
		FROM contratos c
		INNER JOIN trabajadores t ON c.trabajador_id = t.id
		LEFT JOIN puestos p ON c.puesto_id = p.id
		LEFT JOIN regimenes_laborales rl ON p.regimen_id = rl.id
		WHERE c.tenant_id = $1 AND c.activo = true
		ORDER BY t.apellido_paterno ASC, t.apellido_materno ASC, t.nombres ASC
	`
	rows, err := r.db.Query(query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.ContratoSelect
	for rows.Next() {
		var c models.ContratoSelect
		if err := rows.Scan(&c.ID, &c.NumeroDocumento, &c.TrabajadorNombre); err != nil {
			return nil, err
		}
		lista = append(lista, c)
	}
	return lista, nil
}

// ObtenerTrabajadorYContratoID retorna el trabajador_id correspondiente a un contrato_id
func (r *LicenciaVacacionRepository) ObtenerTrabajadorYContratoID(tenantID int, contratoID int) (int, error) {
	var trabajadorID int
	query := `SELECT trabajador_id FROM contratos WHERE id = $1 AND tenant_id = $2`
	err := r.db.QueryRow(query, contratoID, tenantID).Scan(&trabajadorID)
	return trabajadorID, err
}
