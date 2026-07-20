package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
	"planilla-rgm/internal/services"
	"strconv"
	"strings"
)

// AFPHandler gestiona el flujo de peticiones web para AFPs y sus Tasas Mensuales
type AFPHandler struct {
	Repo    *repository.AFPRepository
	Service *services.AFPService
}

// VistaUI renderiza el contenedor principal de la vista de AFPs
func (h *AFPHandler) VistaUI(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("ui/templates/admin/afps_ui.html")
	if err != nil {
		http.Error(w, "Error cargando la plantilla principal: "+err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

// ListarAFPs devuelve la tabla o las filas de las AFPs registradas
func (h *AFPHandler) ListarAFPs(w http.ResponseWriter, r *http.Request) {
	busqueda := r.URL.Query().Get("buscar")
	afps, err := h.Repo.ObtenerTodos(busqueda)
	if err != nil {
		http.Error(w, "Error al obtener la lista de AFPs: "+err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFiles("ui/templates/admin/afps_ui.html")
	if err != nil {
		http.Error(w, "Error cargando plantilla: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Si lo solicita HTMX para refrescar la tabla, enviamos solo el bloque
	if r.Header.Get("HX-Request") == "true" && r.Header.Get("HX-Target") == "lista-afps-cuerpo" {
		tmpl.ExecuteTemplate(w, "filas_afps", afps)
		return
	}

	tmpl.ExecuteTemplate(w, "tabla_afps", afps)
}

// CrearAFP procesa el formulario para crear una nueva AFP
func (h *AFPHandler) CrearAFP(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error al leer el formulario", http.StatusBadRequest)
		return
	}

	nombre := strings.TrimSpace(r.FormValue("nombre"))
	codigoSBS := strings.TrimSpace(r.FormValue("codigo_sbs"))
	activo := r.FormValue("activo") == "on"

	if nombre == "" {
		http.Error(w, "El nombre de la AFP es requerido", http.StatusBadRequest)
		return
	}

	afp := models.AFP{
		Nombre:    nombre,
		CodigoSBS: codigoSBS,
		Activo:    activo,
	}

	err = h.Repo.Crear(&afp)
	if err != nil {
		http.Error(w, "Error al guardar AFP en la BD: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", "afpCreada")
	h.ListarAFPs(w, r)
}

// EditarAFPUI retorna el modal de edición de una AFP cargado dinámicamente
func (h *AFPHandler) EditarAFPUI(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	afp, err := h.Repo.ObtenerPorID(id)
	if err != nil {
		http.Error(w, "AFP no encontrada", http.StatusNotFound)
		return
	}

	tmpl, err := template.ParseFiles("ui/templates/admin/afps_ui.html")
	if err != nil {
		http.Error(w, "Error cargando plantilla: "+err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.ExecuteTemplate(w, "formulario_editar_afp", afp)
}

// ActualizarAFP actualiza los datos de una AFP existente
func (h *AFPHandler) ActualizarAFP(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error al leer el formulario", http.StatusBadRequest)
		return
	}

	id, _ := strconv.Atoi(r.FormValue("id"))
	nombre := strings.TrimSpace(r.FormValue("nombre"))
	codigoSBS := strings.TrimSpace(r.FormValue("codigo_sbs"))
	activo := r.FormValue("activo") == "on"

	if nombre == "" {
		http.Error(w, "El nombre de la AFP es requerido", http.StatusBadRequest)
		return
	}

	afp := models.AFP{
		ID:        id,
		Nombre:    nombre,
		CodigoSBS: codigoSBS,
		Activo:    activo,
	}

	err = h.Repo.Actualizar(&afp)
	if err != nil {
		http.Error(w, "Error al actualizar la AFP: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.VistaUI(w, r)
}

// ListarTasas obtiene e inyecta la lista de tasas mensuales para un año/mes
func (h *AFPHandler) ListarTasas(w http.ResponseWriter, r *http.Request) {
	anio, _ := strconv.Atoi(r.URL.Query().Get("anio"))
	if anio <= 0 {
		anio = 2026 // Año por defecto en el sistema
	}

	mes, _ := strconv.Atoi(r.URL.Query().Get("mes"))
	if mes < 1 || mes > 12 {
		mes = 5 // Mayo por defecto
	}

	tasas, err := h.Repo.ObtenerTasasPorMes(anio, mes)
	if err != nil {
		http.Error(w, "Error al obtener tasas mensuales: "+err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFiles("ui/templates/admin/afps_ui.html")
	if err != nil {
		http.Error(w, "Error cargando plantilla: "+err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.ExecuteTemplate(w, "tabla_tasas", tasas)
}

// ImportarCSV recibe el archivo CSV, lo procesa y devuelve el HTML de éxito/error
func (h *AFPHandler) ImportarCSV(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20) // Límite de 10 MB
	if err != nil {
		http.Error(w, "Error al procesar formulario de subida", http.StatusBadRequest)
		return
	}

	anio, _ := strconv.Atoi(r.FormValue("anio"))
	mes, _ := strconv.Atoi(r.FormValue("mes"))

	if anio <= 0 || mes < 1 || mes > 12 {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<span style="color: #b71c1c;">⚠️ Año o Mes inválidos para la importación.</span>`))
		return
	}

	file, _, err := r.FormFile("archivo_csv")
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<span style="color: #b71c1c;">⚠️ No se encontró el archivo CSV en el formulario.</span>`))
		return
	}
	defer file.Close()

	// Procesar a través del servicio
	err = h.Service.ProcesarCSV(file, anio, mes)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		w.Write(fmt.Appendf(nil, `<span style="color: #b71c1c;">⚠️ Error de importación: %s</span>`, err.Error()),
		)
		return
	}

	// Responder éxito, emitiendo cabecera HTMX para recargar la tabla de tasas
	w.Header().Set("HX-Trigger", "tasasActualizadas")
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`<span style="color: #1b5e20;">✔️ Tasas importadas exitosamente. La vista previa se ha actualizado.</span>`))
}
