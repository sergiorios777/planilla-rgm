package repository

import (
	"database/sql"
	"log"
	"planilla-rgm/internal/models"

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
		SELECT p.id, p.nombre, p.sueldo_presupuestado, rl.descripcion
		FROM puestos p
		INNER JOIN regimenes_laborales rl ON p.regimen_id = rl.id
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
		rows.Scan(&p.ID, &p.Nombre, &p.SueldoPresupuestado, &p.RegimenDesc)
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
		       m.codigo, fr.rubro, rl.descripcion
		FROM puestos p
		INNER JOIN metas_presupuestales m ON p.meta_id = m.id
		INNER JOIN fuentes_rubros fr ON p.fuente_rubro_id = fr.id
		INNER JOIN regimenes_laborales rl ON p.regimen_id = rl.id
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
			&p.MetaCodigo, &p.FuenteRubroDesc, &p.RegimenDesc)
		if err == nil {
			lista = append(lista, p)
		}
	}
	return lista, nil
}

func (r *PuestoRepository) Crear(p *models.Puesto) error {
	query := `
		INSERT INTO puestos (tenant_id, meta_id, fuente_rubro_id, regimen_id, nombre, sueldo_presupuestado, estado, activo, es_dietario)
		VALUES ($1, $2, $3, $4, $5, $6, 'VACANTE', $7, $8) RETURNING id
	`
	return r.db.QueryRow(query, p.TenantID, p.MetaID, p.FuenteRubroID, p.RegimenID, p.Nombre, p.SueldoPresupuestado, p.Activo, p.EsDietario).Scan(&p.ID)
}

func (r *PuestoRepository) Actualizar(p *models.Puesto) error {
	query := `
		UPDATE puestos 
		SET nombre = $1, meta_id = $2, fuente_rubro_id = $3, regimen_id = $4, sueldo_presupuestado = $5, activo = $6, es_dietario = $7
		WHERE id = $8 AND tenant_id = $9
	`
	_, err := r.db.Exec(query, p.Nombre, p.MetaID, p.FuenteRubroID, p.RegimenID, p.SueldoPresupuestado, p.Activo, p.EsDietario, p.ID, p.TenantID)
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

	stmt, _ := tx.Prepare(`
		INSERT INTO puesto_conceptos (puesto_id, concepto_tenant_id, monto, activo) 
		VALUES ($1, $2, $3, true)
		ON CONFLICT (puesto_id, concepto_tenant_id) DO NOTHING -- Evita duplicados por si acaso
	`)
	defer stmt.Close()

	for _, ctID := range conceptoTenantIDs {
		// Por defecto el monto es 0 (para conceptos variables/calculados),
		// pero podríamos inyectar el sueldo_presupuestado si logramos identificar cuál es el concepto de sueldo
		monto := 0.00
		_, err := stmt.Exec(puestoID, ctID, monto)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func (r *PuestoRepository) RestaurarPlantillaBase(puestoID int, tenantID int, codigosSUNAT []string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	log.Println("----------------")
	log.Println("puestoID", puestoID)
	log.Println("tenantID", tenantID)
	log.Println("codigosSUNAT", codigosSUNAT)

	// 1. Borrar actuales
	_, err = tx.Exec(`DELETE FROM puesto_conceptos WHERE puesto_id = $1`, puestoID)
	if err != nil {
		tx.Rollback()
		return err
	}

	// 2. Traducir códigos a IDs locales
	// (Reutilizamos la lógica de consulta de IDs que ya tenemos)
	query := `
		SELECT ct.id FROM conceptos_tenant ct
		INNER JOIN conceptos_maestros cm ON ct.concepto_id = cm.id
		WHERE ct.tenant_id = $1 AND cm.codigo = ANY($2) AND ct.activo = true`

	rows, err := tx.Query(query, tenantID, pq.Array(codigosSUNAT))
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
	log.Println("idsLocales", idsLocales)
	log.Println("----------------")
	rows.Close()

	// 3. Insertar de nuevo con monto 0
	stmt, _ := tx.Prepare(`INSERT INTO puesto_conceptos (puesto_id, concepto_tenant_id, monto) VALUES ($1, $2, 0)`)
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
		       p.nombre, p.sueldo_presupuestado, p.estado, p.activo, p.es_dietario, rl.codigo 
		FROM puestos p 
		INNER JOIN regimenes_laborales rl ON p.regimen_id = rl.id 
		WHERE p.id = $1 AND p.tenant_id = $2
	`
	err := r.db.QueryRow(query, id, tenantID).Scan(
		&p.ID, &p.TenantID, &p.MetaID, &p.FuenteRubroID, &p.RegimenID,
		&p.Nombre, &p.SueldoPresupuestado, &p.Estado, &p.Activo, &p.EsDietario, &p.RegimenCodigo,
	)
	return p, err
}

// DB es un getter para acceder a la conexión desde los handlers si es necesario
func (r *PuestoRepository) DB() *sql.DB {
	return r.db
}
