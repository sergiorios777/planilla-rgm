package repository_test

import (
	"bytes"
	"fmt"
	"html/template"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"planilla-rgm/internal/config"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
	"planilla-rgm/internal/services"
)

func TestPlameModalTrabajadores(t *testing.T) {
	_ = godotenv.Load("../../.env")
	db, err := config.InitDB()
	if err != nil {
		t.Skip("Saltando test de BD local:", err)
		return
	}

	repo := repository.NewPlanillaRepository(db)
	plameRepo := repository.NewPlameRepository(db)
	licRepo := repository.NewLicenciaVacacionRepository(db)
	plameSvc := services.NewPlameService(plameRepo, repo, licRepo)

	// 1. Crear tenant temporal
	var tenantID int
	testRuc := fmt.Sprintf("20%09d", time.Now().UnixNano()%1000000000)
	err = db.QueryRow("INSERT INTO tenants (nombre, ruc, activo) VALUES ('Tenant Test Plame Modal', $1, true) RETURNING id", testRuc).Scan(&tenantID)
	if err != nil {
		t.Fatalf("Error creando tenant: %v", err)
	}
	defer func() {
		db.Exec("DELETE FROM tenants WHERE id = $1", tenantID)
	}()

	// 2. Crear 2 trabajadores: uno con apellido materno NULL y otro con apellido materno normal
	var trab1ID, trab2ID int
	doc1 := fmt.Sprintf("%08d", time.Now().UnixNano()%100000000)
	err = db.QueryRow(`
		INSERT INTO trabajadores (tenant_id, numero_documento, nombres, apellido_paterno, apellido_materno, activo)
		VALUES ($1, $2, 'Carlos', 'Gomez', '', true) RETURNING id
	`, tenantID, doc1).Scan(&trab1ID)
	if err != nil {
		t.Fatalf("Error creando trabajador 1: %v", err)
	}

	doc2 := fmt.Sprintf("%08d", (time.Now().UnixNano()+1)%100000000)
	err = db.QueryRow(`
		INSERT INTO trabajadores (tenant_id, numero_documento, nombres, apellido_paterno, apellido_materno, activo)
		VALUES ($1, $2, 'Ana', 'Rios', 'Vargas', true) RETURNING id
	`, tenantID, doc2).Scan(&trab2ID)
	if err != nil {
		t.Fatalf("Error creando trabajador 2: %v", err)
	}

	// 3. Crear puesto y contratos
	var puestoID int
	err = db.QueryRow(`
		INSERT INTO puestos (tenant_id, nombre, sueldo_presupuestado, regimen_id, activo)
		VALUES ($1, 'Analista Plame', 3000.00, 1, true) RETURNING id
	`, tenantID).Scan(&puestoID)
	if err != nil {
		t.Fatalf("Error creando puesto: %v", err)
	}

	var contrato1ID, contrato2ID int
	err = db.QueryRow(`
		INSERT INTO contratos (tenant_id, trabajador_id, puesto_id, fecha_inicio, activo)
		VALUES ($1, $2, $3, '2026-01-01', true) RETURNING id
	`, tenantID, trab1ID, puestoID).Scan(&contrato1ID)
	if err != nil {
		t.Fatalf("Error creando contrato 1: %v", err)
	}

	err = db.QueryRow(`
		INSERT INTO contratos (tenant_id, trabajador_id, puesto_id, fecha_inicio, activo)
		VALUES ($1, $2, $3, '2026-01-01', true) RETURNING id
	`, tenantID, trab2ID, puestoID).Scan(&contrato2ID)
	if err != nil {
		t.Fatalf("Error creando contrato 2: %v", err)
	}

	// 4. Crear planilla
	var planillaID int
	err = db.QueryRow(`
		INSERT INTO planillas (tenant_id, mes, anio, tipo, descripcion, estado)
		VALUES ($1, 9, 2026, 'MENSUAL', 'Planilla Test Plame', 'BORRADOR') RETURNING id
	`, tenantID).Scan(&planillaID)
	if err != nil {
		t.Fatalf("Error creando planilla: %v", err)
	}

	// 5. Crear planilla_detalles
	var det1ID, det2ID int
	err = db.QueryRow(`
		INSERT INTO planilla_detalles (planilla_id, contrato_id, total_ingresos, total_retenciones, total_aportes, neto_pagar)
		VALUES ($1, $2, 3000.00, 0.00, 270.00, 3000.00) RETURNING id
	`, planillaID, contrato1ID).Scan(&det1ID)
	if err != nil {
		t.Fatalf("Error creando detalle 1: %v", err)
	}

	err = db.QueryRow(`
		INSERT INTO planilla_detalles (planilla_id, contrato_id, total_ingresos, total_retenciones, total_aportes, neto_pagar)
		VALUES ($1, $2, 3000.00, 0.00, 270.00, 3000.00) RETURNING id
	`, planillaID, contrato2ID).Scan(&det2ID)
	if err != nil {
		t.Fatalf("Error creando detalle 2: %v", err)
	}

	// 6. Crear concepto tenant y planilla_conceptos
	var concTenantID int
	err = db.QueryRow(`
		INSERT INTO conceptos_tenant (tenant_id, concepto_id, nombre_personalizado, activo, es_remunerativa)
		VALUES ($1, (SELECT id FROM conceptos_maestros LIMIT 1), 'Sueldo Básico', true, true) RETURNING id
	`, tenantID).Scan(&concTenantID)
	if err != nil {
		t.Fatalf("Error creando concepto tenant: %v", err)
	}

	// Insertar planilla_conceptos con codigo SUNAT '0121'
	_, err = db.Exec(`
		INSERT INTO planilla_conceptos (planilla_detalle_id, concepto_tenant_id, tipo_concepto, monto, codigo_sunat, nombre_en_boleta)
		VALUES ($1, $2, 'INGRESO', 3000.00, '0121', 'Sueldo Básico'),
		       ($3, $2, 'INGRESO', 3000.00, '0121', 'Sueldo Básico')
	`, det1ID, concTenantID, det2ID)
	if err != nil {
		t.Fatalf("Error creando planilla conceptos: %v", err)
	}

	// 7. Probar AsegurarSnapshot
	err = plameSvc.AsegurarSnapshot(planillaID, tenantID)
	if err != nil {
		t.Fatalf("Error en AsegurarSnapshot: %v", err)
	}

	// 8. Probar ObtenerConceptosAgrupados
	agrupados, err := plameSvc.ObtenerConceptosAgrupados(planillaID, tenantID)
	if err != nil {
		t.Fatalf("Error en ObtenerConceptosAgrupados: %v", err)
	}
	t.Logf("Conceptos agrupados obtenidos: %d", len(agrupados))
	for _, c := range agrupados {
		t.Logf(" -> CodigoSunat: '%s', Desc: '%s', Trabajadores: %d, Devengado: %.2f", c.CodigoSunatActual, c.DescripcionSunatActual, c.TotalTrabajadores, c.TotalDevengado)
	}

	// 9. Probar renderizar la vista principal y verificar el botón
	planilla, err := repo.ObtenerPorID(planillaID, tenantID)
	if err != nil {
		t.Fatalf("Error obteniendo planilla: %v", err)
	}

	tmplMain, err := template.ParseFiles("../../ui/templates/tenant/planilla_sunat_codigos_ui.html")
	if err != nil {
		t.Fatalf("Error parseando plantilla principal: %v", err)
	}
	var bufMain bytes.Buffer
	err = tmplMain.Execute(&bufMain, map[string]interface{}{
		"Planilla":  planilla,
		"Conceptos": agrupados,
		"Maestros":  []models.ConceptoMaestro{},
	})
	if err != nil {
		t.Fatalf("Error ejecutando plantilla principal: %v", err)
	}
	t.Logf("Vista principal renderizada con éxito! Longitud: %d bytes", bufMain.Len())
	
	// Buscar el fragmento del botón en el HTML renderizado
	renderedStr := bufMain.String()
	idxBtn := bytes.Index(bufMain.Bytes(), []byte("abrirModalEditarCodigoSunat"))
	if idxBtn != -1 {
		start := idxBtn - 50
		if start < 0 { start = 0 }
		end := idxBtn + 150
		if end > len(renderedStr) { end = len(renderedStr) }
		t.Logf("HTML renderizado del botón: \n%s", renderedStr[start:end])
	} else {
		t.Logf("NO se encontró 'abrirModalEditarCodigoSunat' en el HTML renderizado!")
	}

	codigoTest := agrupados[0].CodigoSunatActual
	t.Logf("Consultando trabajadores para codigo SUNAT: '%s'", codigoTest)

	// 9. Probar ObtenerTrabajadoresPorConcepto
	trabajadores, err := plameSvc.ObtenerTrabajadoresPorConcepto(planillaID, tenantID, codigoTest)
	if err != nil {
		t.Fatalf("Error en ObtenerTrabajadoresPorConcepto: %v", err)
	}
	t.Logf("Trabajadores obtenidos: %d", len(trabajadores))
	for _, tr := range trabajadores {
		t.Logf(" -> Doc: %s, Nombre: '%s', Dev: %.2f, Pag: %.2f", tr.NumeroDocumento, tr.NombreCompleto, tr.MontoDevengado, tr.MontoPagado)
	}

	if len(trabajadores) != 2 {
		t.Errorf("Se esperaban 2 trabajadores, se obtuvieron: %d", len(trabajadores))
	}

	// 10. Probar renderizar el template fragmento_trabajadores_por_concepto
	planilla, err = repo.ObtenerPorID(planillaID, tenantID)
	if err != nil {
		t.Fatalf("Error obteniendo planilla: %v", err)
	}

	datosTrab := map[string]interface{}{
		"Planilla":         planilla,
		"CodigoSunat":      codigoTest,
		"DescripcionSunat": "REMUNERACIÓN O JORNAL BÁSICO",
		"TotalDevengado":   6000.00,
		"TotalPagado":      6000.00,
		"TotalAjustados":   0,
		"Trabajadores":     trabajadores,
	}

	tmplTrab, err := template.ParseFiles("../../ui/templates/tenant/plame_concepto_trabajadores_ui.html")
	if err != nil {
		t.Fatalf("Error parseando plantilla plame_concepto_trabajadores_ui.html: %v", err)
	}

	var bufTrab bytes.Buffer
	err = tmplTrab.Execute(&bufTrab, datosTrab)
	if err != nil {
		t.Fatalf("Error ejecutando plame_concepto_trabajadores_ui.html: %v", err)
	}
	t.Logf("Vista plame_concepto_trabajadores_ui.html renderizada con éxito! Longitud: %d bytes", bufTrab.Len())

	// 11. Probar ObtenerDetalleTrabajador y su modal dentro de plame_concepto_trabajadores_ui.html
	if len(trabajadores) > 0 {
		detTrabID := trabajadores[0].PlanillaDetalleID
		conceptosDet, err := plameSvc.ObtenerDetalleTrabajador(detTrabID, tenantID)
		if err != nil {
			t.Fatalf("Error en ObtenerDetalleTrabajador: %v", err)
		}
		t.Logf("Conceptos detalle trabajador obtenidos: %d", len(conceptosDet))

		maestros, err := repo.ObtenerMaestrosSunat()
		if err != nil {
			t.Fatalf("Error obteniendo maestros SUNAT: %v", err)
		}

		datosModal := map[string]interface{}{
			"Planilla":          planilla,
			"PlanillaDetalleID": detTrabID,
			"TrabajadorNombre":  trabajadores[0].NombreCompleto,
			"TrabajadorDoc":     trabajadores[0].NumeroDocumento,
			"RegimenNombre":     trabajadores[0].RegimenNombre,
			"Conceptos":         conceptosDet,
			"Maestros":          maestros,
			"OrigenVista":       "trabajadores",
			"CodigoSunatFiltro": codigoTest,
		}

		var bufModal bytes.Buffer
		err = tmplTrab.ExecuteTemplate(&bufModal, "modal_editar_plame_trabajador_content", datosModal)
		if err != nil {
			t.Fatalf("Error ejecutando modal_editar_plame_trabajador_content: %v", err)
		}
		t.Logf("Modal editar trabajador renderizado con éxito! Longitud: %d bytes", bufModal.Len())

		// 12. Probar renderizar vista de reasignación masiva plame_reasignar_concepto_ui.html
		datosReasignar := map[string]interface{}{
			"Planilla":               planilla,
			"CodigoSunatActual":      codigoTest,
			"DescripcionSunatActual": "REMUNERACIÓN O JORNAL BÁSICO",
			"TipoConcepto":           "INGRESO",
			"TotalTrabajadores":      len(trabajadores),
			"TotalDevengado":         6000.00,
			"TotalPagado":            6000.00,
			"TrabajadoresAfectados":  trabajadores,
			"Maestros":               maestros,
		}

		tmplReasignar, err := template.ParseFiles("../../ui/templates/tenant/plame_reasignar_concepto_ui.html")
		if err != nil {
			t.Fatalf("Error parseando plantilla plame_reasignar_concepto_ui.html: %v", err)
		}

		var bufReasignar bytes.Buffer
		err = tmplReasignar.Execute(&bufReasignar, datosReasignar)
		if err != nil {
			t.Fatalf("Error ejecutando plame_reasignar_concepto_ui.html: %v", err)
		}
		t.Logf("Vista plame_reasignar_concepto_ui.html renderizada con éxito! Longitud: %d bytes", bufReasignar.Len())
	}
}

func TestPlameHubView(t *testing.T) {
	_ = godotenv.Load("../../.env")
	db, err := config.InitDB()
	if err != nil {
		t.Skip("Saltando test de BD local:", err)
		return
	}

	repo := repository.NewPlanillaRepository(db)
	plameRepo := repository.NewPlameRepository(db)
	licRepo := repository.NewLicenciaVacacionRepository(db)
	plameSvc := services.NewPlameService(plameRepo, repo, licRepo)

	// Crear tenant de prueba
	var tenantID int
	testRuc := fmt.Sprintf("20%09d", time.Now().UnixNano()%1000000000)
	err = db.QueryRow("INSERT INTO tenants (nombre, ruc, activo) VALUES ('Tenant Test Plame Hub', $1, true) RETURNING id", testRuc).Scan(&tenantID)
	if err != nil {
		t.Fatalf("Error creando tenant: %v", err)
	}
	defer func() {
		db.Exec("DELETE FROM tenants WHERE id = $1", tenantID)
	}()

	// 1. Obtener resumen del periodo
	resumen, err := plameSvc.ObtenerResumenPeriodo(tenantID, 2026, 9)
	if err != nil {
		t.Fatalf("Error en ObtenerResumenPeriodo: %v", err)
	}
	t.Logf("Resumen obtenido: %+v", resumen)

	// 2. Obtener planillas del periodo
	planillas, err := plameSvc.ObtenerPeriodoPlanillas(tenantID, 2026, 9)
	if err != nil {
		t.Fatalf("Error en ObtenerPeriodoPlanillas: %v", err)
	}
	t.Logf("Planillas del periodo: %d", len(planillas))

	// 3. Probar renderizar el template completo ui/templates/tenant/plame_hub_ui.html
	datos := map[string]interface{}{
		"Anio":      2026,
		"Mes":       9,
		"Resumen":   resumen,
		"Planillas": planillas,
	}

	tmpl, err := template.ParseFiles("../../ui/templates/tenant/plame_hub_ui.html")
	if err != nil {
		t.Fatalf("Error parseando plantilla plame_hub_ui.html: %v", err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, datos)
	if err != nil {
		t.Fatalf("Error ejecutando plantilla plame_hub_ui.html: %v", err)
	}
	t.Logf("Plantilla plame_hub_ui.html ejecutada con éxito! Longitud: %d bytes", buf.Len())

	// 4. Probar fragmento_planillas_periodo
	var bufFrag bytes.Buffer
	err = tmpl.ExecuteTemplate(&bufFrag, "fragmento_planillas_periodo", datos)
	if err != nil {
		t.Fatalf("Error ejecutando fragmento_planillas_periodo: %v", err)
	}
	t.Logf("Fragmento fragmento_planillas_periodo ejecutado con éxito! Longitud: %d bytes", bufFrag.Len())
}



