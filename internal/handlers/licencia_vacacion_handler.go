package handlers

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
	"planilla-rgm/internal/services"
	"strconv"
	"strings"
	"time"
)

type LicenciaVacacionHandler struct {
	Service        *services.LicenciaVacacionService
	Repo           *repository.LicenciaVacacionRepository
	TrabajadorRepo *repository.TrabajadorRepository
}

func NewLicenciaVacacionHandler(repo *repository.LicenciaVacacionRepository, trabajadorRepo *repository.TrabajadorRepository) *LicenciaVacacionHandler {
	return &LicenciaVacacionHandler{
		Service:        services.NewLicenciaVacacionService(repo),
		Repo:           repo,
		TrabajadorRepo: trabajadorRepo,
	}
}

// VistaUI renderiza la vista principal del módulo de vacaciones y licencias
func (h *LicenciaVacacionHandler) VistaUI(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	if tenantID == 0 {
		http.Error(w, "Acceso no autorizado", http.StatusUnauthorized)
		return
	}

	now := time.Now()
	anio := now.Year()
	mes := int(now.Month())

	kpis, err := h.Service.ObtenerKPIs(tenantID, anio, mes)
	if err != nil {
		log.Printf("Error obteniendo KPIs de vacaciones: %v", err)
		kpis = &models.KpisLicenciaVacacion{}
	}

	contratos, _ := h.Repo.ObtenerContratosActivosSelect(tenantID)
	catalogoSunat, _ := h.Repo.ObtenerTiposSuspensionSunat()
	lista, _ := h.Service.Listar(tenantID, "", "TODOS", "TODOS", 0, 0)

	datos := map[string]interface{}{
		"KPIs":          kpis,
		"Lista":         lista,
		"Contratos":     contratos,
		"CatalogoSunat": catalogoSunat,
		"AnioActual":    anio,
		"MesActual":     mes,
	}

	tmpl, err := template.ParseFiles("ui/templates/tenant/vacaciones_licencias_ui.html")
	if err != nil {
		http.Error(w, "Error cargando plantilla de vacaciones: "+err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, datos)
}

// Listar retorna únicamente el fragmento HTML de la tabla con los filtros aplicados
func (h *LicenciaVacacionHandler) Listar(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	if tenantID == 0 {
		http.Error(w, "Acceso no autorizado", http.StatusUnauthorized)
		return
	}

	buscar := strings.TrimSpace(r.URL.Query().Get("buscar"))
	tipo := r.URL.Query().Get("tipo")
	estado := r.URL.Query().Get("estado")

	anio, _ := strconv.Atoi(r.URL.Query().Get("anio"))
	mes, _ := strconv.Atoi(r.URL.Query().Get("mes"))

	lista, err := h.Service.Listar(tenantID, buscar, tipo, estado, anio, mes)
	if err != nil {
		http.Error(w, "Error listando registros: "+err.Error(), http.StatusInternalServerError)
		return
	}

	datos := map[string]interface{}{
		"Lista": lista,
	}

	tmpl, err := template.ParseFiles("ui/templates/tenant/vacaciones_licencias_ui.html")
	if err != nil {
		http.Error(w, "Error cargando plantilla: "+err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "tabla_registros", datos)
}

// ModalCrearUI renderiza el formulario dentro del modal para registrar nueva vacación o licencia
func (h *LicenciaVacacionHandler) ModalCrearUI(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	if tenantID == 0 {
		http.Error(w, "Acceso no autorizado", http.StatusUnauthorized)
		return
	}

	contratos, _ := h.Repo.ObtenerContratosActivosSelect(tenantID)
	catalogoSunat, _ := h.Repo.ObtenerTiposSuspensionSunat()

	datos := map[string]interface{}{
		"Contratos":     contratos,
		"CatalogoSunat": catalogoSunat,
		"FechaHoy":      time.Now().Format("2006-01-02"),
	}

	tmpl, err := template.ParseFiles("ui/templates/tenant/vacaciones_licencias_ui.html")
	if err != nil {
		http.Error(w, "Error cargando modal: "+err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "formulario_crear_modal", datos)
}

// Crear procesa la inserción de una nueva vacación o licencia
func (h *LicenciaVacacionHandler) Crear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	tenantID := obtenerTenantID(r)
	if tenantID == 0 {
		http.Error(w, "Acceso no autorizado", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error al procesar formulario", http.StatusBadRequest)
		return
	}

	contratoID, _ := strconv.Atoi(r.FormValue("contrato_id"))
	tipo := r.FormValue("tipo")
	subtipo := r.FormValue("subtipo")
	codigoSunat := r.FormValue("codigo_sunat_suspension")
	fechaInicio := r.FormValue("fecha_inicio")
	fechaFin := r.FormValue("fecha_fin")
	docAprob := r.FormValue("documento_aprobacion")
	fechaAprob := r.FormValue("fecha_aprobacion")
	observaciones := r.FormValue("observaciones")
	estado := r.FormValue("estado")
	if estado == "" {
		estado = "APROBADO"
	}

	var cIDVal *int
	if contratoID > 0 {
		cIDVal = &contratoID
	}

	var fAprobVal *string
	if strings.TrimSpace(fechaAprob) != "" {
		fAprobVal = &fechaAprob
	}

	item := &models.LicenciaVacacion{
		TenantID:              tenantID,
		ContratoID:            cIDVal,
		Tipo:                  tipo,
		Subtipo:               subtipo,
		CodigoSunatSuspension: codigoSunat,
		FechaInicio:           fechaInicio,
		FechaFin:              fechaFin,
		DocumentoAprobacion:   docAprob,
		FechaAprobacion:       fAprobVal,
		Observaciones:         observaciones,
		Estado:                estado,
	}

	err := h.Service.Crear(tenantID, item)
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf(`
			<div id="alerta-error-modal" hx-swap-oob="true">
				<div class="alert-box alert-danger mb-md">
					<strong>⚠️ Error:</strong> %s
				</div>
			</div>
		`, err.Error())))
		return
	}

	// Refrescar KPIs y cerrar modal
	now := time.Now()
	kpis, _ := h.Service.ObtenerKPIs(tenantID, now.Year(), int(now.Month()))

	w.Header().Set("HX-Trigger", `{"recargarTablaVacaciones": true, "cerrarModalVacacion": true}`)
	tmpl, _ := template.ParseFiles("ui/templates/tenant/vacaciones_licencias_ui.html")
	tmpl.ExecuteTemplate(w, "kpis_bento", map[string]interface{}{"KPIs": kpis})
}

// ModalEditarUI renderiza el formulario de edición de una incidencia
func (h *LicenciaVacacionHandler) ModalEditarUI(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	if tenantID == 0 {
		http.Error(w, "Acceso no autorizado", http.StatusUnauthorized)
		return
	}

	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	item, err := h.Repo.ObtenerPorID(id, tenantID)
	if err != nil || item == nil {
		http.Error(w, "Registro no encontrado", http.StatusNotFound)
		return
	}

	contratos, _ := h.Repo.ObtenerContratosActivosSelect(tenantID)
	catalogoSunat, _ := h.Repo.ObtenerTiposSuspensionSunat()

	datos := map[string]interface{}{
		"Item":          item,
		"Contratos":     contratos,
		"CatalogoSunat": catalogoSunat,
	}

	tmpl, err := template.ParseFiles("ui/templates/tenant/vacaciones_licencias_ui.html")
	if err != nil {
		http.Error(w, "Error cargando modal de edición: "+err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "formulario_editar_modal", datos)
}

// Actualizar procesa la modificación de una vacación o licencia
func (h *LicenciaVacacionHandler) Actualizar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	tenantID := obtenerTenantID(r)
	if tenantID == 0 {
		http.Error(w, "Acceso no autorizado", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error al procesar formulario", http.StatusBadRequest)
		return
	}

	id, _ := strconv.Atoi(r.FormValue("id"))
	tipo := r.FormValue("tipo")
	subtipo := r.FormValue("subtipo")
	codigoSunat := r.FormValue("codigo_sunat_suspension")
	fechaInicio := r.FormValue("fecha_inicio")
	fechaFin := r.FormValue("fecha_fin")
	docAprob := r.FormValue("documento_aprobacion")
	fechaAprob := r.FormValue("fecha_aprobacion")
	observaciones := r.FormValue("observaciones")
	estado := r.FormValue("estado")

	var fAprobVal *string
	if strings.TrimSpace(fechaAprob) != "" {
		fAprobVal = &fechaAprob
	}

	// Obtener registro previo para conservar trabajador_id y contrato_id
	previo, err := h.Repo.ObtenerPorID(id, tenantID)
	if err != nil || previo == nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`<div class="alert-box alert-danger">Registro no encontrado</div>`))
		return
	}

	previo.Tipo = tipo
	previo.Subtipo = subtipo
	previo.CodigoSunatSuspension = codigoSunat
	previo.FechaInicio = fechaInicio
	previo.FechaFin = fechaFin
	previo.DocumentoAprobacion = docAprob
	previo.FechaAprobacion = fAprobVal
	previo.Observaciones = observaciones
	previo.Estado = estado

	err = h.Service.Actualizar(tenantID, previo)
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf(`
			<div id="alerta-error-modal-edit" hx-swap-oob="true">
				<div class="alert-box alert-danger mb-md">
					<strong>⚠️ Error:</strong> %s
				</div>
			</div>
		`, err.Error())))
		return
	}

	now := time.Now()
	kpis, _ := h.Service.ObtenerKPIs(tenantID, now.Year(), int(now.Month()))

	w.Header().Set("HX-Trigger", `{"recargarTablaVacaciones": true, "cerrarModalVacacion": true}`)
	tmpl, _ := template.ParseFiles("ui/templates/tenant/vacaciones_licencias_ui.html")
	tmpl.ExecuteTemplate(w, "kpis_bento", map[string]interface{}{"KPIs": kpis})
}

// Eliminar borra un registro de vacaciones o licencia
func (h *LicenciaVacacionHandler) Eliminar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	tenantID := obtenerTenantID(r)
	if tenantID == 0 {
		http.Error(w, "Acceso no autorizado", http.StatusUnauthorized)
		return
	}

	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	if id <= 0 {
		id, _ = strconv.Atoi(r.FormValue("id"))
	}

	err := h.Service.Eliminar(id, tenantID)
	if err != nil {
		http.Error(w, "Error al eliminar: "+err.Error(), http.StatusInternalServerError)
		return
	}

	now := time.Now()
	kpis, _ := h.Service.ObtenerKPIs(tenantID, now.Year(), int(now.Month()))

	w.Header().Set("HX-Trigger", `{"recargarTablaVacaciones": true}`)
	tmpl, _ := template.ParseFiles("ui/templates/tenant/vacaciones_licencias_ui.html")
	tmpl.ExecuteTemplate(w, "kpis_bento", map[string]interface{}{"KPIs": kpis})
}
