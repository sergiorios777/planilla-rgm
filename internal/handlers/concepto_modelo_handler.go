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

type ConceptoModeloHandler struct {
	Repo               *repository.ConceptoModeloRepository
	PuestoRepo         *repository.PuestoRepository         // Lo necesitaremos para el select
	ConceptoTenantRepo *repository.ConceptoTenantRepository // Conceptos Maestros y clasificadores
	TenantRepo         *repository.TenantRepository         // Para la sincronización masiva
	Service            *services.ConceptoModeloService
	NotificacionRepo   *repository.NotificacionRepository
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

// Listar extrae la tabla filtrada y paginada para HTMX
func (h *ConceptoModeloHandler) Listar(w http.ResponseWriter, r *http.Request) {
	busqueda := r.URL.Query().Get("buscar")
	atributo := r.URL.Query().Get("atributo")
	regimenIDStr := r.URL.Query().Get("regimen_id")
	paginaStr := r.URL.Query().Get("pagina")
	limiteStr := r.URL.Query().Get("limite")

	limite, err := strconv.Atoi(limiteStr)
	if err != nil || limite <= 0 {
		limite = 10 // Por defecto mostramos 10
	}

	pagina, err := strconv.Atoi(paginaStr)
	if err != nil || pagina <= 0 {
		pagina = 1 // Por defecto empezamos en la página 1
	}

	regimenID, _ := strconv.Atoi(regimenIDStr)

	offset := (pagina - 1) * limite

	modelos, totalRegistros, err := h.Repo.ObtenerTodosPaginacion(busqueda, atributo, regimenID, limite, offset)
	if err != nil {
		http.Error(w, "Error al obtener modelos con paginación", http.StatusInternalServerError)
		return
	}

	totalPaginas := (totalRegistros + limite - 1) / limite
	if totalPaginas == 0 {
		totalPaginas = 1
	}

	datosVista := struct {
		Conceptos       []models.ConceptoModelo
		TotalPaginas    int
		PaginaActual    int
		PaginaAnterior  int
		PaginaSiguiente int
	}{
		Conceptos:       modelos,
		TotalPaginas:    totalPaginas,
		PaginaActual:    pagina,
		PaginaAnterior:  pagina - 1,
		PaginaSiguiente: pagina + 1,
	}

	tmpl, _ := template.ParseFiles("ui/templates/admin/conceptos_modelo_ui.html")
	tmpl.ExecuteTemplate(w, "tabla_modelos", datosVista)
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
		ConceptoID:               0,
		NombrePersonalizado:      r.FormValue("nombre_personalizado"),
		FrecuenciaMeses:          r.FormValue("frecuencia_meses"),
		ClasificadorID:           clasificadorID,
		EsExtraordinario:         r.FormValue("es_extraordinario") == "true",
		RequiereMonto:            r.FormValue("requiere_monto") == "true",
		EsPensionable:            r.FormValue("es_pensionable") == "true",
		EsRemunerativa:           r.FormValue("es_remunerativa") == "true",
		EsBaseCts:                r.FormValue("es_base_cts") == "true",
		EsBaseBeneficiosSociales: r.FormValue("es_base_beneficios_sociales") == "true",
		EsOcasional:              r.FormValue("es_ocasional") == "true",
		EsAfectoCargasSociales:   r.FormValue("es_afecto_cargas_sociales") == "true",
		RegimenesIDs:             ids,
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
		ID:                       conceptoIDReal,
		ConceptoID:               cMaestroID,
		NombrePersonalizado:      r.FormValue("nombre_personalizado"),
		FrecuenciaMeses:          r.FormValue("frecuencia_meses"),
		ClasificadorID:           clasificadorID,
		EsExtraordinario:         r.FormValue("es_extraordinario") == "true",
		RequiereMonto:            r.FormValue("requiere_monto") == "true",
		EsPensionable:            r.FormValue("es_pensionable") == "true",
		EsRemunerativa:           r.FormValue("es_remunerativa") == "true",
		EsBaseCts:                r.FormValue("es_base_cts") == "true",
		EsBaseBeneficiosSociales: r.FormValue("es_base_beneficios_sociales") == "true",
		EsOcasional:              r.FormValue("es_ocasional") == "true",
		EsAfectoCargasSociales:   r.FormValue("es_afecto_cargas_sociales") == "true",
		RegimenesIDs:             idsRegimen,
	}

	err := h.Repo.Actualizar(&editado)
	if err != nil {
		log.Println("❌ ERROR AL ACTUALIZAR EN BD:", err)
		http.Error(w, "Error interno al actualizar en BD", http.StatusInternalServerError)
		return
	}

	// Disparamos el cierre del modal
	w.Header().Set("HX-Trigger", "cerrarModal")
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

// Sincronizar realiza la propagación masiva de conceptos modelo a todos los tenants activos
func (h *ConceptoModeloHandler) Sincronizar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	r.ParseForm()
	modo := r.FormValue("modo")
	fechaInicio := r.FormValue("fecha_inicio")
	fechaFin := r.FormValue("fecha_fin")

	if modo == "FECHAS" && (fechaInicio == "" || fechaFin == "") {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
			<div id="alerta-sincronizacion" hx-swap-oob="true">
				<article style="background-color: #ffcdd2; color: #b71c1c; padding: 1rem; margin-bottom: 1rem; border-radius: 5px; border: 1px solid #ef9a9a;">
					❌ Error: Debe especificar ambas fechas (Inicio y Fin) para la sincronización por fechas.
				</article>
			</div>
		`))
		return
	}

	tenants, err := h.TenantRepo.ObtenerTodos("")
	if err != nil {
		log.Println("❌ Error al obtener inquilinos:", err)
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
			<div id="alerta-sincronizacion" hx-swap-oob="true">
				<article style="background-color: #ffcdd2; color: #b71c1c; padding: 1rem; margin-bottom: 1rem; border-radius: 5px; border: 1px solid #ef9a9a;">
					❌ Error al obtener la lista de municipalidades desde el servidor.
				</article>
			</div>
		`))
		return
	}

	var exitos, fallas int
	var erroresDetalle []string

	for _, tenant := range tenants {
		if !tenant.Activo {
			continue
		}

		err := h.ConceptoTenantRepo.SincronizarDesdeModeloAvanzado(tenant.ID, modo, fechaInicio, fechaFin)
		if err != nil {
			fallas++
			erroresDetalle = append(erroresDetalle, fmt.Sprintf("<li><strong>%s (RUC: %s)</strong>: %s</li>", tenant.Nombre, tenant.Ruc, err.Error()))
			log.Printf("❌ Error sincronizando tenant %d (%s): %v\n", tenant.ID, tenant.Nombre, err)
		} else {
			if errSeed := h.Service.SembrarBaseRegimenTenant(tenant.ID); errSeed != nil {
				log.Printf("⚠️ Advertencia: No se pudo sembrar base cálculo para tenant %d: %v\n", tenant.ID, errSeed)
			}
			exitos++
		}

	}

	w.Header().Set("HX-Trigger", "cerrarModal")
	w.Header().Set("Content-Type", "text/html")

	var htmlResponse string
	if fallas > 0 {
		htmlResponse = fmt.Sprintf(`
			<div id="alerta-sincronizacion" hx-swap-oob="true">
				<article style="background-color: #fff3e0; color: #e65100; padding: 1rem; margin-bottom: 1rem; border-radius: 5px; border: 1px solid #ffe0b2;">
					<strong>⚠️ Sincronización masiva finalizada con advertencias:</strong>
					<p style="margin: 0.5rem 0;">Exitosas: %d | Fallidas: %d</p>
					<ul style="margin-top: 0.5rem; margin-bottom: 0; padding-left: 1.5rem;">
						%s
					</ul>
				</article>
			</div>
		`, exitos, fallas, strings.Join(erroresDetalle, ""))
	} else {
		htmlResponse = fmt.Sprintf(`
			<div id="alerta-sincronizacion" hx-swap-oob="true">
				<article style="background-color: #e8f5e9; color: #2e7d32; padding: 1rem; margin-bottom: 1rem; border-radius: 5px; border: 1px solid #c8e6c9;">
					✅ Sincronización masiva finalizada con éxito. Se procesaron %d municipalidades activas satisfactoriamente.
				</article>
			</div>
		`, exitos)
	}

	w.Write([]byte(htmlResponse))
	h.Listar(w, r)
}

// ImportarCSV recibe el archivo subido por el Super Admin y realiza la carga masiva transaccional
func (h *ConceptoModeloHandler) ImportarCSV(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	// Limitar tamaño del multipart form a 10MB
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		log.Println("❌ Error al parsear multipart form:", err)
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
			<article style="background-color: #ffcdd2; color: #b71c1c; padding: 1rem; margin-bottom: 1rem; border-radius: 5px; border: 1px solid #ef9a9a;">
				❌ Error: El archivo subido excede el límite permitido de tamaño o no es válido.
			</article>
		`))
		return
	}

	file, _, err := r.FormFile("archivo_csv")
	if err != nil {
		log.Println("❌ Error al recuperar archivo_csv:", err)
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
			<article style="background-color: #ffcdd2; color: #b71c1c; padding: 1rem; margin-bottom: 1rem; border-radius: 5px; border: 1px solid #ef9a9a;">
				❌ Error: No se ha proporcionado un archivo válido en el campo "archivo_csv".
			</article>
		`))
		return
	}
	defer file.Close()

	exitosos, err := h.Service.ImportarDesdeCSV(file)
	if err != nil {
		log.Printf("❌ Error al importar conceptos modelo desde CSV: %v\n", err)
		w.Header().Set("Content-Type", "text/html")
		w.Write(fmt.Appendf(nil, `
			<article style="background-color: #ffcdd2; color: #b71c1c; padding: 1rem; margin-bottom: 1rem; border-radius: 5px; border: 1px solid #ef9a9a;">
				❌ Error de Validación/Importación: %s
			</article>
		`, err.Error()))
		return
	}

	log.Printf("✅ Carga masiva completada: %d conceptos modelo importados.\n", exitosos)

	// Registrar la notificación en la base de datos
	tID, uID := obtenerUsuarioTenantID(r)
	notif := models.Notificacion{
		TenantID:  tID,
		UsuarioID: uID,
		Titulo:    "Carga Masiva de Conceptos Modelo",
		Mensaje:   fmt.Sprintf("Se han importado/actualizado correctamente %d conceptos modelo.", exitosos),
		Tipo:      "PROCESO_EXITOSO",
		Leido:     false,
	}
	if errNotif := h.NotificacionRepo.Crear(&notif); errNotif != nil {
		log.Printf("⚠️ No se pudo registrar la notificación en la BD: %v\n", errNotif)
	}

	// Responder con HTMX Headers para cerrar el modal y refrescar la grilla principal
	w.Header().Set("HX-Trigger", "cerrarModalImportar, recargarTablaModelos")
	w.Header().Set("Content-Type", "text/html")
	w.Write(fmt.Appendf(nil, `
		<article style="background-color: #e8f5e9; color: #2e7d32; padding: 1rem; margin-bottom: 1rem; border-radius: 5px; border: 1px solid #c8e6c9;">
			✅ Carga masiva exitosa: se importaron/actualizaron %d conceptos modelo correctamente.
		</article>
	`, exitosos))
}

// PlantillaCSV sirve un archivo CSV de ejemplo con la estructura requerida
func (h *ConceptoModeloHandler) PlantillaCSV(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=plantilla_conceptos_modelo.csv")
	
	// Escribir cabecera y una fila de ejemplo (16 columnas)
	cabecera := "codigo_sunat,nombre_personalizado_unico_,frecuencia_meses,clasificador_codigo,es_extraordinario,requiere_monto,es_pensionable,es_remunerativa,es_base_cts,es_base_beneficios_sociales,es_ocasional,es_afecto_cargas_sociales,dl_276,dl_728,dl_1057,ley_30057\n"
	ejemplo := "0121,Remuneración Principal Básica,\"1,2,3,4,5,6,7,8,9,10,11,12\",2.1.1.1.1.1,0,0,1,1,1,1,0,1,1,1,0,1\n"
	
	w.Write([]byte(cabecera + ejemplo))
}

