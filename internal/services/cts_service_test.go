package services

import (
	"bytes"
	"math"
	"testing"
	"time"

	"planilla-rgm/internal/config"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"

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

func TestCtsHelpers(t *testing.T) {
	// Test de Date Math helpers en CtsService
	repo := repository.NewCtsRepository(nil)
	svc := NewCtsService(repo, nil)

	t.Run("calcularMesesYAnosServicio", func(t *testing.T) {
		start := time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(2022, time.June, 15, 0, 0, 0, 0, time.UTC)

		anos, meses := svc.calcularMesesYAnosServicio(start, end)
		if anos != 2 || meses != 5 {
			t.Errorf("got %d años, %d meses; want 2 años, 5 meses", anos, meses)
		}
	})

	t.Run("calcularMesesInterseccion completo", func(t *testing.T) {
		start := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(2025, time.December, 31, 0, 0, 0, 0, time.UTC)

		desde := time.Date(2025, time.May, 1, 0, 0, 0, 0, time.UTC)
		hasta := time.Date(2025, time.October, 31, 23, 59, 59, 0, time.UTC)

		meses := calcularMesesInterseccion(start, &end, desde, hasta)
		if meses != 6 {
			t.Errorf("got %d meses; want 6 meses", meses)
		}
	})

	t.Run("calcularMesesInterseccion parcial", func(t *testing.T) {
		start := time.Date(2025, time.June, 15, 0, 0, 0, 0, time.UTC) // Empieza a mitad de junio
		end := time.Date(2025, time.September, 10, 0, 0, 0, 0, time.UTC)

		desde := time.Date(2025, time.May, 1, 0, 0, 0, 0, time.UTC)
		hasta := time.Date(2025, time.October, 31, 23, 59, 59, 0, time.UTC)

		// Debe contar sólo Julio y Agosto completos (2 meses)
		meses := calcularMesesInterseccion(start, &end, desde, hasta)
		if meses != 2 {
			t.Errorf("got %d meses; want 2 meses", meses)
		}
	})
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

	repo := repository.NewCtsRepository(db)
	svc := NewCtsService(repo, db)

	// Calcular liquidación simulada ingresando manualmente fecha inicio y cese
	// Trabajado: 3 años y 2 meses (total 38 meses)
	inicio := time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC)
	cese := time.Date(2023, time.March, 1, 0, 0, 0, 0, time.UTC)

	l, err := svc.CalcularLiquidacion(contratoID, inicio, cese, "RENUNCIA")
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
