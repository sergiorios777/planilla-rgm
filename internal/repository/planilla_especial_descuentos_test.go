package repository_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"planilla-rgm/internal/config"
	"planilla-rgm/internal/repository"
)

func TestProcesarPlanillaEspecial_ConRetencionesJudiciales(t *testing.T) {
	_ = godotenv.Load("../../.env")
	db, err := config.InitDB()
	if err != nil {
		t.Skip("Saltando test de BD local:", err)
		return
	}

	ctx := context.Background()
	repo := repository.NewPlanillaRepository(db)

	// 1. Crear tenant temporal
	var tenantID int
	testRuc := fmt.Sprintf("20%09d", time.Now().UnixNano()%1000000000)
	err = db.QueryRow("INSERT INTO tenants (nombre, ruc, activo) VALUES ('Tenant Test Especial', $1, true) RETURNING id", testRuc).Scan(&tenantID)
	if err != nil {
		t.Fatalf("Error creando tenant: %v", err)
	}
	defer func() {
		db.Exec("DELETE FROM tenants WHERE id = $1", tenantID)
	}()

	// 2. Crear trabajador y puesto
	var trabajadorID int
	docTest := fmt.Sprintf("%08d", time.Now().UnixNano()%100000000)
	err = db.QueryRow(`
		INSERT INTO trabajadores (tenant_id, numero_documento, nombres, apellido_paterno, apellido_materno, activo)
		VALUES ($1, $2, 'Juan', 'Perez', 'Gomez', true) RETURNING id
	`, tenantID, docTest).Scan(&trabajadorID)
	if err != nil {
		t.Fatalf("Error creando trabajador: %v", err)
	}

	var puestoID int
	err = db.QueryRow(`
		INSERT INTO puestos (tenant_id, nombre, sueldo_presupuestado, regimen_id, activo)
		VALUES ($1, 'Especialista Administrativo', 2500.00, 1, true) RETURNING id
	`, tenantID).Scan(&puestoID)
	if err != nil {
		t.Fatalf("Error creando puesto: %v", err)
	}

	var contratoID int
	err = db.QueryRow(`
		INSERT INTO contratos (tenant_id, trabajador_id, puesto_id, fecha_inicio, activo)
		VALUES ($1, $2, $3, '2026-01-01', true) RETURNING id
	`, tenantID, trabajadorID, puestoID).Scan(&contratoID)
	if err != nil {
		t.Fatalf("Error creando contrato: %v", err)
	}

	// 3. Crear concepto de ingreso extraordinario y concepto de retención
	var maestroIngresoID, maestroRetencionID int
	_ = db.QueryRow(`SELECT id FROM conceptos_maestros WHERE UPPER(tipo) = 'INGRESO' LIMIT 1`).Scan(&maestroIngresoID)
	_ = db.QueryRow(`SELECT id FROM conceptos_maestros WHERE UPPER(tipo) = 'RETENCION' LIMIT 1`).Scan(&maestroRetencionID)

	var conceptoBonoID, conceptoRetencionID int
	err = db.QueryRow(`
		INSERT INTO conceptos_tenant (tenant_id, concepto_id, nombre_personalizado, activo, es_extraordinario)
		VALUES ($1, $2, 'Bono Extraordinario DU 012', true, true) RETURNING id
	`, tenantID, maestroIngresoID).Scan(&conceptoBonoID)
	if err != nil {
		t.Fatalf("Error creando concepto bono: %v", err)
	}

	err = db.QueryRow(`
		INSERT INTO conceptos_tenant (tenant_id, concepto_id, nombre_personalizado, activo)
		VALUES ($1, $2, 'Retención Judicial Alimentos', true) RETURNING id
	`, tenantID, maestroRetencionID).Scan(&conceptoRetencionID)
	if err != nil {
		t.Fatalf("Error creando concepto retencion: %v", err)
	}

	// 4. Crear Descuento Judicial del 30% para el trabajador
	var descuentoID int
	err = db.QueryRow(`
		INSERT INTO descuentos (
			tenant_id, trabajador_id, concepto_tenant_id, tipo_descuento, documento_ordenador,
			descripcion, tipo_calculo, base_calculo, porcentaje, inicio_vigencia, activo
		) VALUES (
			$1, $2, $3, 'JUDICIAL', 'RESOLUCION',
			'Retención Judicial Alimentos 30%', 'PORCENTAJE', 'BRUTO_AFECTO', 30.00, '2026-01-01', true
		) RETURNING id
	`, tenantID, trabajadorID, conceptoRetencionID).Scan(&descuentoID)
	if err != nil {
		t.Fatalf("Error creando descuento: %v", err)
	}

	// Simular que el descuento tiene asociados conceptos mensuales regulares de su puesto (ej. Remuneración Básica)
	_, _ = db.Exec(`INSERT INTO descuento_conceptos (descuento_id, concepto_tenant_id) VALUES ($1, 999999)`, descuentoID)

	// 5. Crear Planilla Especial en estado BORRADOR
	var planillaID int
	err = db.QueryRow(`
		INSERT INTO planillas (tenant_id, descripcion, anio, mes, tipo, es_extraordinaria, estado)
		VALUES ($1, 'Planilla Extraordinaria Bono DU 012', 2026, 8, 'EXTRAORDINARIA', true, 'BORRADOR')
		RETURNING id
	`, tenantID).Scan(&planillaID)
	if err != nil {
		t.Fatalf("Error creando planilla: %v", err)
	}

	// CASO A: Procesar con aplicarRetencionesJudiciales = true
	conceptosInput := []repository.PlanillaEspecialConceptoInput{
		{ConceptoTenantID: conceptoBonoID, Monto: 500.00},
	}
	err = repo.ProcesarPlanillaEspecial(ctx, planillaID, tenantID, conceptosInput, []int{contratoID}, nil, true)
	if err != nil {
		t.Fatalf("Error en ProcesarPlanillaEspecial (con retención): %v", err)
	}

	// Validar detalle
	var totalIngresos, totalRetenciones, netoPagar float64
	err = db.QueryRow(`
		SELECT total_ingresos, total_retenciones, neto_pagar 
		FROM planilla_detalles WHERE planilla_id = $1 AND contrato_id = $2
	`, planillaID, contratoID).Scan(&totalIngresos, &totalRetenciones, &netoPagar)
	if err != nil {
		t.Fatalf("Error consultando detalle procesado: %v", err)
	}

	// 30% de 500.00 = 150.00
	if totalIngresos != 500.00 {
		t.Errorf("got total_ingresos = %v; want 500.00", totalIngresos)
	}
	if totalRetenciones != 150.00 {
		t.Errorf("got total_retenciones = %v; want 150.00", totalRetenciones)
	}
	if netoPagar != 350.00 {
		t.Errorf("got neto_pagar = %v; want 350.00", netoPagar)
	}

	// Verificar líneas de planilla_conceptos
	var countIngreso, countRetencion int
	_ = db.QueryRow(`
		SELECT COUNT(*) FROM planilla_conceptos pc
		INNER JOIN planilla_detalles pd ON pc.planilla_detalle_id = pd.id
		WHERE pd.planilla_id = $1 AND pc.tipo_concepto = 'INGRESO'
	`, planillaID).Scan(&countIngreso)

	_ = db.QueryRow(`
		SELECT COUNT(*) FROM planilla_conceptos pc
		INNER JOIN planilla_detalles pd ON pc.planilla_detalle_id = pd.id
		WHERE pd.planilla_id = $1 AND pc.tipo_concepto = 'RETENCION'
	`, planillaID).Scan(&countRetencion)

	if countIngreso != 1 {
		t.Errorf("got %d conceptos INGRESO; want 1", countIngreso)
	}
	if countRetencion != 1 {
		t.Errorf("got %d conceptos RETENCION; want 1", countRetencion)
	}

	// CASO B: Reprocesar con aplicarRetencionesJudiciales = false (Bono inembargable por ley)
	err = repo.ProcesarPlanillaEspecial(ctx, planillaID, tenantID, conceptosInput, []int{contratoID}, nil, false)
	if err != nil {
		t.Fatalf("Error en ProcesarPlanillaEspecial (sin retención): %v", err)
	}

	err = db.QueryRow(`
		SELECT total_ingresos, total_retenciones, neto_pagar 
		FROM planilla_detalles WHERE planilla_id = $1 AND contrato_id = $2
	`, planillaID, contratoID).Scan(&totalIngresos, &totalRetenciones, &netoPagar)
	if err != nil {
		t.Fatalf("Error consultando detalle procesado: %v", err)
	}

	if totalIngresos != 500.00 {
		t.Errorf("got total_ingresos = %v; want 500.00", totalIngresos)
	}
	if totalRetenciones != 0.00 {
		t.Errorf("got total_retenciones = %v; want 0.00", totalRetenciones)
	}
	if netoPagar != 500.00 {
		t.Errorf("got neto_pagar = %v; want 500.00", netoPagar)
	}

	// CASO C: ObtenerFormulacionEspecial debe devolver los conceptos (solo ingresos) y trabajadores formulados
	conceptosForm, trabsForm, errForm := repo.ObtenerFormulacionEspecial(planillaID, tenantID)
	if errForm != nil {
		t.Fatalf("Error en ObtenerFormulacionEspecial: %v", errForm)
	}
	if len(conceptosForm) != 1 {
		t.Errorf("got %d conceptos; want 1", len(conceptosForm))
	}
	if len(trabsForm) != 1 {
		t.Errorf("got %d trabajadores; want 1", len(trabsForm))
	}
	if len(trabsForm) > 0 {
		if trabsForm[0].ContratoID != contratoID {
			t.Errorf("got contratoID = %d; want %d", trabsForm[0].ContratoID, contratoID)
		}
		if !trabsForm[0].TieneRetencionJudicial {
			t.Errorf("esperaba TieneRetencionJudicial = true")
		}
		if trabsForm[0].MontoRetencionJudicial <= 0 {
			t.Errorf("esperaba MontoRetencionJudicial > 0, got %v", trabsForm[0].MontoRetencionJudicial)
		}
	}

	// CASO D: AgregarBeneficiariosBorrador debe funcionar sin 'driver: bad connection'
	var trabajador2ID, contrato2ID int
	doc2 := fmt.Sprintf("%08d", (time.Now().UnixNano()+100)%100000000)
	_ = db.QueryRow(`
		INSERT INTO trabajadores (tenant_id, numero_documento, nombres, apellido_paterno, apellido_materno, activo)
		VALUES ($1, $2, 'Maria', 'Lopez', 'Diaz', true) RETURNING id
	`, tenantID, doc2).Scan(&trabajador2ID)
	_ = db.QueryRow(`
		INSERT INTO contratos (tenant_id, trabajador_id, puesto_id, fecha_inicio, activo)
		VALUES ($1, $2, $3, '2026-01-01', true) RETURNING id
	`, tenantID, trabajador2ID, puestoID).Scan(&contrato2ID)

	agregados, omitidos, errAdd := repo.AgregarBeneficiariosBorrador(ctx, planillaID, tenantID, []int{contrato2ID})
	if errAdd != nil {
		t.Fatalf("Error en AgregarBeneficiariosBorrador: %v", errAdd)
	}
	if agregados != 1 {
		t.Errorf("got agregados = %d; want 1 (omitidos=%d)", agregados, omitidos)
	}

	// Verificar que ahora hay 2 trabajadores y que el segundo no tiene retención judicial
	_, trabsForm2, errForm2 := repo.ObtenerFormulacionEspecial(planillaID, tenantID)
	if errForm2 != nil {
		t.Fatalf("Error en ObtenerFormulacionEspecial (2): %v", errForm2)
	}
	if len(trabsForm2) != 2 {
		t.Errorf("got %d trabajadores; want 2", len(trabsForm2))
	}
	for _, tr := range trabsForm2 {
		if tr.ContratoID == contrato2ID {
			if tr.TieneRetencionJudicial {
				t.Errorf("el trabajador 2 no debería tener retención judicial")
			}
			if tr.MontoRetencionJudicial != 0 {
				t.Errorf("el trabajador 2 debería tener MontoRetencionJudicial = 0, got %v", tr.MontoRetencionJudicial)
			}
		}
	}
}
