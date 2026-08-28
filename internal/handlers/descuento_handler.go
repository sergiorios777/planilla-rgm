package handlers

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
	"strconv"
	"strings"
	"time"
)

type DescuentoHandler struct {
	Repo                  *repository.DescuentoRepository
	TrabajadorRepo        *repository.TrabajadorRepository
	EntidadFinancieraRepo *repository.EntidadFinancieraRepository
}

func NewDescuentoHandler(db *sql.DB) *DescuentoHandler {
	return &DescuentoHandler{
		Repo:                  repository.NewDescuentoRepository(db),
		TrabajadorRepo:        repository.NewTrabajadorRepository(db),
		EntidadFinancieraRepo: repository.NewEntidadFinancieraRepository(db),
	}
}

// VistaUI renderiza la página principal de descuentos
func (h *DescuentoHandler) VistaUI(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	if tenantID == 0 {
		http.Error(w, "Acceso no autorizado", http.StatusUnauthorized)
		return
	}

	filtro := models.DescuentoFiltroDTO{
		Estado: "TODOS",
	}

	limite := 15
	pagina := 1
	offset := 0

	descuentos, totalRegistros, err := h.Repo.ListarPaginado(tenantID, filtro, limite, offset)
	if err != nil {
		log.Println("Error listando descuentos:", err)
		http.Error(w, "Error al listar descuentos", http.StatusInternalServerError)
		return
	}

	kpi, _ := h.Repo.ObtenerKPIs(tenantID)

	totalPaginas := (totalRegistros + limite - 1) / limite
	if totalPaginas == 0 {
		totalPaginas = 1
	}

	paginacion := models.CalcularPaginacion(
		pagina,
		totalPaginas,
		totalRegistros,
		"/tenant/descuentos/lista",
		"#lista-descuentos",
		"#form-filtros-descuentos",
	)

	datos := map[string]interface{}{
		"Descuentos": descuentos,
		"KPI":        kpi,
		"Paginacion": paginacion,
		"Filtro":     filtro,
	}

	tmpl, err := template.ParseFiles("ui/templates/tenant/descuentos_ui.html", "ui/templates/components/paginacion.html")
	if err != nil {
		log.Println("Error parseando descuentos_ui.html:", err)
		http.Error(w, "Error cargando plantilla: "+err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, datos)
}

// ListarHTMX maneja las búsquedas y paginación reactiva vía HTMX
func (h *DescuentoHandler) ListarHTMX(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	if tenantID == 0 {
		http.Error(w, "No autorizado", http.StatusUnauthorized)
		return
	}

	r.ParseForm()
	busqueda := r.FormValue("buscar")
	tipo := r.FormValue("tipo_descuento")
	estado := r.FormValue("estado")
	if estado == "" {
		estado = "TODOS"
	}

	paginaStr := r.FormValue("pagina")
	pagina, err := strconv.Atoi(paginaStr)
	if err != nil || pagina <= 0 {
		pagina = 1
	}
	limite := 15
	offset := (pagina - 1) * limite

	filtro := models.DescuentoFiltroDTO{
		Busqueda:      busqueda,
		TipoDescuento: tipo,
		Estado:        estado,
	}

	descuentos, totalRegistros, err := h.Repo.ListarPaginado(tenantID, filtro, limite, offset)
	if err != nil {
		log.Println("Error en ListarHTMX descuentos:", err)
		http.Error(w, "Error consultando descuentos", http.StatusInternalServerError)
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
		"/tenant/descuentos/lista",
		"#lista-descuentos",
		"#form-filtros-descuentos",
	)

	kpi, _ := h.Repo.ObtenerKPIs(tenantID)

	datos := map[string]interface{}{
		"Descuentos": descuentos,
		"KPI":        kpi,
		"Paginacion": paginacion,
	}

	tmpl, err := template.ParseFiles("ui/templates/tenant/descuentos_ui.html", "ui/templates/components/paginacion.html")
	if err != nil {
		http.Error(w, "Error de plantilla: "+err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "tabla_descuentos", datos)
}

// FormularioNuevoUI renderiza la vista de página completa para registrar un nuevo descuento
func (h *DescuentoHandler) FormularioNuevoUI(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	if tenantID == 0 {
		http.Error(w, "No autorizado", http.StatusUnauthorized)
		return
	}

	trabajadores, _ := h.TrabajadorRepo.ObtenerTodos(tenantID)
	conceptosRetencion, _ := h.Repo.ObtenerConceptosRetencionTenant(tenantID)
	entidades, _ := h.EntidadFinancieraRepo.ListarTodas()
	conceptosIngreso, _ := h.Repo.ObtenerConceptosIngresoPorTrabajador(tenantID, 0)

	datos := map[string]interface{}{
		"Trabajadores":       trabajadores,
		"ConceptosRetencion": conceptosRetencion,
		"Entidades":          entidades,
		"ConceptosIngreso":   conceptosIngreso,
		"FechaHoy":           time.Now().Format("2006-01-02"),
	}

	tmpl, err := template.ParseFiles("ui/templates/tenant/descuento_crear_ui.html")
	if err != nil {
		log.Println("Error cargando descuento_crear_ui.html:", err)
		http.Error(w, "Error de plantilla: "+err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, datos)
}

// ModalCrear abre el formulario para registrar un nuevo descuento (fallback)
func (h *DescuentoHandler) ModalCrear(w http.ResponseWriter, r *http.Request) {
	h.FormularioNuevoUI(w, r)
}

// Crear procesa la inserción del descuento y sus conceptos afectos
func (h *DescuentoHandler) Crear(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	if tenantID == 0 {
		http.Error(w, "No autorizado", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Formulario inválido", http.StatusBadRequest)
		return
	}

	trabajadorID, _ := strconv.Atoi(r.FormValue("trabajador_id"))
	conceptoTenantID, _ := strconv.Atoi(r.FormValue("concepto_tenant_id"))
	tipoDescuento := r.FormValue("tipo_descuento")
	documentoOrdenador := r.FormValue("documento_ordenador")
	detalleDocumento := strings.TrimSpace(r.FormValue("detalle_documento"))
	descripcion := strings.TrimSpace(r.FormValue("descripcion"))
	tipoCalculo := r.FormValue("tipo_calculo")
	baseCalculo := r.FormValue("base_calculo")

	porcentaje, _ := strconv.ParseFloat(r.FormValue("porcentaje"), 64)
	montoFijo, _ := strconv.ParseFloat(r.FormValue("monto_fijo"), 64)
	montoTotalDeuda, _ := strconv.ParseFloat(r.FormValue("monto_total_deuda"), 64)
	cuotasTotales, _ := strconv.Atoi(r.FormValue("cuotas_totales"))

	inicioVigenciaStr := r.FormValue("inicio_vigencia")
	finVigenciaStr := r.FormValue("fin_vigencia")

	inicioVigencia, err := time.Parse("2006-01-02", inicioVigenciaStr)
	if err != nil {
		inicioVigencia = time.Now()
	}

	var finVigencia *time.Time
	if finVigenciaStr != "" {
		if fv, err := time.Parse("2006-01-02", finVigenciaStr); err == nil {
			finVigencia = &fv
		}
	}

	// Beneficiario
	benTipoDoc := r.FormValue("beneficiario_tipo_documento")
	benNumDoc := strings.TrimSpace(r.FormValue("beneficiario_numero_documento"))
	benNombre := strings.TrimSpace(r.FormValue("beneficiario_nombre"))
	benCuenta := strings.TrimSpace(r.FormValue("beneficiario_cuenta"))
	benCCI := strings.TrimSpace(r.FormValue("beneficiario_cci"))

	var entFinID *int
	if efStr := r.FormValue("entidad_financiera_id"); efStr != "" && efStr != "0" {
		if idVal, err := strconv.Atoi(efStr); err == nil && idVal > 0 {
			entFinID = &idVal
		}
	}

	// Conceptos afectos
	var conceptosIDs []int
	for _, cidStr := range r.Form["conceptos_afectos_ids"] {
		if cid, err := strconv.Atoi(cidStr); err == nil && cid > 0 {
			conceptosIDs = append(conceptosIDs, cid)
		}
	}

	d := models.Descuento{
		TenantID:                    tenantID,
		TrabajadorID:                trabajadorID,
		ConceptoTenantID:            conceptoTenantID,
		TipoDescuento:               tipoDescuento,
		DocumentoOrdenador:          documentoOrdenador,
		DetalleDocumento:            detalleDocumento,
		Descripcion:                 descripcion,
		TipoCalculo:                 tipoCalculo,
		BaseCalculo:                 baseCalculo,
		Porcentaje:                  porcentaje,
		MontoFijo:                   montoFijo,
		MontoTotalDeuda:             montoTotalDeuda,
		CuotasTotales:               cuotasTotales,
		InicioVigencia:              inicioVigencia,
		FinVigencia:                 finVigencia,
		Activo:                      true,
		BeneficiarioTipoDocumento:   benTipoDoc,
		BeneficiarioNumeroDocumento: benNumDoc,
		BeneficiarioNombre:          benNombre,
		EntidadFinancieraID:         entFinID,
		BeneficiarioCuenta:          benCuenta,
		BeneficiarioCCI:             benCCI,
	}

	_, err = h.Repo.Crear(&d, conceptosIDs)
	if err != nil {
		log.Println("Error creando descuento:", err)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf(`<div id="form-descuento-error"><article style="background-color: #ffcdd2; color: #b71c1c; padding: 1rem; margin-bottom: 1rem; border-radius: 5px;">❌ Error: %s</article></div>`, err.Error())))
		return
	}

	// Redirigir a la vista principal de descuentos
	h.VistaUI(w, r)
}

// FormularioEditarUI renderiza la vista de página completa para editar un descuento
func (h *DescuentoHandler) FormularioEditarUI(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))

	descuento, err := h.Repo.ObtenerPorID(id, tenantID)
	if err != nil || descuento == nil {
		http.Error(w, "Descuento no encontrado", http.StatusNotFound)
		return
	}

	trabajadores, _ := h.TrabajadorRepo.ObtenerTodos(tenantID)
	conceptosRetencion, _ := h.Repo.ObtenerConceptosRetencionTenant(tenantID)
	entidades, _ := h.EntidadFinancieraRepo.ListarTodas()
	infoTrabajador, _ := h.Repo.ObtenerInfoTrabajadorPuesto(tenantID, descuento.TrabajadorID)
	if infoTrabajador != nil {
		mapaAfectos := make(map[int]bool)
		for _, aid := range descuento.ConceptosAfectosIDs {
			mapaAfectos[aid] = true
		}
		for i := range infoTrabajador.Conceptos {
			if mapaAfectos[infoTrabajador.Conceptos[i].ConceptoTenantID] {
				infoTrabajador.Conceptos[i].Seleccionado = true
			}
		}
	}

	datos := map[string]interface{}{
		"Descuento":          descuento,
		"InfoTrabajador":     infoTrabajador,
		"Trabajadores":       trabajadores,
		"ConceptosRetencion": conceptosRetencion,
		"Entidades":          entidades,
	}

	tmpl, err := template.ParseFiles("ui/templates/tenant/descuento_editar_ui.html")
	if err != nil {
		log.Println("Error cargando descuento_editar_ui.html:", err)
		http.Error(w, "Error de plantilla: "+err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, datos)
}

// ModalEditar abre el formulario para editar un descuento (fallback)
func (h *DescuentoHandler) ModalEditar(w http.ResponseWriter, r *http.Request) {
	h.FormularioEditarUI(w, r)
}

// Actualizar guarda las modificaciones de un descuento
func (h *DescuentoHandler) Actualizar(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	if tenantID == 0 {
		http.Error(w, "No autorizado", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Formulario inválido", http.StatusBadRequest)
		return
	}

	id, _ := strconv.Atoi(r.FormValue("id"))
	conceptoTenantID, _ := strconv.Atoi(r.FormValue("concepto_tenant_id"))
	tipoDescuento := r.FormValue("tipo_descuento")
	documentoOrdenador := r.FormValue("documento_ordenador")
	detalleDocumento := strings.TrimSpace(r.FormValue("detalle_documento"))
	descripcion := strings.TrimSpace(r.FormValue("descripcion"))
	tipoCalculo := r.FormValue("tipo_calculo")
	baseCalculo := r.FormValue("base_calculo")

	porcentaje, _ := strconv.ParseFloat(r.FormValue("porcentaje"), 64)
	montoFijo, _ := strconv.ParseFloat(r.FormValue("monto_fijo"), 64)
	montoTotalDeuda, _ := strconv.ParseFloat(r.FormValue("monto_total_deuda"), 64)
	montoAcumulado, _ := strconv.ParseFloat(r.FormValue("monto_acumulado"), 64)
	cuotasTotales, _ := strconv.Atoi(r.FormValue("cuotas_totales"))
	cuotaActual, _ := strconv.Atoi(r.FormValue("cuota_actual"))

	inicioVigenciaStr := r.FormValue("inicio_vigencia")
	finVigenciaStr := r.FormValue("fin_vigencia")

	inicioVigencia, err := time.Parse("2006-01-02", inicioVigenciaStr)
	if err != nil {
		inicioVigencia = time.Now()
	}

	var finVigencia *time.Time
	if finVigenciaStr != "" {
		if fv, err := time.Parse("2006-01-02", finVigenciaStr); err == nil {
			finVigencia = &fv
		}
	}

	activo := r.FormValue("activo") == "true" || r.FormValue("activo") == "on" || r.FormValue("activo") == "1"
	motivoBaja := strings.TrimSpace(r.FormValue("motivo_baja"))

	benTipoDoc := r.FormValue("beneficiario_tipo_documento")
	benNumDoc := strings.TrimSpace(r.FormValue("beneficiario_numero_documento"))
	benNombre := strings.TrimSpace(r.FormValue("beneficiario_nombre"))
	benCuenta := strings.TrimSpace(r.FormValue("beneficiario_cuenta"))
	benCCI := strings.TrimSpace(r.FormValue("beneficiario_cci"))

	var entFinID *int
	if efStr := r.FormValue("entidad_financiera_id"); efStr != "" && efStr != "0" {
		if idVal, err := strconv.Atoi(efStr); err == nil && idVal > 0 {
			entFinID = &idVal
		}
	}

	var conceptosIDs []int
	for _, cidStr := range r.Form["conceptos_afectos_ids"] {
		if cid, err := strconv.Atoi(cidStr); err == nil && cid > 0 {
			conceptosIDs = append(conceptosIDs, cid)
		}
	}

	d := models.Descuento{
		ID:                          id,
		TenantID:                    tenantID,
		ConceptoTenantID:            conceptoTenantID,
		TipoDescuento:               tipoDescuento,
		DocumentoOrdenador:          documentoOrdenador,
		DetalleDocumento:            detalleDocumento,
		Descripcion:                 descripcion,
		TipoCalculo:                 tipoCalculo,
		BaseCalculo:                 baseCalculo,
		Porcentaje:                  porcentaje,
		MontoFijo:                   montoFijo,
		MontoTotalDeuda:             montoTotalDeuda,
		MontoAcumulado:              montoAcumulado,
		CuotasTotales:               cuotasTotales,
		CuotaActual:                 cuotaActual,
		InicioVigencia:              inicioVigencia,
		FinVigencia:                 finVigencia,
		Activo:                      activo,
		MotivoBaja:                  motivoBaja,
		BeneficiarioTipoDocumento:   benTipoDoc,
		BeneficiarioNumeroDocumento: benNumDoc,
		BeneficiarioNombre:          benNombre,
		EntidadFinancieraID:         entFinID,
		BeneficiarioCuenta:          benCuenta,
		BeneficiarioCCI:             benCCI,
	}

	err = h.Repo.Actualizar(&d, conceptosIDs)
	if err != nil {
		log.Println("Error actualizando descuento:", err)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf(`<div id="form-descuento-error"><article style="background-color: #ffcdd2; color: #b71c1c; padding: 1rem; margin-bottom: 1rem; border-radius: 5px;">❌ Error: %s</article></div>`, err.Error())))
		return
	}

	// Redirigir a la vista principal de descuentos
	h.VistaUI(w, r)
}

// ToggleActivo cambia el estado de activación de un descuento
func (h *DescuentoHandler) ToggleActivo(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	r.ParseForm()

	idStr := r.FormValue("id")
	if idStr == "" {
		idStr = r.URL.Query().Get("id")
	}
	id, _ := strconv.Atoi(idStr)

	activoStr := r.FormValue("activo")
	if activoStr == "" {
		activoStr = r.URL.Query().Get("activo")
	}
	activo := activoStr == "true"

	motivoBaja := r.FormValue("motivo_baja")
	if motivoBaja == "" {
		motivoBaja = r.URL.Query().Get("motivo_baja")
	}

	err := h.Repo.ToggleActivo(id, tenantID, activo, motivoBaja)
	if err != nil {
		http.Error(w, "Error cambiando estado: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Refrescar listado
	h.ListarHTMX(w, r)
}

// Eliminar remueve un descuento
func (h *DescuentoHandler) Eliminar(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	r.ParseForm()

	idStr := r.FormValue("id")
	if idStr == "" {
		idStr = r.URL.Query().Get("id")
	}
	id, _ := strconv.Atoi(idStr)

	err := h.Repo.Eliminar(id, tenantID)
	if err != nil {
		http.Error(w, "Error eliminando: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.ListarHTMX(w, r)
}

// ConceptosPorTrabajadorHTMX devuelve la lista de opciones de conceptos de ingreso para el selector TomSelect
func (h *DescuentoHandler) ConceptosPorTrabajadorHTMX(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	trabajadorID, _ := strconv.Atoi(r.URL.Query().Get("trabajador_id"))

	conceptos, err := h.Repo.ObtenerConceptosIngresoPorTrabajador(tenantID, trabajadorID)
	if err != nil {
		http.Error(w, "Error obteniendo conceptos: "+err.Error(), http.StatusInternalServerError)
		return
	}

	for _, c := range conceptos {
		fmt.Fprintf(w, `<option value="%d">[%s] %s</option>`, c.ID, c.ConceptoCodigo, c.NombrePersonalizado)
	}
}

// InfoTrabajadorHTMX devuelve la tarjeta con puesto y régimen, y el bloque de switches de conceptos de ingreso del puesto
func (h *DescuentoHandler) InfoTrabajadorHTMX(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	trabajadorID, _ := strconv.Atoi(r.URL.Query().Get("trabajador_id"))

	info, err := h.Repo.ObtenerInfoTrabajadorPuesto(tenantID, trabajadorID)
	if err != nil {
		http.Error(w, "Error obteniendo información: "+err.Error(), http.StatusInternalServerError)
		return
	}

	tmplHTML := `<div id="info-puesto-trabajador">
    {{if .TieneContrato}}
    <div class="card p-xs bg-card-kpi border-radius mt-xs">
        <div class="flex-row-between flex-items-center flex-wrap flex-gap-sm">
            <div>
                <span class="text-muted font-xs text-uppercase">Puesto Actual Asignado:</span>
                <strong class="font-sm text-primary block">{{.PuestoNombre}}</strong>
            </div>
            <div>
                <span class="text-muted font-xs text-uppercase">Régimen Laboral:</span>
                <div><mark class="badge badge-info font-xs">{{.RegimenCodigo}} - {{.RegimenNombre}}</mark></div>
            </div>
        </div>
    </div>
    {{else if gt .TrabajadorID 0}}
    <div class="card p-xs bg-card-kpi border-radius mt-xs">
        <small class="text-warning">⚠️ Sin contrato activo registrado. Se muestran los conceptos remunerativos generales.</small>
    </div>
    {{else}}
    <small class="text-muted">Seleccione un trabajador para consultar su puesto y régimen asignado.</small>
    {{end}}
</div>

<div id="contenedor-conceptos-afectos" hx-swap-oob="true">
    <div class="flex-row-between flex-items-center mb-xs">
        <label class="font-bold text-primary font-sm mb-0">
            Conceptos de Ingreso Afectos (Base Computable del Puesto)
        </label>
        <div class="flex-gap-xs">
            <button type="button" class="secondary outline btn-compact font-xs" onclick="marcarTodosConceptos(true)">
                Marcar Todos
            </button>
            <button type="button" class="secondary outline btn-compact font-xs" onclick="marcarTodosConceptos(false)">
                Desmarcar Todos
            </button>
        </div>
    </div>
    <p class="font-xs text-muted mb-xs">
        Active los conceptos remunerativos que integran la base imponible del descuento. Si no selecciona ninguno, la retención se calculará sobre el total de haberes afectos de la boleta.
    </p>
    
    <div class="grid-conceptos-switches">
        {{range .Conceptos}}
        <div class="card p-xs flex-row-between flex-items-center mb-0 border-light">
            <label class="flex-inline flex-items-center mb-0 font-sm cursor-pointer flex-gap-sm">
                <input type="checkbox" name="conceptos_afectos_ids" value="{{.ConceptoTenantID}}" role="switch" {{if .Seleccionado}}checked{{end}}>
                <span><strong class="stat-mono">[{{.ConceptoCodigo}}]</strong> {{.NombrePersonalizado}}</span>
            </label>
            {{if gt .Monto 0.0}}
            <span class="badge badge-secondary font-xs stat-mono">S/ {{printf "%.2f" .Monto}}</span>
            {{end}}
        </div>
        {{else}}
        <p class="text-muted font-sm p-sm text-center">No se encontraron conceptos de ingreso configurados para este puesto.</p>
        {{end}}
    </div>
</div>`

	t, err := template.New("info_trabajador").Parse(tmplHTML)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	t.Execute(w, info)
}
