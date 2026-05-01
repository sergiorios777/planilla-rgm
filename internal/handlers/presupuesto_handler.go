package handlers

import (
	"encoding/csv"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"time"

	"planilla-rgm/internal/repository"
	"planilla-rgm/internal/services"

	"github.com/jung-kurt/gofpdf"
	"github.com/xuri/excelize/v2"
)

// PresupuestoHandler maneja las peticiones web del módulo de Presupuesto Anual
type PresupuestoHandler struct {
	Service      *services.PresupuestoService
	PlanillaRepo *repository.PlanillaRepository // Lo necesitamos para traer los parámetros globales (UIT, etc.)
}

// NewPresupuestoHandler es el constructor
func NewPresupuestoHandler(svc *services.PresupuestoService, pRepo *repository.PlanillaRepository) *PresupuestoHandler {
	return &PresupuestoHandler{
		Service:      svc,
		PlanillaRepo: pRepo,
	}
}

// IndexUI carga la vista principal (panel de control) del módulo
func (h *PresupuestoHandler) IndexUI(w http.ResponseWriter, r *http.Request) {
	// Preparar la vista (crearemos este HTML en la siguiente fase)
	tmpl, err := template.ParseFiles("ui/templates/tenant/presupuesto_ui.html")
	if err != nil {
		http.Error(w, "Error cargando la vista: "+err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.ExecuteTemplate(w, "presupuesto_index", nil)
}

// Generar captura la petición del formulario, recolecta variables y lanza el motor
func (h *PresupuestoHandler) Generar(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	tenantID := obtenerTenantID(r)

	anioStr := r.FormValue("anio")
	anio, err := strconv.Atoi(anioStr)
	if err != nil {
		w.Write([]byte(`<article style="background-color: #ffebee; color: #c62828;">Error: Año inválido.</article>`))
		return
	}

	// 0. Recolectar variables de la UI
	tipoProyeccion := r.FormValue("tipo_proyeccion")
	// mesBaseStr := r.FormValue("mes_base") // <-- en el html esta deshabilitado pero lo dejamos para futuro
	// mesBase, err := strconv.Atoi(mesBaseStr)
	// if err != nil {
	// 	w.Write([]byte(`<article style="background-color: #ffebee; color: #c62828;">Error: Mes base inválido.</article>`))
	// 	return
	// }
	mesCorteStr := r.FormValue("mes_corte")
	mesCorte, err := strconv.Atoi(mesCorteStr)
	if err != nil {
		w.Write([]byte(`<article style="background-color: #ffebee; color: #c62828;">Error: Mes de corte inválido.</article>`))
		return
	}

	// 1. Recolectar parámetros globales y catálogos para el simulador
	// Solicitamos los parámetros asumiendo el mes 1 (Enero) del año solicitado
	parametros, _ := h.PlanillaRepo.ObtenerParametrosGlobales(anio, 1)
	mapaCodigos, _ := h.PlanillaRepo.ObtenerMapaCodigosID()
	mapaAfectaciones, _ := h.PlanillaRepo.ObtenerAfectacionesGlobales()

	// 2. Ejecutar el servicio matemático dependiendo del tipo de proyección
	switch tipoProyeccion {
	case "PIA":
		err = h.Service.GenerarProyeccionPIA(tenantID, anio, parametros, mapaCodigos, mapaAfectaciones)
	case "VIGENTE":
		err = h.Service.GenerarProyeccionAnioVigente(tenantID, anio, mesCorte, parametros, mapaCodigos, mapaAfectaciones)
	default:
		err = fmt.Errorf("tipo de proyección no reconocido: %s", tipoProyeccion)
	}

	if err != nil {
		// Retornamos el error formateado para que HTMX lo muestre en pantalla
		w.Write([]byte(`<article style="background-color: #ffebee; color: #c62828;">Error al generar la proyección: ` + err.Error() + `</article>`))
		return
	}

	// 3. Notificar éxito y entregar un botón para cargar la matriz
	tipoTexto := "Presupuesto Institucional de Apertura (PIA)"
	if tipoProyeccion == "VIGENTE" {
		tipoTexto = "Presupuesto de Año Vigente"
	}

	mensajeExito := `
	<article style="background-color: #e8f5e9; color: #2e7d32; border-color: #c8e6c9;">
		<strong>¡Éxito!</strong> La proyección del ` + tipoTexto + ` para el año ` + anioStr + ` ha sido generada correctamente.
		<div style="margin-top: 1rem;">
			<button class="primary" hx-get="/tenant/presupuesto/matriz?anio=` + anioStr + `&tipo=` + tipoProyeccion + `" hx-target="#matriz-contenedor">
				👀 Ver Matriz Generada
			</button>
		</div>
	</article>
	<div id="matriz-contenedor" style="margin-top: 2rem;"></div>
	`
	w.Write([]byte(mensajeExito))
}

// CargarMatriz recupera los datos de la BD y renderiza solo la tabla HTML
func (h *PresupuestoHandler) CargarMatriz(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)

	anioStr := r.URL.Query().Get("anio")
	anio, err := strconv.Atoi(anioStr)
	if err != nil || anio == 0 {
		// Por defecto buscamos 2027 si no se envía el parámetro
		anio = 2027
	}

	// Consultamos a la base de datos
	matriz, err := h.Service.PresupuestoRepo.ObtenerMatrizPorAnio(tenantID, anio)
	if err != nil {
		w.Write([]byte(`<article style="background-color: #ffebee; color: #c62828;">Error consultando la base de datos.</article>`))
		return
	}

	if len(matriz) == 0 {
		w.Write([]byte(`<article>No hay datos generados para el año ` + strconv.Itoa(anio) + `. Por favor, genera la proyección primero.</article>`))
		return
	}

	// Renderizamos solo el fragmento de la tabla (lo crearemos en el siguiente paso)
	tmpl, err := template.ParseFiles("ui/templates/tenant/presupuesto_ui.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.ExecuteTemplate(w, "presupuesto_matriz", map[string]interface{}{
		"Anio":   anio,
		"Matriz": matriz,
	})
}

// ExportarCSV genera un archivo de texto separado por comas
func (h *PresupuestoHandler) ExportarCSV(w http.ResponseWriter, r *http.Request) {
	// 1. Obtener parámetros del formulario
	tenantID := obtenerTenantID(r)
	anio, _ := strconv.Atoi(r.URL.Query().Get("anio"))

	// 2. Traer datos de la base de datos usando tu repositorio existente
	matriz, err := h.Service.PresupuestoRepo.ObtenerMatrizPorAnio(tenantID, anio)
	if err != nil || len(matriz) == 0 {
		http.Error(w, "No hay datos para exportar en el año seleccionado", http.StatusNotFound)
		return
	}

	// 3. Configurar cabeceras HTTP para forzar la descarga en el navegador
	nombreArchivo := fmt.Sprintf("Presupuesto_%d_%s.csv", anio, time.Now().Format("20060102"))
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="`+nombreArchivo+`"`)

	// 4. Escribir el CSV
	writer := csv.NewWriter(w)
	defer writer.Flush() // Asegura que todo el buffer se envíe al final

	// Escribir la cabecera
	cabeceras := []string{"Meta", "Fuente_Rubro", "Clasificador", "Ene", "Feb", "Mar", "Abr", "May", "Jun", "Jul", "Ago", "Sep", "Oct", "Nov", "Dic", "Total_Anual"}
	writer.Write(cabeceras)

	// Escribir las filas dinámicamente
	for _, fila := range matriz {
		filaStrings := []string{
			fila.MetaCodigo,
			fila.FuenteRubroCodigo,
			fila.ClasificadorCodigoLimpio,
			fmt.Sprintf("%.2f", fila.Meses[0]), fmt.Sprintf("%.2f", fila.Meses[1]),
			fmt.Sprintf("%.2f", fila.Meses[2]), fmt.Sprintf("%.2f", fila.Meses[3]),
			fmt.Sprintf("%.2f", fila.Meses[4]), fmt.Sprintf("%.2f", fila.Meses[5]),
			fmt.Sprintf("%.2f", fila.Meses[6]), fmt.Sprintf("%.2f", fila.Meses[7]),
			fmt.Sprintf("%.2f", fila.Meses[8]), fmt.Sprintf("%.2f", fila.Meses[9]),
			fmt.Sprintf("%.2f", fila.Meses[10]), fmt.Sprintf("%.2f", fila.Meses[11]),
			fmt.Sprintf("%.2f", fila.TotalAnual),
		}
		writer.Write(filaStrings)
	}
}

// ExportarExcel genera un archivo nativo .xlsx
func (h *PresupuestoHandler) ExportarExcel(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	anio, _ := strconv.Atoi(r.URL.Query().Get("anio"))

	matriz, err := h.Service.PresupuestoRepo.ObtenerMatrizPorAnio(tenantID, anio)
	if err != nil || len(matriz) == 0 {
		http.Error(w, "No hay datos para exportar", http.StatusNotFound)
		return
	}

	// 1. Inicializar excelize
	f := excelize.NewFile()
	hoja := "Sheet1"
	f.SetSheetName(f.GetSheetName(1), hoja)

	// 2. Establecer cabeceras en la primera fila (A1, B1, etc.)
	cabeceras := []string{"Meta", "Fuente/Rubro", "Clasificador", "Ene", "Feb", "Mar", "Abr", "May", "Jun", "Jul", "Ago", "Sep", "Oct", "Nov", "Dic", "Total Anual"}
	for i, col := range cabeceras {
		// Calcula la letra de la columna (A, B, C...)
		letraColumna, _ := excelize.ColumnNumberToName(i + 1)
		celda := fmt.Sprintf("%s1", letraColumna)
		f.SetCellValue(hoja, celda, col)
	}

	// 3. Llenar los datos empezando en la fila 2
	for indexFila, data := range matriz {
		numeroFila := indexFila + 2 // Empezamos en la fila 2 porque la 1 es la cabecera
		f.SetCellValue(hoja, fmt.Sprintf("A%d", numeroFila), data.MetaCodigo)
		f.SetCellValue(hoja, fmt.Sprintf("B%d", numeroFila), data.FuenteRubroCodigo)
		f.SetCellValue(hoja, fmt.Sprintf("C%d", numeroFila), data.ClasificadorCodigoLimpio)

		// Llenar meses
		for i := 0; i < 12; i++ {
			letraColumna, _ := excelize.ColumnNumberToName(i + 4) // Los meses empiezan en la columna D (4)
			f.SetCellValue(hoja, fmt.Sprintf("%s%d", letraColumna, numeroFila), data.Meses[i])
		}
		// Llenar total
		f.SetCellValue(hoja, fmt.Sprintf("P%d", numeroFila), data.TotalAnual) // Columna P es la 16
	}

	// 4. Configurar respuesta y enviar archivo al navegador
	nombreArchivo := fmt.Sprintf("Presupuesto_%d.xlsx", anio)
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="`+nombreArchivo+`"`)

	f.Write(w) // Escribe los bytes generados directamente en la respuesta HTTP
}

// ExportarPDF genera un documento usando gofpdf en formato Apaisado (Landscape)
func (h *PresupuestoHandler) ExportarPDF(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	anio, _ := strconv.Atoi(r.URL.Query().Get("anio"))

	matriz, err := h.Service.PresupuestoRepo.ObtenerMatrizPorAnio(tenantID, anio)
	if err != nil || len(matriz) == 0 {
		http.Error(w, "No hay datos para exportar", http.StatusNotFound)
		return
	}

	// 1. Inicializar PDF: L (Landscape), mm (Milímetros), A4 (Tamaño hoja)
	pdf := gofpdf.New("L", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 12)

	tr := pdf.UnicodeTranslatorFromDescriptor("")

	// ==============================================================
	// NUEVO: Intentar cargar el logo de la entidad
	// ==============================================================
	// Asumimos que los logos se guardan con el ID del tenant, por ejemplo: logo_1.png
	// Asegúrate de importar el paquete "os" en la cabecera de tu archivo si no lo tienes.
	rutaLogo := "ui/static/uploads/logos/logo_" + strconv.Itoa(tenantID) + ".png"

	// os.Stat verifica si el archivo existe en el disco duro
	if _, err := os.Stat(rutaLogo); err == nil {
		// Insertar imagen: Ruta, X, Y, Ancho (Alto 0 mantiene proporción)
		pdf.ImageOptions(rutaLogo, 10, 8, 25, 0, false, gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}, 0, "")
	}

	// 2. Imprimir Título principal (ajustamos un poco la posición Y hacia abajo por el logo)
	pdf.SetFont("Arial", "B", 14)
	pdf.SetXY(40, 15) // Movemos el cursor a la derecha del logo
	titulo := tr(fmt.Sprintf("Matriz de Presupuesto Institucional - Año %d", anio))
	pdf.Cell(40, 10, titulo)
	pdf.Ln(15) // Salto de línea

	// 2. Configurar anchuras de columnas (Total ~ 280mm de ancho en A4 Landscape)
	anchos := []float64{15, 20, 25, 17, 17, 17, 17, 17, 17, 17, 17, 17, 17, 17, 17, 20}
	cabeceras := []string{"Meta", "Fuente", "Clasif.", "Ene", "Feb", "Mar", "Abr", "May", "Jun", "Jul", "Ago", "Sep", "Oct", "Nov", "Dic", "Total"}

	// Imprimir cabeceras
	pdf.SetFont("Arial", "B", 8)
	for i, col := range cabeceras {
		pdf.CellFormat(anchos[i], 7, col, "1", 0, "C", false, 0, "")
	}
	pdf.Ln(-1)

	// 3. Imprimir filas de datos
	pdf.SetFont("Arial", "", 7)
	for _, data := range matriz {
		// Acortamos textos si es necesario para que entren en el PDF
		pdf.CellFormat(anchos[0], 6, data.MetaCodigo, "1", 0, "L", false, 0, "")
		pdf.CellFormat(anchos[1], 6, data.FuenteRubroCodigo, "1", 0, "L", false, 0, "")
		pdf.CellFormat(anchos[2], 6, data.ClasificadorCodigoLimpio, "1", 0, "L", false, 0, "")

		for i := 0; i < 12; i++ {
			pdf.CellFormat(anchos[i+3], 6, fmt.Sprintf("%.2f", data.Meses[i]), "1", 0, "R", false, 0, "")
		}
		pdf.CellFormat(anchos[15], 6, fmt.Sprintf("%.2f", data.TotalAnual), "1", 0, "R", false, 0, "")
		pdf.Ln(-1)
	}

	// 4. Enviar al navegador
	nombreArchivo := fmt.Sprintf("Presupuesto_%d.pdf", anio)
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="`+nombreArchivo+`"`)

	pdf.Output(w)
}
