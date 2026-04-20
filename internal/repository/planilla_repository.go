package repository

import (
	"database/sql"
	"planilla-rgm/internal/models"
	"strconv"
	"strings"
)

type PlanillaRepository struct {
	db *sql.DB
}

type ConceptoTemp struct {
	TenantID         int
	MaestroID        int
	MaestroCodigo    string
	Tipo             string
	Monto            float64
	Frecuencia       string
	EsExtraordinario bool
}

func NewPlanillaRepository(db *sql.DB) *PlanillaRepository {
	return &PlanillaRepository{db: db}
}

// ObtenerTodos trae el historial de planillas de la entidad
func (r *PlanillaRepository) ObtenerTodos(tenantID int) ([]models.Planilla, error) {
	query := `
		SELECT id, tenant_id, anio, mes, descripcion, estado 
		FROM planillas 
		WHERE tenant_id = $1 
		ORDER BY anio DESC, mes DESC, id DESC
	`
	rows, err := r.db.Query(query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.Planilla
	for rows.Next() {
		var p models.Planilla
		err := rows.Scan(&p.ID, &p.TenantID, &p.Anio, &p.Mes, &p.Descripcion, &p.Estado)
		if err == nil {
			lista = append(lista, p)
		}
	}
	return lista, nil
}

// Crear inserta una nueva cabecera mensual
func (r *PlanillaRepository) Crear(p *models.Planilla) error {
	query := `
		INSERT INTO planillas (tenant_id, anio, mes, descripcion, estado)
		VALUES ($1, $2, $3, $4, 'BORRADOR') RETURNING id
	`
	return r.db.QueryRow(query, p.TenantID, p.Anio, p.Mes, p.Descripcion).Scan(&p.ID)
}

// ObtenerPeriodoPlanilla extrae el año y mes a procesar
func (r *PlanillaRepository) ObtenerPeriodoPlanilla(planillaID int, tenantID int) (int, int, error) {
	var anio, mes int
	query := `SELECT anio, mes FROM planillas WHERE id = $1 AND tenant_id = $2`
	err := r.db.QueryRow(query, planillaID, tenantID).Scan(&anio, &mes)
	return anio, mes, err
}

// ObtenerParametrosGlobales extrae todas las variables de ley vigentes para ese mes/año
func (r *PlanillaRepository) ObtenerParametrosGlobales(anio int, mes int) (map[string]float64, error) {
	parametros := make(map[string]float64)
	query := `
		SELECT clave, valor FROM parametros_globales 
		WHERE fecha_desde <= (make_date($1, $2, 1) + interval '1 month' - interval '1 day')::date
		  AND (fecha_hasta IS NULL OR fecha_hasta >= make_date($1, $2, 1)::date)
	`
	rows, err := r.db.Query(query, anio, mes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var clave string
		var valor float64
		rows.Scan(&clave, &valor)
		parametros[clave] = valor
	}
	return parametros, nil
}

// ObtenerContratosActivosPlanilla busca a todos los que trabajaron en ese mes
func (r *PlanillaRepository) ObtenerContratosActivosPlanilla(tenantID int, anio int, mes int) ([]models.ContratoPlanilla, error) {
	query := `
		SELECT c.id, c.puesto_id, rl.codigo
		FROM contratos c
		INNER JOIN puestos p ON c.puesto_id = p.id
		INNER JOIN regimenes_laborales rl ON p.regimen_id = rl.id
		WHERE c.tenant_id = $1 AND c.activo = true
		  AND c.fecha_inicio <= (make_date($2, $3, 1) + interval '1 month' - interval '1 day')::date
		  AND (c.fecha_fin IS NULL OR c.fecha_fin >= make_date($2, $3, 1)::date)
	`
	rows, err := r.db.Query(query, tenantID, anio, mes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.ContratoPlanilla
	for rows.Next() {
		var c models.ContratoPlanilla
		rows.Scan(&c.ID, &c.PuestoID, &c.Regimen)
		lista = append(lista, c)
	}
	return lista, nil
}

// ObtenerConceptosPuesto trae la estructura de costos de una plaza específica
func (r *PlanillaRepository) ObtenerConceptosPuesto(puestoID int) ([]models.ConceptoPlanilla, error) {
	query := `
		SELECT pc.concepto_tenant_id, cm.id, cm.codigo, cm.tipo, pc.monto, ct.frecuencia_meses, ct.es_extraordinario
		FROM puesto_conceptos pc
		INNER JOIN conceptos_tenant ct ON pc.concepto_tenant_id = ct.id
		INNER JOIN conceptos_maestros cm ON ct.concepto_id = cm.id
		WHERE pc.puesto_id = $1 AND pc.activo = true
	`
	rows, err := r.db.Query(query, puestoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.ConceptoPlanilla
	for rows.Next() {
		var cp models.ConceptoPlanilla
		var m sql.NullFloat64
		rows.Scan(&cp.TenantID, &cp.MaestroID, &cp.MaestroCodigo, &cp.Tipo, &m, &cp.Frecuencia, &cp.EsExtraordinario)
		if m.Valid {
			cp.Monto = m.Float64
		}
		lista = append(lista, cp)
	}
	return lista, nil
}

// GuardarPlanillaCalculada recibe los cálculos en memoria y los inserta en bloque
func (r *PlanillaRepository) GuardarPlanillaCalculada(planillaID int, boletas []models.BoletaResultado) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	// 1. Limpieza de reprocesamientos (Rerun)
	_, err = tx.Exec(`DELETE FROM planilla_detalles WHERE planilla_id = $1`, planillaID)
	if err != nil {
		tx.Rollback()
		return err
	}

	// 2. Guardado en bloque
	for _, boleta := range boletas {
		var detalleID int

		// Guardamos la cabecera de la boleta
		err = tx.QueryRow(`
			INSERT INTO planilla_detalles (planilla_id, contrato_id, total_ingresos, total_retenciones, total_aportes, neto_pagar) 
			VALUES ($1, $2, $3, $4, $5, $6) RETURNING id
		`, planillaID, boleta.ContratoID, boleta.TotalIngresos, boleta.TotalRetenciones, boleta.TotalAportes, boleta.NetoPagar).Scan(&detalleID)

		if err != nil {
			tx.Rollback()
			return err
		}

		// Guardamos el detalle (los rubros)
		for _, concepto := range boleta.LineasConceptos {
			_, err = tx.Exec(`
				INSERT INTO planilla_conceptos (planilla_detalle_id, concepto_tenant_id, tipo_concepto, monto)
				VALUES ($1, $2, $3, $4)
			`, detalleID, concepto.ConceptoTenantID, concepto.TipoConcepto, concepto.Monto)

			if err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	return tx.Commit()
}

// ObtenerDetalles trae la lista de boletas generadas en una planilla
func (r *PlanillaRepository) ObtenerDetalles(planillaID int, tenantID int) ([]models.PlanillaDetalle, error) {
	query := `
		SELECT d.id, d.planilla_id, d.contrato_id, d.total_ingresos, d.total_retenciones, d.total_aportes, d.neto_pagar,
		       t.apellido_paterno || ' ' || t.apellido_materno || ', ' || t.nombres AS trabajador_nombre,
		       t.numero_documento AS trabajador_doc,
		       p.nombre AS puesto_nombre
		FROM planilla_detalles d
		INNER JOIN planillas pl ON d.planilla_id = pl.id
		INNER JOIN contratos c ON d.contrato_id = c.id
		INNER JOIN trabajadores t ON c.trabajador_id = t.id
		INNER JOIN puestos p ON c.puesto_id = p.id
		WHERE d.planilla_id = $1 AND pl.tenant_id = $2
		ORDER BY t.apellido_paterno ASC
	`
	rows, err := r.db.Query(query, planillaID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.PlanillaDetalle
	for rows.Next() {
		var d models.PlanillaDetalle
		err := rows.Scan(&d.ID, &d.PlanillaID, &d.ContratoID, &d.TotalIngresos, &d.TotalRetenciones, &d.TotalAportes, &d.NetoPagar,
			&d.TrabajadorNombre, &d.TrabajadorDoc, &d.PuestoNombre)
		if err == nil {
			lista = append(lista, d)
		}
	}
	return lista, nil
}

// ObtenerRetencionesPrevias calcula la suma de retenciones según el cuadro de SUNAT
func (r *PlanillaRepository) ObtenerRetencionesPrevias(contratoID int, anio int, mesActual int) (float64, error) {
	// Definimos hasta qué mes debemos sumar según el cuadro que me pasaste
	mesHasta := 0
	switch {
	case mesActual >= 1 && mesActual <= 3:
		return 0.00, nil // Nada que sumar
	case mesActual == 4:
		mesHasta = 3
	case mesActual >= 5 && mesActual <= 7:
		mesHasta = 4
	case mesActual == 8:
		mesHasta = 7
	case mesActual >= 9 && mesActual <= 11:
		mesHasta = 8
	case mesActual == 12:
		mesHasta = 11
	}

	// Consulta SQL para sumar las retenciones de 5ta (Maestro 0605) de los meses previos
	query := `
		SELECT COALESCE(SUM(pc.monto), 0)
		FROM planilla_conceptos pc
		INNER JOIN planilla_detalles pd ON pc.planilla_detalle_id = pd.id
		INNER JOIN planillas p ON pd.planilla_id = p.id
		INNER JOIN conceptos_tenant ct ON pc.concepto_tenant_id = ct.id
		INNER JOIN conceptos_maestros cm ON ct.concepto_id = cm.id
		WHERE pd.contrato_id = $1 
		  AND p.anio = $2 
		  AND p.mes >= 1 AND p.mes <= $3
		  AND cm.codigo = '0605' -- Código SUNAT para Renta de 5ta
	`

	var suma float64
	err := r.db.QueryRow(query, contratoID, anio, mesHasta).Scan(&suma)
	return suma, err
}

// clasificarIngresos5ta procesa en memoria los montos de Remuneración Mensual, No Mensual y Extraordinaria
func (r *PlanillaRepository) clasificarIngresos5ta(conceptosPuesto []ConceptoTemp, derivados5ta []int, mesActual int) (float64, float64, float64) {
	mensual := 0.00
	noMensual := 0.00
	extraDelMes := 0.00
	mesActualStr := strconv.Itoa(mesActual)

	// Convertimos el slice de IDs derivados a un mapa para búsquedas ultra-rápidas
	afectosA5ta := make(map[int]bool)
	for _, id := range derivados5ta {
		afectosA5ta[id] = true
	}

	for _, cp := range conceptosPuesto {
		// Si este ingreso no es base imponible para 5ta categoría, lo saltamos
		if !afectosA5ta[cp.MaestroID] {
			continue
		}

		// A. INGRESOS EXTRAORDINARIOS
		if cp.EsExtraordinario {
			aplicaEsteMes := false
			for _, mStr := range strings.Split(cp.Frecuencia, ",") {
				if strings.TrimSpace(mStr) == mesActualStr {
					aplicaEsteMes = true
					break
				}
			}
			if aplicaEsteMes {
				extraDelMes += cp.Monto
			}
		} else {
			// B. INGRESOS ORDINARIOS
			mesesArray := strings.Split(cp.Frecuencia, ",")

			if len(mesesArray) >= 12 {
				// Ordinario Mensual (Se paga todo el año)
				mensual += cp.Monto
			} else {
				// Ordinario No Mensual (Se paga en ciertos meses, deducimos los ya pasados)
				vecesRestantes := 0
				for _, mStr := range mesesArray {
					mInt, _ := strconv.Atoi(strings.TrimSpace(mStr))
					if mInt >= mesActual {
						vecesRestantes++
					}
				}
				noMensual += (cp.Monto * float64(vecesRestantes))
			}
		}
	}

	return mensual, noMensual, extraDelMes
}

// ObtenerAfectacionesGlobales trae el mapa completo de relaciones Base -> Derivados
func (r *PlanillaRepository) ObtenerAfectacionesGlobales() (map[int][]int, error) {
	query := `SELECT concepto_base_id, concepto_derivado_id FROM conceptos_afectaciones`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Creamos un diccionario: [ID de EsSalud] = [ID Sueldo, ID Bono...]
	mapa := make(map[int][]int)
	for rows.Next() {
		var base, derivado int
		rows.Scan(&base, &derivado)
		mapa[base] = append(mapa[base], derivado)
	}
	return mapa, nil
}
