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
	err := godotenv.Load()
	if err != nil {
		log.Printf("Advertencia: %v", err)
	}

	connStr := os.Getenv("DB_CONNECTION_STRING")
	if connStr == "" {
		log.Fatalf("la variable DB_CONNECTION_STRING no está configurada")
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("error al abrir la base de datos: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("error al conectar: %v", err)
	}

	queries := []string{
		`ALTER TABLE trabajadores ADD COLUMN IF NOT EXISTS banco VARCHAR(100);`,
		`ALTER TABLE trabajadores ADD COLUMN IF NOT EXISTS cuenta VARCHAR(50);`,
		`ALTER TABLE trabajadores ADD COLUMN IF NOT EXISTS cci VARCHAR(50);`,
	}

	for i, q := range queries {
		log.Printf("Ejecutando consulta %d...", i+1)
		_, err := db.Exec(q)
		if err != nil {
			log.Fatalf("error al ejecutar consulta %d: %v", i+1, err)
		}
	}

	fmt.Println("🎉 Migración banco, cuenta, cci aplicada correctamente.")
}
