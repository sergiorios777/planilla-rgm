package services

import (
	"planilla-rgm/internal/calculadoras"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
	"strconv"
	"strings"
)

type PlanillaService struct {
	Repo *repository.PlanillaRepository
}

func NewPlanillaService(repo *repository.PlanillaRepository) *PlanillaService {
	return &PlanillaService{Repo: repo}
}

func (s *PlanillaService) Procesar(planillaID int, tenantID int) error {

	// 1. EXTRAER CONTEXTO GLOBAL
	anio, mes, err := s.Repo.ObtenerPeriodoPlanilla(planillaID, tenantID)
	if err != nil {
		return err
	}

	parametros, err := s.Repo.ObtenerParametrosGlobales(anio, mes)
	if err != nil {
		return err
	}

	contratos, err := s.Repo.ObtenerContratosActivosPlanilla(tenantID, anio, mes)
	if err != nil {
		return err
	}

	// 2. MAGIA FASE B: Traemos todas las reglas de afectación a la memoria (1 sola consulta)
	mapaOcurrenciasGlobal, _ := s.Repo.ObtenerOcurrenciasParaProcesar(tenantID, planillaID)
	mapaAfectacionesGlobal, _ := s.Repo.ObtenerAfectacionesGlobales()
	mapaTasasAFPMes, _ := s.Repo.ObtenerTasasAFPMes(anio, mes)

	var boletasFinales []models.BoletaResultado

	// 3. BUCLE DE CÁLCULO EN MEMORIA (Ultrarápido)
	for _, contrato := range contratos {
		conceptosPlaza, _ := s.Repo.ObtenerConceptosPuesto(contrato.PuestoID)

		boleta := models.BoletaResultado{ContratoID: contrato.ID}

		// Ahora SÍ usamos el maletín (desaparece el error de Go)
		ctxTrabajador := models.ContextoCalculo{
			ParametrosGlobales: parametros,
			RegimenCodigo:      contrato.Regimen,
			IngresosProcesados: make(map[string]float64),
			MesActual:          mes,
			RegimenPensionario: contrato.RegimenPensionario,
		}

		// ASIGNACIÓN DINÁMICA DE TASAS AFP (Cero consultas SQL)
		if contrato.RegimenPensionario == "AFP" && contrato.AfpID > 0 {
			tasasAFP := mapaTasasAFPMes[contrato.AfpID]

			ctxTrabajador.TasaAfpAporte = tasasAFP.Aporte
			ctxTrabajador.TasaAfpPrima = tasasAFP.Prima

			if contrato.AfpTipoComision == "FLUJO" {
				ctxTrabajador.TasaAfpComision = tasasAFP.Flujo
			} else {
				ctxTrabajador.TasaAfpComision = tasasAFP.Mixta
			}
		}

		// Convertimos el mes de la planilla a string para compararlo con la frecuencia
		mesActualStr := strconv.Itoa(mes)

		// Extraemos las ocurrencias específicas de este trabajador
		ocurrenciasTrabajador := mapaOcurrenciasGlobal[contrato.ID]

		// Base para calcular el valor del día/minuto
		remuneracionComputable := 0.00

		// --- PASADA 1: PROCESAR INGRESOS DEL MES ---
		for _, cp := range conceptosPlaza {
			tipoUpper := strings.ToUpper(cp.Tipo)

			if tipoUpper == "INGRESO" {
				// 1. VALIDACIÓN DE FRECUENCIA (El Filtro)
				aplicaEsteMes := false
				mesesFrecuencia := strings.Split(cp.Frecuencia, ",")

				for _, mStr := range mesesFrecuencia {
					// Limpiamos posibles espacios en blanco (ej: " 7" -> "7")
					if strings.TrimSpace(mStr) == mesActualStr {
						aplicaEsteMes = true
						break
					}
				}

				// Si el concepto no pertenece a este mes, lo ignoramos por completo para esta boleta
				if !aplicaEsteMes {
					continue
				}

				// 2. ACUMULACIÓN (Solo llega aquí si aplica este mes)
				boleta.TotalIngresos += cp.Monto

				// Guardamos en el maletín (Base imponible para la Pasada 2)
				ctxTrabajador.IngresosProcesados[strconv.Itoa(cp.MaestroID)] = cp.Monto

				// Registramos la línea para el detalle de la boleta
				boleta.LineasConceptos = append(boleta.LineasConceptos, models.PlanillaConcepto{
					ConceptoTenantID: cp.TenantID,
					TipoConcepto:     tipoUpper,
					Monto:            cp.Monto,
				})
			}
		}

		// --- PASADA 1.5: CÁLCULO PREVIO DE ASISTENCIA ---
		diasFalta := 0.00
		minutosTardanza := 0.00
		for _, oc := range ocurrenciasTrabajador {
			if oc.Tipo == "INASISTENCIA" {
				diasFalta += oc.Cantidad
			}
			if oc.Tipo == "TARDANZA" {
				minutosTardanza += oc.Cantidad
			}
			boleta.OcurrenciasProcesadas = append(boleta.OcurrenciasProcesadas, oc.ID)
		}

		// Usamos el archivo calculadoras/asistencia.go que creamos
		descuentoFaltas := calculadoras.CalcularDescuentoFaltas(remuneracionComputable, diasFalta)
		descuentoTardanzas := calculadoras.CalcularDescuentoTardanzas(remuneracionComputable, minutosTardanza)
		totalDescuentosAsistencia := descuentoFaltas + descuentoTardanzas

		// --- PASADA 2: RETENCIONES Y APORTES ---
		for _, cp := range conceptosPlaza {
			tipoUpper := strings.ToUpper(cp.Tipo)
			if tipoUpper == "INGRESO" {
				continue
			}

			// =========================================================
			// 1. VALIDACIÓN DE FRECUENCIA (El Filtro)
			// =========================================================
			aplicaEsteMes := false
			mesesFrecuencia := strings.Split(cp.Frecuencia, ",")

			for _, mStr := range mesesFrecuencia {
				// Utilizamos el 'mesActualStr' que definiste en la Pasada 1
				if strings.TrimSpace(mStr) == mesActualStr {
					aplicaEsteMes = true
					break
				}
			}

			// Si la retención o aporte no corresponde a este mes, lo ignoramos
			if !aplicaEsteMes {
				continue
			}

			// =========================================================
			// 2. CÁLCULO DE LA BASE IMPONIBLE
			// =========================================================
			baseImponible := 0.00
			if derivados, existe := mapaAfectacionesGlobal[cp.MaestroID]; existe {
				for _, derivadoID := range derivados {
					// Sumamos extrayendo de la Pasada 1 (Cero consultas SQL)
					baseImponible += ctxTrabajador.IngresosProcesados[strconv.Itoa(derivadoID)]
				}
			}

			// CLAVE LEGAL: Las inasistencias reducen la base imponible general
			baseImponible -= totalDescuentosAsistencia
			if baseImponible < 0 {
				baseImponible = 0
			}

			montoFinal := cp.Monto // Por si es monto fijo

			// =========================================================
			// 3. RUTEO A CALCULADORAS EXPERTAS
			// =========================================================
			switch cp.MaestroCodigo {
			case "0704": // TARDANZAS
				montoFinal = descuentoTardanzas

			case "0705": // INASISTENCIAS
				montoFinal = descuentoFaltas

			case "0804": // ESSALUD
				montoFinal = calculadoras.CalcularEsSalud(baseImponible, ctxTrabajador)

			case "S101": // RENTA DE 4TA
				montoFinal = calculadoras.CalcularRenta4ta(baseImponible, ctxTrabajador)

			case "0605": // RENTA DE 5TA
				// 1. Obtenemos los ingresos que afectan a 5ta desde nuestro mapa rápido
				derivados5ta := mapaAfectacionesGlobal[cp.MaestroID]

				// 2. Clasificamos el dinero en memoria
				remMensual, remNoMensual, extraMes := s.clasificarIngresos5ta(conceptosPlaza, derivados5ta, mes)

				// 3. Extraemos el historial de la BD
				retPrevias, _ := s.Repo.ObtenerRetencionesPrevias(contrato.ID, anio, mes)

				// 4. Llenamos el Maletín
				ctxTrabajador.RemuneracionNoMensual = remNoMensual
				ctxTrabajador.IngresoExtraordinarioDelMes = extraMes
				ctxTrabajador.Retenciones5taPrevias = retPrevias

				// 5. Calculamos
				montoFinal = calculadoras.CalcularRenta5ta(remMensual, ctxTrabajador)

			case "0607": // ONP
				montoFinal = calculadoras.CalcularONP(baseImponible, ctxTrabajador)

			case "0608": // AFP APORTE OBLIGATORIO
				montoFinal = calculadoras.CalcularAFPAporte(baseImponible, ctxTrabajador)

			case "0606": // AFP PRIMA DE SEGURO (Con Tope SBS)
				montoFinal = calculadoras.CalcularAFPPrima(baseImponible, ctxTrabajador)

			case "0601": // AFP COMISIÓN PORCENTUAL
				montoFinal = calculadoras.CalcularAFPComision(baseImponible, ctxTrabajador)
			}

			// =========================================================
			// 4. ACUMULACIÓN EN LA BOLETA FINAL
			// =========================================================
			switch tipoUpper {
			case "RETENCION":
				boleta.TotalRetenciones += montoFinal
			case "APORTE":
				boleta.TotalAportes += montoFinal
			}

			// Agregamos la línea al detalle de la boleta (asegurando de usar montoFinal)
			boleta.LineasConceptos = append(boleta.LineasConceptos, models.PlanillaConcepto{
				ConceptoTenantID: cp.TenantID,
				TipoConcepto:     tipoUpper,
				Monto:            montoFinal, // ¡Ojo aquí! Usamos la variable ya calculada
			})
		}

		boleta.NetoPagar = boleta.TotalIngresos - boleta.TotalRetenciones
		boletasFinales = append(boletasFinales, boleta)
	}

	// 4. GUARDADO EN BLOQUE
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
