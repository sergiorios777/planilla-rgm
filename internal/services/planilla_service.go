package services

import (
	"errors"
	"log"
	"planilla-rgm/internal/calculadoras"
	"planilla-rgm/internal/helpers"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
	"strconv"
	"strings"
	"time"
)

type PlanillaService struct {
	Repo *repository.PlanillaRepository
}

func NewPlanillaService(repo *repository.PlanillaRepository) *PlanillaService {
	return &PlanillaService{Repo: repo}
}

func (s *PlanillaService) Eliminar(planillaID int, tenantID int) error {
	return s.Repo.Eliminar(planillaID, tenantID)
}

func (s *PlanillaService) Procesar(planillaID int, tenantID int) error {
	// 0. Validar estado antes de comenzar
	estado, err := s.Repo.ObtenerEstado(planillaID, tenantID)
	if err == nil && estado == "CERRADA" {
		return errors.New("operación denegada: la planilla ya está CERRADA y no puede recalcularse")
	}

	// 1. EXTRAER CONTEXTO GLOBAL (FASE B)

	// log.Println("------------- Iniciando Procesar Planilla ------------- ")
	anio, mes, _ := s.Repo.ObtenerPeriodoPlanilla(planillaID, tenantID)
	// log.Println("Anio: ", anio, " Mes: ", mes)
	parametros, _ := s.Repo.ObtenerParametrosGlobales(anio, mes)
	// log.Println("Parametros: ", parametros)
	contratos, _ := s.Repo.ObtenerContratosActivosPlanilla(tenantID, anio, mes)
	// log.Println("Contratos: ", contratos)
	mapaCodigos, _ := s.Repo.ObtenerMapaCodigosID()
	// log.Println("Mapa Codigos: ", mapaCodigos)
	mapaAfectacionesGlobal, _ := s.Repo.ObtenerAfectacionesGlobales()
	// log.Println("Mapa Afectaciones: ", mapaAfectacionesGlobal)
	mapaOcurrenciasGlobal, _ := s.Repo.ObtenerOcurrenciasParaProcesar(tenantID, planillaID)
	// log.Println("Mapa Ocurrencias: ", mapaOcurrenciasGlobal)
	mapaTasasAFPMes, _ := s.Repo.ObtenerTasasAFPMes(anio, mes)
	// log.Println("Mapa Tasas AFP: ", mapaTasasAFPMes)

	// Masivos para evitar N+1
	var contratoIDs []int
	for _, c := range contratos {
		contratoIDs = append(contratoIDs, c.ID)
	}
	mapaConceptosContrato, _ := s.Repo.ObtenerConceptosPorContratoMasivo(contratoIDs)
	// log.Println("Mapa Conceptos: ", mapaConceptosPuesto)
	mapa5taPrevias, _ := s.Repo.ObtenerRetencionesPreviasMasivo(contratoIDs, anio, mes)
	// log.Println("Mapa 5ta Previas: ", mapa5taPrevias)
	mapaIngresosPrevios, _ := s.Repo.ObtenerIngresosPreviosMasivo(contratoIDs, anio, mes)
	reglasFinanciamiento, _ := s.Repo.ObtenerReglasFinanciamientoPorTenant(tenantID)

	// =========================================================
	// 2. INICIO DE LA CONCURRENCIA (WORKER POOL)
	// =========================================================
	numWorkers := 8
	jobs := make(chan models.JobPlanilla, len(contratos))
	results := make(chan models.ResultPlanilla, len(contratos))

	// Lanzamos los Workers
	for w := 1; w <= numWorkers; w++ {
		go func() {
			for job := range jobs {
				boleta, err := s.calcularBoletaContrato(job)
				results <- models.ResultPlanilla{Boleta: boleta, Error: err}
			}
		}()
	}

	// Enviamos los trabajos
	for _, contrato := range contratos {
		jobs <- models.JobPlanilla{
			Contrato:               contrato,
			ConceptosPlaza:         mapaConceptosContrato[contrato.ID],
			Ocurrencias:            mapaOcurrenciasGlobal[contrato.ID],
			TasasAFP:               mapaTasasAFPMes[contrato.AfpID],
			Retenciones5taPrevias:  mapa5taPrevias[contrato.ID],
			IngresosPrevios:        mapaIngresosPrevios[contrato.ID],
			MesActual:              mes,
			Anio:                   anio,
			ParametrosGlobales:     parametros,
			MapaCodigos:            mapaCodigos,
			MapaAfectacionesGlobal: mapaAfectacionesGlobal,
			ReglasFinanciamiento:   reglasFinanciamiento,
		}
	}
	close(jobs) // Cerramos para que los workers sepan que no hay más

	// Recolectamos resultados
	var boletasFinales []models.BoletaResultado
	for i := 0; i < len(contratos); i++ {
		res := <-results
		if res.Error == nil {
			boletasFinales = append(boletasFinales, res.Boleta)
		}
	}

	// 3. GUARDADO FINAL
	log.Printf("Guardando %d boletas calculadas de forma concurrente...", len(boletasFinales))
	return s.Repo.GuardarPlanillaCalculada(planillaID, boletasFinales)
}

// clasificarIngresos5ta procesa en memoria los montos de Remuneración Mensual, No Mensual y Extraordinaria
func (s *PlanillaService) clasificarIngresos5ta(conceptosPlaza []models.ConceptoPlanilla, derivados5ta []int, mesActual int) (float64, float64, float64) {
	mensual := 0.00
	noMensual := 0.00
	extraDelMes := 0.00
	mesActualStr := strconv.Itoa(mesActual)

	// Mapa rápido para saber si un concepto afecta a 5ta
	afectosA5ta := make(map[int]bool)
	for _, id := range derivados5ta {
		afectosA5ta[id] = true
	}

	for _, cp := range conceptosPlaza {
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
				// Mensual
				mensual += cp.Monto
			} else {
				// No Mensual (deducimos los meses ya pasados)
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

// obtenerBaseImponible suma los ingresos procesados que afectan a un concepto específico
func (s *PlanillaService) obtenerBaseImponible(maestroID int, ingresosProcesados map[string]float64, mapaAfectaciones map[int][]int) float64 {
	suma := 0.0
	// Buscamos en nuestro "Mapa de la Fase B" qué ingresos afectan a este maestroID
	if derivados, existe := mapaAfectaciones[maestroID]; existe {
		for _, derivadoID := range derivados {
			// Sumamos el monto que guardamos en la Pasada 1
			suma += ingresosProcesados[strconv.Itoa(derivadoID)]
		}
	}
	return suma
}

func (s *PlanillaService) calcularBoletaContrato(job models.JobPlanilla) (models.BoletaResultado, error) {
	boleta := models.BoletaResultado{
		ContratoID:                     job.Contrato.ID,
		TrabajadorNombreCompleto:       job.Contrato.TrabajadorNombreCompleto,
		TrabajadorNumeroDocumento:      job.Contrato.TrabajadorNumeroDocumento,
		PuestoCodigoAirhsp:             job.Contrato.PuestoCodigoAirhsp,
		PuestoNombre:                   job.Contrato.PuestoNombre,
		OrganigramaDocumentoAprobacion: job.Contrato.OrganigramaDocumentoAprobacion,
		UnidadOrganicaNombre:           job.Contrato.UnidadOrganicaNombre,
		UnidadOrganicaTipo:             job.Contrato.UnidadOrganicaTipo,
		SueldoBasicoHistorico:          job.Contrato.SueldoBasicoHistorico,
	}

	mesActualStr := strconv.Itoa(job.MesActual)

	// Preparamos nuestro "Maletín" con los datos que vienen en el Job
	ctxTrabajador := models.ContextoCalculo{
		ParametrosGlobales:    job.ParametrosGlobales,
		RegimenCodigo:         job.Contrato.Regimen,
		IngresosProcesados:    make(map[string]float64),
		MesActual:             job.MesActual,
		Retenciones5taPrevias: job.Retenciones5taPrevias, // OJO: revisar luego
		IngresosPrevios:       job.IngresosPrevios,
		RegimenPensionario:    job.Contrato.RegimenPensionario,
	}

	// 💡 ASIGNACIÓN DINÁMICA DE TASAS AFP
	if job.Contrato.RegimenPensionario == "AFP" && job.Contrato.AfpID > 0 {
		ctxTrabajador.TasaAfpAporte = job.TasasAFP.Aporte
		ctxTrabajador.TasaAfpPrima = job.TasasAFP.Prima
		if job.Contrato.AfpTipoComision == "FLUJO" {
			ctxTrabajador.TasaAfpComision = job.TasasAFP.Flujo
		} else {
			ctxTrabajador.TasaAfpComision = job.TasasAFP.Mixta
		}
	}

	// Calculamos días y factor de prorrateo
	diasLaborados := s.calcularDiasLaborados(job.Contrato.FechaInicio, job.Contrato.FechaFin, job.Anio, job.MesActual)
	factorProrrateo := diasLaborados / 30.0

	// Calcular gratificación y bonificación extraordinaria si corresponde (DL 728 en Julio/Diciembre)
	var calculadaGrati, calculadaBonExt float64
	var calculadoGratiDone bool
	if (job.MesActual == 7 || job.MesActual == 12) && job.Contrato.Regimen == "728" {
		var semDesde, semHasta time.Time
		if job.MesActual == 7 {
			semDesde = time.Date(job.Anio, time.January, 1, 0, 0, 0, 0, time.UTC)
			semHasta = time.Date(job.Anio, time.June, 30, 23, 59, 59, 0, time.UTC)
		} else {
			semDesde = time.Date(job.Anio, time.July, 1, 0, 0, 0, 0, time.UTC)
			semHasta = time.Date(job.Anio, time.December, 31, 23, 59, 59, 0, time.UTC)
		}

		mesesCalculados := calculadoras.CalcularMesesSemestreGratificacion(job.Contrato.FechaInicio, job.Contrato.FechaFin, semDesde, semHasta)

		sueldoBase := job.Contrato.SueldoBasicoHistorico
		if sueldoBase <= 0 {
			sueldoBase = 1025.00
		}
		var asignacionFamiliar float64
		for _, cp := range job.ConceptosPlaza {
			if cp.CodigoInterno == "ASIG_FAM_DL728" {
				rmv, existe := job.ParametrosGlobales["RMV"]
				if !existe {
					rmv = 1025.00
				}
				asignacionFamiliar = calculadoras.CalcularAsignacionFamiliar(rmv)
			}
		}

		remComputable := sueldoBase + asignacionFamiliar
		calculadaGrati, calculadaBonExt = calculadoras.CalcularGratificacionDL728(remComputable, mesesCalculados)
		calculadoGratiDone = true
	}

	var calculadaGratiCAS, aporteEsSaludGratiCAS float64
	var calculadoGratiCASDone bool
	if (job.MesActual == 7 || job.MesActual == 12) && (job.Contrato.Regimen == "1057" || job.Contrato.Regimen == "CAS") {
		var semDesde, semHasta time.Time
		if job.MesActual == 7 {
			semDesde = time.Date(job.Anio, time.January, 1, 0, 0, 0, 0, time.UTC)
			semHasta = time.Date(job.Anio, time.June, 30, 23, 59, 59, 0, time.UTC)
		} else {
			semDesde = time.Date(job.Anio, time.July, 1, 0, 0, 0, 0, time.UTC)
			semHasta = time.Date(job.Anio, time.December, 31, 23, 59, 59, 0, time.UTC)
		}

		meses, dias := helpers.CalcularMesesYDiasSemestreGratificacionCAS(job.Contrato.FechaInicio, job.Contrato.FechaFin, semDesde, semHasta)

		remComputable := job.Contrato.SueldoBasicoHistorico
		if remComputable <= 0 {
			remComputable = 1025.00
		}

		if s.Repo != nil && s.Repo.GetDB() != nil {
			baseRepo := repository.NewBaseRegimenRepository(s.Repo.GetDB())
			if baseRepo != nil {
				montoDataDriven, err := baseRepo.ObtenerMontoVariable(job.Contrato.TenantID, job.Contrato.PuestoID, job.Contrato.RegimenID, "GRATIFICACION", "RETRIBUCION_MENSUAL")
				if err == nil && montoDataDriven > 0 {
					remComputable = montoDataDriven
				}
			}
		}

		calculadaGratiCAS, aporteEsSaludGratiCAS = calculadoras.CalcularGratificacionDL1057(remComputable, job.MesActual, job.Anio, meses, dias)
		calculadoGratiCASDone = true
	}

	// --- PASADA 1: PROCESAR INGRESOS ---
	for _, cp := range job.ConceptosPlaza {
		if strings.ToUpper(cp.Tipo) != "INGRESO" {
			continue
		}

		// Filtro de meses
		aplica := false
		for _, mStr := range strings.Split(cp.Frecuencia, ",") {
			if strings.TrimSpace(mStr) == mesActualStr {
				aplica = true
				break
			}
		}
		if !aplica {
			continue
		}

		// Aplica prorrateo solo si NO es extraordinario
		montoProporcional := cp.Monto
		isGratificationConcept := false
		if calculadoGratiDone {
			if cp.CodigoInterno == "GRATI_JUL_DL_728" || cp.CodigoInterno == "GRATI_DIC_DL_728" || cp.CodigoSunat == "0406" {
				montoProporcional = calculadaGrati
				isGratificationConcept = true
			} else if cp.CodigoInterno == "BON_EXTR_JUL_DL_728" || cp.CodigoInterno == "BON_EXTR_DIC_DL_728" || cp.CodigoSunat == "0312" {
				montoProporcional = calculadaBonExt
				isGratificationConcept = true
			}
		}
		if calculadoGratiCASDone {
			if cp.CodigoInterno == "GRATI_JUL_DL_1057" || cp.CodigoInterno == "GRATI_DIC_DL_1057" || cp.CodigoSunat == "2006" {
				montoProporcional = calculadaGratiCAS
				isGratificationConcept = true
			}
		}

		isAsigFam := cp.CodigoInterno == "ASIG_FAM_DL728"
		if isAsigFam {
			rmv, existe := job.ParametrosGlobales["RMV"]
			if !existe {
				rmv = 1025.00
			}
			montoProporcional = calculadoras.CalcularAsignacionFamiliar(rmv)
		}

		if !cp.EsExtraordinario && !isGratificationConcept && !isAsigFam {
			montoProporcional = cp.Monto * factorProrrateo
		}

		boleta.TotalIngresos += montoProporcional
		ctxTrabajador.IngresosProcesados[strconv.Itoa(cp.MaestroID)] = montoProporcional

		var ctIDVal *int
		if cp.TenantID > 0 {
			v := cp.TenantID
			ctIDVal = &v
		}
		metaID, rubroID := s.resolverMetaYRubro("INGRESO", cp, job.Contrato, job.Contrato.RegimenID, job.ReglasFinanciamiento)
		boleta.LineasConceptos = append(boleta.LineasConceptos, models.PlanillaConcepto{
			ConceptoTenantID: ctIDVal,
			MetaID:           metaID,
			FuenteRubroID:    rubroID,
			TipoConcepto:     "INGRESO",
			Monto:            montoProporcional,
			MaestroID:        cp.MaestroID,
			CodigoSunat:      cp.CodigoSunat,
			NombreEnBoleta:   cp.Nombre,
		})
	}

	// --- PASADA 1.5: ASISTENCIA ---
	idFaltas := job.MapaCodigos["0705"]
	remComputable := s.obtenerBaseImponible(idFaltas, ctxTrabajador.IngresosProcesados, job.MapaAfectacionesGlobal)

	diasFalta, minsTardanza := 0.0, 0.0
	for _, oc := range job.Ocurrencias {
		if oc.Tipo == "INASISTENCIA" {
			diasFalta += oc.Cantidad
		}
		if oc.Tipo == "TARDANZA" {
			minsTardanza += oc.Cantidad
		}
		boleta.OcurrenciasProcesadas = append(boleta.OcurrenciasProcesadas, oc.ID)
	}

	descFaltas := calculadoras.CalcularDescuentoFaltas(remComputable, diasFalta)
	descTardanzas := calculadoras.CalcularDescuentoTardanzas(remComputable, minsTardanza)
	totalDescAsis := descFaltas + descTardanzas

	// --- PASADA 2: RETENCIONES Y APORTES ---
	for _, cp := range job.ConceptosPlaza {
		tipo := strings.ToUpper(cp.Tipo)
		if tipo == "INGRESO" {
			continue
		}

		// Filtro con excepción de asistencia
		esAsis := (cp.CodigoInterno == "0704" || cp.CodigoInterno == "0705")
		aplica := esAsis
		if !esAsis {
			for _, mStr := range strings.Split(cp.Frecuencia, ",") {
				if strings.TrimSpace(mStr) == mesActualStr {
					aplica = true
					break
				}
			}
		}
		if !aplica {
			continue
		}

		base := s.obtenerBaseImponible(cp.MaestroID, ctxTrabajador.IngresosProcesados, job.MapaAfectacionesGlobal)
		base -= totalDescAsis
		if base < 0 {
			base = 0
		}

		montoFinal := cp.Monto
		switch cp.CodigoInterno {
		case "0704":
			montoFinal = descTardanzas
		case "0705":
			montoFinal = descFaltas
		case "0804":
			montoFinal = calculadoras.CalcularEsSalud(base, ctxTrabajador) + aporteEsSaludGratiCAS
		case "S101":
			montoFinal = calculadoras.CalcularRenta4ta(base, ctxTrabajador)
		case "0605":
			// Lógica de 5ta usando datos del job
			remM, remNM, extra := s.clasificarIngresos5ta(job.ConceptosPlaza, job.MapaAfectacionesGlobal[cp.MaestroID], job.MesActual)
			ctxTrabajador.RemuneracionNoMensual = remNM
			ctxTrabajador.IngresoExtraordinarioDelMes = extra
			ctxTrabajador.Retenciones5taPrevias = job.Retenciones5taPrevias
			montoFinal = calculadoras.CalcularRenta5ta(remM, ctxTrabajador)
		case "0607":
			montoFinal = calculadoras.CalcularONP(base, ctxTrabajador)
		case "0608":
			montoFinal = calculadoras.CalcularAFPAporte(base, ctxTrabajador)
		case "0606":
			montoFinal = calculadoras.CalcularAFPPrima(base, ctxTrabajador)
		case "0601":
			montoFinal = calculadoras.CalcularAFPComision(base, ctxTrabajador)
		}

		if esAsis && montoFinal <= 0 {
			continue
		}

		if tipo == "RETENCION" {
			boleta.TotalRetenciones += montoFinal
		}
		if tipo == "APORTE" {
			boleta.TotalAportes += montoFinal
		}

		var ctIDVal *int
		if cp.TenantID > 0 {
			v := cp.TenantID
			ctIDVal = &v
		}
		metaID, rubroID := s.resolverMetaYRubro(tipo, cp, job.Contrato, job.Contrato.RegimenID, job.ReglasFinanciamiento)
		boleta.LineasConceptos = append(boleta.LineasConceptos, models.PlanillaConcepto{
			ConceptoTenantID: ctIDVal,
			MetaID:           metaID,
			FuenteRubroID:    rubroID,
			TipoConcepto:     tipo,
			Monto:            montoFinal,
			MaestroID:        cp.MaestroID,
			CodigoSunat:      cp.CodigoSunat,
			NombreEnBoleta:   cp.Nombre,
		})
	}

	boleta.NetoPagar = boleta.TotalIngresos - boleta.TotalRetenciones
	return boleta, nil
}

// calcularDiasLaborados determina cuántos días del mes (base 30) se deben pagar
func (s *PlanillaService) calcularDiasLaborados(fechaInicio time.Time, fechaFin *time.Time, anio int, mes int) float64 {
	primerDiaMes := time.Date(anio, time.Month(mes), 1, 0, 0, 0, 0, time.UTC)

	// ELIMINADO: ultimoDiaMes := primerDiaMes.AddDate(0, 1, -1)

	// Punto de partida: El trabajador inicia el día 1 del mes, a menos que su contrato diga lo contrario
	diaInicioEfectivo := 1
	if fechaInicio.After(primerDiaMes) || fechaInicio.Equal(primerDiaMes) {
		if int(fechaInicio.Month()) == mes && fechaInicio.Year() == anio {
			diaInicioEfectivo = fechaInicio.Day()
		}
	}

	// Punto de fin: El trabajador termina el día 30, a menos que el contrato termine este mes
	diaFinEfectivo := 30
	if fechaFin != nil {
		if int(fechaFin.Month()) == mes && fechaFin.Year() == anio {
			diaFinEfectivo = fechaFin.Day()
			if diaFinEfectivo > 30 {
				diaFinEfectivo = 30
			}
		}
	}

	dias := float64(diaFinEfectivo - diaInicioEfectivo + 1)
	if dias < 0 {
		return 0
	}
	if dias > 30 {
		return 30
	}
	return dias
}

// SimularCostoMensualPuesto crea un entorno ideal (sin faltas) para calcular cuánto le cuesta
// a la Municipalidad (Genérica 2.1) un puesto específico en un mes determinado.
func (s *PlanillaService) SimularCostoMensualPuesto(
	puestoID int,
	regimenCodigo string,
	conceptosPlaza []models.ConceptoPlanilla,
	anio int,
	mes int,
	parametros map[string]float64,
	mapaCodigos map[string]int,
	mapaAfectaciones map[int][]int,
) (float64, float64, error) {

	// 1. Crear fechas ideales (Mes completo)
	primerDia := time.Date(anio, time.Month(mes), 1, 0, 0, 0, 0, time.UTC)
	ultimoDia := primerDia.AddDate(0, 1, -1) // Truco de Go para ir al último día del mes

	// 2. Crear un contrato ficticio perfecto
	contratoIdeal := models.ContratoPlanilla{
		ID:                 999999, // ID Ficticio para evitar cruces
		PuestoID:           puestoID,
		Regimen:            regimenCodigo,
		FechaInicio:        primerDia,
		FechaFin:           &ultimoDia,
		RegimenPensionario: "ONP", // Indiferente para la Genérica 2.1 (es retención, no aporte)
	}

	// 3. Empaquetar el "Job" para nuestro motor de cálculo
	jobIdeal := models.JobPlanilla{
		Contrato:               contratoIdeal,
		ConceptosPlaza:         conceptosPlaza,
		Ocurrencias:            []models.OcurrenciaAsistencia{}, // 0 faltas, 0 tardanzas
		MesActual:              mes,
		Anio:                   anio,
		ParametrosGlobales:     parametros,
		MapaCodigos:            mapaCodigos,
		MapaAfectacionesGlobal: mapaAfectaciones,
		// Los acumulados de 5ta los dejamos en 0 porque para proyecciones presupuestales
		// la renta de 5ta (que es retención) no afecta el costo total del empleador.
		Retenciones5taPrevias: 0,
		IngresosPrevios:       0,
	}

	// 4. Ejecutar el motor de cálculo exacto
	boleta, err := s.calcularBoletaContrato(jobIdeal)
	if err != nil {
		return 0, 0, err
	}

	// 5. Retornar los dos componentes del costo municipal:
	// Ingresos (Sueldo, Aguinaldos) y Aportes (EsSalud)
	return boleta.TotalIngresos, boleta.TotalAportes, nil
}

// resolverMetaYRubro determina la meta_id y fuente_rubro_id para una línea de concepto
func (s *PlanillaService) resolverMetaYRubro(
	tipoConcepto string,
	cp models.ConceptoPlanilla,
	puesto models.ContratoPlanilla,
	regimenID int,
	reglas []models.ReglaFinanciamientoConcepto,
) (*int, *int) {
	tipoUpper := strings.ToUpper(strings.TrimSpace(tipoConcepto))
	if tipoUpper == "RETENCION" {
		return nil, nil
	}

	conceptoTenantID := cp.ConceptoTenantID
	if conceptoTenantID == 0 {
		conceptoTenantID = cp.TenantID
	}

	// Buscar regla de excepción activa
	var reglaEncontrada *models.ReglaFinanciamientoConcepto
	for i := range reglas {
		r := &reglas[i]
		if !r.Activo {
			continue
		}
		if r.ConceptoTenantID != conceptoTenantID {
			continue
		}

		if r.RegimenID != nil && *r.RegimenID > 0 {
			if *r.RegimenID == regimenID {
				reglaEncontrada = r
				break
			}
		} else if reglaEncontrada == nil {
			reglaEncontrada = r
		}
	}

	var metaID *int = puesto.MetaID
	var fuenteRubroID *int = puesto.FuenteRubroID

	if reglaEncontrada != nil {
		if reglaEncontrada.MetaID != nil && *reglaEncontrada.MetaID > 0 {
			metaID = reglaEncontrada.MetaID
		}
		if reglaEncontrada.FuenteRubroID != nil && *reglaEncontrada.FuenteRubroID > 0 {
			fuenteRubroID = reglaEncontrada.FuenteRubroID
		}
	}

	return metaID, fuenteRubroID
}
