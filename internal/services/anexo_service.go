package services

import (
	"bytes"
	"fmt"
	"math"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
	"strings"

	"github.com/jung-kurt/gofpdf"
	"github.com/xuri/excelize/v2"
)

type AnexoService struct {
	AnexoRepo    *repository.AnexoRepository
	PlanillaRepo *repository.PlanillaRepository
	TenantRepo   *repository.TenantRepository
}

func NewAnexoService(anexoRepo *repository.AnexoRepository, planillaRepo *repository.PlanillaRepository, tenantRepo *repository.TenantRepository) *AnexoService {
	return &AnexoService{
		AnexoRepo:    anexoRepo,
		PlanillaRepo: planillaRepo,
		TenantRepo:   tenantRepo,
	}
}

// CalcularAjustesRedondeo calcula la diferencia entre la suma exacta y el monto redondeado SUNAT para ONP, Renta 4ta y Renta 5ta
func (s *AnexoService) CalcularAjustesRedondeo(planillaID, tenantID int) ([]models.AjusteRedondeoSunat, error) {
	if s.AnexoRepo == nil {
		return nil, nil
	}

	conceptos, err := s.AnexoRepo.ObtenerSumatoriasSunat(planillaID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("error al obtener sumatorias SUNAT: %w", err)
	}

	// Sumadores acumulados
	var totalONP, totalRenta4ta, totalRenta5ta float64
	var nombreONP, nombreRenta4ta, nombreRenta5ta string

	for _, c := range conceptos {
		cod := strings.TrimSpace(c.CodigoSunat)
		nom := strings.ToUpper(strings.TrimSpace(c.NombreEnBoleta))

		// Identificar ONP (0607 o por nombre)
		if cod == "0607" || strings.Contains(nom, "ONP") || strings.Contains(nom, "SNP") || strings.Contains(nom, "19990") {
			totalONP += c.MontoTotal
			if nombreONP == "" {
				nombreONP = c.NombreEnBoleta
			}
		} else if cod == "S101" || strings.Contains(nom, "CUARTA") || strings.Contains(nom, "4TA") {
			totalRenta4ta += c.MontoTotal
			if nombreRenta4ta == "" {
				nombreRenta4ta = c.NombreEnBoleta
			}
		} else if cod == "0605" || strings.Contains(nom, "QUINTA") || strings.Contains(nom, "5TA") {
			totalRenta5ta += c.MontoTotal
			if nombreRenta5ta == "" {
				nombreRenta5ta = c.NombreEnBoleta
			}
		}
	}

	// Imprimir los totales en la terminal
	fmt.Println("Total ONP: ", totalONP)
	fmt.Println("Total Renta 4ta: ", totalRenta4ta)
	fmt.Println("Total Renta 5ta: ", totalRenta5ta)

	var resultados []models.AjusteRedondeoSunat

	// Helper para procesar cada concepto tributario
	procesarAjuste := func(clave, nombreDefecto, nombreCapturado string, montoExacto float64, codigos []string, palabraClave string) {
		if montoExacto <= 0 {
			return
		}

		montoRedondeado := math.Round(montoExacto)
		diferencia := math.Round((montoRedondeado-montoExacto)*100) / 100

		nombreFinal := nombreDefecto
		if nombreCapturado != "" {
			nombreFinal = nombreCapturado
		}

		ajuste := models.AjusteRedondeoSunat{
			ConceptoClave:   clave,
			NombreConcepto:  nombreFinal,
			MontoExacto:     montoExacto,
			MontoRedondeado: montoRedondeado,
			Diferencia:      diferencia,
		}

		// Si existe diferencia de redondeo, buscar la (Meta, Clasificador) target
		if math.Abs(diferencia) > 0.001 {
			meta, clasif, err := s.AnexoRepo.ObtenerTargetCompromisoAjuste(planillaID, tenantID, codigos, palabraClave)
			if err == nil {
				ajuste.MetaCodigoTarget = meta
				ajuste.ClasificadorTarget = clasif
			}
		}

		resultados = append(resultados, ajuste)
	}

	procesarAjuste("ONP", "SNP DL 19990 - ONP", nombreONP, totalONP, []string{"0607"}, "ONP")
	procesarAjuste("RENTA_4TA", "Renta de Cuarta Categoría - Retenciones", nombreRenta4ta, totalRenta4ta, []string{"S101"}, "CUARTA")
	procesarAjuste("RENTA_5TA", "Renta de Quinta Categoría - Retenciones", nombreRenta5ta, totalRenta5ta, []string{"0605"}, "QUINTA")

	return resultados, nil
}

// ObtenerDatosAnexo1 prepara los datos consolidados del Anexo 1 con filtro y ajustes por redondeo SUNAT aplicados
func (s *AnexoService) ObtenerDatosAnexo1(planillaID, tenantID int) (*models.DatosAnexo1, error) {
	planilla, err := s.PlanillaRepo.ObtenerPorID(planillaID, tenantID)
	if err != nil || planilla == nil {
		return nil, fmt.Errorf("planilla no encontrada")
	}

	tenantNombre := "ENTIDAD / INSTITUCIÓN"
	tenantRUC := ""
	if s.TenantRepo != nil {
		tenant, err := s.TenantRepo.ObtenerPorID(tenantID)
		if err == nil && tenant != nil {
			tenantNombre = tenant.Nombre
			tenantRUC = tenant.Ruc
		}
	}

	// 1. Obtener items filtrados (sin RETENCIONES)
	items, err := s.AnexoRepo.ObtenerCompromisoPresupuestal(planillaID, tenantID)
	if err != nil {
		return nil, err
	}

	// 2. Calcular ajustes por redondeo SUNAT
	ajustes, err := s.CalcularAjustesRedondeo(planillaID, tenantID)
	if err != nil {
		ajustes = nil // Si hay error en redondeo no bloquea la consulta principal
	}

	// 3. Aplicar diferencias de redondeo a las metas/clasificadores target
	for _, ajuste := range ajustes {
		if math.Abs(ajuste.Diferencia) > 0.001 && ajuste.MetaCodigoTarget != "" && ajuste.ClasificadorTarget != "" {
			for i := range items {
				if items[i].MetaCodigo == ajuste.MetaCodigoTarget && items[i].ClasificadorCodigo == ajuste.ClasificadorTarget {
					if items[i].MontoTotal+ajuste.Diferencia >= 0 {
						items[i].MontoTotal = math.Round((items[i].MontoTotal+ajuste.Diferencia)*100) / 100
					}
					break
				}
			}
		}
	}

	// 4. Agrupar por Meta Presupuestal y calcular subtotales
	mapaMetas := make(map[string]*models.ResumenMetaCompromiso)
	var ordenMetas []string
	var montoTotal float64

	for _, item := range items {
		montoTotal += item.MontoTotal

		resumen, existe := mapaMetas[item.MetaCodigo]
		if !existe {
			resumen = &models.ResumenMetaCompromiso{
				MetaCodigo:      item.MetaCodigo,
				MetaDescripcion: item.MetaDescripcion,
				Items:           []models.ItemCompromisoPresupuestal{},
				TotalMeta:       0,
			}
			mapaMetas[item.MetaCodigo] = resumen
			ordenMetas = append(ordenMetas, item.MetaCodigo)
		}
		resumen.Items = append(resumen.Items, item)
		resumen.TotalMeta += item.MontoTotal
	}

	var resumenMetas []models.ResumenMetaCompromiso
	for _, codigo := range ordenMetas {
		resumenMetas = append(resumenMetas, *mapaMetas[codigo])
	}

	return &models.DatosAnexo1{
		TenantNombre:     tenantNombre,
		TenantRUC:        tenantRUC,
		PlanillaID:       planilla.ID,
		PlanillaDesc:     planilla.Descripcion,
		PlanillaAnio:     planilla.Anio,
		PlanillaMes:      planilla.Mes,
		PlanillaEstado:   planilla.Estado,
		Items:            items,
		ResumenMetas:     resumenMetas,
		AjustesAplicados: ajustes,
		MontoTotal:       math.Round(montoTotal*100) / 100,
	}, nil
}

// GenerarAnexo1PDF genera el documento PDF institucional para el Anexo 1 (Compromiso Presupuestal)
func (s *AnexoService) GenerarAnexo1PDF(data *models.DatosAnexo1) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(12, 12, 12)
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	pdf.AliasNbPages("{nb}")
	pdf.SetFooterFunc(func() {
		pdf.SetY(-12)
		pdf.SetFont("Arial", "I", 8)
		pdf.SetTextColor(100, 100, 100)
		infoPie := fmt.Sprintf("Anexo 1 - Compromiso Presupuestal | %s | Periodo: %02d/%d", data.PlanillaDesc, data.PlanillaMes, data.PlanillaAnio)
		pdf.CellFormat(130, 6, tr(infoPie), "", 0, "L", false, 0, "")
		pdf.CellFormat(56, 6, tr(fmt.Sprintf("Página %d de {nb}", pdf.PageNo())), "", 0, "R", false, 0, "")
	})

	pdf.AddPage()

	// Cabecera Institucional
	pdf.SetFont("Arial", "B", 13)
	pdf.SetTextColor(31, 78, 121)
	pdf.CellFormat(0, 7, tr(strings.ToUpper(data.TenantNombre)), "", 1, "C", false, 0, "")

	if data.TenantRUC != "" {
		pdf.SetFont("Arial", "", 9)
		pdf.SetTextColor(80, 80, 80)
		pdf.CellFormat(0, 5, tr("RUC: "+data.TenantRUC), "", 1, "C", false, 0, "")
	}

	pdf.Ln(2)
	pdf.SetFont("Arial", "B", 11)
	pdf.SetTextColor(0, 0, 0)
	pdf.CellFormat(0, 6, tr("ANEXO 1: DETALLE DEL COMPROMISO PRESUPUESTAL"), "B", 1, "C", false, 0, "")
	pdf.Ln(3)

	// Metadatos de la planilla
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(25, 5, tr("Planilla:"), "", 0, "L", false, 0, "")
	pdf.SetFont("Arial", "", 9)
	pdf.CellFormat(100, 5, tr(data.PlanillaDesc), "", 0, "L", false, 0, "")

	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(20, 5, tr("Periodo:"), "", 0, "L", false, 0, "")
	pdf.SetFont("Arial", "", 9)
	pdf.CellFormat(41, 5, tr(fmt.Sprintf("%02d/%d", data.PlanillaMes, data.PlanillaAnio)), "", 1, "L", false, 0, "")

	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(25, 5, tr("Estado:"), "", 0, "L", false, 0, "")
	if strings.EqualFold(data.PlanillaEstado, "BORRADOR") {
		pdf.SetTextColor(200, 100, 0)
	} else {
		pdf.SetTextColor(0, 120, 0)
	}
	pdf.CellFormat(100, 5, tr(strings.ToUpper(data.PlanillaEstado)), "", 1, "L", false, 0, "")
	pdf.SetTextColor(0, 0, 0)

	pdf.Ln(4)

	// Tabla de Contenido
	wMeta := 22.0
	wClasif := 30.0
	wDesc := 104.0
	wMonto := 30.0

	// Cabecera Tabla
	pdf.SetFillColor(31, 78, 121)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(wMeta, 7, tr("META"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(wClasif, 7, tr("CLASIFICADOR"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(wDesc, 7, tr("DESCRIPCIÓN DEL CLASIFICADOR"), "1", 0, "L", true, 0, "")
	pdf.CellFormat(wMonto, 7, tr("MONTO (S/)"), "1", 1, "R", true, 0, "")

	pdf.SetTextColor(0, 0, 0)

	for _, metaGroup := range data.ResumenMetas {
		// Cabecera de Meta
		pdf.SetFillColor(235, 240, 245)
		pdf.SetFont("Arial", "B", 8)
		metaHeaderStr := fmt.Sprintf("META: %s - %s", metaGroup.MetaCodigo, metaGroup.MetaDescripcion)
		pdf.CellFormat(wMeta+wClasif+wDesc, 6, tr(metaHeaderStr), "1", 0, "L", true, 0, "")
		pdf.CellFormat(wMonto, 6, "", "1", 1, "R", true, 0, "")

		// Filas por clasificador
		pdf.SetFont("Arial", "", 8)
		for i, item := range metaGroup.Items {
			fill := (i%2 == 1)
			if fill {
				pdf.SetFillColor(250, 250, 250)
			} else {
				pdf.SetFillColor(255, 255, 255)
			}

			descTruncated := item.ClasificadorDescripcion
			if len(descTruncated) > 65 {
				descTruncated = descTruncated[:62] + "..."
			}

			pdf.CellFormat(wMeta, 6, tr(item.MetaCodigo), "1", 0, "C", fill, 0, "")
			pdf.CellFormat(wClasif, 6, tr(item.ClasificadorCodigo), "1", 0, "C", fill, 0, "")
			pdf.CellFormat(wDesc, 6, tr(descTruncated), "1", 0, "L", fill, 0, "")
			pdf.CellFormat(wMonto, 6, fmt.Sprintf("%.2f", item.MontoTotal), "1", 1, "R", fill, 0, "")
		}

		// Subtotal Meta
		pdf.SetFillColor(220, 230, 242)
		pdf.SetFont("Arial", "B", 8)
		pdf.CellFormat(wMeta+wClasif+wDesc, 6, tr("SUBTOTAL META "+metaGroup.MetaCodigo), "1", 0, "R", true, 0, "")
		pdf.CellFormat(wMonto, 6, fmt.Sprintf("%.2f", metaGroup.TotalMeta), "1", 1, "R", true, 0, "")
	}

	// Gran Total
	pdf.SetFillColor(31, 78, 121)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(wMeta+wClasif+wDesc, 7, tr("TOTAL COMPROMISO PRESUPUESTAL S/"), "1", 0, "R", true, 0, "")
	pdf.CellFormat(wMonto, 7, fmt.Sprintf("%.2f", data.MontoTotal), "1", 1, "R", true, 0, "")

	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, fmt.Errorf("error al generar PDF de Anexo 1: %w", err)
	}

	return buf.Bytes(), nil
}

// GenerarAnexo1Excel genera un hoja de cálculo Excel (.xlsx) estructurada para el Anexo 1
func (s *AnexoService) GenerarAnexo1Excel(data *models.DatosAnexo1) ([]byte, error) {
	f := excelize.NewFile()
	sheet := "Anexo 1 - Compromiso"
	f.SetSheetName("Sheet1", sheet)

	// Estilos
	styleTitle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14, Color: "1F4E79"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	styleSubtitle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10, Color: "555555"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	styleHeader, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF", Size: 10},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"1F4E79"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "left", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})

	styleMetaHeader, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 10, Color: "1F4E79"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"EBF0F5"}, Pattern: 1},
		Border: []excelize.Border{
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})

	styleDataText, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Size: 10},
		Border: []excelize.Border{
			{Type: "top", Color: "E0E0E0", Style: 1},
			{Type: "bottom", Color: "E0E0E0", Style: 1},
			{Type: "left", Color: "E0E0E0", Style: 1},
			{Type: "right", Color: "E0E0E0", Style: 1},
		},
	})

	styleDataCenter, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10},
		Alignment: &excelize.Alignment{Horizontal: "center"},
		Border: []excelize.Border{
			{Type: "top", Color: "E0E0E0", Style: 1},
			{Type: "bottom", Color: "E0E0E0", Style: 1},
			{Type: "left", Color: "E0E0E0", Style: 1},
			{Type: "right", Color: "E0E0E0", Style: 1},
		},
	})

	styleDataMoney, _ := f.NewStyle(&excelize.Style{
		Font:         &excelize.Font{Size: 10},
		Alignment:    &excelize.Alignment{Horizontal: "right"},
		CustomNumFmt: strPtr("#,##0.00"),
		Border: []excelize.Border{
			{Type: "top", Color: "E0E0E0", Style: 1},
			{Type: "bottom", Color: "E0E0E0", Style: 1},
			{Type: "left", Color: "E0E0E0", Style: 1},
			{Type: "right", Color: "E0E0E0", Style: 1},
		},
	})

	styleSubtotal, _ := f.NewStyle(&excelize.Style{
		Font:         &excelize.Font{Bold: true, Size: 10},
		Fill:         excelize.Fill{Type: "pattern", Color: []string{"DC8F2"}, Pattern: 1},
		Alignment:    &excelize.Alignment{Horizontal: "right"},
		CustomNumFmt: strPtr("#,##0.00"),
		Border: []excelize.Border{
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
		},
	})

	styleTotal, _ := f.NewStyle(&excelize.Style{
		Font:         &excelize.Font{Bold: true, Size: 11, Color: "FFFFFF"},
		Fill:         excelize.Fill{Type: "pattern", Color: []string{"1F4E79"}, Pattern: 1},
		Alignment:    &excelize.Alignment{Horizontal: "right"},
		CustomNumFmt: strPtr("#,##0.00"),
		Border: []excelize.Border{
			{Type: "top", Color: "000000", Style: 2},
			{Type: "bottom", Color: "000000", Style: 2},
		},
	})

	// Título
	f.MergeCell(sheet, "A1", "E1")
	f.SetCellValue(sheet, "A1", data.TenantNombre)
	f.SetCellStyle(sheet, "A1", "E1", styleTitle)

	f.MergeCell(sheet, "A2", "E2")
	f.SetCellValue(sheet, "A2", fmt.Sprintf("ANEXO 1: DETALLE DEL COMPROMISO PRESUPUESTAL - PERIODO %02d/%d", data.PlanillaMes, data.PlanillaAnio))
	f.SetCellStyle(sheet, "A2", "E2", styleSubtitle)

	f.SetCellValue(sheet, "A3", "Planilla:")
	f.SetCellValue(sheet, "B3", data.PlanillaDesc)

	row := 5
	// Cabeceras de tabla
	headers := []string{"Meta Código", "Meta Descripción", "Clasificador", "Descripción Clasificador", "Monto (S/)"}
	cols := []string{"A", "B", "C", "D", "E"}

	for i, h := range headers {
		cell := fmt.Sprintf("%s%d", cols[i], row)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, styleHeader)
	}

	row++

	for _, metaGroup := range data.ResumenMetas {
		// Subcabecera de Meta
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("META %s: %s", metaGroup.MetaCodigo, metaGroup.MetaDescripcion))
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("E%d", row), styleMetaHeader)
		row++

		for _, item := range metaGroup.Items {
			f.SetCellValue(sheet, fmt.Sprintf("A%d", row), item.MetaCodigo)
			f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), styleDataCenter)

			f.SetCellValue(sheet, fmt.Sprintf("B%d", row), item.MetaDescripcion)
			f.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), styleDataText)

			f.SetCellValue(sheet, fmt.Sprintf("C%d", row), item.ClasificadorCodigo)
			f.SetCellStyle(sheet, fmt.Sprintf("C%d", row), fmt.Sprintf("C%d", row), styleDataCenter)

			f.SetCellValue(sheet, fmt.Sprintf("D%d", row), item.ClasificadorDescripcion)
			f.SetCellStyle(sheet, fmt.Sprintf("D%d", row), fmt.Sprintf("D%d", row), styleDataText)

			f.SetCellValue(sheet, fmt.Sprintf("E%d", row), item.MontoTotal)
			f.SetCellStyle(sheet, fmt.Sprintf("E%d", row), fmt.Sprintf("E%d", row), styleDataMoney)

			row++
		}

		// Subtotal
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Subtotal Meta "+metaGroup.MetaCodigo)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), metaGroup.TotalMeta)
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("E%d", row), styleSubtotal)
		row++
	}

	// Gran Total
	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "TOTAL COMPROMISO PRESUPUESTAL S/")
	f.SetCellValue(sheet, fmt.Sprintf("E%d", row), data.MontoTotal)
	f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("E%d", row), styleTotal)

	// Ancho de columnas
	f.SetColWidth(sheet, "A", "A", 14)
	f.SetColWidth(sheet, "B", "B", 35)
	f.SetColWidth(sheet, "C", "C", 16)
	f.SetColWidth(sheet, "D", "D", 45)
	f.SetColWidth(sheet, "E", "E", 18)

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("error al escribir Excel de Anexo 1: %w", err)
	}

	return buf.Bytes(), nil
}

func strPtr(s string) *string {
	return &s
}
