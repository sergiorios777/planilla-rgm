package services

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"planilla-rgm/internal/models"
	"strings"

	"github.com/jung-kurt/gofpdf"
)

type PdfService struct{}

func NewPdfService() *PdfService {
	return &PdfService{}
}

// obtenerRutaLogo busca la existencia física del archivo del logo en el sistema
func obtenerRutaLogo(logoURL string) string {
	if logoURL == "" {
		return ""
	}
	candidatos := []string{
		logoURL,
		strings.TrimPrefix(logoURL, "/"),
		filepath.Join("ui", strings.TrimPrefix(logoURL, "/")),
		filepath.Join("ui", logoURL),
	}
	for _, c := range candidatos {
		if info, err := os.Stat(c); err == nil && !info.IsDir() {
			return c
		}
	}
	return ""
}

// GenerarReportePlanilla crea el "Boletón" con alturas uniformes y cabecera de dos filas
func (s *PdfService) GenerarReportePlanilla(datos *models.DatosReportePlanilla) ([]byte, error) {
	pdf := gofpdf.New("L", "mm", "A4", "")
	pdf.SetMargins(10, 10, 10)
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	// Pie de página: Descripción, periodo, indicador de BORRADOR y número de página
	pdf.AliasNbPages("{nb}")
	pdf.SetFooterFunc(func() {
		pdf.SetY(-12)

		infoPie := fmt.Sprintf("%s | Periodo: %02d/%d", datos.PlanillaDesc, datos.PlanillaMes, datos.PlanillaAnio)

		// Lado izquierdo: indicador BORRADOR (si aplica) y datos de la planilla
		if strings.EqualFold(strings.TrimSpace(datos.PlanillaEstado), "BORRADOR") {
			pdf.SetFont("Arial", "B", 8)
			pdf.SetTextColor(200, 0, 0)
			pdf.CellFormat(22, 6, "BORRADOR", "", 0, "L", false, 0, "")
			pdf.SetTextColor(0, 0, 0)
			pdf.SetFont("Arial", "I", 8)
			pdf.CellFormat(180, 6, tr(" | "+infoPie), "", 0, "L", false, 0, "")
		} else {
			pdf.SetFont("Arial", "I", 8)
			pdf.CellFormat(200, 6, tr(infoPie), "", 0, "L", false, 0, "")
		}

		// Lado derecho: Número de página en formato "Página X de Y"
		pdf.SetFont("Arial", "I", 8)
		pdf.SetX(10)
		pdf.CellFormat(277, 6, tr(fmt.Sprintf("Página %d de {nb}", pdf.PageNo())), "", 0, "R", false, 0, "")
	})

	pdf.AddPage()

	// Logo de la Municipalidad en la primera página (esquina superior izquierda)
	if rutaLogo := obtenerRutaLogo(datos.TenantLogoURL); rutaLogo != "" {
		pdf.ImageOptions(rutaLogo, 10, 8, 22, 0, false, gofpdf.ImageOptions{ReadDpi: true}, 0, "")
	}

	// Cabecera del Documento
	pdf.SetFont("Arial", "B", 14)
	pdf.CellFormat(277, 8, tr(datos.TenantNombre), "", 1, "C", false, 0, "")
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(277, 8, tr(fmt.Sprintf("REPORTE DE PLANILLA - %s", datos.PlanillaDesc)), "", 1, "C", false, 0, "")
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(277, 6, tr(fmt.Sprintf("Periodo: %02d / %d | RUC: %s", datos.PlanillaMes, datos.PlanillaAnio, datos.TenantRUC)), "", 1, "C", false, 0, "")
	pdf.Ln(5)

	for _, b := range datos.Boletas {
		// --- 1. CÁLCULO DE ALTURA UNIFORME ---
		// Determinamos cuántas líneas de conceptos tiene el grupo más largo
		maxLineas := len(b.Ingresos)
		if len(b.Retenciones) > maxLineas {
			maxLineas = len(b.Retenciones)
		}
		if len(b.Aportes) > maxLineas {
			maxLineas = len(b.Aportes)
		}

		altFilaConcepto := 4.0
		altEncabezadoCol := 5.0
		altTotalCol := 5.0
		// Altura total de las cajas de datos (Cabecera + filas + totales)
		alturaCajaDatos := altEncabezadoCol + (float64(maxLineas) * altFilaConcepto) + altTotalCol

		// Control de Salto de Página (Si la cabecera de 2 filas + la caja no entran)
		if pdf.GetY()+alturaCajaDatos+15 > 190 {
			pdf.AddPage()
		}

		// --- 2. CABECERA DEL TRABAJADOR (2 FILAS) ---
		pdf.SetFillColor(235, 235, 235)
		pdf.SetFont("Arial", "B", 9)

		// Fila 1: Trabajador
		pdf.CellFormat(277, 6, tr(fmt.Sprintf(" TRABAJADOR: %s (DNI: %s)", b.TrabajadorNombre, b.TrabajadorDoc)), "LTR", 1, "L", true, 0, "")
		// Fila 2: Cargo y Régimen
		pdf.CellFormat(277, 6, tr(fmt.Sprintf(" CARGO: %s | RÉGIMEN: %s", b.Cargo, b.Regimen)), "LBR", 1, "L", true, 0, "")

		yInicial := pdf.GetY()

		// --- 3. DIBUJO DE COLUMNAS CON ALTURA FIJA ---

		// Columna 1: INGRESOS
		pdf.SetXY(10, yInicial)
		pdf.SetFont("Arial", "B", 8)
		pdf.CellFormat(95, altEncabezadoCol, "INGRESOS", "1", 2, "C", false, 0, "")
		pdf.SetFont("Arial", "", 8)
		for _, c := range b.Ingresos {
			pdf.CellFormat(75, altFilaConcepto, tr(c.Nombre), "L", 0, "L", false, 0, "")
			pdf.CellFormat(20, altFilaConcepto, formatearMonto(c.Monto), "R", 2, "R", false, 0, "")
			pdf.SetX(10)
		}
		// Rellenar espacio vacío para mantener el borde uniforme
		lineasRestantes := maxLineas - len(b.Ingresos)
		for i := 0; i < lineasRestantes; i++ {
			pdf.CellFormat(95, altFilaConcepto, "", "LR", 2, "", false, 0, "")
			pdf.SetX(10)
		}
		pdf.SetFont("Arial", "B", 8)
		pdf.CellFormat(75, altTotalCol, "Total Ingresos:", "LB", 0, "L", false, 0, "")
		pdf.CellFormat(20, altTotalCol, formatearMonto(b.TotalIngresos), "RB", 2, "R", false, 0, "")

		// Columna 2: RETENCIONES
		pdf.SetXY(105, yInicial)
		pdf.SetFont("Arial", "B", 8)
		pdf.CellFormat(70, altEncabezadoCol, "RETENCIONES / DESCUENTOS", "1", 2, "C", false, 0, "")
		pdf.SetFont("Arial", "", 8)
		for _, c := range b.Retenciones {
			pdf.CellFormat(50, altFilaConcepto, tr(c.Nombre), "L", 0, "L", false, 0, "")
			pdf.CellFormat(20, altFilaConcepto, formatearMonto(c.Monto), "R", 2, "R", false, 0, "")
			pdf.SetX(105)
		}
		lineasRestantes = maxLineas - len(b.Retenciones)
		for i := 0; i < lineasRestantes; i++ {
			pdf.CellFormat(70, altFilaConcepto, "", "LR", 2, "", false, 0, "")
			pdf.SetX(105)
		}
		pdf.SetFont("Arial", "B", 8)
		pdf.CellFormat(50, altTotalCol, "Total Retenciones:", "LB", 0, "L", false, 0, "")
		pdf.CellFormat(20, altTotalCol, formatearMonto(b.TotalRetenciones), "RB", 2, "R", false, 0, "")

		// Columna 3: APORTES
		pdf.SetXY(175, yInicial)
		pdf.SetFont("Arial", "B", 8)
		pdf.CellFormat(60, altEncabezadoCol, "APORTES ENTIDAD", "1", 2, "C", false, 0, "")
		pdf.SetFont("Arial", "", 8)
		for _, c := range b.Aportes {
			pdf.CellFormat(40, altFilaConcepto, tr(c.Nombre), "L", 0, "L", false, 0, "")
			pdf.CellFormat(20, altFilaConcepto, formatearMonto(c.Monto), "R", 2, "R", false, 0, "")
			pdf.SetX(175)
		}
		lineasRestantes = maxLineas - len(b.Aportes)
		for i := 0; i < lineasRestantes; i++ {
			pdf.CellFormat(60, altFilaConcepto, "", "LR", 2, "", false, 0, "")
			pdf.SetX(175)
		}
		pdf.SetFont("Arial", "B", 8)
		pdf.CellFormat(40, altTotalCol, "Total Aportes:", "LB", 0, "L", false, 0, "")
		pdf.CellFormat(20, altTotalCol, formatearMonto(b.TotalAportes), "RB", 2, "R", false, 0, "")

		// --- 4. COLUMNA NETO A PAGAR (ALINEACIÓN VERTICAL) ---
		pdf.SetXY(235, yInicial)
		pdf.SetFont("Arial", "B", 10)
		// Pintamos el título en la parte superior
		pdf.CellFormat(52, 7, "NETO A PAGAR", "LTR", 2, "C", false, 0, "")

		// Calculamos espacio restante para centrar el monto grande
		espacioParaMonto := alturaCajaDatos - 7 // Altura total menos el título
		pdf.SetFont("Arial", "B", 14)
		pdf.CellFormat(52, espacioParaMonto, fmt.Sprintf("S/ %s", formatearMonto(b.NetoPagar)), "LBR", 1, "C", false, 0, "")

		// Espacio de separación para el siguiente trabajador
		pdf.SetXY(10, yInicial+alturaCajaDatos+2)
	}

	// Totales Finales del Reporte
	if pdf.GetY() > 135 {
		pdf.AddPage()
	}
	pdf.Ln(5)
	pdf.SetFont("Arial", "B", 10)
	pdf.SetFillColor(200, 200, 200)
	pdf.CellFormat(277, 8, fmt.Sprintf("RESUMEN TOTAL PLANILLA:   INGRESOS: S/ %s   |   RETENCIONES: S/ %s   |   APORTES: S/ %s   |   NETO TOTAL: S/ %s",
		formatearMonto(datos.TotalIngresos),
		formatearMonto(datos.TotalRetenciones),
		formatearMonto(datos.TotalAportes),
		formatearMonto(datos.TotalNeto)), "1", 1, "C", true, 0, "")

	// 1. Acumuladores para los conceptos de ONP, Renta 4ta y Renta 5ta
	var totalONP, totalRenta4, totalRenta5 float64
	for _, b := range datos.Boletas {
		for _, c := range b.Retenciones {
			nombreUpper := strings.ToUpper(strings.TrimSpace(c.Nombre))
			if strings.Contains(nombreUpper, "ONP") || strings.Contains(nombreUpper, "SNP") || strings.Contains(nombreUpper, "19990") {
				totalONP += c.Monto
			} else if strings.Contains(nombreUpper, "CUARTA") || strings.Contains(nombreUpper, "4TA") {
				totalRenta4 += c.Monto
			} else if strings.Contains(nombreUpper, "QUINTA") || strings.Contains(nombreUpper, "5TA") {
				totalRenta5 += c.Monto
			}
		}
	}

	// 2. Cálculos según fórmulas del requerimiento
	costoPlanilla := datos.TotalIngresos + datos.TotalAportes

	ajusteONP := math.Round(totalONP) - totalONP
	ajusteRenta4 := math.Round(totalRenta4) - totalRenta4
	ajusteRenta5 := math.Round(totalRenta5) - totalRenta5

	if math.Abs(ajusteONP) < 0.0001 {
		ajusteONP = 0.0
	}
	if math.Abs(ajusteRenta4) < 0.0001 {
		ajusteRenta4 = 0.0
	}
	if math.Abs(ajusteRenta5) < 0.0001 {
		ajusteRenta5 = 0.0
	}

	costoTotal := costoPlanilla + ajusteONP + ajusteRenta4 + ajusteRenta5

	// 3. Dibujo de la Tabla Resumen (Cuadro Centrado)
	pdf.Ln(4)
	xCuadro := 58.5
	anchos := []float64{140.0, 40.0}

	pdf.SetX(xCuadro)
	pdf.SetFont("Arial", "B", 9)
	pdf.SetFillColor(230, 230, 230)
	pdf.CellFormat(anchos[0], 6, tr("Detalle"), "1", 0, "L", true, 0, "")
	pdf.CellFormat(anchos[1], 6, tr("Monto S/"), "1", 1, "R", true, 0, "")

	// Fila A
	pdf.SetX(xCuadro)
	pdf.SetFont("Arial", "", 8.5)
	textoA := fmt.Sprintf("A. Costo de la planilla (%02d/%d)", datos.PlanillaMes, datos.PlanillaAnio)
	pdf.CellFormat(anchos[0], 5.5, tr(textoA), "1", 0, "L", false, 0, "")
	pdf.CellFormat(anchos[1], 5.5, formatearMonto(costoPlanilla), "1", 1, "R", false, 0, "")

	// Fila B
	pdf.SetX(xCuadro)
	pdf.CellFormat(anchos[0], 5.5, tr("B. Ajuste redondeo ONP"), "1", 0, "L", false, 0, "")
	pdf.CellFormat(anchos[1], 5.5, formatearMonto(ajusteONP), "1", 1, "R", false, 0, "")

	// Fila C
	pdf.SetX(xCuadro)
	pdf.CellFormat(anchos[0], 5.5, tr("C. Ajuste redondeo Renta de 4ta Categoría"), "1", 0, "L", false, 0, "")
	pdf.CellFormat(anchos[1], 5.5, formatearMonto(ajusteRenta4), "1", 1, "R", false, 0, "")

	// Fila D
	pdf.SetX(xCuadro)
	pdf.CellFormat(anchos[0], 5.5, tr("D. Ajuste redondeo Renta de 5ta Categoría"), "1", 0, "L", false, 0, "")
	pdf.CellFormat(anchos[1], 5.5, formatearMonto(ajusteRenta5), "1", 1, "R", false, 0, "")

	// Fila E
	pdf.SetX(xCuadro)
	pdf.SetFont("Arial", "B", 9)
	pdf.SetFillColor(240, 240, 240)
	pdf.CellFormat(anchos[0], 6, tr("E. Costo total de la planilla (A+B+C+D)"), "1", 0, "L", true, 0, "")
	pdf.CellFormat(anchos[1], 6, formatearMonto(costoTotal), "1", 1, "R", true, 0, "")

	var buf bytes.Buffer
	err := pdf.Output(&buf)
	return buf.Bytes(), err
}

// GenerarBoletasPDF crea un archivo con todas las boletas (2 por página)
func (s *PdfService) GenerarBoletasPDF(datos *models.DatosReportePlanilla) ([]byte, error) {
	// "P" para Portrait (Vertical)
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(10, 10, 10)
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	if len(datos.Boletas) == 0 {
		pdf.AddPage()
		pdf.SetFont("Arial", "I", 10)
		pdf.CellFormat(190, 10, tr("No se encontraron boletas de pago para esta planilla."), "", 1, "C", false, 0, "")
	} else {
		boletasPorPagina := 0

		for _, b := range datos.Boletas {
			if boletasPorPagina == 0 {
				pdf.AddPage()
				// Dibujamos la primera boleta arriba (Y = 10)
				s.dibujarBoleta(pdf, tr, b, datos, 10)
				boletasPorPagina++
			} else {
				// Dibujamos la línea de corte punteada a la mitad de la hoja (Y = 148.5)
				pdf.SetDrawColor(150, 150, 150)
				pdf.SetDashPattern([]float64{2, 2}, 0)
				pdf.Line(10, 148.5, 200, 148.5)
				pdf.SetDashPattern([]float64{}, 0) // Restaurar línea sólida
				pdf.SetDrawColor(0, 0, 0)

				// Dibujamos la segunda boleta abajo (Y = 155)
				s.dibujarBoleta(pdf, tr, b, datos, 155)
				boletasPorPagina = 0 // Reiniciamos para la siguiente página
			}
		}
	}

	var buf bytes.Buffer
	err := pdf.Output(&buf)
	return buf.Bytes(), err
}

// dibujarBoleta es el "sello" que estampa una boleta en una coordenada Y específica
func (s *PdfService) dibujarBoleta(pdf *gofpdf.Fpdf, tr func(string) string, b *models.BoletaReporte, datos *models.DatosReportePlanilla, startY float64) {
	pdf.SetXY(10, startY)

	// 1. Cabecera de la Boleta (2 Columnas: 70mm y 120mm)
	// Ajuste dinámico de fuente para prevenir que un nombre largo de municipalidad desconfigure la columna 1
	fontSizeMuni := 10.0
	pdf.SetFont("Arial", "B", fontSizeMuni)
	for fontSizeMuni > 6.5 && pdf.GetStringWidth(tr(datos.TenantNombre)) > 78.0 {
		fontSizeMuni -= 0.5
		pdf.SetFont("Arial", "B", fontSizeMuni)
	}

	// Fila 1: Nombre de la Municipalidad (70mm) | BOLETA DE PAGO - DESCRIPCIÓN (120mm)
	pdf.CellFormat(80, 5, tr(datos.TenantNombre), "", 0, "L", false, 0, "")
	pdf.SetFont("Arial", "B", 9.5)
	pdf.CellFormat(110, 5, tr(fmt.Sprintf("BOLETA DE PAGO - %s", strings.ToUpper(datos.PlanillaDesc))), "", 1, "R", false, 0, "")

	// Fila 2: RUC (70mm) | Periodo y Año (120mm)
	pdf.SetFont("Arial", "", 8.5)
	pdf.CellFormat(80, 4, tr(fmt.Sprintf("RUC: %s", datos.TenantRUC)), "", 0, "L", false, 0, "")
	pdf.CellFormat(110, 4, tr(fmt.Sprintf("PERIODO: %02d / %d", datos.PlanillaMes, datos.PlanillaAnio)), "", 1, "R", false, 0, "")
	pdf.Ln(2)

	// 2. Datos del Trabajador (Normas vigentes Perú)
	pdf.SetFillColor(245, 245, 245)
	pdf.SetFont("Arial", "B", 7.5)

	// Sistema Previsional
	sistPrevisional := b.RegimenPensionario
	if strings.EqualFold(b.RegimenPensionario, "AFP") && b.AfpNombre != "" && b.AfpNombre != "-" {
		sistPrevisional = fmt.Sprintf("AFP (%s)", b.AfpNombre)
	}

	// Fila 1: Trabajador, DNI y sistema de pensiones
	pdf.CellFormat(100, 4.5, tr(fmt.Sprintf(" TRABAJADOR: %s", b.TrabajadorNombre)), "LT", 0, "L", true, 0, "")
	pdf.CellFormat(35, 4.5, tr(fmt.Sprintf(" DNI: %s ", b.TrabajadorDoc)), "T", 0, "R", true, 0, "")
	pdf.CellFormat(55, 4.5, tr(fmt.Sprintf(" SIST. PENSIONES: %s", sistPrevisional)), "TR", 1, "L", true, 0, "")

	// Fila 2: Domicilio, sexo y CUSPP
	dirTxt := b.Direccion
	if strings.TrimSpace(dirTxt) == "" {
		dirTxt = "-"
	}
	pdf.CellFormat(100, 4.5, tr(fmt.Sprintf(" DOMICILIO: %s", dirTxt)), "L", 0, "L", true, 0, "")
	pdf.CellFormat(35, 4.5, tr(fmt.Sprintf(" SEXO: %s ", b.Sexo)), "", 0, "R", true, 0, "")
	pdf.CellFormat(55, 4.5, tr(fmt.Sprintf(" CUSPP: %s", b.Cuspp)), "R", 1, "L", true, 0, "")

	// Fila 3: Cargo, fecha de nacimiento y fecha de ingreso
	pdf.CellFormat(100, 4.5, tr(fmt.Sprintf(" PUESTO: %s", b.Cargo)), "L", 0, "L", true, 0, "")
	pdf.CellFormat(35, 4.5, tr(fmt.Sprintf(" F. NAC.: %s ", b.FechaNacimiento)), "", 0, "R", true, 0, "")
	pdf.CellFormat(55, 4.5, tr(fmt.Sprintf(" F. INGRESO: %s ", b.FechaIngreso)), "R", 1, "L", true, 0, "")

	// Fila 4: Régimen laboral y F. Cese
	ceseTxt := b.FechaCese
	if strings.TrimSpace(ceseTxt) == "" {
		ceseTxt = "-"
	}
	pdf.CellFormat(135, 4.5, tr(fmt.Sprintf(" RÉGIMEN LAB.: %s", b.Regimen)), "LB", 0, "L", true, 0, "")
	pdf.CellFormat(55, 4.5, tr(fmt.Sprintf(" F. CESE: %s", ceseTxt)), "BR", 1, "L", true, 0, "")
	pdf.Ln(2)

	// 3. Tablas de Conceptos (2 Columnas: Ingresos vs Retenciones)
	yTablas := pdf.GetY()

	// Calculamos cuál columna tiene más líneas para igualar alturas
	maxLineas := len(b.Ingresos)
	if len(b.Retenciones) > maxLineas {
		maxLineas = len(b.Retenciones)
	}

	altFila := 4.0
	anchoCol := 95.0
	anchoMonto := 20.0
	anchoTexto := anchoCol - anchoMonto

	// Columna Izquierda: INGRESOS
	pdf.SetXY(10, yTablas)
	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(anchoCol, 5, "INGRESOS", "1", 2, "C", false, 0, "")
	pdf.SetFont("Arial", "", 8)
	for _, c := range b.Ingresos {
		pdf.CellFormat(anchoTexto, altFila, tr(c.Nombre), "L", 0, "L", false, 0, "")
		pdf.CellFormat(anchoMonto, altFila, formatearMonto(c.Monto), "R", 2, "R", false, 0, "")
		pdf.SetX(10)
	}
	for i := 0; i < (maxLineas - len(b.Ingresos)); i++ {
		pdf.CellFormat(anchoCol, altFila, "", "LR", 2, "", false, 0, "")
		pdf.SetX(10)
	}
	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(anchoTexto, 5, "Total Ingresos:", "LB", 0, "L", false, 0, "")
	pdf.CellFormat(anchoMonto, 5, formatearMonto(b.TotalIngresos), "RB", 2, "R", false, 0, "")

	// Columna Derecha: RETENCIONES
	pdf.SetXY(105, yTablas)
	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(anchoCol, 5, "RETENCIONES Y DESCUENTOS", "1", 2, "C", false, 0, "")
	pdf.SetFont("Arial", "", 8)
	for _, c := range b.Retenciones {
		pdf.CellFormat(anchoTexto, altFila, tr(c.Nombre), "L", 0, "L", false, 0, "")
		pdf.CellFormat(anchoMonto, altFila, formatearMonto(c.Monto), "R", 2, "R", false, 0, "")
		pdf.SetX(105)
	}
	for i := 0; i < (maxLineas - len(b.Retenciones)); i++ {
		pdf.CellFormat(anchoCol, altFila, "", "LR", 2, "", false, 0, "")
		pdf.SetX(105)
	}
	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(anchoTexto, 5, "Total Retenciones:", "LB", 0, "L", false, 0, "")
	pdf.CellFormat(anchoMonto, 5, formatearMonto(b.TotalRetenciones), "RB", 2, "R", false, 0, "")

	// 4. Pie de Boleta: Neto a Pagar y Aportes
	yPie := pdf.GetY()

	// Aportes (Izquierda)
	pdf.SetXY(10, yPie+2)
	pdf.SetFont("Arial", "B", 7)
	strAportes := "Aportes del Empleador: "
	for _, c := range b.Aportes {
		strAportes += fmt.Sprintf("%s (S/ %s)   ", c.Nombre, formatearMonto(c.Monto))
	}
	pdf.CellFormat(120, 8, tr(strAportes), "1", 0, "L", false, 0, "")

	// Neto a Pagar (Derecha)
	pdf.SetXY(135, yPie+2)
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(35, 8, "NETO A PAGAR:", "LTB", 0, "R", true, 0, "")
	pdf.SetFont("Arial", "B", 11)
	pdf.CellFormat(30, 8, fmt.Sprintf("S/ %s", formatearMonto(b.NetoPagar)), "RTB", 0, "R", true, 0, "")

	// Firmas
	pdf.SetXY(10, yPie+30)
	pdf.SetFont("Arial", "", 8)
	pdf.CellFormat(95, 4, "_________________________________", "", 0, "C", false, 0, "")
	pdf.CellFormat(95, 4, "_________________________________", "", 1, "C", false, 0, "")
	pdf.CellFormat(95, 4, "Firma del Empleador", "", 0, "C", false, 0, "")
	pdf.CellFormat(95, 4, tr("Firma del Trabajador"), "", 1, "C", false, 0, "")
}

// GenerarReporteLiquidacionPDF crea el documento oficial de liquidación de beneficios en PDF A4 vertical
func (s *PdfService) GenerarReporteLiquidacionPDF(datos *models.DatosReporteLiquidacion) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(10, 10, 10)
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	pdf.AddPage()

	// 1. Logotipo de la Municipalidad/Tenant (si está disponible)
	if rutaLogo := obtenerRutaLogo(datos.TenantLogoURL); rutaLogo != "" {
		pdf.ImageOptions(rutaLogo, 10, 8, 24, 0, false, gofpdf.ImageOptions{ReadDpi: true}, 0, "")
	}

	// 2. Encabezado Institucional y Título Principal
	pdf.SetFillColor(0, 112, 192) // Azul corporativo #0070C0
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Arial", "B", 14)
	pdf.CellFormat(190, 9, tr("LIQUIDACIÓN DE BENEFICIOS SOCIALES"), "1", 1, "C", true, 0, "")

	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(190, 5, tr(datos.TenantNombre), "", 1, "C", false, 0, "")
	pdf.SetFont("Arial", "", 8.5)
	pdf.CellFormat(190, 4, tr(fmt.Sprintf("RUC: %s", datos.TenantRUC)), "", 1, "C", false, 0, "")
	pdf.Ln(3)

	// 3. Bloque de Datos del Colaborador
	pdf.SetFillColor(240, 244, 248)
	pdf.SetFont("Arial", "B", 8)

	l := datos.Liquidacion
	fechaIngresoStr := datos.FechaIngreso.Format("02/01/2006")
	fechaCeseStr := l.FechaCese.Format("02/01/2006")
	tiempoServicioStr := fmt.Sprintf("%d años, %d meses, %d días", l.AnosServicios, l.MesesServicios, l.DiasServicios)

	pdf.CellFormat(40, 5, tr("  TRABAJADOR:"), "LT", 0, "L", true, 0, "")
	pdf.SetFont("Arial", "", 8)
	pdf.CellFormat(150, 5, tr(fmt.Sprintf("%s (DNI: %s)", l.TrabajadorNombre, l.TrabajadorDocumento)), "TR", 1, "L", true, 0, "")

	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(40, 5, tr("  CARGO / PUESTO:"), "L", 0, "L", true, 0, "")
	pdf.SetFont("Arial", "", 8)
	pdf.CellFormat(150, 5, tr(fmt.Sprintf("%s | RÉGIMEN: D.L. %s", l.PuestoNombre, l.Regimen)), "R", 1, "L", true, 0, "")

	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(40, 5, tr("  FECHA INGRESO:"), "L", 0, "L", true, 0, "")
	pdf.SetFont("Arial", "", 8)
	pdf.CellFormat(55, 5, fechaIngresoStr, "", 0, "L", true, 0, "")
	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(40, 5, tr("FECHA CESE:"), "", 0, "L", true, 0, "")
	pdf.SetFont("Arial", "", 8)
	pdf.CellFormat(55, 5, fechaCeseStr, "R", 1, "L", true, 0, "")

	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(40, 5, tr("  MOTIVO CESE:"), "LB", 0, "L", true, 0, "")
	pdf.SetFont("Arial", "", 8)
	pdf.CellFormat(55, 5, tr(l.Motivo), "B", 0, "L", true, 0, "")
	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(40, 5, tr("TIEMPO LABORADO:"), "B", 0, "L", true, 0, "")
	pdf.SetFont("Arial", "", 8)
	pdf.CellFormat(55, 5, tr(tiempoServicioStr), "RB", 1, "L", true, 0, "")

	pdf.Ln(4)

	// Helper para secciones
	dibujarEncabezadoSeccion := func(num string, titulo string) {
		pdf.SetFillColor(0, 112, 192)
		pdf.SetTextColor(255, 255, 255)
		pdf.SetFont("Arial", "B", 8.5)
		pdf.CellFormat(150, 5, tr(fmt.Sprintf(" %s.- %s", num, titulo)), "1", 0, "L", true, 0, "")
		pdf.CellFormat(40, 5, "MONTO S/", "1", 1, "C", true, 0, "")
		pdf.SetTextColor(0, 0, 0)
	}

	// 1. Remuneración Computable
	dibujarEncabezadoSeccion("1", "REMUNERACIÓN COMPUTABLE")
	pdf.SetFont("Arial", "", 8)
	pdf.CellFormat(150, 4.5, tr("   • Sueldo Básico Presupuestado"), "L", 0, "L", false, 0, "")
	pdf.CellFormat(40, 4.5, formatearMonto(datos.SueldoBasico), "R", 1, "R", false, 0, "")

	if datos.AsignacionFamiliar > 0 {
		pdf.CellFormat(150, 4.5, tr("   • Asignación Familiar"), "L", 0, "L", false, 0, "")
		pdf.CellFormat(40, 4.5, formatearMonto(datos.AsignacionFamiliar), "R", 1, "R", false, 0, "")
	}
	if datos.PromedioGratificacion > 0 {
		pdf.CellFormat(150, 4.5, tr("   • Promedio Gratificación (1/6)"), "L", 0, "L", false, 0, "")
		pdf.CellFormat(40, 4.5, formatearMonto(datos.PromedioGratificacion), "R", 1, "R", false, 0, "")
	}

	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(150, 5, tr("   TOTAL REMUNERACIÓN COMPUTABLE"), "LBT", 0, "L", false, 0, "")
	pdf.CellFormat(40, 5, formatearMonto(datos.RemuneracionComputable), "RBT", 1, "R", false, 0, "")
	pdf.Ln(3)

	// 2. Cálculo de CTS
	dibujarEncabezadoSeccion("2", "CÁLCULO DE COMPENSACIÓN POR TIEMPO DE SERVICIOS (CTS)")
	pdf.SetFont("Arial", "I", 7.5)
	pdf.CellFormat(190, 4, tr(fmt.Sprintf("   Periodo computable: %s al %s", datos.CtsPeriodoInicio, datos.CtsPeriodoFin)), "LR", 1, "L", false, 0, "")

	pdf.SetFont("Arial", "", 8)
	labelMesesCts := fmt.Sprintf("   • Por %d mes(es) computables (Rem. / 12 x %d)", datos.CtsMeses, datos.CtsMeses)
	pdf.CellFormat(150, 4.5, tr(labelMesesCts), "L", 0, "L", false, 0, "")
	pdf.CellFormat(40, 4.5, formatearMonto(datos.MontoCtsMeses), "R", 1, "R", false, 0, "")

	if datos.CtsDias > 0 {
		labelDiasCts := fmt.Sprintf("   • Por %d día(s) computables (Rem. / 12 / 30 x %d)", datos.CtsDias, datos.CtsDias)
		pdf.CellFormat(150, 4.5, tr(labelDiasCts), "L", 0, "L", false, 0, "")
		pdf.CellFormat(40, 4.5, formatearMonto(datos.MontoCtsDias), "R", 1, "R", false, 0, "")
	}

	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(150, 5, tr("   CTS A PAGAR"), "LBT", 0, "L", false, 0, "")
	pdf.CellFormat(40, 5, formatearMonto(l.MontoCts), "RBT", 1, "R", false, 0, "")
	pdf.Ln(3)

	// 3. Vacaciones Truncas y No Gozadas
	dibujarEncabezadoSeccion("3", "VACACIONES TRUNCAS Y PENDIENTES DE GOCE")
	pdf.SetFont("Arial", "", 8)

	labelMesesVac := fmt.Sprintf("   • Vacaciones truncas por %d mes(es)", datos.VacacionesMeses)
	pdf.CellFormat(150, 4.5, tr(labelMesesVac), "L", 0, "L", false, 0, "")
	pdf.CellFormat(40, 4.5, formatearMonto(l.MontoVacacionesTruncas), "R", 1, "R", false, 0, "")

	if l.MontoVacacionesNoGozadas > 0 {
		pdf.CellFormat(150, 4.5, tr("   • Vacaciones no gozadas (Periodos acumulados)"), "L", 0, "L", false, 0, "")
		pdf.CellFormat(40, 4.5, formatearMonto(l.MontoVacacionesNoGozadas), "R", 1, "R", false, 0, "")
	}
	if l.MontoIndemnizacionVacacional > 0 {
		pdf.CellFormat(150, 4.5, tr("   • Indemnización vacacional (DL 728)"), "L", 0, "L", false, 0, "")
		pdf.CellFormat(40, 4.5, formatearMonto(l.MontoIndemnizacionVacacional), "R", 1, "R", false, 0, "")
	}

	pdf.SetFont("Arial", "I", 7.5)
	labelDescuento := fmt.Sprintf("   • Descuento previsional (%s sobre vacaciones brutas)", datos.DescuentoPensionNombre)
	pdf.CellFormat(150, 4.5, tr(labelDescuento), "L", 0, "L", false, 0, "")
	pdf.CellFormat(40, 4.5, fmt.Sprintf("(%s)", formatearMonto(datos.MontoDescuentoPension)), "R", 1, "R", false, 0, "")

	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(150, 5, tr("   VACACIONES NETAS A PAGAR"), "LBT", 0, "L", false, 0, "")
	pdf.CellFormat(40, 5, formatearMonto(datos.VacacionesNetas), "RBT", 1, "R", false, 0, "")
	pdf.Ln(3)

	// 4. Gratificaciones Truncas y Bonificación Especial
	dibujarEncabezadoSeccion("4", "GRATIFICACIONES TRUNCAS Y BONIFICACIÓN EXTRAORDINARIA")
	pdf.SetFont("Arial", "I", 7.5)
	pdf.CellFormat(190, 4, tr(fmt.Sprintf("   Semestre: %s (%s al %s)", datos.GratiSemestreTipo, datos.GratiPeriodoInicio, datos.GratiPeriodoFin)), "LR", 1, "L", false, 0, "")

	pdf.SetFont("Arial", "", 8)
	labelGrati := fmt.Sprintf("   • Gratificación trunca (%d mes/es, %d día/s)", datos.GratiMeses, datos.GratiDias)
	pdf.CellFormat(150, 4.5, tr(labelGrati), "L", 0, "L", false, 0, "")
	pdf.CellFormat(40, 4.5, formatearMonto(l.MontoGratiTrunca), "R", 1, "R", false, 0, "")

	if datos.BonificacionEspecial > 0 {
		pdf.CellFormat(150, 4.5, tr("   • Bonificación extraordinaria (9% Ley 29351/30334)"), "L", 0, "L", false, 0, "")
		pdf.CellFormat(40, 4.5, formatearMonto(datos.BonificacionEspecial), "R", 1, "R", false, 0, "")
	}

	montoGratiTotal := l.MontoGratiTrunca + datos.BonificacionEspecial
	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(150, 5, tr("   GRATIFICACIÓN Y BONIFICACIÓN A PAGAR"), "LBT", 0, "L", false, 0, "")
	pdf.CellFormat(40, 5, formatearMonto(montoGratiTotal), "RBT", 1, "R", false, 0, "")
	pdf.Ln(4)

	// 5. Total General de la Liquidación
	pdf.SetFillColor(0, 32, 96) // Azul Oscuro #002060
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(150, 7, tr("   TOTAL NETO A RECIBIR POR LIQUIDACIÓN DE BENEFICIOS"), "1", 0, "L", true, 0, "")
	pdf.SetFont("Arial", "B", 11)
	pdf.CellFormat(40, 7, fmt.Sprintf("S/ %s", formatearMonto(datos.TotalLiquidacion)), "1", 1, "R", true, 0, "")
	pdf.SetTextColor(0, 0, 0)

	// Monto en letras
	pdf.SetFont("Arial", "B", 8.5)
	pdf.SetTextColor(192, 0, 0)
	pdf.CellFormat(190, 6, tr(datos.MontoEnLetras), "LBR", 1, "C", false, 0, "")
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(4)

	// 6. Declaración de Conformidad y Espacio de Firmas
	pdf.SetFont("Arial", "", 7.5)
	declText := "FIRMO LA PRESENTE COMO CONSTANCIA DE HABER RECIBIDO LA INTEGRIDAD DE MI LIQUIDACIÓN DE BENEFICIOS SOCIALES DE CONFORMIDAD AL D.LEG. Nº 650 Y NO TENIENDO NADA QUE RECLAMAR."
	pdf.MultiCell(190, 3.5, tr(declText), "", "J", false)
	pdf.Ln(8)

	pdf.SetFont("Arial", "", 8)
	fechaEmisionStr := fmt.Sprintf("Fecha de emisión: %s", datos.FechaEmisionTexto)
	pdf.CellFormat(190, 4, tr(fechaEmisionStr), "", 1, "L", false, 0, "")
	pdf.Ln(18)

	// Firmas
	pdf.CellFormat(95, 4, "_________________________________", "", 0, "C", false, 0, "")
	pdf.CellFormat(95, 4, "_________________________________", "", 1, "C", false, 0, "")

	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(95, 4, tr(datos.TenantNombre), "", 0, "C", false, 0, "")
	pdf.CellFormat(95, 4, tr(l.TrabajadorNombre), "", 1, "C", false, 0, "")

	pdf.SetFont("Arial", "", 7.5)
	pdf.CellFormat(95, 4, "EMPLEADOR", "", 0, "C", false, 0, "")
	pdf.CellFormat(95, 4, tr(fmt.Sprintf("DNI: %s", l.TrabajadorDocumento)), "", 1, "C", false, 0, "")

	var buf bytes.Buffer
	err := pdf.Output(&buf)
	return buf.Bytes(), err
}

// formatearMonto convierte un float64 a string con separador de miles (coma) y 2 decimales.
// Ejemplo: 13345.34 -> 13,345.34
func formatearMonto(n float64) string {
	s := fmt.Sprintf("%.2f", n)
	partes := strings.Split(s, ".")
	entero := partes[0]
	decimal := partes[1]

	var resultado []byte
	longitud := len(entero)

	for i, char := range entero {
		// Si no es el primer dígito y faltan múltiplos de 3 para terminar, ponemos coma
		if i > 0 && (longitud-i)%3 == 0 && entero[i-1] != '-' {
			resultado = append(resultado, ',')
		}
		resultado = append(resultado, byte(char))
	}

	return string(resultado) + "." + decimal
}
