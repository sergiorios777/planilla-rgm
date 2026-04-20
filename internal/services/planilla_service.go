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
	mapaAfectacionesGlobal, _ := s.Repo.ObtenerAfectacionesGlobales()

	var boletasFinales []models.BoletaResultado

	// 3. BUCLE DE CÁLCULO EN MEMORIA (Ularrápido)
	for _, contrato := range contratos {
		conceptosPlaza, _ := s.Repo.ObtenerConceptosPuesto(contrato.PuestoID)

		boleta := models.BoletaResultado{ContratoID: contrato.ID}

		// Ahora SÍ usamos el maletín (desaparece el error de Go)
		ctxTrabajador := models.ContextoCalculo{
			ParametrosGlobales: parametros,
			RegimenCodigo:      contrato.Regimen,
			IngresosProcesados: make(map[string]float64),
			MesActual:          mes,
		}

		// --- PASADA 1: PROCESAR INGRESOS ---
		for _, cp := range conceptosPlaza {
			tipoUpper := strings.ToUpper(cp.Tipo)

			// (Nota: Aquí aplicarías tu validación de frecuencia por meses)

			if tipoUpper == "INGRESO" {
				boleta.TotalIngresos += cp.Monto
				// Guardamos en el maletín para cruzarlo luego
				ctxTrabajador.IngresosProcesados[strconv.Itoa(cp.MaestroID)] = cp.Monto

				// Agregamos la línea a la boleta final
				boleta.LineasConceptos = append(boleta.LineasConceptos, models.PlanillaConcepto{
					ConceptoTenantID: cp.TenantID, TipoConcepto: tipoUpper, Monto: cp.Monto,
				})
			}
		}

		// --- PASADA 2: RETENCIONES Y APORTES ---
		for _, cp := range conceptosPlaza {
			tipoUpper := strings.ToUpper(cp.Tipo)
			if tipoUpper == "INGRESO" {
				continue
			}

			// 💡 TU IDEA EN ACCIÓN: Calculamos la base imponible cruzando los mapas en memoria
			baseImponible := 0.00
			if derivados, existe := mapaAfectacionesGlobal[cp.MaestroID]; existe {
				for _, derivadoID := range derivados {
					// Sumamos extrayendo de la Pasada 1 (Cero consultas SQL)
					baseImponible += ctxTrabajador.IngresosProcesados[strconv.Itoa(derivadoID)]
				}
			}

			montoFinal := cp.Monto // Por si es monto fijo

			// Llamamos a las calculadoras que hicimos ayer
			switch cp.MaestroCodigo {
			case "0804": // ESSALUD
				montoFinal = calculadoras.CalcularEsSalud(baseImponible, ctxTrabajador)
			case "S101": // RENTA DE 4TA
				montoFinal = calculadoras.CalcularRenta4ta(baseImponible, ctxTrabajador)
			case "0605": // RENTA DE 5TA
				// 1. Obtenemos los ingresos que afectan a 5ta desde nuestro mapa rápido (¡Cero BD!)
				derivados5ta := mapaAfectacionesGlobal[cp.MaestroID]

				// 2. Clasificamos el dinero en memoria
				remMensual, remNoMensual, extraMes := s.clasificarIngresos5ta(conceptosPlaza, derivados5ta, mes)

				// 3. Extraemos el historial (Esto SÍ requiere BD, asegúrate de tener
				//    la función ObtenerRetencionesPrevias en tu repository)
				retPrevias, _ := s.Repo.ObtenerRetencionesPrevias(contrato.ID, anio, mes)

				// 4. Llenamos el Maletín
				ctxTrabajador.RemuneracionNoMensual = remNoMensual
				ctxTrabajador.IngresoExtraordinarioDelMes = extraMes
				ctxTrabajador.Retenciones5taPrevias = retPrevias

				// 5. Calculamos
				montoFinal = calculadoras.CalcularRenta5ta(remMensual, ctxTrabajador)
			}

			if tipoUpper == "RETENCION" {
				boleta.TotalRetenciones += montoFinal
			}
			if tipoUpper == "APORTE" {
				boleta.TotalAportes += montoFinal
			}

			boleta.LineasConceptos = append(boleta.LineasConceptos, models.PlanillaConcepto{
				ConceptoTenantID: cp.TenantID, TipoConcepto: tipoUpper, Monto: montoFinal,
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
