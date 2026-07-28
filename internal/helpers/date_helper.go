package helpers

import (
	"time"
)

// CalcularMesesYAnosServicio calcula los años y meses transcurridos entre dos fechas
func CalcularMesesYAnosServicio(start, end time.Time) (int, int) {
	years, months, _ := CalcularTiempoServicioCompleto(start, end)
	return years, months
}

// CalcularTiempoServicioCompleto calcula los años, meses y días transcurridos entre dos fechas
func CalcularTiempoServicioCompleto(start, end time.Time) (int, int, int) {
	if start.After(end) {
		return 0, 0, 0
	}
	years := end.Year() - start.Year()
	months := int(end.Month() - start.Month())
	days := end.Day() - start.Day()

	if days < 0 {
		months--
		prevMonth := end.AddDate(0, -1, 0)
		daysInPrevMonth := time.Date(prevMonth.Year(), prevMonth.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
		days += daysInPrevMonth
	}
	if months < 0 {
		years--
		months += 12
	}
	if years < 0 {
		return 0, 0, 0
	}
	return years, months, days
}

// CalcularMesesYDiasTruncas calcula los meses y días truncos en el periodo actual
func CalcularMesesYDiasTruncas(start, end time.Time) (int, int) {
	if start.After(end) {
		return 0, 0
	}
	// Última fecha de aniversario antes o igual al fin
	anniversary := time.Date(end.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
	if anniversary.After(end) {
		anniversary = anniversary.AddDate(-1, 0, 0)
	}

	// Calcular meses y días entre el aniversario y el fin
	months := 0
	curr := anniversary
	for {
		next := curr.AddDate(0, 1, 0)
		if next.After(end) {
			break
		}
		months++
		curr = next
	}
	days := int(end.Sub(curr).Hours() / 24)
	return months, days
}

// CalcularMesesInterseccion calcula la cantidad de meses completos laborados dentro de un rango de fechas
func CalcularMesesInterseccion(start time.Time, end *time.Time, desde, hasta time.Time) int {
	s := start
	var e time.Time
	if end == nil {
		e = hasta
	} else {
		e = *end
	}

	if s.Before(desde) {
		s = desde
	}
	if e.After(hasta) {
		e = hasta
	}
	if s.After(e) {
		return 0
	}

	months := 0
	curr := time.Date(s.Year(), s.Month(), 1, 0, 0, 0, 0, time.UTC)
	for curr.Before(e) || curr.Equal(e) {
		mStart := curr
		mEnd := time.Date(curr.Year(), curr.Month()+1, 1, 0, 0, 0, 0, time.UTC).Add(-time.Second)

		// Si el rango de labores [s, e] cubre todo el mes de calendario, lo consideramos completo
		if (s.Before(mStart) || s.Equal(mStart)) && (e.After(mEnd) || e.Equal(mEnd)) {
			months++
		}
		curr = curr.AddDate(0, 1, 0)
	}
	return months
}

// ObtenerPorcentajeGradualCAS retorna la escala de gradualidad legal de la Ley N° 32563 / DS N° 142-2026-EF
func ObtenerPorcentajeGradualCAS(anio int) float64 {
	switch {
	case anio == 2026:
		return 0.10
	case anio == 2027:
		return 0.20
	case anio == 2028:
		return 0.30
	case anio == 2029:
		return 0.50
	case anio >= 2030:
		return 1.00
	default:
		return 0.00
	}
}

// CalcularMesesYDiasSemestreGratificacionCAS calcula los meses calendario completos y días laborados dentro del rango de un semestre
func CalcularMesesYDiasSemestreGratificacionCAS(start time.Time, end *time.Time, semDesde, semHasta time.Time) (int, int) {
	s := start
	var e time.Time
	if end == nil {
		e = semHasta
	} else {
		e = *end
	}

	if s.Before(semDesde) {
		s = semDesde
	}
	if e.After(semHasta) {
		e = semHasta
	}
	if s.After(e) {
		return 0, 0
	}

	months := 0
	days := 0

	curr := time.Date(semDesde.Year(), semDesde.Month(), 1, 0, 0, 0, 0, time.UTC)
	for curr.Before(semHasta) || curr.Equal(semHasta) {
		mStart := curr
		mEnd := time.Date(curr.Year(), curr.Month()+1, 1, 0, 0, 0, 0, time.UTC).Add(-time.Second)

		if (s.Before(mStart) || s.Equal(mStart)) && (e.After(mEnd) || e.Equal(mEnd)) {
			months++
		} else {
			// Calcular días en este mes no completo si hay intersección
			if !(s.After(mEnd) || e.Before(mStart)) {
				dStart := s
				if dStart.Before(mStart) {
					dStart = mStart
				}
				dEnd := e
				if dEnd.After(mEnd) {
					dEnd = mEnd
				}
				diasEnMes := int(dEnd.Sub(dStart).Hours()/24) + 1
				days += diasEnMes
			}
		}
		curr = curr.AddDate(0, 1, 0)
	}

	return months, days
}
