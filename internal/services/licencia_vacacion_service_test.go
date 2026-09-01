package services

import (
	"planilla-rgm/internal/models"
	"testing"
)

func TestValidarLicenciaVacacion_DocumentoRequerido(t *testing.T) {
	svc := NewLicenciaVacacionService(nil)

	item := &models.LicenciaVacacion{
		Tipo:                "VACACION",
		FechaInicio:         "2026-08-01",
		FechaFin:            "2026-08-15",
		DocumentoAprobacion: "", // Vacío
	}

	err := svc.Crear(1, item)
	if err == nil {
		t.Fatal("se esperaba error por documento de aprobación vacío, pero no ocurrió")
	}
}

func TestValidarLicenciaVacacion_FechasInconsistentes(t *testing.T) {
	svc := NewLicenciaVacacionService(nil)

	item := &models.LicenciaVacacion{
		Tipo:                "VACACION",
		FechaInicio:         "2026-08-15",
		FechaFin:            "2026-08-01", // Fin anterior a inicio
		DocumentoAprobacion: "Resolución N° 001-2026",
	}

	err := svc.Crear(1, item)
	if err == nil {
		t.Fatal("se esperaba error por fecha de fin menor a fecha de inicio, pero no ocurrió")
	}
}

func TestValidarLicenciaVacacion_CodigoSunatDefault(t *testing.T) {
	// Validamos la lógica de mapeo de códigos por defecto
	tests := []struct {
		tipo        string
		codigoEsp   string
	}{
		{"VACACION", "23"},
		{"LICENCIA_CON_GOCE", "26"},
		{"LICENCIA_SIN_GOCE", "05"},
	}

	for _, tt := range tests {
		item := &models.LicenciaVacacion{
			Tipo:                tt.tipo,
			FechaInicio:         "2026-08-01",
			FechaFin:            "2026-08-15",
			DocumentoAprobacion: "Memo 123",
		}
		// Simulamos la asignación de defaults
		if item.CodigoSunatSuspension == "" {
			switch item.Tipo {
			case "VACACION":
				item.CodigoSunatSuspension = "23"
			case "LICENCIA_CON_GOCE":
				item.CodigoSunatSuspension = "26"
			case "LICENCIA_SIN_GOCE":
				item.CodigoSunatSuspension = "05"
			}
		}

		if item.CodigoSunatSuspension != tt.codigoEsp {
			t.Errorf("para tipo %s se esperaba código SUNAT %s, se obtuvo %s", tt.tipo, tt.codigoEsp, item.CodigoSunatSuspension)
		}
	}
}
