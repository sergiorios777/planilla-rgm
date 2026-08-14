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

	"github.com/xuri/excelize/v2"
)

type PuestoHandler struct {
	Repo            *repository.PuestoRepository
	MetaRepo        *repository.MetaRepository
	FuenteRubroRepo *repository.FuenteRubroRepository
	OrganigramaRepo *repository.OrganigramaRepository
}

func (h *PuestoHandler) VistaUI(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)

	// Preparamos listas para los combos
	metas, _ := h.MetaRepo.ObtenerTodos(tenantID)
	fuentes, _ := h.FuenteRubroRepo.ObtenerPorAnio(2026, "")
	regimenes, _ := h.Repo.ObtenerRegimenes()
	unidades, _ := h.OrganigramaRepo.ObtenerUnidadesDelOrganigramaActivo(tenantID)

	datos := map[string]interface{}{
		"Metas":           metas,
		"Fuentes":         fuentes,
		"Regimenes":       regimenes,
		"Unidades":        unidades,
		"CurrentUnidadID": 0,
	}

	tmpl, err := template.ParseFiles("ui/templates/tenant/puestos_ui.html")

	if err != nil {
		log.Println("❌ Error CRÍTICO al cargar la plantilla de puestos:", err)
		http.Error(w, "Error interno del servidor al cargar la interfaz", 500)
		return
	}

	err = tmpl.Execute(w, datos)
	if err != nil {
		log.Println("❌ Error al renderizar la plantilla:", err)
	}
}

func (h *PuestoHandler) Listar(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	busqueda := r.URL.Query().Get("buscar")
	metaIDStr := r.URL.Query().Get("meta_id")
	regimenIDStr := r.URL.Query().Get("regimen_id")
	unidadIDStr := r.URL.Query().Get("unidad_organica_id")
	estado := r.URL.Query().Get("estado")
	limiteStr := r.URL.Query().Get("limite")
	paginaStr := r.URL.Query().Get("pagina")

	metaID, err := strconv.Atoi(metaIDStr)
	regimenID, err := strconv.Atoi(regimenIDStr)
	unidadID, _ := strconv.Atoi(unidadIDStr)

	limite, err := strconv.Atoi(limiteStr)
	if err != nil || limite <= 0 {
		limite = 10 // Por defecto mostramos 10
	}

	pagina, err := strconv.Atoi(paginaStr)
	if err != nil || pagina <= 0 {
		pagina = 1 // Por defecto empezamos en la página 1
	}

	offset := (pagina - 1) * limite

	puestos, totalRegistros, err := h.Repo.ObtenerTodosPaginacion(tenantID, metaID, regimenID, unidadID, busqueda, estado, limite, offset)
	if err != nil {
		http.Error(w, "Error al obtener las metas", http.StatusInternalServerError)
		return
	}

	totalPaginas := (totalRegistros + limite - 1) / limite
	if totalPaginas == 0 {
		totalPaginas = 1
	}

	paginacion := models.CalcularPaginacion(
		pagina,
		totalPaginas,
		totalRegistros,
		"/tenant/puestos/lista",
		"#lista-puestos",
		"#form-filtros-puestos",
	)

	datosVista := struct {
		Puestos    []models.Puesto
		Paginacion models.PaginacionDTO
	}{
		Puestos:    puestos,
		Paginacion: paginacion,
	}

	tmpl, _ := template.ParseFiles("ui/templates/tenant/puestos_ui.html", "ui/templates/components/paginacion.html")
	tmpl.ExecuteTemplate(w, "tabla_puestos", datosVista)
}

func (h *PuestoHandler) Crear(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	metaID, _ := strconv.Atoi(r.FormValue("meta_id"))
	fuenteID, _ := strconv.Atoi(r.FormValue("fuente_rubro_id"))
	regimenID, _ := strconv.Atoi(r.FormValue("regimen_id"))
	sueldo, _ := strconv.ParseFloat(r.FormValue("sueldo_presupuestado"), 64)

	var unidadOrganicaID *int
	idStr := r.FormValue("unidad_organica_id")
	if idStr != "" && idStr != "0" {
		idVal, err := strconv.Atoi(idStr)
		if err == nil {
			unidadOrganicaID = &idVal
		}
	}

	var codigoAirhsp *string
	airhspStr := r.FormValue("codigo_airhsp")
	if airhspStr != "" {
		codigoAirhsp = &airhspStr
	}

	nuevoPuesto := models.Puesto{
		TenantID:            obtenerTenantID(r),
		MetaID:              metaID,
		FuenteRubroID:       fuenteID,
		RegimenID:           regimenID,
		Nombre:              r.FormValue("nombre"),
		SueldoPresupuestado: sueldo,
		Activo:              r.FormValue("activo") == "on",
		EsDietario:          r.FormValue("es_dietario") == "on",
		UnidadOrganicaID:    unidadOrganicaID,
		CodigoAirhsp:        codigoAirhsp,
	}

	servicioPuesto := services.PuestoService{Repo: h.Repo}
	err := servicioPuesto.CrearPuestoConPlantilla(&nuevoPuesto)
	if err != nil {
		log.Println("Error creando puesto con plantilla:", err)
		http.Error(w, "Error al crear el puesto: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", "refrescarPuestos")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("✅ Plaza creada correctamente."))
}

// Editar prepara los datos del puesto y las listas para el formulario de edición
func (h *PuestoHandler) Editar(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	tenantID := obtenerTenantID(r)

	// 1. Buscamos el puesto actual
	puesto, _ := h.Repo.ObtenerPorID(id, tenantID)

	// 2. Necesitamos las listas para los combos (Selects)
	metas, _ := h.MetaRepo.ObtenerTodos(tenantID)
	fuentes, _ := h.FuenteRubroRepo.ObtenerPorAnio(2026, "")
	regimenes, _ := h.Repo.ObtenerRegimenes()

	datos := map[string]interface{}{
		"Puesto":    puesto,
		"Metas":     metas,
		"Fuentes":   fuentes,
		"Regimenes": regimenes,
	}

	// 💡 ENVIAMOS SOLO EL FRAGMENTO: "formulario_editar"
	tmpl, _ := template.ParseFiles("ui/templates/tenant/puestos_ui.html")
	tmpl.ExecuteTemplate(w, "formulario_editar", datos)
}

// EditarUI prepara los datos del puesto y las listas para el modal de edición
func (h *PuestoHandler) EditarUI(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	tenantID := obtenerTenantID(r)
	currentUnidadID := 0

	puesto, err := h.Repo.ObtenerPorID(id, tenantID)
	if err != nil {
		log.Println("Error al obtener puesto por ID:", err)
		http.Error(w, "No se pudo obtener el puesto", http.StatusInternalServerError)
		return
	}

	if puesto.UnidadOrganicaID != nil {
		currentUnidadID = *puesto.UnidadOrganicaID
	}

	metas, _ := h.MetaRepo.ObtenerTodos(tenantID)
	fuentes, _ := h.FuenteRubroRepo.ObtenerPorAnio(2026, "")
	regimenes, _ := h.Repo.ObtenerRegimenes()
	unidades, _ := h.OrganigramaRepo.ObtenerUnidadesDelOrganigramaActivo(tenantID)

	datos := map[string]interface{}{
		"Puesto":          puesto,
		"Metas":           metas,
		"Fuentes":         fuentes,
		"Regimenes":       regimenes,
		"Unidades":        unidades,
		"CurrentUnidadID": currentUnidadID,
	}

	tmpl, err := template.ParseFiles("ui/templates/tenant/puestos_ui.html")
	if err != nil {
		log.Println("Error al leer puestos_ui.html:", err)
		http.Error(w, "Error de plantilla", http.StatusInternalServerError)
		return
	}

	err = tmpl.ExecuteTemplate(w, "formulario_editar", datos)
	if err != nil {
		log.Println("Error al ejecutar fragmento formulario_editar:", err)
		http.Error(w, "Error al inyectar fragmento", http.StatusInternalServerError)
	}
}

// Actualizar procesa los cambios y refresca la lista
func (h *PuestoHandler) Actualizar(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id, _ := strconv.Atoi(r.FormValue("id"))

	metaID, _ := strconv.Atoi(r.FormValue("meta_id"))
	fuenteID, _ := strconv.Atoi(r.FormValue("fuente_rubro_id"))
	regimenID, _ := strconv.Atoi(r.FormValue("regimen_id"))
	sueldo, _ := strconv.ParseFloat(r.FormValue("sueldo_presupuestado"), 64)

	var unidadOrganicaID *int
	idStr := r.FormValue("unidad_organica_id")
	if idStr != "" && idStr != "0" {
		idVal, err := strconv.Atoi(idStr)
		if err == nil {
			unidadOrganicaID = &idVal
		}
	}

	var codigoAirhsp *string
	airhspStr := r.FormValue("codigo_airhsp")
	if airhspStr != "" {
		codigoAirhsp = &airhspStr
	}

	puestoActualizado := models.Puesto{
		ID:                  id,
		TenantID:            obtenerTenantID(r),
		MetaID:              metaID,
		FuenteRubroID:       fuenteID,
		RegimenID:           regimenID,
		Nombre:              r.FormValue("nombre"),
		SueldoPresupuestado: sueldo,
		Activo:              r.FormValue("activo") == "on",
		EsDietario:          r.FormValue("es_dietario") == "on",
		UnidadOrganicaID:    unidadOrganicaID,
		CodigoAirhsp:        codigoAirhsp,
	}

	err := h.Repo.Actualizar(&puestoActualizado)
	if err != nil {
		log.Println("Error al actualizar puesto:", err)
		http.Error(w, "Error al actualizar el puesto: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", "refrescarPuestos")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("✅ Plaza actualizada correctamente."))
}

// FormularioCrearUI devuelve el formulario limpio
func (h *PuestoHandler) FormularioCrearUI(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)

	metas, _ := h.MetaRepo.ObtenerTodos(tenantID)
	fuentes, _ := h.FuenteRubroRepo.ObtenerPorAnio(2026, "")
	regimenes, _ := h.Repo.ObtenerRegimenes()

	datos := map[string]interface{}{
		"Metas":     metas,
		"Fuentes":   fuentes,
		"Regimenes": regimenes,
	}

	tmpl, _ := template.ParseFiles("ui/templates/tenant/puestos_ui.html")
	tmpl.ExecuteTemplate(w, "formulario_crear", datos)
}

// AsignarConceptosUI carga el modal con la estructura de conceptos del puesto
func (h *PuestoHandler) AsignarConceptosUI(w http.ResponseWriter, r *http.Request) {
	puestoID, _ := strconv.Atoi(r.URL.Query().Get("id"))
	tenantID := obtenerTenantID(r) // Tu función para obtener el ID de la municipalidad
	// Obtenemos la lista combinada (Conceptos del Tenant + lo que tiene el Puesto)
	asignaciones, err := h.Repo.ObtenerConceptosParaAsignacion(puestoID, tenantID)
	if err != nil {
		log.Println("Error al obtener conceptos para asignación:", err)
		http.Error(w, "Error interno", 500)
		return
	}

	data := map[string]interface{}{
		"PuestoID":     puestoID,
		"Asignaciones": asignaciones,
	}

	// CAPTURAMOS EL ERROR DE LA PLANTILLA
	tmpl, err := template.ParseFiles("ui/templates/tenant/puestos_ui.html")
	if err != nil {
		log.Println("❌ Error al leer puestos_ui.html:", err)
		http.Error(w, "Error de plantilla", 500)
		return
	}

	err = tmpl.ExecuteTemplate(w, "formulario_asignar_conceptos", data)
	if err != nil {
		log.Println("❌ Error al ejecutar el fragmento HTMX:", err)
		http.Error(w, "Error al inyectar fragmento", 500)
	}
}

// GuardarAsignacion procesa el formulario enviado por HTMX
func (h *PuestoHandler) GuardarAsignacion(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	puestoID, _ := strconv.Atoi(r.FormValue("puesto_id"))

	// Leemos qué conceptos fueron marcados (switches encendidos)
	idsSeleccionados := r.Form["concepto_tenant_ids"]

	var listaParaGuardar []models.ConceptoAsignacion
	for _, idStr := range idsSeleccionados {
		id, _ := strconv.Atoi(idStr)

		// Leemos el monto específico para este ID (ej: monto_45)
		montoStr := r.FormValue("monto_" + idStr)
		monto, _ := strconv.ParseFloat(montoStr, 64)

		listaParaGuardar = append(listaParaGuardar, models.ConceptoAsignacion{
			ConceptoTenantID: id,
			Monto:            monto,
			Asignado:         true,
		})
	}

	err := h.Repo.GuardarAsignacionConceptos(puestoID, listaParaGuardar)
	if err != nil {
		log.Println("Error al guardar asignación:", err)
		http.Error(w, "No se pudo guardar la estructura de pago", 500)
		return
	}

	// Si todo sale bien, enviamos la señal de éxito para cerrar el modal
	w.Header().Set("HX-Trigger", "cerrarModalAsignacion")
	w.Write([]byte("✅ Estructura de pago actualizada correctamente."))
}

// DescargarPlantilla genera y envía un archivo Excel base para la importación de puestos
func (h *PuestoHandler) DescargarPlantilla(w http.ResponseWriter, r *http.Request) {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Puestos"
	f.SetSheetName("Sheet1", sheet)

	// Cabeceras
	cabeceras := []string{
		"NOMBRE", "SUELDO_PRESUPUESTADO", "REGIMEN_LABORAL", "META_PRESUPUESTAL",
		"CODIGO_FUENTE_RUBRO", "CODIGO_MEF_UNIDAD", "CODIGO_AIRHSP", "ES_DIETARIO", "ACTIVO",
	}
	for i, cabecera := range cabeceras {
		col := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue(sheet, col, cabecera)
	}

	// Datos de ejemplo
	ejemplos := [][]interface{}{
		{"Especialista en Logística II", 3500.00, "276", "0015", "1.00", "300003", "000456", "NO", "SI"},
		{"Especialista en Sistemas I", 4200.00, "1057", "0016", "2.09", "", "", "NO", "SI"},
		{"Regidor", 1500.00, "30057", "", "", "", "", "SI", "SI"},
	}
	for rIdx, fila := range ejemplos {
		for cIdx, valor := range fila {
			col := fmt.Sprintf("%c%d", 'A'+cIdx, rIdx+2)
			f.SetCellValue(sheet, col, valor)
		}
	}

	// Hoja de instrucciones
	instruccionesSheet := "Instrucciones"
	f.NewSheet(instruccionesSheet)
	f.SetCellValue(instruccionesSheet, "A1", "INSTRUCCIONES DE LLENADO PARA PUESTOS")
	f.SetCellValue(instruccionesSheet, "A3", "NOMBRE: Obligatorio. Nombre de la plaza (ej. Especialista en Logística II). Max 150 caracteres.")
	f.SetCellValue(instruccionesSheet, "A4", "SUELDO_PRESUPUESTADO: Obligatorio. Monto mensual presupuestado (ej. 3500.00).")
	f.SetCellValue(instruccionesSheet, "A5", "REGIMEN_LABORAL: Obligatorio. Código del régimen laboral. Debe ser uno de: 276, 728, 1057, 30057.")
	f.SetCellValue(instruccionesSheet, "A6", "META_PRESUPUESTAL: Opcional. Código de la meta presupuestal del año en curso (ej. 0015). Si no existe, se informará como advertencia y se dejará en blanco.")
	f.SetCellValue(instruccionesSheet, "A7", "CODIGO_FUENTE_RUBRO: Opcional. Código del rubro/fuente del año en curso (ej. 1.00, 2.09, 5.07). Si no existe, se informará como advertencia y se dejará en blanco.")
	f.SetCellValue(instruccionesSheet, "A8", "CODIGO_MEF_UNIDAD: Opcional. Código MEF de la unidad orgánica correspondiente (ej. 300003). Si no existe, se informará como advertencia y se dejará en blanco.")
	f.SetCellValue(instruccionesSheet, "A9", "CODIGO_AIRHSP: Opcional. Código del aplicativo informático AIRHSP. Max 50 caracteres.")
	f.SetCellValue(instruccionesSheet, "A10", "ES_DIETARIO: Opcional. SI o NO. Indica si es dieta para regidores. Por defecto es NO.")
	f.SetCellValue(instruccionesSheet, "A11", "ACTIVO: Opcional. SI o NO. Indica si la plaza está vigente. Por defecto es SI.")

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=plantilla_puestos.xlsx")

	if err := f.Write(w); err != nil {
		log.Printf("[ERROR] Al generar plantilla Excel de puestos: %v", err)
	}
}

// ImportarExcel procesa la subida de un archivo Excel, lo valida de manera atómica con un pool de workers y lo importa
func (h *PuestoHandler) ImportarExcel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	tenantID := obtenerTenantID(r)

	// 1. Leer el formulario multipart (máximo 10 MB)
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<p style="color:red; margin:0;">⚠️ Error al procesar el formulario de subida.</p>`))
		return
	}

	file, _, err := r.FormFile("archivo_excel")
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<p style="color:red; margin:0;">⚠️ Error al leer el archivo seleccionado.</p>`))
		return
	}
	defer file.Close()

	// 2. Abrir el libro Excel
	f, err := excelize.OpenReader(file)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<p style="color:red; margin:0;">⚠️ El archivo subido no es un formato de Excel válido.</p>`))
		return
	}
	defer f.Close()

	hoja := f.GetSheetName(0)
	filas, err := f.GetRows(hoja)
	if err != nil || len(filas) == 0 {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<p style="color:red; margin:0;">⚠️ No se pudo leer el contenido de la primera hoja del Excel.</p>`))
		return
	}

	// 3. Cargar catálogos en memoria para resolución de claves
	regList, err := h.Repo.ObtenerRegimenes()
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		w.Write(fmt.Appendf(nil, `<p style="color:red; margin:0;">⚠️ Error al cargar catálogo de regímenes: %v</p>`, err))
		return
	}
	mapaRegimenes := make(map[string]int)
	for _, rg := range regList {
		mapaRegimenes[strings.TrimSpace(rg.Codigo)] = rg.ID
	}

	metaList, err := h.MetaRepo.ObtenerTodos(tenantID)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		w.Write(fmt.Appendf(nil, `<p style="color:red; margin:0;">⚠️ Error al cargar catálogo de metas: %v</p>`, err))
		return
	}
	mapaMetas := make(map[string]int)
	for _, m := range metaList {
		mapaMetas[strings.TrimSpace(m.Codigo)] = m.ID
	}

	fuenteList, err := h.FuenteRubroRepo.ObtenerPorAnio(2026, "")
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		w.Write(fmt.Appendf(nil, `<p style="color:red; margin:0;">⚠️ Error al cargar catálogo de fuentes/rubros: %v</p>`, err))
		return
	}
	mapaFuentes := make(map[string]int)
	for _, fr := range fuenteList {
		mapaFuentes[strings.TrimSpace(fr.CodigoFuenteRubro)] = fr.ID
	}

	unidadList, err := h.OrganigramaRepo.ObtenerUnidadesDelOrganigramaActivo(tenantID)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		w.Write(fmt.Appendf(nil, `<p style="color:red; margin:0;">⚠️ Error al cargar catálogo de unidades orgánicas: %v</p>`, err))
		return
	}
	mapaUnidades := make(map[string]int)
	for _, u := range unidadList {
		if u.CodigoMef != "" {
			mapaUnidades[strings.TrimSpace(u.CodigoMef)] = u.ID
		}
	}

	// Estructuras para la concurrencia
	type RowJob struct {
		Index int
		Row   []string
	}
	type RowResult struct {
		Index    int
		Puesto   models.Puesto
		Warnings []string
		Error    error
	}

	numFilas := len(filas)
	jobs := make(chan RowJob, numFilas)
	results := make(chan RowResult, numFilas)

	// Worker Pool: 4 workers
	numWorkers := 4
	if numFilas-1 < numWorkers {
		numWorkers = numFilas - 1
	}
	if numWorkers <= 0 {
		numWorkers = 1
	}

	for wIdx := 0; wIdx < numWorkers; wIdx++ {
		go func() {
			for job := range jobs {
				fila := job.Row
				numFila := job.Index + 1

				// Ignorar fila vacía
				filaVacia := true
				for _, celda := range fila {
					if strings.TrimSpace(celda) != "" {
						filaVacia = false
						break
					}
				}
				if filaVacia {
					results <- RowResult{Index: job.Index, Error: nil}
					continue
				}

				// Validar columnas mínimas (al menos Nombre, Sueldo y Régimen)
				if len(fila) < 3 {
					results <- RowResult{Index: job.Index, Error: fmt.Errorf("Fila %d: Columnas incompletas (mínimo se requieren 3 columnas: Nombre, Sueldo Presupuestado y Régimen Laboral)", numFila)}
					continue
				}

				nombre := strings.TrimSpace(fila[0])
				sueldoRaw := strings.TrimSpace(fila[1])
				regimenRaw := strings.TrimSpace(fila[2])

				// Val: Nombre
				if nombre == "" {
					results <- RowResult{Index: job.Index, Error: fmt.Errorf("Fila %d: El Nombre del puesto es obligatorio", numFila)}
					continue
				}
				if len(nombre) > 150 {
					results <- RowResult{Index: job.Index, Error: fmt.Errorf("Fila %d: El nombre del puesto '%s' supera los 150 caracteres", numFila, nombre)}
					continue
				}

				// Val: Sueldo
				sueldo, errSueldo := strconv.ParseFloat(sueldoRaw, 64)
				if errSueldo != nil || sueldo < 0 {
					results <- RowResult{Index: job.Index, Error: fmt.Errorf("Fila %d: Sueldo Presupuestado '%s' inválido. Debe ser un número positivo", numFila, sueldoRaw)}
					continue
				}

				// Val: Régimen Laboral (Obligatorio)
				regID, existeReg := mapaRegimenes[regimenRaw]
				if !existeReg {
					results <- RowResult{Index: job.Index, Error: fmt.Errorf("Fila %d: El Código de Régimen Laboral '%s' es obligatorio y no existe en el sistema", numFila, regimenRaw)}
					continue
				}

				var warnings []string

				// Val: Meta Presupuestal (Opcional)
				var metaID int
				if len(fila) >= 4 {
					metaRaw := strings.TrimSpace(fila[3])
					if metaRaw != "" {
						mID, existeMeta := mapaMetas[metaRaw]
						if existeMeta {
							metaID = mID
						} else {
							warnings = append(warnings, fmt.Sprintf("Fila %d: El código de Meta '%s' no existe para el año en curso. Se importará sin asignar meta.", numFila, metaRaw))
						}
					}
				}

				// Val: Fuente/Rubro (Opcional)
				var fuenteID int
				if len(fila) >= 5 {
					fuenteRaw := strings.TrimSpace(fila[4])
					if fuenteRaw != "" {
						fID, existeFuente := mapaFuentes[fuenteRaw]
						if existeFuente {
							fuenteID = fID
						} else {
							warnings = append(warnings, fmt.Sprintf("Fila %d: El código de Fuente y Rubro '%s' no existe en el sistema. Se importará sin asignar fuente/rubro.", numFila, fuenteRaw))
						}
					}
				}

				// Val: Unidad Orgánica (Opcional)
				var unidadID *int
				if len(fila) >= 6 {
					unidadRaw := strings.TrimSpace(fila[5])
					if unidadRaw != "" {
						uID, existeUnidad := mapaUnidades[unidadRaw]
						if existeUnidad {
							unidadID = &uID
						} else {
							warnings = append(warnings, fmt.Sprintf("Fila %d: El código MEF de Unidad Orgánica '%s' no existe en el organigrama activo. Se importará sin asignar unidad.", numFila, unidadRaw))
						}
					}
				}

				// Val: Código AIRHSP (Opcional)
				var airhsp *string
				if len(fila) >= 7 {
					airRaw := strings.TrimSpace(fila[6])
					if airRaw != "" {
						if len(airRaw) > 50 {
							results <- RowResult{Index: job.Index, Error: fmt.Errorf("Fila %d: El Código AIRHSP '%s' supera los 50 caracteres", numFila, airRaw)}
							continue
						}
						airhsp = &airRaw
					}
				}

				// Val: Es Dietario (Opcional)
				esDietario := false
				if len(fila) >= 8 {
					dietRaw := strings.ToUpper(strings.TrimSpace(fila[7]))
					if dietRaw != "" {
						esDietario = (dietRaw == "SI" || dietRaw == "TRUE" || dietRaw == "1" || dietRaw == "DIETARIO")
					}
				}

				// Val: Activo (Opcional)
				activo := true
				if len(fila) >= 9 {
					activoRaw := strings.ToUpper(strings.TrimSpace(fila[8]))
					if activoRaw != "" {
						activo = (activoRaw == "SI" || activoRaw == "TRUE" || activoRaw == "1" || activoRaw == "ACTIVO")
					}
				}

				p := models.Puesto{
					TenantID:            tenantID,
					MetaID:              metaID,
					FuenteRubroID:       fuenteID,
					RegimenID:           regID,
					Nombre:              nombre,
					SueldoPresupuestado: sueldo,
					Activo:              activo,
					EsDietario:          esDietario,
					UnidadOrganicaID:    unidadID,
					CodigoAirhsp:        airhsp,
				}

				results <- RowResult{Index: job.Index, Puesto: p, Warnings: warnings, Error: nil}
			}
		}()
	}

	// Enviar a procesar (saltando fila 0 de cabecera)
	for i := 1; i < len(filas); i++ {
		jobs <- RowJob{Index: i, Row: filas[i]}
	}
	close(jobs)

	// Recopilar resultados
	var puestos []models.Puesto
	var warnings []string
	var validationError error

	for i := 1; i < len(filas); i++ {
		res := <-results
		if res.Error != nil {
			validationError = res.Error
			continue
		}
		if res.Puesto.Nombre == "" {
			continue
		}
		puestos = append(puestos, res.Puesto)
		if len(res.Warnings) > 0 {
			warnings = append(warnings, res.Warnings...)
		}
	}

	if validationError != nil {
		w.Header().Set("Content-Type", "text/html")
		w.Write(fmt.Appendf(nil, `<p style="color:red; margin:0;">⚠️ %v. Se canceló toda la importación.</p>`, validationError))
		return
	}

	if len(puestos) == 0 {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<p style="color:red; margin:0;">⚠️ El archivo Excel no contiene filas de puestos válidas para importar.</p>`))
		return
	}

	// 4. Inserción atómica en base de datos
	err = h.Repo.ImportarPuestos(tenantID, puestos)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		w.Write(fmt.Appendf(nil, `<p style="color:red; margin:0;">⚠️ Error de Base de Datos: %v. Se canceló toda la importación.</p>`, err))
		return
	}

	// 5. Devolver HTML con el resultado y advertencias
	w.Header().Set("Content-Type", "text/html")

	warningsHTML := ""
	if len(warnings) > 0 {
		warningsList := ""
		for _, wr := range warnings {
			warningsList += fmt.Sprintf("<li><small>%s</small></li>", wr)
		}
		warningsHTML = fmt.Sprintf(`
			<details style="margin-top: 1rem; border: 1px solid #ffe0b2; background: #fff8e1; border-radius: 4px; padding: 0.5rem 1rem;">
				<summary style="color: #e65100; font-weight: 500; cursor: pointer;">⚠️ Observaciones y Advertencias (%d asignaciones omitidas)</summary>
				<ul style="margin: 0.5rem 0 0 1.2rem; padding: 0; color: #5d4037; text-align: left;">
					%s
				</ul>
			</details>
		`, len(warnings), warningsList)
	}

	w.Write(fmt.Appendf(nil, `
		<article style="background-color: #e8f5e9; color: #1b5e20; padding: 1rem; border-radius: 5px; margin: 0; text-align: center;">
			✅ Importación Exitosa.<br>
			Se registraron <strong>%d</strong> puestos de trabajo (plazas) correctamente y la transacción fue confirmada.
			%s
		</article>
	`, len(puestos), warningsHTML))
}
