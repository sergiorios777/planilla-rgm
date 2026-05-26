package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"planilla-rgm/internal/config"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/services"
	"time"
)

type ReporteHandler struct {
	Service *services.ReporteService
}

// VistaUI carga la pantalla inicial de reportes
func (h *ReporteHandler) VistaUI(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("ui/templates/tenant/reportes_ui.html")
	if err != nil {
		http.Error(w, "Error cargando la plantilla de reportes: "+err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

// FiltrarUI filtra la lista de reportes en memoria asíncronamente con HTMX
func (h *ReporteHandler) FiltrarUI(w http.ResponseWriter, r *http.Request) {
	modulo := r.URL.Query().Get("modulo")
	if modulo == "" {
		modulo = "TODOS"
	}

	var filtrados []models.Reporte
	for _, rep := range config.ListaReportes {
		if modulo == "TODOS" || rep.Modulo == modulo {
			filtrados = append(filtrados, rep)
		}
	}

	tmpl, err := template.ParseFiles("ui/templates/tenant/reportes_ui.html")
	if err != nil {
		http.Error(w, "Error al filtrar catálogo", http.StatusInternalServerError)
		return
	}

	// Renderizar solo la subplantilla para el target de HTMX
	tmpl.ExecuteTemplate(w, "lista_reportes", map[string]interface{}{
		"Reportes": filtrados,
		"MesActual": int(time.Now().Month()),
	})
}

// ExportarPDF genera un reporte en formato PDF delegándolo al servicio
func (h *ReporteHandler) ExportarPDF(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	id := r.URL.Query().Get("id")

	params := map[string]string{
		"mes":  r.URL.Query().Get("mes"),
		"dias": r.URL.Query().Get("dias"),
	}

	buffer, nombreArchivo, err := h.Service.GenerarPDF(tenantID, id, params)
	if err != nil {
		http.Error(w, "Error al generar el reporte PDF: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", nombreArchivo))
	
	buffer.WriteTo(w)
}

// ExportarExcel genera el archivo en Excel delegándolo al servicio
func (h *ReporteHandler) ExportarExcel(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	id := r.URL.Query().Get("id")

	params := map[string]string{
		"mes":  r.URL.Query().Get("mes"),
		"dias": r.URL.Query().Get("dias"),
	}

	buffer, nombreArchivo, err := h.Service.GenerarExcel(tenantID, id, params)
	if err != nil {
		http.Error(w, "Error al generar el reporte Excel: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", nombreArchivo))

	buffer.WriteTo(w)
}
