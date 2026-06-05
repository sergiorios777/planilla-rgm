package calculadoras

import (
	"math"
)

// CalcularAsignacionFamiliar calcula la asignación familiar (10% de la RMV)
func CalcularAsignacionFamiliar(rmv float64) float64 {
	monto := 0.10 * rmv
	return math.Round(monto*100) / 100
}
