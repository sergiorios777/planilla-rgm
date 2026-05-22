package handlers

import (
	"html/template"
	"log"
	"net/http"
	"planilla-rgm/internal/config"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
	"planilla-rgm/internal/services"
	"sort"
	"strconv"
	"strings"
)

type ContratoHandler struct {
	Repo           *repository.ContratoRepository
	TrabajadorRepo *repository.TrabajadorRepository // Lo necesitamos para el select
	PuestoRepo     *repository.PuestoRepository
}

func (h *ContratoHandler) VistaUI(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)

	trabajadores, _ := h.TrabajadorRepo.ObtenerTodos(tenantID)
	puestos, _ := h.PuestoRepo.ObtenerVacantes(tenantID)
	regimenes, _ := h.PuestoRepo.ObtenerRegimenes()

	datos := map[string]interface{}{
		"Trabajadores":       trabajadores,
		"Puestos":            puestos,
		"RegimenesLaborales": regimenes,
	}

	tmpl, _ := template.ParseFiles("ui/templates/tenant/contratos_ui.html")
	tmpl.Execute(w, datos)
}

func (h *ContratoHandler) Listar(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	busqueda := r.URL.Query().Get("buscar")
	limiteStr := r.URL.Query().Get("limite")
	paginaStr := r.URL.Query().Get("pagina")
	regimenStr := r.URL.Query().Get("regimen_laboral_id")

	limite, err := strconv.Atoi(limiteStr)
	if err != nil || limite <= 0 {
		limite = 10 // Por defecto mostramos 10
	}

	pagina, err := strconv.Atoi(paginaStr)
	if err != nil || pagina <= 0 {
		pagina = 1 // Por defecto empezamos en la página 1
	}

	regimenID, err := strconv.Atoi(regimenStr)
	if err != nil {
		regimenID = 0
	}

	offset := (pagina - 1) * limite

	contratos, totalRegistros, err := h.Repo.ObtenerTodosPaginado(tenantID, busqueda, regimenID, limite, offset)
	if err != nil {
		http.Error(w, "Error al obtener los contratos", http.StatusInternalServerError)
		return
	}
	totalPaginas := (totalRegistros + limite - 1) / limite

	if totalPaginas == 0 {
		totalPaginas = 1
	}

	// Construimos los datos struc y objetos al vuelo
	datosPaginacion := struct {
		Contratos       []models.Contrato
		TotalPaginas    int
		PaginaActual    int
		PaginaAnterior  int
		PaginaSiguiente int
	}{
		Contratos:       contratos,
		TotalPaginas:    totalPaginas,
		PaginaActual:    pagina,
		PaginaAnterior:  pagina - 1,
		PaginaSiguiente: pagina + 1,
	}

	tmpl, _ := template.ParseFiles("ui/templates/tenant/contratos_ui.html")
	tmpl.ExecuteTemplate(w, "tabla_contratos", datosPaginacion)
}

func (h *ContratoHandler) Crear(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	tenantID := obtenerTenantID(r)
	tID, _ := strconv.Atoi(r.FormValue("trabajador_id"))
	pID, _ := strconv.Atoi(r.FormValue("puesto_id"))

	// === 1. NUEVA VALIDACIÓN DE NEGOCIO ===
	tieneActivo, err := h.Repo.TieneContratoActivo(tID, tenantID)
	if err != nil {
		http.Error(w, "Error validando el estado del trabajador", http.StatusInternalServerError)
		return
	}

	if tieneActivo {
		// TRUCO HTMX: Devolvemos un fragmento HTML con la etiqueta hx-swap-oob="true".
		// HTMX buscará el div con id="alerta-contrato" y le inyectará este error, sin tocar la tabla.
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`
			<div id="alerta-contrato" hx-swap-oob="true">
				<article style="background-color: #ffcdd2; color: #b71c1c; padding: 1rem; margin-bottom: 1rem; border-radius: 5px; font-weight: bold;">
					❌ Error: El trabajador seleccionado ya posee un contrato activo (Plaza ocupada). Debe finalizarlo antes de asignarle uno nuevo.
				</article>
			</div>
		`))
		return
	}
	// =======================================

	fFinStr := r.FormValue("fecha_fin")
	var fFin *string
	if strings.TrimSpace(fFinStr) != "" {
		fFin = &fFinStr
	}

	nuevoContrato := models.Contrato{
		TenantID:     tenantID,
		TrabajadorID: tID,
		PuestoID:     pID,
		FechaInicio:  r.FormValue("fecha_inicio"),
		FechaFin:     fFin,
		Activo:       r.FormValue("activo") == "on",
		TipoContrato: r.FormValue("tipo_contrato"),
	}

	// Instanciamos el contrato service para llamar a la funcion CrearContrato
	servicioContrato := services.ContratoService{
		RepoPuesto:     h.PuestoRepo,
		Repo:           h.Repo,
		RepoTrabajador: h.TrabajadorRepo,
	}

	// Si el contrato se crea con éxito, enviamos una orden OOB para "limpiar" cualquier alerta anterior
	w.Write([]byte(`<div id="alerta-contrato" hx-swap-oob="true"></div>`))

	// Disparamos la creación e inyección automática de conceptos y pensiones
	err = servicioContrato.CrearContrato(&nuevoContrato)
	if err != nil {
		log.Println("Error al crear contrato:", err)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`
			<div id="alerta-contrato" hx-swap-oob="true">
				<article style="background-color: #ffcdd2; color: #b71c1c; padding: 1rem; margin-bottom: 1rem; border-radius: 5px; font-weight: bold;">
					❌ Error: No se pudo generar el contrato. Verifique la configuración del régimen y los clasificadores.
				</article>
			</div>
		`))
		return
	}

	// Finalmente, devolvemos la tabla actualizada como siempre
	h.Listar(w, r)
}

// FormularioCrearUI devuelve el form limpio
func (h *ContratoHandler) FormularioCrearUI(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	trabajadores, _ := h.TrabajadorRepo.ObtenerTodos(tenantID)
	puestos, _ := h.PuestoRepo.ObtenerVacantes(tenantID)
	datos := map[string]interface{}{"Trabajadores": trabajadores, "Puestos": puestos}

	tmpl, _ := template.ParseFiles("ui/templates/tenant/contratos_ui.html")
	tmpl.ExecuteTemplate(w, "formulario_crear", datos)
}

// FormularioDinamicoUI devuelve el formulario de creación parcial/completo con opciones dinámicas de contrato
func (h *ContratoHandler) FormularioDinamicoUI(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	puestoIDStr := r.URL.Query().Get("puesto_id")
	pID, _ := strconv.Atoi(puestoIDStr)

	// Extraemos otros campos para preservar su estado
	trabajadorIDStr := r.URL.Query().Get("trabajador_id")
	tID, _ := strconv.Atoi(trabajadorIDStr)
	fechaInicio := r.URL.Query().Get("fecha_inicio")
	fechaFin := r.URL.Query().Get("fecha_fin")
	
	_, hasPuesto := r.URL.Query()["puesto_id"]
	_, hasActivo := r.URL.Query()["activo"]
	activo := hasActivo || !hasPuesto

	var opciones []string
	if pID > 0 {
		puesto, err := h.PuestoRepo.ObtenerPorID(pID, tenantID)
		if err == nil {
			key := config.MapRegimenToKey(puesto.RegimenCodigo)
			if mapOpciones, ok := config.ClasificadorMefPorContrato[key]; ok {
				for k := range mapOpciones {
					opciones = append(opciones, k)
				}
				sort.Strings(opciones)
			}
		}
	}

	trabajadores, _ := h.TrabajadorRepo.ObtenerTodos(tenantID)
	puestos, _ := h.PuestoRepo.ObtenerVacantes(tenantID)

	datos := map[string]interface{}{
		"Trabajadores":             trabajadores,
		"Puestos":                  puestos,
		"PuestoSeleccionadoID":     pID,
		"OpcionesContrato":         opciones,
		"TrabajadorSeleccionadoID": tID,
		"FechaInicio":              fechaInicio,
		"FechaFin":                 fechaFin,
		"Activo":                   activo,
	}

	tmpl, _ := template.ParseFiles("ui/templates/tenant/contratos_ui.html")
	tmpl.ExecuteTemplate(w, "formulario_crear", datos)
}

// EditarUI carga el formulario de edición
func (h *ContratoHandler) EditarUI(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	tenantID := obtenerTenantID(r)

	contrato, _ := h.Repo.ObtenerPorID(id, tenantID)

	tmpl, _ := template.ParseFiles("ui/templates/tenant/contratos_ui.html")
	tmpl.ExecuteTemplate(w, "formulario_editar", contrato)
}

// Actualizar guarda cambios, recarga tabla y limpia form
func (h *ContratoHandler) Actualizar(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id, _ := strconv.Atoi(r.FormValue("id"))
	puestoID, _ := strconv.Atoi(r.FormValue("puesto_id"))

	fFinStr := r.FormValue("fecha_fin")
	var fFin *string
	if strings.TrimSpace(fFinStr) != "" {
		fFin = &fFinStr
	}

	cActualizado := models.Contrato{
		ID:          id,
		TenantID:    obtenerTenantID(r),
		PuestoID:    puestoID, // Lo enviamos oculto para poder liberar la plaza si se inactiva
		FechaInicio: r.FormValue("fecha_inicio"),
		FechaFin:    fFin,
		Activo:      r.FormValue("activo") == "on",
	}

	h.Repo.Actualizar(&cActualizado)

	// Pedimos recargar la tabla
	w.Header().Set("HX-Trigger", "recargarTablaContratos")

	// Volvemos al form de creación
	h.FormularioCrearUI(w, r)
}
