package helpers

import (
	"testing"
)

func TestNumeroALetras(t *testing.T) {
	casos := []struct {
		monto    float64
		esperado string
	}{
		{0.00, "SON: CERO Y 00/100 SOLES"},
		{1.00, "SON: UN Y 00/100 SOLES"},
		{1585.23, "SON: UN MIL QUINIENTOS OCHENTA Y CINCO Y 23/100 SOLES"},
		{448.68, "SON: CUATROCIENTOS CUARENTA Y OCHO Y 68/100 SOLES"},
		{100.00, "SON: CIEN Y 00/100 SOLES"},
		{2000.50, "SON: DOS MIL Y 50/100 SOLES"},
		{1000000.00, "SON: UN MILLON Y 00/100 SOLES"},
	}

	for _, c := range casos {
		resultado := NumeroALetras(c.monto)
		if resultado != c.esperado {
			t.Errorf("Para %.2f, se esperaba %q pero se obtuvo %q", c.monto, c.esperado, resultado)
		}
	}
}
