package main

import (
	"log"
	"planilla-rgm/internal/config"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
	service "planilla-rgm/internal/services"
)

func main() {
	// 1. Conectamos a la BD usando nuestra configuración existente
	db, err := config.InitDB()
	if err != nil {
		log.Fatalf("Error conectando a BD: %v", err)
	}
	defer db.Close()

	// 2. Encriptamos la contraseña "admin123"
	hash, err := service.HashPassword("admin123")
	if err != nil {
		log.Fatalf("Error encriptando contraseña: %v", err)
	}

	// 3. Preparamos el modelo del Súper Administrador
	// TenantID es nil (nulo) porque es el dueño del SaaS, no un inquilino
	admin := models.Usuario{
		TenantID: nil,
		Nombre:   "Súper Administrador",
		Email:    "admin@rgm.com",
		Password: hash,
		Rol:      "super_admin",
	}

	// 4. Guardamos en la base de datos
	repo := repository.NewUsuarioRepository(db)
	err = repo.Crear(&admin)
	if err != nil {
		log.Fatalf("Error creando el administrador (¿Quizás ya existe?): %v", err)
	}

	log.Println("✅ Súper Administrador creado exitosamente (admin@rgm.com / admin123)")
}
