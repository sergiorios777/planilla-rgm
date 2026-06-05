package services

import (
	"planilla-rgm/internal/models"
	"testing"
	"time"
)

func TestPlanillaService_CalcularBoletaContrato_Gratificacion(t *testing.T) {
	s := &PlanillaService{Repo: nil} // No repo needed since we only test calcularBoletaContrato

	// Contract details
	contrato := models.ContratoPlanilla{
		ID:                        101,
		PuestoID:                  201,
		Regimen:                   "728",
		RegimenPensionario:        "ONP",
		FechaInicio:               time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC),
		SueldoBasicoHistorico:     2000.00,
		TrabajadorNombreCompleto:  "Perez Gomez, Juan",
		TrabajadorNumeroDocumento: "99991111",
	}

	// Conceptos Plaza (Gratification in July + Asignación Familiar + Sueldo)
	conceptosPlaza := []models.ConceptoPlanilla{
		{
			MaestroID:     1,
			CodigoInterno: "2002",
			CodigoSunat:   "2002",
			Tipo:          "INGRESO",
			Monto:         2000.00,
			Frecuencia:    "1,2,3,4,5,6,7,8,9,10,11,12",
		},
		{
			MaestroID:     2,
			CodigoInterno: "ASIG_FAM_DL728",
			CodigoSunat:   "ASIG_FAM_DL728",
			Tipo:          "INGRESO",
			Monto:         102.50,
			Frecuencia:    "1,2,3,4,5,6,7,8,9,10,11,12",
		},
		{
			MaestroID:     3,
			CodigoInterno: "GRATI_JUL_DL_728",
			CodigoSunat:   "0406",
			Tipo:          "INGRESO",
			Monto:         0.00, // Should be calculated dynamically
			Frecuencia:    "7",
		},
		{
			MaestroID:     4,
			CodigoInterno: "BON_EXTR_JUL_DL_728",
			CodigoSunat:   "0312",
			Tipo:          "INGRESO",
			Monto:         0.00, // Should be calculated dynamically
			Frecuencia:    "7",
		},
	}

	job := models.JobPlanilla{
		Contrato:           contrato,
		ConceptosPlaza:     conceptosPlaza,
		MesActual:          7, // July
		Anio:               2026,
		ParametrosGlobales: map[string]float64{},
		MapaCodigos:        map[string]int{"0705": 99}, // Faltas mapping
	}

	boleta, err := s.calcularBoletaContrato(job)
	if err != nil {
		t.Fatalf("Error calculando boleta: %v", err)
	}

	// Remuneración Computable = 2000 + 102.50 = 2102.50
	// Meses laborados = 6 (Jan to Jun)
	// Gratificación Base = 2102.50
	// Bonificación Extraordinaria = 2102.50 * 0.09 = 189.23

	var gratiMonto, bonExtMonto float64
	for _, lc := range boleta.LineasConceptos {
		if lc.CodigoSunat == "0406" {
			gratiMonto = lc.Monto
		}
		if lc.CodigoSunat == "0312" {
			bonExtMonto = lc.Monto
		}
	}

	if gratiMonto != 2102.50 {
		t.Errorf("got gratificación = %v; want 2102.50", gratiMonto)
	}
	if bonExtMonto != 189.23 {
		t.Errorf("got bonificación extraordinaria = %v; want 189.23", bonExtMonto)
	}
}

func TestPlanillaService_CalcularBoletaContrato_AsignacionFamiliar(t *testing.T) {
	s := &PlanillaService{Repo: nil}

	contrato := models.ContratoPlanilla{
		ID:                        102,
		PuestoID:                  202,
		Regimen:                   "728",
		RegimenPensionario:        "ONP",
		FechaInicio:               time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC),
		SueldoBasicoHistorico:     2000.00,
		TrabajadorNombreCompleto:  "Sanches, Maria",
		TrabajadorNumeroDocumento: "99992222",
	}

	conceptosPlaza := []models.ConceptoPlanilla{
		{
			MaestroID:     2,
			CodigoInterno: "ASIG_FAM_DL728",
			CodigoSunat:   "ASIG_FAM_DL728",
			Tipo:          "INGRESO",
			Monto:         0.00, // Should be calculated dynamically (10% of RMV)
			Frecuencia:    "1,2,3,4,5,6,7,8,9,10,11,12",
		},
	}

	job := models.JobPlanilla{
		Contrato:           contrato,
		ConceptosPlaza:     conceptosPlaza,
		MesActual:          5, // May
		Anio:               2026,
		ParametrosGlobales: map[string]float64{"RMV": 1025.00},
		MapaCodigos:        map[string]int{"0705": 99},
	}

	boleta, err := s.calcularBoletaContrato(job)
	if err != nil {
		t.Fatalf("Error calculando boleta: %v", err)
	}

	var asigFamMonto float64
	for _, lc := range boleta.LineasConceptos {
		if lc.CodigoSunat == "ASIG_FAM_DL728" {
			asigFamMonto = lc.Monto
		}
	}

	// 10% of 1025.00 = 102.50
	if asigFamMonto != 102.50 {
		t.Errorf("got asignacion familiar = %v; want 102.50", asigFamMonto)
	}
}
