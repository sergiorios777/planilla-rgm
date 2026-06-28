package services

import (
	"bytes"
	"math"
	"testing"
	"time"

	"planilla-rgm/internal/config"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"

	"github.com/joho/godotenv"
	"github.com/xuri/excelize/v2"
)

func TestCtsService(t *testing.T) {
	db, err := config.InitDB()
	if err != nil {
		t.Skip("Saltando test de base de datos local:", err)
		return
	}

	// 1. Crear un tenant temporal
	var tenantID int
	err = db.QueryRow("INSERT INTO tenants (nombre, ruc, activo) VALUES ('CTS Test Tenant', '10203040506', true) RETURNING id").Scan(&tenantID)
	if err != nil {
		t.Fatalf("Error creando tenant: %v", err)
	}
	defer func() {
		db.Exec("DELETE FROM tenants WHERE id = $1", tenantID)
	}()

	// 2. Crear un trabajador y un puesto
	var trabajadorID int
	err = db.QueryRow(`
		INSERT INTO trabajadores (tenant_id, tipo_documento, numero_documento, nombres, apellido_paterno, apellido_materno, fecha_nacimiento, sexo, regimen_pensionario)
		VALUES ($1, 'DNI', '99991111', 'Juan', 'Perez', 'Gomez', '1990-01-01', 'M', 'ONP') RETURNING id`, tenantID).Scan(&trabajadorID)
	if err != nil {
		t.Fatalf("Error creando trabajador: %v", err)
	}

	var regimenID int
	err = db.QueryRow("SELECT id FROM regimenes_laborales WHERE codigo = '728'").Scan(&regimenID)
	if err != nil {
		t.Fatalf("Error obteniendo regimen DL 728: %v", err)
	}

	var puestoID int
	err = db.QueryRow(`
		INSERT INTO puestos (tenant_id, regimen_id, nombre, estado, sueldo_presupuestado)
		VALUES ($1, $2, 'Obrero Municipal', 'VACANTE', 1500.00) RETURNING id`, tenantID, regimenID).Scan(&puestoID)
	if err != nil {
		t.Fatalf("Error creando puesto: %v", err)
	}

	// 3. Crear un contrato DL 728
	var contratoID int
	err = db.QueryRow(`
		INSERT INTO contratos (tenant_id, trabajador_id, puesto_id, fecha_inicio, activo)
		VALUES ($1, $2, $3, '2025-01-01', true) RETURNING id`, tenantID, trabajadorID, puestoID).Scan(&contratoID)
	if err != nil {
		t.Fatalf("Error creando contrato: %v", err)
	}

	repo := repository.NewCtsRepository(db)
	svc := NewCtsService(repo, db)

	// 4. Ejecutar el cálculo semestral
	count, err := svc.ProcesarCtsSemestral(tenantID, 2026, "MAYO")
	if err != nil {
		t.Fatalf("Error al procesar CTS semestral: %v", err)
	}

	if count != 1 {
		t.Errorf("got %d trabajadores procesados; want 1", count)
	}
}



func TestExcelParsingInMemory(t *testing.T) {
	// Generar un Excel en memoria para testear el parser
	f := excelize.NewFile()
	sheet := "Sheet1"
	f.SetCellValue(sheet, "A1", "Documento")
	f.SetCellValue(sheet, "B1", "Monto")
	f.SetCellValue(sheet, "A2", "99991111")
	f.SetCellValue(sheet, "B2", "1200.00")

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatalf("error writing excel to buffer: %v", err)
	}

	db, err := config.InitDB()
	if err != nil {
		t.Skip("Saltando test de base de datos local:", err)
		return
	}

	// Creamos registros temporales de prueba y limpiamos al terminar
	var tenantID int
	err = db.QueryRow("INSERT INTO tenants (nombre, ruc, activo) VALUES ('CTS Excel Test Tenant', '20203040506', true) RETURNING id").Scan(&tenantID)
	if err != nil {
		t.Fatalf("error al crear tenant: %v", err)
	}
	defer func() {
		db.Exec("DELETE FROM tenants WHERE id = $1", tenantID)
	}()

	var trabajadorID int
	err = db.QueryRow(`
		INSERT INTO trabajadores (tenant_id, tipo_documento, numero_documento, nombres, apellido_paterno, apellido_materno, fecha_nacimiento, sexo, regimen_pensionario)
		VALUES ($1, 'DNI', '99991111', 'Maria', 'Paz', 'Flores', '1991-01-01', 'F', 'ONP') RETURNING id`, tenantID).Scan(&trabajadorID)
	if err != nil {
		t.Fatalf("error al crear trabajador: %v", err)
	}

	var regimenID int
	err = db.QueryRow("SELECT id FROM regimenes_laborales WHERE codigo = '728'").Scan(&regimenID)
	if err != nil {
		t.Fatalf("error obteniendo regimen DL 728: %v", err)
	}

	var puestoID int
	err = db.QueryRow(`
		INSERT INTO puestos (tenant_id, regimen_id, nombre, estado, sueldo_presupuestado)
		VALUES ($1, $2, 'Auxiliar', 'VACANTE', 1200.00) RETURNING id`, tenantID, regimenID).Scan(&puestoID)
	if err != nil {
		t.Fatalf("error al crear puesto: %v", err)
	}

	var contratoID int
	err = db.QueryRow(`
		INSERT INTO contratos (tenant_id, trabajador_id, puesto_id, fecha_inicio, activo)
		VALUES ($1, $2, $3, '2025-01-01', true) RETURNING id`, tenantID, trabajadorID, puestoID).Scan(&contratoID)
	if err != nil {
		t.Fatalf("error al crear contrato: %v", err)
	}

	repo := repository.NewCtsRepository(db)
	svc := NewCtsService(repo, db)

	// Crear una planilla cabecera temporal
	p := &models.PlanillaCts{
		TenantID: tenantID,
		Anio:     2026,
		Periodo:  "MAYO",
		Estado:   "BORRADOR",
	}
	if err := repo.CrearPlanillaCts(p); err != nil {
		t.Fatalf("error al crear planilla cabecera: %v", err)
	}

	// Insertar detalle vacío
	detalles := []models.PlanillaCtsDetalle{
		{
			PlanillaCtsID:          p.ID,
			ContratoID:             contratoID,
			SueldoBasico:           1200.00,
			AsignacionFamilia:      0,
			SextoGratificacion:     0,
			PromedioVariables:      0,
			RemuneracionComputable: 1200.00,
			MesesComputables:       6,
			DiasFaltas:             0,
			MontoDescuentoFaltas:   0,
			MontoCts:               600.00,
		},
	}
	if err := repo.GuardarDetallesCts(detalles); err != nil {
		t.Fatalf("error al guardar detalles iniciales: %v", err)
	}

	// Correr el parser sobre el Excel en memoria
	procesados, err := svc.ProcesarExcelGratificaciones(p.ID, &buf)
	if err != nil {
		t.Fatalf("error al procesar excel: %v", err)
	}

	if procesados != 1 {
		t.Errorf("got %d rows processed; want 1", procesados)
	}

	// Verificar que el sexto de gratificación se actualizó a 1200 / 6 = 200
	detallesActuales, err := repo.ObtenerDetallesCts(p.ID)
	if err != nil {
		t.Fatalf("error al obtener detalles: %v", err)
	}

	if len(detallesActuales) != 1 {
		t.Fatalf("got %d detalles; want 1", len(detallesActuales))
	}

	d := detallesActuales[0]
	if d.SextoGratificacion != 200.00 {
		t.Errorf("got sexto = %v; want 200.00", d.SextoGratificacion)
	}

	// Remuneración computable = 1200 (sueldo) + 200 (grati/6) = 1400.00
	if d.RemuneracionComputable != 1400.00 {
		t.Errorf("got remComputable = %v; want 1400.00", d.RemuneracionComputable)
	}

	// CTS neto = (1400 / 12) * 6 = 700.00
	if d.MontoCts != 700.00 {
		t.Errorf("got montoCts = %v; want 700.00", d.MontoCts)
	}
}

func TestCalcularLiquidacionLey30057(t *testing.T) {
	db, err := config.InitDB()
	if err != nil {
		t.Skip("Saltando test de base de datos local:", err)
		return
	}

	// Creamos registros temporales de prueba y limpiamos al terminar
	var tenantID int
	err = db.QueryRow("INSERT INTO tenants (nombre, ruc, activo) VALUES ('CTS Cese Test Tenant', '20203040507', true) RETURNING id").Scan(&tenantID)
	if err != nil {
		t.Fatalf("error al crear tenant: %v", err)
	}
	defer func() {
		db.Exec("DELETE FROM tenants WHERE id = $1", tenantID)
	}()

	var trabajadorID int
	err = db.QueryRow(`
		INSERT INTO trabajadores (tenant_id, tipo_documento, numero_documento, nombres, apellido_paterno, apellido_materno, fecha_nacimiento, sexo, regimen_pensionario)
		VALUES ($1, 'DNI', '99991112', 'Carlos', 'Rojas', 'Lima', '1985-05-15', 'M', 'ONP') RETURNING id`, tenantID).Scan(&trabajadorID)
	if err != nil {
		t.Fatalf("error al crear trabajador: %v", err)
	}

	var regimenID int
	err = db.QueryRow("SELECT id FROM regimenes_laborales WHERE codigo = '30057'").Scan(&regimenID)
	if err != nil {
		t.Fatalf("error obteniendo regimen Ley 30057: %v", err)
	}

	var puestoID int
	err = db.QueryRow(`
		INSERT INTO puestos (tenant_id, regimen_id, nombre, estado, sueldo_presupuestado)
		VALUES ($1, $2, 'Servidor Civil', 'VACANTE', 3000.00) RETURNING id`, tenantID, regimenID).Scan(&puestoID)
	if err != nil {
		t.Fatalf("error al crear puesto: %v", err)
	}

	var contratoID int
	err = db.QueryRow(`
		INSERT INTO contratos (tenant_id, trabajador_id, puesto_id, fecha_inicio, activo)
		VALUES ($1, $2, $3, '2020-01-01', true) RETURNING id`, tenantID, trabajadorID, puestoID).Scan(&contratoID)
	if err != nil {
		t.Fatalf("error al crear contrato: %v", err)
	}

	vacSvc := NewVacacionesService(repository.NewBaseRegimenRepository(db))
	liqSvc := NewLiquidacionService(db, vacSvc)

	// Calcular liquidación simulada ingresando manualmente fecha inicio y cese
	// Trabajado: 3 años y 2 meses (total 38 meses)
	inicio := time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC)
	cese := time.Date(2023, time.March, 1, 0, 0, 0, 0, time.UTC)

	l, err := liqSvc.CalcularLiquidacion(contratoID, inicio, cese, "RENUNCIA", 0, 0)
	if err != nil {
		t.Fatalf("error al calcular liquidación: %v", err)
	}

	if l.AnosServicios != 3 || l.MesesServicios != 2 {
		t.Errorf("got %d años, %d meses; want 3 años, 2 meses", l.AnosServicios, l.MesesServicios)
	}

	// Ley 30057: (Promedio mensual / 12) * mesesTotales
	// mesesTotales = 38
	// sueldo = 3000.00 (por defecto ya que no hay planillas históricas en la prueba)
	// CTS = (3000 / 12) * 38 = 250 * 38 = 9500.00
	expectedCts := 9500.00
	if math.Abs(l.MontoCts-expectedCts) > 0.001 {
		t.Errorf("got CTS = %v; want %v", l.MontoCts, expectedCts)
	}
}

func TestCalcularLiquidacionVacacionesTruncasDataDriven(t *testing.T) {
	_ = godotenv.Load("../../.env")
	db, err := config.InitDB()
	if err != nil {
		t.Skip("Saltando test de base de datos local:", err)
		return
	}

	// 1. Limpieza y creación de tenant temporal
	var tenantID int
	err = db.QueryRow("INSERT INTO tenants (nombre, ruc, activo) VALUES ('CTS Vac Test Tenant', '20203040509', true) RETURNING id").Scan(&tenantID)
	if err != nil {
		t.Fatalf("error al crear tenant: %v", err)
	}
	defer func() {
		db.Exec("DELETE FROM base_regimen_tenant WHERE tenant_id = $1", tenantID)
		db.Exec("DELETE FROM conceptos_tenant WHERE tenant_id = $1", tenantID)
		db.Exec("DELETE FROM tenants WHERE id = $1", tenantID)
	}()

	var trabajadorID int
	err = db.QueryRow(`
		INSERT INTO trabajadores (tenant_id, tipo_documento, numero_documento, nombres, apellido_paterno, apellido_materno, fecha_nacimiento, sexo, regimen_pensionario)
		VALUES ($1, 'DNI', '99991119', 'Ramiro', 'Vaca', 'Test', '1988-06-12', 'M', 'ONP') RETURNING id`, tenantID).Scan(&trabajadorID)
	if err != nil {
		t.Fatalf("error al crear trabajador: %v", err)
	}

	var regimenID int
	err = db.QueryRow("SELECT id FROM regimenes_laborales WHERE codigo = '728'").Scan(&regimenID)
	if err != nil {
		t.Fatalf("error obteniendo regimen DL 728: %v", err)
	}

	var puestoID int
	err = db.QueryRow(`
		INSERT INTO puestos (tenant_id, regimen_id, nombre, estado, sueldo_presupuestado)
		VALUES ($1, $2, 'Obrero Test', 'VACANTE', 2400.00) RETURNING id`, tenantID, regimenID).Scan(&puestoID)
	if err != nil {
		t.Fatalf("error al crear puesto: %v", err)
	}

	var contratoID int
	err = db.QueryRow(`
		INSERT INTO contratos (tenant_id, trabajador_id, puesto_id, fecha_inicio, activo)
		VALUES ($1, $2, $3, '2025-01-01', true) RETURNING id`, tenantID, trabajadorID, puestoID).Scan(&contratoID)
	if err != nil {
		t.Fatalf("error al crear contrato: %v", err)
	}

	// 2. Sincronizar conceptos espejo para el tenant
	conceptoModeloRepo := repository.NewConceptoModeloRepository(db)
	conceptoModeloService := NewConceptoModeloService(conceptoModeloRepo, db)
	// Clonar los modelos globales al tenant
	_, _ = db.Exec(`
		INSERT INTO conceptos_tenant (tenant_id, concepto_id, nombre_personalizado, frecuencia_meses, activo, clasificador_id, es_extraordinario, requiere_monto, modelo_id)
		SELECT $1, concepto_id, nombre_personalizado, frecuencia_meses, true, clasificador_id, es_extraordinario, requiere_monto, id
		FROM conceptos_modelo
		ON CONFLICT (tenant_id, modelo_id) DO NOTHING
	`, tenantID)

	// Sembrar base regimen tenant
	err = conceptoModeloService.SembrarBaseRegimenTenant(tenantID)
	if err != nil {
		t.Fatalf("error al sembrar base_regimen_tenant: %v", err)
	}

	// 3. Obtener el ID del concepto tenant para Remuneración Obrero Permanente
	var conceptoTenantID int
	err = db.QueryRow("SELECT id FROM conceptos_tenant WHERE tenant_id = $1 AND modelo_id = 127", tenantID).Scan(&conceptoTenantID)
	if err != nil {
		t.Fatalf("error obteniendo concepto tenant espejo de MUC/Sueldo: %v", err)
	}

	// 4. Asignar el concepto al puesto (Sueldo = 2400)
	_, err = db.Exec("INSERT INTO puesto_conceptos (puesto_id, concepto_tenant_id, monto, activo) VALUES ($1, $2, 2400.00, true)", puestoID, conceptoTenantID)
	if err != nil {
		t.Fatalf("error asignando sueldo al puesto: %v", err)
	}

	// 5. Calcular liquidación simulada
	// Trabajado: del 2025-01-01 al 2025-04-16 (3 meses completos y 15 días)
	inicio := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	cese := time.Date(2025, time.April, 16, 0, 0, 0, 0, time.UTC)

	vacSvc := NewVacacionesService(repository.NewBaseRegimenRepository(db))
	liqSvc := NewLiquidacionService(db, vacSvc)

	l, err := liqSvc.CalcularLiquidacion(contratoID, inicio, cese, "RENUNCIA", 0, 0)
	if err != nil {
		t.Fatalf("error al calcular liquidación: %v", err)
	}

	// Computable: 2400.00
	// Meses: 3, Dias: 15
	// Vacaciones = (2400/12)*3 + (2400/360)*15 = 200*3 + 6.6667*15 = 600 + 100 = 700.00
	expectedVac := 700.00
	if math.Abs(l.MontoVacacionesTruncas-expectedVac) > 0.001 {
		t.Errorf("got Vacaciones Truncas = %v; want %v", l.MontoVacacionesTruncas, expectedVac)
	}
}

