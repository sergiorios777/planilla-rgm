package repository_test

import (
	"planilla-rgm/internal/repository"
	"testing"
)

func TestResolverCodigoPadre(t *testing.T) {
	tests := []struct {
		codigo string
		want   string
	}{
		// Series Estándar (0100 - 0900)
		{"0121", "0100"},
		{"0100", ""},
		{"0601", "0600"},
		{"0701", "0700"},
		{"0804", "0800"},

		// Excepción Serie 1000 (Sector Público)
		{"1000", ""},
		{"1001", "1000"},
		{"1099", "1000"},
		{"1100", "1000"},
		{"1101", "1000"},
		{"1200", "1000"},

		// Excepción Serie 2000 (Régimen Laboral Público)
		{"2000", ""},
		{"2001", "2000"},
		{"2099", "2000"},
		{"2100", "2000"},
		{"2101", "2000"},
		{"2200", "2000"},
		{"2999", "2000"},

		// Conceptos Internos derivados de SUNAT o Auxiliares
		{"0312", "0300"}, // Bonificación extraordinaria (origen = interno)
		{"0406", "0400"}, // Gratificaciones DL 728 (origen = interno)
		{"2002", "2000"}, // Asignación Familiar DL 728 (origen = interno)
		{"S101", "S100"}, // Retenciones Auxiliares
		{"S202", "S200"}, // Ingresos Auxiliares (Gratificación CAS)

		// Códigos inválidos o no estándar
		{"ABC", ""},
		{"12345", ""},
	}

	for _, tt := range tests {
		got := repository.ResolverCodigoPadre(tt.codigo)
		if got != tt.want {
			t.Errorf("ResolverCodigoPadre(%q) = %q; se esperaba %q", tt.codigo, got, tt.want)
		}
	}
}
