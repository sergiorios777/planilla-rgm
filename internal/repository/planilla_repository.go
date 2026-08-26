package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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

// ObtenerTodos trae el historial de planillas de la entidad con acumulados financieros (exclusivo para planillas mensuales Ordinarias y Extraordinarias)
func (r *PlanillaRepository) ObtenerTodos(tenantID int) ([]models.Planilla, error) {
	query := `
		SELECT 
			p.id, p.tenant_id, p.anio, p.mes, p.descripcion, p.estado, p.es_extraordinaria, p.tipo,
			COALESCE(SUM(pd.total_ingresos), 0.00) AS total_ingresos,
			COALESCE(SUM(pd.total_aportes), 0.00) AS total_aportes,
			COALESCE(SUM(pd.total_ingresos + pd.total_aportes), 0.00) AS costo_total
		FROM planillas p
		LEFT JOIN planilla_detalles pd ON p.id = pd.planilla_id
		WHERE p.tenant_id = $1 AND p.tipo IN ('ORDINARIA', 'EXTRAORDINARIA')
		GROUP BY p.id, p.tenant_id, p.anio, p.mes, p.descripcion, p.estado, p.es_extraordinaria, p.tipo
		ORDER BY p.anio DESC, p.mes DESC, p.id DESC
	`
	rows, err := r.db.Query(query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.Planilla
	for rows.Next() {
		var p models.Planilla
		err := rows.Scan(
			&p.ID, &p.TenantID, &p.Anio, &p.Mes, &p.Descripcion, &p.Estado, &p.EsExtraordinaria, &p.Tipo,
			&p.TotalIngresos, &p.TotalAportes, &p.CostoTotal,
		)
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
	tipo := p.Tipo
	if tipo == "" {
		if p.EsExtraordinaria {
			tipo = "EXTRAORDINARIA"
		} else {
			tipo = "ORDINARIA"
		}
	}

	query := `
		INSERT INTO planillas (tenant_id, anio, mes, descripcion, estado, es_extraordinaria, tipo)
		VALUES ($1, $2, $3, $4, 'BORRADOR', $5, $6) RETURNING id
	`
	return r.db.QueryRow(query, p.TenantID, p.Anio, p.Mes, p.Descripcion, p.EsExtraordinaria, tipo).Scan(&p.ID)
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
		SELECT c.id, c.tenant_id, c.trabajador_id, c.puesto_id, p.regimen_id, rl.codigo, 
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
		err := rows.Scan(&c.ID, &c.TenantID, &c.TrabajadorID, &c.PuestoID, &c.RegimenID, &c.Regimen, &c.RegimenPensionario, &c.AfpID, &c.AfpTipoComision, &c.FechaInicio, &c.FechaFin,
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
			COALESCE(pd.trabajador_numero_documento, t.numero_documento, ''), 
			COALESCE(pd.trabajador_nombre_completo, TRIM(t.apellido_paterno || ' ' || t.apellido_materno || ', ' || t.nombres), ''),
			COALESCE(pd.puesto_nombre, p.nombre, 'Sin Plaza Asignada'),
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
		ORDER BY COALESCE(pd.trabajador_nombre_completo, TRIM(t.apellido_paterno || ' ' || t.apellido_materno || ', ' || t.nombres)) ASC
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
	query := `SELECT id, tenant_id, anio, mes, descripcion, estado, es_extraordinaria, tipo FROM planillas WHERE id = $1 AND tenant_id = $2`
	err := r.db.QueryRow(query, planillaID, tenantID).Scan(&p.ID, &p.TenantID, &p.Anio, &p.Mes, &p.Descripcion, &p.Estado, &p.EsExtraordinaria, &p.Tipo)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// ObtenerTipo obtiene el tipo de una planilla (ORDINARIA, EXTRAORDINARIA, CTS, CESE)
func (r *PlanillaRepository) ObtenerTipo(planillaID int, tenantID int) (string, error) {
	var tipo string
	query := `SELECT tipo FROM planillas WHERE id = $1 AND tenant_id = $2`
	err := r.db.QueryRow(query, planillaID, tenantID).Scan(&tipo)
	return tipo, err
}

// ObtenerCtsIDPorPlanillaID obtiene el ID de la planilla CTS asociada a una planilla espejo
func (r *PlanillaRepository) ObtenerCtsIDPorPlanillaID(planillaID int, tenantID int) (int, error) {
	var ctsID int
	query := `SELECT id FROM planillas_cts WHERE planilla_id = $1 AND tenant_id = $2`
	err := r.db.QueryRow(query, planillaID, tenantID).Scan(&ctsID)
	return ctsID, err
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
		SELECT t.tipo_documento, t.numero_documento, COALESCE(NULLIF(pc.codigo_sunat, ''), cm.codigo) AS codigo, pc.monto
		FROM planilla_conceptos pc
		INNER JOIN planilla_detalles pd ON pc.planilla_detalle_id = pd.id
		INNER JOIN contratos c ON pd.contrato_id = c.id
		INNER JOIN trabajadores t ON c.trabajador_id = t.id
		LEFT JOIN conceptos_maestros cm ON pc.maestro_id = cm.id
		INNER JOIN planillas pl ON pd.planilla_id = pl.id
		WHERE pd.planilla_id = $1 AND pl.tenant_id = $2 
		  AND (cm.origen = 'sunat' OR pc.codigo_sunat IS NOT NULL) 
		  AND pc.monto > 0
		ORDER BY t.numero_documento, codigo
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

func (r *PlanillaRepository) ObtenerReglasFinanciamientoPorConceptoID(ctx context.Context, conceptoTenantID int, tenantID int) ([]models.ReglaFinanciamientoConcepto, error) {
	query := `
		SELECT r.id, r.tenant_id, r.concepto_tenant_id, r.regimen_id, r.meta_id, r.fuente_rubro_id, r.activo, r.created_at, r.updated_at,
		       COALESCE(ct_cm.descripcion, ct.nombre_personalizado, '') AS concepto_nombre,
		       COALESCE(rl.descripcion, 'Todos') AS regimen_nombre,
		       COALESCE(m.codigo, '') AS meta_codigo,
		       COALESCE(m.descripcion, '') AS meta_descripcion,
		       COALESCE(fr.codigo_fuente_rubro, fr.rubro, '') AS fuente_rubro_codigo,
		       COALESCE(fr.rubro, '') AS fuente_rubro_descripcion
		FROM reglas_financiamiento_concepto r
		INNER JOIN conceptos_tenant ct ON r.concepto_tenant_id = ct.id
		LEFT JOIN conceptos_maestros ct_cm ON ct.concepto_id = ct_cm.id
		LEFT JOIN regimenes_laborales rl ON r.regimen_id = rl.id
		LEFT JOIN metas_presupuestales m ON r.meta_id = m.id
		LEFT JOIN fuentes_rubros fr ON r.fuente_rubro_id = fr.id
		WHERE r.concepto_tenant_id = $1 AND r.tenant_id = $2
		ORDER BY r.id DESC
	`
	rows, err := r.db.QueryContext(ctx, query, conceptoTenantID, tenantID)
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
	if err := rows.Err(); err != nil {
		return nil, err
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return lista, nil
}

// ObtenerTrabajadoresEspecialPaginacion obtiene trabajadores activos y su información de puesto/régimen/meta para la planilla especial
func (r *PlanillaRepository) ObtenerTrabajadoresEspecialPaginacion(tenantID int, busqueda string, regimenID int, metaID int, unidadID int, limite int, offset int) ([]models.TrabajadorEspecialItem, int, error) {
	whereClause := `WHERE c.tenant_id = $1 AND c.activo = true AND t.activo = true`
	args := []interface{}{tenantID}
	paramIdx := 2

	if busqueda != "" {
		whereClause += fmt.Sprintf(` AND (t.numero_documento ILIKE $%d OR t.nombres || ' ' || t.apellido_paterno || ' ' || t.apellido_materno ILIKE $%d OR t.apellido_paterno || ' ' || t.apellido_materno || ', ' || t.nombres ILIKE $%d OR t.apellido_paterno || ' ' || t.apellido_materno || ' ' || t.nombres ILIKE $%d)`, paramIdx, paramIdx, paramIdx, paramIdx)
		args = append(args, "%"+busqueda+"%")
		paramIdx++
	}
	if regimenID > 0 {
		whereClause += fmt.Sprintf(` AND p.regimen_id = $%d`, paramIdx)
		args = append(args, regimenID)
		paramIdx++
	}
	if metaID > 0 {
		whereClause += fmt.Sprintf(` AND p.meta_id = $%d`, paramIdx)
		args = append(args, metaID)
		paramIdx++
	}
	if unidadID > 0 {
		whereClause += fmt.Sprintf(` AND p.unidad_organica_id = $%d`, paramIdx)
		args = append(args, unidadID)
		paramIdx++
	}

	var total int
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM contratos c
		INNER JOIN trabajadores t ON c.trabajador_id = t.id
		LEFT JOIN puestos p ON c.puesto_id = p.id
		%s
	`, whereClause)

	err := r.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		log.Println("❌ ERROR EN COUNT QUERY:", err)
		return nil, 0, err
	}

	query := fmt.Sprintf(`
		SELECT c.id AS contrato_id,
		       t.id AS trabajador_id,
		       t.numero_documento,
		       t.apellido_paterno || ' ' || t.apellido_materno || ', ' || t.nombres AS nombre_completo,
		       COALESCE(p.nombre, 'Sin Plaza') AS puesto_nombre,
		       COALESCE(rl.id, 0) AS regimen_id,
		       COALESCE(rl.descripcion, 'Sin Régimen') AS regimen_nombre,
		       COALESCE(m.id, 0) AS meta_id,
		       COALESCE(m.codigo, '') AS meta_codigo,
		       COALESCE(m.descripcion, '') AS meta_descripcion,
		       COALESCE(uo.id, 0) AS unidad_organica_id,
		       COALESCE(uo.nombre, '') AS unidad_organica_nombre
		FROM contratos c
		INNER JOIN trabajadores t ON c.trabajador_id = t.id
		LEFT JOIN puestos p ON c.puesto_id = p.id
		LEFT JOIN regimenes_laborales rl ON p.regimen_id = rl.id
		LEFT JOIN metas_presupuestales m ON p.meta_id = m.id
		LEFT JOIN unidades_organicas uo ON p.unidad_organica_id = uo.id
		%s
		ORDER BY t.apellido_paterno ASC, t.apellido_materno ASC, t.nombres ASC
		LIMIT $%d OFFSET $%d
	`, whereClause, paramIdx, paramIdx+1)

	args = append(args, limite, offset)
	rows, err := r.db.Query(query, args...)
	if err != nil {
		log.Println("❌ ERROR EN QUERY PRINCIPAL:", err)
		return nil, 0, err
	}
	defer rows.Close()

	var lista []models.TrabajadorEspecialItem
	for rows.Next() {
		var item models.TrabajadorEspecialItem
		err := rows.Scan(
			&item.ContratoID, &item.TrabajadorID, &item.NumeroDocumento, &item.NombreCompleto,
			&item.PuestoNombre, &item.RegimenID, &item.RegimenNombre,
			&item.MetaID, &item.MetaCodigo, &item.MetaDescripcion,
			&item.UnidadOrganicaID, &item.UnidadOrganicaNombre,
		)
		if err != nil {
			return nil, 0, err
		}
		lista = append(lista, item)
	}
	return lista, total, rows.Err()
}

// ObtenerTrabajadoresEspecialTodos obtiene TODOS los trabajadores activos que coinciden con los filtros sin paginar
func (r *PlanillaRepository) ObtenerTrabajadoresEspecialTodos(tenantID int, busqueda string, regimenID int, metaID int, unidadID int) ([]models.TrabajadorEspecialItem, error) {
	whereClause := `WHERE c.tenant_id = $1 AND c.activo = true AND t.activo = true`
	args := []interface{}{tenantID}
	paramIdx := 2

	if busqueda != "" {
		whereClause += fmt.Sprintf(` AND (t.numero_documento ILIKE $%d OR t.nombres || ' ' || t.apellido_paterno || ' ' || t.apellido_materno ILIKE $%d OR t.apellido_paterno || ' ' || t.apellido_materno || ', ' || t.nombres ILIKE $%d OR t.apellido_paterno || ' ' || t.apellido_materno || ' ' || t.nombres ILIKE $%d)`, paramIdx, paramIdx, paramIdx, paramIdx)
		args = append(args, "%"+busqueda+"%")
		paramIdx++
	}
	if regimenID > 0 {
		whereClause += fmt.Sprintf(` AND p.regimen_id = $%d`, paramIdx)
		args = append(args, regimenID)
		paramIdx++
	}
	if metaID > 0 {
		whereClause += fmt.Sprintf(` AND p.meta_id = $%d`, paramIdx)
		args = append(args, metaID)
		paramIdx++
	}
	if unidadID > 0 {
		whereClause += fmt.Sprintf(` AND p.unidad_organica_id = $%d`, paramIdx)
		args = append(args, unidadID)
		paramIdx++
	}

	query := fmt.Sprintf(`
		SELECT c.id AS contrato_id,
		       t.id AS trabajador_id,
		       t.numero_documento,
		       t.apellido_paterno || ' ' || t.apellido_materno || ', ' || t.nombres AS nombre_completo,
		       COALESCE(p.nombre, 'Sin Plaza') AS puesto_nombre,
		       COALESCE(rl.id, 0) AS regimen_id,
		       COALESCE(rl.descripcion, 'Sin Régimen') AS regimen_nombre,
		       COALESCE(m.id, 0) AS meta_id,
		       COALESCE(m.codigo, '') AS meta_codigo,
		       COALESCE(m.descripcion, '') AS meta_descripcion,
		       COALESCE(uo.id, 0) AS unidad_organica_id,
		       COALESCE(uo.nombre, '') AS unidad_organica_nombre
		FROM contratos c
		INNER JOIN trabajadores t ON c.trabajador_id = t.id
		LEFT JOIN puestos p ON c.puesto_id = p.id
		LEFT JOIN regimenes_laborales rl ON p.regimen_id = rl.id
		LEFT JOIN metas_presupuestales m ON p.meta_id = m.id
		LEFT JOIN unidades_organicas uo ON p.unidad_organica_id = uo.id
		%s
		ORDER BY t.apellido_paterno ASC, t.apellido_materno ASC, t.nombres ASC
	`, whereClause)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.TrabajadorEspecialItem
	for rows.Next() {
		var item models.TrabajadorEspecialItem
		err := rows.Scan(
			&item.ContratoID, &item.TrabajadorID, &item.NumeroDocumento, &item.NombreCompleto,
			&item.PuestoNombre, &item.RegimenID, &item.RegimenNombre,
			&item.MetaID, &item.MetaCodigo, &item.MetaDescripcion,
			&item.UnidadOrganicaID, &item.UnidadOrganicaNombre,
		)
		if err != nil {
			return nil, err
		}
		lista = append(lista, item)
	}
	return lista, rows.Err()
}

type PlanillaEspecialConceptoInput struct {
	ConceptoTenantID int     `json:"concepto_tenant_id"`
	Monto            float64 `json:"monto"`
}

// ProcesarPlanillaEspecial ejecuta la formulación de una planilla extraordinaria/especial
func (r *PlanillaRepository) ProcesarPlanillaEspecial(
	ctx context.Context,
	planillaID int,
	tenantID int,
	conceptosInput []PlanillaEspecialConceptoInput,
	contratosIDs []int,
	montosCustom map[string]float64,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Verificar estado de la planilla
	var estado string
	var esExtraordinaria bool
	err = tx.QueryRowContext(ctx, `SELECT estado, es_extraordinaria FROM planillas WHERE id = $1 AND tenant_id = $2`, planillaID, tenantID).Scan(&estado, &esExtraordinaria)
	if err != nil {
		return fmt.Errorf("no se encontró la planilla: %w", err)
	}
	if estado != "BORRADOR" {
		return fmt.Errorf("solo se pueden procesar planillas en estado BORRADOR")
	}

	// 2. Limpiar cálculos anteriores si existieran
	_, err = tx.ExecContext(ctx, `
		DELETE FROM planilla_conceptos 
		WHERE planilla_detalle_id IN (SELECT id FROM planilla_detalles WHERE planilla_id = $1)
	`, planillaID)
	if err != nil {
		return fmt.Errorf("error limpiando conceptos anteriores: %w", err)
	}

	_, err = tx.ExecContext(ctx, `DELETE FROM planilla_detalles WHERE planilla_id = $1`, planillaID)
	if err != nil {
		return fmt.Errorf("error limpiando detalles anteriores: %w", err)
	}

	// 3. Cargar Reglas Tenant activas
	reglasTenantRows, err := tx.QueryContext(ctx, `
		SELECT concepto_tenant_id, regimen_id, meta_id, fuente_rubro_id 
		FROM reglas_financiamiento_concepto 
		WHERE tenant_id = $1 AND activo = true
	`, tenantID)
	if err != nil {
		return fmt.Errorf("error al cargar reglas tenant: %w", err)
	}
	defer reglasTenantRows.Close()

	type keyRegla struct {
		conceptoID int
		regimenID  int
	}
	reglasTenantMap := make(map[keyRegla]struct{ metaID, rubroID *int })
	for reglasTenantRows.Next() {
		var cID int
		var regID, mID, rID sql.NullInt64
		if err := reglasTenantRows.Scan(&cID, &regID, &mID, &rID); err == nil {
			var rIDVal int
			if regID.Valid {
				rIDVal = int(regID.Int64)
			}
			k := keyRegla{conceptoID: cID, regimenID: rIDVal}
			var metaPtr, rubroPtr *int
			if mID.Valid && mID.Int64 > 0 {
				v := int(mID.Int64)
				metaPtr = &v
			}
			if rID.Valid && rID.Int64 > 0 {
				v := int(rID.Int64)
				rubroPtr = &v
			}
			reglasTenantMap[k] = struct{ metaID, rubroID *int }{metaPtr, rubroPtr}
		}
	}
	if err := reglasTenantRows.Err(); err != nil {
		return fmt.Errorf("error al iterar reglas tenant: %w", err)
	}

	// 4. Cargar Reglas Modelo SaaS activas
	reglasModeloRows, err := tx.QueryContext(ctx, `
		SELECT concepto_modelo_id, regimen_id, meta_id, fuente_rubro_id 
		FROM reglas_financiamiento_modelo 
		WHERE activo = true
	`)
	if err == nil {
		defer reglasModeloRows.Close()
	}
	reglasModeloMap := make(map[keyRegla]struct{ metaID, rubroID *int })
	if reglasModeloRows != nil {
		for reglasModeloRows.Next() {
			var cmID int
			var regID, mID, rID sql.NullInt64
			if err := reglasModeloRows.Scan(&cmID, &regID, &mID, &rID); err == nil {
				var rIDVal int
				if regID.Valid {
					rIDVal = int(regID.Int64)
				}
				k := keyRegla{conceptoID: cmID, regimenID: rIDVal}
				var metaPtr, rubroPtr *int
				if mID.Valid && mID.Int64 > 0 {
					v := int(mID.Int64)
					metaPtr = &v
				}
				if rID.Valid && rID.Int64 > 0 {
					v := int(rID.Int64)
					rubroPtr = &v
				}
				reglasModeloMap[k] = struct{ metaID, rubroID *int }{metaPtr, rubroPtr}
			}
		}
	}

	// 5. Cargar información de los conceptos seleccionados
	type conceptoInfo struct {
		id                  int
		nombrePersonalizado string
		conceptoMaestroID   int
		modeloID            int
		codigoSunat         string
		conceptoTipo        string
	}
	conceptosMap := make(map[int]conceptoInfo)

	for _, cInput := range conceptosInput {
		var ci conceptoInfo
		ci.id = cInput.ConceptoTenantID
		var modID sql.NullInt64
		queryC := `
			SELECT ct.nombre_personalizado, ct.concepto_id, COALESCE(ct.modelo_id, 0), cm.codigo, cm.tipo
			FROM conceptos_tenant ct
			INNER JOIN conceptos_maestros cm ON ct.concepto_id = cm.id
			WHERE ct.id = $1 AND ct.tenant_id = $2
		`
		err := tx.QueryRowContext(ctx, queryC, cInput.ConceptoTenantID, tenantID).Scan(
			&ci.nombrePersonalizado, &ci.conceptoMaestroID, &modID, &ci.codigoSunat, &ci.conceptoTipo,
		)
		if err != nil {
			return fmt.Errorf("error al obtener concepto %d: %w", cInput.ConceptoTenantID, err)
		}
		if modID.Valid {
			ci.modeloID = int(modID.Int64)
		}
		conceptosMap[cInput.ConceptoTenantID] = ci
	}

	// 6. Iterar por cada contrato/trabajador seleccionado
	stmtDetalle, err := tx.PrepareContext(ctx, `
		INSERT INTO planilla_detalles (
			planilla_id, contrato_id, total_ingresos, total_retenciones, total_aportes, neto_pagar,
			trabajador_nombre_completo, trabajador_numero_documento, puesto_nombre
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id
	`)
	if err != nil {
		return err
	}
	defer stmtDetalle.Close()

	stmtConcepto, err := tx.PrepareContext(ctx, `
		INSERT INTO planilla_conceptos (planilla_detalle_id, concepto_tenant_id, tipo_concepto, monto, maestro_id, codigo_sunat, nombre_en_boleta, meta_id, fuente_rubro_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`)
	if err != nil {
		return err
	}
	defer stmtConcepto.Close()

	for _, contratoID := range contratosIDs {
		// Obtener plaza, régimen y defaults del contrato
		var puestoID sql.NullInt64
		var regimenID int
		var puestoMetaID, puestoRubroID sql.NullInt64
		var trabajadorNombre, trabajadorDoc, puestoNombre string

		queryContrato := `
			SELECT c.puesto_id, COALESCE(p.regimen_id, 0), p.meta_id, p.fuente_rubro_id,
			       COALESCE(TRIM(t.apellido_paterno || ' ' || t.apellido_materno || ', ' || t.nombres), ''),
			       COALESCE(t.numero_documento, ''),
			       COALESCE(p.nombre, 'Sin Plaza Asignada')
			FROM contratos c
			LEFT JOIN trabajadores t ON c.trabajador_id = t.id
			LEFT JOIN puestos p ON c.puesto_id = p.id
			WHERE c.id = $1 AND c.tenant_id = $2
		`
		err := tx.QueryRowContext(ctx, queryContrato, contratoID, tenantID).Scan(
			&puestoID, &regimenID, &puestoMetaID, &puestoRubroID,
			&trabajadorNombre, &trabajadorDoc, &puestoNombre,
		)
		if err != nil {
			return fmt.Errorf("error al consultar contrato %d: %w", contratoID, err)
		}

		var defaultMetaPtr, defaultRubroPtr *int
		if puestoMetaID.Valid && puestoMetaID.Int64 > 0 {
			v := int(puestoMetaID.Int64)
			defaultMetaPtr = &v
		}
		if puestoRubroID.Valid && puestoRubroID.Int64 > 0 {
			v := int(puestoRubroID.Int64)
			defaultRubroPtr = &v
		}

		var totalIngresos, totalRetenciones, totalAportes float64
		var detalleID int

		err = stmtDetalle.QueryRowContext(ctx, planillaID, contratoID, 0, 0, 0, 0, trabajadorNombre, trabajadorDoc, puestoNombre).Scan(&detalleID)
		if err != nil {
			return fmt.Errorf("error al insertar detalle de planilla: %w", err)
		}

		for _, cInput := range conceptosInput {
			ci, ok := conceptosMap[cInput.ConceptoTenantID]
			if !ok {
				continue
			}

			// Resolver Meta y Rubro con la jerarquía: Tenant -> Modelo SaaS -> Default Puesto
			var finalMetaID, finalRubroID *int

			// 1. Regla Tenant (específica por régimen o general por concepto)
			if rt, ok := reglasTenantMap[keyRegla{conceptoID: ci.id, regimenID: regimenID}]; ok {
				if rt.metaID != nil {
					finalMetaID = rt.metaID
				}
				if rt.rubroID != nil {
					finalRubroID = rt.rubroID
				}
			} else if rtGen, ok := reglasTenantMap[keyRegla{conceptoID: ci.id, regimenID: 0}]; ok {
				if rtGen.metaID != nil {
					finalMetaID = rtGen.metaID
				}
				if rtGen.rubroID != nil {
					finalRubroID = rtGen.rubroID
				}
			}

			// 2. Regla Modelo SaaS (si no se resolvió en Tenant)
			if finalMetaID == nil || finalRubroID == nil {
				if ci.modeloID > 0 {
					if rm, ok := reglasModeloMap[keyRegla{conceptoID: ci.modeloID, regimenID: regimenID}]; ok {
						if finalMetaID == nil && rm.metaID != nil {
							finalMetaID = rm.metaID
						}
						if finalRubroID == nil && rm.rubroID != nil {
							finalRubroID = rm.rubroID
						}
					} else if rmGen, ok := reglasModeloMap[keyRegla{conceptoID: ci.modeloID, regimenID: 0}]; ok {
						if finalMetaID == nil && rmGen.metaID != nil {
							finalMetaID = rmGen.metaID
						}
						if finalRubroID == nil && rmGen.rubroID != nil {
							finalRubroID = rmGen.rubroID
						}
					}
				}
			}

			// 3. Fallback al Puesto / Plaza
			if finalMetaID == nil {
				finalMetaID = defaultMetaPtr
			}
			if finalRubroID == nil {
				finalRubroID = defaultRubroPtr
			}

			montoEfectivo := cInput.Monto
			if len(montosCustom) > 0 {
				customKey := fmt.Sprintf("monto_custom_%d_%d", contratoID, cInput.ConceptoTenantID)
				if val, ok := montosCustom[customKey]; ok {
					montoEfectivo = val
				}
			}
			if montoEfectivo <= 0 {
				continue
			}

			// Totales de boleta
			tipoUpper := strings.ToUpper(strings.TrimSpace(ci.conceptoTipo))
			switch tipoUpper {
			case "INGRESO", "":
				totalIngresos += montoEfectivo
			case "RETENCION":
				totalRetenciones += montoEfectivo
			case "APORTE":
				totalAportes += montoEfectivo
			}

			_, err = stmtConcepto.ExecContext(
				ctx, detalleID, ci.id, ci.conceptoTipo, montoEfectivo,
				ci.conceptoMaestroID, ci.codigoSunat, ci.nombrePersonalizado,
				finalMetaID, finalRubroID,
			)
			if err != nil {
				return fmt.Errorf("error al insertar concepto en boleta: %w", err)
			}
		}

		netoPagar := totalIngresos - totalRetenciones
		_, err = tx.ExecContext(ctx, `
			UPDATE planilla_detalles 
			SET total_ingresos = $1, total_retenciones = $2, total_aportes = $3, neto_pagar = $4
			WHERE id = $5
		`, totalIngresos, totalRetenciones, totalAportes, netoPagar, detalleID)
		if err != nil {
			return fmt.Errorf("error actualizando totales de detalle: %w", err)
		}
	}

	return tx.Commit()
}

// RecalcularPlanillaEspecial re-evalúa metas, rubros y totales para una planilla extraordinaria sin modificar los conceptos y trabajadores ya formulados
func (r *PlanillaRepository) RecalcularPlanillaEspecial(ctx context.Context, planillaID int, tenantID int) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Verificar estado de la planilla
	var estado string
	var esExtraordinaria bool
	err = tx.QueryRowContext(ctx, `SELECT estado, es_extraordinaria FROM planillas WHERE id = $1 AND tenant_id = $2`, planillaID, tenantID).Scan(&estado, &esExtraordinaria)
	if err != nil {
		return fmt.Errorf("no se encontró la planilla: %w", err)
	}
	if estado != "BORRADOR" {
		return fmt.Errorf("solo se pueden procesar planillas en estado BORRADOR")
	}
	if !esExtraordinaria {
		return fmt.Errorf("la planilla no es extraordinaria")
	}

	// 2. Cargar Reglas Tenant activas
	reglasTenantRows, err := tx.QueryContext(ctx, `
		SELECT concepto_tenant_id, regimen_id, meta_id, fuente_rubro_id 
		FROM reglas_financiamiento_concepto 
		WHERE tenant_id = $1 AND activo = true
	`, tenantID)
	if err != nil {
		return fmt.Errorf("error al cargar reglas tenant: %w", err)
	}
	defer reglasTenantRows.Close()

	type keyRegla struct {
		conceptoID int
		regimenID  int
	}
	reglasTenantMap := make(map[keyRegla]struct{ metaID, rubroID *int })
	for reglasTenantRows.Next() {
		var cID int
		var regID, mID, rID sql.NullInt64
		if err := reglasTenantRows.Scan(&cID, &regID, &mID, &rID); err == nil {
			var rIDVal int
			if regID.Valid {
				rIDVal = int(regID.Int64)
			}
			k := keyRegla{conceptoID: cID, regimenID: rIDVal}
			var metaPtr, rubroPtr *int
			if mID.Valid && mID.Int64 > 0 {
				v := int(mID.Int64)
				metaPtr = &v
			}
			if rID.Valid && rID.Int64 > 0 {
				v := int(rID.Int64)
				rubroPtr = &v
			}
			reglasTenantMap[k] = struct{ metaID, rubroID *int }{metaPtr, rubroPtr}
		}
	}
	if err := reglasTenantRows.Err(); err != nil {
		return fmt.Errorf("error al iterar reglas tenant: %w", err)
	}

	// 3. Cargar Reglas Modelo SaaS activas
	reglasModeloRows, err := tx.QueryContext(ctx, `
		SELECT concepto_modelo_id, regimen_id, meta_id, fuente_rubro_id 
		FROM reglas_financiamiento_modelo 
		WHERE activo = true
	`)
	if err == nil {
		defer reglasModeloRows.Close()
	}
	reglasModeloMap := make(map[keyRegla]struct{ metaID, rubroID *int })
	if reglasModeloRows != nil {
		for reglasModeloRows.Next() {
			var cmID int
			var regID, mID, rID sql.NullInt64
			if err := reglasModeloRows.Scan(&cmID, &regID, &mID, &rID); err == nil {
				var rIDVal int
				if regID.Valid {
					rIDVal = int(regID.Int64)
				}
				k := keyRegla{conceptoID: cmID, regimenID: rIDVal}
				var metaPtr, rubroPtr *int
				if mID.Valid && mID.Int64 > 0 {
					v := int(mID.Int64)
					metaPtr = &v
				}
				if rID.Valid && rID.Int64 > 0 {
					v := int(rID.Int64)
					rubroPtr = &v
				}
				reglasModeloMap[k] = struct{ metaID, rubroID *int }{metaPtr, rubroPtr}
			}
		}
	}

	// 4. Leer los detalles de la planilla
	rowsDetalles, err := tx.QueryContext(ctx, `
		SELECT pd.id, pd.contrato_id, COALESCE(p.regimen_id, 0), p.meta_id, p.fuente_rubro_id
		FROM planilla_detalles pd
		INNER JOIN contratos c ON pd.contrato_id = c.id
		LEFT JOIN puestos p ON c.puesto_id = p.id
		WHERE pd.planilla_id = $1
	`, planillaID)
	if err != nil {
		return fmt.Errorf("error leyendo detalles de planilla: %w", err)
	}
	defer rowsDetalles.Close()

	type detalleItem struct {
		id              int
		contratoID      int
		regimenID       int
		defaultMetaPtr  *int
		defaultRubroPtr *int
	}
	var detalles []detalleItem

	for rowsDetalles.Next() {
		var item detalleItem
		var pMetaID, pRubroID sql.NullInt64
		if err := rowsDetalles.Scan(&item.id, &item.contratoID, &item.regimenID, &pMetaID, &pRubroID); err == nil {
			if pMetaID.Valid && pMetaID.Int64 > 0 {
				v := int(pMetaID.Int64)
				item.defaultMetaPtr = &v
			}
			if pRubroID.Valid && pRubroID.Int64 > 0 {
				v := int(pRubroID.Int64)
				item.defaultRubroPtr = &v
			}
			detalles = append(detalles, item)
		}
	}
	if err := rowsDetalles.Err(); err != nil {
		return fmt.Errorf("error leyendo detalles de planilla: %w", err)
	}

	// 5. Para cada detalle, actualizar sus conceptos con las reglas vigentes y recalcular totales
	stmtUpdateConcepto, err := tx.PrepareContext(ctx, `
		UPDATE planilla_conceptos 
		SET meta_id = $1, fuente_rubro_id = $2 
		WHERE id = $3
	`)
	if err != nil {
		return err
	}
	defer stmtUpdateConcepto.Close()

	for _, det := range detalles {
		rowsConc, err := tx.QueryContext(ctx, `
			SELECT pc.id, pc.concepto_tenant_id, pc.maestro_id, pc.tipo_concepto, pc.monto
			FROM planilla_conceptos pc
			WHERE pc.planilla_detalle_id = $1
		`, det.id)
		if err != nil {
			continue
		}

		type concRow struct {
			id               int
			conceptoTenantID sql.NullInt64
			maestroID        sql.NullInt64
			tipo             string
			monto            float64
		}
		var conceptos []concRow

		for rowsConc.Next() {
			var cr concRow
			rowsConc.Scan(&cr.id, &cr.conceptoTenantID, &cr.maestroID, &cr.tipo, &cr.monto)
			conceptos = append(conceptos, cr)
		}
		if err := rowsConc.Err(); err != nil {
			rowsConc.Close()
			return fmt.Errorf("error al iterar conceptos de detalle %d: %w", det.id, err)
		}
		rowsConc.Close()

		var totalIngresos, totalRetenciones, totalAportes float64

		for _, cr := range conceptos {
			cID := 0
			if cr.conceptoTenantID.Valid {
				cID = int(cr.conceptoTenantID.Int64)
			}
			modID := 0
			if cID > 0 {
				var mID sql.NullInt64
				tx.QueryRowContext(ctx, `SELECT modelo_id FROM conceptos_tenant WHERE id = $1`, cID).Scan(&mID)
				if mID.Valid {
					modID = int(mID.Int64)
				}
			}

			// Resolver Meta y Rubro con la jerarquía: Tenant -> Modelo SaaS -> Default Puesto
			var finalMetaID, finalRubroID *int

			// 1. Regla Tenant (específica por régimen o general por concepto)
			if rt, ok := reglasTenantMap[keyRegla{conceptoID: cID, regimenID: det.regimenID}]; ok {
				if rt.metaID != nil {
					finalMetaID = rt.metaID
				}
				if rt.rubroID != nil {
					finalRubroID = rt.rubroID
				}
			} else if rtGen, ok := reglasTenantMap[keyRegla{conceptoID: cID, regimenID: 0}]; ok {
				if rtGen.metaID != nil {
					finalMetaID = rtGen.metaID
				}
				if rtGen.rubroID != nil {
					finalRubroID = rtGen.rubroID
				}
			}

			// 2. Regla Modelo SaaS
			if finalMetaID == nil || finalRubroID == nil {
				if modID > 0 {
					if rm, ok := reglasModeloMap[keyRegla{conceptoID: modID, regimenID: det.regimenID}]; ok {
						if finalMetaID == nil && rm.metaID != nil {
							finalMetaID = rm.metaID
						}
						if finalRubroID == nil && rm.rubroID != nil {
							finalRubroID = rm.rubroID
						}
					} else if rmGen, ok := reglasModeloMap[keyRegla{conceptoID: modID, regimenID: 0}]; ok {
						if finalMetaID == nil && rmGen.metaID != nil {
							finalMetaID = rmGen.metaID
						}
						if finalRubroID == nil && rmGen.rubroID != nil {
							finalRubroID = rmGen.rubroID
						}
					}
				}
			}

			// 3. Fallback al Puesto / Plaza
			if finalMetaID == nil {
				finalMetaID = det.defaultMetaPtr
			}
			if finalRubroID == nil {
				finalRubroID = det.defaultRubroPtr
			}

			stmtUpdateConcepto.ExecContext(ctx, finalMetaID, finalRubroID, cr.id)

			tipoUpper := strings.ToUpper(strings.TrimSpace(cr.tipo))
			switch tipoUpper {
			case "INGRESO", "":
				totalIngresos += cr.monto
			case "RETENCION":
				totalRetenciones += cr.monto
			case "APORTE":
				totalAportes += cr.monto
			}
		}

		netoPagar := totalIngresos - totalRetenciones
		_, err = tx.ExecContext(ctx, `
			UPDATE planilla_detalles 
			SET total_ingresos = $1, total_retenciones = $2, total_aportes = $3, neto_pagar = $4
			WHERE id = $5
		`, totalIngresos, totalRetenciones, totalAportes, netoPagar, det.id)
		if err != nil {
			return fmt.Errorf("error actualizando totales de detalle %d: %w", det.id, err)
		}
	}

	return tx.Commit()
}

// ObtenerFormulacionEspecial recupera los conceptos y trabajadores ya formulados de una planilla extraordinaria
func (r *PlanillaRepository) ObtenerFormulacionEspecial(planillaID int, tenantID int) ([]models.ConceptoFormulacionEspecial, []models.TrabajadorFormulacionEspecial, error) {
	// 1. Obtener Conceptos Formulados
	queryConceptos := `
		SELECT DISTINCT ON (ct.id)
			ct.id,
			ct.nombre_personalizado,
			cm.codigo,
			COALESCE(cr.codigo, 'N/A') as clasificador,
			COALESCE(ct.es_ocasional, false),
			COALESCE(ct.es_extraordinario, false),
			COALESCE(ct.modalidad_entrega, 'PERMANENTE') as modalidad_entrega,
			COALESCE(ct.es_pensionable, false),
			COALESCE(ct.es_remunerativa, false),
			pc.monto,
			pc.id
		FROM planilla_conceptos pc
		INNER JOIN planilla_detalles pd ON pc.planilla_detalle_id = pd.id
		INNER JOIN conceptos_tenant ct ON pc.concepto_tenant_id = ct.id
		INNER JOIN conceptos_maestros cm ON ct.concepto_id = cm.id
		LEFT JOIN clasificadores_mef cr ON ct.clasificador_id = cr.id
		WHERE pd.planilla_id = $1
		ORDER BY ct.id, pc.id ASC
	`
	rowsConc, err := r.db.Query(queryConceptos, planillaID)
	if err != nil {
		log.Println("❌ ERROR EN queryConceptos ObtenerFormulacionEspecial:", err)
		return nil, nil, err
	}
	defer rowsConc.Close()

	var conceptos []models.ConceptoFormulacionEspecial
	for rowsConc.Next() {
		var c models.ConceptoFormulacionEspecial
		var pcID int
		err := rowsConc.Scan(
			&c.ID, &c.NombrePersonalizado, &c.CodigoSunat, &c.ClasificadorCodigo,
			&c.EsOcasional, &c.EsExtraordinario, &c.ModalidadEntrega, &c.EsPensionable, &c.EsRemunerativa,
			&c.MontoBase, &pcID,
		)
		if err == nil {
			conceptos = append(conceptos, c)
		}
	}
	if err := rowsConc.Err(); err != nil {
		return nil, nil, err
	}

	// 2. Obtener Trabajadores y sus montos custom
	queryTrabajadores := `
		SELECT 
			pd.id as detalle_id,
			pd.contrato_id,
			COALESCE(pd.trabajador_nombre_completo, TRIM(t.apellido_paterno || ' ' || t.apellido_materno || ', ' || t.nombres)) as nombre_completo,
			COALESCE(pd.trabajador_numero_documento, t.numero_documento) as numero_documento,
			COALESCE(pd.puesto_nombre, p.nombre, 'Sin Plaza') as puesto_nombre,
			COALESCE(uo.nombre, '') as unidad_organica_nombre,
			COALESCE(rl.descripcion, 'Sin Régimen') as regimen_nombre,
			COALESCE(m.codigo, '') as meta_codigo,
			COALESCE(m.descripcion, '') as meta_descripcion
		FROM planilla_detalles pd
		INNER JOIN contratos c ON pd.contrato_id = c.id
		INNER JOIN trabajadores t ON c.trabajador_id = t.id
		LEFT JOIN puestos p ON c.puesto_id = p.id
		LEFT JOIN unidades_organicas uo ON p.unidad_organica_id = uo.id
		LEFT JOIN regimenes_laborales rl ON p.regimen_id = rl.id
		LEFT JOIN metas_presupuestales m ON p.meta_id = m.id
		WHERE pd.planilla_id = $1
		ORDER BY pd.id ASC
	`
	rowsTrab, err := r.db.Query(queryTrabajadores, planillaID)
	if err != nil {
		log.Println("❌ ERROR EN queryTrabajadores ObtenerFormulacionEspecial:", err)
		return conceptos, nil, err
	}
	defer rowsTrab.Close()

	type trabTemp struct {
		detalleID int
		item      models.TrabajadorFormulacionEspecial
	}
	var listaTemp []trabTemp

	for rowsTrab.Next() {
		var tt trabTemp
		tt.item.MontosCustom = make(map[string]float64)
		err := rowsTrab.Scan(
			&tt.detalleID, &tt.item.ContratoID, &tt.item.NombreCompleto, &tt.item.NumeroDocumento,
			&tt.item.PuestoNombre, &tt.item.UnidadOrganicaNombre, &tt.item.RegimenNombre,
			&tt.item.MetaCodigo, &tt.item.MetaDescripcion,
		)
		if err == nil {
			listaTemp = append(listaTemp, tt)
		}
	}
	if err := rowsTrab.Err(); err != nil {
		return conceptos, nil, err
	}

	var trabajadores []models.TrabajadorFormulacionEspecial
	for _, tt := range listaTemp {
		rowsC, err := r.db.Query(`SELECT concepto_tenant_id, monto FROM planilla_conceptos WHERE planilla_detalle_id = $1`, tt.detalleID)
		if err == nil {
			for rowsC.Next() {
				var cTenantID sql.NullInt64
				var monto float64
				if err := rowsC.Scan(&cTenantID, &monto); err == nil && cTenantID.Valid {
					tt.item.MontosCustom[fmt.Sprintf("%d", cTenantID.Int64)] = monto
				}
			}
			if err := rowsC.Err(); err != nil {
				rowsC.Close()
				return conceptos, nil, err
			}
			rowsC.Close()
		}
		trabajadores = append(trabajadores, tt.item)
	}

	return conceptos, trabajadores, nil
}

// ObtenerConceptosSunatAgrupados obtiene la lista agrupada de conceptos para auditar códigos SUNAT en una planilla
func (r *PlanillaRepository) ObtenerConceptosSunatAgrupados(planillaID int, tenantID int) ([]models.ConceptoSunatAgrupado, error) {
	query := `
		SELECT 
			pc.concepto_tenant_id,
			COALESCE(pc.maestro_id, 0) AS maestro_id,
			COALESCE(NULLIF(pc.codigo_sunat, ''), cm.codigo, '') AS codigo_sunat_actual,
			COALESCE(pc.nombre_en_boleta, ct.nombre_personalizado, 'CONCEPTO SIN NOMBRE') AS nombre_concepto,
			pc.tipo_concepto,
			COUNT(DISTINCT pd.id) AS total_trabajadores,
			COALESCE(SUM(pc.monto), 0.00) AS total_monto,
			COALESCE(ct.concepto_id, 0) AS maestro_id_original
		FROM planilla_conceptos pc
		INNER JOIN planilla_detalles pd ON pc.planilla_detalle_id = pd.id
		INNER JOIN planillas p ON pd.planilla_id = p.id
		LEFT JOIN conceptos_tenant ct ON pc.concepto_tenant_id = ct.id
		LEFT JOIN conceptos_maestros cm ON pc.maestro_id = cm.id
		WHERE p.id = $1 AND p.tenant_id = $2
		GROUP BY 
			pc.concepto_tenant_id, 
			pc.maestro_id, 
			pc.codigo_sunat, 
			cm.codigo, 
			pc.nombre_en_boleta, 
			ct.nombre_personalizado, 
			pc.tipo_concepto, 
			ct.concepto_id
		ORDER BY 
			CASE pc.tipo_concepto 
				WHEN 'INGRESO' THEN 1 
				WHEN 'RETENCION' THEN 2 
				WHEN 'APORTE' THEN 3 
				ELSE 4 
			END, 
			nombre_concepto ASC
	`
	rows, err := r.db.Query(query, planillaID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.ConceptoSunatAgrupado
	for rows.Next() {
		var item models.ConceptoSunatAgrupado
		var cTenantID sql.NullInt64
		err := rows.Scan(
			&cTenantID,
			&item.MaestroID,
			&item.CodigoSunatActual,
			&item.NombreConcepto,
			&item.TipoConcepto,
			&item.TotalTrabajadores,
			&item.TotalMonto,
			&item.MaestroIDOriginal,
		)
		if err != nil {
			return nil, err
		}
		if cTenantID.Valid {
			idVal := int(cTenantID.Int64)
			item.ConceptoTenantID = &idVal
		}
		lista = append(lista, item)
	}
	return lista, nil
}

// ObtenerMaestrosSunat obtiene todos los conceptos oficiales de la Tabla 22 de SUNAT
func (r *PlanillaRepository) ObtenerMaestrosSunat() ([]models.ConceptoMaestro, error) {
	query := `
		SELECT id, parent_id, codigo, codigo_interno, descripcion, tipo, activo, origen
		FROM conceptos_maestros
		WHERE origen = 'sunat' AND activo = true
		ORDER BY 
			CASE tipo 
				WHEN 'INGRESO' THEN 1 
				WHEN 'DESCUENTO' THEN 2 
				WHEN 'APORTE_TRABAJADOR' THEN 3 
				WHEN 'APORTE_EMPLEADOR' THEN 4 
				ELSE 5 
			END, 
			codigo ASC
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.ConceptoMaestro
	for rows.Next() {
		var m models.ConceptoMaestro
		var parentID sql.NullInt64
		err := rows.Scan(
			&m.ID,
			&parentID,
			&m.Codigo,
			&m.CodigoInterno,
			&m.Descripcion,
			&m.Tipo,
			&m.Activo,
			&m.Origen,
		)
		if err != nil {
			return nil, err
		}
		if parentID.Valid {
			pID := int(parentID.Int64)
			m.ParentID = &pID
		}
		lista = append(lista, m)
	}
	return lista, nil
}

// ActualizarCodigoSunatConceptoMasivo actualiza en lote el código SUNAT y maestro_id para un concepto dentro de una planilla
func (r *PlanillaRepository) ActualizarCodigoSunatConceptoMasivo(planillaID int, tenantID int, conceptoTenantID *int, nombreEnBoleta string, nuevoMaestroID int, actualizarDefault bool) error {
	// 1. Validar estado de la planilla
	var estado string
	err := r.db.QueryRow(`SELECT estado FROM planillas WHERE id = $1 AND tenant_id = $2`, planillaID, tenantID).Scan(&estado)
	if err != nil {
		return fmt.Errorf("planilla no encontrada: %w", err)
	}
	if estado == "CERRADA" {
		return fmt.Errorf("la planilla se encuentra CERRADA y no permite modificaciones")
	}

	// 2. Resolver código oficial del maestro SUNAT
	var nuevoCodigoSunat string
	err = r.db.QueryRow(`SELECT codigo FROM conceptos_maestros WHERE id = $1 AND origen = 'sunat'`, nuevoMaestroID).Scan(&nuevoCodigoSunat)
	if err != nil {
		return fmt.Errorf("código maestro SUNAT no válido: %w", err)
	}

	// 3. Ejecutar actualización transaccional
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if conceptoTenantID != nil && *conceptoTenantID > 0 {
		_, err = tx.Exec(`
			UPDATE planilla_conceptos pc
			SET codigo_sunat = $1,
			    maestro_id = $2
			FROM planilla_detalles pd
			WHERE pc.planilla_detalle_id = pd.id
			  AND pd.planilla_id = $3
			  AND pc.concepto_tenant_id = $4
		`, nuevoCodigoSunat, nuevoMaestroID, planillaID, *conceptoTenantID)
		if err != nil {
			return fmt.Errorf("error actualizando conceptos de planilla: %w", err)
		}

		if actualizarDefault {
			_, err = tx.Exec(`
				UPDATE conceptos_tenant
				SET concepto_id = $1,
				    updated_at = CURRENT_TIMESTAMP
				WHERE id = $2 AND tenant_id = $3
			`, nuevoMaestroID, *conceptoTenantID, tenantID)
			if err != nil {
				return fmt.Errorf("error actualizando concepto predeterminado: %w", err)
			}
		}
	} else {
		_, err = tx.Exec(`
			UPDATE planilla_conceptos pc
			SET codigo_sunat = $1,
			    maestro_id = $2
			FROM planilla_detalles pd
			WHERE pc.planilla_detalle_id = pd.id
			  AND pd.planilla_id = $3
			  AND pc.concepto_tenant_id IS NULL
			  AND pc.nombre_en_boleta = $4
		`, nuevoCodigoSunat, nuevoMaestroID, planillaID, nombreEnBoleta)
		if err != nil {
			return fmt.Errorf("error actualizando conceptos de planilla sin tenant: %w", err)
		}
	}

	return tx.Commit()
}
