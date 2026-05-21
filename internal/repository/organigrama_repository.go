package repository

import (
	"database/sql"
	"errors"
	"planilla-rgm/internal/models"
)

type OrganigramaRepository struct {
	db *sql.DB
}

func NewOrganigramaRepository(db *sql.DB) *OrganigramaRepository {
	return &OrganigramaRepository{db: db}
}

// CrearOrganigrama inserta un nuevo organigrama
func (r *OrganigramaRepository) CrearOrganigrama(o *models.Organigrama) error {
	query := `
		INSERT INTO organigramas (tenant_id, documento_aprobacion, descripcion, fecha_vigencia, activo)
		VALUES ($1, $2, $3, $4, false) RETURNING id, created_at, updated_at
	`
	return r.db.QueryRow(query, o.TenantID, o.DocumentoAprobacion, o.Descripcion, o.FechaVigencia).
		Scan(&o.ID, &o.CreatedAt, &o.UpdatedAt)
}

// ObtenerOrganigramasPorTenant obtiene todas las versiones de organigramas
func (r *OrganigramaRepository) ObtenerOrganigramasPorTenant(tenantID int) ([]models.Organigrama, error) {
	query := `
		SELECT id, tenant_id, documento_aprobacion, descripcion, fecha_vigencia, activo, created_at, updated_at
		FROM organigramas
		WHERE tenant_id = $1
		ORDER BY fecha_vigencia DESC, id DESC
	`
	rows, err := r.db.Query(query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.Organigrama
	for rows.Next() {
		var o models.Organigrama
		err := rows.Scan(&o.ID, &o.TenantID, &o.DocumentoAprobacion, &o.Descripcion, &o.FechaVigencia, &o.Activo, &o.CreatedAt, &o.UpdatedAt)
		if err == nil {
			lista = append(lista, o)
		}
	}
	return lista, nil
}

// ObtenerOrganigramaActivo obtiene la versión activa actual
func (r *OrganigramaRepository) ObtenerOrganigramaActivo(tenantID int) (*models.Organigrama, error) {
	var o models.Organigrama
	query := `
		SELECT id, tenant_id, documento_aprobacion, descripcion, fecha_vigencia, activo
		FROM organigramas
		WHERE tenant_id = $1 AND activo = true
		LIMIT 1
	`
	err := r.db.QueryRow(query, tenantID).Scan(&o.ID, &o.TenantID, &o.DocumentoAprobacion, &o.Descripcion, &o.FechaVigencia, &o.Activo)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &o, nil
}

// ActivarOrganigrama activa un organigrama y desactiva los demás para el tenant
func (r *OrganigramaRepository) ActivarOrganigrama(id, tenantID int) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	// 1. Apagar todos
	_, err = tx.Exec(`UPDATE organigramas SET activo = false WHERE tenant_id = $1`, tenantID)
	if err != nil {
		tx.Rollback()
		return err
	}

	// 2. Encender el seleccionado
	_, err = tx.Exec(`UPDATE organigramas SET activo = true WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// CrearUnidad crea una unidad orgánica
func (r *OrganigramaRepository) CrearUnidad(u *models.UnidadOrganica) error {
	query := `
		INSERT INTO unidades_organicas (tenant_id, organigrama_id, parent_id, codigo_mef, nombre, tipo)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id
	`
	return r.db.QueryRow(query, u.TenantID, u.OrganigramaID, u.ParentID, u.CodigoMef, u.Nombre, u.Tipo).Scan(&u.ID)
}

// ActualizarUnidad modifica una unidad orgánica existente
func (r *OrganigramaRepository) ActualizarUnidad(u *models.UnidadOrganica) error {
	query := `
		UPDATE unidades_organicas
		SET parent_id = $1, codigo_mef = $2, nombre = $3, tipo = $4, updated_at = CURRENT_TIMESTAMP
		WHERE id = $5 AND tenant_id = $6
	`
	_, err := r.db.Exec(query, u.ParentID, u.CodigoMef, u.Nombre, u.Tipo, u.ID, u.TenantID)
	return err
}

// EliminarUnidad elimina una unidad orgánica
func (r *OrganigramaRepository) EliminarUnidad(id, tenantID int) error {
	query := `DELETE FROM unidades_organicas WHERE id = $1 AND tenant_id = $2`
	_, err := r.db.Exec(query, id, tenantID)
	return err
}

// ObtenerUnidades obtiene la lista plana de unidades orgánicas de un organigrama
func (r *OrganigramaRepository) ObtenerUnidades(organigramaID int) ([]models.UnidadOrganica, error) {
	query := `
		SELECT id, tenant_id, organigrama_id, parent_id, codigo_mef, nombre, tipo
		FROM unidades_organicas
		WHERE organigrama_id = $1
		ORDER BY parent_id NULLS FIRST, nombre ASC
	`
	rows, err := r.db.Query(query, organigramaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.UnidadOrganica
	for rows.Next() {
		var u models.UnidadOrganica
		var parentID sql.NullInt64
		err := rows.Scan(&u.ID, &u.TenantID, &u.OrganigramaID, &parentID, &u.CodigoMef, &u.Nombre, &u.Tipo)
		if err == nil {
			if parentID.Valid {
				v := int(parentID.Int64)
				u.ParentID = &v
			}
			lista = append(lista, u)
		}
	}
	return lista, nil
}

// ObtenerUnidadPorID obtiene los datos de una unidad específica
func (r *OrganigramaRepository) ObtenerUnidadPorID(id int) (*models.UnidadOrganica, error) {
	var u models.UnidadOrganica
	var parentID sql.NullInt64
	query := `
		SELECT id, tenant_id, organigrama_id, parent_id, codigo_mef, nombre, tipo
		FROM unidades_organicas
		WHERE id = $1
	`
	err := r.db.QueryRow(query, id).Scan(&u.ID, &u.TenantID, &u.OrganigramaID, &parentID, &u.CodigoMef, &u.Nombre, &u.Tipo)
	if err != nil {
		return nil, err
	}
	if parentID.Valid {
		v := int(parentID.Int64)
		u.ParentID = &v
	}
	return &u, nil
}

// ObtenerArbolNodos construye el árbol visual y calcula las plazas por cada nodo orgánico
func (r *OrganigramaRepository) ObtenerArbolNodos(organigramaID int) ([]models.UnidadNodo, error) {
	// 1. Obtener todas las unidades
	unidades, err := r.ObtenerUnidades(organigramaID)
	if err != nil {
		return nil, err
	}

	// 2. Obtener conteo de puestos por unidad activa
	puestosQuery := `
		SELECT unidad_organica_id, COUNT(*) 
		FROM puestos 
		WHERE unidad_organica_id IS NOT NULL AND activo = true
		GROUP BY unidad_organica_id
	`
	rows, err := r.db.Query(puestosQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	conteoPuestos := make(map[int]int)
	for rows.Next() {
		var uid, count int
		if err := rows.Scan(&uid, &count); err == nil {
			conteoPuestos[uid] = count
		}
	}

	// 3. Crear nodos e indexar
	nodosMap := make(map[int]*models.UnidadNodo)

	for _, u := range unidades {
		n := &models.UnidadNodo{
			ID:           u.ID,
			Nombre:       u.Nombre,
			Tipo:         u.Tipo,
			CodigoMef:    u.CodigoMef,
			ParentID:     u.ParentID,
			TotalPuestos: conteoPuestos[u.ID],
			Hijos:        []models.UnidadNodo{},
		}
		nodosMap[u.ID] = n
	}

	// 4. Conectar jerarquía de árbol en memoria (construyendo las listas de Hijos en los punteros)
	for _, u := range unidades {
		if u.ParentID == nil {
			continue
		}
		padre, existe := nodosMap[*u.ParentID]
		if existe {
			// Añadimos una copia por ahora
			padre.Hijos = append(padre.Hijos, *nodosMap[u.ID])
		}
	}

	// Dado que agregamos copias, si un hijo tiene a su vez hijos, el padre original no verá los hijos del hijo si lo hicimos en orden incorrecto.
	// Para hacerlo de forma robusta y recursiva sin punteros complejos en la struct UnidadNodo:
	// Construiremos el árbol recursivo de forma limpia:
	return construirArbol(unidades, conteoPuestos), nil
}

func construirArbol(unidades []models.UnidadOrganica, conteoPuestos map[int]int) []models.UnidadNodo {
	var raices []models.UnidadNodo
	// Mapeamos hijos por parent_id
	hijosMap := make(map[int][]models.UnidadOrganica)
	for _, u := range unidades {
		if u.ParentID != nil {
			hijosMap[*u.ParentID] = append(hijosMap[*u.ParentID], u)
		}
	}

	// Función recursiva para armar un nodo y sus descendientes
	var construirNodo func(u models.UnidadOrganica) models.UnidadNodo
	construirNodo = func(u models.UnidadOrganica) models.UnidadNodo {
		nodo := models.UnidadNodo{
			ID:           u.ID,
			Nombre:       u.Nombre,
			Tipo:         u.Tipo,
			CodigoMef:    u.CodigoMef,
			ParentID:     u.ParentID,
			TotalPuestos: conteoPuestos[u.ID],
			Hijos:        []models.UnidadNodo{},
		}
		for _, hijo := range hijosMap[u.ID] {
			nodo.Hijos = append(nodo.Hijos, construirNodo(hijo))
		}
		return nodo
	}

	// Encontrar raíces
	for _, u := range unidades {
		if u.ParentID == nil {
			raices = append(raices, construirNodo(u))
		}
	}
	return raices
}

// ClonarEstructuraYTrasladarPuestos clona el organigrama y traslada plazas a la nueva versión usando fila de resolución dinámica
func (r *OrganigramaRepository) ClonarEstructuraYTrasladarPuestos(tenantID, origenID, destinoID int) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	// 1. Leer Unidades del Origen
	unidades, err := r.ObtenerUnidades(origenID)
	if err != nil {
		tx.Rollback()
		return err
	}

	mapaIDs := make(map[int]int) // ViejoID -> NuevoID
	pendientes := unidades
	
	// 2. Loop de resolución de jerarquías (Postponing recursivo)
	for len(pendientes) > 0 {
		progreso := false
		var siguienteRonda []models.UnidadOrganica

		for _, u := range pendientes {
			// Si no tiene padre, o el padre ya fue clonado e insertado:
			if u.ParentID == nil || mapaIDs[*u.ParentID] > 0 {
				var parentIDInsert *int
				if u.ParentID != nil {
					newParent := mapaIDs[*u.ParentID]
					parentIDInsert = &newParent
				}

				var nuevoID int
				insertQuery := `
					INSERT INTO unidades_organicas (tenant_id, organigrama_id, parent_id, codigo_mef, nombre, tipo)
					VALUES ($1, $2, $3, $4, $5, $6) RETURNING id
				`
				err = tx.QueryRow(insertQuery, tenantID, destinoID, parentIDInsert, u.CodigoMef, u.Nombre, u.Tipo).Scan(&nuevoID)
				if err != nil {
					tx.Rollback()
					return err
				}

				mapaIDs[u.ID] = nuevoID
				progreso = true
			} else {
				// Postponer para la siguiente vuelta
				siguienteRonda = append(siguienteRonda, u)
			}
		}

		if !progreso && len(pendientes) > 0 {
			// Detección de bucles infinitos o referencias rotas
			tx.Rollback()
			return errors.New("error en la clonación: se detectó una jerarquía cíclica o huérfana en las dependencias orgánicas")
		}

		pendientes = siguienteRonda
	}

	// 3. Trasladar Puestos al Nuevo ID correspondientes
	stmtUpdatePuesto, err := tx.Prepare(`
		UPDATE puestos 
		SET unidad_organica_id = $1 
		WHERE unidad_organica_id = $2 AND tenant_id = $3
	`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmtUpdatePuesto.Close()

	for viejoID, nuevoID := range mapaIDs {
		_, err = stmtUpdatePuesto.Exec(nuevoID, viejoID, tenantID)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	// 4. Apagar V1 y encender V2
	_, err = tx.Exec(`UPDATE organigramas SET activo = false WHERE id = $1 AND tenant_id = $2`, origenID, tenantID)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`UPDATE organigramas SET activo = true WHERE id = $1 AND tenant_id = $2`, destinoID, tenantID)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}
