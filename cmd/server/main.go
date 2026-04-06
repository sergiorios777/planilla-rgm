package main

import (
	"html/template"
	"log"
	"net/http"

	"planilla-rgm/internal/config"
	"planilla-rgm/internal/handlers"
	"planilla-rgm/internal/repository"
)

// PageData almacena los datos que pasaremos a la interfaz gráfica
type PageData struct {
	Title string
}

func main() {
	// 1. Inicializar y verificar la base de datos
	db, err := config.InitDB()
	if err != nil {
		log.Fatalf("No se pudo iniciar la base de datos: %v", err)
	}
	// Nos aseguramos de cerrar la conexión si el servidor se detiene
	defer db.Close()

	// Inicializamos el repositorio y el controlador
	tenantRepo := repository.NewTenantRepository(db)
	adminHandler := handlers.AdminHandler{Repo: tenantRepo}
	mefRepo := repository.NewMefRepository(db)
	mefHandler := handlers.MefHandler{Repo: mefRepo}
	conceptoRepo := repository.NewConceptoRepository(db)
	conceptoHandler := handlers.ConceptoHandler{Repo: conceptoRepo}

	// 2. Ruta principal: Renderiza la interfaz de HTMX + Pico.css
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Buscamos el HTML en la nueva ruta relativa
		tmpl, err := template.ParseFiles("ui/templates/layouts/index.html")
		if err != nil {
			// Si hay un error (ej. archivo no encontrado), lo imprimimos en consola
			// y mandamos un Error 500 al navegador para no tener "Empty Response"
			log.Printf("Error cargando plantilla: %v", err)
			http.Error(w, "Error interno: No se encontró la plantilla HTML", http.StatusInternalServerError)
			return
		}

		tmpl.Execute(w, map[string]string{"Title": "Planillas RGM - Panel Admin"})
	})

	// Ruta: HTMX llamará aquí para obtener la tabla de inquilinos
	http.HandleFunc("/admin/inquilinos", adminHandler.ListarInquilinos)

	// Ruta: HTMX llamará aquí al enviar el formulario
	http.HandleFunc("/admin/inquilinos/crear", adminHandler.CrearInquilino)

	http.HandleFunc("/admin/mef", mefHandler.ListarClasificadores)
	http.HandleFunc("/admin/mef/crear", mefHandler.CrearClasificador)
	http.HandleFunc("/admin/mef/importar", mefHandler.ImportarCSV)
	http.HandleFunc("/admin/mef/vincular", mefHandler.VincularJerarquiaManual)

	// Rutas de interfaz (Sirven el marco HTML del módulo)
	http.HandleFunc("/admin/ui/inquilinos", adminHandler.VistaUI)
	http.HandleFunc("/admin/ui/mef", mefHandler.VistaUI)

	// Nuevas rutas de Conceptos Maestros
	http.HandleFunc("/admin/ui/conceptos", conceptoHandler.VistaUI)
	http.HandleFunc("/admin/conceptos/lista", conceptoHandler.ListarConceptos)
	http.HandleFunc("/admin/conceptos/importar", conceptoHandler.ImportarCSV)

	// 4. Iniciar el servidor
	log.Println("🚀 Servidor iniciado en http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
