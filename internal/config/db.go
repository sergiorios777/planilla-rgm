package config

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// InitDB abre la conexión con la base de datos y verifica que funcione
func InitDB() (*sql.DB, error) {
	// 1. Cargar las variables desde el archivo .env
	err := godotenv.Load()
	if err != nil {
		log.Println("Advertencia: No se encontró el archivo .env. Asegúrate de haberlo creado en la raíz del proyecto.")
	}

	// 2. Leer la cadena de conexión desde las variables de entorno
	connStr := os.Getenv("DB_CONNECTION_STRING")
	if connStr == "" {
		return nil, fmt.Errorf("la variable DB_CONNECTION_STRING no está configurada")
	}

	// 3. Abrir la conexión con PostgreSQL usando la variable segura
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("error al abrir la base de datos: %v", err)
	}

	// 4. Verificar que la conexión es exitosa
	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("error al conectar con la base de datos: %v", err)
	}

	log.Println("✅ Conexión exitosa a la base de datos planilla_rgm")
	return db, nil
}
