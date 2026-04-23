package calculadoras

import (
	"math"
	"planilla-rgm/internal/models"
)

// CalcularONP calcula el descuento del Sistema Nacional de Pensiones
func CalcularONP(baseImponible float64, ctx models.ContextoCalculo) float64 {
	tasaONP, existe := ctx.ParametrosGlobales["TASA_ONP"]
	if !existe {
		tasaONP = 0.13 // 13% por defecto legal
	}
	return math.Round(baseImponible*tasaONP*100) / 100
}

// CalcularAFPAporte calcula el Aporte Obligatorio (10% por defecto)
func CalcularAFPAporte(baseImponible float64, ctx models.ContextoCalculo) float64 {
	return math.Round(baseImponible*ctx.TasaAfpAporte*100) / 100
}

// CalcularAFPPrima calcula el seguro de invalidez respetando el Tope de la SBS
func CalcularAFPPrima(baseImponible float64, ctx models.ContextoCalculo) float64 {
	// EL TOPE LEGAL (Remuneración Máxima Asegurable) sigue siendo global
	topePrima, existeTope := ctx.ParametrosGlobales["TOPE_PRIMA_SEGURO"]
	if !existeTope {
		topePrima = 11824.90 // Tope de salvavidas
	}

	baseCalculo := baseImponible
	if baseCalculo > topePrima {
		baseCalculo = topePrima
	}

	return math.Round(baseCalculo*ctx.TasaAfpPrima*100) / 100
}

// CalcularAFPComision aplica la tasa de comisión (Flujo o Mixta) ya decidida
func CalcularAFPComision(baseImponible float64, ctx models.ContextoCalculo) float64 {
	return math.Round(baseImponible*ctx.TasaAfpComision*100) / 100
}
