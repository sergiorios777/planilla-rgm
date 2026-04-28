package routes

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"planilla-rgm/internal/handlers"
	"planilla-rgm/internal/middleware"
	"planilla-rgm/internal/repository"
)

func ConfigurarRutas(db *sql.DB) *http.ServeMux {
	mux := http.NewServeMux()

	// 1. Inicialización de Repositorios y Handlers
	// Repositorios
	usuarioRepo := repository.NewUsuarioRepository(db)

	// Handlers
	authHandler := handlers.AuthHandler{Repo: usuarioRepo}

	// 1.5 Archivos Estáticos
	fs := http.FileServer(http.Dir("ui/static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// Ruta principal
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

	// 2. Rutas Públicas (Sin protección)
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
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
	mux.HandleFunc("/logout", authHandler.CerrarSesion)

	// 3. REGISTRO POR ENTORNOS (Para facilitar la revisión)
	registrarRutasAdmin(mux, db)
	registrarRutasTenant(mux, db)

	return mux
}

// --- SECCIÓN ADMINISTRATIVA ---
func registrarRutasAdmin(mux *http.ServeMux, db *sql.DB) {
	adminRepo := repository.NewTenantRepository(db)
	h := handlers.AdminHandler{Repo: adminRepo}

	// Esta ruta carga el "esqueleto" (menú lateral, cabecera, CSS)
	mux.HandleFunc("/admin", middleware.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles("ui/templates/layouts/index.html")
		if err != nil {
			http.Error(w, "Error cargando la vista principal", http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, nil)
	}))

	// Ruta: HTMX llamará aquí para obtener la tabla de inquilinos
	mux.HandleFunc("/admin/ui/inquilinos", middleware.RequireAuth(h.VistaUI))
	mux.HandleFunc("/admin/inquilinos", middleware.RequireAuth(h.ListarInquilinos))
	mux.HandleFunc("/admin/inquilinos/crear", middleware.RequireAuth(h.CrearInquilino))
	mux.HandleFunc("/admin/inquilinos/editar_ui", middleware.RequireAuth(h.EditarUI))
	mux.HandleFunc("/admin/inquilinos/actualizar", middleware.RequireAuth(h.ActualizarInquilino))

	// Rutas del MEF (Clasificadores )
	mefRepo := repository.NewMefRepository(db)
	m := handlers.MefHandler{Repo: mefRepo}
	mux.HandleFunc("/admin/ui/mef", middleware.RequireAuth(m.VistaUI))
	mux.HandleFunc("/admin/mef", middleware.RequireAuth(m.ListarClasificadores))
	mux.HandleFunc("/admin/mef/crear", middleware.RequireAuth(m.CrearClasificador))
	mux.HandleFunc("/admin/mef/importar", middleware.RequireAuth(m.ImportarCSV))
	mux.HandleFunc("/admin/mef/vincular", middleware.RequireAuth(m.VincularJerarquiaManual))

	// Rutas de Conceptos Maestros
	conceptoRepo := repository.NewConceptoRepository(db)
	c := handlers.ConceptoHandler{Repo: conceptoRepo}
	mux.HandleFunc("/admin/ui/conceptos", middleware.RequireAuth(c.VistaUI))
	mux.HandleFunc("/admin/conceptos/lista", middleware.RequireAuth(c.ListarConceptos))
	mux.HandleFunc("/admin/conceptos/importar", middleware.RequireAuth(c.ImportarCSV))

	// Rutas de Parámetros Globales
	parametroRepo := repository.NewParametroRepository(db)
	p := handlers.ParametroHandler{Repo: parametroRepo}
	mux.HandleFunc("/admin/ui/parametros", middleware.RequireAuth(p.VistaUI))
	mux.HandleFunc("/admin/parametros/lista", middleware.RequireAuth(p.Listar))
	mux.HandleFunc("/admin/parametros/guardar", middleware.RequireAuth(p.Guardar))

	// Rutas protegidas de Usuarios
	usuarioRepo := repository.NewUsuarioRepository(db)
	tenantRepo := repository.NewTenantRepository(db)
	u := handlers.UsuarioHandler{UserRepo: usuarioRepo, TenantRepo: tenantRepo}
	mux.HandleFunc("/admin/ui/usuarios", middleware.RequireAuth(u.VistaUI))
	mux.HandleFunc("/admin/usuarios/lista", middleware.RequireAuth(u.Listar))
	mux.HandleFunc("/admin/usuarios/crear", middleware.RequireAuth(u.Crear))
	mux.HandleFunc("/admin/usuarios/editar_ui", middleware.RequireAuth(u.EditarUI))
	mux.HandleFunc("/admin/usuarios/actualizar", middleware.RequireAuth(u.ActualizarUsuario))

	// Rutas de Fuentes y Rubros
	fuenteRubroRepo := repository.NewFuenteRubroRepository(db)
	f := handlers.FuenteRubroHandler{Repo: fuenteRubroRepo}
	mux.HandleFunc("/admin/ui/fuentes-rubros", middleware.RequireAuth(f.VistaUI))
	mux.HandleFunc("/admin/fuentes-rubros/lista", middleware.RequireAuth(f.Listar))
}

// --- SECCIÓN TENANT (MUNICIPALIDADES) ---
func registrarRutasTenant(mux *http.ServeMux, db *sql.DB) {
	// Ruta al esqueleto de tenant
	mux.HandleFunc("/tenant", middleware.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles("ui/templates/layouts/tenant_index.html")
		if err != nil {
			http.Error(w, "Error cargando la vista principal del inquilino", http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, nil)
	}))

	// Rutas de Inquilino
	tenantRepo := repository.NewTenantRepository(db)
	tenantHandler := handlers.TenantHandler{Repo: tenantRepo}
	mux.HandleFunc("/tenant/ui/perfil", middleware.RequireAuth(tenantHandler.PerfilUI))
	mux.HandleFunc("/tenant/perfil/actualizar", middleware.RequireAuth(tenantHandler.ActualizarPerfil))

	// Rutas de Trabajadores (Protegidas)
	trabajadorRepo := repository.NewTrabajadorRepository(db)
	trabajadorHandler := handlers.TrabajadorHandler{Repo: trabajadorRepo}
	mux.HandleFunc("/tenant/ui/trabajadores", middleware.RequireAuth(trabajadorHandler.VistaUI))
	mux.HandleFunc("/tenant/trabajadores/lista", middleware.RequireAuth(trabajadorHandler.Listar))
	mux.HandleFunc("/tenant/trabajadores/crear", middleware.RequireAuth(trabajadorHandler.Crear))
	mux.HandleFunc("/tenant/trabajadores/editar_ui", middleware.RequireAuth(trabajadorHandler.EditarUI))
	mux.HandleFunc("/tenant/trabajadores/actualizar", middleware.RequireAuth(trabajadorHandler.Actualizar))

	// Rutas de Metas
	metaRepo := repository.NewMetaRepository(db)
	metaHandler := handlers.MetaHandler{Repo: metaRepo}
	mux.HandleFunc("/tenant/ui/metas", middleware.RequireAuth(metaHandler.VistaUI))
	mux.HandleFunc("/tenant/metas/lista", middleware.RequireAuth(metaHandler.Listar))
	mux.HandleFunc("/tenant/metas/crear", middleware.RequireAuth(metaHandler.Crear))

	// Rutas de Puestos
	puestoRepo := repository.NewPuestoRepository(db)
	fuenteRubroRepo := repository.NewFuenteRubroRepository(db)
	puestoHandler := handlers.PuestoHandler{Repo: puestoRepo, MetaRepo: metaRepo, FuenteRubroRepo: fuenteRubroRepo}
	mux.HandleFunc("/tenant/ui/puestos", middleware.RequireAuth(puestoHandler.VistaUI))
	mux.HandleFunc("/tenant/puestos/lista", middleware.RequireAuth(puestoHandler.Listar))
	mux.HandleFunc("/tenant/puestos/crear", middleware.RequireAuth(puestoHandler.Crear))
	mux.HandleFunc("/tenant/puestos/editar", middleware.RequireAuth(puestoHandler.Editar))
	mux.HandleFunc("/tenant/puestos/actualizar", middleware.RequireAuth(puestoHandler.Actualizar))

	// Rutas protegidas de Contratos
	contratoRepo := repository.NewContratoRepository(db)
	contratoHandler := handlers.ContratoHandler{Repo: contratoRepo, TrabajadorRepo: trabajadorRepo, PuestoRepo: puestoRepo}
	mux.HandleFunc("/tenant/ui/contratos", middleware.RequireAuth(contratoHandler.VistaUI))
	mux.HandleFunc("/tenant/contratos/lista", middleware.RequireAuth(contratoHandler.Listar))
	mux.HandleFunc("/tenant/contratos/crear", middleware.RequireAuth(contratoHandler.Crear))

	// Rutas protegidas (Bajo la sección de Inquilinos/Presupuesto)
	conceptoTenantRepo := repository.NewConceptoTenantRepository(db)
	conceptoTenantHandler := handlers.ConceptoTenantHandler{Repo: conceptoTenantRepo}
	mux.HandleFunc("/tenant/ui/conceptos-locales", middleware.RequireAuth(conceptoTenantHandler.VistaUI))
	mux.HandleFunc("/tenant/conceptos-locales/lista", middleware.RequireAuth(conceptoTenantHandler.Listar))
	mux.HandleFunc("/tenant/conceptos-locales/crear", middleware.RequireAuth(conceptoTenantHandler.Crear))

	// Rutas protegidas (Agrega esto junto a las de Puestos)
	puestoConceptoRepo := repository.NewPuestoConceptoRepository(db)
	puestoConceptoHandler := handlers.PuestoConceptoHandler{Repo: puestoConceptoRepo, PuestoRepo: puestoRepo}
	mux.HandleFunc("/tenant/puestos-conceptos/ui", middleware.RequireAuth(puestoConceptoHandler.VistaUI))
	mux.HandleFunc("/tenant/puestos-conceptos/lista", middleware.RequireAuth(puestoConceptoHandler.Listar))
	mux.HandleFunc("/tenant/puestos-conceptos/crear", middleware.RequireAuth(puestoConceptoHandler.Crear))
	mux.HandleFunc("/tenant/puestos-conceptos/eliminar", middleware.RequireAuth(puestoConceptoHandler.Eliminar))
	mux.HandleFunc("/tenant/puestos-conceptos/restaurar", middleware.RequireAuth(puestoConceptoHandler.RestaurarCostosBase))
	mux.HandleFunc("/tenant/puestos-conceptos/editar-monto-ui", middleware.RequireAuth(puestoConceptoHandler.EditarMontoUI))
	mux.HandleFunc("/tenant/puestos-conceptos/actualizar-monto", middleware.RequireAuth(puestoConceptoHandler.ActualizarMonto))

	// Rutas protegidas (Bajo una nueva sección de Procesamiento)
	planillaRepo := repository.NewPlanillaRepository(db)
	planillaHandler := handlers.PlanillaHandler{Repo: planillaRepo}
	mux.HandleFunc("/tenant/ui/planillas", middleware.RequireAuth(planillaHandler.VistaUI))
	mux.HandleFunc("/tenant/planillas/lista", middleware.RequireAuth(planillaHandler.Listar))
	mux.HandleFunc("/tenant/planillas/crear", middleware.RequireAuth(planillaHandler.Crear))
	mux.HandleFunc("/tenant/planillas/procesar", middleware.RequireAuth(planillaHandler.Procesar))
	mux.HandleFunc("/tenant/planillas/detalle/ui", middleware.RequireAuth(planillaHandler.VistaDetalle))

	// Asistencias
	asistenciaRepo := repository.NewAsistenciaRepository(db)
	asistenciaHandler := handlers.AsistenciaHandler{Repo: asistenciaRepo}
	mux.HandleFunc("/tenant/ui/asistencia", middleware.RequireAuth(asistenciaHandler.VistaUI))
	mux.HandleFunc("/tenant/asistencia/lista", middleware.RequireAuth(asistenciaHandler.Listar))
	mux.HandleFunc("/tenant/asistencia/crear", middleware.RequireAuth(asistenciaHandler.Crear))

	// Ruta para descargar el PDF de planilla y boeltas
	mux.HandleFunc("/tenant/planillas/descargar-reporte", middleware.RequireAuth(planillaHandler.DescargarReportePDF))
	mux.HandleFunc("/tenant/planillas/descargar-boletas", middleware.RequireAuth(planillaHandler.DescargarBoletasPDF))
}
