package calculadoras

import "planilla-rgm/internal/helpers"

// CalcularCtsSemestralDL728 calcula la CTS para el régimen laboral DL 728 (actividad privada).
// Retorna el monto bruto de CTS, el descuento por inasistencias y el monto neto a pagar.
func CalcularCtsSemestralDL728(remComputable float64, mesesLaborados int, diasFaltas int) (montoBruto float64, descuentoFaltas float64, montoNeto float64) {
	if mesesLaborados < 1 {
		return 0.0, 0.0, 0.0
	}

	// CTS Semestral Base = (Remuneración computable / 12) * meses laborados
	baseMensual := remComputable / 12.0
	montoBruto = baseMensual * float64(mesesLaborados)

	// Deducción por faltas injustificadas: 1/30 del dozavo (baseMensual) por cada día de falta
	if diasFaltas > 0 {
		descuentoFaltas = (baseMensual / 30.0) * float64(diasFaltas)
		montoNeto = montoBruto - descuentoFaltas
	} else {
		montoNeto = montoBruto
	}

	// Salvaguarda para evitar montos negativos
	if montoNeto < 0 {
		montoNeto = 0
	}

	return montoBruto, descuentoFaltas, montoNeto
}

// CalcularCtsLey30057 calcula la CTS para el régimen de Servicio Civil.
// Equivale al 100% del promedio mensual de valorizaciones por cada año trabajado (fracciones por dozavos).
func CalcularCtsLey30057(promedioValorizaciones float64, mesesTotales int) float64 {
	if mesesTotales < 1 {
		return 0.0
	}

	// Equivale a: (Promedio mensual / 12) * meses totales
	return (promedioValorizaciones / 12.0) * float64(mesesTotales)
}

// CalcularCtsDL276 calcula la CTS al cese para el régimen del Decreto Legislativo 276.
// Equivale al 100% de la remuneración total del cese por cada año completo de servicio o fracción mayor a 6 meses.
func CalcularCtsDL276(remuneracionTotal float64, mesesTotales int) float64 {
	anos := mesesTotales / 12
	mesesRestantes := mesesTotales % 12

	// Fracción de meses mayor a 6 meses computa como 1 año completo adicional
	if mesesRestantes > 6 {
		anos++
	}

	return remuneracionTotal * float64(anos)
}

// CalcularCtsDL1057 calcula la CTS al cese para el régimen CAS (Ley 32563 / DS 142-2026-EF).
// Se calcula en función del porcentaje de gradualidad del año de cese y los años completos de servicio,
// computándose como 1 año completo adicional si la fracción restante de meses es mayor a 6 meses.
func CalcularCtsDL1057(remuneracionMensual float64, anioCese int, mesesTotales int) float64 {
	if mesesTotales < 1 {
		return 0.0
	}

	porcentajeGradual := helpers.ObtenerPorcentajeGradualCAS(anioCese)
	baseCTS := remuneracionMensual * porcentajeGradual

	anos := mesesTotales / 12
	mesesRestantes := mesesTotales % 12

	// Fracción de meses mayor a 6 meses computa como 1 año completo adicional
	if mesesRestantes > 6 {
		anos++
	}

	if anos == 0 {
		return 0.0
	}

	return round(baseCTS * float64(anos))
}
