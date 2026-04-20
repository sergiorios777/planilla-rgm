package calculadoras

import (
	"planilla-rgm/internal/models"
)

// CalcularRenta4ta retiene el 8% basándose en los topes legales de SUNAT.
func CalcularRenta4ta(baseImponible float64, ctx models.ContextoCalculo) float64 {

	// 1. Validar Suspensión Activa
	if ctx.TieneSuspension {
		return 0.00
	}

	// 2. Extraer parámetros (Para 2026, el tope mensual de proyección es S/ 4,010)
	topeMensualGeneral, existeTopeGen := ctx.ParametrosGlobales["TOPE_4TA_MENSUAL_GENERAL"]
	if !existeTopeGen {
		// Salvavidas asumiendo UIT 2026 de 5500
		topeMensualGeneral = 4010.00
	}

	tasa4ta, existeTasa := ctx.ParametrosGlobales["TASA_RENTA_4TA"]
	if !existeTasa {
		tasa4ta = 0.08 // 8%
	}

	// 3. Regla del Régimen
	if ctx.RegimenCodigo == "1057" { // Régimen CAS
		// Para CAS, si el ingreso del mes supera los 4,010, se le retiene el 8% de TODO el monto.
		if baseImponible > topeMensualGeneral {
			return baseImponible * tasa4ta
		}
		return 0.00
	}

	// ====================================================================
	// 💡 Lógica Futura: Terceros / Locadores de Servicio (Recibo por Honorarios)
	// Si más adelante metes locadores a tu sistema, la regla de ellos sí es S/ 1,500 por recibo.
	// ====================================================================
	// topeRecibo, _ := ctx.ParametrosGlobales["TOPE_4TA_RECIBO"]
	// if baseImponible > topeRecibo { return baseImponible * tasa4ta }

	// Por defecto, si no es CAS ni tercero explícito, y supera el tope general, retener.
	if baseImponible > topeMensualGeneral {
		return baseImponible * tasa4ta
	}

	return 0.00
}
