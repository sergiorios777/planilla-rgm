package main

import (
	"log"
	"net/http"

	"planilla-rgm/internal/config"
	"planilla-rgm/internal/routes"
)

func main() {
	// 1. Inicializar y verificar la base de datos
	db, err := config.InitDB()
	if err != nil {
		log.Fatalf("No se pudo iniciar la base de datos: %v", err)
	}
	defer db.Close()

	// 2. Configurar el Enrutador Central (inyectando la DB)
	mux := routes.ConfigurarRutas(db)

	// 3. Iniciar el servidor
	log.Println("🚀 Servidor iniciado en http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux)) // 💡 Pasamos 'mux' en lugar de 'nil'
}
