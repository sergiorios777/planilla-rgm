package repository

import (
	"database/sql"
	"fmt"
	"planilla-rgm/internal/models"
	"strings"
)

type MefMucRepository struct {
	db *sql.DB
}

func NewMefMucRepository(db *sql.DB) *MefMucRepository {
	repo := &MefMucRepository{db: db}
	repo.AutoMigrar()
	return repo
}

// AutoMigrar garantiza que la tabla mef_muc_valores esté creada en la base de datos
func (r *MefMucRepository) AutoMigrar() {
	query := `
		CREATE TABLE IF NOT EXISTS mef_muc_valores (
			id SERIAL PRIMARY KEY,
			norma_legal VARCHAR(150) NOT NULL,
			fecha_norma DATE NOT NULL,
			grupo_ocupacional VARCHAR(50) NOT NULL,
			nivel_remunerativo VARCHAR(20) NOT NULL,
			monto_muc NUMERIC(12,2) NOT NULL DEFAULT 0.00,
			activo BOOLEAN NOT NULL DEFAULT true,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_mef_muc_norma ON mef_muc_valores(norma_legal);
		CREATE INDEX IF NOT EXISTS idx_mef_muc_grupo_nivel ON mef_muc_valores(grupo_ocupacional, nivel_remunerativo);
		DELETE FROM mef_muc_valores WHERE norma_legal ~ '^\d{4}-\d{2}-\d{2}$';
	`
	_, _ = r.db.Exec(query)
}

// ObtenerNormasLegales devuelve el listado de normas legales registradas sin duplicados
func (r *MefMucRepository) ObtenerNormasLegales() ([]string, error) {
	query := `SELECT DISTINCT norma_legal FROM mef_muc_valores ORDER BY norma_legal ASC`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var normas []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err == nil {
			normas = append(normas, n)
		}
	}
	return normas, nil
}

// ListarPaginado consulta registros de MUC aplicando filtros dinámicos y paginación
func (r *MefMucRepository) ListarPaginado(filtros models.MefMucFiltros) ([]models.MefMucValor, int, error) {
	if filtros.Limite <= 0 {
		filtros.Limite = 15
	}
	if filtros.Pagina <= 0 {
		filtros.Pagina = 1
	}
	offset := (filtros.Pagina - 1) * filtros.Limite

	var where []string
	var args []interface{}
	argIdx := 1

	if strings.TrimSpace(filtros.NormaLegal) != "" {
		where = append(where, fmt.Sprintf("norma_legal = $%d", argIdx))
		args = append(args, strings.TrimSpace(filtros.NormaLegal))
		argIdx++
	}

	if strings.TrimSpace(filtros.FechaNorma) != "" {
		where = append(where, fmt.Sprintf("fecha_norma = $%d", argIdx))
		args = append(args, strings.TrimSpace(filtros.FechaNorma))
		argIdx++
	}

	if filtros.Activo == "activos" {
		where = append(where, fmt.Sprintf("activo = $%d", argIdx))
		args = append(args, true)
		argIdx++
	} else if filtros.Activo == "inactivos" {
		where = append(where, fmt.Sprintf("activo = $%d", argIdx))
		args = append(args, false)
		argIdx++
	}

	if strings.TrimSpace(filtros.Buscar) != "" {
		busqueda := "%" + strings.TrimSpace(filtros.Buscar) + "%"
		where = append(where, fmt.Sprintf("(grupo_ocupacional ILIKE $%d OR nivel_remunerativo ILIKE $%d OR norma_legal ILIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, busqueda)
		argIdx++
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}

	// Contar total de registros
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM mef_muc_valores %s", whereClause)
	var total int
	err := r.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Consultar registros paginados
	dataQuery := fmt.Sprintf(`
		SELECT id, norma_legal, fecha_norma, grupo_ocupacional, nivel_remunerativo, monto_muc, activo, created_at, updated_at
		FROM mef_muc_valores
		%s
		ORDER BY fecha_norma DESC, grupo_ocupacional ASC, nivel_remunerativo ASC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	args = append(args, filtros.Limite, offset)
	rows, err := r.db.Query(dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var lista []models.MefMucValor
	for rows.Next() {
		var v models.MefMucValor
		err := rows.Scan(
			&v.ID, &v.NormaLegal, &v.FechaNorma, &v.GrupoOcupacional,
			&v.NivelRemunerativo, &v.MontoMuc, &v.Activo, &v.CreatedAt, &v.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		v.FechaNormaFormato = v.FechaNorma.Format("2006-01-02")
		v.MontoMucFormato = models.FormatearMontoMUC(v.MontoMuc)
		lista = append(lista, v)
	}

	return lista, total, nil
}

// ObtenerPorID obtiene un único registro MUC por su ID
func (r *MefMucRepository) ObtenerPorID(id int) (*models.MefMucValor, error) {
	query := `
		SELECT id, norma_legal, fecha_norma, grupo_ocupacional, nivel_remunerativo, monto_muc, activo, created_at, updated_at
		FROM mef_muc_valores
		WHERE id = $1
	`
	var v models.MefMucValor
	err := r.db.QueryRow(query, id).Scan(
		&v.ID, &v.NormaLegal, &v.FechaNorma, &v.GrupoOcupacional,
		&v.NivelRemunerativo, &v.MontoMuc, &v.Activo, &v.CreatedAt, &v.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	v.FechaNormaFormato = v.FechaNorma.Format("2006-01-02")
	v.MontoMucFormato = models.FormatearMontoMUC(v.MontoMuc)
	return &v, nil
}

// Crear inserta un nuevo registro en mef_muc_valores
func (r *MefMucRepository) Crear(v *models.MefMucValor) error {
	query := `
		INSERT INTO mef_muc_valores (norma_legal, fecha_norma, grupo_ocupacional, nivel_remunerativo, monto_muc, activo)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRow(
		query,
		strings.TrimSpace(v.NormaLegal),
		v.FechaNorma,
		strings.TrimSpace(v.GrupoOcupacional),
		strings.TrimSpace(v.NivelRemunerativo),
		v.MontoMuc,
		v.Activo,
	).Scan(&v.ID, &v.CreatedAt, &v.UpdatedAt)
}

// Actualizar edita los datos de un registro MUC existente
func (r *MefMucRepository) Actualizar(v *models.MefMucValor) error {
	query := `
		UPDATE mef_muc_valores
		SET norma_legal = $1, fecha_norma = $2, grupo_ocupacional = $3, nivel_remunerativo = $4, monto_muc = $5, activo = $6, updated_at = NOW()
		WHERE id = $7
	`
	_, err := r.db.Exec(
		query,
		strings.TrimSpace(v.NormaLegal),
		v.FechaNorma,
		strings.TrimSpace(v.GrupoOcupacional),
		strings.TrimSpace(v.NivelRemunerativo),
		v.MontoMuc,
		v.Activo,
		v.ID,
	)
	return err
}

// CambiarEstado conmuta el atributo 'activo' de un registro MUC
func (r *MefMucRepository) CambiarEstado(id int, activo bool) error {
	query := `UPDATE mef_muc_valores SET activo = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(query, activo, id)
	return err
}

// ImportarCSVBulk inserta o actualiza masivamente un conjunto de registros de MUC
func (r *MefMucRepository) ImportarCSVBulk(lista []models.MefMucValor) (int, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO mef_muc_valores (norma_legal, fecha_norma, grupo_ocupacional, nivel_remunerativo, monto_muc, activo)
		VALUES ($1, $2, $3, $4, $5, $6)
	`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	insertados := 0
	for _, v := range lista {
		_, err := stmt.Exec(
			strings.TrimSpace(v.NormaLegal),
			v.FechaNorma,
			strings.TrimSpace(v.GrupoOcupacional),
			strings.TrimSpace(v.NivelRemunerativo),
			v.MontoMuc,
			v.Activo,
		)
		if err == nil {
			insertados++
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return insertados, nil
}
