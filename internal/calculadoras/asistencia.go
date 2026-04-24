package calculadoras

import (
	"math"
)

// CalcularDescuentoFaltas aplica la regla de 1/30 + el proporcional dominical (1/7)
func CalcularDescuentoFaltas(remuneracionComputable float64, diasFalta float64) float64 {
	if diasFalta <= 0 {
		return 0.00
	}

	valorDia := remuneracionComputable / 30.0
	descuentoDirecto := valorDia * diasFalta
	descuentoProporcional := descuentoDirecto / 7.0

	totalDescuento := descuentoDirecto + descuentoProporcional

	// Redondeo a 2 decimales para la contabilidad
	return math.Round(totalDescuento*100) / 100
}

// CalcularDescuentoTardanzas aplica la regla de minutos en base a 8 horas (480 min)
func CalcularDescuentoTardanzas(remuneracionComputable float64, minutosTardanza float64) float64 {
	if minutosTardanza <= 0 {
		return 0.00
	}

	valorDia := remuneracionComputable / 30.0
	valorMinuto := valorDia / 480.0

	totalDescuento := valorMinuto * minutosTardanza

	return math.Round(totalDescuento*100) / 100
}
