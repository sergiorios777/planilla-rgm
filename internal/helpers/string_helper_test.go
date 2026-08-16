package helpers

import (
	"testing"
)

func TestFormatearRol(t *testing.T) {
	pruebas := []struct {
		input    string
		esperado string
	}{
		{"tenant_admin", "Administrador"},
		{"tenant_operator", "Operador"},
		{"super_admin", "Súper Admin"},
		{"operador_planilla", "Operador planilla"},
		{"asistente_de_recursos_humanos", "Asistente de recursos humanos"},
		{"", "Administrador"},
	}

	for _, p := range pruebas {
		resultado := FormatearRol(p.input)
		if resultado != p.esperado {
			t.Errorf("FormatearRol(%q) = %q; se esperaba %q", p.input, resultado, p.esperado)
		}
	}
}
