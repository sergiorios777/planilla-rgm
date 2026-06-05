package calculadoras

import (
	"math"
	"time"
)

// CalcularGratificacionDL728 calcula la gratificación base y bonificación extraordinaria (9%)
func CalcularGratificacionDL728(remComputable float64, mesesTrabajados int) (float64, float64) {
	if mesesTrabajados <= 0 {
		return 0.0, 0.0
	}
	base := (remComputable / 6.0) * float64(mesesTrabajados)
	baseRounded := math.Round(base*100) / 100

	bono := baseRounded * 0.09
	bonoRounded := math.Round(bono*100) / 100

	return baseRounded, bonoRounded
}

// CalcularMesesSemestreGratificacion cuenta los meses calendario completos trabajados dentro del semestre
func CalcularMesesSemestreGratificacion(fechaInicio time.Time, fechaFin *time.Time, semDesde, semHasta time.Time) int {
	mesesCompletos := 0

	// Recorrer cada mes en el rango del semestre
	curr := time.Date(semDesde.Year(), semDesde.Month(), 1, 0, 0, 0, 0, time.UTC)
	for curr.Before(semHasta) || curr.Equal(semHasta) {
		mStart := curr
		// Último día de este mes
		mEnd := time.Date(curr.Year(), curr.Month()+1, 1, 0, 0, 0, 0, time.UTC).Add(-time.Second)

		// Para que el mes sea completo, el inicio del contrato debe ser en o antes de mStart,
		// y el fin de contrato debe ser nulo o en o después de mEnd.
		started := fechaInicio.Before(mStart) || fechaInicio.Equal(mStart)
		ended := true
		if fechaFin != nil {
			ended = fechaFin.After(mEnd) || fechaFin.Equal(mEnd)
		}

		if started && ended {
			mesesCompletos++
		}

		curr = curr.AddDate(0, 1, 0)
	}

	return mesesCompletos
}
