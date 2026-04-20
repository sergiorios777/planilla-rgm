package handlers

import (
	"html/template"
	"net/http"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
	service "planilla-rgm/internal/services"
	"strconv"
)

type UsuarioHandler struct {
	UserRepo   *repository.UsuarioRepository
	TenantRepo *repository.TenantRepository
}

// VistaUI carga la estructura principal y le envía los inquilinos para el select
func (h *UsuarioHandler) VistaUI(w http.ResponseWriter, r *http.Request) {
	// Traemos todos los inquilinos para el menú desplegable
	inquilinos, _ := h.TenantRepo.ObtenerTodos()

	tmpl, _ := template.ParseFiles("ui/templates/admin/usuarios_ui.html")
	tmpl.Execute(w, inquilinos) // Le pasamos los inquilinos al renderizar la vista base
}

// Listar devuelve solo la tabla de usuarios
func (h *UsuarioHandler) Listar(w http.ResponseWriter, r *http.Request) {
	usuarios, _ := h.UserRepo.ObtenerTodos()
	tmpl, _ := template.ParseFiles("ui/templates/admin/usuarios_ui.html")
	tmpl.ExecuteTemplate(w, "tabla_usuarios", usuarios)
}

// Crear procesa el formulario, encripta la clave y guarda
func (h *UsuarioHandler) Crear(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	tenantIDStr := r.FormValue("tenant_id")
	activo := r.FormValue("activo") == "on"
	var tenantID *int

	// Si seleccionó una municipalidad, convertimos el ID. Si es vacío, queda como nil (Súper Admin)
	if tenantIDStr != "" {
		id, _ := strconv.Atoi(tenantIDStr)
		tenantID = &id
	}

	// Encriptamos la contraseña obligatoriamente
	hash, err := service.HashPassword(r.FormValue("password"))
	if err != nil {
		http.Error(w, "Error al encriptar contraseña", http.StatusInternalServerError)
		return
	}

	nuevoUsuario := models.Usuario{
		TenantID: tenantID,
		Nombre:   r.FormValue("nombre"),
		Email:    r.FormValue("email"),
		Password: hash,
		Rol:      r.FormValue("rol"),
		Activo:   activo,
	}

	h.UserRepo.Crear(&nuevoUsuario)

	// Devolvemos la tabla actualizada
	h.Listar(w, r)
}

// EditarUI devuelve el formulario precargado
func (h *UsuarioHandler) EditarUI(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))

	usuario, err := h.UserRepo.ObtenerPorID(id)
	if err != nil {
		http.Error(w, "Usuario no encontrado", http.StatusNotFound)
		return
	}

	inquilinos, _ := h.TenantRepo.ObtenerTodos()

	// Truco: Go maneja TenantID como un puntero (*int). Para hacer la comparación
	// en el HTML fácilmente, sacamos su valor a una variable entera simple.
	tenantActual := 0
	if usuario.TenantID != nil {
		tenantActual = *usuario.TenantID
	}

	// Enviamos todos los datos empaquetados en un mapa
	datosVista := map[string]interface{}{
		"Usuario":      usuario,
		"Inquilinos":   inquilinos,
		"TenantActual": tenantActual,
	}

	tmpl, _ := template.ParseFiles("ui/templates/admin/usuarios_ui.html")
	tmpl.ExecuteTemplate(w, "formulario_editar", datosVista)
}

// ActualizarUsuario procesa el formulario de edición
func (h *UsuarioHandler) ActualizarUsuario(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	id, _ := strconv.Atoi(r.FormValue("id"))

	tenantIDStr := r.FormValue("tenant_id")
	activo := r.FormValue("activo") == "on"
	var tenantID *int
	if tenantIDStr != "" {
		tid, _ := strconv.Atoi(tenantIDStr)
		tenantID = &tid
	}

	usuarioEditado := models.Usuario{
		ID:       id,
		TenantID: tenantID,
		Nombre:   r.FormValue("nombre"),
		Email:    r.FormValue("email"),
		Rol:      r.FormValue("rol"),
		Activo:   activo,
	}

	// Si escribió algo en la contraseña, la encriptamos. Si no, la dejamos vacía.
	passwordForm := r.FormValue("password")
	if passwordForm != "" {
		hash, _ := service.HashPassword(passwordForm)
		usuarioEditado.Password = hash
	}

	h.UserRepo.Actualizar(&usuarioEditado)

	// Devolvemos la vista principal para recargar la tabla y resetear el formulario
	h.VistaUI(w, r)
}
