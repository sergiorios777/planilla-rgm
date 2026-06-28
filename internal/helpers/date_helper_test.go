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
