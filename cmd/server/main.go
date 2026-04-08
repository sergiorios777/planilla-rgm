package main

import (
	"html/template"
	"log"
	"net/http"

	"planilla-rgm/internal/config"
	"planilla-rgm/internal/handlers"
	"planilla-rgm/internal/middleware"
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
	parametroRepo := repository.NewParametroRepository(db)
	parametroHandler := handlers.ParametroHandler{Repo: parametroRepo}
	usuarioRepo := repository.NewUsuarioRepository(db)
	authHandler := handlers.AuthHandler{Repo: usuarioRepo}
	usuarioHandler := handlers.UsuarioHandler{
		UserRepo:   usuarioRepo,
		TenantRepo: tenantRepo,
	}
	tenantHandler := handlers.TenantHandler{Repo: tenantRepo}
	trabajadorRepo := repository.NewTrabajadorRepository(db)
	trabajadorHandler := handlers.TrabajadorHandler{Repo: trabajadorRepo}
	contratoRepo := repository.NewContratoRepository(db)
	contratoHandler := handlers.ContratoHandler{
		Repo:           contratoRepo,
		TrabajadorRepo: trabajadorRepo,
	}

	// Servir archivos estáticos (CSS, JS, Imágenes)
	fs := http.FileServer(http.Dir("ui/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

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

	// === RUTAS DE AUTENTICACIÓN ===
	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			authHandler.MostrarLogin(w, r)
		case http.MethodPost:
			authHandler.ProcesarLogin(w, r)
		default:
			// Buena práctica: Si alguien intenta usar PUT o DELETE en esta ruta, lo bloqueamos
			http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		}
	})

	// Ruta para cerrar sesión
	http.HandleFunc("/logout", authHandler.CerrarSesion)

	// === RUTA PRINCIPAL DEL DASHBOARD ===
	// Esta ruta carga el "esqueleto" (menú lateral, cabecera, CSS)
	http.HandleFunc("/admin", middleware.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles("ui/templates/layouts/index.html")
		if err != nil {
			http.Error(w, "Error cargando la vista principal", http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, nil)
	}))

	// Ruta: HTMX llamará aquí para obtener la tabla de inquilinos
	http.HandleFunc("/admin/ui/inquilinos", middleware.RequireAuth(adminHandler.VistaUI))
	http.HandleFunc("/admin/inquilinos", middleware.RequireAuth(adminHandler.ListarInquilinos))
	http.HandleFunc("/admin/inquilinos/crear", middleware.RequireAuth(adminHandler.CrearInquilino))
	http.HandleFunc("/admin/inquilinos/editar_ui", middleware.RequireAuth(adminHandler.EditarUI))
	http.HandleFunc("/admin/inquilinos/actualizar", middleware.RequireAuth(adminHandler.ActualizarInquilino))

	http.HandleFunc("/admin/ui/mef", middleware.RequireAuth(mefHandler.VistaUI))
	http.HandleFunc("/admin/mef", middleware.RequireAuth(mefHandler.ListarClasificadores))
	http.HandleFunc("/admin/mef/crear", middleware.RequireAuth(mefHandler.CrearClasificador))
	http.HandleFunc("/admin/mef/importar", middleware.RequireAuth(mefHandler.ImportarCSV))
	http.HandleFunc("/admin/mef/vincular", middleware.RequireAuth(mefHandler.VincularJerarquiaManual))

	// Rutas de Conceptos Maestros
	http.HandleFunc("/admin/ui/conceptos", middleware.RequireAuth(conceptoHandler.VistaUI))
	http.HandleFunc("/admin/conceptos/lista", middleware.RequireAuth(conceptoHandler.ListarConceptos))
	http.HandleFunc("/admin/conceptos/importar", middleware.RequireAuth(conceptoHandler.ImportarCSV))

	// Rutas de Parámetros Globales
	http.HandleFunc("/admin/ui/parametros", middleware.RequireAuth(parametroHandler.VistaUI))
	http.HandleFunc("/admin/parametros/lista", middleware.RequireAuth(parametroHandler.Listar))
	http.HandleFunc("/admin/parametros/guardar", middleware.RequireAuth(parametroHandler.Guardar))

	// Rutas protegidas de Usuarios
	http.HandleFunc("/admin/ui/usuarios", middleware.RequireAuth(usuarioHandler.VistaUI))
	http.HandleFunc("/admin/usuarios/lista", middleware.RequireAuth(usuarioHandler.Listar))
	http.HandleFunc("/admin/usuarios/crear", middleware.RequireAuth(usuarioHandler.Crear))
	http.HandleFunc("/admin/usuarios/editar_ui", middleware.RequireAuth(usuarioHandler.EditarUI))
	http.HandleFunc("/admin/usuarios/actualizar", middleware.RequireAuth(usuarioHandler.ActualizarUsuario))

	// === RUTA PRINCIPAL DEL INQUILINO (Protegida) ===
	http.HandleFunc("/tenant", middleware.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles("ui/templates/layouts/tenant_index.html")
		if err != nil {
			http.Error(w, "Error cargando la vista principal del inquilino", http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, nil)
	}))

	// Rutas de Inquilino
	http.HandleFunc("/tenant/ui/perfil", middleware.RequireAuth(tenantHandler.PerfilUI))
	http.HandleFunc("/tenant/perfil/actualizar", middleware.RequireAuth(tenantHandler.ActualizarPerfil))

	// Rutas de Trabajadores (Protegidas)
	http.HandleFunc("/tenant/ui/trabajadores", middleware.RequireAuth(trabajadorHandler.VistaUI))
	http.HandleFunc("/tenant/trabajadores/lista", middleware.RequireAuth(trabajadorHandler.Listar))
	http.HandleFunc("/tenant/trabajadores/crear", middleware.RequireAuth(trabajadorHandler.Crear))
	http.HandleFunc("/tenant/trabajadores/editar_ui", middleware.RequireAuth(trabajadorHandler.EditarUI))
	http.HandleFunc("/tenant/trabajadores/actualizar", middleware.RequireAuth(trabajadorHandler.Actualizar))

	// Rutas protegidas de Contratos
	http.HandleFunc("/tenant/ui/contratos", middleware.RequireAuth(contratoHandler.VistaUI))
	http.HandleFunc("/tenant/contratos/lista", middleware.RequireAuth(contratoHandler.Listar))
	http.HandleFunc("/tenant/contratos/crear", middleware.RequireAuth(contratoHandler.Crear))

	// 4. Iniciar el servidor
	log.Println("🚀 Servidor iniciado en http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
