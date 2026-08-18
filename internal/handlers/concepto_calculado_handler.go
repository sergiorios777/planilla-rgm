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
)

type ConceptoCalculadoHandler struct {
	Repo                  *repository.BaseRegimenRepository
	PuestoRepo            *repository.PuestoRepository
	ConceptoModeloService *services.ConceptoModeloService
	TenantRepo            *repository.TenantRepository
}

func NewConceptoCalculadoHandler(repo *repository.BaseRegimenRepository, puestoRepo *repository.PuestoRepository, cms *services.ConceptoModeloService, tenantRepo *repository.TenantRepository) *ConceptoCalculadoHandler {
	return &ConceptoCalculadoHandler{
		Repo:                  repo,
		PuestoRepo:            puestoRepo,
		ConceptoModeloService: cms,
		TenantRepo:            tenantRepo,
	}
}

// VistaUI carga la página principal del módulo
func (h *ConceptoCalculadoHandler) VistaUI(w http.ResponseWriter, r *http.Request) {
	regimenes, _ := h.PuestoRepo.ObtenerRegimenes()

	data := map[string]interface{}{
		"Regimenes": regimenes,
		"Variables": []string{
			"REMUNERACION_BASICA",
			"MUC",
			"BET",
			"BET_FIJO",
			"BET_VARIABLE",
			"RETRIBUCION_MENSUAL",
			"VALORIZACION_PRINCIPAL",
			"VALORIZACION_AJUSTADA",
			"ASIGNACION_FAMILIAR",
			"SEXTO_GRATIFICACION",
			"REMUNERACION_VARIABLE",
			"REMUNERACION_COMPUTABLE",
		},
	}

	tmpl, err := template.ParseFiles("ui/templates/admin/conceptos_calculados_ui.html")
	if err != nil {
		http.Error(w, "Error cargando plantilla: "+err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, data)
}

// Listar extrae la tabla filtrada para HTMX
func (h *ConceptoCalculadoHandler) Listar(w http.ResponseWriter, r *http.Request) {
	calculados, err := h.Repo.ListarConceptosCalculados()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFiles("ui/templates/admin/conceptos_calculados_ui.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "tabla_calculados", calculados)
}

// Crear inserta un nuevo concepto calculado global
func (h *ConceptoCalculadoHandler) Crear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	nombre := strings.TrimSpace(r.FormValue("nombre"))
	tipo := r.FormValue("tipo")
	codigo := strings.ToUpper(strings.TrimSpace(r.FormValue("codigo_interno")))

	if nombre == "" || tipo == "" || codigo == "" {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<p style="color:red; font-weight:bold;">Todos los campos son obligatorios</p>`))
		return
	}

	c := &models.ConceptoCalculado{
		Nombre:        nombre,
		Tipo:          tipo,
		CodigoInterno: codigo,
	}

	err := h.Repo.CrearConceptoCalculado(c)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<p style="color:red; font-weight:bold;">Error: ` + err.Error() + `</p>`))
		return
	}

	w.Header().Set("HX-Trigger", "cerrarModalCalculado")
	h.Listar(w, r)
}

// Eliminar quita un concepto calculado global
func (h *ConceptoCalculadoHandler) Eliminar(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	err := h.Repo.EliminarConceptoCalculado(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.Listar(w, r)
}

// VistaAfectaciones muestra el detalle de afectaciones del concepto seleccionado
func (h *ConceptoCalculadoHandler) VistaAfectaciones(w http.ResponseWriter, r *http.Request) {
	conceptoCalculadoID, _ := strconv.Atoi(r.URL.Query().Get("id"))

	afectaciones, err := h.Repo.ListarAfectacionesDefault(conceptoCalculadoID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Agrupamos afectaciones por Régimen para facilitar la visualización en la UI
	agrupadas := make(map[string][]models.BaseRegimenDefaultDTO)
	for _, a := range afectaciones {
		agrupadas[a.RegimenDesc] = append(agrupadas[a.RegimenDesc], a)
	}

	tmpl, err := template.ParseFiles("ui/templates/admin/conceptos_calculados_ui.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	regimenes, _ := h.PuestoRepo.ObtenerRegimenes()

	data := map[string]interface{}{
		"CalculadoID":  conceptoCalculadoID,
		"Afectaciones": agrupadas,
		"Regimenes":    regimenes,
		"Variables": []string{
			"REMUNERACION_BASICA",
			"MUC",
			"BET",
			"BET_FIJO",
			"BET_VARIABLE",
			"RETRIBUCION_MENSUAL",
			"VALORIZACION_PRINCIPAL",
			"VALORIZACION_AJUSTADA",
			"ASIGNACION_FAMILIAR",
			"SEXTO_GRATIFICACION",
			"REMUNERACION_VARIABLE",
			"REMUNERACION_COMPUTABLE",
		},
	}

	tmpl.ExecuteTemplate(w, "seccion_afectaciones", data)
}

// OpcionesModelo retorna los <option> para el selector de conceptos modelo basado en el régimen seleccionado
func (h *ConceptoCalculadoHandler) OpcionesModelo(w http.ResponseWriter, r *http.Request) {
	regimenID, _ := strconv.Atoi(r.URL.Query().Get("regimen_id"))

	conceptos, err := h.Repo.ObtenerConceptosModeloPorRegimen(regimenID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	html := `<option value="" disabled selected>Seleccione un concepto modelo...</option>`
	for _, c := range conceptos {
		html += fmt.Sprintf(`<option value="%d">%s</option>`, c.ID, c.NombrePersonalizado)
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// AgregarAfectacion inserta una nueva afectación a la plantilla por defecto
func (h *ConceptoCalculadoHandler) AgregarAfectacion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	r.ParseForm()
	calcID, _ := strconv.Atoi(r.FormValue("concepto_calculado_id"))
	regimenID, _ := strconv.Atoi(r.FormValue("regimen_id"))
	modeloID, _ := strconv.Atoi(r.FormValue("concepto_modelo_id"))
	variable := r.FormValue("variable_calculo")

	if calcID == 0 || regimenID == 0 || modeloID == 0 || variable == "" {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<p style="color:red; font-weight:bold;">Todos los campos son obligatorios</p>`))
		return
	}

	err := h.Repo.AgregarAfectacionDefault(calcID, regimenID, modeloID, variable)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<p style="color:red; font-weight:bold;">Error: ` + err.Error() + `</p>`))
		return
	}

	// Recargar la sección de afectaciones
	r.URL.RawQuery = fmt.Sprintf("id=%d", calcID)
	h.VistaAfectaciones(w, r)
}

// EliminarAfectacion remueve una afectación de la plantilla por defecto
func (h *ConceptoCalculadoHandler) EliminarAfectacion(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	calcID, _ := strconv.Atoi(r.URL.Query().Get("calc_id"))

	err := h.Repo.EliminarAfectacionDefault(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Recargar la sección de afectaciones
	r.URL.RawQuery = fmt.Sprintf("id=%d", calcID)
	h.VistaAfectaciones(w, r)
}

// Propagar realiza el proceso de Sembrado para todos los tenants activos
func (h *ConceptoCalculadoHandler) Propagar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	tenants, err := h.TenantRepo.ObtenerTodos("")
	if err != nil {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<span style="color:red; font-weight:bold;">❌ Error al cargar inquilinos</span>`))
		return
	}

	var exitos, fallas int
	for _, tenant := range tenants {
		if !tenant.Activo {
			continue
		}
		err = h.ConceptoModeloService.SembrarBaseRegimenTenant(tenant.ID)
		if err != nil {
			fallas++
			log.Printf("❌ Error propagando reglas a tenant %d: %v", tenant.ID, err)
		} else {
			exitos++
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write(fmt.Appendf(nil, `<span style="color:#2e7d32; font-weight:bold;">✅ Propagación exitosa (%d listos, %d fallas)</span>`, exitos, fallas))
}
