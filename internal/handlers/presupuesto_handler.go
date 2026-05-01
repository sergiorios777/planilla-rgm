package handlers

import (
	"html/template"
	"net/http"
	"strconv"

	"planilla-rgm/internal/repository"
	"planilla-rgm/internal/services"
)

// PresupuestoHandler maneja las peticiones web del módulo de Presupuesto Anual
type PresupuestoHandler struct {
	Service      *services.PresupuestoService
	PlanillaRepo *repository.PlanillaRepository // Lo necesitamos para traer los parámetros globales (UIT, etc.)
}

// NewPresupuestoHandler es el constructor
func NewPresupuestoHandler(svc *services.PresupuestoService, pRepo *repository.PlanillaRepository) *PresupuestoHandler {
	return &PresupuestoHandler{
		Service:      svc,
		PlanillaRepo: pRepo,
	}
}

// IndexUI carga la vista principal (panel de control) del módulo
func (h *PresupuestoHandler) IndexUI(w http.ResponseWriter, r *http.Request) {
	// Preparar la vista (crearemos este HTML en la siguiente fase)
	tmpl, err := template.ParseFiles("ui/templates/tenant/presupuesto_ui.html")
	if err != nil {
		http.Error(w, "Error cargando la vista: "+err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.ExecuteTemplate(w, "presupuesto_index", nil)
}

// Generar captura la petición del formulario, recolecta variables y lanza el motor
func (h *PresupuestoHandler) Generar(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	tenantID := obtenerTenantID(r) // Utiliza tu función auxiliar existente

	anioStr := r.FormValue("anio")
	anio, err := strconv.Atoi(anioStr)
	if err != nil {
		w.Write([]byte(`<article style="background-color: #ffebee; color: #c62828;">Error: Año inválido.</article>`))
		return
	}

	// 1. Recolectar parámetros globales y catálogos para el simulador
	// Solicitamos los parámetros asumiendo el mes 1 (Enero) del año solicitado
	parametros, _ := h.PlanillaRepo.ObtenerParametrosGlobales(anio, 1)
	mapaCodigos, _ := h.PlanillaRepo.ObtenerMapaCodigosID()
	mapaAfectaciones, _ := h.PlanillaRepo.ObtenerAfectacionesGlobales()

	// 2. Ejecutar el servicio matemático que construimos anteriormente
	err = h.Service.GenerarProyeccionPIA(tenantID, anio, parametros, mapaCodigos, mapaAfectaciones)
	if err != nil {
		// Retornamos el error formateado para que HTMX lo muestre en pantalla
		w.Write([]byte(`<article style="background-color: #ffebee; color: #c62828;">Error al generar la proyección: ` + err.Error() + `</article>`))
		return
	}

	// 3. Notificar éxito y entregar un botón para cargar la matriz
	mensajeExito := `
	<article style="background-color: #e8f5e9; color: #2e7d32; border-color: #c8e6c9;">
		<strong>¡Éxito!</strong> La proyección del Presupuesto Analítico de Personal (PIA) para el año ` + anioStr + ` ha sido generada y guardada correctamente.
		<div style="margin-top: 1rem;">
			<button class="primary" hx-get="/tenant/presupuesto/matriz?anio=` + anioStr + `" hx-target="#matriz-contenedor">
				👀 Ver Matriz Generada
			</button>
		</div>
	</article>
	<div id="matriz-contenedor" style="margin-top: 2rem;"></div>
	`
	w.Write([]byte(mensajeExito))
}

// CargarMatriz recupera los datos de la BD y renderiza solo la tabla HTML
func (h *PresupuestoHandler) CargarMatriz(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)

	anioStr := r.URL.Query().Get("anio")
	anio, err := strconv.Atoi(anioStr)
	if err != nil || anio == 0 {
		// Por defecto buscamos 2027 si no se envía el parámetro
		anio = 2027
	}

	// Consultamos a la base de datos
	matriz, err := h.Service.PresupuestoRepo.ObtenerMatrizPorAnio(tenantID, anio)
	if err != nil {
		w.Write([]byte(`<article style="background-color: #ffebee; color: #c62828;">Error consultando la base de datos.</article>`))
		return
	}

	if len(matriz) == 0 {
		w.Write([]byte(`<article>No hay datos generados para el año ` + strconv.Itoa(anio) + `. Por favor, genera la proyección primero.</article>`))
		return
	}

	// Renderizamos solo el fragmento de la tabla (lo crearemos en el siguiente paso)
	tmpl, err := template.ParseFiles("ui/templates/tenant/presupuesto_ui.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.ExecuteTemplate(w, "presupuesto_matriz", map[string]interface{}{
		"Anio":   anio,
		"Matriz": matriz,
	})
}
