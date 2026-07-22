package services

import (
	"bytes"
	"fmt"
	"planilla-rgm/internal/models"
	"strings"

	"github.com/jung-kurt/gofpdf"
)

type PdfService struct{}

func NewPdfService() *PdfService {
	return &PdfService{}
}

// GenerarReportePlanilla crea el "Boletón" con alturas uniformes y cabecera de dos filas
func (s *PdfService) GenerarReportePlanilla(datos *models.DatosReportePlanilla) ([]byte, error) {
	pdf := gofpdf.New("L", "mm", "A4", "")
	pdf.SetMargins(10, 10, 10)
	pdf.AddPage()

	tr := pdf.UnicodeTranslatorFromDescriptor("")

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

	// Totales Finales del Reporte (Opcional, al final del documento)
	if pdf.GetY() > 170 {
		pdf.AddPage()
	}
	pdf.Ln(5)
	pdf.SetFont("Arial", "B", 10)
	pdf.SetFillColor(200, 200, 200)
	pdf.CellFormat(277, 8, fmt.Sprintf("RESUMEN TOTAL PLANILLA:   INGRESOS: S/ %s   |   RETENCIONES: S/ %s   |   NETO TOTAL: S/ %s ",
		formatearMonto(datos.TotalIngresos),
		formatearMonto(datos.TotalRetenciones),
		formatearMonto(datos.TotalNeto)), "1", 1, "C", true, 0, "")

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

	var buf bytes.Buffer
	err := pdf.Output(&buf)
	return buf.Bytes(), err
}

// dibujarBoleta es el "sello" que estampa una boleta en una coordenada Y específica
func (s *PdfService) dibujarBoleta(pdf *gofpdf.Fpdf, tr func(string) string, b *models.BoletaReporte, datos *models.DatosReportePlanilla, startY float64) {
	pdf.SetXY(10, startY)

	// 1. Cabecera de la Boleta
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(190, 6, tr(datos.TenantNombre), "", 1, "C", false, 0, "")
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(190, 5, tr(fmt.Sprintf("BOLETA DE PAGO - %s", datos.PlanillaDesc)), "", 1, "C", false, 0, "")
	pdf.SetFont("Arial", "", 9)
	pdf.CellFormat(190, 5, tr(fmt.Sprintf("Periodo: %02d / %d | RUC: %s", datos.PlanillaMes, datos.PlanillaAnio, datos.TenantRUC)), "", 1, "C", false, 0, "")
	pdf.Ln(2)

	// 2. Datos del Trabajador
	pdf.SetFillColor(240, 240, 240)
	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(190, 5, tr(fmt.Sprintf(" TRABAJADOR: %s (DNI: %s)", b.TrabajadorNombre, b.TrabajadorDoc)), "LTR", 1, "L", true, 0, "")
	pdf.CellFormat(190, 5, tr(fmt.Sprintf(" CARGO: %s | RÉGIMEN: %s", b.Cargo, b.Regimen)), "LBR", 1, "L", true, 0, "")

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
	pdf.SetXY(10, yPie+25)
	pdf.SetFont("Arial", "", 8)
	pdf.CellFormat(95, 4, "_________________________________", "", 0, "C", false, 0, "")
	pdf.CellFormat(95, 4, "_________________________________", "", 1, "C", false, 0, "")
	pdf.CellFormat(95, 4, "Firma del Empleador", "", 0, "C", false, 0, "")
	pdf.CellFormat(95, 4, tr("Firma del Trabajador"), "", 1, "C", false, 0, "")
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
