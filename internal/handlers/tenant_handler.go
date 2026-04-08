package handlers

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"planilla-rgm/internal/middleware"
	"planilla-rgm/internal/repository"
	"strconv"
	"strings"
)

type TenantHandler struct {
	Repo *repository.TenantRepository
}

// PerfilUI muestra la información actual de la municipalidad
func (h *TenantHandler) PerfilUI(w http.ResponseWriter, r *http.Request) {
	// Extraemos el tenant_id del JWT (gracias al middleware)
	tenantID := r.Context().Value(middleware.UsuarioIDKey).(float64) // El JWT guarda números como float64

	// En un paso real, el middleware debería darnos el TenantID directamente.
	// Por ahora, asumiremos que el usuario logueado tiene un tenant_id asociado.
	// (Asegúrate de haber guardado el tenant_id en el contexto en tu middleware)

	// Para esta demo, usaremos el ID que viene en el contexto (debes ajustar tu middleware para pasar TenantID)
	tID := int(tenantID)

	tenant, _ := h.Repo.ObtenerPorID(tID)

	tmpl, _ := template.ParseFiles("ui/templates/tenant/perfil_ui.html")
	tmpl.Execute(w, tenant)
}

// ActualizarPerfil procesa datos y el archivo del logo
func (h *TenantHandler) ActualizarPerfil(w http.ResponseWriter, r *http.Request) {
	// 1. Limitar el tamaño del archivo (2MB)
	r.ParseMultipartForm(2 << 20)

	tID, _ := strconv.Atoi(r.FormValue("id"))
	tenant, _ := h.Repo.ObtenerPorID(tID)

	// 2. Procesar el Logo si se subió uno nuevo
	file, header, err := r.FormFile("logo")
	if err == nil {
		defer file.Close()

		// Crear nombre único: logo_ID_nombre.ext
		ext := filepath.Ext(header.Filename)
		nombreArchivo := fmt.Sprintf("logo_%d%s", tID, ext)
		rutaDestino := filepath.Join("ui", "static", "uploads", "logos", nombreArchivo)

		// Guardar archivo en disco
		dst, _ := os.Create(rutaDestino)
		defer dst.Close()
		io.Copy(dst, file)

		// Actualizar URL en el modelo
		url := "/static/uploads/logos/" + nombreArchivo
		tenant.LogoURL = &url
	}

	// 3. Actualizar otros campos
	dir := r.FormValue("direccion")
	frase := r.FormValue("frase_gestion")
	slug := strings.ToLower(r.FormValue("slug"))

	tenant.Direccion = &dir
	tenant.FraseGestion = &frase
	tenant.Slug = &slug
	tenant.Nombre = r.FormValue("nombre")

	h.Repo.Actualizar(tenant)

	// Recargar la vista con un mensaje de éxito (o simplemente refrescar)
	h.PerfilUI(w, r)
}
