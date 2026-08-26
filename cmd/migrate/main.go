package main

import (
	"log"
	"os"
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

	log.Println("✅ Tablas de entidades financieras, descuentos y descuento_conceptos migradas exitosamente")
}
