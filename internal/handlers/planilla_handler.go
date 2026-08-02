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
	Repo         *repository.PlanillaRepository
	AnexoService *services.AnexoService
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

	nuevaPlanilla := models.Planilla{
		TenantID:    tenantID,
		Anio:        anio,
		Mes:         mes,
		Descripcion: r.FormValue("descripcion"),
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

// Procesar ejecuta el motor de cálculo y redirige a la vista de detalle
func (h *PlanillaHandler) Procesar(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	planillaID, _ := strconv.Atoi(r.URL.Query().Get("id"))

	servicioPlanilla := services.NewPlanillaService(h.Repo)
	err := servicioPlanilla.Procesar(planillaID, tenantID)
	if err != nil {
		// Si el motor falla, mostramos una alerta sin salir de la página
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`
			<div id="alerta-planilla" hx-swap-oob="true">
				<article style="background-color: #ffcdd2; color: #b71c1c; padding: 1rem; margin-bottom: 1rem; border-radius: 5px;">
					❌ Error en el Motor de Cálculo: ` + err.Error() + `
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



