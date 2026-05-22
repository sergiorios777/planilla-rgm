package repository

import (
	"database/sql"
	"fmt"
	"log"
	"planilla-rgm/internal/models"
	"strings"

	"github.com/lib/pq"
)

type PuestoRepository struct {
	db *sql.DB
}

func NewPuestoRepository(db *sql.DB) *PuestoRepository {
	return &PuestoRepository{db: db}
}

// ObtenerVacantes lista solo los puestos que no están ocupados
func (r *PuestoRepository) ObtenerVacantes(tenantID int) ([]models.Puesto, error) {
	query := `
		SELECT p.id, p.nombre, p.sueldo_presupuestado, rl.descripcion,
		       COALESCE(u.nombre, 'Sin asignar') AS unidad_organica_nombre
		FROM puestos p
		INNER JOIN regimenes_laborales rl ON p.regimen_id = rl.id
		LEFT JOIN unidades_organicas u ON p.unidad_organica_id = u.id
		WHERE p.tenant_id = $1 AND p.estado = 'VACANTE' AND p.activo = true
		ORDER BY p.nombre ASC
	`
	rows, err := r.db.Query(query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.Puesto
	for rows.Next() {
		var p models.Puesto
		rows.Scan(&p.ID, &p.Nombre, &p.SueldoPresupuestado, &p.RegimenDesc, &p.UnidadOrganicaNombre)
		lista = append(lista, p)
	}
	return lista, nil
}

// (Añade estas funciones debajo de la que ya tienes "ObtenerVacantes")

func (r *PuestoRepository) ObtenerRegimenes() ([]models.RegimenLaboral, error) {
	query := `SELECT id, codigo, descripcion FROM regimenes_laborales ORDER BY id ASC`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.RegimenLaboral
	for rows.Next() {
		var reg models.RegimenLaboral
		rows.Scan(&reg.ID, &reg.Codigo, &reg.Descripcion)
		lista = append(lista, reg)
	}
	return lista, nil
}

func (r *PuestoRepository) ObtenerTodos(tenantID int) ([]models.Puesto, error) {
	query := `
		SELECT p.id, p.nombre, p.sueldo_presupuestado, p.estado, p.activo,
		       m.codigo, fr.rubro, rl.descripcion, p.unidad_organica_id, p.codigo_airhsp,
		       COALESCE(u.nombre, 'Sin asignar') AS unidad_organica_nombre
		FROM puestos p
		INNER JOIN metas_presupuestales m ON p.meta_id = m.id
		INNER JOIN fuentes_rubros fr ON p.fuente_rubro_id = fr.id
		INNER JOIN regimenes_laborales rl ON p.regimen_id = rl.id
		LEFT JOIN unidades_organicas u ON p.unidad_organica_id = u.id
		WHERE p.tenant_id = $1
		ORDER BY p.id DESC
	`
	rows, err := r.db.Query(query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.Puesto
	for rows.Next() {
		var p models.Puesto
		err := rows.Scan(&p.ID, &p.Nombre, &p.SueldoPresupuestado, &p.Estado, &p.Activo,
			&p.MetaCodigo, &p.FuenteRubroDesc, &p.RegimenDesc, &p.UnidadOrganicaID, &p.CodigoAirhsp, &p.UnidadOrganicaNombre)
		if err == nil {
			lista = append(lista, p)
		}
	}
	return lista, nil
}

// ObtenerTodosPaginacion todos los registros para paginacion
func (r *PuestoRepository) ObtenerTodosPaginacion(tenantID int, metaID int, regimenID int, unidadOrganicaID int, busqueda string, estado string, limite int, offset int) ([]models.Puesto, int, error) {
	whereClause := "WHERE p.tenant_id = $1"

	params := []interface{}{tenantID}
	paramIndex := 2

	if metaID > 0 {
		whereClause += fmt.Sprintf(" AND p.meta_id = $%d", paramIndex)
		params = append(params, metaID)
		paramIndex++
	}
	if regimenID > 0 {
		whereClause += fmt.Sprintf(" AND p.regimen_id = $%d", paramIndex)
		params = append(params, regimenID)
		paramIndex++
	}
	if unidadOrganicaID > 0 {
		whereClause += fmt.Sprintf(" AND p.unidad_organica_id = $%d", paramIndex)
		params = append(params, unidadOrganicaID)
		paramIndex++
	}
	if busqueda != "" {
		whereClause += fmt.Sprintf(" AND p.nombre ILIKE $%d", paramIndex)
		params = append(params, "%"+busqueda+"%")
		paramIndex++
	}
	if estado != "" {
		estado = strings.ToUpper(estado)
		whereClause += fmt.Sprintf(" AND p.estado = $%d", paramIndex)
		params = append(params, estado)
		paramIndex++
	}

	var totalRegistros int
	countQuery := fmt.Sprintf(
		`
			SELECT COUNT(*) FROM puestos p 
			INNER JOIN regimenes_laborales rl ON p.regimen_id = rl.id 
			LEFT JOIN metas_presupuestales m ON p.meta_id = m.id 
			LEFT JOIN fuentes_rubros fr ON p.fuente_rubro_id = fr.id 
			%s
		`, whereClause)

	err := r.db.QueryRow(countQuery, params...).Scan(&totalRegistros)
	if err != nil {
		log.Println("Error al obtener el total de registros (en puesto_repository):", err)
		return nil, 0, err
	}

	query := fmt.Sprintf(`
		SELECT p.id, p.nombre, p.sueldo_presupuestado, p.estado, p.activo,
		       m.codigo, fr.rubro, rl.descripcion, p.es_dietario,
		       p.unidad_organica_id, p.codigo_airhsp, COALESCE(u.nombre, 'Sin asignar') AS unidad_organica_nombre
		FROM puestos p
		INNER JOIN regimenes_laborales rl ON p.regimen_id = rl.id
		LEFT JOIN metas_presupuestales m ON p.meta_id = m.id
		LEFT JOIN fuentes_rubros fr ON p.fuente_rubro_id = fr.id
		LEFT JOIN unidades_organicas u ON p.unidad_organica_id = u.id
		%s
		ORDER BY p.id DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, paramIndex, paramIndex+1)

	params = append(params, limite, offset)

	rows, err := r.db.Query(query, params...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var lista []models.Puesto
	for rows.Next() {
		var p models.Puesto
		err := rows.Scan(&p.ID, &p.Nombre, &p.SueldoPresupuestado, &p.Estado, &p.Activo, &p.MetaCodigo, &p.FuenteRubroDesc, &p.RegimenDesc, &p.EsDietario, &p.UnidadOrganicaID, &p.CodigoAirhsp, &p.UnidadOrganicaNombre)
		if err == nil {
			lista = append(lista, p)
		}
	}
	return lista, totalRegistros, nil
}

func (r *PuestoRepository) Crear(p *models.Puesto) error {
	query := `
		INSERT INTO puestos (tenant_id, meta_id, fuente_rubro_id, regimen_id, nombre, sueldo_presupuestado, estado, activo, es_dietario, unidad_organica_id, codigo_airhsp)
		VALUES ($1, $2, $3, $4, $5, $6, 'VACANTE', $7, $8, $9, $10) RETURNING id
	`
	return r.db.QueryRow(query, p.TenantID, p.MetaID, p.FuenteRubroID, p.RegimenID, p.Nombre, p.SueldoPresupuestado, p.Activo, p.EsDietario, p.UnidadOrganicaID, p.CodigoAirhsp).Scan(&p.ID)
}

func (r *PuestoRepository) Actualizar(p *models.Puesto) error {
	query := `
		UPDATE puestos 
		SET nombre = $1, meta_id = $2, fuente_rubro_id = $3, regimen_id = $4, sueldo_presupuestado = $5, activo = $6, es_dietario = $7, unidad_organica_id = $8, codigo_airhsp = $9
		WHERE id = $10 AND tenant_id = $11
	`
	_, err := r.db.Exec(query, p.Nombre, p.MetaID, p.FuenteRubroID, p.RegimenID, p.SueldoPresupuestado, p.Activo, p.EsDietario, p.UnidadOrganicaID, p.CodigoAirhsp, p.ID, p.TenantID)
	return err
}

// ObtenerConceptosTenantPorCodigosSUNAT traduce los códigos universales a los IDs locales de la municipalidad
func (r *PuestoRepository) ObtenerConceptosTenantPorCodigosSUNAT(tenantID int, codigosSUNAT []string) ([]int, error) {
	if len(codigosSUNAT) == 0 {
		return nil, nil
	}

	query := `
		SELECT ct.id 
		FROM conceptos_tenant ct
		INNER JOIN conceptos_maestros cm ON ct.concepto_id = cm.id
		WHERE ct.tenant_id = $1 AND cm.codigo = ANY($2) AND ct.activo = true
	`
	// Usamos pq.Array para enviar el slice de strings de forma segura
	rows, err := r.db.Query(query, tenantID, pq.Array(codigosSUNAT))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// AsignarConceptosAPuesto realiza un Bulk Insert en puesto_conceptos
func (r *PuestoRepository) AsignarConceptosAPuesto(puestoID int, conceptoTenantIDs []int, sueldoBase float64) error {
	if len(conceptoTenantIDs) == 0 {
		return nil
	}

	// Iniciamos transacción para asegurar que se guarden todos o ninguno
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	// Consultamos los códigos SUNAT de los conceptos tenant a asignar
	rows, err := tx.Query(`
		SELECT ct.id, cm.codigo 
		FROM conceptos_tenant ct 
		INNER JOIN conceptos_maestros cm ON ct.concepto_id = cm.id 
		WHERE ct.id = ANY($1)
	`, pq.Array(conceptoTenantIDs))
	if err != nil {
		tx.Rollback()
		return err
	}
	defer rows.Close()

	conceptosSueldo := make(map[int]bool)
	for rows.Next() {
		var id int
		var codigo string
		if err := rows.Scan(&id, &codigo); err == nil {
			if codigo == "2001" || codigo == "2039" {
				conceptosSueldo[id] = true
			}
		}
	}

	stmt, err := tx.Prepare(`
		INSERT INTO puesto_conceptos (puesto_id, concepto_tenant_id, monto, activo) 
		VALUES ($1, $2, $3, true)
		ON CONFLICT (puesto_id, concepto_tenant_id) DO NOTHING -- Evita duplicados por si acaso
	`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, ctID := range conceptoTenantIDs {
		monto := 0.00
		if conceptosSueldo[ctID] {
			monto = sueldoBase
		}
		_, err := stmt.Exec(puestoID, ctID, monto)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

// ObtenerConceptoRemunerativoPorClasificador busca el ID de un concepto de tenant configurado bajo un régimen y con un clasificador específico
func (r *PuestoRepository) ObtenerConceptoRemunerativoPorClasificador(tenantID int, regimenID int, codigoMefLimpio string) (int, error) {
	var id int
	query := `
		SELECT ct.id 
		FROM conceptos_tenant ct
		INNER JOIN clasificadores_mef mef ON ct.clasificador_id = mef.id
		INNER JOIN regimen_concepto_tenant rct ON ct.id = rct.concepto_tenant_id
		WHERE ct.tenant_id = $1
		  AND rct.regimen_id = $2
		  AND mef.codigo_limpio = $3
		  AND ct.activo = true
		LIMIT 1
	`
	err := r.db.QueryRow(query, tenantID, regimenID, codigoMefLimpio).Scan(&id)
	return id, err
}

// ObtenerConceptosModeloPorRegimen lee la tabla intermedia y trae los conceptos
// que corresponden al régimen, excluyendo automáticamente los previsionales (ONP/AFP).
func (r *PuestoRepository) ObtenerConceptosModeloPorRegimen(tenantID int, regimenID int) ([]int, error) {
	query := `
		SELECT ct.id 
		FROM conceptos_tenant ct
		INNER JOIN regimen_concepto_tenant rct ON ct.id = rct.concepto_tenant_id
		INNER JOIN conceptos_maestros cma ON ct.concepto_id = cma.id
		WHERE rct.tenant_id = $1 
		  AND rct.regimen_id = $2 
		  AND ct.activo = true
		  AND cma.codigo NOT IN ('0601', '0606', '0607', '0608')
	`
	rows, err := r.db.Query(query, tenantID, regimenID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// RestaurarPlantillaBase reescrito para usar el nuevo catálogo SaaS
func (r *PuestoRepository) RestaurarPlantillaBase(puestoID int, tenantID int, regimenID int) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	// 1. Borramos la configuración actual del puesto, PERO PROTEGEMOS LAS PENSIONES
	deleteQuery := `
		DELETE FROM puesto_conceptos pc
		USING conceptos_tenant ct, conceptos_maestros cma
		WHERE pc.concepto_tenant_id = ct.id 
		  AND ct.concepto_id = cma.id
		  AND pc.puesto_id = $1
		  AND cma.codigo NOT IN ('0601', '0606', '0607', '0608')
	`
	_, err = tx.Exec(deleteQuery, puestoID)
	if err != nil {
		tx.Rollback()
		return err
	}

	// 2. Traemos la lista base desde el modelo (ya excluye pensiones)
	query := `
		SELECT ct.id 
		FROM conceptos_tenant ct
		INNER JOIN regimen_concepto_tenant rct ON ct.id = rct.concepto_tenant_id
		INNER JOIN conceptos_maestros cma ON ct.concepto_id = cma.id
		WHERE rct.tenant_id = $1 
		  AND rct.regimen_id = $2
		  AND ct.activo = true
		  AND cma.codigo NOT IN ('0601', '0606', '0607', '0608')
	`
	rows, err := tx.Query(query, tenantID, regimenID)
	if err != nil {
		tx.Rollback()
		return err
	}

	var idsLocales []int
	for rows.Next() {
		var id int
		rows.Scan(&id)
		idsLocales = append(idsLocales, id)
	}
	rows.Close()

	// 3. Insertamos de nuevo usando ON CONFLICT para máxima seguridad
	stmt, _ := tx.Prepare(`
		INSERT INTO puesto_conceptos (puesto_id, concepto_tenant_id, monto, activo) 
		VALUES ($1, $2, 0, true)
		ON CONFLICT (puesto_id, concepto_tenant_id) DO NOTHING
	`)
	for _, ctID := range idsLocales {
		tx.Stmt(stmt).Exec(puestoID, ctID)
	}

	return tx.Commit()
}

// ObtenerPorID trae todos los datos de un puesto específico y el código de su régimen laboral
func (r *PuestoRepository) ObtenerPorID(id int, tenantID int) (models.Puesto, error) {
	var p models.Puesto
	query := `
		SELECT p.id, p.tenant_id, p.meta_id, p.fuente_rubro_id, p.regimen_id,
		       p.nombre, p.sueldo_presupuestado, p.estado, p.activo, p.es_dietario, rl.codigo,
		       p.unidad_organica_id, p.codigo_airhsp
		FROM puestos p 
		INNER JOIN regimenes_laborales rl ON p.regimen_id = rl.id 
		WHERE p.id = $1 AND p.tenant_id = $2
	`
	err := r.db.QueryRow(query, id, tenantID).Scan(
		&p.ID, &p.TenantID, &p.MetaID, &p.FuenteRubroID, &p.RegimenID,
		&p.Nombre, &p.SueldoPresupuestado, &p.Estado, &p.Activo, &p.EsDietario, &p.RegimenCodigo,
		&p.UnidadOrganicaID, &p.CodigoAirhsp,
	)
	return p, err
}

// DB es un getter para acceder a la conexión desde los handlers si es necesario
func (r *PuestoRepository) DB() *sql.DB {
	return r.db
}

// ObtenerPuestosParaPAP extrae los puestos activos con su jerarquía presupuestal
func (r *PuestoRepository) ObtenerPuestosParaPAP(tenantID int) ([]models.PuestoPAP, error) {
	query := `
		SELECT p.id, rl.codigo, 
		       m.codigo, m.descripcion, 
		       -- Enviamos el ID como código corto, y el nombre completo a la descripción (que soporta 255 caracteres)
		       CAST(fr.id AS VARCHAR), fr.fuente_financiamiento || ' | ' || fr.rubro 
		FROM puestos p
		INNER JOIN regimenes_laborales rl ON p.regimen_id = rl.id
		INNER JOIN metas_presupuestales m ON p.meta_id = m.id
		INNER JOIN fuentes_rubros fr ON p.fuente_rubro_id = fr.id
		WHERE p.tenant_id = $1 AND p.activo = true
	`
	rows, err := r.db.Query(query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.PuestoPAP
	for rows.Next() {
		var p models.PuestoPAP
		err := rows.Scan(&p.ID, &p.RegimenCodigo, &p.MetaCodigo, &p.MetaDescripcion, &p.FuenteRubroCodigo, &p.FuenteRubroDescripcion)
		if err == nil {
			lista = append(lista, p)
		}
	}
	return lista, nil
}

// ObtenerConceptosParaAsignacion devuelve todos los conceptos activos del tenant
// y marca (LEFT JOIN) cuáles ya tiene asignados el puesto actualmente.
func (r *PuestoRepository) ObtenerConceptosParaAsignacion(puestoID, tenantID int) ([]models.ConceptoAsignacion, error) {
	query := `
		SELECT 
			ct.id AS concepto_tenant_id, 
			ct.nombre_personalizado, 
			cm.tipo, 
			ct.requiere_monto,
			CASE WHEN pc.id IS NOT NULL AND pc.activo = true THEN true ELSE false END AS asignado,
			COALESCE(pc.monto, 0.00) AS monto
		FROM conceptos_tenant ct
		INNER JOIN conceptos_maestros cm ON ct.concepto_id = cm.id
		-- El cruce clave: Solo unimos si el puesto coincide
		LEFT JOIN puesto_conceptos pc ON ct.id = pc.concepto_tenant_id AND pc.puesto_id = $1
		WHERE ct.tenant_id = $2 AND ct.activo = true
		ORDER BY cm.tipo DESC, ct.nombre_personalizado ASC
	`

	rows, err := r.db.Query(query, puestoID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.ConceptoAsignacion
	for rows.Next() {
		var c models.ConceptoAsignacion
		err := rows.Scan(&c.ConceptoTenantID, &c.Nombre, &c.Tipo, &c.RequiereMonto, &c.Asignado, &c.Monto)
		if err != nil {
			return nil, err
		}
		lista = append(lista, c)
	}
	return lista, nil
}

// GuardarAsignacionConceptos actualiza la estructura de pago de un puesto.
// Usa una transacción para limpiar lo anterior y guardar la nueva selección.
func (r *PuestoRepository) GuardarAsignacionConceptos(puestoID int, asignaciones []models.ConceptoAsignacion) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	// 1. Borramos la configuración anterior del puesto (Lienzo en blanco)
	_, err = tx.Exec(`DELETE FROM puesto_conceptos WHERE puesto_id = $1`, puestoID)
	if err != nil {
		tx.Rollback()
		return err
	}

	// 2. Insertamos solo los conceptos que vienen marcados como "Asignado"
	queryInsert := `
		INSERT INTO puesto_conceptos (puesto_id, concepto_tenant_id, monto, activo) 
		VALUES ($1, $2, $3, true)
	`
	for _, c := range asignaciones {
		if c.Asignado {
			_, err = tx.Exec(queryInsert, puestoID, c.ConceptoTenantID, c.Monto)
			if err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	// Confirmamos los cambios
	return tx.Commit()
}
