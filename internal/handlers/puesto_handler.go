package handlers

import (
	"html/template"
	"log"
	"net/http"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
	"planilla-rgm/internal/services"
	"strconv"
)

type PuestoHandler struct {
	Repo            *repository.PuestoRepository
	MetaRepo        *repository.MetaRepository
	FuenteRubroRepo *repository.FuenteRubroRepository
	OrganigramaRepo *repository.OrganigramaRepository
}

func (h *PuestoHandler) VistaUI(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)

	// Preparamos listas para los combos
	metas, _ := h.MetaRepo.ObtenerTodos(tenantID)
	fuentes, _ := h.FuenteRubroRepo.ObtenerPorAnio(2026, "")
	regimenes, _ := h.Repo.ObtenerRegimenes()
	unidades, _ := h.OrganigramaRepo.ObtenerUnidadesDelOrganigramaActivo(tenantID)

	datos := map[string]interface{}{
		"Metas":           metas,
		"Fuentes":         fuentes,
		"Regimenes":       regimenes,
		"Unidades":        unidades,
		"CurrentUnidadID": 0,
	}

	tmpl, err := template.ParseFiles("ui/templates/tenant/puestos_ui.html")

	if err != nil {
		log.Println("❌ Error CRÍTICO al cargar la plantilla de puestos:", err)
		http.Error(w, "Error interno del servidor al cargar la interfaz", 500)
		return
	}

	err = tmpl.Execute(w, datos)
	if err != nil {
		log.Println("❌ Error al renderizar la plantilla:", err)
	}
}

func (h *PuestoHandler) Listar(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	busqueda := r.URL.Query().Get("buscar")
	metaIDStr := r.URL.Query().Get("meta_id")
	regimenIDStr := r.URL.Query().Get("regimen_id")
	unidadIDStr := r.URL.Query().Get("unidad_organica_id")
	estado := r.URL.Query().Get("estado")
	limiteStr := r.URL.Query().Get("limite")
	paginaStr := r.URL.Query().Get("pagina")

	metaID, err := strconv.Atoi(metaIDStr)
	regimenID, err := strconv.Atoi(regimenIDStr)
	unidadID, _ := strconv.Atoi(unidadIDStr)

	limite, err := strconv.Atoi(limiteStr)
	if err != nil || limite <= 0 {
		limite = 10 // Por defecto mostramos 10
	}

	pagina, err := strconv.Atoi(paginaStr)
	if err != nil || pagina <= 0 {
		pagina = 1 // Por defecto empezamos en la página 1
	}

	offset := (pagina - 1) * limite

	puestos, totalRegistros, err := h.Repo.ObtenerTodosPaginacion(tenantID, metaID, regimenID, unidadID, busqueda, estado, limite, offset)
	if err != nil {
		http.Error(w, "Error al obtener las metas", http.StatusInternalServerError)
		return
	}

	totalPaginas := (totalRegistros + limite - 1) / limite

	if totalPaginas == 0 {
		totalPaginas = 1
	}

	datosVista := struct {
		Puestos         []models.Puesto
		TotalPaginas    int
		PaginaActual    int
		PaginaAnterior  int
		PaginaSiguiente int
	}{
		Puestos:         puestos,
		TotalPaginas:    totalPaginas,
		PaginaActual:    pagina,
		PaginaAnterior:  pagina - 1,
		PaginaSiguiente: pagina + 1,
	}

	tmpl, _ := template.ParseFiles("ui/templates/tenant/puestos_ui.html")
	tmpl.ExecuteTemplate(w, "tabla_puestos", datosVista)
}

func (h *PuestoHandler) Crear(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	metaID, _ := strconv.Atoi(r.FormValue("meta_id"))
	fuenteID, _ := strconv.Atoi(r.FormValue("fuente_rubro_id"))
	regimenID, _ := strconv.Atoi(r.FormValue("regimen_id"))
	sueldo, _ := strconv.ParseFloat(r.FormValue("sueldo_presupuestado"), 64)

	var unidadOrganicaID *int
	idStr := r.FormValue("unidad_organica_id")
	if idStr != "" && idStr != "0" {
		idVal, err := strconv.Atoi(idStr)
		if err == nil {
			unidadOrganicaID = &idVal
		}
	}

	var codigoAirhsp *string
	airhspStr := r.FormValue("codigo_airhsp")
	if airhspStr != "" {
		codigoAirhsp = &airhspStr
	}

	nuevoPuesto := models.Puesto{
		TenantID:            obtenerTenantID(r),
		MetaID:              metaID,
		FuenteRubroID:       fuenteID,
		RegimenID:           regimenID,
		Nombre:              r.FormValue("nombre"),
		SueldoPresupuestado: sueldo,
		Activo:              r.FormValue("activo") == "on",
		EsDietario:          r.FormValue("es_dietario") == "on",
		UnidadOrganicaID:    unidadOrganicaID,
		CodigoAirhsp:        codigoAirhsp,
	}

	servicioPuesto := services.PuestoService{Repo: h.Repo}
	err := servicioPuesto.CrearPuestoConPlantilla(&nuevoPuesto)
	if err != nil {
		log.Println("Error creando puesto con plantilla:", err)
		http.Error(w, "Error al crear el puesto: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", "refrescarPuestos")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("✅ Plaza creada correctamente."))
}

// Editar prepara los datos del puesto y las listas para el formulario de edición
func (h *PuestoHandler) Editar(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	tenantID := obtenerTenantID(r)

	// 1. Buscamos el puesto actual
	puesto, _ := h.Repo.ObtenerPorID(id, tenantID)

	// 2. Necesitamos las listas para los combos (Selects)
	metas, _ := h.MetaRepo.ObtenerTodos(tenantID)
	fuentes, _ := h.FuenteRubroRepo.ObtenerPorAnio(2026, "")
	regimenes, _ := h.Repo.ObtenerRegimenes()

	datos := map[string]interface{}{
		"Puesto":    puesto,
		"Metas":     metas,
		"Fuentes":   fuentes,
		"Regimenes": regimenes,
	}

	// 💡 ENVIAMOS SOLO EL FRAGMENTO: "formulario_editar"
	tmpl, _ := template.ParseFiles("ui/templates/tenant/puestos_ui.html")
	tmpl.ExecuteTemplate(w, "formulario_editar", datos)
}

// EditarUI prepara los datos del puesto y las listas para el modal de edición
func (h *PuestoHandler) EditarUI(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	tenantID := obtenerTenantID(r)
	currentUnidadID := 0

	puesto, err := h.Repo.ObtenerPorID(id, tenantID)
	if err != nil {
		log.Println("Error al obtener puesto por ID:", err)
		http.Error(w, "No se pudo obtener el puesto", http.StatusInternalServerError)
		return
	}

	if puesto.UnidadOrganicaID != nil {
		currentUnidadID = *puesto.UnidadOrganicaID
	}

	metas, _ := h.MetaRepo.ObtenerTodos(tenantID)
	fuentes, _ := h.FuenteRubroRepo.ObtenerPorAnio(2026, "")
	regimenes, _ := h.Repo.ObtenerRegimenes()
	unidades, _ := h.OrganigramaRepo.ObtenerUnidadesDelOrganigramaActivo(tenantID)

	datos := map[string]interface{}{
		"Puesto":          puesto,
		"Metas":           metas,
		"Fuentes":         fuentes,
		"Regimenes":       regimenes,
		"Unidades":        unidades,
		"CurrentUnidadID": currentUnidadID,
	}

	tmpl, err := template.ParseFiles("ui/templates/tenant/puestos_ui.html")
	if err != nil {
		log.Println("Error al leer puestos_ui.html:", err)
		http.Error(w, "Error de plantilla", http.StatusInternalServerError)
		return
	}

	err = tmpl.ExecuteTemplate(w, "formulario_editar", datos)
	if err != nil {
		log.Println("Error al ejecutar fragmento formulario_editar:", err)
		http.Error(w, "Error al inyectar fragmento", http.StatusInternalServerError)
	}
}

// Actualizar procesa los cambios y refresca la lista
func (h *PuestoHandler) Actualizar(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id, _ := strconv.Atoi(r.FormValue("id"))

	metaID, _ := strconv.Atoi(r.FormValue("meta_id"))
	fuenteID, _ := strconv.Atoi(r.FormValue("fuente_rubro_id"))
	regimenID, _ := strconv.Atoi(r.FormValue("regimen_id"))
	sueldo, _ := strconv.ParseFloat(r.FormValue("sueldo_presupuestado"), 64)

	var unidadOrganicaID *int
	idStr := r.FormValue("unidad_organica_id")
	if idStr != "" && idStr != "0" {
		idVal, err := strconv.Atoi(idStr)
		if err == nil {
			unidadOrganicaID = &idVal
		}
	}

	var codigoAirhsp *string
	airhspStr := r.FormValue("codigo_airhsp")
	if airhspStr != "" {
		codigoAirhsp = &airhspStr
	}

	puestoActualizado := models.Puesto{
		ID:                  id,
		TenantID:            obtenerTenantID(r),
		MetaID:              metaID,
		FuenteRubroID:       fuenteID,
		RegimenID:           regimenID,
		Nombre:              r.FormValue("nombre"),
		SueldoPresupuestado: sueldo,
		Activo:              r.FormValue("activo") == "on",
		EsDietario:          r.FormValue("es_dietario") == "on",
		UnidadOrganicaID:    unidadOrganicaID,
		CodigoAirhsp:        codigoAirhsp,
	}

	err := h.Repo.Actualizar(&puestoActualizado)
	if err != nil {
		log.Println("Error al actualizar puesto:", err)
		http.Error(w, "Error al actualizar el puesto: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", "refrescarPuestos")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("✅ Plaza actualizada correctamente."))
}

// FormularioCrearUI devuelve el formulario limpio
func (h *PuestoHandler) FormularioCrearUI(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)

	metas, _ := h.MetaRepo.ObtenerTodos(tenantID)
	fuentes, _ := h.FuenteRubroRepo.ObtenerPorAnio(2026, "")
	regimenes, _ := h.Repo.ObtenerRegimenes()

	datos := map[string]interface{}{
		"Metas":     metas,
		"Fuentes":   fuentes,
		"Regimenes": regimenes,
	}

	tmpl, _ := template.ParseFiles("ui/templates/tenant/puestos_ui.html")
	tmpl.ExecuteTemplate(w, "formulario_crear", datos)
}

// AsignarConceptosUI carga el modal con la estructura de conceptos del puesto
func (h *PuestoHandler) AsignarConceptosUI(w http.ResponseWriter, r *http.Request) {
	puestoID, _ := strconv.Atoi(r.URL.Query().Get("id"))
	tenantID := obtenerTenantID(r) // Tu función para obtener el ID de la municipalidad
	// Obtenemos la lista combinada (Conceptos del Tenant + lo que tiene el Puesto)
	asignaciones, err := h.Repo.ObtenerConceptosParaAsignacion(puestoID, tenantID)
	if err != nil {
		log.Println("Error al obtener conceptos para asignación:", err)
		http.Error(w, "Error interno", 500)
		return
	}

	data := map[string]interface{}{
		"PuestoID":     puestoID,
		"Asignaciones": asignaciones,
	}

	// CAPTURAMOS EL ERROR DE LA PLANTILLA
	tmpl, err := template.ParseFiles("ui/templates/tenant/puestos_ui.html")
	if err != nil {
		log.Println("❌ Error al leer puestos_ui.html:", err)
		http.Error(w, "Error de plantilla", 500)
		return
	}

	err = tmpl.ExecuteTemplate(w, "formulario_asignar_conceptos", data)
	if err != nil {
		log.Println("❌ Error al ejecutar el fragmento HTMX:", err)
		http.Error(w, "Error al inyectar fragmento", 500)
	}
}

// GuardarAsignacion procesa el formulario enviado por HTMX
func (h *PuestoHandler) GuardarAsignacion(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	puestoID, _ := strconv.Atoi(r.FormValue("puesto_id"))

	// Leemos qué conceptos fueron marcados (switches encendidos)
	idsSeleccionados := r.Form["concepto_tenant_ids"]

	var listaParaGuardar []models.ConceptoAsignacion
	for _, idStr := range idsSeleccionados {
		id, _ := strconv.Atoi(idStr)

		// Leemos el monto específico para este ID (ej: monto_45)
		montoStr := r.FormValue("monto_" + idStr)
		monto, _ := strconv.ParseFloat(montoStr, 64)

		listaParaGuardar = append(listaParaGuardar, models.ConceptoAsignacion{
			ConceptoTenantID: id,
			Monto:            monto,
			Asignado:         true,
		})
	}

	err := h.Repo.GuardarAsignacionConceptos(puestoID, listaParaGuardar)
	if err != nil {
		log.Println("Error al guardar asignación:", err)
		http.Error(w, "No se pudo guardar la estructura de pago", 500)
		return
	}

	// Si todo sale bien, enviamos la señal de éxito para cerrar el modal
	w.Header().Set("HX-Trigger", "cerrarModalAsignacion")
	w.Write([]byte("✅ Estructura de pago actualizada correctamente."))
}
