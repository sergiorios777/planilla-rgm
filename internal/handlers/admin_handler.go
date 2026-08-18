package handlers

import (
	"html/template"
	"log"
	"net/http"

	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"

	"strconv"
)

// AdminHandler agrupa las funciones web del panel del Super Admin
type AdminHandler struct {
	Repo               *repository.TenantRepository
	ConceptoTenantRepo *repository.ConceptoTenantRepository
}

// ListarInquilinos obtiene los datos y los inyecta en una plantilla HTML
func (h *AdminHandler) ListarInquilinos(w http.ResponseWriter, r *http.Request) {
	// 1. Pedimos los datos al repositorio
	busqueda := r.URL.Query().Get("buscar")
	tenants, err := h.Repo.ObtenerTodos(busqueda)
	if err != nil {
		http.Error(w, "Error al obtener la lista de inquilinos de la base de datos", http.StatusInternalServerError)
		return
	}

	// 2. Cargamos el archivo HTML que diseñaremos en el siguiente paso
	// Si la petición la hizo HTMX (al teclear), SOLO devolvemos las filas (<tr>)
	// Usamos GetHeader para detectar esto
	if r.Header.Get("HX-Target") == "true" {
		tmpl, err := template.ParseFiles("ui/templates/admin/inquilinos_ui.html")
		if err != nil {
			http.Error(w, "Error cargando la vista HTML", http.StatusInternalServerError)
			return
		}
		tmpl.ExecuteTemplate(w, "filas_inquilinos", tenants)
		return
	}

	// 3. Si es una página normal, renderizamos todo el contenido
	tmpl, err := template.ParseFiles("ui/templates/admin/inquilinos_ui.html")
	if err != nil {
		http.Error(w, "Error cargando la vista HTML", http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "tabla_inquilinos", tenants)
}

// CrearInquilino recibe los datos del formulario HTMX y guarda la entidad
func (h *AdminHandler) CrearInquilino(w http.ResponseWriter, r *http.Request) {
	// 1. Leemos los datos que vienen del formulario HTML
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error al leer el formulario", http.StatusBadRequest)
		return
	}

	nombre := r.FormValue("nombre")
	ruc := r.FormValue("ruc")
	activo := r.FormValue("activo") == "on"
	tipoEntidad := r.FormValue("tipo_entidad")
	if tipoEntidad == "" {
		tipoEntidad = "GOBIERNO_LOCAL"
	}

	// 2. Preparamos nuestro modelo de Go
	nuevoTenant := models.Tenant{
		Nombre:      nombre,
		Ruc:         ruc,
		Activo:      activo,
		TipoEntidad: tipoEntidad,
	}

	// 3. Le pedimos al repositorio que lo guarde en PostgreSQL
	err = h.Repo.Crear(&nuevoTenant)
	if err != nil {
		http.Error(w, "Error al guardar en la base de datos", http.StatusInternalServerError)
		return
	}

	// 4. Clonamos el catálogo maestro al nuevo inquilino automáticamente
	// (Asumimos que h.Repo.Crear actualiza nuevoTenant.ID con el ID generado)
	if nuevoTenant.ID > 0 {
		errClon := h.ConceptoTenantRepo.ClonarDesdeModelo(nuevoTenant.ID)
		if errClon != nil {
			// Solo logueamos el error, no detenemos el flujo porque el tenant ya se creó
			log.Println("⚠️ Advertencia: Error al clonar conceptos modelo para el tenant", nuevoTenant.ID, ":", errClon)
		}

		errRel := h.ConceptoTenantRepo.ClonarRelacionesRegimen(nuevoTenant.ID)
		if errRel != nil {
			log.Println("⚠️ Advertencia: Error al clonar relaciones de régimen para el tenant", nuevoTenant.ID, ":", errRel)
		}

		errReglas := h.ConceptoTenantRepo.ClonarReglasFinanciamientoModelo(nuevoTenant.ID)
		if errReglas != nil {
			log.Println("⚠️ Advertencia: Error al clonar reglas de financiamiento para el tenant", nuevoTenant.ID, ":", errReglas)
		}
	}

	// 5. TRUCO DE HTMX: En lugar de redirigir, simplemente volvemos a llamar
	// a la función ListarInquilinos para que devuelva la tabla HTML actualizada.
	h.ListarInquilinos(w, r)
}

// VistaUI devuelve la estructura HTML de la página de inquilinos
func (h *AdminHandler) VistaUI(w http.ResponseWriter, r *http.Request) {
	tmpl, _ := template.ParseFiles("ui/templates/admin/inquilinos_ui.html")
	tmpl.Execute(w, nil)
}

// EditarUI devuelve solo el formulario de edición cargado con los datos
func (h *AdminHandler) EditarUI(w http.ResponseWriter, r *http.Request) {
	// Obtenemos el ID de la URL (ej: /admin/inquilinos/editar?id=5)
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))

	inquilino, err := h.Repo.ObtenerPorID(id)
	if err != nil {
		http.Error(w, "Inquilino no encontrado", http.StatusNotFound)
		return
	}

	tmpl, _ := template.ParseFiles("ui/templates/admin/inquilinos_ui.html")
	// Ejecutamos solo el bloque "formulario_editar"
	tmpl.ExecuteTemplate(w, "formulario_editar", inquilino)
}

// ActualizarInquilino procesa el formulario y devuelve la vista limpia
func (h *AdminHandler) ActualizarInquilino(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	id, _ := strconv.Atoi(r.FormValue("id"))
	tipoEntidad := r.FormValue("tipo_entidad")
	if tipoEntidad == "" {
		tipoEntidad = "GOBIERNO_LOCAL"
	}

	inquilino := models.Tenant{
		ID:          id,
		Nombre:      r.FormValue("nombre"),
		Ruc:         r.FormValue("ruc"),
		Activo:      r.FormValue("activo") == "on",
		TipoEntidad: tipoEntidad,
	}

	err := h.Repo.Actualizar(&inquilino)
	if err != nil {
		http.Error(w, "Error al actualizar el inquilino en la base de datos: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Después de actualizar, recargamos toda la UI de inquilinos
	// para que la tabla se actualice y el formulario vuelva a ser el de "Crear"
	h.VistaUI(w, r)
}
