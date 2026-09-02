package services

import (
	"fmt"
	"math"
	"planilla-rgm/internal/config"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
	"testing"
	"time"

	"github.com/joho/godotenv"
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

func TestPlanillaService_CalcularBoletaContrato_GratificacionCAS(t *testing.T) {
	s := &PlanillaService{Repo: nil}

	contrato := models.ContratoPlanilla{
		ID:                        103,
		PuestoID:                  203,
		Regimen:                   "1057",
		RegimenPensionario:        "ONP",
		FechaInicio:               time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC),
		SueldoBasicoHistorico:     3000.00,
		TrabajadorNombreCompleto:  "Ramirez, Jose",
		TrabajadorNumeroDocumento: "99993333",
	}

	conceptosPlaza := []models.ConceptoPlanilla{
		{
			MaestroID:     10,
			CodigoInterno: "2039",
			CodigoSunat:   "2039",
			Tipo:          "INGRESO",
			Monto:         3000.00,
			Frecuencia:    "1,2,3,4,5,6,7,8,9,10,11,12",
		},
		{
			MaestroID:     11,
			CodigoInterno: "GRATI_JUL_DL_1057",
			CodigoSunat:   "2006",
			Tipo:          "INGRESO",
			Monto:         0.00, // Dinámico
			Frecuencia:    "7",
		},
		{
			MaestroID:     12,
			CodigoInterno: "0804",
			CodigoSunat:   "0804",
			Tipo:          "APORTE",
			Monto:         0.00,
			Frecuencia:    "1,2,3,4,5,6,7,8,9,10,11,12",
		},
	}

	job := models.JobPlanilla{
		Contrato:               contrato,
		ConceptosPlaza:         conceptosPlaza,
		MesActual:              7, // Julio
		Anio:                   2026,
		ParametrosGlobales:     map[string]float64{"TASA_ESSALUD": 0.09, "UIT": 5500.00, "RMV": 1130.00},
		MapaCodigos:            map[string]int{"0705": 99},
		MapaAfectacionesGlobal: map[int][]int{12: {10}}, // Concepto 12 (EsSalud) se calcula sobre Concepto 10 (Sueldo 3000)
	}

	boleta, err := s.calcularBoletaContrato(job)
	if err != nil {
		t.Fatalf("Error calculando boleta: %v", err)
	}

	var gratiMonto, essaludMonto float64
	for _, lc := range boleta.LineasConceptos {
		if lc.CodigoSunat == "2006" {
			gratiMonto = lc.Monto
		}
		if lc.CodigoSunat == "0804" {
			essaludMonto = lc.Monto
		}
	}

	// 2026 -> 10% de S/3000 = S/300 base grati -> Grati = S/300.00
	// EsSalud grati (9% de S/300) = S/27.00
	// EsSalud sueldo CAS (tope 45% UIT = 2475, 9% de 2475) = S/222.75
	// Total EsSalud = 222.75 + 27.00 = 249.75
	if gratiMonto != 300.00 {
		t.Errorf("got gratificación CAS = %v; want 300.00", gratiMonto)
	}
	if math.Abs(essaludMonto-249.75) > 0.001 {
		t.Errorf("got EsSalud total = %v; want 249.75", essaludMonto)
	}
}

func TestAuditoriaYActualizacionCodigosSunat(t *testing.T) {
	_ = godotenv.Load("../../.env")
	db, err := config.InitDB()
	if err != nil {
		t.Skip("Saltando test de BD local:", err)
		return
	}

	// 1. Crear tenant temporal
	var tenantID int
	testRuc := fmt.Sprintf("20%09d", time.Now().UnixNano()%1000000000)
	err = db.QueryRow("INSERT INTO tenants (nombre, ruc, activo) VALUES ('Tenant Test SUNAT', $1, true) RETURNING id", testRuc).Scan(&tenantID)
	if err != nil {
		t.Fatalf("Error creando tenant: %v", err)
	}
	defer func() {
		db.Exec("DELETE FROM tenants WHERE id = $1", tenantID)
	}()

	// 2. Obtener dos conceptos maestros oficiales de SUNAT
	var maestroID1, maestroID2 int
	var cod1, cod2 string
	err = db.QueryRow("SELECT id, codigo FROM conceptos_maestros WHERE origen = 'sunat' AND codigo = '0121' LIMIT 1").Scan(&maestroID1, &cod1)
	if err != nil {
		// Fallback a cualquier maestro SUNAT
		err = db.QueryRow("SELECT id, codigo FROM conceptos_maestros WHERE origen = 'sunat' ORDER BY id ASC LIMIT 1").Scan(&maestroID1, &cod1)
		if err != nil {
			t.Fatalf("Error obteniendo maestro 1: %v", err)
		}
	}
	err = db.QueryRow("SELECT id, codigo FROM conceptos_maestros WHERE origen = 'sunat' AND id != $1 ORDER BY id DESC LIMIT 1", maestroID1).Scan(&maestroID2, &cod2)
	if err != nil {
		t.Fatalf("Error obteniendo maestro 2: %v", err)
	}

	// 3. Crear concepto tenant
	var conceptoTenantID int
	err = db.QueryRow(`
		INSERT INTO conceptos_tenant (tenant_id, concepto_id, nombre_personalizado, activo)
		VALUES ($1, $2, 'Sueldo Básico Personalizado', true) RETURNING id
	`, tenantID, maestroID1).Scan(&conceptoTenantID)
	if err != nil {
		t.Fatalf("Error creando concepto tenant: %v", err)
	}

	// 4. Crear trabajador y puesto
	var regimenID int
	err = db.QueryRow("SELECT id FROM regimenes_laborales LIMIT 1").Scan(&regimenID)
	if err != nil {
		t.Fatalf("Error obteniendo regimen laboral: %v", err)
	}

	var puestoID int
	err = db.QueryRow("INSERT INTO puestos (tenant_id, regimen_id, nombre, estado, sueldo_presupuestado) VALUES ($1, $2, 'Puesto Test', 'OCUPADO', 2500.00) RETURNING id", tenantID, regimenID).Scan(&puestoID)
	if err != nil {
		t.Fatalf("Error creando puesto: %v", err)
	}

	var trabajadorID int
	err = db.QueryRow(`
		INSERT INTO trabajadores (tenant_id, tipo_documento, numero_documento, nombres, apellido_paterno, apellido_materno, fecha_nacimiento, sexo, regimen_pensionario)
		VALUES ($1, 'DNI', '88887777', 'Carlos', 'Test', 'Sunat', '1990-01-01', 'M', 'ONP') RETURNING id
	`, tenantID).Scan(&trabajadorID)
	if err != nil {
		t.Fatalf("Error creando trabajador: %v", err)
	}

	var contratoID int
	err = db.QueryRow(`
		INSERT INTO contratos (tenant_id, trabajador_id, puesto_id, fecha_inicio, activo)
		VALUES ($1, $2, $3, '2026-01-01', true) RETURNING id
	`, tenantID, trabajadorID, puestoID).Scan(&contratoID)
	if err != nil {
		t.Fatalf("Error creando contrato: %v", err)
	}

	// 5. Crear planilla y detalles
	var planillaID int
	err = db.QueryRow(`
		INSERT INTO planillas (tenant_id, anio, mes, descripcion, estado, tipo)
		VALUES ($1, 2026, 8, 'Planilla Agosto Auditoria SUNAT', 'BORRADOR', 'ORDINARIA') RETURNING id
	`, tenantID).Scan(&planillaID)
	if err != nil {
		t.Fatalf("Error creando planilla: %v", err)
	}

	var detalleID int
	err = db.QueryRow(`
		INSERT INTO planilla_detalles (planilla_id, contrato_id, total_ingresos, total_retenciones, total_aportes, neto_pagar, trabajador_nombre_completo, trabajador_numero_documento)
		VALUES ($1, $2, 2500.00, 0.00, 0.00, 2500.00, 'Test Sunat Carlos', '88887777') RETURNING id
	`, planillaID, contratoID).Scan(&detalleID)
	if err != nil {
		t.Fatalf("Error creando detalle: %v", err)
	}

	// Insertar concepto en planilla_conceptos
	_, err = db.Exec(`
		INSERT INTO planilla_conceptos (planilla_detalle_id, concepto_tenant_id, maestro_id, tipo_concepto, monto, codigo_sunat, nombre_en_boleta)
		VALUES ($1, $2, $3, 'INGRESO', 2500.00, $4, 'Sueldo Básico Personalizado')
	`, detalleID, conceptoTenantID, maestroID1, cod1)
	if err != nil {
		t.Fatalf("Error insertando concepto de planilla: %v", err)
	}

	repo := repository.NewPlanillaRepository(db)

	// 6. Test ObtenerConceptosSunatAgrupados
	agrupados, err := repo.ObtenerConceptosSunatAgrupados(planillaID, tenantID)
	if err != nil {
		t.Fatalf("Error en ObtenerConceptosSunatAgrupados: %v", err)
	}
	if len(agrupados) != 1 {
		t.Fatalf("got %d conceptos agrupados; want 1", len(agrupados))
	}
	if agrupados[0].CodigoSunatActual != cod1 {
		t.Errorf("got codigo actual %s; want %s", agrupados[0].CodigoSunatActual, cod1)
	}
	if agrupados[0].TotalTrabajadores != 1 {
		t.Errorf("got total trabajadores %d; want 1", agrupados[0].TotalTrabajadores)
	}
	if agrupados[0].TotalMonto != 2500.00 {
		t.Errorf("got total monto %f; want 2500.00", agrupados[0].TotalMonto)
	}

	// 7. Test ObtenerMaestrosSunat
	maestros, err := repo.ObtenerMaestrosSunat()
	if err != nil || len(maestros) == 0 {
		t.Fatalf("Error en ObtenerMaestrosSunat: %v (len: %d)", err, len(maestros))
	}

	// 8. Test ActualizarCodigoSunatConceptoMasivo (con actualizarDefault = true)
	codigoRetornado, err := repo.ActualizarCodigoSunatConceptoMasivo(planillaID, tenantID, &conceptoTenantID, "", maestroID2, true)
	if err != nil {
		t.Fatalf("Error en ActualizarCodigoSunatConceptoMasivo: %v", err)
	}
	if codigoRetornado != cod2 {
		t.Errorf("got codigoRetornado %s; want %s", codigoRetornado, cod2)
	}

	// Verificar que planilla_conceptos se actualizó
	var nuevoCodGuardado string
	var nuevoMaestroGuardado int
	err = db.QueryRow("SELECT codigo_sunat, maestro_id FROM planilla_conceptos WHERE planilla_detalle_id = $1", detalleID).Scan(&nuevoCodGuardado, &nuevoMaestroGuardado)
	if err != nil {
		t.Fatalf("Error consultando planilla_conceptos actualizado: %v", err)
	}
	if nuevoCodGuardado != cod2 {
		t.Errorf("got codigo_sunat %s; want %s", nuevoCodGuardado, cod2)
	}
	if nuevoMaestroGuardado != maestroID2 {
		t.Errorf("got maestro_id %d; want %d", nuevoMaestroGuardado, maestroID2)
	}

	// Verificar que conceptos_tenant se actualizó a futuro
	var conceptoIDTenantActualizado int
	_ = db.QueryRow("SELECT concepto_id FROM conceptos_tenant WHERE id = $1", conceptoTenantID).Scan(&conceptoIDTenantActualizado)
	if conceptoIDTenantActualizado != maestroID2 {
		t.Errorf("got conceptos_tenant.concepto_id %d; want %d", conceptoIDTenantActualizado, maestroID2)
	}

	// 9. Verificar que ObtenerDatosPlameRemuneraciones emite el nuevo código
	plameRem, err := repo.ObtenerDatosPlameRemuneraciones(planillaID, tenantID)
	if err != nil {
		t.Fatalf("Error en ObtenerDatosPlameRemuneraciones: %v", err)
	}
	if len(plameRem) != 1 || plameRem[0].CodigoConcepto != cod2 {
		t.Errorf("PLAME .rem emitió código %v; want %s", plameRem, cod2)
	}

	// 10. Test bloqueo de planilla cerrada
	_, _ = db.Exec("UPDATE planillas SET estado = 'CERRADA' WHERE id = $1", planillaID)
	_, err = repo.ActualizarCodigoSunatConceptoMasivo(planillaID, tenantID, &conceptoTenantID, "", maestroID1, false)
	if err == nil {
		t.Errorf("ActualizarCodigoSunatConceptoMasivo debió fallar en planilla CERRADA, pero no retornó error")
	}
}

