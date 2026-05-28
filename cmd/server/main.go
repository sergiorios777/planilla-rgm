package main

import (
	"log"
	"net/http"
	"time"

	"planilla-rgm/internal/config"
	"planilla-rgm/internal/repository"
	"planilla-rgm/internal/routes"
	"planilla-rgm/internal/services"
)

func main() {
	// 1. Inicializar y verificar la base de datos
	db, err := config.InitDB()
	if err != nil {
		log.Fatalf("No se pudo iniciar la base de datos: %v", err)
	}
	defer db.Close()

	// Iniciar Daemon de Tareas en segundo plano (Frecuencia: 5 minutos)
	mailService := services.NewMailService()
	tareaRepo := repository.NewAdminTareaRepository(db)
	notifRepo := repository.NewNotificacionRepository(db)
	observador := services.NewTareaObservadorService(tareaRepo, notifRepo, mailService)
	observador.Iniciar(5 * time.Minute)

	// 2. Configurar el Enrutador Central (inyectando la DB)
	mux := routes.ConfigurarRutas(db)

	// 3. Iniciar el servidor
	log.Println("🚀 Servidor iniciado en http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux)) // 💡 Pasamos 'mux' en lugar de 'nil'
}
