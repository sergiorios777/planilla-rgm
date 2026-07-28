package calculadoras

import (
	"math"
	"planilla-rgm/internal/helpers"
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

// CalcularGratificacionDL1057 calcula la gratificación ordinaria o trunca para el régimen CAS (Ley 32563 / DS 142-2026-EF).
// Computa sextos por meses completos y treintavos de sexto por días laborados.
// Excepción legal: Para Julio de 2026 no se cuentan días sueltos, solo meses completos.
// Retorna la gratificación calculada y el aporte patronal a EsSalud (9%).
func CalcularGratificacionDL1057(remuneracionMensual float64, mesPago int, anioPago int, mesesLaborados int, diasLaborados int) (montoGrati float64, aporteEsSalud float64) {
	if mesesLaborados <= 0 && diasLaborados <= 0 {
		return 0.0, 0.0
	}

	// Regla de Excepción: Para Julio de 2026 no se cuentan días sueltos
	diasAComputar := diasLaborados
	if mesPago == 7 && anioPago == 2026 {
		diasAComputar = 0
	}

	porcentajeGradual := helpers.ObtenerPorcentajeGradualCAS(anioPago)
	baseCalculo := remuneracionMensual * porcentajeGradual

	montoMeses := (baseCalculo / 6.0) * float64(mesesLaborados)
	montoDias := (baseCalculo / 6.0 / 30.0) * float64(diasAComputar)

	montoGrati = round(montoMeses + montoDias)
	aporteEsSalud = round(montoGrati * 0.09)

	return montoGrati, aporteEsSalud
}
