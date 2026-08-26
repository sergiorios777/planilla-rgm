package services_test

import (
	"testing"
	"time"

	"planilla-rgm/internal/models"
	"planilla-rgm/internal/services"
)

func TestPlanillaService_DescuentosJudiciales_NetoLey(t *testing.T) {
	service := services.NewPlanillaService(nil)

	// Conceptos de la plaza
	ctSueldoID := 101
	ctAsigFamID := 102
	ctJudicialID := 201

	conceptosPlaza := []models.ConceptoPlanilla{
		{
			MaestroID:        1,
			ConceptoTenantID: ctSueldoID,
			Tipo:             "INGRESO",
			Nombre:           "Remuneración Básica",
			Monto:            2500.00,
			Frecuencia:       "1,2,3,4,5,6,7,8,9,10,11,12",
			CodigoSunat:      "0121",
			CodigoInterno:    "REM_BASICA",
		},
		{
			MaestroID:        2,
			ConceptoTenantID: ctAsigFamID,
			Tipo:             "INGRESO",
			Nombre:           "Asignación Familiar",
			Monto:            102.50,
			Frecuencia:       "1,2,3,4,5,6,7,8,9,10,11,12",
			CodigoSunat:      "0201",
			CodigoInterno:    "ASIG_FAM",
		},
		{
			MaestroID:        3,
			ConceptoTenantID: 301,
			Tipo:             "RETENCION",
			Nombre:           "ONP 13%",
			Monto:            0,
			Frecuencia:       "1,2,3,4,5,6,7,8,9,10,11,12",
			CodigoSunat:      "0607",
			CodigoInterno:    "0607",
		},
	}

	// 30% sobre Neto de Ley (Sueldo + Asig Fam)
	descuentoJudicial := models.DescuentoConConceptos{
		Descuento: models.Descuento{
			ID:                 1,
			TrabajadorID:       10,
			ConceptoTenantID:   ctJudicialID,
			ConceptoNombre:     "Retención Judicial Alimentos",
			ConceptoCodigoSunat: "0703",
			TipoDescuento:      "JUDICIAL",
			Descripcion:        "Retención Alimentos Exp. 00234-2024",
			TipoCalculo:        "PORCENTAJE",
			BaseCalculo:        "NETO_LEY",
			Porcentaje:         30.00,
			Activo:             true,
		},
		ConceptosTenantIDs: []int{ctSueldoID, ctAsigFamID},
	}

	job := models.JobPlanilla{
		Contrato: models.ContratoPlanilla{
			ID:                        1,
			TrabajadorID:              10,
			TrabajadorNombreCompleto:  "PÉREZ LÓPEZ, JUAN",
			TrabajadorNumeroDocumento: "12345678",
			Regimen:                   "728",
			RegimenPensionario:        "ONP",
			FechaInicio:               time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		ConceptosPlaza: conceptosPlaza,
		MesActual:      5,
		Anio:           2026,
		ParametrosGlobales: map[string]float64{
			"RMV": 1025.00,
		},
		MapaCodigos: map[string]int{
			"0607": 3,
		},
		MapaAfectacionesGlobal: map[int][]int{
			3: {1, 2}, // ONP grava Maestro 1 y 2
		},
		DescuentosTrabajador: []models.DescuentoConConceptos{descuentoJudicial},
	}

	boleta, err := service.CalcularBoletaContratoExposed(job)
	if err != nil {
		t.Fatalf("Error calculando boleta: %v", err)
	}

	// Total Ingresos = 2500 + 102.50 = 2602.50
	expectedIngresos := 2602.50
	if boleta.TotalIngresos != expectedIngresos {
		t.Errorf("TotalIngresos esperado %.2f, obtenido %.2f", expectedIngresos, boleta.TotalIngresos)
	}

	// ONP = 13% de 2602.50 = 338.325 -> 338.33
	// Base Neta = 2602.50 - 338.325 = 2264.175
	// Retención Judicial 30% de 2264.175 = 679.2525 -> 679.25
	var montoJudicial float64
	var foundJudicial bool
	for _, linea := range boleta.LineasConceptos {
		if linea.CodigoSunat == "0703" {
			foundJudicial = true
			montoJudicial = linea.Monto
		}
	}

	if !foundJudicial {
		t.Fatalf("No se encontró la línea de retención judicial 0703 en la boleta")
	}

	expectedJudicial := 679.25
	if montoJudicial < expectedJudicial-0.02 || montoJudicial > expectedJudicial+0.02 {
		t.Errorf("Monto de Retención Judicial esperado ~%.2f, obtenido %.2f", expectedJudicial, montoJudicial)
	}

	expectedNeto := boleta.TotalIngresos - boleta.TotalRetenciones
	if boleta.NetoPagar != expectedNeto {
		t.Errorf("NetoPagar esperado %.2f, obtenido %.2f", expectedNeto, boleta.NetoPagar)
	}
}

func TestPlanillaService_DescuentosJudiciales_BrutoAfecto(t *testing.T) {
	service := services.NewPlanillaService(nil)

	ctSueldoID := 101
	ctBonoID := 102

	conceptosPlaza := []models.ConceptoPlanilla{
		{
			MaestroID:        1,
			ConceptoTenantID: ctSueldoID,
			Tipo:             "INGRESO",
			Nombre:           "Remuneración Básica",
			Monto:            2000.00,
			Frecuencia:       "1,2,3,4,5,6,7,8,9,10,11,12",
			CodigoSunat:      "0121",
		},
		{
			MaestroID:        2,
			ConceptoTenantID: ctBonoID,
			Tipo:             "INGRESO",
			Nombre:           "Bono Extraordinario",
			Monto:            500.00,
			Frecuencia:       "1,2,3,4,5,6,7,8,9,10,11,12",
			CodigoSunat:      "0312",
		},
	}

	// Cuota Sindical: 2% sobre BRUTO_AFECTO solo del Sueldo Básico (Concepto 101)
	descuentoSindical := models.DescuentoConConceptos{
		Descuento: models.Descuento{
			ID:                 2,
			TrabajadorID:       10,
			ConceptoTenantID:   202,
			ConceptoNombre:     "Cuota Sindical SUTRAMUN",
			ConceptoCodigoSunat: "0708",
			TipoDescuento:      "SINDICAL",
			Descripcion:        "Cuota Sindical",
			TipoCalculo:        "PORCENTAJE",
			BaseCalculo:        "BRUTO_AFECTO",
			Porcentaje:         2.00,
			Activo:             true,
		},
		ConceptosTenantIDs: []int{ctSueldoID}, // Solo afecta al sueldo
	}

	job := models.JobPlanilla{
		Contrato: models.ContratoPlanilla{
			ID:                        1,
			TrabajadorID:              10,
			TrabajadorNombreCompleto:  "QUISPE CONDORI, MARIO",
			TrabajadorNumeroDocumento: "87654321",
			Regimen:                   "728",
			RegimenPensionario:        "SIN_REGIMEN",
			FechaInicio:               time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		ConceptosPlaza:         conceptosPlaza,
		MesActual:              5,
		Anio:                   2026,
		ParametrosGlobales:     map[string]float64{},
		MapaCodigos:            map[string]int{},
		MapaAfectacionesGlobal: map[int][]int{},
		DescuentosTrabajador:   []models.DescuentoConConceptos{descuentoSindical},
	}

	boleta, err := service.CalcularBoletaContratoExposed(job)
	if err != nil {
		t.Fatalf("Error calculando boleta: %v", err)
	}

	// Sueldo 2000 * 2% = 40.00 (ignora el bono de 500)
	var montoSindical float64
	for _, linea := range boleta.LineasConceptos {
		if linea.CodigoSunat == "0708" {
			montoSindical = linea.Monto
		}
	}

	if montoSindical != 40.00 {
		t.Errorf("Cuota sindical esperada 40.00, obtenida %.2f", montoSindical)
	}
}

func TestPlanillaService_DescuentosPrestamo_MontoFijoYAmortizacion(t *testing.T) {
	service := services.NewPlanillaService(nil)

	conceptosPlaza := []models.ConceptoPlanilla{
		{
			MaestroID:        1,
			ConceptoTenantID: 101,
			Tipo:             "INGRESO",
			Nombre:           "Remuneración Básica",
			Monto:            3000.00,
			Frecuencia:       "1,2,3,4,5,6,7,8,9,10,11,12",
			CodigoSunat:      "0121",
		},
	}

	// Préstamo de cuota fija S/ 300.00 pero le restan solo S/ 150.00 de deuda total
	descuentoPrestamo := models.DescuentoConConceptos{
		Descuento: models.Descuento{
			ID:                 3,
			TrabajadorID:       10,
			ConceptoTenantID:   203,
			ConceptoNombre:     "Préstamo Banco de la Nación",
			ConceptoCodigoSunat: "0709",
			TipoDescuento:      "PRESTAMO",
			Descripcion:        "Préstamo Personal BN",
			TipoCalculo:        "MONTO_FIJO",
			MontoFijo:          300.00,
			MontoTotalDeuda:    1000.00,
			MontoAcumulado:     850.00, // Saldo restante = 150.00
			Activo:             true,
		},
	}

	job := models.JobPlanilla{
		Contrato: models.ContratoPlanilla{
			ID:                        1,
			TrabajadorID:              10,
			TrabajadorNombreCompleto:  "GARCIA ROSAS, ELENA",
			TrabajadorNumeroDocumento: "11223344",
			Regimen:                   "728",
			RegimenPensionario:        "SIN_REGIMEN",
			FechaInicio:               time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		ConceptosPlaza:         conceptosPlaza,
		MesActual:              5,
		Anio:                   2026,
		ParametrosGlobales:     map[string]float64{},
		MapaCodigos:            map[string]int{},
		MapaAfectacionesGlobal: map[int][]int{},
		DescuentosTrabajador:   []models.DescuentoConConceptos{descuentoPrestamo},
	}

	boleta, err := service.CalcularBoletaContratoExposed(job)
	if err != nil {
		t.Fatalf("Error calculando boleta: %v", err)
	}

	var montoPrestamo float64
	for _, linea := range boleta.LineasConceptos {
		if linea.CodigoSunat == "0709" {
			montoPrestamo = linea.Monto
		}
	}

	if montoPrestamo != 150.00 {
		t.Errorf("Préstamo amortizado tope esperado 150.00, obtenido %.2f", montoPrestamo)
	}
}

func TestPlanillaService_DescuentosInactivos_NoAfectan(t *testing.T) {
	service := services.NewPlanillaService(nil)

	conceptosPlaza := []models.ConceptoPlanilla{
		{
			MaestroID:        1,
			ConceptoTenantID: 101,
			Tipo:             "INGRESO",
			Nombre:           "Remuneración Básica",
			Monto:            2500.00,
			Frecuencia:       "1,2,3,4,5,6,7,8,9,10,11,12",
			CodigoSunat:      "0121",
		},
	}

	job := models.JobPlanilla{
		Contrato: models.ContratoPlanilla{
			ID:                        1,
			TrabajadorID:              10,
			TrabajadorNombreCompleto:  "LOPEZ VEGA, CARLOS",
			TrabajadorNumeroDocumento: "99887766",
			Regimen:                   "728",
			RegimenPensionario:        "SIN_REGIMEN",
			FechaInicio:               time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		ConceptosPlaza:         conceptosPlaza,
		MesActual:              5,
		Anio:                   2026,
		ParametrosGlobales:     map[string]float64{},
		MapaCodigos:            map[string]int{},
		MapaAfectacionesGlobal: map[int][]int{},
		DescuentosTrabajador:   []models.DescuentoConConceptos{}, // Sin descuentos activos
	}

	boleta, err := service.CalcularBoletaContratoExposed(job)
	if err != nil {
		t.Fatalf("Error calculando boleta: %v", err)
	}

	if boleta.TotalRetenciones != 0 {
		t.Errorf("TotalRetenciones esperado 0, obtenido %.2f", boleta.TotalRetenciones)
	}
	if boleta.NetoPagar != 2500.00 {
		t.Errorf("NetoPagar esperado 2500.00, obtenido %.2f", boleta.NetoPagar)
	}
}
