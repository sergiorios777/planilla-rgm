package handlers

import (
	"html/template"
	"log"
	"net/http"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
	"strconv"
	"strings"
)

type ConceptoTenantHandler struct {
	Repo       *repository.ConceptoTenantRepository
	PuestoRepo *repository.PuestoRepository
}

func (h *ConceptoTenantHandler) VistaUI(w http.ResponseWriter, r *http.Request) {
	maestros, _ := h.Repo.ObtenerMaestros()
	clasificadores, _ := h.Repo.ObtenerClasificadores()
	regimenes, _ := h.PuestoRepo.ObtenerRegimenes()

	datos := map[string]interface{}{
		"Maestros":       maestros,
		"Clasificadores": clasificadores,
		"Regimenes":      regimenes,
	}

	tmpl, _ := template.ParseFiles("ui/templates/tenant/conceptos_tenant_ui.html")
	tmpl.Execute(w, datos)
}

func (h *ConceptoTenantHandler) Listar(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	busqueda := r.URL.Query().Get("buscar")
	paginaStr := r.URL.Query().Get("pagina")
	limiteStr := r.URL.Query().Get("limite")

	limite, err := strconv.Atoi(limiteStr)
	if err != nil || limite <= 0 {
		limite = 10 // Por defecto mostramos 10
	}

	pagina, err := strconv.Atoi(paginaStr)
	if err != nil || pagina <= 0 {
		pagina = 1 // Por defecto empezamos en la página 1
	}

	offset := (pagina - 1) * limite

	conceptos, totalRegistros, err := h.Repo.ObtenerTodosPaginacion(tenantID, busqueda, limite, offset)
	if err != nil {
		http.Error(w, "Error al obtener los conceptos", http.StatusInternalServerError)
		return
	}

	totalPaginas := (totalRegistros + limite - 1) / limite

	if totalPaginas == 0 {
		totalPaginas = 1
	}

	datosVista := struct {
		Conceptos       []models.ConceptoTenant
		TotalPaginas    int
		PaginaActual    int
		PaginaAnterior  int
		PaginaSiguiente int
	}{
		Conceptos:       conceptos,
		TotalPaginas:    totalPaginas,
		PaginaActual:    pagina,
		PaginaAnterior:  pagina - 1,
		PaginaSiguiente: pagina + 1,
	}

	tmpl, _ := template.ParseFiles("ui/templates/tenant/conceptos_tenant_ui.html")
	tmpl.ExecuteTemplate(w, "tabla_conceptos_tenant", datosVista)
}

func (h *ConceptoTenantHandler) Crear(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	tenantID := obtenerTenantID(r)
	cID, _ := strconv.Atoi(r.FormValue("concepto_id"))

	clasifStr := r.FormValue("clasificador_id")
	var clasifID *int
	if strings.TrimSpace(clasifStr) != "" {
		idParsed, err := strconv.Atoi(clasifStr)
		if err == nil {
			clasifID = &idParsed
		}
	}

	var regimenesIDs []int
	for _, idStr := range r.Form["regimenes_ids"] {
		idParsed, err := strconv.Atoi(idStr)
		if err == nil {
			regimenesIDs = append(regimenesIDs, idParsed)
		}
	}

	nuevoConcepto := models.ConceptoTenant{
		TenantID:                 tenantID,
		ConceptoID:               cID,
		NombrePersonalizado:      r.FormValue("nombre_personalizado"),
		FrecuenciaMeses:          r.FormValue("frecuencia_meses"),
		ClasificadorID:           clasifID,
		Activo:                   r.FormValue("activo") == "on",
		EsExtraordinario:         r.FormValue("es_extraordinario") == "on",
		EsPensionable:            r.FormValue("es_pensionable") == "on",
		EsRemunerativa:           r.FormValue("es_remunerativa") == "on",
		EsBaseCts:                r.FormValue("es_base_cts") == "on",
		EsBaseBeneficiosSociales: r.FormValue("es_base_beneficios_sociales") == "on",
		RegimenesIDs:             regimenesIDs,
	}

	err := h.Repo.Crear(&nuevoConcepto)
	if err != nil {
		// Validamos si el error es por la restricción UNIQUE de la base de datos
		if strings.Contains(err.Error(), "unique_nombre_concepto_tenant") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`
				<div id="alerta-concepto" hx-swap-oob="true">
					<article style="background-color: #ffcdd2; color: #b71c1c; padding: 1rem; margin-bottom: 1rem; border-radius: 5px;">
						❌ Error: Ya existe un concepto con ese "Nombre Personalizado". Por favor, usa un nombre distinto (Ej. "Sueldo Base CAS" en lugar de "Sueldo Base").
					</article>
				</div>
			`))
			return
		}
		http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
		return
	}

	// Limpiamos alertas previas y actualizamos la tabla
	w.Write([]byte(`<div id="alerta-concepto" hx-swap-oob="true"></div>`))
	h.Listar(w, r)
}

// EditarUI carga el formulario de edición en el contenedor principal
func (h *ConceptoTenantHandler) EditarUI(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	tenantID := obtenerTenantID(r)

	// 1. Traer el concepto a editar
	c, err := h.Repo.ObtenerPorID(id, tenantID)
	if err != nil {
		log.Println("Error obteniendo concepto", err)
		return
	}

	// Extraer el valor del puntero clasificador_id *int
	var clasificadorID int
	if c.ClasificadorID != nil {
		clasificadorID = *c.ClasificadorID
	}

	// 2. Traer las listas para los <select> (USA TUS FUNCIONES EXISTENTES AQUÍ)
	maestros, _ := h.Repo.ObtenerMaestros()
	clasificadores, _ := h.Repo.ObtenerClasificadores()
	regimenes, _ := h.PuestoRepo.ObtenerRegimenes()

	regimenesIDs, err := h.Repo.ObtenerRegimenesPorConcepto(c.ID, tenantID)
	if err != nil {
		log.Println("Error al obtener regímenes del concepto local:", err)
	}

	marcados := make(map[int]bool)
	for _, rid := range regimenesIDs {
		marcados[rid] = true
	}

	// 3. Enviar todo a la plantilla
	data := map[string]interface{}{
		"Concepto":                   c,
		"ClasificadorIDSeleccionado": clasificadorID,
		"Maestros":                   maestros,
		"Clasificadores":             clasificadores,
		"Regimenes":                  regimenes,
		"RegimenesMarcados":          marcados,
	}

	tmpl, err := template.ParseFiles("ui/templates/tenant/conceptos_tenant_ui.html")
	if err != nil {
		log.Println("Error parseando plantilla:", err)
		return
	}

	err = tmpl.ExecuteTemplate(w, "formulario_editar", data)
	if err != nil {
		log.Println("Error ejecutando plantilla:", err)
		return
	}
}

// Actualizar guarda los cambios, refresca la tabla y resetea el formulario
func (h *ConceptoTenantHandler) Actualizar(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id, _ := strconv.Atoi(r.FormValue("id"))
	tenantID := obtenerTenantID(r)

	conceptoID, _ := strconv.Atoi(r.FormValue("concepto_id"))

	// Manejo del clasificador nulo/vacío
	var clasificadorID *int
	if classF := r.FormValue("clasificador_id"); classF != "" {
		val, _ := strconv.Atoi(classF)
		clasificadorID = &val
	}

	nombre := r.FormValue("nombre_personalizado")
	frecuencia := r.FormValue("frecuencia_meses")
	activo := r.FormValue("activo") == "on"

	var regimenesIDs []int
	for _, idStr := range r.Form["regimenes_ids"] {
		idParsed, err := strconv.Atoi(idStr)
		if err == nil {
			regimenesIDs = append(regimenesIDs, idParsed)
		}
	}

	editado := models.ConceptoTenant{
		ID:                       id,
		TenantID:                 tenantID,
		ConceptoID:               conceptoID,
		NombrePersonalizado:      nombre,
		FrecuenciaMeses:          frecuencia,
		ClasificadorID:           clasificadorID,
		Activo:                   activo,
		EsExtraordinario:         r.FormValue("es_extraordinario") == "on",
		EsPensionable:            r.FormValue("es_pensionable") == "on",
		EsRemunerativa:           r.FormValue("es_remunerativa") == "on",
		EsBaseCts:                r.FormValue("es_base_cts") == "on",
		EsBaseBeneficiosSociales: r.FormValue("es_base_beneficios_sociales") == "on",
		RegimenesIDs:             regimenesIDs,
	}

	// 1. Actualizar en BD
	err := h.Repo.Actualizar(&editado)
	if err != nil {
		log.Println("Error actualizando concepto tenant:", err)
		http.Error(w, "Error al actualizar concepto en base de datos", http.StatusInternalServerError)
		return
	}

	// 2. Le decimos a HTMX: "Por favor, dispara el evento que recarga la tabla"
	w.Header().Set("HX-Trigger", "recargarTablaConceptos")

	// 3. Devolvemos el formulario de CREAR para dejar la pantalla limpia
	h.FormularioCrearUI(w, r)
}

// FormularioCrearUI devuelve el fragmento HTML del formulario de creación limpio
func (h *ConceptoTenantHandler) FormularioCrearUI(w http.ResponseWriter, r *http.Request) {
	// tenantID := obtenerTenantID(r)

	// 1. Traer los datos para los selectores (Usa tus nombres de función reales)
	maestros, _ := h.Repo.ObtenerMaestros()
	clasificadores, _ := h.Repo.ObtenerClasificadores()
	regimenes, _ := h.PuestoRepo.ObtenerRegimenes()

	data := map[string]interface{}{
		"Maestros":       maestros,
		"Clasificadores": clasificadores,
		"Regimenes":      regimenes,
	}

	// 2. Renderizar solo el bloque "formulario_crear" definido en tu HTML
	tmpl, _ := template.ParseFiles("ui/templates/tenant/conceptos_tenant_ui.html")
	tmpl.ExecuteTemplate(w, "formulario_crear", data)
}

// 3. FilaUI devuelve el fragmento HTML de solo lectura
func (h *ConceptoTenantHandler) FilaUI(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	tenantID := obtenerTenantID(r)

	c, _ := h.Repo.ObtenerPorID(id, tenantID)

	badgeTipo := `<mark style="background-color: #bbdefb; color: #1565c0; padding: 2px 6px; border-radius: 4px; font-size: 0.7rem;">APORTE</mark>`
	switch c.ConceptoTipo {
	case "INGRESO":
		badgeTipo = `<mark style="background-color: #c8e6c9; color: #1b5e20; padding: 2px 6px; border-radius: 4px; font-size: 0.7rem;">INGRESO</mark>`
	case "RETENCION":
		badgeTipo = `<mark style="background-color: #ffcdd2; color: #b71c1c; padding: 2px 6px; border-radius: 4px; font-size: 0.7rem;">RETENCIÓN</mark>`
	}

	textoEstado := `<span style="color: #b71c1c; font-size: 0.8rem; font-weight: bold;">Inactivo</span>`
	if c.Activo {
		textoEstado = `<span style="color: #1b5e20; font-size: 0.8rem; font-weight: bold;">Activo</span>`
	}

	siaf := c.ClasificadorCodigo
	if siaf == "" {
		siaf = "N/A"
	}

	html := `
	<tr id="concepto-` + strconv.Itoa(c.ID) + `">
		<td>
			<strong>` + c.NombrePersonalizado + `</strong><br>
			` + badgeTipo + `
		</td>
		<td><small>Cód: ` + c.ConceptoCodigo + `</small></td>
		<td>
			<small>Meses: ` + c.FrecuenciaMeses + `</small><br>
			<small>SIAF: <strong>` + siaf + `</strong></small>
		</td>
		<td style="text-align: right;">
			<div style="display: flex; gap: 5px; justify-content: flex-end; align-items: center;">
				` + textoEstado + `
				<button class="outline secondary" style="padding: 2px 10px; margin-bottom: 0; font-size: 0.8rem;"
				        hx-get="/tenant/conceptos-locales/editar-ui?id=` + strconv.Itoa(c.ID) + `" hx-target="closest tr" hx-swap="outerHTML">
					✏️
				</button>
			</div>
		</td>
	</tr>
	`
	w.Write([]byte(html))
}

func (h *ConceptoTenantHandler) Restaurar(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r) // Tu helper para obtener el ID de la sesión/contexto

	err := h.Repo.ClonarDesdeModelo(tenantID)
	if err != nil {
		http.Error(w, "No se pudieron restaurar los conceptos", http.StatusInternalServerError)
		return
	}

	err = h.Repo.ClonarRelacionesRegimen(tenantID)
	if err != nil {
		log.Println("⚠️ Advertencia: Error al clonar relaciones de régimen:", err)
		http.Error(w, "Error parcial: conceptos restaurados pero fallaron relaciones", http.StatusInternalServerError)
		return
	}

	// Refrescamos la lista para que el usuario vea los cambios
	h.Listar(w, r)
}
