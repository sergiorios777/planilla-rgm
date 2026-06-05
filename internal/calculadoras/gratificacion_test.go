package calculadoras

import (
	"testing"
	"time"
)

func TestCalcularGratificacionDL728(t *testing.T) {
	// Caso completo: sueldo computable 2102.50, 6 meses worked
	base, bono := CalcularGratificacionDL728(2102.50, 6)
	if base != 2102.50 {
		t.Errorf("got base = %v; want 2102.50", base)
	}
	if bono != 189.23 { // 2102.50 * 0.09 = 189.225 -> round to 189.23
		t.Errorf("got bono = %v; want 189.23", bono)
	}

	// Caso parcial: sueldo computable 2000.00, 3 meses worked
	base2, bono2 := CalcularGratificacionDL728(2000.00, 3)
	if base2 != 1000.00 {
		t.Errorf("got base = %v; want 1000.00", base2)
	}
	if bono2 != 90.00 {
		t.Errorf("got bono = %v; want 90.00", bono2)
	}
}

func TestCalcularMesesSemestreGratificacion(t *testing.T) {
	semDesde := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	semHasta := time.Date(2026, time.June, 30, 23, 59, 59, 0, time.UTC)

	t.Run("Ingreso antes del semestre", func(t *testing.T) {
		ingreso := time.Date(2025, time.December, 15, 0, 0, 0, 0, time.UTC)
		meses := CalcularMesesSemestreGratificacion(ingreso, nil, semDesde, semHasta)
		if meses != 6 {
			t.Errorf("got %d; want 6", meses)
		}
	})

	t.Run("Ingreso a mitad de mes", func(t *testing.T) {
		ingreso := time.Date(2026, time.February, 10, 0, 0, 0, 0, time.UTC)
		meses := CalcularMesesSemestreGratificacion(ingreso, nil, semDesde, semHasta)
		if meses != 4 { // Marzo, Abril, Mayo, Junio
			t.Errorf("got %d; want 4", meses)
		}
	})

	t.Run("Con fecha de cese a mitad de semestre", func(t *testing.T) {
		ingreso := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
		cese := time.Date(2026, time.May, 15, 0, 0, 0, 0, time.UTC)
		meses := CalcularMesesSemestreGratificacion(ingreso, &cese, semDesde, semHasta)
		if meses != 4 { // Ene, Feb, Mar, Abr (Mayo cesó el 15, no es completo)
			t.Errorf("got %d; want 4", meses)
		}
	})
}
