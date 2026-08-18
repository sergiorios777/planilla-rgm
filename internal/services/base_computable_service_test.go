package services

import (
	"testing"
	"time"

	"github.com/joho/godotenv"
	"planilla-rgm/internal/config"
)

func TestBaseComputableService(t *testing.T) {
	_ = godotenv.Load("../../.env")
	db, err := config.InitDB()
	if err != nil {
		t.Skip("Saltando test de base de datos local:", err)
		return
	}

	// 1. Crear Tenant tipo GOBIERNO_LOCAL
	var tenantID int
	err = db.QueryRow("INSERT INTO tenants (nombre, ruc, activo, tipo_entidad) VALUES ('Muni Base Computable Test', '20123456789', true, 'GOBIERNO_LOCAL') RETURNING id").Scan(&tenantID)
	if err != nil {
		t.Fatalf("Error creando tenant: %v", err)
	}
	defer func() {
		db.Exec("DELETE FROM tenants WHERE id = $1", tenantID)
	}()

	svc := NewBaseComputableService(db)

	// 2. Test para Régimen D.L. 276 en Gobierno Local (Ley 32199 / D.S. 420-2019-EF)
	t.Run("DL 276 Gobierno Local - Conceptos Permanentes", func(t *testing.T) {
		var regimen276ID int
		err = db.QueryRow("SELECT id FROM regimenes_laborales WHERE codigo = '276'").Scan(&regimen276ID)
		if err != nil {
			t.Fatalf("Error obteniendo régimen 276: %v", err)
		}

		// Crear puesto 276
		var puestoID int
		err = db.QueryRow(`
			INSERT INTO puestos (tenant_id, regimen_id, nombre, estado, sueldo_presupuestado)
			VALUES ($1, $2, 'Especialista Administrativo 276', 'VACANTE', 2000.00) RETURNING id`, tenantID, regimen276ID).Scan(&puestoID)
		if err != nil {
			t.Fatalf("Error creando puesto 276: %v", err)
		}

		// Crear concepto maestro y conceptos tenant permanentes
		var maestroID int
		err = db.QueryRow("SELECT id FROM conceptos_maestros WHERE codigo_interno = '2001' OR codigo = '2001' LIMIT 1").Scan(&maestroID)
		if err != nil {
			err = db.QueryRow("INSERT INTO conceptos_maestros (codigo, descripcion, tipo, codigo_interno, origen) VALUES ('2001', 'Haber Básico 276', 'Ingreso', '2001', 'interno') RETURNING id").Scan(&maestroID)
		}

		var conceptoTenant1, conceptoTenant2 int
		_ = db.QueryRow(`
			INSERT INTO conceptos_tenant (tenant_id, concepto_id, nombre_personalizado, modalidad_entrega, activo)
			VALUES ($1, $2, 'Haber Básico', 'PERMANENTE', true) RETURNING id`, tenantID, maestroID).Scan(&conceptoTenant1)

		_ = db.QueryRow(`
			INSERT INTO conceptos_tenant (tenant_id, concepto_id, nombre_personalizado, modalidad_entrega, activo)
			VALUES ($1, $2, 'Costo de Vida Municipal', 'PERMANENTE', true) RETURNING id`, tenantID, maestroID).Scan(&conceptoTenant2)

		// Concepto Ocasional (No debe sumar a la base permanente)
		var conceptoOcasional int
		_ = db.QueryRow(`
			INSERT INTO conceptos_tenant (tenant_id, concepto_id, nombre_personalizado, modalidad_entrega, activo)
			VALUES ($1, $2, 'Bono Ocasional Cierre Pliego', 'OCASIONAL', true) RETURNING id`, tenantID, maestroID).Scan(&conceptoOcasional)

		// Asignar al puesto
		_, _ = db.Exec("INSERT INTO puesto_conceptos (puesto_id, concepto_tenant_id, monto, activo) VALUES ($1, $2, 1200.00, true)", puestoID, conceptoTenant1)
		_, _ = db.Exec("INSERT INTO puesto_conceptos (puesto_id, concepto_tenant_id, monto, activo) VALUES ($1, $2, 800.00, true)", puestoID, conceptoTenant2)
		_, _ = db.Exec("INSERT INTO puesto_conceptos (puesto_id, concepto_tenant_id, monto, activo) VALUES ($1, $2, 500.00, true)", puestoID, conceptoOcasional)

		// Resolver base para CTS
		fechaCorte := time.Date(2026, time.August, 15, 0, 0, 0, 0, time.UTC)
		desglose, err := svc.ResolverBaseComputable(tenantID, 0, puestoID, regimen276ID, "276", BeneficioCTS, fechaCorte)
		if err != nil {
			t.Fatalf("Error resolviendo base CTS 276: %v", err)
		}

		// En Gobiernos Locales debe sumar 1200 + 800 = 2000 (excluyendo los 500 del ocasional)
		esperado := 2000.00
		if desglose.TotalComputable != esperado {
			t.Errorf("Esperado base CTS 276 municipal = %.2f, obtenido = %.2f", esperado, desglose.TotalComputable)
		}
	})

	// 3. Test para Régimen D.L. 728 (Diferenciación entre CTS y Vacaciones)
	t.Run("DL 728 - Diferenciacion CTS vs Vacaciones", func(t *testing.T) {
		var regimen728ID int
		err = db.QueryRow("SELECT id FROM regimenes_laborales WHERE codigo = '728'").Scan(&regimen728ID)
		if err != nil {
			t.Fatalf("Error obteniendo régimen 728: %v", err)
		}

		var puesto728ID int
		err = db.QueryRow(`
			INSERT INTO puestos (tenant_id, regimen_id, nombre, estado, sueldo_presupuestado)
			VALUES ($1, $2, 'Operador 728', 'VACANTE', 3000.00) RETURNING id`, tenantID, regimen728ID).Scan(&puesto728ID)
		if err != nil {
			t.Fatalf("Error creando puesto 728: %v", err)
		}

		fechaCorte := time.Date(2026, time.August, 15, 0, 0, 0, 0, time.UTC)

		// Resolver para CTS (debe incorporar 1/6 de gratificación)
		desgloseCTS, err := svc.ResolverBaseComputable(tenantID, 0, puesto728ID, regimen728ID, "728", BeneficioCTS, fechaCorte)
		if err != nil {
			t.Fatalf("Error resolviendo base CTS 728: %v", err)
		}

		// Resolver para Vacaciones (NO debe incorporar 1/6 de gratificación)
		desgloseVac, err := svc.ResolverBaseComputable(tenantID, 0, puesto728ID, regimen728ID, "728", BeneficioVacaciones, fechaCorte)
		if err != nil {
			t.Fatalf("Error resolviendo base Vacaciones 728: %v", err)
		}

		if desgloseCTS.TotalComputable <= desgloseVac.TotalComputable {
			t.Errorf("La base de CTS (%.2f) debe ser mayor que la de Vacaciones (%.2f) por la incorporación del 1/6 de gratificación",
				desgloseCTS.TotalComputable, desgloseVac.TotalComputable)
		}

		if desgloseVac.SextoGrati != 0 {
			t.Errorf("La base de vacaciones no debe contener sexto de gratificación, obtenido = %.2f", desgloseVac.SextoGrati)
		}
	})

	// 4. Test para Régimen D.L. 1057 (CAS)
	t.Run("DL 1057 CAS - Retribucion Mensual", func(t *testing.T) {
		var regimenCASID int
		err = db.QueryRow("SELECT id FROM regimenes_laborales WHERE codigo IN ('1057', 'CAS') LIMIT 1").Scan(&regimenCASID)
		if err != nil {
			t.Fatalf("Error obteniendo régimen CAS: %v", err)
		}

		var puestoCASID int
		err = db.QueryRow(`
			INSERT INTO puestos (tenant_id, regimen_id, nombre, estado, sueldo_presupuestado)
			VALUES ($1, $2, 'Consultor CAS', 'VACANTE', 4500.00) RETURNING id`, tenantID, regimenCASID).Scan(&puestoCASID)
		if err != nil {
			t.Fatalf("Error creando puesto CAS: %v", err)
		}

		fechaCorte := time.Date(2026, time.August, 15, 0, 0, 0, 0, time.UTC)
		desgloseCAS, err := svc.ResolverBaseComputable(tenantID, 0, puestoCASID, regimenCASID, "1057", BeneficioCTS, fechaCorte)
		if err != nil {
			t.Fatalf("Error resolviendo base CAS: %v", err)
		}

		if desgloseCAS.TotalComputable != 4500.00 {
			t.Errorf("Esperado 4500.00 para CAS, obtenido = %.2f", desgloseCAS.TotalComputable)
		}
	})
}
