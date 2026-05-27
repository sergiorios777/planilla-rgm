package handlers

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"planilla-rgm/internal/repository"
	"strings"
)

type TenantHandler struct {
	Repo *repository.TenantRepository
}

// PerfilUI muestra la información actual de la municipalidad
func (h *TenantHandler) PerfilUI(w http.ResponseWriter, r *http.Request) {
	// Extraemos el tenant_id de la sesión ( JWT )
	var tID int
	if val, ok := r.Context().Value("tenant_id").(float64); ok {
		tID = int(val)
	} else {
		http.Error(w, "Acceso no autorizado: no se encontró tenant en la sesión", http.StatusUnauthorized)
		return
	}

	tenant, err := h.Repo.ObtenerPorID(tID)
	if err != nil || tenant == nil {
		http.Error(w, "No se encontró el perfil institucional", http.StatusNotFound)
		return
	}

	tmpl, err := template.ParseFiles("ui/templates/tenant/perfil_ui.html")
	if err != nil {
		http.Error(w, "Error al cargar plantilla: "+err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, tenant)
}

// ActualizarPerfil procesa datos y el archivo del logo
func (h *TenantHandler) ActualizarPerfil(w http.ResponseWriter, r *http.Request) {
	// 1. Limitar el tamaño del archivo (2MB)
	err := r.ParseMultipartForm(2 << 20)
	if err != nil {
		http.Error(w, "Archivo demasiado grande o formulario inválido", http.StatusBadRequest)
		return
	}

	// Extraemos el tenant_id seguro de la sesión en lugar de confiar en el formulario HTML
	var tID int
	if val, ok := r.Context().Value("tenant_id").(float64); ok {
		tID = int(val)
	} else {
		http.Error(w, "Acceso no autorizado", http.StatusUnauthorized)
		return
	}

	tenant, err := h.Repo.ObtenerPorID(tID)
	if err != nil || tenant == nil {
		http.Error(w, "Perfil institucional no encontrado", http.StatusNotFound)
		return
	}

	// 2. Procesar el Logo si se subió uno nuevo
	file, header, err := r.FormFile("logo")
	if err == nil {
		defer file.Close()

		// Crear nombre único: logo_ID_nombre.ext
		ext := filepath.Ext(header.Filename)
		nombreArchivo := fmt.Sprintf("logo_%d%s", tID, ext)
		rutaDestino := filepath.Join("ui", "static", "uploads", "logos", nombreArchivo)

		// Asegurar que la ruta exista y crear el archivo
		dst, err := os.Create(rutaDestino)
		if err == nil {
			defer dst.Close()
			io.Copy(dst, file)

			// Actualizar URL en el modelo
			url := "/static/uploads/logos/" + nombreArchivo
			tenant.LogoURL = &url
		}
	}

	// 3. Actualizar otros campos (mapeando a nil si están vacíos para guardarse como NULL)
	var dir *string
	if dirVal := strings.TrimSpace(r.FormValue("direccion")); dirVal != "" {
		dir = &dirVal
	}
	var frase *string
	if fraseVal := strings.TrimSpace(r.FormValue("frase_gestion")); fraseVal != "" {
		frase = &fraseVal
	}
	var slug *string
	if slugVal := strings.TrimSpace(strings.ToLower(r.FormValue("slug"))); slugVal != "" {
		slug = &slugVal
	}

	tenant.Direccion = dir
	tenant.FraseGestion = frase
	tenant.Slug = slug
	tenant.Nombre = r.FormValue("nombre")

	err = h.Repo.Actualizar(tenant)
	if err != nil {
		http.Error(w, "Error al guardar en la base de datos: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Recargar la vista del perfil
	h.PerfilUI(w, r)
}
