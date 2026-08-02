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

	var totalONP, totalRenta4ta, totalRenta5ta float64
	var nombreONP, nombreRenta4ta, nombreRenta5ta string

	for _, c := range conceptos {
		cod := strings.TrimSpace(c.CodigoSunat)
		nom := strings.ToUpper(strings.TrimSpace(c.NombreEnBoleta))

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

	var resultados []models.AjusteRedondeoSunat

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

		if math.Abs(diferencia) > 0.001 {
			target, err := s.AnexoRepo.ObtenerTargetCompromisoAjuste(planillaID, tenantID, codigos, palabraClave)
			if err == nil {
				ajuste.MetaCodigoTarget = target.MetaCodigo
				ajuste.ClasificadorTarget = target.ClasificadorCodigo
				ajuste.CodigoSunatIngresoTarget = target.CodigoSunatIngreso
				ajuste.NombreIngresoTarget = target.NombreIngreso
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

	items, err := s.AnexoRepo.ObtenerCompromisoPresupuestal(planillaID, tenantID)
	if err != nil {
		return nil, err
	}

	ajustes, err := s.CalcularAjustesRedondeo(planillaID, tenantID)
	if err != nil {
		ajustes = nil
	}

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

// ObtenerDatosAnexo1A prepara los datos consolidados del Anexo 1A (Resumen por Conceptos de Planilla)
func (s *AnexoService) ObtenerDatosAnexo1A(planillaID, tenantID int) (*models.DatosAnexo1A, error) {
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

	items, err := s.AnexoRepo.ObtenerResumenConceptosPlanilla(planillaID, tenantID)
	if err != nil {
		return nil, err
	}

	ajustes, _ := s.CalcularAjustesRedondeo(planillaID, tenantID)

	// Aplicar diferencias de redondeo directamente sobre las retenciones Y sobre el concepto de INGRESO remunerativo target
	for _, ajuste := range ajustes {
		if math.Abs(ajuste.Diferencia) > 0.001 {
			// A. Sumar diferencia a la RETENCION SUNAT
			for i := range items {
				if items[i].TipoConcepto == "RETENCION" {
					cod := strings.TrimSpace(items[i].CodigoSunat)
					nom := strings.ToUpper(strings.TrimSpace(items[i].NombreConcepto))

					esTarget := false
					switch ajuste.ConceptoClave {
					case "ONP":
						esTarget = (cod == "0607" || strings.Contains(nom, "ONP") || strings.Contains(nom, "SNP") || strings.Contains(nom, "19990"))
					case "RENTA_4TA":
						esTarget = (cod == "S101" || strings.Contains(nom, "CUARTA") || strings.Contains(nom, "4TA"))
					case "RENTA_5TA":
						esTarget = (cod == "0605" || strings.Contains(nom, "QUINTA") || strings.Contains(nom, "5TA"))
					}

					if esTarget {
						items[i].MontoTotal = math.Round((items[i].MontoTotal+ajuste.Diferencia)*100) / 100
						break
					}
				}
			}

			// B. Sumar la misma diferencia al INGRESO remunerativo target para mantener cuadre perfecto
			if ajuste.NombreIngresoTarget != "" {
				aplicado := false
				for i := range items {
					if items[i].TipoConcepto == "INGRESO" {
						if items[i].CodigoSunat == ajuste.CodigoSunatIngresoTarget && items[i].NombreConcepto == ajuste.NombreIngresoTarget {
							items[i].MontoTotal = math.Round((items[i].MontoTotal+ajuste.Diferencia)*100) / 100
							aplicado = true
							break
						}
					}
				}
				// Fallback si no hubo coincidencia exacta por nombre: sumar al primer concepto INGRESO
				if !aplicado {
					for i := range items {
						if items[i].TipoConcepto == "INGRESO" {
							items[i].MontoTotal = math.Round((items[i].MontoTotal+ajuste.Diferencia)*100) / 100
							break
						}
					}
				}
			}
		}
	}

	mapaGrupos := map[string]*models.GrupoResumenConcepto{
		"INGRESO": {
			TipoConcepto: "INGRESO",
			Titulo:       "1. INGRESOS Y REMUNERACIONES",
			Items:        []models.ItemResumenConcepto{},
			TotalGrupo:   0,
		},
		"RETENCION": {
			TipoConcepto: "RETENCION",
			Titulo:       "2. RETENCIONES / DESCUENTOS AL TRABAJADOR",
			Items:        []models.ItemResumenConcepto{},
			TotalGrupo:   0,
		},
		"APORTE": {
			TipoConcepto: "APORTE",
			Titulo:       "3. APORTES DE LA ENTIDAD / EMPLEADOR",
			Items:        []models.ItemResumenConcepto{},
			TotalGrupo:   0,
		},
	}

	for _, item := range items {
		tipo := strings.ToUpper(strings.TrimSpace(item.TipoConcepto))
		grupo, existe := mapaGrupos[tipo]
		if existe {
			grupo.Items = append(grupo.Items, item)
			grupo.TotalGrupo += item.MontoTotal
		}
	}

	for k := range mapaGrupos {
		mapaGrupos[k].TotalGrupo = math.Round(mapaGrupos[k].TotalGrupo*100) / 100
	}

	ordenTipos := []string{"INGRESO", "RETENCION", "APORTE"}
	var grupos []models.GrupoResumenConcepto
	for _, t := range ordenTipos {
		if g, ok := mapaGrupos[t]; ok && len(g.Items) > 0 {
			grupos = append(grupos, *g)
		}
	}

	totalIngresos := mapaGrupos["INGRESO"].TotalGrupo
	totalRetenciones := mapaGrupos["RETENCION"].TotalGrupo
	totalAportes := mapaGrupos["APORTE"].TotalGrupo
	costoTotal := math.Round((totalIngresos+totalAportes)*100) / 100

	return &models.DatosAnexo1A{
		TenantNombre:     tenantNombre,
		TenantRUC:        tenantRUC,
		PlanillaID:       planilla.ID,
		PlanillaDesc:     planilla.Descripcion,
		PlanillaAnio:     planilla.Anio,
		PlanillaMes:      planilla.Mes,
		PlanillaEstado:   planilla.Estado,
		Grupos:           grupos,
		TotalIngresos:    totalIngresos,
		TotalRetenciones: totalRetenciones,
		TotalAportes:     totalAportes,
		CostoTotal:       costoTotal,
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

	wMeta := 14.0
	wClasif := 24.0
	wDesc := 118.0
	wMonto := 30.0

	pdf.SetFillColor(31, 78, 121)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(wMeta, 7, tr("META"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(wClasif, 7, tr("CLASIFICADOR"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(wDesc, 7, tr("DESCRIPCIÓN DEL CLASIFICADOR"), "1", 0, "L", true, 0, "")
	pdf.CellFormat(wMonto, 7, tr("MONTO (S/)"), "1", 1, "R", true, 0, "")

	pdf.SetTextColor(0, 0, 0)

	for _, metaGroup := range data.ResumenMetas {
		pdf.SetFillColor(235, 240, 245)
		pdf.SetFont("Arial", "B", 8)
		metaHeaderStr := fmt.Sprintf("META: %s - %s", metaGroup.MetaCodigo, metaGroup.MetaDescripcion)
		pdf.CellFormat(wMeta+wClasif+wDesc, 6, tr(metaHeaderStr), "1", 0, "L", true, 0, "")
		pdf.CellFormat(wMonto, 6, "", "1", 1, "R", true, 0, "")

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

		pdf.SetFillColor(220, 230, 242)
		pdf.SetFont("Arial", "B", 8)
		pdf.CellFormat(wMeta+wClasif+wDesc, 6, tr("SUBTOTAL META "+metaGroup.MetaCodigo), "1", 0, "R", true, 0, "")
		pdf.CellFormat(wMonto, 6, fmt.Sprintf("%.2f", metaGroup.TotalMeta), "1", 1, "R", true, 0, "")
	}

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

// GenerarAnexo1APDF genera el documento PDF institucional para el Anexo 1A (Resumen por Conceptos de Planilla)
func (s *AnexoService) GenerarAnexo1APDF(data *models.DatosAnexo1A) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(12, 12, 12)
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	pdf.AliasNbPages("{nb}")
	pdf.SetFooterFunc(func() {
		pdf.SetY(-12)
		pdf.SetFont("Arial", "I", 8)
		pdf.SetTextColor(100, 100, 100)
		infoPie := fmt.Sprintf("Anexo 1A - Resumen por Conceptos | %s | Periodo: %02d/%d", data.PlanillaDesc, data.PlanillaMes, data.PlanillaAnio)
		pdf.CellFormat(130, 6, tr(infoPie), "", 0, "L", false, 0, "")
		pdf.CellFormat(56, 6, tr(fmt.Sprintf("Página %d de {nb}", pdf.PageNo())), "", 0, "R", false, 0, "")
	})

	pdf.AddPage()

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
	pdf.CellFormat(0, 6, tr("ANEXO 1A: RESUMEN POR CONCEPTOS DE PLANILLA"), "B", 1, "C", false, 0, "")
	pdf.Ln(3)

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

	wCod := 30.0
	wDesc := 126.0
	wMonto := 30.0

	pdf.SetFillColor(31, 78, 121)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(wCod, 7, tr("CÓDIGO SUNAT"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(wDesc, 7, tr("DESCRIPCIÓN DEL CONCEPTO"), "1", 0, "L", true, 0, "")
	pdf.CellFormat(wMonto, 7, tr("MONTO (S/)"), "1", 1, "R", true, 0, "")

	pdf.SetTextColor(0, 0, 0)

	for _, g := range data.Grupos {
		pdf.SetFillColor(235, 240, 245)
		pdf.SetFont("Arial", "B", 8)
		pdf.CellFormat(wCod+wDesc, 6, tr(g.Titulo), "1", 0, "L", true, 0, "")
		pdf.CellFormat(wMonto, 6, "", "1", 1, "R", true, 0, "")

		pdf.SetFont("Arial", "", 8)
		for i, item := range g.Items {
			fill := (i%2 == 1)
			if fill {
				pdf.SetFillColor(250, 250, 250)
			} else {
				pdf.SetFillColor(255, 255, 255)
			}

			nomTruncated := item.NombreConcepto
			if len(nomTruncated) > 80 {
				nomTruncated = nomTruncated[:77] + "..."
			}

			pdf.CellFormat(wCod, 6, tr(item.CodigoSunat), "1", 0, "C", fill, 0, "")
			pdf.CellFormat(wDesc, 6, tr(nomTruncated), "1", 0, "L", fill, 0, "")
			pdf.CellFormat(wMonto, 6, fmt.Sprintf("%.2f", item.MontoTotal), "1", 1, "R", fill, 0, "")
		}

		pdf.SetFillColor(220, 230, 242)
		pdf.SetFont("Arial", "B", 8)
		pdf.CellFormat(wCod+wDesc, 6, tr("SUBTOTAL "+g.Titulo), "1", 0, "R", true, 0, "")
		pdf.CellFormat(wMonto, 6, fmt.Sprintf("%.2f", g.TotalGrupo), "1", 1, "R", true, 0, "")
	}

	pdf.Ln(2)

	// Cuadro resumen final de costos
	pdf.SetFillColor(245, 247, 250)
	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(wCod+wDesc, 6, tr("SUMATORIA TOTAL DE INGRESOS (1):"), "1", 0, "R", true, 0, "")
	pdf.CellFormat(wMonto, 6, fmt.Sprintf("%.2f", data.TotalIngresos), "1", 1, "R", true, 0, "")

	pdf.CellFormat(wCod+wDesc, 6, tr("SUMATORIA TOTAL DE RETENCIONES (2):"), "1", 0, "R", true, 0, "")
	pdf.CellFormat(wMonto, 6, fmt.Sprintf("%.2f", data.TotalRetenciones), "1", 1, "R", true, 0, "")

	pdf.CellFormat(wCod+wDesc, 6, tr("SUMATORIA TOTAL DE APORTES EMPLEADOR (3):"), "1", 0, "R", true, 0, "")
	pdf.CellFormat(wMonto, 6, fmt.Sprintf("%.2f", data.TotalAportes), "1", 1, "R", true, 0, "")

	pdf.SetFillColor(31, 78, 121)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(wCod+wDesc, 7, tr("COSTO TOTAL DE PLANILLA S/ (INGRESOS + APORTES):"), "1", 0, "R", true, 0, "")
	pdf.CellFormat(wMonto, 7, fmt.Sprintf("%.2f", data.CostoTotal), "1", 1, "R", true, 0, "")

	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, fmt.Errorf("error al generar PDF de Anexo 1A: %w", err)
	}

	return buf.Bytes(), nil
}

// GenerarAnexo1Excel genera un hoja de cálculo Excel (.xlsx) estructurada para el Anexo 1
func (s *AnexoService) GenerarAnexo1Excel(data *models.DatosAnexo1) ([]byte, error) {
	f := excelize.NewFile()
	sheet := "Anexo 1 - Compromiso"
	f.SetSheetName("Sheet1", sheet)

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

	f.MergeCell(sheet, "A1", "E1")
	f.SetCellValue(sheet, "A1", data.TenantNombre)
	f.SetCellStyle(sheet, "A1", "E1", styleTitle)

	f.MergeCell(sheet, "A2", "E2")
	f.SetCellValue(sheet, "A2", fmt.Sprintf("ANEXO 1: DETALLE DEL COMPROMISO PRESUPUESTAL - PERIODO %02d/%d", data.PlanillaMes, data.PlanillaAnio))
	f.SetCellStyle(sheet, "A2", "E2", styleSubtitle)

	f.SetCellValue(sheet, "A3", "Planilla:")
	f.SetCellValue(sheet, "B3", data.PlanillaDesc)

	row := 5
	headers := []string{"Meta Código", "Meta Descripción", "Clasificador", "Descripción Clasificador", "Monto (S/)"}
	cols := []string{"A", "B", "C", "D", "E"}

	for i, h := range headers {
		cell := fmt.Sprintf("%s%d", cols[i], row)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, styleHeader)
	}

	row++

	for _, metaGroup := range data.ResumenMetas {
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

		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Subtotal Meta "+metaGroup.MetaCodigo)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), metaGroup.TotalMeta)
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("E%d", row), styleSubtotal)
		row++
	}

	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "TOTAL COMPROMISO PRESUPUESTAL S/")
	f.SetCellValue(sheet, fmt.Sprintf("E%d", row), data.MontoTotal)
	f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("E%d", row), styleTotal)

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

// GenerarAnexo1AExcel genera una hoja de cálculo Excel (.xlsx) estructurada para el Anexo 1A
func (s *AnexoService) GenerarAnexo1AExcel(data *models.DatosAnexo1A) ([]byte, error) {
	f := excelize.NewFile()
	sheet := "Anexo 1A - Conceptos"
	f.SetSheetName("Sheet1", sheet)

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

	styleGroupHeader, _ := f.NewStyle(&excelize.Style{
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

	f.MergeCell(sheet, "A1", "D1")
	f.SetCellValue(sheet, "A1", data.TenantNombre)
	f.SetCellStyle(sheet, "A1", "D1", styleTitle)

	f.MergeCell(sheet, "A2", "D2")
	f.SetCellValue(sheet, "A2", fmt.Sprintf("ANEXO 1A: RESUMEN POR CONCEPTOS DE PLANILLA - PERIODO %02d/%d", data.PlanillaMes, data.PlanillaAnio))
	f.SetCellStyle(sheet, "A2", "D2", styleSubtitle)

	f.SetCellValue(sheet, "A3", "Planilla:")
	f.SetCellValue(sheet, "B3", data.PlanillaDesc)

	row := 5
	headers := []string{"Tipo Concepto", "Código SUNAT", "Descripción del Concepto", "Monto Total (S/)"}
	cols := []string{"A", "B", "C", "D"}

	for i, h := range headers {
		cell := fmt.Sprintf("%s%d", cols[i], row)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, styleHeader)
	}

	row++

	for _, g := range data.Grupos {
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), g.Titulo)
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("D%d", row), styleGroupHeader)
		row++

		for _, item := range g.Items {
			f.SetCellValue(sheet, fmt.Sprintf("A%d", row), item.TipoConcepto)
			f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), styleDataCenter)

			f.SetCellValue(sheet, fmt.Sprintf("B%d", row), item.CodigoSunat)
			f.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), styleDataCenter)

			f.SetCellValue(sheet, fmt.Sprintf("C%d", row), item.NombreConcepto)
			f.SetCellStyle(sheet, fmt.Sprintf("C%d", row), fmt.Sprintf("C%d", row), styleDataText)

			f.SetCellValue(sheet, fmt.Sprintf("D%d", row), item.MontoTotal)
			f.SetCellStyle(sheet, fmt.Sprintf("D%d", row), fmt.Sprintf("D%d", row), styleDataMoney)

			row++
		}

		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Subtotal "+g.Titulo)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), g.TotalGrupo)
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("D%d", row), styleSubtotal)
		row++
	}

	row++
	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "TOTAL INGRESOS Y REMUNERACIONES (1)")
	f.SetCellValue(sheet, fmt.Sprintf("D%d", row), data.TotalIngresos)
	f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("D%d", row), styleSubtotal)
	row++

	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "TOTAL RETENCIONES / DESCUENTOS (2)")
	f.SetCellValue(sheet, fmt.Sprintf("D%d", row), data.TotalRetenciones)
	f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("D%d", row), styleSubtotal)
	row++

	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "TOTAL APORTES EMPLEADOR (3)")
	f.SetCellValue(sheet, fmt.Sprintf("D%d", row), data.TotalAportes)
	f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("D%d", row), styleSubtotal)
	row++

	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "COSTO TOTAL PLANILLA S/ (INGRESOS + APORTES)")
	f.SetCellValue(sheet, fmt.Sprintf("D%d", row), data.CostoTotal)
	f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("D%d", row), styleTotal)

	f.SetColWidth(sheet, "A", "A", 18)
	f.SetColWidth(sheet, "B", "B", 16)
	f.SetColWidth(sheet, "C", "C", 50)
	f.SetColWidth(sheet, "D", "D", 22)

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("error al escribir Excel de Anexo 1A: %w", err)
	}

	return buf.Bytes(), nil
}

// ObtenerDatosAnexo2 prepara los datos consolidados del Anexo 2 (Resumen por AFP)
func (s *AnexoService) ObtenerDatosAnexo2(planillaID, tenantID int) (*models.DatosAnexo2, error) {
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

	items, err := s.AnexoRepo.ObtenerResumenAFP(planillaID, tenantID)
	if err != nil {
		return nil, err
	}

	var totalAporte, totalComision, totalPrima, granTotal float64
	for _, item := range items {
		totalAporte += item.AporteObligatorio
		totalComision += item.Comision
		totalPrima += item.PrimaSeguro
		granTotal += item.TotalAFP
	}

	return &models.DatosAnexo2{
		TenantNombre:           tenantNombre,
		TenantRUC:              tenantRUC,
		PlanillaID:             planilla.ID,
		PlanillaDesc:           planilla.Descripcion,
		PlanillaAnio:           planilla.Anio,
		PlanillaMes:            planilla.Mes,
		PlanillaEstado:         planilla.Estado,
		Items:                  items,
		TotalAporteObligatorio: math.Round(totalAporte*100) / 100,
		TotalComision:          math.Round(totalComision*100) / 100,
		TotalPrimaSeguro:       math.Round(totalPrima*100) / 100,
		GranTotal:              math.Round(granTotal*100) / 100,
	}, nil
}

// GenerarAnexo2PDF genera el documento PDF institucional para el Anexo 2 (Resumen por AFP)
func (s *AnexoService) GenerarAnexo2PDF(data *models.DatosAnexo2) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(12, 12, 12)
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	pdf.AliasNbPages("{nb}")
	pdf.SetFooterFunc(func() {
		pdf.SetY(-12)
		pdf.SetFont("Arial", "I", 8)
		pdf.SetTextColor(100, 100, 100)
		infoPie := fmt.Sprintf("Anexo 2 - Resumen por AFP | %s | Periodo: %02d/%d", data.PlanillaDesc, data.PlanillaMes, data.PlanillaAnio)
		pdf.CellFormat(130, 6, tr(infoPie), "", 0, "L", false, 0, "")
		pdf.CellFormat(56, 6, tr(fmt.Sprintf("Página %d de {nb}", pdf.PageNo())), "", 0, "R", false, 0, "")
	})

	pdf.AddPage()

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
	pdf.CellFormat(0, 6, tr("ANEXO 2: RESUMEN POR AFP"), "B", 1, "C", false, 0, "")
	pdf.Ln(3)

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

	wAFP := 50.0
	wAporte := 34.0
	wComis := 34.0
	wPrima := 34.0
	wTotal := 34.0

	pdf.SetFillColor(31, 78, 121)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(wAFP, 7, tr("AFP"), "1", 0, "L", true, 0, "")
	pdf.CellFormat(wAporte, 7, tr("APORTE OBLIG. (0608)"), "1", 0, "R", true, 0, "")
	pdf.CellFormat(wComis, 7, tr("COMISIÓN AFP (0601)"), "1", 0, "R", true, 0, "")
	pdf.CellFormat(wPrima, 7, tr("PRIMA SEGURO (0606)"), "1", 0, "R", true, 0, "")
	pdf.CellFormat(wTotal, 7, tr("TOTAL S/"), "1", 1, "R", true, 0, "")

	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Arial", "", 8)

	for i, item := range data.Items {
		fill := (i%2 == 1)
		if fill {
			pdf.SetFillColor(250, 250, 250)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}

		pdf.CellFormat(wAFP, 6, tr(item.AFPNombre), "1", 0, "L", fill, 0, "")
		pdf.CellFormat(wAporte, 6, fmt.Sprintf("%.2f", item.AporteObligatorio), "1", 0, "R", fill, 0, "")
		pdf.CellFormat(wComis, 6, fmt.Sprintf("%.2f", item.Comision), "1", 0, "R", fill, 0, "")
		pdf.CellFormat(wPrima, 6, fmt.Sprintf("%.2f", item.PrimaSeguro), "1", 0, "R", fill, 0, "")
		pdf.CellFormat(wTotal, 6, fmt.Sprintf("%.2f", item.TotalAFP), "1", 1, "R", fill, 0, "")
	}

	// Totales
	pdf.SetFillColor(31, 78, 121)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(wAFP, 7, tr("TOTALES S/"), "1", 0, "L", true, 0, "")
	pdf.CellFormat(wAporte, 7, fmt.Sprintf("%.2f", data.TotalAporteObligatorio), "1", 0, "R", true, 0, "")
	pdf.CellFormat(wComis, 7, fmt.Sprintf("%.2f", data.TotalComision), "1", 0, "R", true, 0, "")
	pdf.CellFormat(wPrima, 7, fmt.Sprintf("%.2f", data.TotalPrimaSeguro), "1", 0, "R", true, 0, "")
	pdf.CellFormat(wTotal, 7, fmt.Sprintf("%.2f", data.GranTotal), "1", 1, "R", true, 0, "")

	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, fmt.Errorf("error al generar PDF de Anexo 2: %w", err)
	}

	return buf.Bytes(), nil
}

// GenerarAnexo2Excel genera un hoja de cálculo Excel (.xlsx) para el Anexo 2
func (s *AnexoService) GenerarAnexo2Excel(data *models.DatosAnexo2) ([]byte, error) {
	f := excelize.NewFile()
	sheet := "Anexo 2 - Resumen AFP"
	f.SetSheetName("Sheet1", sheet)

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

	styleDataText, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Size: 10},
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

	f.MergeCell(sheet, "A1", "E1")
	f.SetCellValue(sheet, "A1", data.TenantNombre)
	f.SetCellStyle(sheet, "A1", "E1", styleTitle)

	f.MergeCell(sheet, "A2", "E2")
	f.SetCellValue(sheet, "A2", fmt.Sprintf("ANEXO 2: RESUMEN POR AFP - PERIODO %02d/%d", data.PlanillaMes, data.PlanillaAnio))
	f.SetCellStyle(sheet, "A2", "E2", styleSubtitle)

	f.SetCellValue(sheet, "A3", "Planilla:")
	f.SetCellValue(sheet, "B3", data.PlanillaDesc)

	row := 5
	headers := []string{"AFP", "Aporte Obligatorio (0608)", "Comisión AFP (0601)", "Prima Seguro (0606)", "Total S/"}
	cols := []string{"A", "B", "C", "D", "E"}

	for i, h := range headers {
		cell := fmt.Sprintf("%s%d", cols[i], row)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, styleHeader)
	}

	row++

	for _, item := range data.Items {
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), item.AFPNombre)
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), styleDataText)

		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), item.AporteObligatorio)
		f.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), styleDataMoney)

		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), item.Comision)
		f.SetCellStyle(sheet, fmt.Sprintf("C%d", row), fmt.Sprintf("C%d", row), styleDataMoney)

		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), item.PrimaSeguro)
		f.SetCellStyle(sheet, fmt.Sprintf("D%d", row), fmt.Sprintf("D%d", row), styleDataMoney)

		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), item.TotalAFP)
		f.SetCellStyle(sheet, fmt.Sprintf("E%d", row), fmt.Sprintf("E%d", row), styleDataMoney)

		row++
	}

	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "TOTALES S/")
	f.SetCellValue(sheet, fmt.Sprintf("B%d", row), data.TotalAporteObligatorio)
	f.SetCellValue(sheet, fmt.Sprintf("C%d", row), data.TotalComision)
	f.SetCellValue(sheet, fmt.Sprintf("D%d", row), data.TotalPrimaSeguro)
	f.SetCellValue(sheet, fmt.Sprintf("E%d", row), data.GranTotal)
	f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("E%d", row), styleTotal)

	f.SetColWidth(sheet, "A", "A", 25)
	f.SetColWidth(sheet, "B", "B", 25)
	f.SetColWidth(sheet, "C", "C", 22)
	f.SetColWidth(sheet, "D", "D", 22)
	f.SetColWidth(sheet, "E", "E", 22)

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("error al escribir Excel de Anexo 2: %w", err)
	}

	return buf.Bytes(), nil
}

// ObtenerDatosAnexo2A prepara los datos consolidados del Anexo 2A (Registro Devengado por Meta/Clasificador por AFP)
func (s *AnexoService) ObtenerDatosAnexo2A(planillaID, tenantID int) (*models.DatosAnexo2A, error) {
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

	items, err := s.AnexoRepo.ObtenerDevengadoAFP(planillaID, tenantID)
	if err != nil {
		return nil, err
	}

	mapaGrupos := make(map[string]*models.GrupoDevengadoAFP)
	var ordenAFPs []string
	var totalAporte, totalComision, totalPrima, granTotal float64

	for _, item := range items {
		totalAporte += item.AporteObligatorio
		totalComision += item.Comision
		totalPrima += item.PrimaSeguro
		granTotal += item.TotalFila

		grupo, existe := mapaGrupos[item.AFPNombre]
		if !existe {
			grupo = &models.GrupoDevengadoAFP{
				AFPNombre:              item.AFPNombre,
				Items:                  []models.ItemDevengadoAFP{},
				TotalAporteObligatorio: 0,
				TotalComision:          0,
				TotalPrimaSeguro:       0,
				TotalGrupo:             0,
			}
			mapaGrupos[item.AFPNombre] = grupo
			ordenAFPs = append(ordenAFPs, item.AFPNombre)
		}
		grupo.Items = append(grupo.Items, item)
		grupo.TotalAporteObligatorio += item.AporteObligatorio
		grupo.TotalComision += item.Comision
		grupo.TotalPrimaSeguro += item.PrimaSeguro
		grupo.TotalGrupo += item.TotalFila
	}

	var grupos []models.GrupoDevengadoAFP
	for _, afp := range ordenAFPs {
		g := mapaGrupos[afp]
		g.TotalAporteObligatorio = math.Round(g.TotalAporteObligatorio*100) / 100
		g.TotalComision = math.Round(g.TotalComision*100) / 100
		g.TotalPrimaSeguro = math.Round(g.TotalPrimaSeguro*100) / 100
		g.TotalGrupo = math.Round(g.TotalGrupo*100) / 100
		grupos = append(grupos, *g)
	}

	return &models.DatosAnexo2A{
		TenantNombre:           tenantNombre,
		TenantRUC:              tenantRUC,
		PlanillaID:             planilla.ID,
		PlanillaDesc:           planilla.Descripcion,
		PlanillaAnio:           planilla.Anio,
		PlanillaMes:            planilla.Mes,
		PlanillaEstado:         planilla.Estado,
		Grupos:                 grupos,
		TotalAporteObligatorio: math.Round(totalAporte*100) / 100,
		TotalComision:          math.Round(totalComision*100) / 100,
		TotalPrimaSeguro:       math.Round(totalPrima*100) / 100,
		GranTotal:              math.Round(granTotal*100) / 100,
	}, nil
}

// GenerarAnexo2APDF genera el documento PDF institucional para el Anexo 2A (Registro Devengado por Meta/Clasificador)
func (s *AnexoService) GenerarAnexo2APDF(data *models.DatosAnexo2A) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(10, 10, 10)
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	pdf.AliasNbPages("{nb}")
	pdf.SetFooterFunc(func() {
		pdf.SetY(-12)
		pdf.SetFont("Arial", "I", 8)
		pdf.SetTextColor(100, 100, 100)
		infoPie := fmt.Sprintf("Anexo 2A - Devengado AFP | %s | Periodo: %02d/%d", data.PlanillaDesc, data.PlanillaMes, data.PlanillaAnio)
		pdf.CellFormat(134, 6, tr(infoPie), "", 0, "L", false, 0, "")
		pdf.CellFormat(56, 6, tr(fmt.Sprintf("Página %d de {nb}", pdf.PageNo())), "", 0, "R", false, 0, "")
	})

	pdf.AddPage()

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
	pdf.CellFormat(0, 6, tr("ANEXO 2A: REGISTRO DEVENGADO DE AFP POR META Y CLASIFICADOR"), "B", 1, "C", false, 0, "")
	pdf.Ln(3)

	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(25, 5, tr("Planilla:"), "", 0, "L", false, 0, "")
	pdf.SetFont("Arial", "", 9)
	pdf.CellFormat(100, 5, tr(data.PlanillaDesc), "", 0, "L", false, 0, "")

	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(20, 5, tr("Periodo:"), "", 0, "L", false, 0, "")
	pdf.SetFont("Arial", "", 9)
	pdf.CellFormat(45, 5, tr(fmt.Sprintf("%02d/%d", data.PlanillaMes, data.PlanillaAnio)), "", 1, "L", false, 0, "")

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

	// Ancho total = 190mm
	wMeta := 16.0
	wClasif := 26.0
	wDesc := 66.0
	wAporte := 21.0
	wComis := 21.0
	wPrima := 20.0
	wTotal := 20.0

	pdf.SetFillColor(31, 78, 121)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(wMeta, 7, tr("META"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(wClasif, 7, tr("CLASIFICADOR"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(wDesc, 7, tr("DESCRIPCIÓN DEL CLASIFICADOR"), "1", 0, "L", true, 0, "")
	pdf.CellFormat(wAporte, 7, tr("APORTE(0608)"), "1", 0, "R", true, 0, "")
	pdf.CellFormat(wComis, 7, tr("COMIS.(0601)"), "1", 0, "R", true, 0, "")
	pdf.CellFormat(wPrima, 7, tr("PRIMA(0606)"), "1", 0, "R", true, 0, "")
	pdf.CellFormat(wTotal, 7, tr("TOTAL S/"), "1", 1, "R", true, 0, "")

	pdf.SetTextColor(0, 0, 0)

	for _, g := range data.Grupos {
		pdf.SetFillColor(235, 240, 245)
		pdf.SetFont("Arial", "B", 8)
		pdf.CellFormat(wMeta+wClasif+wDesc+wAporte+wComis+wPrima+wTotal, 6, tr("AFP: "+g.AFPNombre), "1", 1, "L", true, 0, "")

		pdf.SetFont("Arial", "", 8)
		for i, item := range g.Items {
			fill := (i%2 == 1)
			if fill {
				pdf.SetFillColor(250, 250, 250)
			} else {
				pdf.SetFillColor(255, 255, 255)
			}

			descTrunc := item.ClasificadorDescripcion
			if len(descTrunc) > 42 {
				descTrunc = descTrunc[:39] + "..."
			}

			pdf.CellFormat(wMeta, 6, tr(item.MetaCodigo), "1", 0, "C", fill, 0, "")
			pdf.CellFormat(wClasif, 6, tr(item.ClasificadorCodigo), "1", 0, "C", fill, 0, "")
			pdf.CellFormat(wDesc, 6, tr(descTrunc), "1", 0, "L", fill, 0, "")
			pdf.CellFormat(wAporte, 6, fmt.Sprintf("%.2f", item.AporteObligatorio), "1", 0, "R", fill, 0, "")
			pdf.CellFormat(wComis, 6, fmt.Sprintf("%.2f", item.Comision), "1", 0, "R", fill, 0, "")
			pdf.CellFormat(wPrima, 6, fmt.Sprintf("%.2f", item.PrimaSeguro), "1", 0, "R", fill, 0, "")
			pdf.CellFormat(wTotal, 6, fmt.Sprintf("%.2f", item.TotalFila), "1", 1, "R", fill, 0, "")
		}

		pdf.SetFillColor(220, 230, 242)
		pdf.SetFont("Arial", "B", 8)
		pdf.CellFormat(wMeta+wClasif+wDesc, 6, tr("SUBTOTAL AFP "+g.AFPNombre), "1", 0, "R", true, 0, "")
		pdf.CellFormat(wAporte, 6, fmt.Sprintf("%.2f", g.TotalAporteObligatorio), "1", 0, "R", true, 0, "")
		pdf.CellFormat(wComis, 6, fmt.Sprintf("%.2f", g.TotalComision), "1", 0, "R", true, 0, "")
		pdf.CellFormat(wPrima, 6, fmt.Sprintf("%.2f", g.TotalPrimaSeguro), "1", 0, "R", true, 0, "")
		pdf.CellFormat(wTotal, 6, fmt.Sprintf("%.2f", g.TotalGrupo), "1", 1, "R", true, 0, "")
	}

	pdf.SetFillColor(31, 78, 121)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(wMeta+wClasif+wDesc, 7, tr("TOTALES DEVENGADO AFP S/"), "1", 0, "R", true, 0, "")
	pdf.CellFormat(wAporte, 7, fmt.Sprintf("%.2f", data.TotalAporteObligatorio), "1", 0, "R", true, 0, "")
	pdf.CellFormat(wComis, 7, fmt.Sprintf("%.2f", data.TotalComision), "1", 0, "R", true, 0, "")
	pdf.CellFormat(wPrima, 7, fmt.Sprintf("%.2f", data.TotalPrimaSeguro), "1", 0, "R", true, 0, "")
	pdf.CellFormat(wTotal, 7, fmt.Sprintf("%.2f", data.GranTotal), "1", 1, "R", true, 0, "")

	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, fmt.Errorf("error al generar PDF de Anexo 2A: %w", err)
	}

	return buf.Bytes(), nil
}

// GenerarAnexo2AExcel genera una hoja de cálculo Excel (.xlsx) para el Anexo 2A
func (s *AnexoService) GenerarAnexo2AExcel(data *models.DatosAnexo2A) ([]byte, error) {
	f := excelize.NewFile()
	sheet := "Anexo 2A - Devengado AFP"
	f.SetSheetName("Sheet1", sheet)

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

	styleGroupHeader, _ := f.NewStyle(&excelize.Style{
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

	f.MergeCell(sheet, "A1", "G1")
	f.SetCellValue(sheet, "A1", data.TenantNombre)
	f.SetCellStyle(sheet, "A1", "G1", styleTitle)

	f.MergeCell(sheet, "A2", "G2")
	f.SetCellValue(sheet, "A2", fmt.Sprintf("ANEXO 2A: REGISTRO DEVENGADO DE AFP POR META Y CLASIFICADOR - PERIODO %02d/%d", data.PlanillaMes, data.PlanillaAnio))
	f.SetCellStyle(sheet, "A2", "G2", styleSubtitle)

	f.SetCellValue(sheet, "A3", "Planilla:")
	f.SetCellValue(sheet, "B3", data.PlanillaDesc)

	row := 5
	headers := []string{"Meta Código", "Clasificador", "Descripción Clasificador", "Aporte Oblig. (0608)", "Comisión (0601)", "Prima Seguro (0606)", "Total S/"}
	cols := []string{"A", "B", "C", "D", "E", "F", "G"}

	for i, h := range headers {
		cell := fmt.Sprintf("%s%d", cols[i], row)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, styleHeader)
	}

	row++

	for _, g := range data.Grupos {
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "AFP: "+g.AFPNombre)
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("G%d", row), styleGroupHeader)
		row++

		for _, item := range g.Items {
			f.SetCellValue(sheet, fmt.Sprintf("A%d", row), item.MetaCodigo)
			f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), styleDataCenter)

			f.SetCellValue(sheet, fmt.Sprintf("B%d", row), item.ClasificadorCodigo)
			f.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), styleDataCenter)

			f.SetCellValue(sheet, fmt.Sprintf("C%d", row), item.ClasificadorDescripcion)
			f.SetCellStyle(sheet, fmt.Sprintf("C%d", row), fmt.Sprintf("C%d", row), styleDataText)

			f.SetCellValue(sheet, fmt.Sprintf("D%d", row), item.AporteObligatorio)
			f.SetCellStyle(sheet, fmt.Sprintf("D%d", row), fmt.Sprintf("D%d", row), styleDataMoney)

			f.SetCellValue(sheet, fmt.Sprintf("E%d", row), item.Comision)
			f.SetCellStyle(sheet, fmt.Sprintf("E%d", row), fmt.Sprintf("E%d", row), styleDataMoney)

			f.SetCellValue(sheet, fmt.Sprintf("F%d", row), item.PrimaSeguro)
			f.SetCellStyle(sheet, fmt.Sprintf("F%d", row), fmt.Sprintf("F%d", row), styleDataMoney)

			f.SetCellValue(sheet, fmt.Sprintf("G%d", row), item.TotalFila)
			f.SetCellStyle(sheet, fmt.Sprintf("G%d", row), fmt.Sprintf("G%d", row), styleDataMoney)

			row++
		}

		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Subtotal AFP "+g.AFPNombre)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), g.TotalAporteObligatorio)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), g.TotalComision)
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), g.TotalPrimaSeguro)
		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), g.TotalGrupo)
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("G%d", row), styleSubtotal)
		row++
	}

	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "TOTALES DEVENGADO AFP S/")
	f.SetCellValue(sheet, fmt.Sprintf("D%d", row), data.TotalAporteObligatorio)
	f.SetCellValue(sheet, fmt.Sprintf("E%d", row), data.TotalComision)
	f.SetCellValue(sheet, fmt.Sprintf("F%d", row), data.TotalPrimaSeguro)
	f.SetCellValue(sheet, fmt.Sprintf("G%d", row), data.GranTotal)
	f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("G%d", row), styleTotal)

	f.SetColWidth(sheet, "A", "A", 14)
	f.SetColWidth(sheet, "B", "B", 18)
	f.SetColWidth(sheet, "C", "C", 45)
	f.SetColWidth(sheet, "D", "D", 20)
	f.SetColWidth(sheet, "E", "E", 18)
	f.SetColWidth(sheet, "F", "F", 18)
	f.SetColWidth(sheet, "G", "G", 20)

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("error al escribir Excel de Anexo 2A: %w", err)
	}

	return buf.Bytes(), nil
}

// ObtenerDatosAnexo3 prepara los datos consolidados del Anexo 3 (Retenciones de SUNAT) con redondeo tributario aplicado
func (s *AnexoService) ObtenerDatosAnexo3(planillaID, tenantID int) (*models.DatosAnexo3, error) {
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

	items, err := s.AnexoRepo.ObtenerRetencionesSunat(planillaID, tenantID)
	if err != nil {
		return nil, err
	}

	ajustes, _ := s.CalcularAjustesRedondeo(planillaID, tenantID)

	// Aplicar diferencias de redondeo SUNAT a los ítems target para que coincida 100% con Anexo 1A y PLAME
	for _, ajuste := range ajustes {
		if math.Abs(ajuste.Diferencia) > 0.001 && ajuste.MetaCodigoTarget != "" && ajuste.ClasificadorTarget != "" {
			for i := range items {
				if items[i].MetaCodigo == ajuste.MetaCodigoTarget && items[i].ClasificadorCodigo == ajuste.ClasificadorTarget {
					switch ajuste.ConceptoClave {
					case "ONP":
						items[i].ONP = math.Round((items[i].ONP+ajuste.Diferencia)*100) / 100
					case "RENTA_4TA":
						items[i].Renta4ta = math.Round((items[i].Renta4ta+ajuste.Diferencia)*100) / 100
					case "RENTA_5TA":
						items[i].Renta5ta = math.Round((items[i].Renta5ta+ajuste.Diferencia)*100) / 100
					}
					items[i].TotalFila = math.Round((items[i].ONP+items[i].Renta4ta+items[i].Renta5ta)*100) / 100
					break
				}
			}
		}
	}

	var totalONP, total4ta, total5ta, granTotal float64
	for _, item := range items {
		totalONP += item.ONP
		total4ta += item.Renta4ta
		total5ta += item.Renta5ta
		granTotal += item.TotalFila
	}

	return &models.DatosAnexo3{
		TenantNombre:   tenantNombre,
		TenantRUC:      tenantRUC,
		PlanillaID:     planilla.ID,
		PlanillaDesc:   planilla.Descripcion,
		PlanillaAnio:   planilla.Anio,
		PlanillaMes:    planilla.Mes,
		PlanillaEstado: planilla.Estado,
		Items:          items,
		TotalONP:       math.Round(totalONP*100) / 100,
		TotalRenta4ta:  math.Round(total4ta*100) / 100,
		TotalRenta5ta:  math.Round(total5ta*100) / 100,
		GranTotal:      math.Round(granTotal*100) / 100,
	}, nil
}

// GenerarAnexo3PDF genera el documento PDF institucional para el Anexo 3 (Retenciones de SUNAT)
func (s *AnexoService) GenerarAnexo3PDF(data *models.DatosAnexo3) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(10, 10, 10)
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	pdf.AliasNbPages("{nb}")
	pdf.SetFooterFunc(func() {
		pdf.SetY(-12)
		pdf.SetFont("Arial", "I", 8)
		pdf.SetTextColor(100, 100, 100)
		infoPie := fmt.Sprintf("Anexo 3 - Retenciones de SUNAT | %s | Periodo: %02d/%d", data.PlanillaDesc, data.PlanillaMes, data.PlanillaAnio)
		pdf.CellFormat(134, 6, tr(infoPie), "", 0, "L", false, 0, "")
		pdf.CellFormat(56, 6, tr(fmt.Sprintf("Página %d de {nb}", pdf.PageNo())), "", 0, "R", false, 0, "")
	})

	pdf.AddPage()

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
	pdf.CellFormat(0, 6, tr("ANEXO 3: RETENCIONES DE SUNAT POR META Y CLASIFICADOR"), "B", 1, "C", false, 0, "")
	pdf.Ln(3)

	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(25, 5, tr("Planilla:"), "", 0, "L", false, 0, "")
	pdf.SetFont("Arial", "", 9)
	pdf.CellFormat(100, 5, tr(data.PlanillaDesc), "", 0, "L", false, 0, "")

	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(20, 5, tr("Periodo:"), "", 0, "L", false, 0, "")
	pdf.SetFont("Arial", "", 9)
	pdf.CellFormat(45, 5, tr(fmt.Sprintf("%02d/%d", data.PlanillaMes, data.PlanillaAnio)), "", 1, "L", false, 0, "")

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

	// Ancho total = 190mm
	wMeta := 16.0
	wClasif := 26.0
	wDesc := 66.0
	wONP := 21.0
	wR4ta := 21.0
	wR5ta := 20.0
	wTotal := 20.0

	pdf.SetFillColor(31, 78, 121)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(wMeta, 7, tr("META"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(wClasif, 7, tr("CLASIFICADOR"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(wDesc, 7, tr("DESCRIPCIÓN DEL CLASIFICADOR"), "1", 0, "L", true, 0, "")
	pdf.CellFormat(wONP, 7, tr("ONP(0607)"), "1", 0, "R", true, 0, "")
	pdf.CellFormat(wR4ta, 7, tr("RENTA 4TA"), "1", 0, "R", true, 0, "")
	pdf.CellFormat(wR5ta, 7, tr("RENTA 5TA"), "1", 0, "R", true, 0, "")
	pdf.CellFormat(wTotal, 7, tr("TOTAL S/"), "1", 1, "R", true, 0, "")

	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Arial", "", 8)

	for i, item := range data.Items {
		fill := (i%2 == 1)
		if fill {
			pdf.SetFillColor(250, 250, 250)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}

		descTrunc := item.ClasificadorDescripcion
		if len(descTrunc) > 42 {
			descTrunc = descTrunc[:39] + "..."
		}

		pdf.CellFormat(wMeta, 6, tr(item.MetaCodigo), "1", 0, "C", fill, 0, "")
		pdf.CellFormat(wClasif, 6, tr(item.ClasificadorCodigo), "1", 0, "C", fill, 0, "")
		pdf.CellFormat(wDesc, 6, tr(descTrunc), "1", 0, "L", fill, 0, "")
		pdf.CellFormat(wONP, 6, fmt.Sprintf("%.2f", item.ONP), "1", 0, "R", fill, 0, "")
		pdf.CellFormat(wR4ta, 6, fmt.Sprintf("%.2f", item.Renta4ta), "1", 0, "R", fill, 0, "")
		pdf.CellFormat(wR5ta, 6, fmt.Sprintf("%.2f", item.Renta5ta), "1", 0, "R", fill, 0, "")
		pdf.CellFormat(wTotal, 6, fmt.Sprintf("%.2f", item.TotalFila), "1", 1, "R", fill, 0, "")
	}

	// Totales
	pdf.SetFillColor(31, 78, 121)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(wMeta+wClasif+wDesc, 7, tr("TOTALES RETENCIONES SUNAT S/"), "1", 0, "R", true, 0, "")
	pdf.CellFormat(wONP, 7, fmt.Sprintf("%.2f", data.TotalONP), "1", 0, "R", true, 0, "")
	pdf.CellFormat(wR4ta, 7, fmt.Sprintf("%.2f", data.TotalRenta4ta), "1", 0, "R", true, 0, "")
	pdf.CellFormat(wR5ta, 7, fmt.Sprintf("%.2f", data.TotalRenta5ta), "1", 0, "R", true, 0, "")
	pdf.CellFormat(wTotal, 7, fmt.Sprintf("%.2f", data.GranTotal), "1", 1, "R", true, 0, "")

	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, fmt.Errorf("error al generar PDF de Anexo 3: %w", err)
	}

	return buf.Bytes(), nil
}

// GenerarAnexo3Excel genera un hoja de cálculo Excel (.xlsx) para el Anexo 3 (Retenciones de SUNAT)
func (s *AnexoService) GenerarAnexo3Excel(data *models.DatosAnexo3) ([]byte, error) {
	f := excelize.NewFile()
	sheet := "Anexo 3 - Retenciones SUNAT"
	f.SetSheetName("Sheet1", sheet)

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

	f.MergeCell(sheet, "A1", "G1")
	f.SetCellValue(sheet, "A1", data.TenantNombre)
	f.SetCellStyle(sheet, "A1", "G1", styleTitle)

	f.MergeCell(sheet, "A2", "G2")
	f.SetCellValue(sheet, "A2", fmt.Sprintf("ANEXO 3: RETENCIONES DE SUNAT POR META Y CLASIFICADOR - PERIODO %02d/%d", data.PlanillaMes, data.PlanillaAnio))
	f.SetCellStyle(sheet, "A2", "G2", styleSubtitle)

	f.SetCellValue(sheet, "A3", "Planilla:")
	f.SetCellValue(sheet, "B3", data.PlanillaDesc)

	row := 5
	headers := []string{"Meta Código", "Clasificador", "Descripción Clasificador", "ONP (0607)", "Renta 4ta (S101)", "Renta 5ta (0605)", "Total S/"}
	cols := []string{"A", "B", "C", "D", "E", "F", "G"}

	for i, h := range headers {
		cell := fmt.Sprintf("%s%d", cols[i], row)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, styleHeader)
	}

	row++

	for _, item := range data.Items {
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), item.MetaCodigo)
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), styleDataCenter)

		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), item.ClasificadorCodigo)
		f.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), styleDataCenter)

		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), item.ClasificadorDescripcion)
		f.SetCellStyle(sheet, fmt.Sprintf("C%d", row), fmt.Sprintf("C%d", row), styleDataText)

		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), item.ONP)
		f.SetCellStyle(sheet, fmt.Sprintf("D%d", row), fmt.Sprintf("D%d", row), styleDataMoney)

		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), item.Renta4ta)
		f.SetCellStyle(sheet, fmt.Sprintf("E%d", row), fmt.Sprintf("E%d", row), styleDataMoney)

		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), item.Renta5ta)
		f.SetCellStyle(sheet, fmt.Sprintf("F%d", row), fmt.Sprintf("F%d", row), styleDataMoney)

		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), item.TotalFila)
		f.SetCellStyle(sheet, fmt.Sprintf("G%d", row), fmt.Sprintf("G%d", row), styleDataMoney)

		row++
	}

	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "TOTALES RETENCIONES SUNAT S/")
	f.SetCellValue(sheet, fmt.Sprintf("D%d", row), data.TotalONP)
	f.SetCellValue(sheet, fmt.Sprintf("E%d", row), data.TotalRenta4ta)
	f.SetCellValue(sheet, fmt.Sprintf("F%d", row), data.TotalRenta5ta)
	f.SetCellValue(sheet, fmt.Sprintf("G%d", row), data.GranTotal)
	f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("G%d", row), styleTotal)

	f.SetColWidth(sheet, "A", "A", 14)
	f.SetColWidth(sheet, "B", "B", 18)
	f.SetColWidth(sheet, "C", "C", 45)
	f.SetColWidth(sheet, "D", "D", 20)
	f.SetColWidth(sheet, "E", "E", 18)
	f.SetColWidth(sheet, "F", "F", 18)
	f.SetColWidth(sheet, "G", "G", 20)

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("error al escribir Excel de Anexo 3: %w", err)
	}

	return buf.Bytes(), nil
}

func strPtr(s string) *string {
	return &s
}
