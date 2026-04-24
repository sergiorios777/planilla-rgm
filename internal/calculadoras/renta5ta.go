package calculadoras

import (
	"log"
	"math"
	"planilla-rgm/internal/models"
)

// CalcularRenta5ta ejecuta el algoritmo oficial de SUNAT en 5 pasos
func CalcularRenta5ta(remuneracionMensual float64, ctx models.ContextoCalculo) float64 {
	uit, existe := ctx.ParametrosGlobales["UIT"]
	if !existe {
		uit = 5500.00 // Asumiendo UIT 2026 referencial
	}

	// ==============================================================
	// PASO 1: Proyección de la Remuneración Bruta Anual
	// ==============================================================
	// SUNAT: (Remuneración Ordinaria del Mes * Meses Faltantes) + Ingresos Previos del Año

	mesesFaltantes := 13 - ctx.MesActual // Si es Abril(4), faltan 9 meses (Abr, May, Jun, Jul, Ago, Sep, Oct, Nov, Dic).

	remuneracionProyectada := remuneracionMensual * float64(mesesFaltantes)

	// 💡 LA CORRECCIÓN MAESTRA: Sumamos los ingresos que ya ocurrieron este año
	remuneracionBrutaAnual := remuneracionProyectada + ctx.RemuneracionNoMensual + ctx.IngresosPrevios

	// ==============================================================
	// PASO 2: Deducción de 7 UIT
	// ==============================================================
	remuneracionNetaAnual := remuneracionBrutaAnual - (7 * uit)
	if remuneracionNetaAnual <= 0 {
		remuneracionNetaAnual = 0 // No sujeto a retención si no supera 7 UIT
	}

	// ==============================================================
	// PASO 3: Cálculo del Impuesto Anual Proyectado (Tramos)
	// ==============================================================
	impuestoAnualProyectado := calcularImpuestoTramos(remuneracionNetaAnual, uit)

	// ==============================================================
	// PASO 4: Monto de la Retención (Fraccionamiento por meses)
	// ==============================================================
	retencionMes := 0.00
	impuestoRestante := impuestoAnualProyectado - ctx.Retenciones5taPrevias
	log.Println("impuestoAnualProyectado: ", impuestoAnualProyectado)
	log.Println("ctx.Retenciones5taPrevias: ", ctx.Retenciones5taPrevias)
	log.Println("impuestoRestante: ", impuestoRestante)

	if impuestoRestante > 0 {
		switch {
		case ctx.MesActual >= 1 && ctx.MesActual <= 3: // Ene - Mar
			retencionMes = impuestoAnualProyectado / 12
		case ctx.MesActual == 4: // Abr
			retencionMes = impuestoRestante / 9
		case ctx.MesActual >= 5 && ctx.MesActual <= 7: // May - Jul
			retencionMes = impuestoRestante / 8
		case ctx.MesActual == 8: // Ago
			retencionMes = impuestoRestante / 5
		case ctx.MesActual >= 9 && ctx.MesActual <= 11: // Sep - Nov
			retencionMes = impuestoRestante / 4
		case ctx.MesActual == 12: // Dic
			retencionMes = impuestoRestante // Se retiene todo el saldo
		}
	}

	// ==============================================================
	// PASO 5: Cálculo adicional por Ingresos Extraordinarios del Mes
	// ==============================================================
	if ctx.IngresoExtraordinarioDelMes > 0 {
		// A y B: Sumamos el extra al proyectado inicial y deducimos 7 UIT
		brutaExtraordinaria := remuneracionBrutaAnual + ctx.IngresoExtraordinarioDelMes
		netaExtraordinaria := brutaExtraordinaria - (7 * uit)
		if netaExtraordinaria < 0 {
			netaExtraordinaria = 0
		}

		// C: Recalculamos el paso 3
		impuestoExtraordinario := calcularImpuestoTramos(netaExtraordinaria, uit)

		// D: Hallamos la diferencia
		retencionAdicional := impuestoExtraordinario - impuestoAnualProyectado
		if retencionAdicional < 0 {
			retencionAdicional = 0
		}

		// E: Sumamos a la retención del mes regular
		retencionMes += retencionAdicional
	}

	return retencionMes
}

// calcularImpuestoTramos aplica las tasas progresivas del Artículo 53 de la LIR
func calcularImpuestoTramos(neta float64, uit float64) float64 {
	impuesto := 0.00
	saldo := neta

	// 1er Tramo: Hasta 5 UIT (8%)
	if saldo > 0 {
		baseTramo := math.Min(saldo, 5*uit)
		impuesto += baseTramo * 0.08
		saldo -= baseTramo
	}
	// 2do Tramo: Más de 5 hasta 20 UIT (14%) -> Capacidad del tramo = 15 UIT
	if saldo > 0 {
		baseTramo := math.Min(saldo, 15*uit)
		impuesto += baseTramo * 0.14
		saldo -= baseTramo
	}
	// 3er Tramo: Más de 20 hasta 35 UIT (17%) -> Capacidad del tramo = 15 UIT
	if saldo > 0 {
		baseTramo := math.Min(saldo, 15*uit)
		impuesto += baseTramo * 0.17
		saldo -= baseTramo
	}
	// 4to Tramo: Más de 35 hasta 45 UIT (20%) -> Capacidad del tramo = 10 UIT
	if saldo > 0 {
		baseTramo := math.Min(saldo, 10*uit)
		impuesto += baseTramo * 0.20
		saldo -= baseTramo
	}
	// 5to Tramo: Más de 45 UIT (30%)
	if saldo > 0 {
		impuesto += saldo * 0.30
	}

	return impuesto
}
