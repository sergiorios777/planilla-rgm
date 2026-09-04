package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
	"planilla-rgm/internal/services"
	"strconv"
	"strings"
	"time"
)

type PlameHandler struct {
	Service      *services.PlameService
	PlanillaRepo *repository.PlanillaRepository
	LicRepo      *repository.LicenciaVacacionRepository
}

func NewPlameHandler(service *services.PlameService, planillaRepo *repository.PlanillaRepository, licRepo *repository.LicenciaVacacionRepository) *PlameHandler {
	return &PlameHandler{
		Service:      service,
		PlanillaRepo: planillaRepo,
		LicRepo:      licRepo,
	}
}

// VistaHub renderiza la vista principal (Dashboard) del módulo de Auditoría y Declaración PLAME
func (h *PlameHandler) VistaHub(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	now := time.Now()
	anio, _ := strconv.Atoi(r.URL.Query().Get("anio"))
	mes, _ := strconv.Atoi(r.URL.Query().Get("mes"))
	if anio <= 0 {
		anio = now.Year()
	}
	if mes <= 0 || mes > 12 {
		mes = int(now.Month())
	}

	resumen, err := h.Service.ObtenerResumenPeriodo(tenantID, anio, mes)
	if err != nil {
		http.Error(w, "Error obteniendo resumen PLAME: "+err.Error(), http.StatusInternalServerError)
		return
	}

	planillas, err := h.Service.ObtenerPeriodoPlanillas(tenantID, anio, mes)
	if err != nil {
		http.Error(w, "Error obteniendo planillas del periodo: "+err.Error(), http.StatusInternalServerError)
		return
	}

	datos := map[string]interface{}{
		"Anio":      anio,
		"Mes":       mes,
		"Resumen":   resumen,
		"Planillas": planillas,
	}

	tmpl, err := template.ParseFiles("ui/templates/tenant/plame_hub_ui.html")
	if err != nil {
		http.Error(w, "Error cargando plantilla de Hub PLAME: "+err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, datos)
}

// ListarPlanillasPeriodoHTMX retorna el fragmento con la tabla de planillas y KPIs al cambiar Año o Mes
func (h *PlameHandler) ListarPlanillasPeriodoHTMX(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	anio, _ := strconv.Atoi(r.URL.Query().Get("anio"))
	mes, _ := strconv.Atoi(r.URL.Query().Get("mes"))
	if anio <= 0 {
		anio = time.Now().Year()
	}
	if mes <= 0 || mes > 12 {
		mes = int(time.Now().Month())
	}

	resumen, err := h.Service.ObtenerResumenPeriodo(tenantID, anio, mes)
	if err != nil {
		http.Error(w, "Error obteniendo resumen: "+err.Error(), http.StatusInternalServerError)
		return
	}

	planillas, err := h.Service.ObtenerPeriodoPlanillas(tenantID, anio, mes)
	if err != nil {
		http.Error(w, "Error obteniendo planillas: "+err.Error(), http.StatusInternalServerError)
		return
	}

	datos := map[string]interface{}{
		"Anio":      anio,
		"Mes":       mes,
		"Resumen":   resumen,
		"Planillas": planillas,
	}

	tmpl, err := template.ParseFiles("ui/templates/tenant/plame_hub_ui.html")
	if err != nil {
		http.Error(w, "Error cargando plantilla: "+err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "fragmento_planillas_periodo", datos)
}

// VistaAuditoria renderiza la pantalla de auditoría y edición macro/micro para una planilla específica
func (h *PlameHandler) VistaAuditoria(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	planillaID, _ := strconv.Atoi(r.URL.Query().Get("id"))
	if planillaID <= 0 {
		planillaID, _ = strconv.Atoi(r.FormValue("planilla_id"))
	}

	planilla, err := h.PlanillaRepo.ObtenerPorID(planillaID, tenantID)
	if err != nil || planilla == nil {
		http.Error(w, "Planilla no encontrada", http.StatusNotFound)
		return
	}

	conceptos, err := h.Service.ObtenerConceptosAgrupados(planillaID, tenantID)
	if err != nil {
		http.Error(w, "Error obteniendo conceptos agrupados: "+err.Error(), http.StatusInternalServerError)
		return
	}

	maestros, err := h.Service.Repo.ObtenerMaestrosSunat()
	if err != nil {
		http.Error(w, "Error obteniendo catálogo SUNAT: "+err.Error(), http.StatusInternalServerError)
		return
	}

	mapaIncidencias, _ := h.LicRepo.ObtenerIncidenciasMes(tenantID, planilla.Anio, planilla.Mes)
	var listaIncidencias []models.PersonalIncidenciaMes
	var totalVacaciones, totalLicConGoce, totalLicSinGoce int
	for _, incs := range mapaIncidencias {
		for _, inc := range incs {
			listaIncidencias = append(listaIncidencias, inc)
			switch inc.Tipo {
			case "VACACION":
				totalVacaciones++
			case "LICENCIA_CON_GOCE":
				totalLicConGoce++
			case "LICENCIA_SIN_GOCE":
				totalLicSinGoce++
			}
		}
	}

	tieneVacacional := false
	for _, c := range conceptos {
		if c.TieneVacacional || c.CodigoSunatActual == "2007" || c.CodigoSunatActual == "2043" || c.CodigoSunatActual == "2049" || c.CodigoSunatActual == "0118" {
			tieneVacacional = true
			break
		}
	}
	alertaVacacionesSin0118 := (totalVacaciones > 0 && !tieneVacacional)

	datos := map[string]interface{}{
		"Planilla":               planilla,
		"Conceptos":              conceptos,
		"Maestros":               maestros,
		"Incidencias":            listaIncidencias,
		"TotalVacaciones":        totalVacaciones,
		"TotalLicConGoce":        totalLicConGoce,
		"TotalLicSinGoce":        totalLicSinGoce,
		"AlertaVacacionesSin0118": alertaVacacionesSin0118,
	}

	tmpl, err := template.ParseFiles("ui/templates/tenant/planilla_sunat_codigos_ui.html")
	if err != nil {
		http.Error(w, "Error cargando plantilla de auditoría: "+err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, datos)
}

// VistaConceptoTrabajadores renderiza la vista completa con el desglose de colaboradores por código SUNAT
func (h *PlameHandler) VistaConceptoTrabajadores(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	planillaID, _ := strconv.Atoi(r.URL.Query().Get("planilla_id"))
	if planillaID <= 0 {
		planillaID, _ = strconv.Atoi(r.FormValue("planilla_id"))
	}
	codigoSunat := r.URL.Query().Get("codigo_sunat")
	if codigoSunat == "" {
		codigoSunat = r.FormValue("codigo_sunat")
	}

	if planillaID <= 0 || codigoSunat == "" {
		http.Error(w, "Parámetros requeridos faltantes (planilla_id, codigo_sunat)", http.StatusBadRequest)
		return
	}

	planilla, err := h.PlanillaRepo.ObtenerPorID(planillaID, tenantID)
	if err != nil || planilla == nil {
		http.Error(w, "Planilla no encontrada", http.StatusNotFound)
		return
	}

	trabajadores, err := h.Service.ObtenerTrabajadoresPorConcepto(planillaID, tenantID, codigoSunat)
	if err != nil {
		http.Error(w, "Error al obtener colaboradores: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var totalDevengado, totalPagado float64
	var totalAjustados int
	var descripcionSunat string
	for _, t := range trabajadores {
		totalDevengado += t.MontoDevengado
		totalPagado += t.MontoPagado
		if t.EsAjusteManual {
			totalAjustados++
		}
		if descripcionSunat == "" && t.DescripcionSunat != "" {
			descripcionSunat = t.DescripcionSunat
		}
	}
	if descripcionSunat == "" {
		descripcionSunat = "Concepto Oficial SUNAT"
	}

	maestros, _ := h.Service.Repo.ObtenerMaestrosSunat()

	datos := map[string]interface{}{
		"Planilla":         planilla,
		"CodigoSunat":      codigoSunat,
		"DescripcionSunat": descripcionSunat,
		"TotalDevengado":   totalDevengado,
		"TotalPagado":      totalPagado,
		"TotalAjustados":   totalAjustados,
		"Trabajadores":     trabajadores,
		"Maestros":         maestros,
	}

	tmpl, err := template.ParseFiles("ui/templates/tenant/plame_concepto_trabajadores_ui.html")
	if err != nil {
		http.Error(w, "Error cargando plantilla de colaboradores: "+err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, datos)
}

// VistaReasignarConcepto renderiza la vista asistida para reasignar masivamente un código SUNAT
func (h *PlameHandler) VistaReasignarConcepto(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	planillaID, _ := strconv.Atoi(r.URL.Query().Get("planilla_id"))
	codigoSunat := r.URL.Query().Get("codigo_sunat")

	if planillaID <= 0 || codigoSunat == "" {
		http.Error(w, "Parámetros requeridos faltantes (planilla_id, codigo_sunat)", http.StatusBadRequest)
		return
	}

	planilla, err := h.PlanillaRepo.ObtenerPorID(planillaID, tenantID)
	if err != nil || planilla == nil {
		http.Error(w, "Planilla no encontrada", http.StatusNotFound)
		return
	}

	trabajadores, err := h.Service.ObtenerTrabajadoresPorConcepto(planillaID, tenantID, codigoSunat)
	if err != nil {
		http.Error(w, "Error al obtener colaboradores afectados: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var totalDevengado, totalPagado float64
	var descripcionActual, tipoConcepto string
	for _, t := range trabajadores {
		totalDevengado += t.MontoDevengado
		totalPagado += t.MontoPagado
		if descripcionActual == "" && t.DescripcionSunat != "" {
			descripcionActual = t.DescripcionSunat
		}
		if tipoConcepto == "" && t.TipoConcepto != "" {
			tipoConcepto = t.TipoConcepto
		}
	}
	if descripcionActual == "" {
		descripcionActual = "CONCEPTO SIN DESCRIPCIÓN"
	}
	if tipoConcepto == "" {
		tipoConcepto = "INGRESO"
	}

	maestros, err := h.Service.Repo.ObtenerMaestrosSunat()
	if err != nil {
		http.Error(w, "Error obteniendo catálogo SUNAT: "+err.Error(), http.StatusInternalServerError)
		return
	}

	datos := map[string]interface{}{
		"Planilla":               planilla,
		"CodigoSunatActual":      codigoSunat,
		"DescripcionSunatActual": descripcionActual,
		"TipoConcepto":           tipoConcepto,
		"TotalTrabajadores":      len(trabajadores),
		"TotalDevengado":         totalDevengado,
		"TotalPagado":            totalPagado,
		"TrabajadoresAfectados":  trabajadores,
		"Maestros":               maestros,
	}

	tmpl, err := template.ParseFiles("ui/templates/tenant/plame_reasignar_concepto_ui.html")
	if err != nil {
		http.Error(w, "Error cargando plantilla de reasignación: "+err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, datos)
}

// VerTrabajadoresPorConceptoHTMX delega a VistaConceptoTrabajadores para compatibilidad
func (h *PlameHandler) VerTrabajadoresPorConceptoHTMX(w http.ResponseWriter, r *http.Request) {
	h.VistaConceptoTrabajadores(w, r)
}

// VistaEditarTrabajador renderiza la vista completa para la edición y desdoblamiento tributario de un colaborador
func (h *PlameHandler) VistaEditarTrabajador(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	detalleID, _ := strconv.Atoi(r.URL.Query().Get("detalle_id"))
	if detalleID <= 0 {
		http.Error(w, "Detalle de planilla no especificado", http.StatusBadRequest)
		return
	}

	conceptos, err := h.Service.ObtenerDetalleTrabajador(detalleID, tenantID)
	if err != nil {
		http.Error(w, "Error obteniendo conceptos de trabajador: "+err.Error(), http.StatusInternalServerError)
		return
	}

	maestros, err := h.Service.Repo.ObtenerMaestrosSunat()
	if err != nil {
		http.Error(w, "Error obteniendo catálogo SUNAT: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var trabajadorNombre, trabajadorDoc, regimenNombre string
	var planillaID int
	var totalDevengado, totalPagado float64
	for _, c := range conceptos {
		totalDevengado += c.MontoDevengado
		totalPagado += c.MontoPagado
	}
	if len(conceptos) > 0 {
		trabajadorNombre = conceptos[0].TrabajadorNombre
		trabajadorDoc = conceptos[0].TrabajadorDocumento
		regimenNombre = conceptos[0].RegimenNombre
		planillaID = conceptos[0].PlanillaID
	}

	planilla, _ := h.PlanillaRepo.ObtenerPorID(planillaID, tenantID)
	origenVista := r.URL.Query().Get("origen_vista")
	codigoSunatFiltro := r.URL.Query().Get("codigo_sunat_filtro")

	datos := map[string]interface{}{
		"Planilla":          planilla,
		"PlanillaDetalleID": detalleID,
		"TrabajadorNombre":  trabajadorNombre,
		"TrabajadorDoc":     trabajadorDoc,
		"RegimenNombre":     regimenNombre,
		"TotalDevengado":    totalDevengado,
		"TotalPagado":       totalPagado,
		"Conceptos":         conceptos,
		"Maestros":          maestros,
		"OrigenVista":       origenVista,
		"CodigoSunatFiltro": codigoSunatFiltro,
	}

	tmpl, err := template.ParseFiles(
		"ui/templates/tenant/plame_trabajador_edicion_ui.html",
		"ui/templates/components/buscador_codigo_sunat_modal.html",
	)
	if err != nil {
		http.Error(w, "Error cargando plantilla de edición de trabajador: "+err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, datos)
}

// ModalEditarTrabajadorHTMX delega a VistaEditarTrabajador para compatibilidad
func (h *PlameHandler) ModalEditarTrabajadorHTMX(w http.ResponseWriter, r *http.Request) {
	h.VistaEditarTrabajador(w, r)
}

// GuardarTrabajadorHTMX guarda las líneas tributarias editadas o desdobladas de un trabajador
func (h *PlameHandler) GuardarTrabajadorHTMX(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	r.ParseForm()
	tenantID := obtenerTenantID(r)

	detalleID, _ := strconv.Atoi(r.FormValue("planilla_detalle_id"))
	planillaID, _ := strconv.Atoi(r.FormValue("planilla_id"))
	if detalleID <= 0 || planillaID <= 0 {
		http.Error(w, "Parámetros inválidos", http.StatusBadRequest)
		return
	}

	codigos := r.Form["codigo_sunat[]"]
	tipos := r.Form["tipo_concepto[]"]
	devengados := r.Form["monto_devengado[]"]
	pagados := r.Form["monto_pagado[]"]
	vacacionales := r.Form["es_vacacional[]"]
	observaciones := r.Form["observacion[]"]

	var items []models.PlanillaPlameConcepto
	for i := 0; i < len(codigos); i++ {
		cod := strings.TrimSpace(codigos[i])
		if cod == "" {
			continue
		}
		tipo := "INGRESO"
		if i < len(tipos) && tipos[i] != "" {
			tipo = tipos[i]
		}
		dev, _ := strconv.ParseFloat(devengados[i], 64)
		pag, _ := strconv.ParseFloat(pagados[i], 64)
		esVac := false
		if i < len(vacacionales) {
			esVac = (vacacionales[i] == "true" || vacacionales[i] == "1")
		}
		obs := ""
		if i < len(observaciones) {
			obs = observaciones[i]
		}

		if dev <= 0 && pag <= 0 {
			continue
		}

		items = append(items, models.PlanillaPlameConcepto{
			CodigoSunat:          cod,
			TipoConcepto:         tipo,
			MontoDevengado:       dev,
			MontoPagado:          pag,
			EsConceptoVacacional: esVac,
			EsAjusteManual:       true,
			ObservacionAjuste:    obs,
		})
	}

	if err := h.Service.GuardarConceptosTrabajador(detalleID, tenantID, items); err != nil {
		http.Error(w, "Error guardando conceptos: "+err.Error(), http.StatusInternalServerError)
		return
	}

	origenVista := r.FormValue("origen_vista")
	codigoSunatFiltro := strings.TrimSpace(r.FormValue("codigo_sunat_filtro"))
	if origenVista == "trabajadores" && codigoSunatFiltro != "" {
		r.URL.RawQuery = fmt.Sprintf("planilla_id=%d&codigo_sunat=%s", planillaID, url.QueryEscape(codigoSunatFiltro))
		h.VistaConceptoTrabajadores(w, r)
		return
	}

	r.URL.RawQuery = fmt.Sprintf("id=%d", planillaID)
	h.VistaAuditoria(w, r)
}

// ActualizarCodigoMasivoHTMX procesa el cambio de código SUNAT para toda la planilla
func (h *PlameHandler) ActualizarCodigoMasivoHTMX(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	r.ParseForm()
	tenantID := obtenerTenantID(r)

	planillaID, _ := strconv.Atoi(r.FormValue("planilla_id"))
	codigoActual := r.FormValue("codigo_sunat_actual")
	nuevoMaestroID, _ := strconv.Atoi(r.FormValue("nuevo_maestro_id"))
	actualizarDefault := r.FormValue("actualizar_default") == "true" || r.FormValue("actualizar_default") == "on" || r.FormValue("actualizar_default") == "1"

	if planillaID <= 0 || nuevoMaestroID <= 0 || codigoActual == "" {
		http.Error(w, "Parámetros inválidos", http.StatusBadRequest)
		return
	}

	_, err := h.Service.ActualizarCodigoMasivo(planillaID, tenantID, codigoActual, nuevoMaestroID, actualizarDefault)
	if err != nil {
		http.Error(w, "Error al actualizar código SUNAT: "+err.Error(), http.StatusInternalServerError)
		return
	}

	r.URL.RawQuery = fmt.Sprintf("id=%d", planillaID)
	h.VistaAuditoria(w, r)
}

// ResetearSnapshotHTMX restablece el snapshot de PLAME de una planilla
func (h *PlameHandler) ResetearSnapshotHTMX(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	r.ParseForm()
	tenantID := obtenerTenantID(r)
	planillaID, _ := strconv.Atoi(r.FormValue("planilla_id"))
	if planillaID <= 0 {
		http.Error(w, "ID de planilla inválido", http.StatusBadRequest)
		return
	}

	if err := h.Service.ResetearSnapshot(planillaID, tenantID); err != nil {
		http.Error(w, "Error restableciendo snapshot: "+err.Error(), http.StatusInternalServerError)
		return
	}

	r.URL.RawQuery = fmt.Sprintf("id=%d", planillaID)
	h.VistaAuditoria(w, r)
}

// ExportarModalHTMX renderiza el contenido del modal de exportación para una planilla
func (h *PlameHandler) ExportarModalHTMX(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	planillaID, _ := strconv.Atoi(r.URL.Query().Get("id"))
	if planillaID <= 0 {
		planillaID, _ = strconv.Atoi(r.FormValue("planilla_id"))
	}

	planilla, err := h.PlanillaRepo.ObtenerPorID(planillaID, tenantID)
	if err != nil || planilla == nil {
		http.Error(w, "Planilla no encontrada", http.StatusNotFound)
		return
	}

	ruc, _ := h.Service.Repo.ObtenerRucTenant(tenantID)
	filenameBase := fmt.Sprintf("0601%d%02d%s", planilla.Anio, planilla.Mes, ruc)

	html := fmt.Sprintf(`
        <div class="pt-sm">
            <p>Descargue los archivos estructurados generados a partir del cálculo y auditoría tributaria de esta planilla:</p>
            <div class="grid mb-md">
                <a href="/tenant/plame/descargar?id=%d&tipo=jor" class="outline primary btn-compact text-center flex-inline flex-items-center flex-center flex-gap-xs">
                    📄 Jornada (.jor)
                </a>
                <a href="/tenant/plame/descargar?id=%d&tipo=rem" class="outline primary btn-compact text-center flex-inline flex-items-center flex-center flex-gap-xs">
                    📄 Remuneraciones (.rem)
                </a>
                <a href="/tenant/plame/descargar?id=%d&tipo=snl" class="outline primary btn-compact text-center flex-inline flex-items-center flex-center flex-gap-xs">
                    📄 Suspensiones (.snl)
                </a>
            </div>
            <div class="text-center mt-md">
                <a href="/tenant/plame/descargar?id=%d&tipo=zip" class="primary btn-compact text-center flex-inline flex-items-center flex-center flex-gap-xs">
                    📦 Descargar Paquete Completo (.ZIP)
                </a>
            </div>
            <small class="text-muted d-block text-center mt-sm">Estructura SUNAT PDT PLAME: %s.[jor|rem|snl]</small>
        </div>
    `, planillaID, planillaID, planillaID, planillaID, filenameBase)

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// DescargarArchivos genera y transmite en stream los archivos .jor, .rem, .snl o .zip
func (h *PlameHandler) DescargarArchivos(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	planillaID, _ := strconv.Atoi(r.URL.Query().Get("id"))
	tipo := strings.ToLower(r.URL.Query().Get("tipo"))

	if planillaID <= 0 || tipo == "" {
		http.Error(w, "Parámetros inválidos", http.StatusBadRequest)
		return
	}

	planilla, err := h.PlanillaRepo.ObtenerPorID(planillaID, tenantID)
	if err != nil || planilla == nil {
		http.Error(w, "Planilla no encontrada", http.StatusNotFound)
		return
	}

	ruc, _ := h.Service.Repo.ObtenerRucTenant(tenantID)
	filenameBase := fmt.Sprintf("0601%d%02d%s", planilla.Anio, planilla.Mes, ruc)

	switch tipo {
	case "jor":
		datos, err := h.Service.Repo.ObtenerDatosPlameJornada(planillaID, tenantID)
		if err != nil {
			http.Error(w, "Error al obtener jornada: "+err.Error(), 500)
			return
		}
		texto := h.Service.GenerarJornadaTexto(datos)

		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.jor"`, filenameBase))
		w.Write([]byte(texto))

	case "rem":
		datosRem, err := h.Service.ObtenerRemuneracionesSnapshot(planillaID, tenantID)
		if err != nil {
			http.Error(w, "Error al obtener remuneraciones de snapshot: "+err.Error(), 500)
			return
		}
		texto := h.Service.GenerarRemuneracionesTexto(datosRem)

		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.rem"`, filenameBase))
		w.Write([]byte(texto))

	case "snl":
		mapaIncs, err := h.LicRepo.ObtenerIncidenciasMes(tenantID, planilla.Anio, planilla.Mes)
		if err != nil {
			http.Error(w, "Error al obtener incidencias: "+err.Error(), 500)
			return
		}
		var listaIncs []models.PersonalIncidenciaMes
		for _, incs := range mapaIncs {
			listaIncs = append(listaIncs, incs...)
		}
		texto := h.Service.GenerarSuspensionesTexto(listaIncs)

		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.snl"`, filenameBase))
		w.Write([]byte(texto))

	case "zip":
		datosJor, err := h.Service.Repo.ObtenerDatosPlameJornada(planillaID, tenantID)
		if err != nil {
			http.Error(w, "Error al obtener jornada: "+err.Error(), 500)
			return
		}
		textoJor := h.Service.GenerarJornadaTexto(datosJor)

		datosRem, err := h.Service.ObtenerRemuneracionesSnapshot(planillaID, tenantID)
		if err != nil {
			http.Error(w, "Error al obtener remuneraciones de snapshot: "+err.Error(), 500)
			return
		}
		textoRem := h.Service.GenerarRemuneracionesTexto(datosRem)

		mapaIncs, _ := h.LicRepo.ObtenerIncidenciasMes(tenantID, planilla.Anio, planilla.Mes)
		var listaIncs []models.PersonalIncidenciaMes
		for _, incs := range mapaIncs {
			listaIncs = append(listaIncs, incs...)
		}
		textoSnl := h.Service.GenerarSuspensionesTexto(listaIncs)

		zipBytes, err := h.Service.GenerarZipCompleto(textoJor, textoRem, textoSnl, filenameBase+".jor", filenameBase+".rem", filenameBase+".snl")
		if err != nil {
			http.Error(w, "Error al generar ZIP: "+err.Error(), 500)
			return
		}

		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.zip"`, filenameBase))
		w.Write(zipBytes)

	default:
		http.Error(w, "Tipo de archivo no soportado", http.StatusBadRequest)
	}
}
