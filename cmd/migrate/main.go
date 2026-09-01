package main

import (
	"log"
	"os"
	"strings"
	"planilla-rgm/internal/config"
)

func main() {
	db, err := config.InitDB()
	if err != nil {
		log.Fatalf("Error conectando a BD: %v", err)
	}
	defer db.Close()

	queries := []string{
		"ALTER TABLE public.tenants ADD COLUMN IF NOT EXISTS tipo_entidad VARCHAR(50) DEFAULT 'GOBIERNO_LOCAL' NOT NULL;",
		"ALTER TABLE public.conceptos_modelo ADD COLUMN IF NOT EXISTS base_calculo_para TEXT[] DEFAULT '{}';",
		"ALTER TABLE public.conceptos_tenant ADD COLUMN IF NOT EXISTS base_calculo_para TEXT[] DEFAULT '{}';",
		"ALTER TABLE public.planillas ADD COLUMN IF NOT EXISTS tipo VARCHAR(30) DEFAULT 'ORDINARIA' NOT NULL;",
		"UPDATE public.planillas SET tipo = 'EXTRAORDINARIA' WHERE es_extraordinaria = true;",
		"UPDATE public.planillas SET tipo = 'ORDINARIA' WHERE (es_extraordinaria = false OR es_extraordinaria IS NULL) AND tipo NOT IN ('EXTRAORDINARIA', 'CTS', 'CESE');",
		"ALTER TABLE public.planillas_cts ADD COLUMN IF NOT EXISTS planilla_id INT REFERENCES public.planillas(id) ON DELETE CASCADE;",
		"CREATE INDEX IF NOT EXISTS idx_planillas_tenant_tipo ON public.planillas(tenant_id, tipo, anio, mes);",
		"CREATE INDEX IF NOT EXISTS idx_planillas_cts_planilla_id ON public.planillas_cts(planilla_id);",
	}

	for _, q := range queries {
		log.Println("Ejecutando:", q)
		_, err := db.Exec(q)
		if err != nil {
			log.Fatalf("Error ejecutando DDL: %v", err)
		}
	}

	// Ejecutar migración del módulo de descuentos y retenciones
	sqlBytes, err := os.ReadFile("internal/repository/migrations/20260825000000_modulo_descuentos_retenciones.sql")
	if err != nil {
		log.Fatalf("Error leyendo archivo de migración de descuentos: %v", err)
	}
	_, err = db.Exec(string(sqlBytes))
	if err != nil {
		log.Fatalf("Error ejecutando migración de descuentos: %v", err)
	}

	cleanUpMigration := func(content string) string {
		parts := strings.Split(content, "-- +goose Down")
		return parts[0]
	}

	// Ejecutar migración de Tabla 21 de SUNAT (suspensiones)
	sqlBytesT21, err := os.ReadFile("internal/repository/migrations/20260830120000_crear_tabla_sunat_21_suspensiones.sql")
	if err != nil {
		log.Fatalf("Error leyendo archivo de migración tabla 21: %v", err)
	}
	_, err = db.Exec(cleanUpMigration(string(sqlBytesT21)))
	if err != nil {
		log.Fatalf("Error ejecutando migración tabla 21: %v", err)
	}
	log.Println("✅ Tabla 21 SUNAT (sunat_tipos_suspension) migrada y poblada exitosamente")

	// Ejecutar migración de vacaciones y licencias
	sqlBytesVac, err := os.ReadFile("internal/repository/migrations/20260830121000_crear_personal_licencias_vacaciones.sql")
	if err != nil {
		log.Fatalf("Error leyendo archivo de migración licencias/vacaciones: %v", err)
	}
	_, err = db.Exec(cleanUpMigration(string(sqlBytesVac)))
	if err != nil {
		log.Fatalf("Error ejecutando migración licencias/vacaciones: %v", err)
	}
	log.Println("✅ Tabla personal_licencias_vacaciones migrada exitosamente")
}
