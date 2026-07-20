package repository

import (
	"database/sql"
	"planilla-rgm/internal/models"
)

// AFPRepository gestiona la comunicación con las tablas 'afps' y 'afp_tasas_mensuales'
type AFPRepository struct {
	db *sql.DB
}

// NewAFPRepository crea una nueva instancia de AFPRepository
func NewAFPRepository(db *sql.DB) *AFPRepository {
	return &AFPRepository{db: db}
}

// ObtenerTodos retorna todas las administradoras registradas
func (r *AFPRepository) ObtenerTodos(busqueda string) ([]models.AFP, error) {
	query := `
		SELECT id, nombre, COALESCE(codigo_sbs, '') as codigo_sbs, activo 
		FROM afps 
		WHERE nombre ILIKE '%' || $1 || '%' OR codigo_sbs ILIKE '%' || $1 || '%' 
		ORDER BY nombre ASC`

	rows, err := r.db.Query(query, busqueda)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.AFP
	for rows.Next() {
		var a models.AFP
		if err := rows.Scan(&a.ID, &a.Nombre, &a.CodigoSBS, &a.Activo); err != nil {
			return nil, err
		}
		lista = append(lista, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return lista, nil
}

// ObtenerPorID busca una AFP específica
func (r *AFPRepository) ObtenerPorID(id int) (*models.AFP, error) {
	var a models.AFP
	query := `SELECT id, nombre, COALESCE(codigo_sbs, '') as codigo_sbs, activo FROM afps WHERE id = $1`
	err := r.db.QueryRow(query, id).Scan(&a.ID, &a.Nombre, &a.CodigoSBS, &a.Activo)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// Crear inserta una nueva AFP
func (r *AFPRepository) Crear(a *models.AFP) error {
	query := `INSERT INTO afps (nombre, codigo_sbs, activo) VALUES ($1, $2, $3) RETURNING id`
	return r.db.QueryRow(query, a.Nombre, a.CodigoSBS, a.Activo).Scan(&a.ID)
}

// Actualizar guarda los cambios en una AFP existente
func (r *AFPRepository) Actualizar(a *models.AFP) error {
	query := `UPDATE afps SET nombre = $1, codigo_sbs = $2, activo = $3 WHERE id = $4`
	_, err := r.db.Exec(query, a.Nombre, a.CodigoSBS, a.Activo, a.ID)
	return err
}

// ObtenerTasasPorMes realiza un LEFT JOIN para obtener la matriz completa de tasas del mes
func (r *AFPRepository) ObtenerTasasPorMes(anio int, mes int) ([]models.AFPTasaVista, error) {
	query := `
		SELECT 
			a.id as afp_id, 
			a.nombre as afp_nombre, 
			COALESCE(a.codigo_sbs, '') as afp_codigo_sbs,
			t.id as tasa_id,
			COALESCE(t.anio, $1) as anio,
			COALESCE(t.mes, $2) as mes,
			COALESCE(t.aporte_obligatorio, 0.0) as aporte_obligatorio,
			COALESCE(t.comision_flujo, 0.0) as comision_flujo,
			COALESCE(t.comision_mixta_flujo, 0.0) as comision_mixta_flujo,
			COALESCE(t.prima_seguro, 0.0) as prima_seguro,
			COALESCE(t.comision_anual_saldo, 0.0) as comision_anual_saldo,
			CASE WHEN t.id IS NOT NULL THEN true ELSE false END as registrado
		FROM afps a
		LEFT JOIN afp_tasas_mensuales t ON a.id = t.afp_id AND t.anio = $1 AND t.mes = $2
		WHERE a.activo = true
		ORDER BY a.nombre ASC`

	rows, err := r.db.Query(query, anio, mes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.AFPTasaVista
	for rows.Next() {
		var v models.AFPTasaVista
		var tasaID sql.NullInt64
		err := rows.Scan(
			&v.AfpID, &v.AfpNombre, &v.AfpCodigoSBS, &tasaID,
			&v.Anio, &v.Mes, &v.AporteObligatorio, &v.ComisionFlujo,
			&v.ComisionMixtaFlujo, &v.PrimaSeguro, &v.ComisionAnualSaldo,
			&v.Registrado,
		)
		if err != nil {
			return nil, err
		}
		if tasaID.Valid {
			val := int(tasaID.Int64)
			v.TasaID = &val
		}
		lista = append(lista, v)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return lista, nil
}

// GuardarTasasMensuales realiza el guardado/actualización masiva de tasas de un mes
func (r *AFPRepository) GuardarTasasMensuales(tasas []models.AFPTasaMensual) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	query := `
		INSERT INTO afp_tasas_mensuales (
			afp_id, anio, mes, aporte_obligatorio, comision_flujo, 
			comision_mixta_flujo, prima_seguro, comision_anual_saldo
		) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (afp_id, anio, mes) 
		DO UPDATE SET 
			aporte_obligatorio = EXCLUDED.aporte_obligatorio,
			comision_flujo = EXCLUDED.comision_flujo,
			comision_mixta_flujo = EXCLUDED.comision_mixta_flujo,
			prima_seguro = EXCLUDED.prima_seguro,
			comision_anual_saldo = EXCLUDED.comision_anual_saldo`

	stmt, err := tx.Prepare(query)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, t := range tasas {
		_, err := stmt.Exec(
			t.AfpID, t.Anio, t.Mes, t.AporteObligatorio, t.ComisionFlujo,
			t.ComisionMixtaFlujo, t.PrimaSeguro, t.ComisionAnualSaldo,
		)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}
