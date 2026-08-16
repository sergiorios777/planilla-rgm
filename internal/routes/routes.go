package routes

import (
	"database/sql"
	"html/template"
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
	landingHandler := handlers.NewLandingHandler()

	// 1.5 Archivos Estáticos
	fs := http.FileServer(http.Dir("ui/static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// Ruta principal (Landing Page Pública)
	mux.HandleFunc("/", landingHandler.MostrarLanding)

	// Ruta de Solicitud de Demo Institucional
	mux.HandleFunc("/solicitar-demo", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			landingHandler.MostrarSolicitudDemo(w, r)
		case http.MethodPost:
			landingHandler.ProcesarSolicitudDemo(w, r)
		default:
			http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		}
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

	// Rutas de Notificaciones (para ambos entornos)
	notifRepo := repository.NewNotificacionRepository(db)
	nh := handlers.NewNotificacionHandler(notifRepo)
	mux.HandleFunc("/notificaciones/campana", middleware.RequireAuth(nh.CampanaContadorUI))
	mux.HandleFunc("/notificaciones/lista", middleware.RequireAuth(nh.ListaNotificacionesUI))

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

	resumenRepo := repository.NewResumenRepository(db)
	resumenService := services.NewResumenService(resumenRepo)
	resumenHandler := handlers.NewResumenHandler(resumenService, adminRepo, nil)

	// 🔒 Reemplazamos RequireAuth por RequireRole("super_admin")
	mux.HandleFunc("/admin", middleware.RequireRole("super_admin", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles("ui/templates/layouts/index.html", "ui/templates/layouts/iconos_sprite.html")
		if err != nil {
			http.Error(w, "Error cargando la vista principal", http.StatusInternalServerError)
			return
		}

		tmpl.Execute(w, nil)
	}))

	mux.HandleFunc("/admin/ui/kpis-parcial", middleware.RequireRole("super_admin", resumenHandler.RefrescarKPIAdminParcial))

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
	mux.HandleFunc("/admin/conceptos/plantilla-ejemplo", middleware.RequireRole("super_admin", c.DescargarPlantillaCSV))

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
	planillaRepo := repository.NewPlanillaRepository(db)
	conceptoModeloService := services.NewConceptoModeloService(conceptosModeloRepo, db)
	notifRepoForConcepts := repository.NewNotificacionRepository(db)
	cm := handlers.ConceptoModeloHandler{
		Repo:               conceptosModeloRepo,
		PuestoRepo:         puestoRepo,
		ConceptoTenantRepo: conceptosTenantRepo,
		TenantRepo:         tenantRepo,
		PlanillaRepo:       planillaRepo,
		Service:            conceptoModeloService,
		NotificacionRepo:   notifRepoForConcepts,
	}
	mux.HandleFunc("/admin/ui/conceptos-modelo", middleware.RequireRole("super_admin", cm.VistaUI))
	mux.HandleFunc("/admin/conceptos-modelo/lista", middleware.RequireRole("super_admin", cm.Listar))
	mux.HandleFunc("/admin/conceptos-modelo/crear", middleware.RequireRole("super_admin", cm.Crear))
	mux.HandleFunc("/admin/conceptos-modelo/editar_ui", middleware.RequireRole("super_admin", cm.EditarUI))
	mux.HandleFunc("/admin/conceptos-modelo/actualizar", middleware.RequireRole("super_admin", cm.Actualizar))
	mux.HandleFunc("/admin/conceptos-modelo/eliminar", middleware.RequireRole("super_admin", cm.Eliminar))
	mux.HandleFunc("/admin/conceptos-modelo/sincronizar", middleware.RequireRole("super_admin", cm.Sincronizar))
	mux.HandleFunc("/admin/conceptos-modelo/importar", middleware.RequireRole("super_admin", cm.ImportarCSV))
	mux.HandleFunc("/admin/conceptos-modelo/plantilla-csv", middleware.RequireRole("super_admin", cm.PlantillaCSV))
	mux.HandleFunc("/admin/conceptos-modelo/reglas-modal", middleware.RequireRole("super_admin", cm.ReglasModal))
	mux.HandleFunc("/admin/conceptos-modelo/reglas/crear", middleware.RequireRole("super_admin", cm.CrearReglaHTMX))
	mux.HandleFunc("/admin/conceptos-modelo/reglas/eliminar", middleware.RequireRole("super_admin", cm.EliminarReglaHTMX))

	// Rutas del Motor de Conceptos Calculados
	baseRegimenRepo := repository.NewBaseRegimenRepository(db)
	calculadoHandler := handlers.NewConceptoCalculadoHandler(baseRegimenRepo, puestoRepo, conceptoModeloService, tenantRepo)

	mux.HandleFunc("/admin/ui/conceptos-calculados", middleware.RequireRole("super_admin", calculadoHandler.VistaUI))
	mux.HandleFunc("/admin/conceptos-calculados/lista", middleware.RequireRole("super_admin", calculadoHandler.Listar))
	mux.HandleFunc("/admin/conceptos-calculados/crear", middleware.RequireRole("super_admin", calculadoHandler.Crear))
	mux.HandleFunc("/admin/conceptos-calculados/eliminar", middleware.RequireRole("super_admin", calculadoHandler.Eliminar))
	mux.HandleFunc("/admin/conceptos-calculados/afectaciones", middleware.RequireRole("super_admin", calculadoHandler.VistaAfectaciones))
	mux.HandleFunc("/admin/conceptos-calculados/afectaciones/agregar", middleware.RequireRole("super_admin", calculadoHandler.AgregarAfectacion))
	mux.HandleFunc("/admin/conceptos-calculados/afectaciones/eliminar", middleware.RequireRole("super_admin", calculadoHandler.EliminarAfectacion))
	mux.HandleFunc("/admin/conceptos-calculados/opciones-modelo", middleware.RequireRole("super_admin", calculadoHandler.OpcionesModelo))
	mux.HandleFunc("/admin/conceptos-calculados/propagar", middleware.RequireRole("super_admin", calculadoHandler.Propagar))

	// Rutas de AFPs y Tasas Mensuales
	afpRepo := repository.NewAFPRepository(db)
	afpService := services.NewAFPService(afpRepo)
	afph := handlers.AFPHandler{Repo: afpRepo, Service: afpService}
	mux.HandleFunc("/admin/ui/afps", middleware.RequireRole("super_admin", afph.VistaUI))
	mux.HandleFunc("/admin/afps", middleware.RequireRole("super_admin", afph.ListarAFPs))
	mux.HandleFunc("/admin/afps/crear", middleware.RequireRole("super_admin", afph.CrearAFP))
	mux.HandleFunc("/admin/afps/editar_ui", middleware.RequireRole("super_admin", afph.EditarAFPUI))
	mux.HandleFunc("/admin/afps/actualizar", middleware.RequireRole("super_admin", afph.ActualizarAFP))
	mux.HandleFunc("/admin/afps/tasas", middleware.RequireRole("super_admin", afph.ListarTasas))
	mux.HandleFunc("/admin/afps/importar", middleware.RequireRole("super_admin", afph.ImportarCSV))

	// Rutas de Tareas Programadas (Super Admin)
	tareaRepo := repository.NewAdminTareaRepository(db)
	tareah := handlers.NewAdminTareaHandler(tareaRepo)
	mux.HandleFunc("/admin/ui/tareas", middleware.RequireRole("super_admin", tareah.VistaUI))
	mux.HandleFunc("/admin/tareas", middleware.RequireRole("super_admin", tareah.Listar))
	mux.HandleFunc("/admin/tareas/crear", middleware.RequireRole("super_admin", tareah.Crear))
	mux.HandleFunc("/admin/tareas/editar_ui", middleware.RequireRole("super_admin", tareah.EditarUI))
	mux.HandleFunc("/admin/tareas/actualizar", middleware.RequireRole("super_admin", tareah.Actualizar))

	// Rutas de Valores Históricos MUC (MEF)
	mefMucRepo := repository.NewMefMucRepository(db)
	mefMucHandler := handlers.NewMefMucHandler(mefMucRepo)
	mux.HandleFunc("/admin/ui/mef-muc", middleware.RequireRole("super_admin", mefMucHandler.VistaUI))
	mux.HandleFunc("/admin/mef-muc/lista", middleware.RequireRole("super_admin", mefMucHandler.Listar))
	mux.HandleFunc("/admin/mef-muc/crear", middleware.RequireRole("super_admin", mefMucHandler.Crear))
	mux.HandleFunc("/admin/mef-muc/formulario-editar", middleware.RequireRole("super_admin", mefMucHandler.EditarForm))
	mux.HandleFunc("/admin/mef-muc/actualizar", middleware.RequireRole("super_admin", mefMucHandler.Actualizar))
	mux.HandleFunc("/admin/mef-muc/toggle-estado", middleware.RequireRole("super_admin", mefMucHandler.ToggleEstado))
	mux.HandleFunc("/admin/mef-muc/importar-csv", middleware.RequireRole("super_admin", mefMucHandler.ImportarCSV))
	mux.HandleFunc("/admin/mef-muc/plantilla-csv", middleware.RequireRole("super_admin", mefMucHandler.DescargarPlantillaCSV))
}

// --- SECCIÓN TENANT (MUNICIPALIDADES) ---
func registrarRutasTenant(mux *http.ServeMux, db *sql.DB) {
	// Repositorio y Servicio de Resumen / KPIs
	resumenRepo := repository.NewResumenRepository(db)
	resumenService := services.NewResumenService(resumenRepo)
	tenantRepo := repository.NewTenantRepository(db)
	usuarioRepo := repository.NewUsuarioRepository(db)
	resumenHandler := handlers.NewResumenHandler(resumenService, tenantRepo, usuarioRepo)

	// Ruta principal al esqueleto de tenant cargado con KPIs
	mux.HandleFunc("/tenant", middleware.RequireAuth(resumenHandler.IndexTenant))
	mux.HandleFunc("/tenant/ui/kpis-parcial", middleware.RequireAuth(resumenHandler.RefrescarKPITenantParcial))

	// Rutas de Inquilino
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
	mux.HandleFunc("/tenant/contratos/baja", middleware.RequireAuth(contratoHandler.ProcesarBaja))
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
	planillaRepo := repository.NewPlanillaRepository(db)
	conceptoTenantHandler := handlers.ConceptoTenantHandler{
		Repo:         conceptoTenantRepo,
		PuestoRepo:   puestoRepo,
		PlanillaRepo: planillaRepo,
	}
	mux.HandleFunc("/tenant/ui/conceptos-locales", middleware.RequireAuth(conceptoTenantHandler.VistaUI))
	mux.HandleFunc("/tenant/conceptos-locales/lista", middleware.RequireAuth(conceptoTenantHandler.Listar))
	mux.HandleFunc("/tenant/conceptos-locales/crear", middleware.RequireAuth(conceptoTenantHandler.Crear))
	mux.HandleFunc("/tenant/conceptos-locales/editar-ui", middleware.RequireAuth(conceptoTenantHandler.EditarUI))
	mux.HandleFunc("/tenant/conceptos-locales/actualizar", middleware.RequireAuth(conceptoTenantHandler.Actualizar))
	mux.HandleFunc("/tenant/conceptos-locales/formulario-crear", middleware.RequireAuth(conceptoTenantHandler.FormularioCrearUI))
	mux.HandleFunc("/tenant/conceptos-locales/fila", middleware.RequireAuth(conceptoTenantHandler.FilaUI))
	mux.HandleFunc("/tenant/conceptos-locales/restaurar", middleware.RequireAuth(conceptoTenantHandler.Restaurar))
	mux.HandleFunc("/tenant/conceptos-locales/modal-agregar-modelo", middleware.RequireAuth(conceptoTenantHandler.ModalAgregarModeloUI))
	mux.HandleFunc("/tenant/conceptos-locales/agregar-modelo", middleware.RequireAuth(conceptoTenantHandler.AgregarModelo))
	mux.HandleFunc("/tenant/conceptos-locales/sincronizar-modelo", middleware.RequireAuth(conceptoTenantHandler.SincronizarModelo))
	mux.HandleFunc("/tenant/conceptos-locales/reglas-modal", middleware.RequireAuth(conceptoTenantHandler.ReglasModal))
	mux.HandleFunc("/tenant/conceptos-locales/reglas/crear", middleware.RequireAuth(conceptoTenantHandler.CrearReglaHTMX))
	mux.HandleFunc("/tenant/conceptos-locales/reglas/eliminar", middleware.RequireAuth(conceptoTenantHandler.EliminarReglaHTMX))
	mux.HandleFunc("/tenant/conceptos-locales/reglas/actualizar", middleware.RequireAuth(conceptoTenantHandler.ActualizarReglaHTMX))
	mux.HandleFunc("/tenant/conceptos-locales/reglas/sincronizar-modelo", middleware.RequireAuth(conceptoTenantHandler.SincronizarReglasModeloHTMX))

	// Rutas protegidas (Agrega esto junto a las de Puestos)
	puestoConceptoRepo := repository.NewPuestoConceptoRepository(db)
	contratoService := &services.ContratoService{
		Repo:           contratoRepo,
		RepoTrabajador: trabajadorRepo,
		RepoPuesto:     puestoRepo,
	}
	notifRepo := repository.NewNotificacionRepository(db)
	puestoConceptoHandler := handlers.PuestoConceptoHandler{
		Repo:             puestoConceptoRepo,
		PuestoRepo:       puestoRepo,
		ContratoService:  contratoService,
		NotificacionRepo: notifRepo,
	}
	mux.HandleFunc("/tenant/puestos-conceptos/ui", middleware.RequireAuth(puestoConceptoHandler.VistaUI))
	mux.HandleFunc("/tenant/puestos-conceptos/lista", middleware.RequireAuth(puestoConceptoHandler.Listar))
	mux.HandleFunc("/tenant/puestos-conceptos/crear", middleware.RequireAuth(puestoConceptoHandler.Crear))
	mux.HandleFunc("/tenant/puestos-conceptos/eliminar", middleware.RequireAuth(puestoConceptoHandler.Eliminar))
	mux.HandleFunc("/tenant/puestos-conceptos/restaurar", middleware.RequireAuth(puestoConceptoHandler.RestaurarCostosBase))
	mux.HandleFunc("/tenant/puestos-conceptos/restaurar-todos", middleware.RequireAuth(puestoConceptoHandler.RestaurarTodosCostosBase))
	mux.HandleFunc("/tenant/puestos-conceptos/editar-monto-ui", middleware.RequireAuth(puestoConceptoHandler.EditarMontoUI))
	mux.HandleFunc("/tenant/puestos-conceptos/actualizar-monto", middleware.RequireAuth(puestoConceptoHandler.ActualizarMonto))

	// Rutas protegidas (Bajo una nueva sección de Procesamiento)
	anexoRepo := repository.NewAnexoRepository(db)
	tenantRepo = repository.NewTenantRepository(db)
	anexoService := services.NewAnexoService(anexoRepo, planillaRepo, tenantRepo)

	planillaHandler := handlers.PlanillaHandler{
		Repo:               planillaRepo,
		PuestoRepo:         puestoRepo,
		ConceptoTenantRepo: conceptoTenantRepo,
		OrganigramaRepo:    organigramaRepo,
		AnexoService:       anexoService,
	}
	mux.HandleFunc("/tenant/ui/planillas", middleware.RequireAuth(planillaHandler.VistaUI))
	mux.HandleFunc("/tenant/planillas/lista", middleware.RequireAuth(planillaHandler.Listar))
	mux.HandleFunc("/tenant/planillas/crear", middleware.RequireAuth(planillaHandler.Crear))
	mux.HandleFunc("/tenant/planillas/procesar", middleware.RequireAuth(planillaHandler.Procesar))
	mux.HandleFunc("/tenant/planillas/especial/ui", middleware.RequireAuth(planillaHandler.VistaFormuladorEspecial))
	mux.HandleFunc("/tenant/planillas/especial/trabajadores", middleware.RequireAuth(planillaHandler.BuscarTrabajadoresEspecial))
	mux.HandleFunc("/tenant/planillas/especial/trabajadores-json", middleware.RequireAuth(planillaHandler.BuscarTrabajadoresEspecialJSON))
	mux.HandleFunc("/tenant/planillas/especial/procesar", middleware.RequireAuth(planillaHandler.ProcesarEspecial))
	mux.HandleFunc("/tenant/planillas/cerrar", middleware.RequireAuth(planillaHandler.CerrarPlanilla))
	mux.HandleFunc("/tenant/planillas/eliminar", middleware.RequireAuth(planillaHandler.Eliminar))
	mux.HandleFunc("/tenant/planillas/detalle/ui", middleware.RequireAuth(planillaHandler.VistaDetalle))
	mux.HandleFunc("/tenant/planillas/boleta-modal", middleware.RequireAuth(planillaHandler.BoletaModalDetalle))
	mux.HandleFunc("/tenant/ui/planillas/rubros-metas", middleware.RequireAuth(planillaHandler.VistaRubrosMetas))
	mux.HandleFunc("/tenant/planillas/conceptos/actualizar-presupuesto-single", middleware.RequireAuth(planillaHandler.ActualizarPresupuestoSingleHTMX))
	mux.HandleFunc("/tenant/planillas/conceptos/actualizar-presupuesto-bulk", middleware.RequireAuth(planillaHandler.ActualizarPresupuestoBulkHTMX))
	mux.HandleFunc("/tenant/planillas/descargar-reporte", middleware.RequireAuth(planillaHandler.DescargarReportePDF))
	mux.HandleFunc("/tenant/planillas/descargar-boletas", middleware.RequireAuth(planillaHandler.DescargarBoletasPDF))
	mux.HandleFunc("/tenant/planillas/exportar-plame-modal", middleware.RequireAuth(planillaHandler.ExportarPlameModal))
	mux.HandleFunc("/tenant/planillas/descargar-plame", middleware.RequireAuth(planillaHandler.DescargarPlame))
	mux.HandleFunc("/tenant/planillas/anexos/ui", middleware.RequireAuth(planillaHandler.VistaAnexos))
	mux.HandleFunc("/tenant/planillas/anexos/anexo1/pdf", middleware.RequireAuth(planillaHandler.DescargarAnexo1PDF))
	mux.HandleFunc("/tenant/planillas/anexos/anexo1/excel", middleware.RequireAuth(planillaHandler.DescargarAnexo1Excel))
	mux.HandleFunc("/tenant/planillas/anexos/anexo1a/pdf", middleware.RequireAuth(planillaHandler.DescargarAnexo1APDF))
	mux.HandleFunc("/tenant/planillas/anexos/anexo1a/excel", middleware.RequireAuth(planillaHandler.DescargarAnexo1AExcel))
	mux.HandleFunc("/tenant/planillas/anexos/anexo2/pdf", middleware.RequireAuth(planillaHandler.DescargarAnexo2PDF))
	mux.HandleFunc("/tenant/planillas/anexos/anexo2/excel", middleware.RequireAuth(planillaHandler.DescargarAnexo2Excel))
	mux.HandleFunc("/tenant/planillas/anexos/anexo2a/pdf", middleware.RequireAuth(planillaHandler.DescargarAnexo2APDF))
	mux.HandleFunc("/tenant/planillas/anexos/anexo2a/excel", middleware.RequireAuth(planillaHandler.DescargarAnexo2AExcel))
	mux.HandleFunc("/tenant/planillas/anexos/anexo3/pdf", middleware.RequireAuth(planillaHandler.DescargarAnexo3PDF))
	mux.HandleFunc("/tenant/planillas/anexos/anexo3/excel", middleware.RequireAuth(planillaHandler.DescargarAnexo3Excel))

	// Rutas API para Presupuesto de Conceptos de Planilla
	mux.HandleFunc("PUT /api/tenant/planillas/{id}/conceptos/{concepto_id}/presupuesto", middleware.RequireAuth(planillaHandler.ActualizarPresupuestoConcepto))
	mux.HandleFunc("POST /api/tenant/planillas/{id}/conceptos/bulk-presupuesto", middleware.RequireAuth(planillaHandler.ActualizarPresupuestoConceptosBulk))

	// Rutas API CRUD para Reglas de Financiamiento
	reglaHandler := handlers.ReglaFinanciamientoHandler{Repo: planillaRepo}
	mux.HandleFunc("GET /api/tenant/reglas-financiamiento", middleware.RequireAuth(reglaHandler.Listar))
	mux.HandleFunc("POST /api/tenant/reglas-financiamiento", middleware.RequireAuth(reglaHandler.Crear))
	mux.HandleFunc("GET /api/tenant/reglas-financiamiento/{id}", middleware.RequireAuth(reglaHandler.Obtener))
	mux.HandleFunc("PUT /api/tenant/reglas-financiamiento/{id}", middleware.RequireAuth(reglaHandler.Actualizar))
	mux.HandleFunc("DELETE /api/tenant/reglas-financiamiento/{id}", middleware.RequireAuth(reglaHandler.Eliminar))
	mux.HandleFunc("/api/tenant/reglas-financiamiento", middleware.RequireAuth(reglaHandler.HandleRoot))
	mux.HandleFunc("/api/tenant/reglas-financiamiento/", middleware.RequireAuth(reglaHandler.HandleWithID))

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

	// Módulo de CTS Semestral
	ctsRepo := repository.NewCtsRepository(db)
	ctsService := services.NewCtsService(ctsRepo, db)
	ctsHandler := handlers.CtsHandler{
		CtsRepo:      ctsRepo,
		CtsService:   ctsService,
		ContratoRepo: contratoRepo,
	}

	// Módulo de Liquidaciones de Cese y Vacaciones
	pdfService := services.NewPdfService()
	vacService := services.NewVacacionesService(repository.NewBaseRegimenRepository(db))
	liquidacionRepo := repository.NewLiquidacionRepository(db)
	liquidacionService := services.NewLiquidacionService(db, vacService)
	liquidacionHandler := handlers.NewLiquidacionHandler(liquidacionRepo, liquidacionService, contratoRepo, pdfService)

	// Rutas de CTS Semestral
	mux.HandleFunc("/tenant/ui/cts", middleware.RequireAuth(ctsHandler.CtsVistaUI))
	mux.HandleFunc("/tenant/cts/lista", middleware.RequireAuth(ctsHandler.ListarPlanillasCts))
	mux.HandleFunc("/tenant/cts/crear", middleware.RequireAuth(ctsHandler.CrearPlanillaCts))
	mux.HandleFunc("/tenant/cts/detalle", middleware.RequireAuth(ctsHandler.VerDetalleCts))
	mux.HandleFunc("/tenant/cts/subir-excel", middleware.RequireAuth(ctsHandler.SubirExcelGratificaciones))
	mux.HandleFunc("/tenant/cts/cerrar", middleware.RequireAuth(ctsHandler.CerrarPlanillaCts))
	mux.HandleFunc("/tenant/cts/eliminar", middleware.RequireAuth(ctsHandler.EliminarPlanillaCts))

	// Rutas de Liquidaciones de Cese
	mux.HandleFunc("/tenant/ui/liquidaciones", middleware.RequireAuth(liquidacionHandler.LiquidacionesVistaUI))
	mux.HandleFunc("/tenant/ui/liquidaciones/nueva", middleware.RequireAuth(liquidacionHandler.FormularioNuevaLiquidacionUI))
	mux.HandleFunc("/tenant/liquidaciones/lista", middleware.RequireAuth(liquidacionHandler.ListarLiquidacionesCese))
	mux.HandleFunc("/tenant/liquidaciones/formulario-crear", middleware.RequireAuth(liquidacionHandler.FormularioCrearUI))
	mux.HandleFunc("/tenant/liquidaciones/calcular", middleware.RequireAuth(liquidacionHandler.CalcularLiquidacionCese))
	mux.HandleFunc("/tenant/liquidaciones/guardar", middleware.RequireAuth(liquidacionHandler.GuardarLiquidacionCese))
	mux.HandleFunc("/tenant/liquidaciones/eliminar", middleware.RequireAuth(liquidacionHandler.EliminarLiquidacionCese))
	mux.HandleFunc("/tenant/liquidaciones/reporte-pdf", middleware.RequireAuth(liquidacionHandler.DescargarReportePDF))
}
