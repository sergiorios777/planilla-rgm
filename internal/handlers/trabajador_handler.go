package handlers

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

type TrabajadorHandler struct {
	Repo *repository.TrabajadorRepository
}

// obtenerTenantID es un helper para sacar el ID de forma segura de la sesión
func obtenerTenantID(r *http.Request) int {
	// El JWT parsea los números como float64, lo convertimos a int
	if val, ok := r.Context().Value("tenant_id").(float64); ok {
		return int(val)
	}
	return 0 // En un caso real, si es 0 deberíamos bloquear la petición
}

func (h *TrabajadorHandler) VistaUI(w http.ResponseWriter, r *http.Request) {
	// Extraemos la lista de AFPs de la BD para poblar el <select> de Crear
	afps, _ := h.Repo.ObtenerAFPsActivas()
	tmpl, _ := template.ParseFiles("ui/templates/tenant/trabajadores_ui.html")

	// Pasamos la lista a la vista principal
	tmpl.Execute(w, map[string]interface{}{
		"ListaAFPs": afps,
	})
}

func (h *TrabajadorHandler) Listar(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	busqueda := r.FormValue("buscar")
	limiteStr := r.FormValue("limite")
	paginaStr := r.FormValue("pagina")

	limite, err := strconv.Atoi(limiteStr)
	if err != nil || limite <= 0 {
		limite = 10 // Por defecto mostramos 10
	}

	pagina, err := strconv.Atoi(paginaStr)
	if err != nil || pagina <= 0 {
		pagina = 1 // Por defecto empezamos en la página 1
	}

	offset := (pagina - 1) * limite

	trabajadores, totalRegistros, err := h.Repo.ObtenerTodosPaginacion(tenantID, busqueda, limite, offset)
	if err != nil {
		http.Error(w, "Error al listar trabajadores", http.StatusInternalServerError)
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
		"/tenant/trabajadores/lista",
		"#lista-trabajadores",
		"#input-buscar-trabajador",
	)

	datos := struct {
		Trabajadores []models.Trabajador
		Paginacion   models.PaginacionDTO
	}{
		Trabajadores: trabajadores,
		Paginacion:   paginacion,
	}

	tmpl, _ := template.ParseFiles("ui/templates/tenant/trabajadores_ui.html", "ui/templates/components/paginacion.html")
	tmpl.ExecuteTemplate(w, "tabla_trabajadores", datos)
}

func (h *TrabajadorHandler) Crear(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	afpID, _ := strconv.Atoi(r.FormValue("afp_id"))
	fechaCese := strings.TrimSpace(r.FormValue("fecha_cese"))
	activo := r.FormValue("activo") == "on"
	if fechaCese != "" {
		activo = false
	}

	nuevoTrabajador := models.Trabajador{
		TenantID:           obtenerTenantID(r),
		TipoDocumento:      r.FormValue("tipo_documento"),
		NumeroDocumento:    r.FormValue("numero_documento"),
		Nombres:            r.FormValue("nombres"),
		ApellidoPaterno:    r.FormValue("apellido_paterno"),
		ApellidoMaterno:    r.FormValue("apellido_materno"),
		FechaNacimiento:    r.FormValue("fecha_nacimiento"),
		FechaIngreso:       r.FormValue("fecha_ingreso"),
		FechaCese:          fechaCese,
		Direccion:          strings.TrimSpace(r.FormValue("direccion")),
		Banco:              strings.TrimSpace(r.FormValue("banco")),
		Cuenta:             strings.TrimSpace(r.FormValue("cuenta")),
		Cci:                strings.TrimSpace(r.FormValue("cci")),
		Sexo:               r.FormValue("sexo"),
		Activo:             activo,
		RegimenPensionario: r.FormValue("regimen_pensionario"),
		AfpID:              afpID,
		AfpTipoComision:    r.FormValue("afp_tipo_comision"),
		Cuspp:              r.FormValue("cuspp"),
	}

	h.Repo.Crear(&nuevoTrabajador)
	h.Listar(w, r)
}

func (h *TrabajadorHandler) EditarUI(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	tenantID := obtenerTenantID(r)

	trabajador, err := h.Repo.ObtenerPorID(id, tenantID)
	if err != nil {
		http.Error(w, "Trabajador no encontrado", http.StatusNotFound)
		return
	}

	// Extraemos las AFPs para el desplegable de edición
	afps, _ := h.Repo.ObtenerAFPsActivas()

	// Creamos una estructura al vuelo para pasar el trabajador Y la lista al template
	data := struct {
		*models.Trabajador
		ListaAFPs map[int]string
	}{
		Trabajador: trabajador,
		ListaAFPs:  afps,
	}

	tmpl, _ := template.ParseFiles("ui/templates/tenant/trabajadores_ui.html")
	tmpl.ExecuteTemplate(w, "formulario_editar", data)
}

func (h *TrabajadorHandler) Actualizar(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	id, _ := strconv.Atoi(r.FormValue("id"))
	afpID, _ := strconv.Atoi(r.FormValue("afp_id"))
	fechaCese := strings.TrimSpace(r.FormValue("fecha_cese"))
	activo := r.FormValue("activo") == "on"
	if fechaCese != "" {
		activo = false
	}

	trabajadorEditado := models.Trabajador{
		ID:                 id,
		TenantID:           obtenerTenantID(r),
		TipoDocumento:      r.FormValue("tipo_documento"),
		NumeroDocumento:    r.FormValue("numero_documento"),
		Nombres:            r.FormValue("nombres"),
		ApellidoPaterno:    r.FormValue("apellido_paterno"),
		ApellidoMaterno:    r.FormValue("apellido_materno"),
		FechaNacimiento:    r.FormValue("fecha_nacimiento"),
		FechaIngreso:       r.FormValue("fecha_ingreso"),
		FechaCese:          fechaCese,
		Direccion:          strings.TrimSpace(r.FormValue("direccion")),
		Banco:              strings.TrimSpace(r.FormValue("banco")),
		Cuenta:             strings.TrimSpace(r.FormValue("cuenta")),
		Cci:                strings.TrimSpace(r.FormValue("cci")),
		Sexo:               r.FormValue("sexo"),
		Activo:             activo,
		RegimenPensionario: r.FormValue("regimen_pensionario"),
		AfpID:              afpID,
		AfpTipoComision:    r.FormValue("afp_tipo_comision"),
		Cuspp:              r.FormValue("cuspp"),
	}

	h.Repo.Actualizar(&trabajadorEditado)
	h.Listar(w, r)
}

// DescargarPlantilla genera y envía un archivo Excel base para la importación de trabajadores
func (h *TrabajadorHandler) DescargarPlantilla(w http.ResponseWriter, r *http.Request) {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Trabajadores"
	f.SetSheetName("Sheet1", sheet)

	// Cabeceras
	cabeceras := []string{
		"TIPO_DOCUMENTO", "NUMERO_DOCUMENTO", "NOMBRES", "APELLIDO_PATERNO", "APELLIDO_MATERNO",
		"FECHA_NACIMIENTO", "FECHA_INGRESO", "SEXO", "REGIMEN_PENSIONARIO", "AFP", "TIPO_COMISION", "CUSPP", "ACTIVO",
	}
	for i, cabecera := range cabeceras {
		col := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue(sheet, col, cabecera)
	}

	// Datos de ejemplo
	ejemplos := [][]interface{}{
		{"DNI", "45678912", "Juan Carlos", "Perez", "Gomez", "1990-05-15", "2020-01-15", "M", "ONP", "", "", "", "SI"},
		{"DNI", "78912345", "Maria Elena", "Flores", "Ramos", "1988-11-23", "2021-03-01", "F", "AFP", "INTEGRA", "MIXTA", "123456789012", "SI"},
		{"CE", "001234567", "John", "Smith", "Doe", "1985-02-10", "2019-07-01", "M", "AFP", "PRIMA", "FLUJO", "987654321098", "NO"},
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
	f.SetCellValue(instruccionesSheet, "A1", "INSTRUCCIONES DE LLENADO PARA TRABAJADORES")
	f.SetCellValue(instruccionesSheet, "A3", "TIPO_DOCUMENTO: Obligatorio. Debe ser uno de: DNI, CE, PASAPORTE")
	f.SetCellValue(instruccionesSheet, "A4", "NUMERO_DOCUMENTO: Obligatorio. Número de identificación. Único por trabajador.")
	f.SetCellValue(instruccionesSheet, "A5", "NOMBRES: Obligatorio. Nombres completos del trabajador.")
	f.SetCellValue(instruccionesSheet, "A6", "APELLIDO_PATERNO: Obligatorio. Apellido paterno.")
	f.SetCellValue(instruccionesSheet, "A7", "APELLIDO_MATERNO: Obligatorio. Apellido materno.")
	f.SetCellValue(instruccionesSheet, "A8", "FECHA_NACIMIENTO: Opcional. Formato YYYY-MM-DD (Ej. 1990-05-15).")
	f.SetCellValue(instruccionesSheet, "A9", "FECHA_INGRESO: Opcional. Fecha de ingreso a la entidad. Formato YYYY-MM-DD (Ej. 2020-01-15).")
	f.SetCellValue(instruccionesSheet, "A10", "SEXO: Obligatorio. Debe ser M o F.")
	f.SetCellValue(instruccionesSheet, "A11", "REGIMEN_PENSIONARIO: Obligatorio. Debe ser ONP o AFP.")
	f.SetCellValue(instruccionesSheet, "A12", "AFP: Obligatorio si REGIMEN_PENSIONARIO es AFP. Debe ser uno de: HABITAT, INTEGRA, PRIMA, PROFUTURO (o sus siglas: HBT, INT, PRM, PFR).")
	f.SetCellValue(instruccionesSheet, "A13", "TIPO_COMISION: Obligatorio si REGIMEN_PENSIONARIO es AFP. Debe ser uno de: FLUJO, MIXTA.")
	f.SetCellValue(instruccionesSheet, "A14", "CUSPP: Recomendado si REGIMEN_PENSIONARIO es AFP. Código Único del Sistema Privado de Pensiones (12 a 20 caracteres).")
	f.SetCellValue(instruccionesSheet, "A15", "ACTIVO: Opcional. Indica si el personal está laborando. Debe ser SI o NO. Por defecto es SI.")

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=plantilla_trabajadores.xlsx")

	if err := f.Write(w); err != nil {
		log.Printf("[ERROR] Al generar plantilla Excel de trabajadores: %v", err)
	}
}

// ImportarExcel procesa la subida de un archivo Excel, lo valida de manera atómica con un pool de workers y lo importa
func (h *TrabajadorHandler) ImportarExcel(w http.ResponseWriter, r *http.Request) {
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

	// Cargar catálogo de AFPs activas
	mapaAFPs, err := h.Repo.ObtenerMapaAFPsParaImportar()
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		w.Write(fmt.Appendf(nil, `<p style="color:red; margin:0;">⚠️ Error al cargar catálogo de AFPs: %v</p>`, err))
		return
	}

	// Estructuras para la concurrencia
	type RowJob struct {
		Index int
		Row   []string
	}
	type RowResult struct {
		Index      int
		Trabajador models.Trabajador
		Error      error
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

				// Validar columnas mínimas
				if len(fila) < 9 {
					results <- RowResult{Index: job.Index, Error: fmt.Errorf("Fila %d: Columnas incompletas (mínimo se requieren 9 columnas: Tipo Doc, Nro Doc, Nombres, Paterno, Materno, F. Nacimiento, F. Ingreso, Sexo, Régimen)", numFila)}
					continue
				}

				tipoDoc := strings.ToUpper(strings.TrimSpace(fila[0]))
				numDoc := strings.TrimSpace(fila[1])
				nombres := strings.TrimSpace(fila[2])
				paterno := strings.TrimSpace(fila[3])
				materno := strings.TrimSpace(fila[4])
				fechaNacRaw := strings.TrimSpace(fila[5])
				fechaIngRaw := strings.TrimSpace(fila[6])
				sexo := strings.ToUpper(strings.TrimSpace(fila[7]))
				regimen := strings.ToUpper(strings.TrimSpace(fila[8]))

				// Val: Tipo Doc
				if tipoDoc != "DNI" && tipoDoc != "CE" && tipoDoc != "PASAPORTE" {
					results <- RowResult{Index: job.Index, Error: fmt.Errorf("Fila %d: Tipo de documento '%s' inválido. Debe ser DNI, CE o PASAPORTE", numFila, tipoDoc)}
					continue
				}

				// Val: Nro Doc
				if numDoc == "" {
					results <- RowResult{Index: job.Index, Error: fmt.Errorf("Fila %d: El número de documento no puede estar vacío", numFila)}
					continue
				}
				if len(numDoc) > 20 {
					results <- RowResult{Index: job.Index, Error: fmt.Errorf("Fila %d: El número de documento '%s' es muy largo (máx. 20 caracteres)", numFila, numDoc)}
					continue
				}

				// Val: Nombres, Apellidos
				if nombres == "" || paterno == "" || materno == "" {
					results <- RowResult{Index: job.Index, Error: fmt.Errorf("Fila %d: Nombres, Apellido Paterno y Apellido Materno son campos obligatorios", numFila)}
					continue
				}
				if len(nombres) > 100 || len(paterno) > 100 || len(materno) > 100 {
					results <- RowResult{Index: job.Index, Error: fmt.Errorf("Fila %d: Nombres o apellidos superan el límite de 100 caracteres", numFila)}
					continue
				}

				// Val: Fecha Nacimiento
				var fechaNac string
				if fechaNacRaw != "" {
					var errFecha error
					fechaNac, errFecha = parseFechaExcel(fechaNacRaw)
					if errFecha != nil {
						results <- RowResult{Index: job.Index, Error: fmt.Errorf("Fila %d (Fecha Nacimiento): %v", numFila, errFecha)}
						continue
					}
				}

				// Val: Fecha Ingreso
				var fechaIng string
				if fechaIngRaw != "" {
					var errFecha error
					fechaIng, errFecha = parseFechaExcel(fechaIngRaw)
					if errFecha != nil {
						results <- RowResult{Index: job.Index, Error: fmt.Errorf("Fila %d (Fecha Ingreso): %v", numFila, errFecha)}
						continue
					}
				}

				// Val: Sexo
				if sexo != "M" && sexo != "F" {
					results <- RowResult{Index: job.Index, Error: fmt.Errorf("Fila %d: Sexo '%s' inválido. Debe ser M o F", numFila, sexo)}
					continue
				}

				// Val: Régimen
				if regimen != "ONP" && regimen != "AFP" {
					results <- RowResult{Index: job.Index, Error: fmt.Errorf("Fila %d: Régimen Pensionario '%s' inválido. Debe ser ONP o AFP", numFila, regimen)}
					continue
				}

				var afpID int
				var afpTipoComision string
				var cuspp string

				if regimen == "AFP" {
					if len(fila) < 11 {
						results <- RowResult{Index: job.Index, Error: fmt.Errorf("Fila %d: Faltan las columnas de AFP y Tipo de Comisión para el régimen AFP", numFila)}
						continue
					}

					afpRaw := strings.ToUpper(strings.TrimSpace(fila[9]))
					tipoComRaw := strings.ToUpper(strings.TrimSpace(fila[10]))

					if len(fila) >= 12 {
						cuspp = strings.TrimSpace(fila[11])
					}

					// Validar AFP
					idAFP, existeAFP := mapaAFPs[afpRaw]
					if !existeAFP {
						results <- RowResult{Index: job.Index, Error: fmt.Errorf("Fila %d: AFP '%s' no registrada o inactiva en el sistema", numFila, afpRaw)}
						continue
					}
					afpID = idAFP

					// Validar Tipo Comisión
					if tipoComRaw != "FLUJO" && tipoComRaw != "MIXTA" {
						results <- RowResult{Index: job.Index, Error: fmt.Errorf("Fila %d: Tipo de Comisión '%s' inválido. Debe ser FLUJO o MIXTA", numFila, tipoComRaw)}
						continue
					}
					afpTipoComision = tipoComRaw

					if len(cuspp) > 20 {
						results <- RowResult{Index: job.Index, Error: fmt.Errorf("Fila %d: El CUSPP '%s' supera los 20 caracteres", numFila, cuspp)}
						continue
					}
				}

				// Val: Activo
				activo := true
				if len(fila) >= 13 {
					activoRaw := strings.ToUpper(strings.TrimSpace(fila[12]))
					if activoRaw != "" {
						switch activoRaw {
						case "SI", "TRUE", "1", "ACTIVO":
							activo = true
						case "NO", "FALSE", "0", "INACTIVO":
							activo = false
						default:
							results <- RowResult{Index: job.Index, Error: fmt.Errorf("Fila %d: Valor Activo '%s' inválido. Use SI, NO, TRUE o FALSE", numFila, activoRaw)}
							continue
						}
					}
				}

				t := models.Trabajador{
					TenantID:           tenantID,
					TipoDocumento:      tipoDoc,
					NumeroDocumento:    numDoc,
					Nombres:            nombres,
					ApellidoPaterno:    paterno,
					ApellidoMaterno:    materno,
					FechaNacimiento:    fechaNac,
					FechaIngreso:       fechaIng,
					Sexo:               sexo,
					Activo:             activo,
					RegimenPensionario: regimen,
					AfpID:              afpID,
					AfpTipoComision:    afpTipoComision,
					Cuspp:              cuspp,
				}

				results <- RowResult{Index: job.Index, Trabajador: t, Error: nil}
			}
		}()
	}

	// Enviar a procesar (saltando fila 0 de cabecera)
	for i := 1; i < len(filas); i++ {
		jobs <- RowJob{Index: i, Row: filas[i]}
	}
	close(jobs)

	// Recopilar resultados
	var trabajadores []models.Trabajador
	seen := make(map[string]int)
	var validationError error

	for i := 1; i < len(filas); i++ {
		res := <-results
		if res.Error != nil {
			validationError = res.Error
			continue
		}
		if res.Trabajador.NumeroDocumento == "" {
			continue
		}

		// Validar duplicados en el propio archivo
		clave := fmt.Sprintf("%s-%s", res.Trabajador.TipoDocumento, res.Trabajador.NumeroDocumento)
		if filaDuplicada, existe := seen[clave]; existe {
			validationError = fmt.Errorf("Fila %d: El trabajador con documento %s %s está repetido dentro del mismo archivo Excel (apareció antes en la fila %d)", res.Index+1, res.Trabajador.TipoDocumento, res.Trabajador.NumeroDocumento, filaDuplicada)
		}
		seen[clave] = res.Index + 1

		trabajadores = append(trabajadores, res.Trabajador)
	}

	if validationError != nil {
		w.Header().Set("Content-Type", "text/html")
		w.Write(fmt.Appendf(nil, `<p style="color:red; margin:0;">⚠️ %v. Se canceló toda la importación.</p>`, validationError))
		return
	}

	if len(trabajadores) == 0 {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<p style="color:red; margin:0;">⚠️ El archivo Excel no contiene filas de trabajadores válidas para importar.</p>`))
		return
	}

	// 4. Inserción atómica en base de datos
	err = h.Repo.ImportarTrabajadores(tenantID, trabajadores)
	if err != nil {
		errorStr := err.Error()
		if strings.Contains(errorStr, "unique_documento_tenant") || strings.Contains(errorStr, "duplicate key") {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`<p style="color:red; margin:0;">⚠️ Error de Importación: Uno de los trabajadores en el archivo ya se encuentra registrado con el mismo tipo y número de documento. Se canceló toda la importación.</p>`))
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.Write(fmt.Appendf(nil, `<p style="color:red; margin:0;">⚠️ Error de Base de Datos: %v. Se canceló toda la importación.</p>`, err))
		return
	}

	// 5. Devolver mensaje de éxito
	w.Header().Set("Content-Type", "text/html")
	w.Write(fmt.Appendf(nil, `
		<article style="background-color: #e8f5e9; color: #1b5e20; padding: 1rem; border-radius: 5px; margin: 0;">
			✅ Importación Exitosa.<br>
			Se registraron <strong>%d</strong> trabajadores correctamente en el legajo y la transacción fue confirmada.
		</article>
	`, len(trabajadores)))
}

// parseFechaExcel es un helper para normalizar la fecha ingresada en Excel
func parseFechaExcel(val string) (string, error) {
	val = strings.TrimSpace(val)
	if val == "" {
		return "", nil
	}
	// Probar YYYY-MM-DD
	if t, err := time.Parse("2006-01-02", val); err == nil {
		return t.Format("2006-01-02"), nil
	}
	// Probar DD/MM/YYYY
	if t, err := time.Parse("02/01/2006", val); err == nil {
		return t.Format("2006-01-02"), nil
	}
	// Probar DD-MM-YYYY
	if t, err := time.Parse("02-01-2006", val); err == nil {
		return t.Format("2006-01-02"), nil
	}
	// Probar YYYY/MM/DD
	if t, err := time.Parse("2006/01/02", val); err == nil {
		return t.Format("2006-01-02"), nil
	}
	return "", fmt.Errorf("formato de fecha '%s' inválido. Use YYYY-MM-DD", val)
}


