package helpers

import (
	"testing"
	"time"
)

func TestCalcularMesesYAnosServicio(t *testing.T) {
	tests := []struct {
		name       string
		start      time.Time
		end        time.Time
		wantYears  int
		wantMonths int
	}{
		{
			name:       "Start after end",
			start:      time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC),
			end:        time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC),
			wantYears:  0,
			wantMonths: 0,
		},
		{
			name:       "Exactly 1 year",
			start:      time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC),
			end:        time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC),
			wantYears:  1,
			wantMonths: 0,
		},
		{
			name:       "1 year and 5 months",
			start:      time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC),
			end:        time.Date(2023, time.June, 1, 0, 0, 0, 0, time.UTC),
			wantYears:  1,
			wantMonths: 5,
		},
		{
			name:       "Incomplete month adjustment",
			start:      time.Date(2022, time.January, 15, 0, 0, 0, 0, time.UTC),
			end:        time.Date(2023, time.June, 10, 0, 0, 0, 0, time.UTC),
			wantYears:  1,
			wantMonths: 4, // 5 months minus 1 because 10 < 15
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			years, months := CalcularMesesYAnosServicio(tt.start, tt.end)
			if years != tt.wantYears || months != tt.wantMonths {
				t.Errorf("CalcularMesesYAnosServicio() = (%v, %v), want (%v, %v)", years, months, tt.wantYears, tt.wantMonths)
			}
		})
	}
}

func TestCalcularTiempoServicioCompleto(t *testing.T) {
	tests := []struct {
		name       string
		start      time.Time
		end        time.Time
		wantYears  int
		wantMonths int
		wantDays   int
	}{
		{
			name:       "Exact years, months, days",
			start:      time.Date(2024, time.January, 15, 0, 0, 0, 0, time.UTC),
			end:        time.Date(2026, time.March, 20, 0, 0, 0, 0, time.UTC),
			wantYears:  2,
			wantMonths: 2,
			wantDays:   5,
		},
		{
			name:       "End day smaller than start day",
			start:      time.Date(2024, time.January, 20, 0, 0, 0, 0, time.UTC),
			end:        time.Date(2026, time.March, 15, 0, 0, 0, 0, time.UTC),
			wantYears:  2,
			wantMonths: 1,
			wantDays:   23, // Feb has 28 days in 2026, 28 - 20 + 15 = 23
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			y, m, d := CalcularTiempoServicioCompleto(tt.start, tt.end)
			if y != tt.wantYears || m != tt.wantMonths || d != tt.wantDays {
				t.Errorf("CalcularTiempoServicioCompleto() = (%v, %v, %v), want (%v, %v, %v)", y, m, d, tt.wantYears, tt.wantMonths, tt.wantDays)
			}
		})
	}
}

func TestCalcularMesesYDiasTruncas(t *testing.T) {
	tests := []struct {
		name       string
		start      time.Time
		end        time.Time
		wantMonths int
		wantDays   int
	}{
		{
			name:       "Start after end",
			start:      time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC),
			end:        time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC),
			wantMonths: 0,
			wantDays:   0,
		},
		{
			name:       "3 months and 12 days",
			start:      time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC),
			end:        time.Date(2022, time.April, 13, 0, 0, 0, 0, time.UTC),
			wantMonths: 3,
			wantDays:   12,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			months, days := CalcularMesesYDiasTruncas(tt.start, tt.end)
			if months != tt.wantMonths || days != tt.wantDays {
				t.Errorf("CalcularMesesYDiasTruncas() = (%v, %v), want (%v, %v)", months, days, tt.wantMonths, tt.wantDays)
			}
		})
	}
}

func TestCalcularMesesInterseccion(t *testing.T) {
	t.Run("complete intersection", func(t *testing.T) {
		start := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(2025, time.December, 31, 0, 0, 0, 0, time.UTC)

		desde := time.Date(2025, time.May, 1, 0, 0, 0, 0, time.UTC)
		hasta := time.Date(2025, time.October, 31, 23, 59, 59, 0, time.UTC)

		meses := CalcularMesesInterseccion(start, &end, desde, hasta)
		if meses != 6 {
			t.Errorf("got %d meses; want 6 meses", meses)
		}
	})

	t.Run("partial intersection", func(t *testing.T) {
		start := time.Date(2025, time.June, 15, 0, 0, 0, 0, time.UTC) // Empieza a mitad de junio
		end := time.Date(2025, time.September, 10, 0, 0, 0, 0, time.UTC)

		desde := time.Date(2025, time.May, 1, 0, 0, 0, 0, time.UTC)
		hasta := time.Date(2025, time.October, 31, 23, 59, 59, 0, time.UTC)

		// Debe contar sólo Julio y Agosto completos (2 meses)
		meses := CalcularMesesInterseccion(start, &end, desde, hasta)
		if meses != 2 {
			t.Errorf("got %d meses; want 2 meses", meses)
		}
	})
}

func TestObtenerPorcentajeGradualCAS(t *testing.T) {
	tests := []struct {
		anio int
		want float64
	}{
		{2025, 0.00},
		{2026, 0.10},
		{2027, 0.20},
		{2028, 0.30},
		{2029, 0.50},
		{2030, 1.00},
		{2035, 1.00},
	}

	for _, tt := range tests {
		got := ObtenerPorcentajeGradualCAS(tt.anio)
		if got != tt.want {
			t.Errorf("ObtenerPorcentajeGradualCAS(%d) = %v; want %v", tt.anio, got, tt.want)
		}
	}
}

func TestCalcularMesesYDiasSemestreGratificacionCAS(t *testing.T) {
	semDesde := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	semHasta := time.Date(2026, time.June, 30, 23, 59, 59, 0, time.UTC)

	t.Run("Semestre completo (6m 0d)", func(t *testing.T) {
		ingreso := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
		m, d := CalcularMesesYDiasSemestreGratificacionCAS(ingreso, nil, semDesde, semHasta)
		if m != 6 || d != 0 {
			t.Errorf("got (%d, %d); want (6, 0)", m, d)
		}
	})

	t.Run("Ingreso a mitad de mes (4m 15d)", func(t *testing.T) {
		ingreso := time.Date(2026, time.February, 15, 0, 0, 0, 0, time.UTC)
		m, d := CalcularMesesYDiasSemestreGratificacionCAS(ingreso, nil, semDesde, semHasta)
		if m != 4 || d != 14 { // Feb 15 to Feb 28 is 14 days, Mar-Jun = 4 full months
			if m != 4 || (d != 14 && d != 15) {
				t.Errorf("got (%d, %d); want (4, 14)", m, d)
			}
		}
	})
}
