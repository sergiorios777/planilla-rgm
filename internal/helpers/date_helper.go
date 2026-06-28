package helpers

import (
	"time"
)

// CalcularMesesYAnosServicio calcula los años y meses transcurridos entre dos fechas
func CalcularMesesYAnosServicio(start, end time.Time) (int, int) {
	if start.After(end) {
		return 0, 0
	}
	years := end.Year() - start.Year()
	months := int(end.Month() - start.Month())
	days := end.Day() - start.Day()

	if days < 0 {
		months--
	}
	totalMonths := years*12 + months
	if totalMonths < 0 {
		totalMonths = 0
	}
	return totalMonths / 12, totalMonths % 12
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
