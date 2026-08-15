package handlers

import (
	"html/template"
	"log"
	"net/http"
)

type LandingHandler struct{}

func NewLandingHandler() *LandingHandler {
	return &LandingHandler{}
}

// MostrarLanding renderiza la vista pública principal de la landing page.
func (h *LandingHandler) MostrarLanding(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	tmpl, err := template.ParseFiles("ui/templates/landing/index.html")
	if err != nil {
		log.Printf("Error cargando plantilla de la landing page: %v", err)
		http.Error(w, "Error interno del servidor al cargar la vista de bienvenida", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title": "Planilla RGM - SaaS de Planillas para Municipalidades del Perú",
		"Year":  2026,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error al renderizar landing page: %v", err)
	}
}

// MostrarSolicitudDemo renderiza la vista con el formulario para solicitar demo.
func (h *LandingHandler) MostrarSolicitudDemo(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("ui/templates/landing/solicitud.html")
	if err != nil {
		log.Printf("Error cargando plantilla de solicitud de demo: %v", err)
		http.Error(w, "Error interno del servidor al cargar la vista de solicitud", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title": "Solicitar Demo - Planilla RGM",
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error al renderizar solicitud de demo: %v", err)
	}
}

// ProcesarSolicitudDemo procesa la información enviada desde el formulario y responde mediante HTMX.
func (h *LandingHandler) ProcesarSolicitudDemo(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error al procesar el formulario", http.StatusBadRequest)
		return
	}

	nombre := r.FormValue("nombre")
	entidad := r.FormValue("entidad")
	email := r.FormValue("email")

	log.Printf("[DEMO REQUEST] Nombre: %s | Entidad: %s | Email: %s", nombre, entidad, email)

	successHTML := `
	<div class="solicitud-card solicitud-success-card" id="form-contenedor">
		<div class="solicitud-success-icon">🎉</div>
		<h2 style="color: var(--landing-tertiary); margin-bottom: 0.5rem;">¡Solicitud Recibida con Éxito!</h2>
		<p style="color: var(--landing-text-variant); max-width: 500px; margin: 0 auto 1.5rem;">
			Gracias <strong>` + template.HTMLEscapeString(nombre) + `</strong>. Hemos registrado la solicitud para la <strong>` + template.HTMLEscapeString(entidad) + `</strong>.
		</p>
		<p style="font-size: 0.9rem; color: var(--landing-text-variant); margin-bottom: 2rem;">
			Un especialista en gestión pública de <strong>Planilla RGM</strong> se pondrá en contacto al correo <code>` + template.HTMLEscapeString(email) + `</code> para agendar la presentación ejecutiva.
		</p>
		<a href="/" class="btn-hero-primary">Volver a la Página Principal</a>
	</div>
	`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(successHTML))
}
