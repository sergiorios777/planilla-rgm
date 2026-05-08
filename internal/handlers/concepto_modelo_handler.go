package handlers

import (
	"html/template"
	"log"
	"net/http"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
	"strconv"
)

type ConceptoModeloHandler struct {
	Repo               *repository.ConceptoModeloRepository
	PuestoRepo         *repository.PuestoRepository         // Lo necesitaremos para el select
	ConceptoTenantRepo *repository.ConceptoTenantRepository // Conceptos Maestros y clasificadores
}

// VistaUI carga la página principal del módulo
func (h *ConceptoModeloHandler) VistaUI(w http.ResponseWriter, r *http.Request) {
	// Obtenemos catálogos para los formularios
	regimenes, _ := h.PuestoRepo.ObtenerRegimenes()
	maestros, _ := h.ConceptoTenantRepo.ObtenerMaestros()
	clasificadores, _ := h.ConceptoTenantRepo.ObtenerClasificadores()

	data := map[string]interface{}{
		"Regimenes":      regimenes,
		"Conceptos":      maestros,
		"Clasificadores": clasificadores,
	}

	tmpl, _ := template.ParseFiles("ui/templates/admin/conceptos_modelo_ui.html")
	tmpl.Execute(w, data)
}

// Listar extrae la tabla filtrada por régimen para HTMX
func (h *ConceptoModeloHandler) Listar(w http.ResponseWriter, r *http.Request) {
	modelos, err := h.Repo.ObtenerTodos()
	if err != nil {
		http.Error(w, "Error al obtener modelos", 500)
		return
	}

	tmpl, _ := template.ParseFiles("ui/templates/admin/conceptos_modelo_ui.html")
	tmpl.ExecuteTemplate(w, "tabla_modelos", modelos)
}

// Crear procesa el formulario y devuelve la tabla actualizada
func (h *ConceptoModeloHandler) Crear(w http.ResponseWriter, r *http.Request) {
	log.Println("🚀 INICIO DE CREAR")
	r.ParseForm()

	regimenesSeleccionados := r.Form["regimenes_ids"]
	var ids []int
	for _, idStr := range regimenesSeleccionados {
		id, _ := strconv.Atoi(idStr)
		ids = append(ids, id)
	}

	var clasificadorID *int
	if idStr := r.FormValue("clasificador_id"); idStr != "" {
		cID, _ := strconv.Atoi(idStr)
		clasificadorID = &cID
	}

	nuevo := models.ConceptoModelo{
		ConceptoID:          0,
		NombrePersonalizado: r.FormValue("nombre_personalizado"),
		FrecuenciaMeses:     r.FormValue("frecuencia_meses"),
		ClasificadorID:      clasificadorID,
		EsExtraordinario:    r.FormValue("es_extraordinario") == "true",
		RequiereMonto:       r.FormValue("requiere_monto") == "true",
		RegimenesIDs:        ids,
	}
	nuevo.ConceptoID, _ = strconv.Atoi(r.FormValue("concepto_id"))

	err := h.Repo.Crear(&nuevo)
	if err != nil {
		log.Println("❌ ERROR AL CREAR EN BD:", err)
		http.Error(w, "Error interno al guardar en BD", http.StatusInternalServerError)
		return
	}

	// MAGIA HTMX: Si todo salió bien, le decimos al navegador que dispare el evento "cerrarModal"
	w.Header().Set("HX-Trigger", "cerrarModal")
	log.Println("✅ Creado con éxito, actualizando tabla...")
	h.Listar(w, r)
}

// EditarUI prepara y muestra el formulario para editar un concepto existente
func (h *ConceptoModeloHandler) EditarUI(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	concepto, _ := h.Repo.ObtenerPorID(id)

	regimenes, _ := h.PuestoRepo.ObtenerRegimenes()
	maestros, _ := h.ConceptoTenantRepo.ObtenerMaestros()
	clasificadores, _ := h.ConceptoTenantRepo.ObtenerClasificadores()

	// Creamos un mapa de regímenes marcados para facilitar el "checked" en el HTML
	marcados := make(map[int]bool)
	for _, rid := range concepto.RegimenesIDs {
		marcados[rid] = true
	}

	data := map[string]interface{}{
		"Concepto":           concepto,
		"Regimenes":          regimenes,
		"RegimenesMarcados":  marcados,
		"Conceptos":          maestros,
		"Clasificadores":     clasificadores,
		"ClasifSeleccionado": 0,
	}
	if concepto.ClasificadorID != nil {
		data["ClasifSeleccionado"] = *concepto.ClasificadorID
	}

	tmpl, _ := template.ParseFiles("ui/templates/admin/conceptos_modelo_ui.html")
	tmpl.ExecuteTemplate(w, "formulario_editar_modelo", data)
}

// Actualizar procesa la edición del concepto
func (h *ConceptoModeloHandler) Actualizar(w http.ResponseWriter, r *http.Request) {
	log.Println("🚀 INICIO DE ACTUALIZAR")
	r.ParseForm()

	conceptoIDReal, _ := strconv.Atoi(r.FormValue("id"))

	regimenesSeleccionados := r.Form["regimenes_ids"]
	var idsRegimen []int
	for _, idStr := range regimenesSeleccionados {
		regID, _ := strconv.Atoi(idStr)
		idsRegimen = append(idsRegimen, regID)
	}

	var clasificadorID *int
	if idStr := r.FormValue("clasificador_id"); idStr != "" {
		cID, _ := strconv.Atoi(idStr)
		clasificadorID = &cID
	}

	cMaestroID, _ := strconv.Atoi(r.FormValue("concepto_id"))
	editado := models.ConceptoModelo{
		ID:                  conceptoIDReal,
		ConceptoID:          cMaestroID,
		NombrePersonalizado: r.FormValue("nombre_personalizado"),
		FrecuenciaMeses:     r.FormValue("frecuencia_meses"),
		ClasificadorID:      clasificadorID,
		EsExtraordinario:    r.FormValue("es_extraordinario") == "true",
		RequiereMonto:       r.FormValue("requiere_monto") == "true",
		RegimenesIDs:        idsRegimen,
	}

	err := h.Repo.Actualizar(&editado)
	if err != nil {
		log.Println("❌ ERROR AL ACTUALIZAR EN BD:", err)
		http.Error(w, "Error interno al actualizar en BD", http.StatusInternalServerError)
		return
	}

	// Disparamos el cierre del modal
	w.Header().Set("HX-Trigger", "cerrarModal")
	log.Println("✅ Actualizado con éxito, refrescando tabla...")
	h.Listar(w, r)
}

// Eliminar quita el registro y actualiza la tabla
func (h *ConceptoModeloHandler) Eliminar(w http.ResponseWriter, r *http.Request) {
	log.Println("🚀 INICIO DE ELIMINAR")
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))

	// Ahora sí capturamos el error
	err := h.Repo.Eliminar(id)
	if err != nil {
		log.Println("❌ ERROR AL ELIMINAR EN BD:", err)
		http.Error(w, "Error al eliminar. Verifique que no esté en uso.", http.StatusInternalServerError)
		return
	}
	log.Println("✅ Eliminado con éxito.")
	h.Listar(w, r)
}
