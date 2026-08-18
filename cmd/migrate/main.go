package main

import (
	"log"
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
	}

	for _, q := range queries {
		log.Println("Ejecutando:", q)
		_, err := db.Exec(q)
		if err != nil {
			log.Fatalf("Error ejecutando DDL: %v", err)
		}
	}

	log.Println("✅ Columnas migradas exitosamente en PostgreSQL")
}
