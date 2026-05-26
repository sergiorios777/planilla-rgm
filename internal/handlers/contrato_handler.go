package handlers

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"planilla-rgm/internal/config"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
	"planilla-rgm/internal/services"
	"sort"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

type ContratoHandler struct {
	Repo           *repository.ContratoRepository
	TrabajadorRepo *repository.TrabajadorRepository // Lo necesitamos para el select
	PuestoRepo     *repository.PuestoRepository
}

func (h *ContratoHandler) VistaUI(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)

	trabajadores, _ := h.TrabajadorRepo.ObtenerTodos(tenantID)
	puestos, _ := h.PuestoRepo.ObtenerVacantes(tenantID)
	regimenes, _ := h.PuestoRepo.ObtenerRegimenes()

	datos := map[string]interface{}{
		"Trabajadores":       trabajadores,
		"Puestos":            puestos,
		"RegimenesLaborales": regimenes,
	}

	tmpl, _ := template.ParseFiles("ui/templates/tenant/contratos_ui.html")
	tmpl.Execute(w, datos)
}

func (h *ContratoHandler) Listar(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	busqueda := r.URL.Query().Get("buscar")
	limiteStr := r.URL.Query().Get("limite")
	paginaStr := r.URL.Query().Get("pagina")
	regimenStr := r.URL.Query().Get("regimen_laboral_id")

	limite, err := strconv.Atoi(limiteStr)
	if err != nil || limite <= 0 {
		limite = 10 // Por defecto mostramos 10
	}

	pagina, err := strconv.Atoi(paginaStr)
	if err != nil || pagina <= 0 {
		pagina = 1 // Por defecto empezamos en la página 1
	}

	regimenID, err := strconv.Atoi(regimenStr)
	if err != nil {
		regimenID = 0
	}

	offset := (pagina - 1) * limite

	contratos, totalRegistros, err := h.Repo.ObtenerTodosPaginado(tenantID, busqueda, regimenID, limite, offset)
	if err != nil {
		http.Error(w, "Error al obtener los contratos", http.StatusInternalServerError)
		return
	}
	totalPaginas := (totalRegistros + limite - 1) / limite

	if totalPaginas == 0 {
		totalPaginas = 1
	}

	// Construimos los datos struc y objetos al vuelo
	datosPaginacion := struct {
		Contratos       []models.Contrato
		TotalPaginas    int
		PaginaActual    int
		PaginaAnterior  int
		PaginaSiguiente int
	}{
		Contratos:       contratos,
		TotalPaginas:    totalPaginas,
		PaginaActual:    pagina,
		PaginaAnterior:  pagina - 1,
		PaginaSiguiente: pagina + 1,
	}

	tmpl, _ := template.ParseFiles("ui/templates/tenant/contratos_ui.html")
	tmpl.ExecuteTemplate(w, "tabla_contratos", datosPaginacion)
}

func (h *ContratoHandler) Crear(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	tenantID := obtenerTenantID(r)
	tID, _ := strconv.Atoi(r.FormValue("trabajador_id"))
	pID, _ := strconv.Atoi(r.FormValue("puesto_id"))

	// === 1. NUEVA VALIDACIÓN DE NEGOCIO ===
	tieneActivo, err := h.Repo.TieneContratoActivo(tID, tenantID)
	if err != nil {
		http.Error(w, "Error validando el estado del trabajador", http.StatusInternalServerError)
		return
	}

	if tieneActivo {
		// TRUCO HTMX: Devolvemos un fragmento HTML con la etiqueta hx-swap-oob="true".
		// HTMX buscará el div con id="alerta-contrato" y le inyectará este error, sin tocar la tabla.
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`
			<div id="alerta-contrato" hx-swap-oob="true">
				<article style="background-color: #ffcdd2; color: #b71c1c; padding: 1rem; margin-bottom: 1rem; border-radius: 5px; font-weight: bold;">
					❌ Error: El trabajador seleccionado ya posee un contrato activo (Plaza ocupada). Debe finalizarlo antes de asignarle uno nuevo.
				</article>
			</div>
		`))
		return
	}
	// =======================================

	fFinStr := r.FormValue("fecha_fin")
	var fFin *string
	if strings.TrimSpace(fFinStr) != "" {
		fFin = &fFinStr
	}

	nuevoContrato := models.Contrato{
		TenantID:     tenantID,
		TrabajadorID: tID,
		PuestoID:     pID,
		FechaInicio:  r.FormValue("fecha_inicio"),
		FechaFin:     fFin,
		Activo:       r.FormValue("activo") == "on",
		TipoContrato: r.FormValue("tipo_contrato"),
	}

	// Instanciamos el contrato service para llamar a la funcion CrearContrato
	servicioContrato := services.ContratoService{
		RepoPuesto:     h.PuestoRepo,
		Repo:           h.Repo,
		RepoTrabajador: h.TrabajadorRepo,
	}

	// Si el contrato se crea con éxito, enviamos una orden OOB para "limpiar" cualquier alerta anterior
	w.Write([]byte(`<div id="alerta-contrato" hx-swap-oob="true"></div>`))

	// Disparamos la creación e inyección automática de conceptos y pensiones
	err = servicioContrato.CrearContrato(&nuevoContrato)
	if err != nil {
		log.Println("Error al crear contrato:", err)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`
			<div id="alerta-contrato" hx-swap-oob="true">
				<article style="background-color: #ffcdd2; color: #b71c1c; padding: 1rem; margin-bottom: 1rem; border-radius: 5px; font-weight: bold;">
					❌ Error: No se pudo generar el contrato. Verifique la configuración del régimen y los clasificadores.
				</article>
			</div>
		`))
		return
	}

	// Finalmente, devolvemos la tabla actualizada como siempre
	h.Listar(w, r)
}

// FormularioCrearUI devuelve el form limpio
func (h *ContratoHandler) FormularioCrearUI(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	trabajadores, _ := h.TrabajadorRepo.ObtenerTodos(tenantID)
	puestos, _ := h.PuestoRepo.ObtenerVacantes(tenantID)
	datos := map[string]interface{}{"Trabajadores": trabajadores, "Puestos": puestos}

	tmpl, _ := template.ParseFiles("ui/templates/tenant/contratos_ui.html")
	tmpl.ExecuteTemplate(w, "formulario_crear", datos)
}

// FormularioDinamicoUI devuelve el formulario de creación parcial/completo con opciones dinámicas de contrato
func (h *ContratoHandler) FormularioDinamicoUI(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	puestoIDStr := r.URL.Query().Get("puesto_id")
	pID, _ := strconv.Atoi(puestoIDStr)

	// Extraemos otros campos para preservar su estado
	trabajadorIDStr := r.URL.Query().Get("trabajador_id")
	tID, _ := strconv.Atoi(trabajadorIDStr)
	fechaInicio := r.URL.Query().Get("fecha_inicio")
	fechaFin := r.URL.Query().Get("fecha_fin")

	_, hasPuesto := r.URL.Query()["puesto_id"]
	_, hasActivo := r.URL.Query()["activo"]
	activo := hasActivo || !hasPuesto

	var opciones []string
	if pID > 0 {
		puesto, err := h.PuestoRepo.ObtenerPorID(pID, tenantID)
		if err == nil {
			key := config.MapRegimenToKey(puesto.RegimenCodigo)
			if mapOpciones, ok := config.ClasificadorMefPorContrato[key]; ok {
				for k := range mapOpciones {
					opciones = append(opciones, k)
				}
				sort.Strings(opciones)
			}
		}
	}

	trabajadores, _ := h.TrabajadorRepo.ObtenerTodos(tenantID)
	puestos, _ := h.PuestoRepo.ObtenerVacantes(tenantID)

	datos := map[string]interface{}{
		"Trabajadores":             trabajadores,
		"Puestos":                  puestos,
		"PuestoSeleccionadoID":     pID,
		"OpcionesContrato":         opciones,
		"TrabajadorSeleccionadoID": tID,
		"FechaInicio":              fechaInicio,
		"FechaFin":                 fechaFin,
		"Activo":                   activo,
	}

	tmpl, _ := template.ParseFiles("ui/templates/tenant/contratos_ui.html")
	tmpl.ExecuteTemplate(w, "formulario_crear", datos)
}

// EditarUI carga el formulario de edición
func (h *ContratoHandler) EditarUI(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	tenantID := obtenerTenantID(r)

	contrato, _ := h.Repo.ObtenerPorID(id, tenantID)

	tmpl, _ := template.ParseFiles("ui/templates/tenant/contratos_ui.html")
	tmpl.ExecuteTemplate(w, "formulario_editar", contrato)
}

// Actualizar guarda cambios, recarga tabla y limpia form
func (h *ContratoHandler) Actualizar(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id, _ := strconv.Atoi(r.FormValue("id"))
	puestoID, _ := strconv.Atoi(r.FormValue("puesto_id"))

	fFinStr := r.FormValue("fecha_fin")
	var fFin *string
	if strings.TrimSpace(fFinStr) != "" {
		fFin = &fFinStr
	}

	cActualizado := models.Contrato{
		ID:          id,
		TenantID:    obtenerTenantID(r),
		PuestoID:    puestoID, // Lo enviamos oculto para poder liberar la plaza si se inactiva
		FechaInicio: r.FormValue("fecha_inicio"),
		FechaFin:    fFin,
		Activo:      r.FormValue("activo") == "on",
	}

	h.Repo.Actualizar(&cActualizado)

	// Pedimos recargar la tabla
	w.Header().Set("HX-Trigger", "recargarTablaContratos")

	// Volvemos al form de creación
	h.FormularioCrearUI(w, r)
}

// DescargarPlantilla genera y envía un archivo Excel base para la importación de contratos
func (h *ContratoHandler) DescargarPlantilla(w http.ResponseWriter, r *http.Request) {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Contratos"
	f.SetSheetName("Sheet1", sheet)

	// Cabeceras
	cabeceras := []string{
		"DOCUMENTO_TRABAJADOR", "CODIGO_AIRHSP", "NOMBRE_PUESTO", "TIPO_CONTRATO",
		"FECHA_INICIO", "FECHA_FIN", "ACTIVO",
	}
	for i, cabecera := range cabeceras {
		col := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue(sheet, col, cabecera)
	}

	// Datos de ejemplo
	ejemplos := [][]interface{}{
		{"74839201", "000456", "Especialista en Logística II", "Nombrado", "2026-01-01", "", "SI"},
		{"83920184", "000789", "Especialista en Sistemas I", "Transitorio", "2026-02-15", "2026-12-31", "SI"},
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
	f.SetCellValue(instruccionesSheet, "A1", "INSTRUCCIONES DE LLENADO PARA CONTRATOS")
	f.SetCellValue(instruccionesSheet, "A3", "DOCUMENTO_TRABAJADOR: Obligatorio. Número de documento (DNI, CE, Pasaporte) de un trabajador registrado y activo.")
	f.SetCellValue(instruccionesSheet, "A4", "CODIGO_AIRHSP: Obligatorio. Código AIRHSP de la plaza (puesto) que ocupará el trabajador. Debe ser una plaza existente, activa y vacante.")
	f.SetCellValue(instruccionesSheet, "A5", "NOMBRE_PUESTO: Opcional e Informativo. Nombre de la plaza (no es validado, sirve de guía).")
	f.SetCellValue(instruccionesSheet, "A6", "TIPO_CONTRATO: Obligatorio. Tipo de contrato MEF. Debe coincidir con los admitidos para el régimen de la plaza (ej. Nombrado, A plazo fijo, Indeterminado, Transitorio).")
	f.SetCellValue(instruccionesSheet, "A7", "FECHA_INICIO: Obligatorio. Fecha de inicio del contrato en formato AAAA-MM-DD (ej: 2026-01-01).")
	f.SetCellValue(instruccionesSheet, "A8", "FECHA_FIN: Opcional. Fecha de término en formato AAAA-MM-DD. Dejar vacío si es un contrato indeterminado.")
	f.SetCellValue(instruccionesSheet, "A9", "ACTIVO: Opcional. SI o NO. Por defecto es SI.")

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=plantilla_contratos.xlsx")

	if err := f.Write(w); err != nil {
		log.Printf("[ERROR] Al generar plantilla Excel de contratos: %v", err)
	}
}

// ImportarExcel procesa la subida de un archivo Excel, lo valida concurrientemente y realiza la importación
func (h *ContratoHandler) ImportarExcel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	tenantID := obtenerTenantID(r)

	// 1. Leer el formulario multipart
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

	// 3. Pre-cargar catálogos en memoria para resolución de claves
	// Trabajadores activos
	trabajadoresList, err := h.TrabajadorRepo.ObtenerTodos(tenantID)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(fmt.Sprintf(`<p style="color:red; margin:0;">⚠️ Error al cargar catálogo de trabajadores: %v</p>`, err)))
		return
	}
	mapaTrabajadores := make(map[string]int)
	for _, t := range trabajadoresList {
		if t.Activo {
			mapaTrabajadores[strings.TrimSpace(t.NumeroDocumento)] = t.ID
		}
	}

	// Base de datos y Trabajadores con contratos activos
	db := h.PuestoRepo.DB()

	// Puestos activos con su régimen
	type PuestoImportInfo struct {
		ID            int
		RegimenID     int
		RegimenCodigo string
		Estado        string
		Nombre        string
	}
	mapaPuestos := make(map[string]PuestoImportInfo)
	rowsPuestos, err := db.Query(`
		SELECT p.id, p.regimen_id, rl.codigo, p.estado, p.nombre, COALESCE(p.codigo_airhsp, '')
		FROM puestos p
		INNER JOIN regimenes_laborales rl ON p.regimen_id = rl.id
		WHERE p.tenant_id = $1 AND p.activo = true
	`, tenantID)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(fmt.Sprintf(`<p style="color:red; margin:0;">⚠️ Error al cargar catálogo de puestos: %v</p>`, err)))
		return
	}
	defer rowsPuestos.Close()

	for rowsPuestos.Next() {
		var info PuestoImportInfo
		var airhsp string
		err := rowsPuestos.Scan(&info.ID, &info.RegimenID, &info.RegimenCodigo, &info.Estado, &info.Nombre, &airhsp)
		if err == nil && airhsp != "" {
			mapaPuestos[strings.TrimSpace(airhsp)] = info
		}
	}
	rowsContratos, err := db.Query(`
		SELECT trabajador_id 
		FROM contratos 
		WHERE tenant_id = $1 AND activo = true
	`, tenantID)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(fmt.Sprintf(`<p style="color:red; margin:0;">⚠️ Error al validar contratos activos: %v</p>`, err)))
		return
	}
	defer rowsContratos.Close()

	mapaTrabajadoresConContratoActivo := make(map[int]bool)
	for rowsContratos.Next() {
		var tID int
		if err := rowsContratos.Scan(&tID); err == nil {
			mapaTrabajadoresConContratoActivo[tID] = true
		}
	}

	// Estructuras para la concurrencia
	type RowJob struct {
		Index int
		Row   []string
	}
	type RowResult struct {
		Index    int
		Contrato models.Contrato
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
					results <- RowResult{Index: job.Index}
					continue
				}

				// Validar número de columnas mínimo
				if len(fila) < 5 {
					results <- RowResult{
						Index: job.Index,
						Error: fmt.Errorf("Fila %d: Columnas incompletas (mínimo se requieren 5 columnas: Documento Trabajador, Código AIRHSP, Nombre Puesto (Informativo), Tipo Contrato y Fecha Inicio)", numFila),
					}
					continue
				}

				docTrabajador := strings.TrimSpace(fila[0])
				codigoAirhsp := strings.TrimSpace(fila[1])
				tipoContrato := strings.TrimSpace(fila[3])
				fechaInicioRaw := strings.TrimSpace(fila[4])

				// Validar obligatoriedad
				if docTrabajador == "" {
					results <- RowResult{Index: job.Index, Error: fmt.Errorf("Fila %d: El Documento del Trabajador es obligatorio", numFila)}
					continue
				}
				if codigoAirhsp == "" {
					results <- RowResult{Index: job.Index, Error: fmt.Errorf("Fila %d: El Código AIRHSP de la plaza es obligatorio", numFila)}
					continue
				}
				if tipoContrato == "" {
					results <- RowResult{Index: job.Index, Error: fmt.Errorf("Fila %d: El Tipo de Contrato es obligatorio", numFila)}
					continue
				}
				if fechaInicioRaw == "" {
					results <- RowResult{Index: job.Index, Error: fmt.Errorf("Fila %d: La Fecha de Inicio es obligatoria", numFila)}
					continue
				}

				// Resoluciones
				tID, existeTrabajador := mapaTrabajadores[docTrabajador]
				if !existeTrabajador {
					results <- RowResult{Index: job.Index, Error: fmt.Errorf("Fila %d: El trabajador con documento '%s' no existe o está inactivo en el sistema", numFila, docTrabajador)}
					continue
				}

				puestoObj, existePuesto := mapaPuestos[codigoAirhsp]
				if !existePuesto {
					results <- RowResult{Index: job.Index, Error: fmt.Errorf("Fila %d: La plaza con código AIRHSP '%s' no existe en el sistema", numFila, codigoAirhsp)}
					continue
				}

				// Validación: trabajador con contrato activo en la DB
				if tieneContrato := mapaTrabajadoresConContratoActivo[tID]; tieneContrato {
					results <- RowResult{Index: job.Index, Error: fmt.Errorf("Fila %d: El trabajador con documento '%s' ya posee un contrato activo en el sistema", numFila, docTrabajador)}
					continue
				}

				// Validación: puesto vacante
				if puestoObj.Estado != "VACANTE" {
					results <- RowResult{Index: job.Index, Error: fmt.Errorf("Fila %d: La plaza con código AIRHSP '%s' ya está ocupada u ocupada temporalmente (Estado: %s)", numFila, codigoAirhsp, puestoObj.Estado)}
					continue
				}

				// Validación: tipo de contrato por régimen
				regCodigo := puestoObj.RegimenCodigo
				regKey := config.MapRegimenToKey(regCodigo)
				if regKey == "" {
					results <- RowResult{Index: job.Index, Error: fmt.Errorf("Fila %d: El régimen laboral de la plaza no es reconocido (%s)", numFila, regCodigo)}
					continue
				}
				options, ok := config.ClasificadorMefPorContrato[regKey]
				if !ok {
					results <- RowResult{Index: job.Index, Error: fmt.Errorf("Fila %d: No hay tipos de contrato configurados para el régimen %s", numFila, regKey)}
					continue
				}
				_, tipoContratoValido := options[tipoContrato]
				if !tipoContratoValido {
					var validTypes []string
					for k := range options {
						validTypes = append(validTypes, k)
					}
					sort.Strings(validTypes)
					results <- RowResult{Index: job.Index, Error: fmt.Errorf("Fila %d: Tipo de contrato '%s' inválido para el régimen '%s'. Valores permitidos: %s", numFila, tipoContrato, regKey, strings.Join(validTypes, ", "))}
					continue
				}

				// Fechas
				fechaInicio, errStart := parseFechaExcel(fechaInicioRaw)
				if errStart != nil {
					results <- RowResult{Index: job.Index, Error: fmt.Errorf("Fila %d: Fecha de Inicio inválida. %v", numFila, errStart)}
					continue
				}

				var fechaFin *string
				if len(fila) >= 6 {
					fechaFinRaw := strings.TrimSpace(fila[5])
					if fechaFinRaw != "" {
						fFin, errEnd := parseFechaExcel(fechaFinRaw)
						if errEnd != nil {
							results <- RowResult{Index: job.Index, Error: fmt.Errorf("Fila %d: Fecha de Fin inválida. %v", numFila, errEnd)}
							continue
						}
						fechaFin = &fFin
					}
				}

				// Activo
				activo := true
				if len(fila) >= 7 {
					activoRaw := strings.ToUpper(strings.TrimSpace(fila[6]))
					if activoRaw != "" {
						activo = (activoRaw == "SI" || activoRaw == "TRUE" || activoRaw == "1" || activoRaw == "ACTIVO")
					}
				}

				c := models.Contrato{
					TenantID:     tenantID,
					TrabajadorID: tID,
					PuestoID:     puestoObj.ID,
					FechaInicio:  fechaInicio,
					FechaFin:     fechaFin,
					Activo:       activo,
					TipoContrato: tipoContrato,
				}

				results <- RowResult{
					Index:    job.Index,
					Contrato: c,
					Error:    nil,
				}
			}
		}()
	}

	// Enviar a procesar (saltando la cabecera)
	for i := 1; i < len(filas); i++ {
		jobs <- RowJob{Index: i, Row: filas[i]}
	}
	close(jobs)

	// Recopilar resultados
	var contratos []models.Contrato
	var errores []string

	// Trackers de duplicados en la misma hoja
	trabajadoresProcesados := make(map[int]int) // trabajador_id -> numFila
	puestosProcesados := make(map[int]int)      // puesto_id -> numFila

	for i := 1; i < len(filas); i++ {
		res := <-results
		if res.Error != nil {
			errores = append(errores, res.Error.Error())
			continue
		}
		if res.Contrato.TrabajadorID == 0 {
			continue // Fila vacía
		}

		numFila := res.Index + 1

		// Validar duplicado de trabajador en la misma hoja
		if prevFila, dup := trabajadoresProcesados[res.Contrato.TrabajadorID]; dup {
			errores = append(errores, fmt.Sprintf("Fila %d: El trabajador ya está asignado a otro contrato en la fila %d de este archivo", numFila, prevFila))
			continue
		}
		// Validar duplicado de puesto en la misma hoja
		if prevFila, dup := puestosProcesados[res.Contrato.PuestoID]; dup {
			errores = append(errores, fmt.Sprintf("Fila %d: La plaza (puesto) ya está siendo ocupada por otro trabajador en la fila %d de este archivo", numFila, prevFila))
			continue
		}

		trabajadoresProcesados[res.Contrato.TrabajadorID] = numFila
		puestosProcesados[res.Contrato.PuestoID] = numFila

		contratos = append(contratos, res.Contrato)
	}

	// 4. Inserción secuencial llamando al servicio para ejecutar inyecciones automáticas
	servicioContrato := services.ContratoService{
		RepoPuesto:     h.PuestoRepo,
		Repo:           h.Repo,
		RepoTrabajador: h.TrabajadorRepo,
	}

	successCount := 0
	for _, c := range contratos {
		err := servicioContrato.CrearContrato(&c)
		rowNum := trabajadoresProcesados[c.TrabajadorID]
		if err != nil {
			errores = append(errores, fmt.Sprintf("Fila %d: Error al guardar contrato o inyectar conceptos: %v", rowNum, err))
		} else {
			successCount++
		}
	}

	// 5. Devolver reporte detallado de importación
	w.Header().Set("Content-Type", "text/html")
	if successCount > 0 {
		// Notificamos a la UI para recargar la tabla de contratos
		w.Header().Set("HX-Trigger", "recargarTablaContratos")
	}

	successHTML := ""
	if successCount > 0 {
		successHTML = fmt.Sprintf(`
			<article style="background-color: #e8f5e9; color: #1b5e20; padding: 1rem; border-radius: 5px; margin-bottom: 1rem; text-align: center; border: 1px solid #c8e6c9;">
				<strong>✅ Importación de Contratos Exitosa</strong><br>
				Se registraron <strong>%d</strong> contratos de trabajo y se inyectó su estructura de costos correctamente.
			</article>
		`, successCount)
	}

	errorsHTML := ""
	if len(errores) > 0 {
		var listItems string
		for _, errStr := range errores {
			listItems += fmt.Sprintf("<li><small>%s</small></li>", errStr)
		}
		errorsHTML = fmt.Sprintf(`
			<details open style="margin-top: 1rem; border: 1px solid #ffcdd2; background: #ffebee; border-radius: 4px; padding: 0.5rem 1rem;">
				<summary style="color: #c62828; font-weight: 500; cursor: pointer;">❌ Filas Omitidas debido a Errores (%d filas)</summary>
				<ul style="margin: 0.5rem 0 0 1.2rem; padding: 0; color: #b71c1c; text-align: left; font-size: 0.85rem;">
					%s
				</ul>
			</details>
		`, len(errores), listItems)
	}

	w.Write([]byte(fmt.Sprintf("%s%s", successHTML, errorsHTML)))
}
