package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
	"planilla-rgm/internal/services"
	"strconv"
	"strings"
)

type PlanillaHandler struct {
	Repo               *repository.PlanillaRepository
	PuestoRepo         *repository.PuestoRepository
	ConceptoTenantRepo *repository.ConceptoTenantRepository
	OrganigramaRepo    *repository.OrganigramaRepository
	AnexoService       *services.AnexoService
}

func (h *PlanillaHandler) VistaUI(w http.ResponseWriter, r *http.Request) {
	tmpl, _ := template.ParseFiles("ui/templates/tenant/planillas_ui.html")
	tmpl.Execute(w, nil)
}

func (h *PlanillaHandler) Listar(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	planillas, _ := h.Repo.ObtenerTodos(tenantID)

	tmpl, _ := template.ParseFiles("ui/templates/tenant/planillas_ui.html")
	tmpl.ExecuteTemplate(w, "tabla_planillas", planillas)
}

func (h *PlanillaHandler) Crear(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	tenantID := obtenerTenantID(r)

	anio, _ := strconv.Atoi(r.FormValue("anio"))
	mes, _ := strconv.Atoi(r.FormValue("mes"))

	esExtraordinaria := r.FormValue("es_extraordinaria") == "true" || r.FormValue("es_extraordinaria") == "on"

	nuevaPlanilla := models.Planilla{
		TenantID:         tenantID,
		Anio:             anio,
		Mes:              mes,
		Descripcion:      r.FormValue("descripcion"),
		EsExtraordinaria: esExtraordinaria,
	}

	err := h.Repo.Crear(&nuevaPlanilla)
	if err != nil {
		// Validamos la restricción UNIQUE que creamos en la migración
		if strings.Contains(err.Error(), "unique_planilla_mes") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`
				<div id="alerta-planilla" hx-swap-oob="true">
					<article style="background-color: #ffcdd2; color: #b71c1c; padding: 1rem; margin-bottom: 1rem; border-radius: 5px;">
						❌ Error: Ya existe una planilla con esa misma descripción para este mes y año.
					</article>
				</div>
			`))
			return
		}
		http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
		return
	}

	// Limpiamos alertas previas y actualizamos la tabla
	w.Write([]byte(`<div id="alerta-planilla" hx-swap-oob="true"></div>`))
	w.Header().Set("HX-Trigger", "cerrarModal")
	h.Listar(w, r)
}

// Procesar ejecuta el motor de cálculo (ordinario o recálculo especial) y redirige a la vista de detalle
func (h *PlanillaHandler) Procesar(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	planillaID, _ := strconv.Atoi(r.URL.Query().Get("id"))

	planilla, err := h.Repo.ObtenerPorID(planillaID, tenantID)
	if err != nil || planilla == nil {
		http.Error(w, "Planilla no encontrada", http.StatusNotFound)
		return
	}

	if planilla.EsExtraordinaria {
		err = h.Repo.RecalcularPlanillaEspecial(r.Context(), planillaID, tenantID)
	} else {
		servicioPlanilla := services.NewPlanillaService(h.Repo)
		err = servicioPlanilla.Procesar(planillaID, tenantID)
	}

	if err != nil {
		// Si el motor falla, mostramos una alerta sin salir de la página
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`
			<div id="alerta-planilla" hx-swap-oob="true">
				<article style="background-color: #ffcdd2; color: #b71c1c; padding: 1rem; margin-bottom: 1rem; border-radius: 5px;">
					❌ Error al Procesar Planilla: ` + err.Error() + `
				</article>
			</div>
		`))
		return
	}

	// NUEVO: En lugar de redirigir, empujamos la nueva URL al navegador
	// y cargamos la vista directamente en el mismo contenedor
	w.Header().Set("HX-Push-Url", "/tenant/planillas/detalle/ui?id="+strconv.Itoa(planillaID))

	// Limpiamos alertas si las hubo y procesamos la vista
	w.Write([]byte(`<div id="alerta-planilla" hx-swap-oob="true"></div>`))

	// Llamamos a la vista de detalle
	r.URL.RawQuery = "id=" + strconv.Itoa(planillaID)
	h.VistaDetalle(w, r)
}

// VistaDetalle carga la pantalla con las boletas generadas
func (h *PlanillaHandler) VistaDetalle(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	planillaID, _ := strconv.Atoi(r.URL.Query().Get("id"))

	detalles, _ := h.Repo.ObtenerDetalles(planillaID, tenantID)
	planilla, _ := h.Repo.ObtenerPorID(planillaID, tenantID)
	metas, _ := h.Repo.ObtenerMetas(tenantID)
	fuentesRubros, _ := h.Repo.ObtenerFuentesRubros()

	planillaEstado := ""
	esExtraordinaria := false
	if planilla != nil {
		planillaEstado = planilla.Estado
		esExtraordinaria = planilla.EsExtraordinaria
	}

	tarjetas := services.GenerarTarjetasDetallePlanilla(detalles)

	datos := map[string]interface{}{
		"PlanillaID":       planillaID,
		"Detalles":         detalles,
		"PlanillaEstado":   planillaEstado,
		"EsExtraordinaria": esExtraordinaria,
		"Metas":            metas,
		"FuentesRubros":    fuentesRubros,
		"Tarjetas":         tarjetas,
	}

	tmpl, _ := template.ParseFiles("ui/templates/tenant/planilla_detalle_ui.html")
	tmpl.Execute(w, datos)
}

// BoletaModalDetalle sirve el HTML del modal con el desglose de conceptos para un trabajador específico
func (h *PlanillaHandler) BoletaModalDetalle(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	detalleID, _ := strconv.Atoi(r.URL.Query().Get("id"))
	if detalleID == 0 {
		http.Error(w, "ID de detalle no válido", http.StatusBadRequest)
		return
	}

	detalle, err := h.Repo.ObtenerDetallePorID(detalleID, tenantID)
	if err != nil {
		http.Error(w, "No se encontró el detalle de boleta: "+err.Error(), http.StatusNotFound)
		return
	}

	tmpl, err := template.ParseFiles("ui/templates/tenant/planilla_detalle_ui.html")
	if err != nil {
		http.Error(w, "Error al cargar plantilla: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = tmpl.ExecuteTemplate(w, "modal_boleta_content", detalle)
	if err != nil {
		log.Println("[ERROR BoletaModalDetalle] Error al ejecutar template:", err)
	}
}

// DescargarAnexo1PDF genera y sirve el archivo PDF para el Anexo 1 (Compromiso Presupuestal)
func (h *PlanillaHandler) DescargarAnexo1PDF(w http.ResponseWriter, r *http.Request) {
	planillaID, _ := strconv.Atoi(r.URL.Query().Get("id"))
	tenantID := obtenerTenantID(r)

	if h.AnexoService == nil {
		http.Error(w, "Servicio de anexos no inicializado", http.StatusInternalServerError)
		return
	}

	datosAnexo, err := h.AnexoService.ObtenerDatosAnexo1(planillaID, tenantID)
	if err != nil {
		http.Error(w, "Error al extraer datos del Anexo 1: "+err.Error(), http.StatusInternalServerError)
		return
	}

	pdfBytes, err := h.AnexoService.GenerarAnexo1PDF(datosAnexo)
	if err != nil {
		http.Error(w, "Error al generar PDF del Anexo 1: "+err.Error(), http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("Anexo_1_Compromiso_Presupuestal_%02d_%d.pdf", datosAnexo.PlanillaMes, datosAnexo.PlanillaAnio)
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Write(pdfBytes)
}

// DescargarAnexo1Excel genera y sirve el libro de cálculo Excel para el Anexo 1 (Compromiso Presupuestal)
func (h *PlanillaHandler) DescargarAnexo1Excel(w http.ResponseWriter, r *http.Request) {
	planillaID, _ := strconv.Atoi(r.URL.Query().Get("id"))
	tenantID := obtenerTenantID(r)

	if h.AnexoService == nil {
		http.Error(w, "Servicio de anexos no inicializado", http.StatusInternalServerError)
		return
	}

	datosAnexo, err := h.AnexoService.ObtenerDatosAnexo1(planillaID, tenantID)
	if err != nil {
		http.Error(w, "Error al extraer datos del Anexo 1: "+err.Error(), http.StatusInternalServerError)
		return
	}

	excelBytes, err := h.AnexoService.GenerarAnexo1Excel(datosAnexo)
	if err != nil {
		http.Error(w, "Error al generar Excel del Anexo 1: "+err.Error(), http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("Anexo_1_Compromiso_Presupuestal_%02d_%d.xlsx", datosAnexo.PlanillaMes, datosAnexo.PlanillaAnio)
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Write(excelBytes)
}

// DescargarAnexo1APDF genera y sirve el archivo PDF para el Anexo 1A (Resumen por Conceptos)
func (h *PlanillaHandler) DescargarAnexo1APDF(w http.ResponseWriter, r *http.Request) {
	planillaID, _ := strconv.Atoi(r.URL.Query().Get("id"))
	tenantID := obtenerTenantID(r)

	if h.AnexoService == nil {
		http.Error(w, "Servicio de anexos no inicializado", http.StatusInternalServerError)
		return
	}

	datosAnexo, err := h.AnexoService.ObtenerDatosAnexo1A(planillaID, tenantID)
	if err != nil {
		http.Error(w, "Error al extraer datos del Anexo 1A: "+err.Error(), http.StatusInternalServerError)
		return
	}

	pdfBytes, err := h.AnexoService.GenerarAnexo1APDF(datosAnexo)
	if err != nil {
		http.Error(w, "Error al generar PDF del Anexo 1A: "+err.Error(), http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("Anexo_1A_Resumen_Conceptos_%02d_%d.pdf", datosAnexo.PlanillaMes, datosAnexo.PlanillaAnio)
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Write(pdfBytes)
}

// DescargarAnexo1AExcel genera y sirve el libro de cálculo Excel para el Anexo 1A (Resumen por Conceptos)
func (h *PlanillaHandler) DescargarAnexo1AExcel(w http.ResponseWriter, r *http.Request) {
	planillaID, _ := strconv.Atoi(r.URL.Query().Get("id"))
	tenantID := obtenerTenantID(r)

	if h.AnexoService == nil {
		http.Error(w, "Servicio de anexos no inicializado", http.StatusInternalServerError)
		return
	}

	datosAnexo, err := h.AnexoService.ObtenerDatosAnexo1A(planillaID, tenantID)
	if err != nil {
		http.Error(w, "Error al extraer datos del Anexo 1A: "+err.Error(), http.StatusInternalServerError)
		return
	}

	excelBytes, err := h.AnexoService.GenerarAnexo1AExcel(datosAnexo)
	if err != nil {
		http.Error(w, "Error al generar Excel del Anexo 1A: "+err.Error(), http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("Anexo_1A_Resumen_Conceptos_%02d_%d.xlsx", datosAnexo.PlanillaMes, datosAnexo.PlanillaAnio)
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Write(excelBytes)
}

// DescargarAnexo2PDF genera y sirve el archivo PDF para el Anexo 2 (Resumen por AFP)
func (h *PlanillaHandler) DescargarAnexo2PDF(w http.ResponseWriter, r *http.Request) {
	planillaID, _ := strconv.Atoi(r.URL.Query().Get("id"))
	tenantID := obtenerTenantID(r)

	if h.AnexoService == nil {
		http.Error(w, "Servicio de anexos no inicializado", http.StatusInternalServerError)
		return
	}

	datosAnexo, err := h.AnexoService.ObtenerDatosAnexo2(planillaID, tenantID)
	if err != nil {
		http.Error(w, "Error al extraer datos del Anexo 2: "+err.Error(), http.StatusInternalServerError)
		return
	}

	pdfBytes, err := h.AnexoService.GenerarAnexo2PDF(datosAnexo)
	if err != nil {
		http.Error(w, "Error al generar PDF del Anexo 2: "+err.Error(), http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("Anexo_2_Resumen_AFP_%02d_%d.pdf", datosAnexo.PlanillaMes, datosAnexo.PlanillaAnio)
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Write(pdfBytes)
}

// DescargarAnexo2Excel genera y sirve el libro de cálculo Excel para el Anexo 2 (Resumen por AFP)
func (h *PlanillaHandler) DescargarAnexo2Excel(w http.ResponseWriter, r *http.Request) {
	planillaID, _ := strconv.Atoi(r.URL.Query().Get("id"))
	tenantID := obtenerTenantID(r)

	if h.AnexoService == nil {
		http.Error(w, "Servicio de anexos no inicializado", http.StatusInternalServerError)
		return
	}

	datosAnexo, err := h.AnexoService.ObtenerDatosAnexo2(planillaID, tenantID)
	if err != nil {
		http.Error(w, "Error al extraer datos del Anexo 2: "+err.Error(), http.StatusInternalServerError)
		return
	}

	excelBytes, err := h.AnexoService.GenerarAnexo2Excel(datosAnexo)
	if err != nil {
		http.Error(w, "Error al generar Excel del Anexo 2: "+err.Error(), http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("Anexo_2_Resumen_AFP_%02d_%d.xlsx", datosAnexo.PlanillaMes, datosAnexo.PlanillaAnio)
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Write(excelBytes)
}

// DescargarAnexo2APDF genera y sirve el archivo PDF para el Anexo 2A (Registro Devengado de AFP)
func (h *PlanillaHandler) DescargarAnexo2APDF(w http.ResponseWriter, r *http.Request) {
	planillaID, _ := strconv.Atoi(r.URL.Query().Get("id"))
	tenantID := obtenerTenantID(r)

	if h.AnexoService == nil {
		http.Error(w, "Servicio de anexos no inicializado", http.StatusInternalServerError)
		return
	}

	datosAnexo, err := h.AnexoService.ObtenerDatosAnexo2A(planillaID, tenantID)
	if err != nil {
		http.Error(w, "Error al extraer datos del Anexo 2A: "+err.Error(), http.StatusInternalServerError)
		return
	}

	pdfBytes, err := h.AnexoService.GenerarAnexo2APDF(datosAnexo)
	if err != nil {
		http.Error(w, "Error al generar PDF del Anexo 2A: "+err.Error(), http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("Anexo_2A_Devengado_AFP_%02d_%d.pdf", datosAnexo.PlanillaMes, datosAnexo.PlanillaAnio)
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Write(pdfBytes)
}

// DescargarAnexo2AExcel genera y sirve el libro de cálculo Excel para el Anexo 2A (Registro Devengado de AFP)
func (h *PlanillaHandler) DescargarAnexo2AExcel(w http.ResponseWriter, r *http.Request) {
	planillaID, _ := strconv.Atoi(r.URL.Query().Get("id"))
	tenantID := obtenerTenantID(r)

	if h.AnexoService == nil {
		http.Error(w, "Servicio de anexos no inicializado", http.StatusInternalServerError)
		return
	}

	datosAnexo, err := h.AnexoService.ObtenerDatosAnexo2A(planillaID, tenantID)
	if err != nil {
		http.Error(w, "Error al extraer datos del Anexo 2A: "+err.Error(), http.StatusInternalServerError)
		return
	}

	excelBytes, err := h.AnexoService.GenerarAnexo2AExcel(datosAnexo)
	if err != nil {
		http.Error(w, "Error al generar Excel del Anexo 2A: "+err.Error(), http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("Anexo_2A_Devengado_AFP_%02d_%d.xlsx", datosAnexo.PlanillaMes, datosAnexo.PlanillaAnio)
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Write(excelBytes)
}

// DescargarAnexo3PDF genera y sirve el archivo PDF para el Anexo 3 (Retenciones de SUNAT)
func (h *PlanillaHandler) DescargarAnexo3PDF(w http.ResponseWriter, r *http.Request) {
	planillaID, _ := strconv.Atoi(r.URL.Query().Get("id"))
	tenantID := obtenerTenantID(r)

	if h.AnexoService == nil {
		http.Error(w, "Servicio de anexos no inicializado", http.StatusInternalServerError)
		return
	}

	datosAnexo, err := h.AnexoService.ObtenerDatosAnexo3(planillaID, tenantID)
	if err != nil {
		http.Error(w, "Error al extraer datos del Anexo 3: "+err.Error(), http.StatusInternalServerError)
		return
	}

	pdfBytes, err := h.AnexoService.GenerarAnexo3PDF(datosAnexo)
	if err != nil {
		http.Error(w, "Error al generar PDF del Anexo 3: "+err.Error(), http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("Anexo_3_Retenciones_SUNAT_%02d_%d.pdf", datosAnexo.PlanillaMes, datosAnexo.PlanillaAnio)
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Write(pdfBytes)
}

// DescargarAnexo3Excel genera y sirve el libro de cálculo Excel para el Anexo 3 (Retenciones de SUNAT)
func (h *PlanillaHandler) DescargarAnexo3Excel(w http.ResponseWriter, r *http.Request) {
	planillaID, _ := strconv.Atoi(r.URL.Query().Get("id"))
	tenantID := obtenerTenantID(r)

	if h.AnexoService == nil {
		http.Error(w, "Servicio de anexos no inicializado", http.StatusInternalServerError)
		return
	}

	datosAnexo, err := h.AnexoService.ObtenerDatosAnexo3(planillaID, tenantID)
	if err != nil {
		http.Error(w, "Error al extraer datos del Anexo 3: "+err.Error(), http.StatusInternalServerError)
		return
	}

	excelBytes, err := h.AnexoService.GenerarAnexo3Excel(datosAnexo)
	if err != nil {
		http.Error(w, "Error al generar Excel del Anexo 3: "+err.Error(), http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("Anexo_3_Retenciones_SUNAT_%02d_%d.xlsx", datosAnexo.PlanillaMes, datosAnexo.PlanillaAnio)
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Write(excelBytes)
}

// VistaAnexos carga la vista con la lista de anexos de la planilla
func (h *PlanillaHandler) VistaAnexos(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	planillaID, _ := strconv.Atoi(r.URL.Query().Get("id"))

	planilla, err := h.Repo.ObtenerPorID(planillaID, tenantID)
	if err != nil || planilla == nil {
		http.Error(w, "Planilla no encontrada", http.StatusNotFound)
		return
	}

	datos := map[string]interface{}{
		"Planilla": planilla,
	}

	tmpl, err := template.ParseFiles("ui/templates/tenant/planillas_anexos_ui.html")
	if err != nil {
		http.Error(w, "Error al cargar vista de anexos: "+err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, datos)
}

// DescargarReportePDF responde a un click del usuario generando y enviando el PDF al vuelo
func (h *PlanillaHandler) DescargarReportePDF(w http.ResponseWriter, r *http.Request) {
	planillaID, _ := strconv.Atoi(r.URL.Query().Get("id"))
	tenantID := obtenerTenantID(r)

	// 1. Obtener la data (La función que hicimos en la Meta 1)
	datos, err := h.Repo.ObtenerDatosParaReporte(planillaID, tenantID)
	if err != nil {
		http.Error(w, "Error al extraer datos: "+err.Error(), 500)
		return
	}

	// 2. Instanciar el servicio y generar el PDF
	pdfService := services.NewPdfService()
	pdfBytes, err := pdfService.GenerarReportePlanilla(datos)
	if err != nil {
		http.Error(w, "Error al generar el PDF: "+err.Error(), 500)
		return
	}

	// 3. Modificar los Headers para que el navegador sepa que es un PDF
	// Usamos "inline" para que intente abrirlo en una pestaña nueva, o "attachment" para forzar descarga
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `inline; filename="Reporte_Planilla.pdf"`)

	// 4. Enviar los bytes crudos
	w.Write(pdfBytes)
}

// DescargarBoletasPDF genera el PDF masivo de boletas
func (h *PlanillaHandler) DescargarBoletasPDF(w http.ResponseWriter, r *http.Request) {
	planillaID, _ := strconv.Atoi(r.URL.Query().Get("id"))
	tenantID := obtenerTenantID(r)

	// Extraemos la misma data rica que usamos para el boletón
	datos, err := h.Repo.ObtenerDatosParaReporte(planillaID, tenantID)
	if err != nil {
		http.Error(w, "Error al extraer datos: "+err.Error(), 500)
		return
	}

	pdfService := services.NewPdfService()
	pdfBytes, err := pdfService.GenerarBoletasPDF(datos) // 💡 Llamamos a la nueva función
	if err != nil {
		http.Error(w, "Error al generar las boletas: "+err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `inline; filename="Boletas_Pago.pdf"`)
	w.Write(pdfBytes)
}

// CerrarPlanilla sella la planilla para evitar futuras modificaciones
func (h *PlanillaHandler) CerrarPlanilla(w http.ResponseWriter, r *http.Request) {
	planillaID, _ := strconv.Atoi(r.URL.Query().Get("id"))
	tenantID := obtenerTenantID(r)

	err := h.Repo.CambiarEstado(planillaID, tenantID, "CERRADA")
	if err != nil {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<div id="alerta-planilla" hx-swap-oob="true"><article style="background-color: #ffcdd2; color: #b71c1c; padding: 1rem;">❌ Error al cerrar: ` + err.Error() + `</article></div>`))
		return
	}

	// Si todo salió bien, empujamos la URL y recargamos la vista de detalle
	w.Header().Set("HX-Push-Url", "/tenant/planillas/detalle/ui?id="+strconv.Itoa(planillaID))
	r.URL.RawQuery = "id=" + strconv.Itoa(planillaID)
	h.VistaDetalle(w, r)
}

// ExportarPlameModal renders the modal content for downloading PLAME files
func (h *PlanillaHandler) ExportarPlameModal(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	planillaID, _ := strconv.Atoi(r.URL.Query().Get("id"))

	planilla, err := h.Repo.ObtenerPorID(planillaID, tenantID)
	if err != nil {
		http.Error(w, "Planilla no encontrada", http.StatusNotFound)
		return
	}

	ruc, err := h.Repo.ObtenerRucTenant(tenantID)
	if err != nil {
		http.Error(w, "RUC no encontrado", http.StatusInternalServerError)
		return
	}

	mesStr := fmt.Sprintf("%02d", planilla.Mes)
	jorFilename := fmt.Sprintf("0601%d%s%s.jor", planilla.Anio, mesStr, ruc)
	remFilename := fmt.Sprintf("0601%d%s%s.rem", planilla.Anio, mesStr, ruc)
	snlFilename := fmt.Sprintf("0601%d%s%s.snl", planilla.Anio, mesStr, ruc)

	datos := map[string]interface{}{
		"Planilla":    planilla,
		"JorFilename": jorFilename,
		"RemFilename": remFilename,
		"SnlFilename": snlFilename,
	}

	tmpl, _ := template.ParseFiles("ui/templates/tenant/planillas_ui.html")
	tmpl.ExecuteTemplate(w, "modal_plame_content", datos)
}

// DescargarPlame streams the .jor, .rem or .zip files for PLAME import
func (h *PlanillaHandler) DescargarPlame(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	planillaID, _ := strconv.Atoi(r.URL.Query().Get("id"))
	tipo := r.URL.Query().Get("tipo")

	planilla, err := h.Repo.ObtenerPorID(planillaID, tenantID)
	if err != nil {
		http.Error(w, "Planilla no encontrada", http.StatusNotFound)
		return
	}

	ruc, err := h.Repo.ObtenerRucTenant(tenantID)
	if err != nil {
		http.Error(w, "RUC no encontrado", http.StatusInternalServerError)
		return
	}

	mesStr := fmt.Sprintf("%02d", planilla.Mes)
	filenameBase := fmt.Sprintf("0601%d%s%s", planilla.Anio, mesStr, ruc)

	plameService := services.NewPlameService(h.Repo)

	switch tipo {
	case "jor":
		datos, err := h.Repo.ObtenerDatosPlameJornada(planillaID, tenantID)
		if err != nil {
			http.Error(w, "Error al obtener datos: "+err.Error(), 500)
			return
		}
		texto := plameService.GenerarJornadaTexto(datos)

		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.jor"`, filenameBase))
		w.Write([]byte(texto))

	case "rem":
		datos, err := h.Repo.ObtenerDatosPlameRemuneraciones(planillaID, tenantID)
		if err != nil {
			http.Error(w, "Error al obtener datos: "+err.Error(), 500)
			return
		}
		texto := plameService.GenerarRemuneracionesTexto(datos)

		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.rem"`, filenameBase))
		w.Write([]byte(texto))

	case "zip":
		datosJor, err := h.Repo.ObtenerDatosPlameJornada(planillaID, tenantID)
		if err != nil {
			http.Error(w, "Error al obtener datos: "+err.Error(), 500)
			return
		}
		textoJor := plameService.GenerarJornadaTexto(datosJor)

		datosRem, err := h.Repo.ObtenerDatosPlameRemuneraciones(planillaID, tenantID)
		if err != nil {
			http.Error(w, "Error al obtener datos: "+err.Error(), 500)
			return
		}
		textoRem := plameService.GenerarRemuneracionesTexto(datosRem)

		zipBytes, err := plameService.GenerarZip(textoJor, textoRem, filenameBase+".jor", filenameBase+".rem")
		if err != nil {
			http.Error(w, "Error al generar ZIP: "+err.Error(), 500)
			return
		}

		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.zip"`, filenameBase))
		w.Write(zipBytes)

	default:
		http.Error(w, "Tipo no soportado", http.StatusBadRequest)
	}
}

// Eliminar borra la planilla y recarga la tabla
func (h *PlanillaHandler) Eliminar(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	planillaID, _ := strconv.Atoi(r.URL.Query().Get("id"))

	servicioPlanilla := services.NewPlanillaService(h.Repo)
	err := servicioPlanilla.Eliminar(planillaID, tenantID)
	if err != nil {
		// Mostramos una alerta sin salir de la página
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`
			<div id="alerta-planilla" hx-swap-oob="true">
				<article style="background-color: #ffcdd2; color: #b71c1c; padding: 1rem; margin-bottom: 1rem; border-radius: 5px;">
					❌ Error: ` + err.Error() + `
				</article>
			</div>
		`))
		return
	}

	// Limpiamos alertas si las hubo y procesamos la vista
	w.Write([]byte(`<div id="alerta-planilla" hx-swap-oob="true"></div>`))
	h.Listar(w, r)
}

type ActualizarPresupuestoConceptoRequest struct {
	MetaID        *int `json:"meta_id"`
	FuenteRubroID *int `json:"fuente_rubro_id"`
}

type ActualizarPresupuestoBulkRequest struct {
	ConceptosIDs  []int `json:"conceptos_ids"`
	MetaID        *int  `json:"meta_id"`
	FuenteRubroID *int  `json:"fuente_rubro_id"`
}

// ActualizarPresupuestoConcepto modifica meta_id y fuente_rubro_id para un concepto de planilla (PUT /api/tenant/planillas/{id}/conceptos/{concepto_id}/presupuesto)
func (h *PlanillaHandler) ActualizarPresupuestoConcepto(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	conceptoIDStr := r.PathValue("concepto_id")
	if conceptoIDStr == "" {
		conceptoIDStr = r.URL.Query().Get("concepto_id")
	}
	conceptoID, err := strconv.Atoi(conceptoIDStr)
	if err != nil || conceptoID <= 0 {
		http.Error(w, "ID de concepto inválido", http.StatusBadRequest)
		return
	}

	var req ActualizarPresupuestoConceptoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Cuerpo de solicitud inválido: "+err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.Repo.ActualizarPresupuestoConceptos(r.Context(), tenantID, []int{conceptoID}, req.MetaID, req.FuenteRubroID); err != nil {
		http.Error(w, "Error al actualizar presupuesto del concepto: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Presupuesto de concepto actualizado correctamente",
	})
}

// ActualizarPresupuestoConceptosBulk modifica meta_id y fuente_rubro_id para múltiples conceptos (POST /api/tenant/planillas/{id}/conceptos/bulk-presupuesto)
func (h *PlanillaHandler) ActualizarPresupuestoConceptosBulk(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	var req ActualizarPresupuestoBulkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Cuerpo de solicitud inválido: "+err.Error(), http.StatusBadRequest)
		return
	}

	if len(req.ConceptosIDs) == 0 {
		http.Error(w, "Debe especificar al menos un ID de concepto", http.StatusBadRequest)
		return
	}

	if err := h.Repo.ActualizarPresupuestoConceptos(r.Context(), tenantID, req.ConceptosIDs, req.MetaID, req.FuenteRubroID); err != nil {
		http.Error(w, "Error al actualizar presupuestos en bloque: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Presupuestos actualizados correctamente",
	})
}

// VistaRubrosMetas renderiza la pantalla dedicada de asignación presupuestal
func (h *PlanillaHandler) VistaRubrosMetas(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	planillaID, _ := strconv.Atoi(r.URL.Query().Get("id"))
	if planillaID <= 0 {
		planillaID, _ = strconv.Atoi(r.FormValue("planilla_id"))
	}

	detalles, _ := h.Repo.ObtenerDetallesConConceptos(planillaID, tenantID)
	planillaEstado, _ := h.Repo.ObtenerEstado(planillaID, tenantID)
	metas, _ := h.Repo.ObtenerMetas(tenantID)
	fuentesRubros, _ := h.Repo.ObtenerFuentesRubros()

	datos := map[string]interface{}{
		"PlanillaID":     planillaID,
		"Detalles":       detalles,
		"PlanillaEstado": planillaEstado,
		"Metas":          metas,
		"FuentesRubros":  fuentesRubros,
	}

	tmpl, _ := template.ParseFiles("ui/templates/tenant/planilla_rubros_metas_ui.html")
	tmpl.Execute(w, datos)
}

// ActualizarPresupuestoSingleHTMX procesa el formulario HTMX de asignación individual
func (h *PlanillaHandler) ActualizarPresupuestoSingleHTMX(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	tenantID := obtenerTenantID(r)
	planillaID, _ := strconv.Atoi(r.FormValue("planilla_id"))
	conceptoID, _ := strconv.Atoi(r.FormValue("concepto_id"))

	var metaID, fuenteRubroID *int
	if mVal := r.FormValue("meta_id"); mVal != "" {
		if id, err := strconv.Atoi(mVal); err == nil && id > 0 {
			metaID = &id
		}
	}
	if rVal := r.FormValue("fuente_rubro_id"); rVal != "" {
		if id, err := strconv.Atoi(rVal); err == nil && id > 0 {
			fuenteRubroID = &id
		}
	}

	if conceptoID > 0 {
		_ = h.Repo.ActualizarPresupuestoConceptos(r.Context(), tenantID, []int{conceptoID}, metaID, fuenteRubroID)
	}

	r.URL.RawQuery = fmt.Sprintf("id=%d", planillaID)
	h.VistaRubrosMetas(w, r)
}

// ActualizarPresupuestoBulkHTMX procesa el formulario HTMX de asignación masiva
func (h *PlanillaHandler) ActualizarPresupuestoBulkHTMX(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	tenantID := obtenerTenantID(r)
	planillaID, _ := strconv.Atoi(r.FormValue("planilla_id"))
	idsStr := r.FormValue("conceptos_ids")

	var conceptosIDs []int
	for _, s := range strings.Split(idsStr, ",") {
		s = strings.TrimSpace(s)
		if id, err := strconv.Atoi(s); err == nil && id > 0 {
			conceptosIDs = append(conceptosIDs, id)
		}
	}

	var metaID, fuenteRubroID *int
	if mVal := r.FormValue("meta_id"); mVal != "" {
		if id, err := strconv.Atoi(mVal); err == nil && id > 0 {
			metaID = &id
		}
	}
	if rVal := r.FormValue("fuente_rubro_id"); rVal != "" {
		if id, err := strconv.Atoi(rVal); err == nil && id > 0 {
			fuenteRubroID = &id
		}
	}

	if len(conceptosIDs) > 0 {
		_ = h.Repo.ActualizarPresupuestoConceptos(r.Context(), tenantID, conceptosIDs, metaID, fuenteRubroID)
	}

	r.URL.RawQuery = fmt.Sprintf("id=%d", planillaID)
	h.VistaRubrosMetas(w, r)
}

// VistaFormuladorEspecial carga la interfaz de formulación de planilla especial / extraordinaria
func (h *PlanillaHandler) VistaFormuladorEspecial(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	planillaID, _ := strconv.Atoi(r.URL.Query().Get("id"))
	if planillaID <= 0 {
		http.Error(w, "ID de planilla no válido", http.StatusBadRequest)
		return
	}

	planilla, err := h.Repo.ObtenerPorID(planillaID, tenantID)
	if err != nil {
		http.Error(w, "No se encontró la planilla: "+err.Error(), http.StatusNotFound)
		return
	}

	// Obtener catálogos
	var conceptosTenant []models.ConceptoTenant
	if h.ConceptoTenantRepo != nil {
		conceptosTenant, _ = h.ConceptoTenantRepo.ObtenerTodos(tenantID)
	}

	var regimenes []models.RegimenLaboral
	if h.PuestoRepo != nil {
		regimenes, _ = h.PuestoRepo.ObtenerRegimenes()
	}

	metas, _ := h.Repo.ObtenerMetas(tenantID)

	var unidades []models.UnidadOrganica
	if h.OrganigramaRepo != nil {
		orgActivo, errOrg := h.OrganigramaRepo.ObtenerOrganigramaActivo(tenantID)
		if errOrg == nil && orgActivo != nil {
			unidades, _ = h.OrganigramaRepo.ObtenerUnidades(orgActivo.ID)
		}
	}

	// Precargar trabajadores (Página 1) para evitar el parpadeo de "Cargando..." al ingresar
	limite := 20
	pagina := 1
	trabajadores, total, _ := h.Repo.ObtenerTrabajadoresEspecialPaginacion(tenantID, "", 0, 0, 0, limite, 0)
	totalPaginas := (total + limite - 1) / limite
	if totalPaginas == 0 {
		totalPaginas = 1
	}

	// Obtener formulación existente si ya fue procesada anteriormente
	formulacionConceptos, formulacionTrabajadores, errForm := h.Repo.ObtenerFormulacionEspecial(planillaID, tenantID)
	if errForm != nil {
		log.Println("❌ ERROR EN ObtenerFormulacionEspecial:", errForm)
	}
	log.Printf("📊 VistaFormuladorEspecial: PlanillaID=%d, Conceptos=%d, Trabajadores=%d", planillaID, len(formulacionConceptos), len(formulacionTrabajadores))

	formulacionTrabajadoresJSON := "[]"
	if len(formulacionTrabajadores) > 0 {
		if jsonBytes, err := json.Marshal(formulacionTrabajadores); err == nil {
			formulacionTrabajadoresJSON = string(jsonBytes)
		}
	}

	datos := map[string]interface{}{
		"Planilla":                    planilla,
		"ConceptosTenant":             conceptosTenant,
		"Regimenes":                   regimenes,
		"Metas":                       metas,
		"Unidades":                    unidades,
		"FormulacionConceptos":        formulacionConceptos,
		"FormulacionTrabajadoresJSON": template.JS(formulacionTrabajadoresJSON),

		"Trabajadores":    trabajadores,
		"TotalRegistros":  total,
		"PaginaActual":    pagina,
		"TotalPaginas":    totalPaginas,
		"PaginaAnterior":  0,
		"PaginaSiguiente": 2,
	}

	tmpl, err := template.ParseFiles("ui/templates/tenant/planilla_especial_ui.html")
	if err != nil {
		log.Println("❌ ERROR al parsear planilla_especial_ui.html:", err)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`
			<article style="background-color: #ffcdd2; color: #b71c1c; padding: 1rem; margin: 1rem; border-radius: 5px;">
				❌ Error al cargar la vista de planilla especial: ` + err.Error() + `
			</article>
		`))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = tmpl.Execute(w, datos)
	if err != nil {
		log.Println("❌ ERROR Ejecutando planilla_especial_ui.html:", err)
		w.Write([]byte(`
			<article style="background-color: #ffcdd2; color: #b71c1c; padding: 1rem; margin: 1rem; border-radius: 5px;">
				❌ Error al renderizar la vista de planilla especial: ` + err.Error() + `
			</article>
		`))
	}
}

// BuscarTrabajadoresEspecial devuelve la tabla paginada de trabajadores filtrados para HTMX
func (h *PlanillaHandler) BuscarTrabajadoresEspecial(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	tenantID := obtenerTenantID(r)
	busqueda := strings.TrimSpace(r.FormValue("q"))
	regimenID, _ := strconv.Atoi(r.FormValue("regimen_id"))
	metaID, _ := strconv.Atoi(r.FormValue("meta_id"))
	unidadID, _ := strconv.Atoi(r.FormValue("unidad_organica_id"))

	pagina, _ := strconv.Atoi(r.FormValue("pagina"))
	if pagina <= 0 {
		pagina = 1
	}
	limite, _ := strconv.Atoi(r.FormValue("limite"))
	if limite <= 0 {
		limite = 20
	}
	offset := (pagina - 1) * limite

	trabajadores, total, err := h.Repo.ObtenerTrabajadoresEspecialPaginacion(tenantID, busqueda, regimenID, metaID, unidadID, limite, offset)
	if err != nil {
		http.Error(w, "Error al obtener trabajadores: "+err.Error(), http.StatusInternalServerError)
		return
	}

	totalPaginas := (total + limite - 1) / limite
	if totalPaginas == 0 {
		totalPaginas = 1
	}

	paginaAnterior := pagina - 1
	paginaSiguiente := pagina + 1
	if paginaSiguiente > totalPaginas {
		paginaSiguiente = totalPaginas
	}

	datos := map[string]interface{}{
		"Trabajadores":    trabajadores,
		"TotalRegistros":  total,
		"PaginaActual":    pagina,
		"TotalPaginas":    totalPaginas,
		"PaginaAnterior":  paginaAnterior,
		"PaginaSiguiente": paginaSiguiente,
	}

	tmpl, err := template.ParseFiles("ui/templates/tenant/planilla_especial_ui.html")
	if err != nil {
		http.Error(w, "Error al cargar fragmento de trabajadores: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmpl.ExecuteTemplate(w, "tabla_trabajadores_modal", datos)
}

// BuscarTrabajadoresEspecialJSON devuelve la lista completa de trabajadores que coinciden con los filtros en formato JSON (para inclusión masiva aditiva)
func (h *PlanillaHandler) BuscarTrabajadoresEspecialJSON(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	tenantID := obtenerTenantID(r)
	busqueda := strings.TrimSpace(r.FormValue("q"))
	regimenID, _ := strconv.Atoi(r.FormValue("regimen_id"))
	metaID, _ := strconv.Atoi(r.FormValue("meta_id"))
	unidadID, _ := strconv.Atoi(r.FormValue("unidad_organica_id"))

	trabajadores, err := h.Repo.ObtenerTrabajadoresEspecialTodos(tenantID, busqueda, regimenID, metaID, unidadID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Error al obtener trabajadores: " + err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success":      true,
		"total":        len(trabajadores),
		"trabajadores": trabajadores,
	})
}

// ProcesarEspecial ejecuta el procesamiento de la planilla extraordinaria/especial
func (h *PlanillaHandler) ProcesarEspecial(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error al procesar formulario: "+err.Error(), http.StatusBadRequest)
		return
	}

	tenantID := obtenerTenantID(r)
	planillaID, _ := strconv.Atoi(r.FormValue("planilla_id"))
	if planillaID <= 0 {
		http.Error(w, "ID de planilla no válido", http.StatusBadRequest)
		return
	}

	log.Println("----Procesar-Especial----")
	log.Println("Planilla ID:", planillaID)

	// Parsear conceptos seleccionados y sus montos
	conceptosIDsStr := r.Form["conceptos_ids"]
	var conceptosInput []repository.PlanillaEspecialConceptoInput
	for _, cIDStr := range conceptosIDsStr {
		cID, err := strconv.Atoi(cIDStr)
		if err == nil && cID > 0 {
			montoStr := r.FormValue(fmt.Sprintf("monto_%d", cID))
			monto, _ := strconv.ParseFloat(montoStr, 64)
			conceptosInput = append(conceptosInput, repository.PlanillaEspecialConceptoInput{
				ConceptoTenantID: cID,
				Monto:            monto,
			})
		}
	}

	if len(conceptosInput) == 0 {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`
			<div id="alerta-especial" hx-swap-oob="true">
				<article style="background-color: #ffcdd2; color: #b71c1c; padding: 1rem; margin-bottom: 1rem; border-radius: 5px;">
					❌ Debe seleccionar al menos un concepto e ingresar un monto mayor a 0.00.
				</article>
			</div>
		`))
		return
	}

	log.Println("Llegamos al final del parseo conceptos seleccionados")

	// Parsear trabajadores/contratos seleccionados
	contratosIDsStr := r.Form["contratos_ids"]
	var contratosIDs []int
	for _, idStr := range contratosIDsStr {
		id, err := strconv.Atoi(idStr)
		if err == nil && id > 0 {
			contratosIDs = append(contratosIDs, id)
		}
	}

	if len(contratosIDs) == 0 {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`
			<div id="alerta-especial" hx-swap-oob="true">
				<article style="background-color: #ffcdd2; color: #b71c1c; padding: 1rem; margin-bottom: 1rem; border-radius: 5px;">
					❌ Debe marcar al menos un trabajador beneficiario.
				</article>
			</div>
		`))
		return
	}

	log.Println("Llegamos al final del parseo contratos seleccionados")

	// Parsear montos personalizados por contrato/concepto si existen
	montosCustom := make(map[string]float64)
	for key, values := range r.Form {
		if strings.HasPrefix(key, "monto_custom_") && len(values) > 0 {
			if val, err := strconv.ParseFloat(values[0], 64); err == nil && val >= 0 {
				montosCustom[key] = val
			}
		}
	}

	log.Println("Llegamos al final del parseo montos personalizados")

	// Ejecutar procesamiento en repositorio
	err := h.Repo.ProcesarPlanillaEspecial(r.Context(), planillaID, tenantID, conceptosInput, contratosIDs, montosCustom)
	if err != nil {
		log.Println("❌ ERROR EN ProcesarPlanillaEspecial:", err)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`
			<div id="alerta-especial">
				<article style="background-color: #ffcdd2; color: #b71c1c; padding: 1rem; margin-bottom: 1rem; border-radius: 5px;">
					❌ Error procesando la planilla especial: ` + err.Error() + `
				</article>
			</div>
		`))
		return
	}

	log.Println("🎉 PLANILLA ESPECIAL PROCESADA EXITOSAMENTE. CARGANDO VISTA DETALLE...")

	// Redirigir suavemente a la vista de detalle de la planilla
	w.Header().Set("HX-Push-Url", "/tenant/planillas/detalle/ui?id="+strconv.Itoa(planillaID))
	r.URL.RawQuery = "id=" + strconv.Itoa(planillaID)
	h.VistaDetalle(w, r)
}
