package routes

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"planilla-rgm/internal/handlers"
	"planilla-rgm/internal/middleware"
	"planilla-rgm/internal/repository"
	"planilla-rgm/internal/services"
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
	conceptoTenantRepo := repository.NewConceptoTenantRepository(db)
	h := handlers.AdminHandler{Repo: adminRepo, ConceptoTenantRepo: conceptoTenantRepo}

	// 🔒 Reemplazamos RequireAuth por RequireRole("super_admin")
	mux.HandleFunc("/admin", middleware.RequireRole("super_admin", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles("ui/templates/layouts/index.html", "ui/templates/layouts/iconos_sprite.html")
		if err != nil {
			http.Error(w, "Error cargando la vista principal", http.StatusInternalServerError)
			return
		}

		tmpl.Execute(w, nil)
	}))

	// Ruta: HTMX llamará aquí para obtener la tabla de inquilinos
	mux.HandleFunc("/admin/ui/inquilinos", middleware.RequireRole("super_admin", h.VistaUI))
	mux.HandleFunc("/admin/inquilinos", middleware.RequireRole("super_admin", h.ListarInquilinos))
	mux.HandleFunc("/admin/inquilinos/crear", middleware.RequireRole("super_admin", h.CrearInquilino))
	mux.HandleFunc("/admin/inquilinos/editar_ui", middleware.RequireRole("super_admin", h.EditarUI))
	mux.HandleFunc("/admin/inquilinos/actualizar", middleware.RequireRole("super_admin", h.ActualizarInquilino))

	// Rutas del MEF (Clasificadores )
	mefRepo := repository.NewMefRepository(db)
	m := handlers.MefHandler{Repo: mefRepo}
	mux.HandleFunc("/admin/ui/mef", middleware.RequireRole("super_admin", m.VistaUI))
	mux.HandleFunc("/admin/mef", middleware.RequireRole("super_admin", m.ListarClasificadores))
	mux.HandleFunc("/admin/mef/crear", middleware.RequireRole("super_admin", m.CrearClasificador))
	mux.HandleFunc("/admin/mef/importar", middleware.RequireRole("super_admin", m.ImportarCSV))
	mux.HandleFunc("/admin/mef/vincular", middleware.RequireRole("super_admin", m.VincularJerarquiaManual))

	// Rutas de Conceptos Maestros
	conceptoRepo := repository.NewConceptoRepository(db)
	c := handlers.ConceptoHandler{Repo: conceptoRepo}
	mux.HandleFunc("/admin/ui/conceptos", middleware.RequireRole("super_admin", c.VistaUI))
	mux.HandleFunc("/admin/conceptos/lista", middleware.RequireRole("super_admin", c.ListarConceptos))
	mux.HandleFunc("/admin/conceptos/importar", middleware.RequireRole("super_admin", c.ImportarCSV))

	// Rutas de Parámetros Globales
	parametroRepo := repository.NewParametroRepository(db)
	p := handlers.ParametroHandler{Repo: parametroRepo}
	mux.HandleFunc("/admin/ui/parametros", middleware.RequireRole("super_admin", p.VistaUI))
	mux.HandleFunc("/admin/parametros/lista", middleware.RequireRole("super_admin", p.Listar))
	mux.HandleFunc("/admin/parametros/guardar", middleware.RequireRole("super_admin", p.Guardar))
	mux.HandleFunc("/admin/parametros/editar_ui", middleware.RequireRole("super_admin", p.EditarUI))
	mux.HandleFunc("/admin/parametros/actualizar", middleware.RequireRole("super_admin", p.ActualizarParametro))

	// Rutas protegidas de Usuarios
	usuarioRepo := repository.NewUsuarioRepository(db)
	tenantRepo := repository.NewTenantRepository(db)
	u := handlers.UsuarioHandler{UserRepo: usuarioRepo, TenantRepo: tenantRepo}
	mux.HandleFunc("/admin/ui/usuarios", middleware.RequireRole("super_admin", u.VistaUI))
	mux.HandleFunc("/admin/usuarios/lista", middleware.RequireRole("super_admin", u.Listar))
	mux.HandleFunc("/admin/usuarios/crear", middleware.RequireRole("super_admin", u.Crear))
	mux.HandleFunc("/admin/usuarios/editar_ui", middleware.RequireRole("super_admin", u.EditarUI))
	mux.HandleFunc("/admin/usuarios/actualizar", middleware.RequireRole("super_admin", u.ActualizarUsuario))

	// Rutas de Fuentes y Rubros
	fuenteRubroRepo := repository.NewFuenteRubroRepository(db)
	f := handlers.FuenteRubroHandler{Repo: fuenteRubroRepo}
	mux.HandleFunc("/admin/ui/fuentes-rubros", middleware.RequireRole("super_admin", f.VistaUI))
	mux.HandleFunc("/admin/fuentes-rubros/lista", middleware.RequireRole("super_admin", f.Listar))

	// Rutas de Conceptos Modelo (Plantillas)
	conceptosModeloRepo := repository.NewConceptoModeloRepository(db)
	puestoRepo := repository.NewPuestoRepository(db)
	conceptosTenantRepo := repository.NewConceptoTenantRepository(db)
	cm := handlers.ConceptoModeloHandler{
		Repo:               conceptosModeloRepo,
		PuestoRepo:         puestoRepo,
		ConceptoTenantRepo: conceptosTenantRepo,
		TenantRepo:         tenantRepo,
	}
	mux.HandleFunc("/admin/ui/conceptos-modelo", middleware.RequireRole("super_admin", cm.VistaUI))
	mux.HandleFunc("/admin/conceptos-modelo/lista", middleware.RequireRole("super_admin", cm.Listar))
	mux.HandleFunc("/admin/conceptos-modelo/crear", middleware.RequireRole("super_admin", cm.Crear))
	mux.HandleFunc("/admin/conceptos-modelo/editar_ui", middleware.RequireRole("super_admin", cm.EditarUI))
	mux.HandleFunc("/admin/conceptos-modelo/actualizar", middleware.RequireRole("super_admin", cm.Actualizar))
	mux.HandleFunc("/admin/conceptos-modelo/eliminar", middleware.RequireRole("super_admin", cm.Eliminar))
	mux.HandleFunc("/admin/conceptos-modelo/sincronizar", middleware.RequireRole("super_admin", cm.Sincronizar))
}

// --- SECCIÓN TENANT (MUNICIPALIDADES) ---
func registrarRutasTenant(mux *http.ServeMux, db *sql.DB) {
	// Ruta al esqueleto de tenant
	mux.HandleFunc("/tenant", middleware.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		// 1. Obtener IDs de la sesión
		var uID, tID int
		if val, ok := r.Context().Value(middleware.UsuarioIDKey).(float64); ok {
			uID = int(val)
		}
		if val, ok := r.Context().Value("tenant_id").(float64); ok {
			tID = int(val)
		}

		// 2. Cargar datos de la BD
		usuarioRepo := repository.NewUsuarioRepository(db)
		tenantRepo := repository.NewTenantRepository(db)

		var tenantNombre string
		if tID > 0 {
			t, err := tenantRepo.ObtenerPorID(tID)
			if err == nil && t != nil {
				tenantNombre = t.Nombre
			}
		}

		var usuarioNombre, usuarioRol string
		if uID > 0 {
			u, err := usuarioRepo.ObtenerPorID(uID)
			if err == nil && u != nil {
				usuarioNombre = u.Nombre
				// Traducir rol de forma amigable
				usuarioRol = u.Rol
				if u.Rol == "tenant_admin" {
					usuarioRol = "Administrador"
				} else if u.Rol == "tenant_operator" {
					usuarioRol = "Operador"
				} else if u.Rol == "super_admin" {
					usuarioRol = "Súper Admin"
				}
			}
		}

		tmpl, err := template.ParseFiles("ui/templates/layouts/tenant_index.html", "ui/templates/layouts/iconos_sprite.html")
		if err != nil {
			http.Error(w, "Error cargando la vista principal del inquilino", http.StatusInternalServerError)
			return
		}

		data := map[string]interface{}{
			"TenantNombre":  tenantNombre,
			"UsuarioNombre": usuarioNombre,
			"UsuarioRol":    usuarioRol,
		}
		tmpl.Execute(w, data)
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
	mux.HandleFunc("/tenant/trabajadores/plantilla", middleware.RequireAuth(trabajadorHandler.DescargarPlantilla))
	mux.HandleFunc("/tenant/trabajadores/importar", middleware.RequireAuth(trabajadorHandler.ImportarExcel))

	// Rutas de Metas
	metaRepo := repository.NewMetaRepository(db)
	metaHandler := handlers.MetaHandler{Repo: metaRepo}
	mux.HandleFunc("/tenant/ui/metas", middleware.RequireAuth(metaHandler.VistaUI))
	mux.HandleFunc("/tenant/metas/lista", middleware.RequireAuth(metaHandler.Listar))
	mux.HandleFunc("/tenant/metas/crear", middleware.RequireAuth(metaHandler.Crear))
	mux.HandleFunc("/tenant/metas/formulario-crear", middleware.RequireAuth(metaHandler.FormularioCrearUI))
	mux.HandleFunc("/tenant/metas/editar-ui", middleware.RequireAuth(metaHandler.EditarUI))
	mux.HandleFunc("/tenant/metas/actualizar", middleware.RequireAuth(metaHandler.Actualizar))
	mux.HandleFunc("/tenant/metas/importar", middleware.RequireAuth(metaHandler.ImportarExcel))
	mux.HandleFunc("/tenant/metas/plantilla", middleware.RequireAuth(metaHandler.DescargarPlantilla))

	// Rutas de Estructura Orgánica (Organigramas)
	organigramaRepo := repository.NewOrganigramaRepository(db)

	// Rutas de Puestos
	puestoRepo := repository.NewPuestoRepository(db)
	fuenteRubroRepo := repository.NewFuenteRubroRepository(db)
	puestoHandler := handlers.PuestoHandler{
		Repo:            puestoRepo,
		MetaRepo:        metaRepo,
		FuenteRubroRepo: fuenteRubroRepo,
		OrganigramaRepo: organigramaRepo,
	}
	mux.HandleFunc("/tenant/ui/puestos", middleware.RequireAuth(puestoHandler.VistaUI))
	mux.HandleFunc("/tenant/puestos/lista", middleware.RequireAuth(puestoHandler.Listar))
	mux.HandleFunc("/tenant/puestos/crear", middleware.RequireAuth(puestoHandler.Crear))
	mux.HandleFunc("/tenant/puestos/editar", middleware.RequireAuth(puestoHandler.Editar))
	mux.HandleFunc("/tenant/puestos/editar-ui", middleware.RequireAuth(puestoHandler.EditarUI))
	mux.HandleFunc("/tenant/puestos/actualizar", middleware.RequireAuth(puestoHandler.Actualizar))
	mux.HandleFunc("/tenant/puestos/formulario-crear", middleware.RequireAuth(puestoHandler.FormularioCrearUI))
	mux.HandleFunc("/tenant/puestos/asignar-conceptos-ui", middleware.RequireAuth(puestoHandler.AsignarConceptosUI))
	mux.HandleFunc("/tenant/puestos/guardar-asignacion", middleware.RequireAuth(puestoHandler.GuardarAsignacion))
	mux.HandleFunc("/tenant/puestos/plantilla", middleware.RequireAuth(puestoHandler.DescargarPlantilla))
	mux.HandleFunc("/tenant/puestos/importar", middleware.RequireAuth(puestoHandler.ImportarExcel))

	// Rutas protegidas de Contratos
	contratoRepo := repository.NewContratoRepository(db)
	contratoHandler := handlers.ContratoHandler{Repo: contratoRepo, TrabajadorRepo: trabajadorRepo, PuestoRepo: puestoRepo}
	mux.HandleFunc("/tenant/ui/contratos", middleware.RequireAuth(contratoHandler.VistaUI))
	mux.HandleFunc("/tenant/contratos/lista", middleware.RequireAuth(contratoHandler.Listar))
	mux.HandleFunc("/tenant/contratos/crear", middleware.RequireAuth(contratoHandler.Crear))
	mux.HandleFunc("/tenant/contratos/formulario-crear", middleware.RequireAuth(contratoHandler.FormularioCrearUI))
	mux.HandleFunc("/tenant/contratos/formulario-dinamico", middleware.RequireAuth(contratoHandler.FormularioDinamicoUI))
	mux.HandleFunc("/tenant/contratos/editar-ui", middleware.RequireAuth(contratoHandler.EditarUI))
	mux.HandleFunc("/tenant/contratos/actualizar", middleware.RequireAuth(contratoHandler.Actualizar))
	mux.HandleFunc("/tenant/contratos/plantilla", middleware.RequireAuth(contratoHandler.DescargarPlantilla))
	mux.HandleFunc("/tenant/contratos/importar", middleware.RequireAuth(contratoHandler.ImportarExcel))

	organigramaHandler := handlers.OrganigramaHandler{Repo: organigramaRepo, PuestoRepo: puestoRepo}
	mux.HandleFunc("/tenant/ui/organigrama", middleware.RequireAuth(organigramaHandler.VistaUI))
	mux.HandleFunc("/tenant/organigrama/arbol", middleware.RequireAuth(organigramaHandler.ArbolUI))
	mux.HandleFunc("/tenant/organigrama/clonar", middleware.RequireAuth(organigramaHandler.ClonarVersion))
	mux.HandleFunc("/tenant/organigrama/unidad/guardar", middleware.RequireAuth(organigramaHandler.GuardarUnidad))
	mux.HandleFunc("/tenant/organigrama/unidad/eliminar", middleware.RequireAuth(organigramaHandler.EliminarUnidad))
	mux.HandleFunc("/tenant/organigrama/unidad/agregar_hijo_ui", middleware.RequireAuth(organigramaHandler.AgregarHijoUI))
	mux.HandleFunc("/tenant/organigrama/unidad/editar_ui", middleware.RequireAuth(organigramaHandler.EditarUnidadUI))
	mux.HandleFunc("/tenant/organigrama/importar", middleware.RequireAuth(organigramaHandler.ImportarExcel))
	mux.HandleFunc("/tenant/organigrama/plantilla", middleware.RequireAuth(organigramaHandler.DescargarPlantilla))

	// Rutas protegidas (Bajo la sección de Inquilinos/Presupuesto)
	conceptoTenantRepo := repository.NewConceptoTenantRepository(db)
	conceptoTenantHandler := handlers.ConceptoTenantHandler{
		Repo:       conceptoTenantRepo,
		PuestoRepo: puestoRepo,
	}
	mux.HandleFunc("/tenant/ui/conceptos-locales", middleware.RequireAuth(conceptoTenantHandler.VistaUI))
	mux.HandleFunc("/tenant/conceptos-locales/lista", middleware.RequireAuth(conceptoTenantHandler.Listar))
	mux.HandleFunc("/tenant/conceptos-locales/crear", middleware.RequireAuth(conceptoTenantHandler.Crear))
	mux.HandleFunc("/tenant/conceptos-locales/editar-ui", middleware.RequireAuth(conceptoTenantHandler.EditarUI))
	mux.HandleFunc("/tenant/conceptos-locales/actualizar", middleware.RequireAuth(conceptoTenantHandler.Actualizar))
	mux.HandleFunc("/tenant/conceptos-locales/formulario-crear", middleware.RequireAuth(conceptoTenantHandler.FormularioCrearUI))
	mux.HandleFunc("/tenant/conceptos-locales/fila", middleware.RequireAuth(conceptoTenantHandler.FilaUI))
	mux.HandleFunc("/tenant/conceptos-locales/restaurar", middleware.RequireAuth(conceptoTenantHandler.Restaurar))

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
	mux.HandleFunc("/tenant/planillas/cerrar", middleware.RequireAuth(planillaHandler.CerrarPlanilla))
	mux.HandleFunc("/tenant/planillas/detalle/ui", middleware.RequireAuth(planillaHandler.VistaDetalle))

	// Asistencias
	asistenciaRepo := repository.NewAsistenciaRepository(db)
	asistenciaHandler := handlers.AsistenciaHandler{Repo: asistenciaRepo}
	mux.HandleFunc("/tenant/ui/asistencia", middleware.RequireAuth(asistenciaHandler.VistaUI))
	mux.HandleFunc("/tenant/asistencia/lista", middleware.RequireAuth(asistenciaHandler.Listar))
	mux.HandleFunc("/tenant/asistencia/crear", middleware.RequireAuth(asistenciaHandler.Crear))
	mux.HandleFunc("/tenant/asistencia/importar", middleware.RequireAuth(asistenciaHandler.ImportarExcel))
	mux.HandleFunc("/tenant/asistencia/formulario-crear", middleware.RequireAuth(asistenciaHandler.FormularioCrearUI))
	mux.HandleFunc("/tenant/asistencia/editar-ui", middleware.RequireAuth(asistenciaHandler.EditarUI))
	mux.HandleFunc("/tenant/asistencia/actualizar", middleware.RequireAuth(asistenciaHandler.Actualizar))

	// Rutas de presupuesto anual de las planillas
	planillaService := services.NewPlanillaService(planillaRepo)
	presupuestoRepo := repository.NewPresupuestoRepository(db)
	presupuestoService := services.NewPresupuestoService(presupuestoRepo, puestoRepo, puestoConceptoRepo, planillaService)
	presupuestoHandler := handlers.NewPresupuestoHandler(presupuestoService, planillaRepo)
	mux.HandleFunc("/tenant/presupuesto/index", middleware.RequireAuth(presupuestoHandler.IndexUI))
	mux.HandleFunc("/tenant/presupuesto/generar", middleware.RequireAuth(presupuestoHandler.Generar))
	mux.HandleFunc("/tenant/presupuesto/matriz", middleware.RequireAuth(presupuestoHandler.CargarMatriz))
	mux.HandleFunc("/tenant/presupuesto/exportar/csv", middleware.RequireAuth(presupuestoHandler.ExportarCSV))
	mux.HandleFunc("/tenant/presupuesto/exportar/excel", middleware.RequireAuth(presupuestoHandler.ExportarExcel))
	mux.HandleFunc("/tenant/presupuesto/exportar/pdf", middleware.RequireAuth(presupuestoHandler.ExportarPDF))

	// Ruta para descargar el PDF de planilla y boeltas
	mux.HandleFunc("/tenant/planillas/descargar-reporte", middleware.RequireAuth(planillaHandler.DescargarReportePDF))
	mux.HandleFunc("/tenant/planillas/descargar-boletas", middleware.RequireAuth(planillaHandler.DescargarBoletasPDF))

	// Módulo de Reportes Generales
	reporteService := services.NewReporteService(
		trabajadorRepo,
		organigramaRepo,
		puestoRepo,
		conceptoTenantRepo,
		contratoRepo,
		tenantRepo,
	)
	reporteHandler := handlers.ReporteHandler{
		Service: reporteService,
	}
	mux.HandleFunc("/tenant/ui/reportes", middleware.RequireAuth(reporteHandler.VistaUI))
	mux.HandleFunc("/tenant/reportes/filtrar", middleware.RequireAuth(reporteHandler.FiltrarUI))
	mux.HandleFunc("/tenant/reportes/ver-pdf", middleware.RequireAuth(reporteHandler.ExportarPDF))
	mux.HandleFunc("/tenant/reportes/descargar-excel", middleware.RequireAuth(reporteHandler.ExportarExcel))
}

