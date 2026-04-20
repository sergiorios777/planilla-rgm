package calculadoras

import (
	"planilla-rgm/internal/models"
)

// CalcularEsSalud recibe el total de la base imponible y el maletín de contexto
func CalcularEsSalud(baseImponible float64, ctx models.ContextoCalculo) float64 {

	// Extraemos los parámetros de la BD. (Usamos valores por defecto como mecanismo de seguridad por si alguien olvida registrarlos)
	tasaEsSalud, existe := ctx.ParametrosGlobales["TASA_ESSALUD"]
	if !existe {
		tasaEsSalud = 0.09
	}

	rmv, existeRMV := ctx.ParametrosGlobales["RMV"]
	if !existeRMV {
		rmv = 1130.00 // RMV vigente al 2026
	}

	// Regla Universal 4: El aporte mínimo siempre será el 9% de la RMV
	aporteMinimo := rmv * tasaEsSalud

	// ==========================================
	// LÓGICA RÉGIMEN CAS
	// ==========================================
	if ctx.RegimenCodigo == "1057" {
		uit, existeUIT := ctx.ParametrosGlobales["UIT"]
		if !existeUIT {
			uit = 5500.00 // UIT vigente 2026
		}

		// Paso 2: Calcular la base imponible máxima (45% de la UIT según indicación)
		baseMaxima := uit * 0.45

		// Paso 3: Verificar si la base del puesto supera la máxima
		baseCalculo := baseImponible
		if baseCalculo > baseMaxima {
			baseCalculo = baseMaxima
		}

		// Calculamos el 9% de la base elegida
		aporte := baseCalculo * tasaEsSalud

		// Paso 4: Verificamos contra el mínimo vital
		if aporte < aporteMinimo {
			return aporteMinimo
		}

		return aporte
	}

	// ==========================================
	// LÓGICA OTROS REGÍMENES (276, 728, SERVIR)
	// ==========================================
	aporteGeneral := baseImponible * tasaEsSalud

	if aporteGeneral < aporteMinimo {
		return aporteMinimo
	}

	return aporteGeneral
}
