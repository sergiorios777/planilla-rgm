package services

import (
	"github.com/joho/godotenv"
	"planilla-rgm/internal/config"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
	"testing"
)

func TestCrearContratoExcluyeOcasionales(t *testing.T) {
	// 1. Cargar variables de entorno para conectarse a la BD de pruebas
	_ = godotenv.Load("../../.env")

	db, err := config.InitDB()
	if err != nil {
		t.Skip("Saltando prueba de integración: Base de datos no disponible o .env faltante")
		return
	}
	defer db.Close()

	// 2. Limpieza de base de datos previa
	limpiarBaseDeDatos := func() {
		// Borrar tenant de prueba 'Test Tenant Contrato' que cascada a todas las demás
		_, _ = db.Exec("DELETE FROM tenants WHERE ruc = '12345678901'")
		_, _ = db.Exec("DELETE FROM conceptos_maestros WHERE codigo IN ('9991', '9992')")
	}
	limpiarBaseDeDatos()
	defer limpiarBaseDeDatos()

	// 3. Crear fixtures
	// A. Tenant de prueba
	var tenantID int
	err = db.QueryRow("INSERT INTO tenants (nombre, ruc, activo) VALUES ('Test Tenant Contrato', '12345678901', true) RETURNING id").Scan(&tenantID)
	if err != nil {
		t.Fatalf("Error creando tenant de prueba: %v", err)
	}

	// B. Regimen laboral DL 276
	var regimenID int
	err = db.QueryRow("SELECT id FROM regimenes_laborales WHERE codigo = '276'").Scan(&regimenID)
	if err != nil {
		t.Fatalf("Error obteniendo regimen DL 276: %v", err)
	}

	// C. Conceptos Maestros
	var maestroRegularID int
	err = db.QueryRow("INSERT INTO conceptos_maestros (codigo, codigo_interno, descripcion, tipo, activo) VALUES ('9991', '9991', 'Maestro Regular', 'Ingreso', true) RETURNING id").Scan(&maestroRegularID)
	if err != nil {
		t.Fatalf("Error creando concepto maestro regular: %v", err)
	}

	var maestroOcasionalID int
	err = db.QueryRow("INSERT INTO conceptos_maestros (codigo, codigo_interno, descripcion, tipo, activo) VALUES ('9992', '9992', 'Maestro Ocasional', 'Ingreso', true) RETURNING id").Scan(&maestroOcasionalID)
	if err != nil {
		t.Fatalf("Error creando concepto maestro ocasional: %v", err)
	}

	// D. Conceptos Locales (Tenant)
	var conceptoRegularID int
	err = db.QueryRow(`
		INSERT INTO conceptos_tenant (tenant_id, concepto_id, nombre_personalizado, activo, es_ocasional, es_pensionable, es_remunerativa)
		VALUES ($1, $2, 'Test Contrato Regular', true, false, false, true) RETURNING id`, tenantID, maestroRegularID).Scan(&conceptoRegularID)
	if err != nil {
		t.Fatalf("Error creando concepto local regular: %v", err)
	}

	var conceptoOcasionalID int
	err = db.QueryRow(`
		INSERT INTO conceptos_tenant (tenant_id, concepto_id, nombre_personalizado, activo, es_ocasional, es_pensionable, es_remunerativa)
		VALUES ($1, $2, 'Test Contrato Ocasional', true, true, false, true) RETURNING id`, tenantID, maestroOcasionalID).Scan(&conceptoOcasionalID)
	if err != nil {
		t.Fatalf("Error creando concepto local ocasional: %v", err)
	}

	// E. Relacionar conceptos con regimen en tenant
	_, err = db.Exec("INSERT INTO regimen_concepto_tenant (tenant_id, regimen_id, concepto_tenant_id) VALUES ($1, $2, $3)", tenantID, regimenID, conceptoRegularID)
	if err != nil {
		t.Fatalf("Error asociando concepto regular a regimen: %v", err)
	}

	_, err = db.Exec("INSERT INTO regimen_concepto_tenant (tenant_id, regimen_id, concepto_tenant_id) VALUES ($1, $2, $3)", tenantID, regimenID, conceptoOcasionalID)
	if err != nil {
		t.Fatalf("Error asociando concepto ocasional a regimen: %v", err)
	}

	// F. Trabajador de prueba
	var trabajadorID int
	err = db.QueryRow(`
		INSERT INTO trabajadores (tenant_id, tipo_documento, numero_documento, nombres, apellido_paterno, apellido_materno, fecha_nacimiento, sexo, activo, regimen_pensionario)
		VALUES ($1, 'DNI', '99999999', 'Test Contrato Juan', 'Perez', 'Gomez', '1990-01-01', 'M', true, 'ONP') RETURNING id`, tenantID).Scan(&trabajadorID)
	if err != nil {
		t.Fatalf("Error creando trabajador de prueba: %v", err)
	}

	// G. Puesto de prueba
	var puestoID int
	err = db.QueryRow(`
		INSERT INTO puestos (tenant_id, meta_id, fuente_rubro_id, regimen_id, nombre, sueldo_presupuestado, estado, activo, es_dietario)
		VALUES ($1, NULL, NULL, $2, 'Test Contrato Puesto', 1500.00, 'VACANTE', true, false) RETURNING id`, tenantID, regimenID).Scan(&puestoID)
	if err != nil {
		t.Fatalf("Error creando puesto de prueba: %v", err)
	}

	// 4. Instanciar el servicio
	repoContrato := repository.NewContratoRepository(db)
	repoTrabajador := repository.NewTrabajadorRepository(db)
	repoPuesto := repository.NewPuestoRepository(db)
	service := &ContratoService{
		Repo:           repoContrato,
		RepoTrabajador: repoTrabajador,
		RepoPuesto:     repoPuesto,
	}

	// 5. Ejecutar la creación del contrato
	contrato := &models.Contrato{
		TenantID:     tenantID,
		TrabajadorID: trabajadorID,
		PuestoID:     puestoID,
		FechaInicio:  "2026-06-01",
		Activo:       true,
		TipoContrato: "Nombrado",
	}

	err = service.CrearContrato(contrato)
	if err != nil {
		t.Fatalf("Error al ejecutar CrearContrato: %v", err)
	}

	// 6. Verificar que los conceptos asignados al puesto contengan el regular pero NO el ocasional
	rows, err := db.Query("SELECT concepto_tenant_id FROM puesto_conceptos WHERE puesto_id = $1", puestoID)
	if err != nil {
		t.Fatalf("Error consultando conceptos asignados al puesto: %v", err)
	}
	defer rows.Close()

	tieneRegular := false
	tieneOcasional := false

	for rows.Next() {
		var cid int
		if err := rows.Scan(&cid); err == nil {
			if cid == conceptoRegularID {
				tieneRegular = true
			}
			if cid == conceptoOcasionalID {
				tieneOcasional = true
			}
		}
	}

	if !tieneRegular {
		t.Errorf("El concepto regular ('Test Contrato Regular') debió asignarse al puesto.")
	}

	if tieneOcasional {
		t.Errorf("El concepto ocasional ('Test Contrato Ocasional') NO debió asignarse automáticamente al puesto.")
	}
}
