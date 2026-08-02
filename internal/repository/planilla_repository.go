package repository

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"planilla-rgm/internal/models"
	"strconv"
	"strings"

	"github.com/lib/pq"
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
	if err := rows.Err(); err != nil {
		return nil, err
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return parametros, nil
}

func (r *PlanillaRepository) GetDB() *sql.DB {
	return r.db
}

// ObtenerContratosActivosPlanilla busca a todos los que trabajaron en ese mes y trae su info de AFP desde el trabajador, así como los datos históricos para snapshot
func (r *PlanillaRepository) ObtenerContratosActivosPlanilla(tenantID int, anio int, mes int) ([]models.ContratoPlanilla, error) {
	query := `
		SELECT c.id, c.tenant_id, c.puesto_id, p.regimen_id, rl.codigo, 
		       COALESCE(t.regimen_pensionario, 'ONP'), COALESCE(t.afp_id, 0), COALESCE(t.afp_tipo_comision, ''),
		       c.fecha_inicio, c.fecha_fin,
		       t.apellido_paterno || ' ' || t.apellido_materno || ', ' || t.nombres AS trabajador_nombre_completo,
		       t.numero_documento AS trabajador_numero_documento,
		       p.nombre AS puesto_nombre,
		       COALESCE(p.codigo_airhsp, '') AS puesto_codigo_airhsp,
		       COALESCE(o.documento_aprobacion, 'N/A') AS organigrama_documento_aprobacion,
		       COALESCE(uo.nombre, 'Sin Unidad') AS unidad_organica_nombre,
		       COALESCE(uo.tipo, 'N/A') AS unidad_organica_tipo,
		       p.sueldo_presupuestado AS sueldo_basico_historico,
		       p.meta_id, p.fuente_rubro_id
		FROM contratos c
		INNER JOIN trabajadores t ON c.trabajador_id = t.id
		INNER JOIN puestos p ON c.puesto_id = p.id
		INNER JOIN regimenes_laborales rl ON p.regimen_id = rl.id
		LEFT JOIN unidades_organicas uo ON p.unidad_organica_id = uo.id
		LEFT JOIN organigramas o ON uo.organigrama_id = o.id
		WHERE c.tenant_id = $1 
		  AND p.es_dietario = false
		  AND c.fecha_inicio <= (make_date($2, $3, 1) + interval '1 month' - interval '1 day')::date
		  AND (
		      -- Condición A: El contrato sigue activo (y su fecha fin, si existe, es futura o del mes actual)
		      (c.activo = true AND (c.fecha_fin IS NULL OR c.fecha_fin >= make_date($2, $3, 1)::date))
		      OR 
		      -- Condición B: El contrato está inactivo, pero cesó dentro de este mismo mes
		      (c.activo = false AND c.fecha_fin IS NOT NULL 
		       AND c.fecha_fin >= make_date($2, $3, 1)::date 
		       AND c.fecha_fin <= (make_date($2, $3, 1) + interval '1 month' - interval '1 day')::date)
		  )
	`
	rows, err := r.db.Query(query, tenantID, anio, mes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.ContratoPlanilla
	for rows.Next() {
		var c models.ContratoPlanilla
		var metaID, fuenteRubroID sql.NullInt64
		err := rows.Scan(&c.ID, &c.TenantID, &c.PuestoID, &c.RegimenID, &c.Regimen, &c.RegimenPensionario, &c.AfpID, &c.AfpTipoComision, &c.FechaInicio, &c.FechaFin,
			&c.TrabajadorNombreCompleto, &c.TrabajadorNumeroDocumento, &c.PuestoNombre, &c.PuestoCodigoAirhsp,
			&c.OrganigramaDocumentoAprobacion, &c.UnidadOrganicaNombre, &c.UnidadOrganicaTipo, &c.SueldoBasicoHistorico,
			&metaID, &fuenteRubroID)
		if err != nil {
			return nil, err
		}
		if metaID.Valid {
			v := int(metaID.Int64)
			c.MetaID = &v
		}
		if fuenteRubroID.Valid {
			v := int(fuenteRubroID.Int64)
			c.FuenteRubroID = &v
		}
		lista = append(lista, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return lista, nil
}

// ObtenerConceptosPuesto trae la estructura de costos de una plaza específica
func (r *PlanillaRepository) ObtenerConceptosPuesto(puestoID int) ([]models.ConceptoPlanilla, error) {
	query := `
		SELECT pc.concepto_tenant_id, cm.id, cm.codigo_interno, cm.codigo, cm.tipo, pc.monto, ct.frecuencia_meses, ct.es_extraordinario
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
		rows.Scan(&cp.TenantID, &cp.MaestroID, &cp.CodigoInterno, &cp.CodigoSunat, &cp.Tipo, &m, &cp.Frecuencia, &cp.EsExtraordinario)
		if m.Valid {
			cp.Monto = m.Float64
		}
		lista = append(lista, cp)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return lista, nil
}

// GuardarPlanillaCalculada recibe los cálculos en memoria y los inserta en bloque con snapshot de inmutabilidad
func (r *PlanillaRepository) GuardarPlanillaCalculada(planillaID int, boletas []models.BoletaResultado) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	// 1. LIMPIEZA
	_, err = tx.Exec(`DELETE FROM planilla_detalles WHERE planilla_id = $1`, planillaID)
	if err != nil {
		tx.Rollback()
		return err
	}

	// 2. PREPARAR SENTENCIAS
	stmtDetalle, _ := tx.Prepare(`
		INSERT INTO planilla_detalles (
			planilla_id, contrato_id, total_ingresos, total_retenciones, total_aportes, neto_pagar,
			trabajador_nombre_completo, trabajador_numero_documento, puesto_codigo_airhsp, puesto_nombre,
			organigrama_documento_aprobacion, unidad_organica_nombre, unidad_organica_tipo, sueldo_basico_historico
		) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14) RETURNING id`)

	stmtConcepto, _ := tx.Prepare(`
		INSERT INTO planilla_conceptos (planilla_detalle_id, concepto_tenant_id, maestro_id, tipo_concepto, monto, codigo_sunat, nombre_en_boleta, meta_id, fuente_rubro_id) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`)

	// 3. BUCLE DE GUARDADO
	for _, b := range boletas {
		var detalleID int
		err := stmtDetalle.QueryRow(
			planillaID, b.ContratoID, b.TotalIngresos, b.TotalRetenciones, b.TotalAportes, b.NetoPagar,
			b.TrabajadorNombreCompleto, b.TrabajadorNumeroDocumento, b.PuestoCodigoAirhsp, b.PuestoNombre,
			b.OrganigramaDocumentoAprobacion, b.UnidadOrganicaNombre, b.UnidadOrganicaTipo, b.SueldoBasicoHistorico,
		).Scan(&detalleID)
		if err != nil {
			log.Println("Error insertando detalle:", err)
			tx.Rollback()
			return err
		}

		for _, linea := range b.LineasConceptos {
			var tenantIDVal interface{}
			if linea.ConceptoTenantID != nil && *linea.ConceptoTenantID > 0 {
				tenantIDVal = *linea.ConceptoTenantID
			} else {
				tenantIDVal = nil
			}
			var metaIDVal interface{}
			if linea.MetaID != nil && *linea.MetaID > 0 {
				metaIDVal = *linea.MetaID
			} else {
				metaIDVal = nil
			}
			var rubroIDVal interface{}
			if linea.FuenteRubroID != nil && *linea.FuenteRubroID > 0 {
				rubroIDVal = *linea.FuenteRubroID
			} else {
				rubroIDVal = nil
			}
			_, err = stmtConcepto.Exec(detalleID, tenantIDVal, linea.MaestroID, linea.TipoConcepto, linea.Monto, linea.CodigoSunat, linea.NombreEnBoleta, metaIDVal, rubroIDVal)
			if err != nil {
				log.Println("Error insertando concepto:", err)
				tx.Rollback()
				return err
			}
		}

		for _, ocID := range b.OcurrenciasProcesadas {
			_, err = tx.Exec(`UPDATE ocurrencias_asistencia SET procesado = true, planilla_id_descuento = $1 WHERE id = $2`, planillaID, ocID)
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
		       COALESCE(p.nombre, 'Sin Plaza Asignada') AS puesto_nombre
		FROM planilla_detalles d
		INNER JOIN planillas pl ON d.planilla_id = pl.id
		INNER JOIN contratos c ON d.contrato_id = c.id
		INNER JOIN trabajadores t ON c.trabajador_id = t.id
		LEFT JOIN puestos p ON c.puesto_id = p.id
		WHERE d.planilla_id = $1 AND pl.tenant_id = $2
		ORDER BY t.apellido_paterno ASC, t.apellido_materno ASC, t.nombres ASC
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return lista, nil
}

// ObtenerDetallePorID obtiene la cabecera y los conceptos calculados para una boleta individual
func (r *PlanillaRepository) ObtenerDetallePorID(detalleID int, tenantID int) (*models.PlanillaDetalle, error) {
	query := `
		SELECT d.id, d.planilla_id, d.contrato_id, d.total_ingresos, d.total_retenciones, d.total_aportes, d.neto_pagar,
		       t.apellido_paterno || ' ' || t.apellido_materno || ', ' || t.nombres AS trabajador_nombre,
		       t.numero_documento AS trabajador_doc,
		       COALESCE(p.nombre, 'Sin Plaza Asignada') AS puesto_nombre
		FROM planilla_detalles d
		INNER JOIN planillas pl ON d.planilla_id = pl.id
		INNER JOIN contratos c ON d.contrato_id = c.id
		INNER JOIN trabajadores t ON c.trabajador_id = t.id
		LEFT JOIN puestos p ON c.puesto_id = p.id
		WHERE d.id = $1 AND pl.tenant_id = $2
	`
	var d models.PlanillaDetalle
	err := r.db.QueryRow(query, detalleID, tenantID).Scan(
		&d.ID, &d.PlanillaID, &d.ContratoID, &d.TotalIngresos, &d.TotalRetenciones, &d.TotalAportes, &d.NetoPagar,
		&d.TrabajadorNombre, &d.TrabajadorDoc, &d.PuestoNombre,
	)
	if err != nil {
		return nil, err
	}

	conceptos, err := r.ObtenerConceptosPorDetalle(detalleID)
	if err == nil {
		d.Conceptos = conceptos
	}
	return &d, nil
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return mapa, nil
}

// ObtenerOcurrenciasParaProcesar trae las faltas libres O las que ya pertenecen a esta planilla
func (r *PlanillaRepository) ObtenerOcurrenciasParaProcesar(tenantID int, planillaID int) (map[int][]models.OcurrenciaAsistencia, error) {
	query := `
		SELECT oa.id, oa.contrato_id, oa.tipo, oa.cantidad
		FROM ocurrencias_asistencia oa
		INNER JOIN contratos c ON oa.contrato_id = c.id
		WHERE c.tenant_id = $1 
          AND (oa.procesado = false OR oa.planilla_id_descuento = $2)
	`
	rows, err := r.db.Query(query, tenantID, planillaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	mapa := make(map[int][]models.OcurrenciaAsistencia)
	for rows.Next() {
		var contratoID int
		var oc models.OcurrenciaAsistencia
		rows.Scan(&oc.ID, &contratoID, &oc.Tipo, &oc.Cantidad)
		mapa[contratoID] = append(mapa[contratoID], oc)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return mapa, nil
}

// ObtenerTasasAFPMes trae el diccionario de comisiones de todas las AFPs para el mes de cálculo
func (r *PlanillaRepository) ObtenerTasasAFPMes(anio int, mes int) (map[int]models.TasasAFP, error) {
	query := `SELECT afp_id, aporte_obligatorio, comision_flujo, comision_mixta_flujo, prima_seguro 
	          FROM afp_tasas_mensuales WHERE anio = $1 AND mes = $2`

	rows, err := r.db.Query(query, anio, mes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	mapa := make(map[int]models.TasasAFP)
	for rows.Next() {
		var afpID int
		var t models.TasasAFP
		rows.Scan(&afpID, &t.Aporte, &t.Flujo, &t.Mixta, &t.Prima)
		mapa[afpID] = t
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return mapa, nil
}

// ObtenerMapaCodigosID crea un diccionario para traducir Códigos SUNAT a IDs internos
func (r *PlanillaRepository) ObtenerMapaCodigosID() (map[string]int, error) {
	query := `SELECT codigo_interno, id FROM conceptos_maestros WHERE activo = true`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	mapa := make(map[string]int)
	for rows.Next() {
		var codigo string
		var id int
		if err := rows.Scan(&codigo, &id); err != nil {
			return nil, err
		}
		mapa[codigo] = id
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return mapa, nil
}

// ObtenerConceptosPorPuestoMasivo trae las estructuras de costos de todos los puestos involucrados
func (r *PlanillaRepository) ObtenerConceptosPorPuestoMasivo(puestoIDs []int) (map[int][]models.ConceptoPlanilla, error) {
	if len(puestoIDs) == 0 {
		return make(map[int][]models.ConceptoPlanilla), nil
	}

	query := `
		SELECT pc.puesto_id, pc.concepto_tenant_id, cm.id, cm.codigo_interno, cm.codigo, cm.tipo, pc.monto,
		       ct.frecuencia_meses, ct.es_extraordinario, COALESCE(cm.parent_id, 0), ct.nombre_personalizado
		FROM puesto_conceptos pc
		INNER JOIN conceptos_tenant ct ON pc.concepto_tenant_id = ct.id
		INNER JOIN conceptos_maestros cm ON ct.concepto_id = cm.id
		WHERE pc.puesto_id = ANY($1) AND pc.activo = true
	`
	rows, err := r.db.Query(query, pq.Array(puestoIDs))
	if err != nil {
		log.Println("Error SQL ObtenerConceptosPorPuestoMasivo:", err)
		return nil, err
	}
	defer rows.Close()

	mapa := make(map[int][]models.ConceptoPlanilla)
	for rows.Next() {
		var pID int
		var cp models.ConceptoPlanilla
		var m sql.NullFloat64 // Usamos esto por si el monto está vacío en la BD

		err := rows.Scan(&pID, &cp.TenantID, &cp.MaestroID, &cp.CodigoInterno, &cp.CodigoSunat, &cp.Tipo, &m,
			&cp.Frecuencia, &cp.EsExtraordinario, &cp.ParentID, &cp.Nombre)
		if err != nil {
			log.Println("Error Scan ObtenerConceptosPorPuestoMasivo:", err)
			continue
		}

		if m.Valid {
			cp.Monto = m.Float64
		}
		mapa[pID] = append(mapa[pID], cp)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return mapa, nil
}

// ObtenerConceptosPorContratoMasivo trae las estructuras de costos de todos los contratos involucrados,
// extrayendo de puesto_conceptos para los activos y de contrato_conceptos_snapshot para los cesados/inactivos
func (r *PlanillaRepository) ObtenerConceptosPorContratoMasivo(contratoIDs []int) (map[int][]models.ConceptoPlanilla, error) {
	if len(contratoIDs) == 0 {
		return make(map[int][]models.ConceptoPlanilla), nil
	}

	query := `
		-- Conceptos de contratos activos obtenidos de puesto_conceptos
		SELECT c.id AS contrato_id, pc.concepto_tenant_id, cm.id AS maestro_id, 
		       cm.codigo_interno, cm.codigo AS codigo_sunat, cm.tipo, pc.monto,
		       ct.frecuencia_meses, ct.es_extraordinario, COALESCE(cm.parent_id, 0) AS parent_id, 
		       ct.nombre_personalizado
		FROM contratos c
		INNER JOIN puesto_conceptos pc ON c.puesto_id = pc.puesto_id AND pc.activo = true
		INNER JOIN conceptos_tenant ct ON pc.concepto_tenant_id = ct.id
		INNER JOIN conceptos_maestros cm ON ct.concepto_id = cm.id
		WHERE c.id = ANY($1) AND c.activo = true

		UNION ALL

		-- Conceptos de contratos inactivos obtenidos de contrato_conceptos_snapshot
		SELECT ccs.contrato_id, ccs.concepto_tenant_id, cm.id AS maestro_id, 
		       cm.codigo_interno, cm.codigo AS codigo_sunat, cm.tipo, ccs.monto,
		       ct.frecuencia_meses, ct.es_extraordinario, COALESCE(cm.parent_id, 0) AS parent_id, 
		       ct.nombre_personalizado
		FROM contrato_conceptos_snapshot ccs
		INNER JOIN conceptos_tenant ct ON ccs.concepto_tenant_id = ct.id
		INNER JOIN conceptos_maestros cm ON ct.concepto_id = cm.id
		WHERE ccs.contrato_id = ANY($1)
	`
	rows, err := r.db.Query(query, pq.Array(contratoIDs))
	if err != nil {
		log.Println("Error SQL ObtenerConceptosPorContratoMasivo:", err)
		return nil, err
	}
	defer rows.Close()

	mapa := make(map[int][]models.ConceptoPlanilla)
	for rows.Next() {
		var contratoID int
		var cp models.ConceptoPlanilla
		var m sql.NullFloat64

		err := rows.Scan(&contratoID, &cp.TenantID, &cp.MaestroID, &cp.CodigoInterno, &cp.CodigoSunat, &cp.Tipo, &m,
			&cp.Frecuencia, &cp.EsExtraordinario, &cp.ParentID, &cp.Nombre)
		if err != nil {
			log.Println("Error Scan ObtenerConceptosPorContratoMasivo:", err)
			continue
		}

		if m.Valid {
			cp.Monto = m.Float64
		}
		mapa[contratoID] = append(mapa[contratoID], cp)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return mapa, nil
}

// ObtenerRetencionesPreviasMasivo trae el acumulado de Renta de 5ta respetando los cortes de SUNAT
func (r *PlanillaRepository) ObtenerRetencionesPreviasMasivo(contratoIDs []int, anio int, mesActual int) (map[int]float64, error) {
	if len(contratoIDs) == 0 {
		return make(map[int]float64), nil
	}

	// 1. LÓGICA TRIBUTARIA SUNAT: Determinar hasta qué mes sumar (Mes de Corte)
	mesCorte := 0
	switch mesActual {
	case 1, 2, 3:
		// Enero, Febrero y Marzo: No se deducen retenciones previas.
		// Retornamos un mapa vacío inmediatamente (¡Ahorramos una consulta SQL!)
		return make(map[int]float64), nil
	case 4:
		mesCorte = 3 // Abril: Suma Enero a Marzo
	case 5, 6, 7:
		mesCorte = 4 // Mayo a Julio: Suma Enero a Abril
	case 8:
		mesCorte = 7 // Agosto: Suma Enero a Julio
	case 9, 10, 11:
		mesCorte = 8 // Septiembre a Noviembre: Suma Enero a Agosto
	case 12:
		mesCorte = 11 // Diciembre: Suma Enero a Noviembre
	}

	// 2. CONSULTA SQL AJUSTADA
	// Usamos p.mes <= $3 (menor o IGUAL al mes de corte)
	query := `
		SELECT pd.contrato_id, SUM(pc.monto)
		FROM planilla_conceptos pc
		INNER JOIN planilla_detalles pd ON pc.planilla_detalle_id = pd.id
		INNER JOIN planillas p ON pd.planilla_id = p.id
		INNER JOIN conceptos_maestros cm ON pc.maestro_id = cm.id
		WHERE pd.contrato_id = ANY($1) 
		  AND p.anio = $2 
		  AND p.mes <= $3 
		  AND cm.codigo = '0605' 
		GROUP BY pd.contrato_id
	`
	rows, err := r.db.Query(query, pq.Array(contratoIDs), anio, mesCorte)
	if err != nil {
		log.Println("Error SQL ObtenerRetencionesPreviasMasivo:", err)
		return nil, err
	}
	defer rows.Close()

	mapa := make(map[int]float64)
	for rows.Next() {
		var cID int
		var suma float64
		rows.Scan(&cID, &suma)
		mapa[cID] = suma
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return mapa, nil
}

// ObtenerIngresosPreviosMasivo trae el acumulado de ingresos reales percibidos en el año
func (r *PlanillaRepository) ObtenerIngresosPreviosMasivo(contratoIDs []int, anio int, mesActual int) (map[int]float64, error) {
	if len(contratoIDs) == 0 {
		return make(map[int]float64), nil
	}

	query := `
		SELECT pd.contrato_id, SUM(pc.monto)
		FROM planilla_conceptos pc
		INNER JOIN planilla_detalles pd ON pc.planilla_detalle_id = pd.id
		INNER JOIN planillas p ON pd.planilla_id = p.id
		WHERE pd.contrato_id = ANY($1) 
		  AND p.anio = $2 
		  AND p.mes < $3 
		  AND pc.tipo_concepto = 'INGRESO'
		GROUP BY pd.contrato_id
	`
	rows, err := r.db.Query(query, pq.Array(contratoIDs), anio, mesActual)
	if err != nil {
		log.Println("Error SQL ObtenerIngresosPreviosMasivo:", err)
		return nil, err
	}
	defer rows.Close()

	mapa := make(map[int]float64)
	for rows.Next() {
		var cID int
		var suma float64
		rows.Scan(&cID, &suma)
		mapa[cID] = suma
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return mapa, nil
}

// ObtenerDatosParaReporte extrae absolutamente todo el detalle de una planilla
// agrupando los conceptos (Ingresos, Retenciones, Aportes) por trabajador, leyendo desde las tablas históricas des-normalizadas.
func (r *PlanillaRepository) ObtenerDatosParaReporte(planillaID int, tenantID int) (*models.DatosReportePlanilla, error) {
	var reporte models.DatosReportePlanilla

	// 1. Obtener Cabecera (Datos de la Muni y de la Planilla)
	queryCabecera := `
		SELECT t.nombre, t.ruc, COALESCE(t.logo_url, ''), p.anio, p.mes, p.descripcion, COALESCE(p.estado, '')
		FROM planillas p
		INNER JOIN tenants t ON p.tenant_id = t.id
		WHERE p.id = $1 AND p.tenant_id = $2
	`
	err := r.db.QueryRow(queryCabecera, planillaID, tenantID).Scan(
		&reporte.TenantNombre, &reporte.TenantRUC, &reporte.TenantLogoURL,
		&reporte.PlanillaAnio, &reporte.PlanillaMes, &reporte.PlanillaDesc,
		&reporte.PlanillaEstado,
	)
	if err != nil {
		return nil, err
	}

	// 2. Obtener los Trabajadores (Directo desde la boleta inmutable y JOIN con el régimen laboral y trabajador)
	queryDetalles := `
		SELECT 
			pd.id, 
			pd.trabajador_numero_documento, 
			pd.trabajador_nombre_completo,
			pd.puesto_nombre,
			COALESCE(rl.descripcion, 'Sin Régimen'),
			pd.total_ingresos, pd.total_retenciones, pd.total_aportes, pd.neto_pagar,
			COALESCE(t.sexo, '-'),
			COALESCE(TO_CHAR(t.fecha_nacimiento, 'DD/MM/YYYY'), '-'),
			COALESCE(TO_CHAR(t.fecha_ingreso, 'DD/MM/YYYY'), '-'),
			COALESCE(TO_CHAR(t.fecha_cese, 'DD/MM/YYYY'), '-'),
			COALESCE(t.direccion, '-'),
			COALESCE(t.regimen_pensionario, 'ONP'),
			COALESCE(a.nombre, '-'),
			COALESCE(t.cuspp, '-')
		FROM planilla_detalles pd
		LEFT JOIN contratos c ON pd.contrato_id = c.id
		LEFT JOIN trabajadores t ON c.trabajador_id = t.id
		LEFT JOIN afps a ON t.afp_id = a.id
		LEFT JOIN puestos p ON c.puesto_id = p.id
		LEFT JOIN regimenes_laborales rl ON p.regimen_id = rl.id
		WHERE pd.planilla_id = $1
		ORDER BY pd.trabajador_nombre_completo ASC
	`
	rowsDet, err := r.db.Query(queryDetalles, planillaID)
	if err != nil {
		return nil, err
	}
	defer rowsDet.Close()

	// Usamos un mapa para ubicar rápidamente la boleta cuando leamos los conceptos
	mapaBoletas := make(map[int]*models.BoletaReporte)

	for rowsDet.Next() {
		b := &models.BoletaReporte{}
		err := rowsDet.Scan(
			&b.DetalleID, &b.TrabajadorDoc, &b.TrabajadorNombre, &b.Cargo, &b.Regimen,
			&b.TotalIngresos, &b.TotalRetenciones, &b.TotalAportes, &b.NetoPagar,
			&b.Sexo, &b.FechaNacimiento, &b.FechaIngreso, &b.FechaCese, &b.Direccion,
			&b.RegimenPensionario, &b.AfpNombre, &b.Cuspp,
		)
		if err != nil {
			return nil, err
		}

		// Sumamos a los totales generales de la Municipalidad
		reporte.TotalIngresos += b.TotalIngresos
		reporte.TotalRetenciones += b.TotalRetenciones
		reporte.TotalAportes += b.TotalAportes
		reporte.TotalNeto += b.NetoPagar

		mapaBoletas[b.DetalleID] = b
		reporte.Boletas = append(reporte.Boletas, b) // Mantenemos el orden alfabético
	}
	if err := rowsDet.Err(); err != nil {
		return nil, err
	}

	// 3. Obtener TODOS los conceptos históricos inmutables
	queryConceptos := `
		SELECT 
			pc.planilla_detalle_id, pc.nombre_en_boleta, pc.tipo_concepto, pc.monto
		FROM planilla_conceptos pc
		WHERE pc.planilla_detalle_id IN (SELECT id FROM planilla_detalles WHERE planilla_id = $1)
		ORDER BY pc.monto DESC -- Opcional: para que los montos más altos salgan primero
	`
	rowsConc, err := r.db.Query(queryConceptos, planillaID)
	if err != nil {
		return nil, err
	}
	defer rowsConc.Close()

	for rowsConc.Next() {
		var detalleID int
		var nombre, tipo string
		var monto float64
		rowsConc.Scan(&detalleID, &nombre, &tipo, &monto)

		// Si el monto es 0, no lo imprimimos para ahorrar espacio en el PDF
		if monto <= 0 {
			continue
		}

		concepto := models.ConceptoReporte{Nombre: nombre, Monto: monto}

		// Apilamos en la columna correspondiente según tu idea de diseño
		if boleta, existe := mapaBoletas[detalleID]; existe {
			tipoNormalizado := strings.ToUpper(strings.TrimSpace(tipo))
			switch tipoNormalizado {
			case "INGRESO":
				boleta.Ingresos = append(boleta.Ingresos, concepto)
			case "RETENCION":
				boleta.Retenciones = append(boleta.Retenciones, concepto)
			case "APORTE":
				boleta.Aportes = append(boleta.Aportes, concepto)
			}
		}
	}
	if err := rowsConc.Err(); err != nil {
		return nil, err
	}

	return &reporte, nil
}

// CambiarEstado actualiza el estado de la planilla (ej: de BORRADOR a CERRADA)
func (r *PlanillaRepository) CambiarEstado(planillaID int, tenantID int, nuevoEstado string) error {
	query := `
		UPDATE planillas 
		SET estado = $1, updated_at = CURRENT_TIMESTAMP 
		WHERE id = $2 AND tenant_id = $3
	`
	_, err := r.db.Exec(query, nuevoEstado, planillaID, tenantID)
	return err
}

// ObtenerEstado (Opcional, pero útil) para verificar antes de hacer operaciones críticas
func (r *PlanillaRepository) ObtenerEstado(planillaID int, tenantID int) (string, error) {
	var estado string
	query := `SELECT estado FROM planillas WHERE id = $1 AND tenant_id = $2`
	err := r.db.QueryRow(query, planillaID, tenantID).Scan(&estado)
	return estado, err
}

// ObtenerPorID obtiene una planilla por su ID y TenantID
func (r *PlanillaRepository) ObtenerPorID(planillaID int, tenantID int) (*models.Planilla, error) {
	var p models.Planilla
	query := `SELECT id, tenant_id, anio, mes, descripcion, estado FROM planillas WHERE id = $1 AND tenant_id = $2`
	err := r.db.QueryRow(query, planillaID, tenantID).Scan(&p.ID, &p.TenantID, &p.Anio, &p.Mes, &p.Descripcion, &p.Estado)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// ObtenerRucTenant obtiene el RUC de un tenant
func (r *PlanillaRepository) ObtenerRucTenant(tenantID int) (string, error) {
	var ruc string
	query := `SELECT ruc FROM tenants WHERE id = $1`
	err := r.db.QueryRow(query, tenantID).Scan(&ruc)
	return ruc, err
}

// ObtenerDatosPlameJornada obtiene los datos de jornada laboral para exportar a PLAME (.jor)
func (r *PlanillaRepository) ObtenerDatosPlameJornada(planillaID int, tenantID int) ([]models.PlameJornada, error) {
	query := `
		SELECT t.tipo_documento, t.numero_documento, COALESCE(SUM(CASE WHEN oa.tipo = 'INASISTENCIA' THEN oa.cantidad ELSE 0 END), 0) AS dias_inasistencia
		FROM planilla_detalles pd
		INNER JOIN contratos c ON pd.contrato_id = c.id
		INNER JOIN trabajadores t ON c.trabajador_id = t.id
		LEFT JOIN ocurrencias_asistencia oa ON oa.contrato_id = c.id AND oa.planilla_id_descuento = pd.planilla_id
		INNER JOIN planillas pl ON pd.planilla_id = pl.id
		WHERE pd.planilla_id = $1 AND pl.tenant_id = $2
		GROUP BY t.tipo_documento, t.numero_documento
		ORDER BY t.numero_documento
	`
	rows, err := r.db.Query(query, planillaID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.PlameJornada
	for rows.Next() {
		var j models.PlameJornada
		if err := rows.Scan(&j.TipoDocumento, &j.NumeroDocumento, &j.DiasInasistencia); err != nil {
			return nil, err
		}
		lista = append(lista, j)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return lista, nil
}

// ObtenerDatosPlameRemuneraciones obtiene los datos de remuneraciones para exportar a PLAME (.rem)
func (r *PlanillaRepository) ObtenerDatosPlameRemuneraciones(planillaID int, tenantID int) ([]models.PlameRemuneracion, error) {
	query := `
		SELECT t.tipo_documento, t.numero_documento, cm.codigo, pc.monto
		FROM planilla_conceptos pc
		INNER JOIN planilla_detalles pd ON pc.planilla_detalle_id = pd.id
		INNER JOIN contratos c ON pd.contrato_id = c.id
		INNER JOIN trabajadores t ON c.trabajador_id = t.id
		INNER JOIN conceptos_maestros cm ON pc.maestro_id = cm.id
		INNER JOIN planillas pl ON pd.planilla_id = pl.id
		WHERE pd.planilla_id = $1 AND pl.tenant_id = $2 AND cm.origen = 'sunat' AND pc.monto > 0
		ORDER BY t.numero_documento, cm.codigo
	`
	rows, err := r.db.Query(query, planillaID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.PlameRemuneracion
	for rows.Next() {
		var rem models.PlameRemuneracion
		if err := rows.Scan(&rem.TipoDocumento, &rem.NumeroDocumento, &rem.CodigoConcepto, &rem.Monto); err != nil {
			return nil, err
		}
		lista = append(lista, rem)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return lista, nil
}

// Eliminar borra la planilla, revierte las ocurrencias asociadas y limpia detalles y conceptos
func (r *PlanillaRepository) Eliminar(planillaID int, tenantID int) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	// 1. Validar que la planilla pertenezca al tenant y no esté CERRADA
	var estado string
	err = tx.QueryRow(`SELECT estado FROM planillas WHERE id = $1 AND tenant_id = $2`, planillaID, tenantID).Scan(&estado)
	if err != nil {
		tx.Rollback()
		return err
	}
	if estado == "CERRADA" {
		tx.Rollback()
		return errors.New("operación denegada: la planilla ya está CERRADA y no puede ser eliminada")
	}

	// 2. Revertir cambios en ocurrencias_asistencia
	_, err = tx.Exec(`UPDATE ocurrencias_asistencia SET procesado = false, planilla_id_descuento = NULL WHERE planilla_id_descuento = $1`, planillaID)
	if err != nil {
		tx.Rollback()
		return err
	}

	// 3. Eliminar la planilla (por cascade de foreign keys, se borran automáticamente planilla_detalles y planilla_conceptos)
	_, err = tx.Exec(`DELETE FROM planillas WHERE id = $1 AND tenant_id = $2`, planillaID, tenantID)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// ActualizarPresupuestoConceptos actualiza masiva o individualmente la meta y fuente_rubro de conceptos de planilla
func (r *PlanillaRepository) ActualizarPresupuestoConceptos(ctx context.Context, tenantID int, conceptosIDs []int, metaID *int, fuenteRubroID *int) error {
	if len(conceptosIDs) == 0 {
		return nil
	}

	var metaVal interface{}
	if metaID != nil && *metaID > 0 {
		metaVal = *metaID
	} else {
		metaVal = nil
	}

	var rubroVal interface{}
	if fuenteRubroID != nil && *fuenteRubroID > 0 {
		rubroVal = *fuenteRubroID
	} else {
		rubroVal = nil
	}

	query := `
		UPDATE planilla_conceptos
		SET meta_id = $1, fuente_rubro_id = $2
		WHERE id = ANY($3)
		  AND planilla_detalle_id IN (
			SELECT pd.id
			FROM planilla_detalles pd
			INNER JOIN planillas pl ON pd.planilla_id = pl.id
			WHERE pl.tenant_id = $4
		  )
	`
	_, err := r.db.ExecContext(ctx, query, metaVal, rubroVal, pq.Array(conceptosIDs), tenantID)
	return err
}

// ObtenerConceptosPorDetalle trae los conceptos de una boleta con meta_id, fuente_rubro_id y JOINs
func (r *PlanillaRepository) ObtenerConceptosPorDetalle(detalleID int) ([]models.PlanillaConcepto, error) {
	query := `
		SELECT pc.id, pc.planilla_detalle_id, pc.concepto_tenant_id, pc.maestro_id, pc.tipo_concepto, pc.monto, pc.codigo_sunat, pc.nombre_en_boleta,
		       pc.meta_id, pc.fuente_rubro_id,
		       COALESCE(m.codigo, '') AS meta_codigo,
		       COALESCE(m.descripcion, '') AS meta_descripcion,
		       COALESCE(fr.codigo_fuente_rubro, fr.rubro, '') AS fuente_rubro_codigo,
		       COALESCE(fr.rubro, '') AS fuente_rubro_descripcion,
		       COALESCE(cmef.codigo, '') AS clasificador_codigo
		FROM planilla_conceptos pc
		LEFT JOIN metas_presupuestales m ON pc.meta_id = m.id
		LEFT JOIN fuentes_rubros fr ON pc.fuente_rubro_id = fr.id
		LEFT JOIN conceptos_tenant ct ON pc.concepto_tenant_id = ct.id
		LEFT JOIN clasificadores_mef cmef ON ct.clasificador_id = cmef.id
		WHERE pc.planilla_detalle_id = $1
		ORDER BY pc.id ASC
	`
	rows, err := r.db.Query(query, detalleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.PlanillaConcepto
	for rows.Next() {
		var c models.PlanillaConcepto
		var metaID, rubroID sql.NullInt64
		err := rows.Scan(
			&c.ID, &c.PlanillaDetalleID, &c.ConceptoTenantID, &c.MaestroID, &c.TipoConcepto, &c.Monto, &c.CodigoSunat, &c.NombreEnBoleta,
			&metaID, &rubroID, &c.MetaCodigo, &c.MetaDescripcion, &c.FuenteRubroCodigo, &c.FuenteRubroDescripcion, &c.ClasificadorCodigo,
		)
		if err != nil {
			return nil, err
		}
		if metaID.Valid {
			val := int(metaID.Int64)
			c.MetaID = &val
		}
		if rubroID.Valid {
			val := int(rubroID.Int64)
			c.FuenteRubroID = &val
		}
		lista = append(lista, c)
	}
	return lista, rows.Err()
}

// ObtenerConceptosPorPlanilla trae todos los conceptos de una planilla con meta_id, fuente_rubro_id y JOINs
func (r *PlanillaRepository) ObtenerConceptosPorPlanilla(planillaID int, tenantID int) ([]models.PlanillaConcepto, error) {
	query := `
		SELECT pc.id, pc.planilla_detalle_id, pc.concepto_tenant_id, pc.maestro_id, pc.tipo_concepto, pc.monto, pc.codigo_sunat, pc.nombre_en_boleta,
		       pc.meta_id, pc.fuente_rubro_id,
		       COALESCE(m.codigo, '') AS meta_codigo,
		       COALESCE(m.descripcion, '') AS meta_descripcion,
		       COALESCE(fr.codigo_fuente_rubro, fr.rubro, '') AS fuente_rubro_codigo,
		       COALESCE(fr.rubro, '') AS fuente_rubro_descripcion,
		       COALESCE(cmef.codigo, '') AS clasificador_codigo
		FROM planilla_conceptos pc
		INNER JOIN planilla_detalles pd ON pc.planilla_detalle_id = pd.id
		INNER JOIN planillas pl ON pd.planilla_id = pl.id
		LEFT JOIN metas_presupuestales m ON pc.meta_id = m.id
		LEFT JOIN fuentes_rubros fr ON pc.fuente_rubro_id = fr.id
		LEFT JOIN conceptos_tenant ct ON pc.concepto_tenant_id = ct.id
		LEFT JOIN clasificadores_mef cmef ON ct.clasificador_id = cmef.id
		WHERE pd.planilla_id = $1 AND pl.tenant_id = $2
		ORDER BY pc.id ASC
	`
	rows, err := r.db.Query(query, planillaID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.PlanillaConcepto
	for rows.Next() {
		var c models.PlanillaConcepto
		var metaID, rubroID sql.NullInt64
		err := rows.Scan(
			&c.ID, &c.PlanillaDetalleID, &c.ConceptoTenantID, &c.MaestroID, &c.TipoConcepto, &c.Monto, &c.CodigoSunat, &c.NombreEnBoleta,
			&metaID, &rubroID, &c.MetaCodigo, &c.MetaDescripcion, &c.FuenteRubroCodigo, &c.FuenteRubroDescripcion, &c.ClasificadorCodigo,
		)
		if err != nil {
			return nil, err
		}
		if metaID.Valid {
			val := int(metaID.Int64)
			c.MetaID = &val
		}
		if rubroID.Valid {
			val := int(rubroID.Int64)
			c.FuenteRubroID = &val
		}
		lista = append(lista, c)
	}
	return lista, rows.Err()
}

// ObtenerDetallesConConceptos trae las boletas de una planilla con todos sus conceptos precargados
func (r *PlanillaRepository) ObtenerDetallesConConceptos(planillaID int, tenantID int) ([]models.PlanillaDetalle, error) {
	detalles, err := r.ObtenerDetalles(planillaID, tenantID)
	if err != nil {
		return nil, err
	}
	if len(detalles) == 0 {
		return detalles, nil
	}

	conceptos, err := r.ObtenerConceptosPorPlanilla(planillaID, tenantID)
	if err != nil {
		return detalles, nil
	}

	mapaConceptos := make(map[int][]models.PlanillaConcepto)
	for _, c := range conceptos {
		mapaConceptos[c.PlanillaDetalleID] = append(mapaConceptos[c.PlanillaDetalleID], c)
	}

	for i := range detalles {
		detalles[i].Conceptos = mapaConceptos[detalles[i].ID]
	}

	return detalles, nil
}

// CRUD ReglasFinanciamientoConcepto

func (r *PlanillaRepository) ObtenerReglasFinanciamiento(ctx context.Context, tenantID int) ([]models.ReglaFinanciamientoConcepto, error) {
	query := `
		SELECT r.id, r.tenant_id, r.concepto_tenant_id, r.regimen_id, r.meta_id, r.fuente_rubro_id, r.activo, r.created_at, r.updated_at,
		       COALESCE(ct_cm.descripcion, ct.nombre_personalizado, '') AS concepto_nombre,
		       COALESCE(rl.descripcion, '') AS regimen_nombre,
		       COALESCE(m.codigo, '') AS meta_codigo,
		       COALESCE(m.descripcion, '') AS meta_descripcion,
		       COALESCE(fr.codigo_fuente_rubro, fr.rubro, '') AS fuente_rubro_codigo,
		       COALESCE(fr.rubro, '') AS fuente_rubro_descripcion
		FROM reglas_financiamiento_concepto r
		INNER JOIN conceptos_tenant ct ON r.concepto_tenant_id = ct.id
		INNER JOIN conceptos_maestros ct_cm ON ct.concepto_id = ct_cm.id
		LEFT JOIN regimenes_laborales rl ON r.regimen_id = rl.id
		LEFT JOIN metas_presupuestales m ON r.meta_id = m.id
		LEFT JOIN fuentes_rubros fr ON r.fuente_rubro_id = fr.id
		WHERE r.tenant_id = $1
		ORDER BY r.id DESC
	`
	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.ReglaFinanciamientoConcepto
	for rows.Next() {
		var reg models.ReglaFinanciamientoConcepto
		var regID, metaID, rubroID sql.NullInt64
		err := rows.Scan(
			&reg.ID, &reg.TenantID, &reg.ConceptoTenantID, &regID, &metaID, &rubroID, &reg.Activo, &reg.CreatedAt, &reg.UpdatedAt,
			&reg.ConceptoNombre, &reg.RegimenNombre, &reg.MetaCodigo, &reg.MetaDescripcion, &reg.FuenteRubroCodigo, &reg.FuenteRubroDescripcion,
		)
		if err != nil {
			return nil, err
		}
		if regID.Valid {
			v := int(regID.Int64)
			reg.RegimenID = &v
		}
		if metaID.Valid {
			v := int(metaID.Int64)
			reg.MetaID = &v
		}
		if rubroID.Valid {
			v := int(rubroID.Int64)
			reg.FuenteRubroID = &v
		}
		lista = append(lista, reg)
	}
	return lista, rows.Err()
}

func (r *PlanillaRepository) ObtenerReglasFinanciamientoPorTenant(tenantID int) ([]models.ReglaFinanciamientoConcepto, error) {
	return r.ObtenerReglasFinanciamiento(context.Background(), tenantID)
}

func (r *PlanillaRepository) ObtenerReglaFinanciamientoPorID(ctx context.Context, id int, tenantID int) (*models.ReglaFinanciamientoConcepto, error) {
	query := `
		SELECT r.id, r.tenant_id, r.concepto_tenant_id, r.regimen_id, r.meta_id, r.fuente_rubro_id, r.activo, r.created_at, r.updated_at,
		       COALESCE(ct_cm.descripcion, ct.nombre_personalizado, '') AS concepto_nombre,
		       COALESCE(rl.descripcion, '') AS regimen_nombre,
		       COALESCE(m.codigo, '') AS meta_codigo,
		       COALESCE(m.descripcion, '') AS meta_descripcion,
		       COALESCE(fr.codigo_fuente_rubro, fr.rubro, '') AS fuente_rubro_codigo,
		       COALESCE(fr.rubro, '') AS fuente_rubro_descripcion
		FROM reglas_financiamiento_concepto r
		INNER JOIN conceptos_tenant ct ON r.concepto_tenant_id = ct.id
		INNER JOIN conceptos_maestros ct_cm ON ct.concepto_id = ct_cm.id
		LEFT JOIN regimenes_laborales rl ON r.regimen_id = rl.id
		LEFT JOIN metas_presupuestales m ON r.meta_id = m.id
		LEFT JOIN fuentes_rubros fr ON r.fuente_rubro_id = fr.id
		WHERE r.id = $1 AND r.tenant_id = $2
	`
	row := r.db.QueryRowContext(ctx, query, id, tenantID)

	var reg models.ReglaFinanciamientoConcepto
	var regID, metaID, rubroID sql.NullInt64
	err := row.Scan(
		&reg.ID, &reg.TenantID, &reg.ConceptoTenantID, &regID, &metaID, &rubroID, &reg.Activo, &reg.CreatedAt, &reg.UpdatedAt,
		&reg.ConceptoNombre, &reg.RegimenNombre, &reg.MetaCodigo, &reg.MetaDescripcion, &reg.FuenteRubroCodigo, &reg.FuenteRubroDescripcion,
	)
	if err != nil {
		return nil, err
	}
	if regID.Valid {
		v := int(regID.Int64)
		reg.RegimenID = &v
	}
	if metaID.Valid {
		v := int(metaID.Int64)
		reg.MetaID = &v
	}
	if rubroID.Valid {
		v := int(rubroID.Int64)
		reg.FuenteRubroID = &v
	}
	return &reg, nil
}

func (r *PlanillaRepository) CrearReglaFinanciamiento(ctx context.Context, regla *models.ReglaFinanciamientoConcepto) error {
	query := `
		INSERT INTO reglas_financiamiento_concepto (tenant_id, concepto_tenant_id, regimen_id, meta_id, fuente_rubro_id, activo)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`
	var regID, metaID, rubroID interface{}
	if regla.RegimenID != nil && *regla.RegimenID > 0 {
		regID = *regla.RegimenID
	}
	if regla.MetaID != nil && *regla.MetaID > 0 {
		metaID = *regla.MetaID
	}
	if regla.FuenteRubroID != nil && *regla.FuenteRubroID > 0 {
		rubroID = *regla.FuenteRubroID
	}

	return r.db.QueryRowContext(ctx, query, regla.TenantID, regla.ConceptoTenantID, regID, metaID, rubroID, regla.Activo).
		Scan(&regla.ID, &regla.CreatedAt, &regla.UpdatedAt)
}

func (r *PlanillaRepository) ActualizarReglaFinanciamiento(ctx context.Context, regla *models.ReglaFinanciamientoConcepto) error {
	query := `
		UPDATE reglas_financiamiento_concepto
		SET concepto_tenant_id = $1, regimen_id = $2, meta_id = $3, fuente_rubro_id = $4, activo = $5, updated_at = CURRENT_TIMESTAMP
		WHERE id = $6 AND tenant_id = $7
	`
	var regID, metaID, rubroID interface{}
	if regla.RegimenID != nil && *regla.RegimenID > 0 {
		regID = *regla.RegimenID
	}
	if regla.MetaID != nil && *regla.MetaID > 0 {
		metaID = *regla.MetaID
	}
	if regla.FuenteRubroID != nil && *regla.FuenteRubroID > 0 {
		rubroID = *regla.FuenteRubroID
	}

	_, err := r.db.ExecContext(ctx, query, regla.ConceptoTenantID, regID, metaID, rubroID, regla.Activo, regla.ID, regla.TenantID)
	return err
}

func (r *PlanillaRepository) EliminarReglaFinanciamiento(ctx context.Context, id int, tenantID int) error {
	query := `DELETE FROM reglas_financiamiento_concepto WHERE id = $1 AND tenant_id = $2`
	_, err := r.db.ExecContext(ctx, query, id, tenantID)
	return err
}

func (r *PlanillaRepository) ObtenerMetas(tenantID int) ([]models.MetaPresupuestal, error) {
	query := `SELECT id, tenant_id, anio, codigo, descripcion, activo FROM metas_presupuestales WHERE tenant_id = $1 AND activo = true ORDER BY codigo ASC`
	rows, err := r.db.Query(query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.MetaPresupuestal
	for rows.Next() {
		var m models.MetaPresupuestal
		if err := rows.Scan(&m.ID, &m.TenantID, &m.Anio, &m.Codigo, &m.Descripcion, &m.Activo); err != nil {
			return nil, err
		}
		lista = append(lista, m)
	}
	return lista, nil
}

func (r *PlanillaRepository) ObtenerFuentesRubros() ([]models.FuenteRubro, error) {
	query := `SELECT id, anio, fuente_financiamiento, rubro, COALESCE(codigo_fuente_rubro, ''), activo FROM fuentes_rubros WHERE activo = true ORDER BY id ASC`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.FuenteRubro
	for rows.Next() {
		var fr models.FuenteRubro
		if err := rows.Scan(&fr.ID, &fr.Anio, &fr.FuenteFinanciamiento, &fr.Rubro, &fr.CodigoFuenteRubro, &fr.Activo); err != nil {
			return nil, err
		}
		lista = append(lista, fr)
	}
	return lista, nil
}



