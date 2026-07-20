//go:build ignore

package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	// 1. Cargar .env
	err := godotenv.Load()
	if err != nil {
		log.Printf("Advertencia: %v", err)
	}

	connStr := os.Getenv("DB_CONNECTION_STRING")
	if connStr == "" {
		log.Fatalf("la variable DB_CONNECTION_STRING no está configurada")
	}

	// 2. Conectar a PostgreSQL
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("error al abrir la base de datos: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("error al conectar: %v", err)
	}

	// 3. Ejecutar las sentencias SQL de la migración
	tx, err := db.Begin()
	if err != nil {
		log.Fatalf("error al iniciar transacción: %v", err)
	}

	queries := []string{
		`ALTER TABLE base_regimen_default DROP CONSTRAINT IF EXISTS chk_variable_calculo_default;`,
		`ALTER TABLE base_regimen_default ADD CONSTRAINT chk_variable_calculo_default CHECK (variable_calculo IN (
			'SUELDO_BASICO', 
			'ASIGNACION_FAMILIAR', 
			'SEXTO_GRATIFICACION', 
			'REMUNERACION_VARIABLE',
			'REMUNERACION_COMPUTABLE',
			'MUC',
			'BET',
			'RETRIBUCION_MENSUAL',
			'VALORIZACION_PRINCIPAL',
			'VALORIZACION_AJUSTADA'
		));`,
		`ALTER TABLE base_regimen_tenant DROP CONSTRAINT IF EXISTS chk_variable_calculo_tenant;`,
		`ALTER TABLE base_regimen_tenant ADD CONSTRAINT chk_variable_calculo_tenant CHECK (variable_calculo IN (
			'SUELDO_BASICO', 
			'ASIGNACION_FAMILIAR', 
			'SEXTO_GRATIFICACION', 
			'REMUNERACION_VARIABLE',
			'REMUNERACION_COMPUTABLE',
			'MUC',
			'BET',
			'RETRIBUCION_MENSUAL',
			'VALORIZACION_PRINCIPAL',
			'VALORIZACION_AJUSTADA'
		));`,
		`INSERT INTO goose_db_version (version_id, is_applied) VALUES (20260717120000, true);`,
	}

	for i, q := range queries {
		log.Printf("Ejecutando consulta %d...", i+1)
		_, err := tx.Exec(q)
		if err != nil {
			tx.Rollback()
			log.Fatalf("error al ejecutar consulta %d: %v", i+1, err)
		}
	}

	err = tx.Commit()
	if err != nil {
		log.Fatalf("error al confirmar transacción: %v", err)
	}

	fmt.Println("🎉 Migración '20260717120000_actualizar_variables_calculo_constraints' ejecutada exitosamente en la base de datos.")
}
