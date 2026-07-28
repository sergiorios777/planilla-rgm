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

func TestCalcularGratificacionDL1057(t *testing.T) {
	t.Run("Julio 2026 Semestre completo (6m 0d, 10% de S/3000 = S/300 grati, S/27 EsSalud)", func(t *testing.T) {
		grati, essalud := CalcularGratificacionDL1057(3000.00, 7, 2026, 6, 0)
		if grati != 300.00 || essalud != 27.00 {
			t.Errorf("got (%v, %v); want (300.00, 27.00)", grati, essalud)
		}
	})

	t.Run("Julio 2026 Excepción días sueltos (4m 15d -> ignora 15d, 4/6 de S/300 = S/200)", func(t *testing.T) {
		grati, essalud := CalcularGratificacionDL1057(3000.00, 7, 2026, 4, 15)
		if grati != 200.00 || essalud != 18.00 {
			t.Errorf("got (%v, %v); want (200.00, 18.00)", grati, essalud)
		}
	})

	t.Run("Julio 2027 Computa meses y días (4m 15d, 20% de S/3000 = S/600 base. 4.5/6 * 600 = S/450)", func(t *testing.T) {
		grati, essalud := CalcularGratificacionDL1057(3000.00, 7, 2027, 4, 15)
		if grati != 450.00 || essalud != 40.50 {
			t.Errorf("got (%v, %v); want (450.00, 40.50)", grati, essalud)
		}
	})

	t.Run("Diciembre 2030 (6m 0d, 100% de S/4000 = S/4000 grati, S/360 EsSalud)", func(t *testing.T) {
		grati, essalud := CalcularGratificacionDL1057(4000.00, 12, 2030, 6, 0)
		if grati != 4000.00 || essalud != 360.00 {
			t.Errorf("got (%v, %v); want (4000.00, 360.00)", grati, essalud)
		}
	})
}
