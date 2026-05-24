package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

type MetaHandler struct {
	Repo *repository.MetaRepository
}

func (h *MetaHandler) VistaUI(w http.ResponseWriter, r *http.Request) {
	tmpl, _ := template.ParseFiles("ui/templates/tenant/metas_ui.html")
	tmpl.Execute(w, nil)
}

func (h *MetaHandler) Listar(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r) // Usamos el helper que ya existe en este paquete
	busqueda := r.URL.Query().Get("buscar")
	estado := r.URL.Query().Get("estado")
	limiteStr := r.URL.Query().Get("limite")
	paginaStr := r.URL.Query().Get("pagina")

	limite, err := strconv.Atoi(limiteStr)
	if err != nil || limite <= 0 {
		limite = 10 // Por defecto mostramos 10
	}

	pagina, err := strconv.Atoi(paginaStr)
	if err != nil || pagina <= 0 {
		pagina = 1 // Por defecto empezamos en la página 1
	}

	offset := (pagina - 1) * limite

	metas, totalRegistros, err := h.Repo.ObtenerTodosPaginacion(tenantID, busqueda, estado, limite, offset)
	if err != nil {
		http.Error(w, "Error al obtener las metas", http.StatusInternalServerError)
		return
	}

	totalPaginas := (totalRegistros + limite - 1) / limite

	if totalPaginas == 0 {
		totalPaginas = 1
	}

	// Construimos los datos struc y objetos al vuelo
	datosVista := struct {
		Metas           []models.MetaPresupuestal
		TotalPaginas    int
		PaginaActual    int
		PaginaAnterior  int
		PaginaSiguiente int
	}{
		Metas:           metas,
		TotalPaginas:    totalPaginas,
		PaginaActual:    pagina,
		PaginaAnterior:  pagina - 1,
		PaginaSiguiente: pagina + 1,
	}

	tmpl, _ := template.ParseFiles("ui/templates/tenant/metas_ui.html")
	tmpl.ExecuteTemplate(w, "tabla_metas", datosVista)
}

func (h *MetaHandler) Crear(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	anio, _ := strconv.Atoi(r.FormValue("anio"))

	nuevaMeta := models.MetaPresupuestal{
		TenantID:    obtenerTenantID(r),
		Anio:        anio,
		Codigo:      r.FormValue("codigo"),
		Descripcion: r.FormValue("descripcion"),
		Activo:      r.FormValue("activo") == "on",
	}

	h.Repo.Crear(&nuevaMeta)
	h.Listar(w, r)
}

// FormularioCrearUI devuelve el formulario limpio
func (h *MetaHandler) FormularioCrearUI(w http.ResponseWriter, r *http.Request) {
	tmpl, _ := template.ParseFiles("ui/templates/tenant/metas_ui.html") // Ajusta el nombre si tu HTML se llama diferente
	tmpl.ExecuteTemplate(w, "formulario_crear", nil)
}

// EditarUI carga el formulario precargado
func (h *MetaHandler) EditarUI(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	tenantID := obtenerTenantID(r)

	meta, _ := h.Repo.ObtenerPorID(id, tenantID)

	tmpl, _ := template.ParseFiles("ui/templates/tenant/metas_ui.html")
	tmpl.ExecuteTemplate(w, "formulario_editar", meta)
}

// Actualizar guarda cambios y recarga la tabla
func (h *MetaHandler) Actualizar(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id, _ := strconv.Atoi(r.FormValue("id"))

	metaActualizada := models.MetaPresupuestal{
		ID:          id,
		TenantID:    obtenerTenantID(r),
		Codigo:      r.FormValue("codigo"),
		Descripcion: r.FormValue("descripcion"),
		Activo:      r.FormValue("activo") == "on",
	}

	h.Repo.Actualizar(&metaActualizada)

	// Pedimos a HTMX que recargue la tabla
	w.Header().Set("HX-Trigger", "recargarTablaMetas")

	// Volvemos a pintar el formulario de creación
	h.FormularioCrearUI(w, r)
}

// DescargarPlantilla genera y sirve la plantilla de metas en formato Excel al vuelo
func (h *MetaHandler) DescargarPlantilla(w http.ResponseWriter, r *http.Request) {
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "Metas"
	f.SetSheetName("Sheet1", sheetName)

	// Encabezados
	f.SetCellValue(sheetName, "A1", "Año")
	f.SetCellValue(sheetName, "B1", "Código")
	f.SetCellValue(sheetName, "C1", "Descripción")
	f.SetCellValue(sheetName, "D1", "Activo")

	// Fila de ejemplo
	f.SetCellValue(sheetName, "A2", 2026)
	f.SetCellValue(sheetName, "B2", "0015")
	f.SetCellValue(sheetName, "C2", "PATRULLAJE POR SECTOR")
	f.SetCellValue(sheetName, "D2", "SI")

	// Ancho de columnas sugerido
	f.SetColWidth(sheetName, "A", "A", 12)
	f.SetColWidth(sheetName, "B", "B", 15)
	f.SetColWidth(sheetName, "C", "C", 40)
	f.SetColWidth(sheetName, "D", "D", 12)

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=plantilla_metas.xlsx")

	if err := f.Write(w); err != nil {
		http.Error(w, "Error al generar la plantilla", http.StatusInternalServerError)
	}
}

// ImportarExcel procesa la subida de un archivo Excel, lo valida de manera atómica y lo importa
func (h *MetaHandler) ImportarExcel(w http.ResponseWriter, r *http.Request) {
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

	var metas []models.MetaPresupuestal
	seen := make(map[string]int) // Control de duplicados en el propio archivo excel

	// 3. Validar fila por fila de forma robusta en memoria antes de tocar BD
	for i, fila := range filas {
		if i == 0 {
			continue // Saltamos la cabecera
		}

		// Si es una fila completamente vacía, la ignoramos de forma segura
		filaVacia := true
		for _, celda := range fila {
			if strings.TrimSpace(celda) != "" {
				filaVacia = false
				break
			}
		}
		if filaVacia {
			continue
		}

		numFila := i + 1

		// Necesitamos al menos Año, Código y Descripción
		if len(fila) < 3 {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(fmt.Sprintf(`<p style="color:red; margin:0;">⚠️ Fila %d: Columnas incompletas. Se requieren: Año, Código y Descripción.</p>`, numFila)))
			return
		}

		anioRaw := strings.TrimSpace(fila[0])
		codigo := strings.TrimSpace(fila[1])
		descripcion := strings.TrimSpace(fila[2])
		activoRaw := "SI"
		if len(fila) >= 4 {
			activoRaw = strings.TrimSpace(fila[3])
		}

		// A. Validar Año
		if anioRaw == "" {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(fmt.Sprintf(`<p style="color:red; margin:0;">⚠️ Fila %d: El campo 'Año' es obligatorio y no puede estar vacío.</p>`, numFila)))
			return
		}
		anio, err := strconv.Atoi(anioRaw)
		if err != nil || anio < 2000 || anio > 2100 {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(fmt.Sprintf(`<p style="color:red; margin:0;">⚠️ Fila %d: El Año '%s' debe ser un número entero válido entre el 2000 y el 2100.</p>`, numFila, anioRaw)))
			return
		}

		// B. Validar Código
		if codigo == "" {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(fmt.Sprintf(`<p style="color:red; margin:0;">⚠️ Fila %d: El campo 'Código' es obligatorio.</p>`, numFila)))
			return
		}
		if len(codigo) > 20 {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(fmt.Sprintf(`<p style="color:red; margin:0;">⚠️ Fila %d: El Código '%s' es demasiado largo (máximo 20 caracteres).</p>`, numFila, codigo)))
			return
		}

		// C. Validar Descripción
		if descripcion == "" {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(fmt.Sprintf(`<p style="color:red; margin:0;">⚠️ Fila %d: La 'Descripción' de la meta es obligatoria.</p>`, numFila)))
			return
		}
		if len(descripcion) > 512 {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(fmt.Sprintf(`<p style="color:red; margin:0;">⚠️ Fila %d: La Descripción supera el límite permitido de 512 caracteres.</p>`, numFila)))
			return
		}

		// D. Validar y normalizar Activo
		activo := true
		if activoRaw != "" {
			val := strings.ToUpper(activoRaw)
			switch val {
			case "SI", "TRUE", "1", "ACTIVO":
				activo = true
			case "NO", "FALSE", "0", "INACTIVO":
				activo = false
			default:
				w.Header().Set("Content-Type", "text/html")
				w.Write([]byte(fmt.Sprintf(`<p style="color:red; margin:0;">⚠️ Fila %d: El valor del campo 'Activo' ('%s') es inválido. Use SI, NO, TRUE o FALSE.</p>`, numFila, activoRaw)))
				return
			}
		}

		// E. Detectar duplicados en el mismo archivo
		clave := fmt.Sprintf("%d-%s", anio, codigo)
		if filaDuplicada, existe := seen[clave]; existe {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(fmt.Sprintf(`<p style="color:red; margin:0;">⚠️ Fila %d: La combinación de Año '%d' y Código '%s' está repetida dentro de este mismo archivo Excel (apareció antes en la fila %d).</p>`, numFila, anio, codigo, filaDuplicada)))
			return
		}
		seen[clave] = numFila

		metas = append(metas, models.MetaPresupuestal{
			Anio:        anio,
			Codigo:      codigo,
			Descripcion: descripcion,
			Activo:      activo,
		})
	}

	if len(metas) == 0 {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<p style="color:red; margin:0;">⚠️ El archivo Excel no contiene filas de metas válidas para importar.</p>`))
		return
	}

	// 4. Inserción atómica en base de datos
	err = h.Repo.ImportarMetas(tenantID, metas)
	if err != nil {
		errorStr := err.Error()
		if strings.Contains(errorStr, "unique_meta_anio_tenant") || strings.Contains(errorStr, "duplicate key") {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`<p style="color:red; margin:0;">⚠️ Error de Importación: Una de las metas en el archivo ya existe registrada en la base de datos (clave duplicada para el mismo Año y Código). Se canceló toda la importación.</p>`))
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(fmt.Sprintf(`<p style="color:red; margin:0;">⚠️ Error de Base de Datos: %v. Se canceló toda la importación.</p>`, err)))
		return
	}

	// 5. Devolver mensaje de éxito
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(fmt.Sprintf(`
		<article style="background-color: #e8f5e9; color: #1b5e20; padding: 1rem; border-radius: 5px; margin: 0;">
			✅ Importación Exitosa.<br>
			Se registraron <strong>%d</strong> metas presupuestales correctamente y la transacción fue confirmada.
		</article>
	`, len(metas))))
}
