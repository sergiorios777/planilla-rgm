package handlers

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
	"strconv"
	"strings"
)

type ConceptoTenantHandler struct {
	Repo         *repository.ConceptoTenantRepository
	PuestoRepo   *repository.PuestoRepository
	PlanillaRepo *repository.PlanillaRepository
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
	regimenIDStr := r.URL.Query().Get("regimen_id")
	estado := r.URL.Query().Get("estado")

	regimenID, _ := strconv.Atoi(regimenIDStr)

	limite, err := strconv.Atoi(limiteStr)
	if err != nil || limite <= 0 {
		limite = 10 // Por defecto mostramos 10
	}

	pagina, err := strconv.Atoi(paginaStr)
	if err != nil || pagina <= 0 {
		pagina = 1 // Por defecto empezamos en la página 1
	}

	offset := (pagina - 1) * limite

	conceptos, totalRegistros, err := h.Repo.ObtenerTodosPaginacion(tenantID, busqueda, regimenID, estado, limite, offset)
	if err != nil {
		http.Error(w, "Error al obtener los conceptos", http.StatusInternalServerError)
		return
	}

	totalPaginas := (totalRegistros + limite - 1) / limite
	if totalPaginas == 0 {
		totalPaginas = 1
	}

	paginacion := models.CalcularPaginacion(
		pagina,
		totalPaginas,
		totalRegistros,
		"/tenant/conceptos-locales/lista",
		"#lista-conceptos-tenant",
		"#form-filtros-conceptos-tenant",
	)

	datosVista := struct {
		Conceptos  []models.ConceptoTenant
		Paginacion models.PaginacionDTO
	}{
		Conceptos:  conceptos,
		Paginacion: paginacion,
	}

	tmpl, _ := template.ParseFiles("ui/templates/tenant/conceptos_tenant_ui.html", "ui/templates/components/paginacion.html")
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

	modalidad := r.FormValue("modalidad_entrega")
	if modalidad == "" {
		if r.FormValue("es_ocasional") == "on" {
			modalidad = models.ModalidadEntregaOcasional
		} else if r.FormValue("es_extraordinario") == "on" {
			modalidad = models.ModalidadEntregaExcepcional
		} else {
			modalidad = models.ModalidadEntregaPermanente
		}
	}

	nuevoConcepto := models.ConceptoTenant{
		TenantID:                 tenantID,
		ConceptoID:               cID,
		NombrePersonalizado:      r.FormValue("nombre_personalizado"),
		FrecuenciaMeses:          r.FormValue("frecuencia_meses"),
		ClasificadorID:           clasifID,
		Activo:                   r.FormValue("activo") == "on",
		EsExtraordinario:         r.FormValue("es_extraordinario") == "on" || modalidad == models.ModalidadEntregaExcepcional,
		EsPensionable:            r.FormValue("es_pensionable") == "on",
		EsRemunerativa:           r.FormValue("es_remunerativa") == "on",
		EsBaseCts:                r.FormValue("es_base_cts") == "on",
		EsBaseBeneficiosSociales: r.FormValue("es_base_beneficios_sociales") == "on",
		EsOcasional:              modalidad == models.ModalidadEntregaOcasional,
		EsAfectoCargasSociales:   r.FormValue("es_afecto_cargas_sociales") == "on",
		ModalidadEntrega:         modalidad,
		BaseCalculoPara:          r.Form["base_calculo_para"],
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

	baseCalculoMarcados := make(map[string]bool)
	for _, b := range c.BaseCalculoPara {
		baseCalculoMarcados[b] = true
	}

	// 3. Enviar todo a la plantilla
	data := map[string]interface{}{
		"Concepto":                   c,
		"ClasificadorIDSeleccionado": clasificadorID,
		"Maestros":                   maestros,
		"Clasificadores":             clasificadores,
		"Regimenes":                  regimenes,
		"RegimenesMarcados":          marcados,
		"BaseCalculoMarcados":        baseCalculoMarcados,
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

	// 1. Obtener el concepto actual de la BD para preservar el campo central es_ocasional
	existente, err := h.Repo.ObtenerPorID(id, tenantID)
	if err != nil {
		log.Println("Error al obtener concepto existente tenant:", err)
		http.Error(w, "Error al recuperar concepto para actualizar", http.StatusInternalServerError)
		return
	}

	modalidadTenant := existente.ModalidadEntrega
	if m := r.FormValue("modalidad_entrega"); m != "" {
		modalidadTenant = m
	}
	if modalidadTenant == "" {
		modalidadTenant = models.ModalidadEntregaPermanente
	}

	editado := models.ConceptoTenant{
		ID:                       id,
		TenantID:                 tenantID,
		ConceptoID:               conceptoID,
		NombrePersonalizado:      nombre,
		FrecuenciaMeses:          frecuencia,
		ClasificadorID:           clasificadorID,
		Activo:                   activo,
		EsExtraordinario:         r.FormValue("es_extraordinario") == "on" || modalidadTenant == models.ModalidadEntregaExcepcional,
		EsPensionable:            r.FormValue("es_pensionable") == "on",
		EsRemunerativa:           r.FormValue("es_remunerativa") == "on",
		EsBaseCts:                r.FormValue("es_base_cts") == "on",
		EsBaseBeneficiosSociales: r.FormValue("es_base_beneficios_sociales") == "on",
		EsOcasional:              modalidadTenant == models.ModalidadEntregaOcasional,
		EsAfectoCargasSociales:   r.FormValue("es_afecto_cargas_sociales") == "on",
		ModalidadEntrega:         modalidadTenant,
		BaseCalculoPara:          r.Form["base_calculo_para"],
		RegimenesIDs:             regimenesIDs,
	}

	// 2. Actualizar en BD
	err = h.Repo.Actualizar(&editado)
	if err != nil {
		log.Println("Error actualizando concepto tenant:", err)
		http.Error(w, "Error al actualizar concepto en base de datos", http.StatusInternalServerError)
		return
	}

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

	badgeModalidad := ""
	switch c.ModalidadEntrega {
	case models.ModalidadEntregaPermanente:
		badgeModalidad = ` <mark style="background-color: #e3f2fd; color: #1565c0; padding: 2px 6px; border-radius: 4px; font-size: 0.65rem; font-weight: bold; border: 1px solid #bbdefb;">PERM</mark>`
	case models.ModalidadEntregaPeriodico:
		badgeModalidad = ` <mark style="background-color: #fff3e0; color: #e65100; padding: 2px 6px; border-radius: 4px; font-size: 0.65rem; font-weight: bold; border: 1px solid #ffe0b2;">PERIÓD</mark>`
	case models.ModalidadEntregaExcepcional:
		badgeModalidad = ` <mark style="background-color: #f3e5f5; color: #6a1b9a; padding: 2px 6px; border-radius: 4px; font-size: 0.65rem; font-weight: bold; border: 1px solid #e1bee7;">EXCEP</mark>`
	case models.ModalidadEntregaOcasional:
		badgeModalidad = ` <mark style="background-color: #eceff1; color: #37474f; padding: 2px 6px; border-radius: 4px; font-size: 0.65rem; font-weight: bold; border: 1px solid #cfd8dc;">OCAS</mark>`
	default:
		if c.EsOcasional {
			badgeModalidad = ` <mark style="background-color: #eceff1; color: #37474f; padding: 2px 6px; border-radius: 4px; font-size: 0.65rem; font-weight: bold; border: 1px solid #cfd8dc;">OCAS</mark>`
		}
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
			` + badgeTipo + badgeModalidad + `
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

	err = h.Repo.ClonarReglasFinanciamientoModelo(tenantID)
	if err != nil {
		log.Println("⚠️ Advertencia: Error al clonar reglas de financiamiento modelo:", err)
	}

	// Refrescamos la lista para que el usuario vea los cambios
	h.Listar(w, r)
}

func (h *ConceptoTenantHandler) ModalAgregarModeloUI(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	disponibles, err := h.Repo.ObtenerModelosDisponibles(tenantID)
	if err != nil {
		log.Println("Error obteniendo modelos disponibles:", err)
		http.Error(w, "Error al obtener conceptos modelo disponibles", http.StatusInternalServerError)
		return
	}

	datos := map[string]interface{}{
		"ModelosDisponibles": disponibles,
	}

	tmpl, err := template.ParseFiles("ui/templates/tenant/conceptos_tenant_ui.html")
	if err != nil {
		log.Println("Error parseando plantilla:", err)
		http.Error(w, "Error al cargar plantilla", http.StatusInternalServerError)
		return
	}

	tmpl.ExecuteTemplate(w, "modal_agregar_modelo", datos)
}

func (h *ConceptoTenantHandler) AgregarModelo(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	tenantID := obtenerTenantID(r)
	modeloID, err := strconv.Atoi(r.FormValue("modelo_id"))
	if err != nil || modeloID <= 0 {
		http.Error(w, "Debe seleccionar un concepto modelo válido", http.StatusBadRequest)
		return
	}

	err = h.Repo.AgregarDesdeModelo(tenantID, modeloID)
	if err != nil {
		log.Println("Error al agregar concepto modelo a tenant:", err)
		http.Error(w, "No se pudo agregar el concepto modelo", http.StatusInternalServerError)
		return
	}

	// Refrescar la tabla principal
	h.Listar(w, r)
}

func (h *ConceptoTenantHandler) SincronizarModelo(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	if id <= 0 {
		id, _ = strconv.Atoi(r.FormValue("id"))
	}
	tenantID := obtenerTenantID(r)

	err := h.Repo.SincronizarConceptoModelo(tenantID, id)
	if err != nil {
		log.Println("Error al sincronizar concepto tenant desde modelo:", err)
		http.Error(w, "Error al sincronizar el concepto con el modelo", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", "recargarTablaConceptos")

	r.URL.RawQuery = "id=" + strconv.Itoa(id)
	h.EditarUI(w, r)
}

// ReglasModal renderiza el modal HTMX con las reglas de financiamiento de un concepto tenant
func (h *ConceptoTenantHandler) ReglasModal(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	conceptoTenantID, _ := strconv.Atoi(r.URL.Query().Get("concepto_tenant_id"))
	if conceptoTenantID <= 0 {
		conceptoTenantID, _ = strconv.Atoi(r.FormValue("concepto_tenant_id"))
	}
	if conceptoTenantID <= 0 {
		http.Error(w, "ID de concepto tenant no válido", http.StatusBadRequest)
		return
	}

	concepto, err := h.Repo.ObtenerPorID(conceptoTenantID, tenantID)
	if err != nil {
		http.Error(w, "No se encontró el concepto local: "+err.Error(), http.StatusNotFound)
		return
	}

	var reglas []models.ReglaFinanciamientoConcepto
	var metas []models.MetaPresupuestal
	var fuentesRubros []models.FuenteRubro

	if h.PlanillaRepo != nil {
		reglas, _ = h.PlanillaRepo.ObtenerReglasFinanciamientoPorConceptoID(r.Context(), conceptoTenantID, tenantID)
		metas, _ = h.PlanillaRepo.ObtenerMetas(tenantID)
		fuentesRubros, _ = h.PlanillaRepo.ObtenerFuentesRubros()
	}
	regimenes, _ := h.PuestoRepo.ObtenerRegimenes()

	datos := map[string]interface{}{
		"Concepto":      concepto,
		"Reglas":        reglas,
		"Regimenes":     regimenes,
		"Metas":          metas,
		"FuentesRubros": fuentesRubros,
	}

	tmpl, err := template.ParseFiles("ui/templates/tenant/conceptos_tenant_ui.html")
	if err != nil {
		http.Error(w, "Error al cargar plantilla: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmpl.ExecuteTemplate(w, "modal_reglas_tenant_content", datos)
}

// CrearReglaHTMX crea una nueva regla de financiamiento local para un concepto tenant
func (h *ConceptoTenantHandler) CrearReglaHTMX(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	tenantID := obtenerTenantID(r)
	conceptoTenantID, _ := strconv.Atoi(r.FormValue("concepto_tenant_id"))

	var regimenID, metaID, fuenteRubroID *int
	if regVal := r.FormValue("regimen_id"); regVal != "" {
		if id, err := strconv.Atoi(regVal); err == nil && id > 0 {
			regimenID = &id
		}
	}
	if metaVal := r.FormValue("meta_id"); metaVal != "" {
		if id, err := strconv.Atoi(metaVal); err == nil && id > 0 {
			metaID = &id
		}
	}
	if rubVal := r.FormValue("fuente_rubro_id"); rubVal != "" {
		if id, err := strconv.Atoi(rubVal); err == nil && id > 0 {
			fuenteRubroID = &id
		}
	}
	activo := r.FormValue("activo") == "true" || r.FormValue("activo") == "on"

	if conceptoTenantID > 0 && h.PlanillaRepo != nil {
		regla := models.ReglaFinanciamientoConcepto{
			TenantID:         tenantID,
			ConceptoTenantID: conceptoTenantID,
			RegimenID:        regimenID,
			MetaID:           metaID,
			FuenteRubroID:    fuenteRubroID,
			Activo:           activo,
		}
		_ = h.PlanillaRepo.CrearReglaFinanciamiento(r.Context(), &regla)
	}

	r.URL.RawQuery = fmt.Sprintf("concepto_tenant_id=%d", conceptoTenantID)
	h.ReglasModal(w, r)
}

// EliminarReglaHTMX elimina una regla de financiamiento local
func (h *ConceptoTenantHandler) EliminarReglaHTMX(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	conceptoTenantID, _ := strconv.Atoi(r.URL.Query().Get("concepto_tenant_id"))

	if id > 0 && h.PlanillaRepo != nil {
		_ = h.PlanillaRepo.EliminarReglaFinanciamiento(r.Context(), id, tenantID)
	}

	r.URL.RawQuery = fmt.Sprintf("concepto_tenant_id=%d", conceptoTenantID)
	h.ReglasModal(w, r)
}

// SincronizarReglasModeloHTMX re-sincroniza las reglas de financiamiento desde el modelo SaaS para un concepto tenant
func (h *ConceptoTenantHandler) SincronizarReglasModeloHTMX(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	conceptoTenantID, _ := strconv.Atoi(r.URL.Query().Get("concepto_tenant_id"))
	if conceptoTenantID <= 0 {
		conceptoTenantID, _ = strconv.Atoi(r.FormValue("concepto_tenant_id"))
	}

	if conceptoTenantID <= 0 {
		http.Error(w, "ID de concepto no válido", http.StatusBadRequest)
		return
	}

	err := h.Repo.SincronizarReglasFinanciamientoDesdeModelo(tenantID, conceptoTenantID)
	if err != nil {
		log.Println("Error al sincronizar reglas desde modelo:", err)
		http.Error(w, "Error al sincronizar reglas desde el modelo SaaS: "+err.Error(), http.StatusInternalServerError)
		return
	}

	r.URL.RawQuery = fmt.Sprintf("concepto_tenant_id=%d", conceptoTenantID)
	h.ReglasModal(w, r)
}

// ActualizarReglaHTMX actualiza una regla de financiamiento local (Meta / Fuente-Rubro / Régimen)
func (h *ConceptoTenantHandler) ActualizarReglaHTMX(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error al procesar datos", http.StatusBadRequest)
		return
	}

	tenantID := obtenerTenantID(r)
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	if id <= 0 {
		id, _ = strconv.Atoi(r.FormValue("id"))
	}
	conceptoTenantID, _ := strconv.Atoi(r.URL.Query().Get("concepto_tenant_id"))
	if conceptoTenantID <= 0 {
		conceptoTenantID, _ = strconv.Atoi(r.FormValue("concepto_tenant_id"))
	}

	if id > 0 && h.PlanillaRepo != nil {
		regla, err := h.PlanillaRepo.ObtenerReglaFinanciamientoPorID(r.Context(), id, tenantID)
		if err == nil && regla != nil {
			if r.Form.Has("meta_id") {
				metaVal := r.FormValue("meta_id")
				if metaVal == "" || metaVal == "0" {
					regla.MetaID = nil
				} else if mID, err := strconv.Atoi(metaVal); err == nil && mID > 0 {
					regla.MetaID = &mID
				}
			}

			if r.Form.Has("fuente_rubro_id") {
				rubVal := r.FormValue("fuente_rubro_id")
				if rubVal == "" || rubVal == "0" {
					regla.FuenteRubroID = nil
				} else if rID, err := strconv.Atoi(rubVal); err == nil && rID > 0 {
					regla.FuenteRubroID = &rID
				}
			}

			if r.Form.Has("regimen_id") {
				regVal := r.FormValue("regimen_id")
				if regVal == "" || regVal == "0" {
					regla.RegimenID = nil
				} else if rgID, err := strconv.Atoi(regVal); err == nil && rgID > 0 {
					regla.RegimenID = &rgID
				}
			}

			_ = h.PlanillaRepo.ActualizarReglaFinanciamiento(r.Context(), regla)
		}
	}

	r.URL.RawQuery = fmt.Sprintf("concepto_tenant_id=%d", conceptoTenantID)
	h.ReglasModal(w, r)
}


