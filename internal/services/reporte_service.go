package services

import (
	"bytes"
	"fmt"
	"os"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
	"strconv"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
	"github.com/xuri/excelize/v2"
)

type ReporteService struct {
	TrabajadorRepo     *repository.TrabajadorRepository
	OrganigramaRepo    *repository.OrganigramaRepository
	PuestoRepo         *repository.PuestoRepository
	ConceptoTenantRepo *repository.ConceptoTenantRepository
	ContratoRepo       *repository.ContratoRepository
	TenantRepo         *repository.TenantRepository
}

func NewReporteService(
	trabajadorRepo *repository.TrabajadorRepository,
	organigramaRepo *repository.OrganigramaRepository,
	puestoRepo *repository.PuestoRepository,
	conceptoTenantRepo *repository.ConceptoTenantRepository,
	contratoRepo *repository.ContratoRepository,
	tenantRepo *repository.TenantRepository,
) *ReporteService {
	return &ReporteService{
		TrabajadorRepo:     trabajadorRepo,
		OrganigramaRepo:    organigramaRepo,
		PuestoRepo:         puestoRepo,
		ConceptoTenantRepo: conceptoTenantRepo,
		ContratoRepo:       contratoRepo,
		TenantRepo:         tenantRepo,
	}
}

// GenerarPDF orquesta la generación de reportes en formato PDF y retorna un buffer con los bytes correspondientes.
func (s *ReporteService) GenerarPDF(tenantID int, id string, params map[string]string) (*bytes.Buffer, string, error) {
	var pdf *gofpdf.Fpdf
	nombreArchivo := fmt.Sprintf("Reporte_%s.pdf", id)

	switch id {
	case "trab_padron":
		// 👥 Padrón General de Personal (Landscape)
		pdf = gofpdf.New("L", "mm", "A4", "")
		pdf.AddPage()
		tr := pdf.UnicodeTranslatorFromDescriptor("")
		s.agregarCabeceraPDF(pdf, tenantID, "👥 PADRÓN GENERAL DE PERSONAL", "Listado completo de trabajadores de la entidad")
		
		trabajadores, err := s.TrabajadorRepo.ObtenerTodos(tenantID)
		if err != nil {
			return nil, "", fmt.Errorf("error cargando trabajadores: %w", err)
		}

		// Cabecera Tabla
		pdf.SetFillColor(230, 235, 245)
		pdf.SetFont("Arial", "B", 8)
		anchos := []float64{10, 20, 25, 75, 20, 15, 30, 60, 22}
		headers := []string{"N°", "Doc", "Nro Doc", "Apellidos y Nombres", "F. Nac.", "Sexo", "Rég. Pens.", "AFP / CUSPP", "Estado"}
		for i, h := range headers {
			pdf.CellFormat(anchos[i], 6, tr(h), "1", 0, "C", true, 0, "")
		}
		pdf.Ln(-1)

		pdf.SetFont("Arial", "", 8)
		for idx, t := range trabajadores {
			nombreCompleto := fmt.Sprintf("%s %s, %s", t.ApellidoPaterno, t.ApellidoMaterno, t.Nombres)
			estado := "INACTIVO"
			if t.Activo {
				estado = "ACTIVO"
			}
			
			afpInfo := "-"
			if t.RegimenPensionario == "AFP" {
				afpInfo = fmt.Sprintf("AFP (%s)", t.Cuspp)
			} else {
				afpInfo = "ONP"
			}

			rellenar := idx%2 == 1
			if rellenar {
				pdf.SetFillColor(245, 247, 250)
			}

			pdf.CellFormat(anchos[0], 5, fmt.Sprintf("%d", idx+1), "1", 0, "C", rellenar, 0, "")
			pdf.CellFormat(anchos[1], 5, tr(t.TipoDocumento), "1", 0, "C", rellenar, 0, "")
			pdf.CellFormat(anchos[2], 5, tr(t.NumeroDocumento), "1", 0, "C", rellenar, 0, "")
			pdf.CellFormat(anchos[3], 5, tr(nombreCompleto), "1", 0, "L", rellenar, 0, "")
			pdf.CellFormat(anchos[4], 5, tr(t.FechaNacimiento), "1", 0, "C", rellenar, 0, "")
			pdf.CellFormat(anchos[5], 5, tr(t.Sexo), "1", 0, "C", rellenar, 0, "")
			pdf.CellFormat(anchos[6], 5, tr(t.RegimenPensionario), "1", 0, "C", rellenar, 0, "")
			pdf.CellFormat(anchos[7], 5, tr(afpInfo), "1", 0, "L", rellenar, 0, "")
			pdf.CellFormat(anchos[8], 5, tr(estado), "1", 0, "C", rellenar, 0, "")
			pdf.Ln(-1)
		}

	case "trab_cumple":
		// 🎂 Cumpleaños del Mes (Portrait)
		mesStr := params["mes"]
		mes, _ := strconv.Atoi(mesStr)
		if mes < 1 || mes > 12 {
			mes = int(time.Now().Month())
		}
		
		mesesNombres := map[int]string{
			1: "ENERO", 2: "FEBRERO", 3: "MARZO", 4: "ABRIL", 5: "MAYO", 6: "JUNIO",
			7: "JULIO", 8: "AGOSTO", 9: "SEPTIEMBRE", 10: "OCTUBRE", 11: "NOVIEMBRE", 12: "DICIEMBRE",
		}

		pdf = gofpdf.New("P", "mm", "A4", "")
		pdf.AddPage()
		tr := pdf.UnicodeTranslatorFromDescriptor("")
		s.agregarCabeceraPDF(pdf, tenantID, "🎂 CUMPLEAÑOS DEL MES", "Personal que celebra su onomástico en: "+mesesNombres[mes])

		cumpleaneros, err := s.TrabajadorRepo.ObtenerCumpleaniosMes(tenantID, mes)
		if err != nil {
			return nil, "", fmt.Errorf("error cargando cumpleañeros: %w", err)
		}

		pdf.SetFillColor(254, 243, 245)
		pdf.SetFont("Arial", "B", 9)
		anchos := []float64{15, 25, 30, 90, 30}
		headers := []string{"Día", "Doc", "Nro Doc", "Apellidos y Nombres", "F. Nacimiento"}
		for i, h := range headers {
			pdf.CellFormat(anchos[i], 6, tr(h), "1", 0, "C", true, 0, "")
		}
		pdf.Ln(-1)

		pdf.SetFont("Arial", "", 9)
		for idx, t := range cumpleaneros {
			nombreCompleto := fmt.Sprintf("%s %s, %s", t.ApellidoPaterno, t.ApellidoMaterno, t.Nombres)
			dia := ""
			if len(t.FechaNacimiento) >= 10 {
				dia = t.FechaNacimiento[8:10]
			}

			rellenar := idx%2 == 1
			if rellenar {
				pdf.SetFillColor(255, 250, 250)
			}

			pdf.CellFormat(anchos[0], 6, tr(dia), "1", 0, "C", rellenar, 0, "")
			pdf.CellFormat(anchos[1], 6, tr(t.TipoDocumento), "1", 0, "C", rellenar, 0, "")
			pdf.CellFormat(anchos[2], 6, tr(t.NumeroDocumento), "1", 0, "C", rellenar, 0, "")
			pdf.CellFormat(anchos[3], 6, tr(nombreCompleto), "1", 0, "L", rellenar, 0, "")
			pdf.CellFormat(anchos[4], 6, tr(t.FechaNacimiento), "1", 0, "C", rellenar, 0, "")
			pdf.Ln(-1)
		}
		
		if len(cumpleaneros) == 0 {
			pdf.SetFont("Arial", "I", 10)
			pdf.Ln(5)
			pdf.CellFormat(190, 8, tr("No se encontraron cumpleaños registrados en este mes."), "", 1, "C", false, 0, "")
		}

	case "org_directorio":
		// 🏢 Directorio de Dependencias (Portrait)
		pdf = gofpdf.New("P", "mm", "A4", "")
		pdf.AddPage()
		tr := pdf.UnicodeTranslatorFromDescriptor("")
		s.agregarCabeceraPDF(pdf, tenantID, "🏢 DIRECTORIO DE DEPENDENCIAS MUNICIPALES", "Estructura orgánica de la versión activa de la municipalidad")

		org, err := s.OrganigramaRepo.ObtenerOrganigramaActivo(tenantID)
		if err != nil || org == nil {
			pdf.SetFont("Arial", "I", 10)
			pdf.CellFormat(190, 8, tr("No se encuentra un organigrama activo configurado."), "", 1, "C", false, 0, "")
		} else {
			unidades, err := s.OrganigramaRepo.ObtenerUnidades(org.ID)
			if err != nil {
				return nil, "", fmt.Errorf("error cargando unidades: %w", err)
			}

			pdf.SetFillColor(230, 240, 235)
			pdf.SetFont("Arial", "B", 9)
			anchos := []float64{15, 35, 100, 40}
			headers := []string{"N°", "Código MEF", "Nombre de Unidad Orgánica", "Tipo Unidad"}
			for i, h := range headers {
				pdf.CellFormat(anchos[i], 6, tr(h), "1", 0, "C", true, 0, "")
			}
			pdf.Ln(-1)

			pdf.SetFont("Arial", "", 9)
			for idx, u := range unidades {
				rellenar := idx%2 == 1
				if rellenar {
					pdf.SetFillColor(245, 249, 246)
				}

				codMef := u.CodigoMef
				if codMef == "" {
					codMef = "-"
				}

				pdf.CellFormat(anchos[0], 6, fmt.Sprintf("%d", idx+1), "1", 0, "C", rellenar, 0, "")
				pdf.CellFormat(anchos[1], 6, tr(codMef), "1", 0, "C", rellenar, 0, "")
				pdf.CellFormat(anchos[2], 6, tr(u.Nombre), "1", 0, "L", rellenar, 0, "")
				pdf.CellFormat(anchos[3], 6, tr(u.Tipo), "1", 0, "C", rellenar, 0, "")
				pdf.Ln(-1)
			}
		}

	case "puesto_resumen":
		// 📊 Ocupabilidad de Plazas (Portrait)
		pdf = gofpdf.New("P", "mm", "A4", "")
		pdf.AddPage()
		tr := pdf.UnicodeTranslatorFromDescriptor("")
		s.agregarCabeceraPDF(pdf, tenantID, "📊 OCUPABILIDAD DE PLAZAS (CAP/PAP)", "Estado ocupacional de las plazas registradas")

		puestos, err := s.PuestoRepo.ObtenerTodos(tenantID)
		if err != nil {
			return nil, "", fmt.Errorf("error cargando puestos: %w", err)
		}

		pdf.SetFillColor(235, 235, 240)
		pdf.SetFont("Arial", "B", 9)
		anchos := []float64{10, 25, 60, 45, 30, 20}
		headers := []string{"N°", "Cód. AIRHSP", "Nombre del Puesto", "Unidad Orgánica", "Régimen", "Estado"}
		for i, h := range headers {
			pdf.CellFormat(anchos[i], 6, tr(h), "1", 0, "C", true, 0, "")
		}
		pdf.Ln(-1)

		pdf.SetFont("Arial", "", 8)
		vacantes := 0
		ocupados := 0

		for idx, p := range puestos {
			if p.Estado == "VACANTE" {
				vacantes++
			} else {
				ocupados++
			}

			codAirhsp := "-"
			if p.CodigoAirhsp != nil && *p.CodigoAirhsp != "" {
				codAirhsp = *p.CodigoAirhsp
			}

			rellenar := idx%2 == 1
			if rellenar {
				pdf.SetFillColor(248, 248, 250)
			}

			pdf.CellFormat(anchos[0], 5, fmt.Sprintf("%d", idx+1), "1", 0, "C", rellenar, 0, "")
			pdf.CellFormat(anchos[1], 5, tr(codAirhsp), "1", 0, "C", rellenar, 0, "")
			pdf.CellFormat(anchos[2], 5, tr(p.Nombre), "1", 0, "L", rellenar, 0, "")
			pdf.CellFormat(anchos[3], 5, tr(p.UnidadOrganicaNombre), "1", 0, "L", rellenar, 0, "")
			pdf.CellFormat(anchos[4], 5, tr(p.RegimenDesc), "1", 0, "C", rellenar, 0, "")
			
			if p.Estado == "VACANTE" {
				pdf.SetTextColor(0, 100, 250)
			} else {
				pdf.SetTextColor(0, 120, 0)
			}
			pdf.CellFormat(anchos[5], 5, tr(p.Estado), "1", 0, "C", rellenar, 0, "")
			pdf.SetTextColor(0, 0, 0)
			pdf.Ln(-1)
		}

		pdf.Ln(4)
		pdf.SetFont("Arial", "B", 10)
		pdf.CellFormat(190, 8, tr(fmt.Sprintf("RESUMEN:   PLAZAS OCUPADAS: %d   |   PLAZAS VACANTES: %d   |   TOTAL PLAZAS: %d", ocupados, vacantes, len(puestos))), "1", 1, "C", false, 0, "")

	case "puesto_pap":
		// 💰 Presupuesto Analítico (PAP) Resumido (Portrait)
		pdf = gofpdf.New("P", "mm", "A4", "")
		pdf.AddPage()
		tr := pdf.UnicodeTranslatorFromDescriptor("")
		s.agregarCabeceraPDF(pdf, tenantID, "💰 PRESUPUESTO ANALÍTICO DE PERSONAL (PAP) RESUMIDO", "Cuadro resumido de costos mensuales por plaza")

		puestos, err := s.PuestoRepo.ObtenerTodos(tenantID)
		if err != nil {
			return nil, "", fmt.Errorf("error cargando puestos: %w", err)
		}

		pdf.SetFillColor(250, 240, 220)
		pdf.SetFont("Arial", "B", 9)
		anchos := []float64{10, 70, 50, 30, 30}
		headers := []string{"N°", "Nombre del Puesto", "Unidad Orgánica", "Sueldo Mensual", "Presupuesto Anual"}
		for i, h := range headers {
			pdf.CellFormat(anchos[i], 6, tr(h), "1", 0, "C", true, 0, "")
		}
		pdf.Ln(-1)

		pdf.SetFont("Arial", "", 9)
		var totalSueldo float64

		for idx, p := range puestos {
			totalSueldo += p.SueldoPresupuestado
			costoAnual := p.SueldoPresupuestado * 12

			rellenar := idx%2 == 1
			if rellenar {
				pdf.SetFillColor(255, 250, 245)
			}

			pdf.CellFormat(anchos[0], 5, fmt.Sprintf("%d", idx+1), "1", 0, "C", rellenar, 0, "")
			pdf.CellFormat(anchos[1], 5, tr(p.Nombre), "1", 0, "L", rellenar, 0, "")
			pdf.CellFormat(anchos[2], 5, tr(p.UnidadOrganicaNombre), "1", 0, "L", rellenar, 0, "")
			pdf.CellFormat(anchos[3], 5, fmt.Sprintf("S/ %.2f", p.SueldoPresupuestado), "1", 0, "R", rellenar, 0, "")
			pdf.CellFormat(anchos[4], 5, fmt.Sprintf("S/ %.2f", costoAnual), "1", 0, "R", rellenar, 0, "")
			pdf.Ln(-1)
		}

		pdf.Ln(4)
		pdf.SetFont("Arial", "B", 10)
		pdf.CellFormat(anchos[0]+anchos[1]+anchos[2], 8, tr("TOTAL COSTO PRESUPUESTADO ESTIMADO:"), "1", 0, "R", false, 0, "")
		pdf.CellFormat(anchos[3], 8, fmt.Sprintf("S/ %.2f", totalSueldo), "1", 0, "R", false, 0, "")
		pdf.CellFormat(anchos[4], 8, fmt.Sprintf("S/ %.2f", totalSueldo*12), "1", 1, "R", false, 0, "")

	case "concepto_maestro":
		// ⚙️ Catálogo Local de Conceptos (Landscape)
		pdf = gofpdf.New("L", "mm", "A4", "")
		pdf.AddPage()
		tr := pdf.UnicodeTranslatorFromDescriptor("")
		s.agregarCabeceraPDF(pdf, tenantID, "⚙️ CATÁLOGO LOCAL DE CONCEPTOS Y AFECTACIONES", "Ingresos, retenciones y aportaciones del tenant")

		conceptos, err := s.ConceptoTenantRepo.ObtenerTodos(tenantID)
		if err != nil {
			return nil, "", fmt.Errorf("error cargando conceptos: %w", err)
		}

		pdf.SetFillColor(230, 230, 240)
		pdf.SetFont("Arial", "B", 8)
		anchos := []float64{10, 15, 75, 20, 20, 20, 20, 40, 20, 17}
		headers := []string{"N°", "Código", "Concepto Local", "Tipo", "Remun.", "Pens.", "Extra.", "Clasificador MEF", "Frec.", "Estado"}
		for i, h := range headers {
			pdf.CellFormat(anchos[i], 6, tr(h), "1", 0, "C", true, 0, "")
		}
		pdf.Ln(-1)

		pdf.SetFont("Arial", "", 8)
		for idx, c := range conceptos {
			activo := "INACTIVO"
			if c.Activo {
				activo = "ACTIVO"
			}

			esRem := "NO"
			if c.EsRemunerativa {
				esRem = "SI"
			}
			esPen := "NO"
			if c.EsPensionable {
				esPen = "SI"
			}
			esExt := "NO"
			if c.EsExtraordinario {
				esExt = "SI"
			}

			codClasif := c.ClasificadorCodigo
			if codClasif == "" {
				codClasif = "-"
			}

			rellenar := idx%2 == 1
			if rellenar {
				pdf.SetFillColor(245, 245, 250)
			}

			pdf.CellFormat(anchos[0], 5, fmt.Sprintf("%d", idx+1), "1", 0, "C", rellenar, 0, "")
			pdf.CellFormat(anchos[1], 5, tr(c.ConceptoCodigo), "1", 0, "C", rellenar, 0, "")
			pdf.CellFormat(anchos[2], 5, tr(c.NombrePersonalizado), "1", 0, "L", rellenar, 0, "")
			pdf.CellFormat(anchos[3], 5, tr(c.ConceptoTipo), "1", 0, "C", rellenar, 0, "")
			pdf.CellFormat(anchos[4], 5, tr(esRem), "1", 0, "C", rellenar, 0, "")
			pdf.CellFormat(anchos[5], 5, tr(esPen), "1", 0, "C", rellenar, 0, "")
			pdf.CellFormat(anchos[6], 5, tr(esExt), "1", 0, "C", rellenar, 0, "")
			pdf.CellFormat(anchos[7], 5, tr(codClasif), "1", 0, "C", rellenar, 0, "")
			pdf.CellFormat(anchos[8], 5, fmt.Sprintf("C/%s M", c.FrecuenciaMeses), "1", 0, "C", rellenar, 0, "")
			pdf.CellFormat(anchos[9], 5, tr(activo), "1", 0, "C", rellenar, 0, "")
			pdf.Ln(-1)
		}

	case "contrato_vence":
		// ⏳ Alertas de Vencimiento de Contrato (Portrait)
		diasStr := params["dias"]
		dias, err := strconv.Atoi(diasStr)
		if err != nil || dias <= 0 {
			dias = 30
		}

		pdf = gofpdf.New("P", "mm", "A4", "")
		pdf.AddPage()
		tr := pdf.UnicodeTranslatorFromDescriptor("")
		s.agregarCabeceraPDF(pdf, tenantID, "⏳ ALERTAS DE VENCIMIENTO DE CONTRATOS", fmt.Sprintf("Contratos transitorios que vencen en los próximos %d días", dias))

		contratos, err := s.ContratoRepo.ObtenerContratosVencimiento(tenantID, dias)
		if err != nil {
			return nil, "", fmt.Errorf("error cargando contratos: %w", err)
		}

		pdf.SetFillColor(255, 235, 235)
		pdf.SetFont("Arial", "B", 9)
		anchos := []float64{10, 20, 65, 45, 25, 25}
		headers := []string{"N°", "Nro Doc", "Trabajador", "Puesto", "F. Inicio", "F. Vencimiento"}
		for i, h := range headers {
			pdf.CellFormat(anchos[i], 6, tr(h), "1", 0, "C", true, 0, "")
		}
		pdf.Ln(-1)

		pdf.SetFont("Arial", "", 8.5)
		for idx, c := range contratos {
			rellenar := idx%2 == 1
			if rellenar {
				pdf.SetFillColor(255, 248, 248)
			}

			fFin := "-"
			if c.FechaFin != nil {
				fFin = *c.FechaFin
			}

			pdf.CellFormat(anchos[0], 5, fmt.Sprintf("%d", idx+1), "1", 0, "C", rellenar, 0, "")
			pdf.CellFormat(anchos[1], 5, tr(c.TrabajadorDoc), "1", 0, "C", rellenar, 0, "")
			pdf.CellFormat(anchos[2], 5, tr(c.TrabajadorNombre), "1", 0, "L", rellenar, 0, "")
			pdf.CellFormat(anchos[3], 5, tr(c.PuestoNombre), "1", 0, "L", rellenar, 0, "")
			pdf.CellFormat(anchos[4], 5, tr(c.FechaInicio), "1", 0, "C", rellenar, 0, "")
			pdf.CellFormat(anchos[5], 5, tr(fFin), "1", 0, "C", rellenar, 0, "")
			pdf.Ln(-1)
		}

		if len(contratos) == 0 {
			pdf.SetFont("Arial", "I", 10)
			pdf.Ln(5)
			pdf.CellFormat(190, 8, tr("No se encontraron contratos próximos a vencer en el rango de días indicado."), "", 1, "C", false, 0, "")
		}

	default:
		return nil, "", fmt.Errorf("reporte no soportado o inválido: %s", id)
	}

	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, "", fmt.Errorf("error escribiendo salida PDF: %w", err)
	}

	return &buf, nombreArchivo, nil
}

// GenerarExcel orquesta la generación de reportes en Excel (con estilos premium) y retorna un buffer con los bytes.
func (s *ReporteService) GenerarExcel(tenantID int, id string, params map[string]string) (*bytes.Buffer, string, error) {
	f := excelize.NewFile()
	hoja := "Reporte"
	f.SetSheetName(f.GetSheetName(0), hoja)

	// Estilos Premium (Azul Marino Municipal)
	styleCabecera, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF", Family: "Segoe UI", Size: 11},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"1F4E79"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "D9D9D9", Style: 1},
			{Type: "right", Color: "D9D9D9", Style: 1},
			{Type: "top", Color: "D9D9D9", Style: 1},
			{Type: "bottom", Color: "D9D9D9", Style: 1},
		},
	})

	styleDatos, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Family: "Segoe UI", Size: 10},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "F2F2F2", Style: 1},
			{Type: "right", Color: "F2F2F2", Style: 1},
			{Type: "top", Color: "F2F2F2", Style: 1},
			{Type: "bottom", Color: "F2F2F2", Style: 1},
		},
	})

	styleCentrado, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Family: "Segoe UI", Size: 10},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "F2F2F2", Style: 1},
			{Type: "right", Color: "F2F2F2", Style: 1},
			{Type: "top", Color: "F2F2F2", Style: 1},
			{Type: "bottom", Color: "F2F2F2", Style: 1},
		},
	})

	styleMonto, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Family: "Segoe UI", Size: 10},
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		NumFmt: 2,
		Border: []excelize.Border{
			{Type: "left", Color: "F2F2F2", Style: 1},
			{Type: "right", Color: "F2F2F2", Style: 1},
			{Type: "top", Color: "F2F2F2", Style: 1},
			{Type: "bottom", Color: "F2F2F2", Style: 1},
		},
	})

	f.SetRowHeight(hoja, 1, 35)
	f.SetRowHeight(hoja, 2, 25)

	switch id {
	case "trab_padron":
		trabajadores, err := s.TrabajadorRepo.ObtenerTodos(tenantID)
		if err != nil {
			return nil, "", fmt.Errorf("error cargando trabajadores: %w", err)
		}

		cabeceras := []string{"N°", "Tipo Doc", "Nro Documento", "Apellido Paterno", "Apellido Materno", "Nombres", "Fecha Nacimiento", "Sexo", "Régimen Pensionario", "AFP/CUSPP", "Estado"}
		for i, h := range cabeceras {
			colName, _ := excelize.ColumnNumberToName(i + 1)
			f.SetCellValue(hoja, colName+"1", h)
			f.SetCellStyle(hoja, colName+"1", colName+"1", styleCabecera)
		}

		for idx, t := range trabajadores {
			rowNum := idx + 2
			f.SetRowHeight(hoja, rowNum, 20)
			
			estado := "INACTIVO"
			if t.Activo {
				estado = "ACTIVO"
			}

			afpVal := "ONP"
			if t.RegimenPensionario == "AFP" {
				afpVal = t.Cuspp
			}

			f.SetCellValue(hoja, fmt.Sprintf("A%d", rowNum), idx+1)
			f.SetCellValue(hoja, fmt.Sprintf("B%d", rowNum), t.TipoDocumento)
			f.SetCellValue(hoja, fmt.Sprintf("C%d", rowNum), t.NumeroDocumento)
			f.SetCellValue(hoja, fmt.Sprintf("D%d", rowNum), t.ApellidoPaterno)
			f.SetCellValue(hoja, fmt.Sprintf("E%d", rowNum), t.ApellidoMaterno)
			f.SetCellValue(hoja, fmt.Sprintf("F%d", rowNum), t.Nombres)
			f.SetCellValue(hoja, fmt.Sprintf("G%d", rowNum), t.FechaNacimiento)
			f.SetCellValue(hoja, fmt.Sprintf("H%d", rowNum), t.Sexo)
			f.SetCellValue(hoja, fmt.Sprintf("I%d", rowNum), t.RegimenPensionario)
			f.SetCellValue(hoja, fmt.Sprintf("J%d", rowNum), afpVal)
			f.SetCellValue(hoja, fmt.Sprintf("K%d", rowNum), estado)

			f.SetCellStyle(hoja, fmt.Sprintf("A%d", rowNum), fmt.Sprintf("A%d", rowNum), styleCentrado)
			f.SetCellStyle(hoja, fmt.Sprintf("B%d", rowNum), fmt.Sprintf("C%d", rowNum), styleCentrado)
			f.SetCellStyle(hoja, fmt.Sprintf("D%d", rowNum), fmt.Sprintf("F%d", rowNum), styleDatos)
			f.SetCellStyle(hoja, fmt.Sprintf("G%d", rowNum), fmt.Sprintf("I%d", rowNum), styleCentrado)
			f.SetCellStyle(hoja, fmt.Sprintf("J%d", rowNum), fmt.Sprintf("J%d", rowNum), styleDatos)
			f.SetCellStyle(hoja, fmt.Sprintf("K%d", rowNum), fmt.Sprintf("K%d", rowNum), styleCentrado)
		}
		
		s.ajustarAnchosColumnas(f, hoja, len(cabeceras))

	case "trab_cumple":
		mesStr := params["mes"]
		mes, _ := strconv.Atoi(mesStr)
		if mes < 1 || mes > 12 {
			mes = int(time.Now().Month())
		}

		cumpleaneros, err := s.TrabajadorRepo.ObtenerCumpleaniosMes(tenantID, mes)
		if err != nil {
			return nil, "", fmt.Errorf("error cargando cumpleañeros: %w", err)
		}

		cabeceras := []string{"Día", "Tipo Doc", "Nro Documento", "Apellidos y Nombres", "Fecha de Nacimiento"}
		for i, h := range cabeceras {
			colName, _ := excelize.ColumnNumberToName(i + 1)
			f.SetCellValue(hoja, colName+"1", h)
			f.SetCellStyle(hoja, colName+"1", colName+"1", styleCabecera)
		}

		for idx, t := range cumpleaneros {
			rowNum := idx + 2
			f.SetRowHeight(hoja, rowNum, 20)

			dia := ""
			if len(t.FechaNacimiento) >= 10 {
				dia = t.FechaNacimiento[8:10]
			}
			nombreCompleto := fmt.Sprintf("%s %s, %s", t.ApellidoPaterno, t.ApellidoMaterno, t.Nombres)

			f.SetCellValue(hoja, fmt.Sprintf("A%d", rowNum), dia)
			f.SetCellValue(hoja, fmt.Sprintf("B%d", rowNum), t.TipoDocumento)
			f.SetCellValue(hoja, fmt.Sprintf("C%d", rowNum), t.NumeroDocumento)
			f.SetCellValue(hoja, fmt.Sprintf("D%d", rowNum), nombreCompleto)
			f.SetCellValue(hoja, fmt.Sprintf("E%d", rowNum), t.FechaNacimiento)

			f.SetCellStyle(hoja, fmt.Sprintf("A%d", rowNum), fmt.Sprintf("C%d", rowNum), styleCentrado)
			f.SetCellStyle(hoja, fmt.Sprintf("D%d", rowNum), fmt.Sprintf("D%d", rowNum), styleDatos)
			f.SetCellStyle(hoja, fmt.Sprintf("E%d", rowNum), fmt.Sprintf("E%d", rowNum), styleCentrado)
		}

		s.ajustarAnchosColumnas(f, hoja, len(cabeceras))

	case "org_directorio":
		org, err := s.OrganigramaRepo.ObtenerOrganigramaActivo(tenantID)
		var unidades []models.UnidadOrganica
		if err == nil && org != nil {
			unidades, _ = s.OrganigramaRepo.ObtenerUnidades(org.ID)
		}

		cabeceras := []string{"N°", "Código MEF", "Nombre de la Oficina", "Tipo de Dependencia"}
		for i, h := range cabeceras {
			colName, _ := excelize.ColumnNumberToName(i + 1)
			f.SetCellValue(hoja, colName+"1", h)
			f.SetCellStyle(hoja, colName+"1", colName+"1", styleCabecera)
		}

		for idx, u := range unidades {
			rowNum := idx + 2
			f.SetRowHeight(hoja, rowNum, 20)

			f.SetCellValue(hoja, fmt.Sprintf("A%d", rowNum), idx+1)
			f.SetCellValue(hoja, fmt.Sprintf("B%d", rowNum), u.CodigoMef)
			f.SetCellValue(hoja, fmt.Sprintf("C%d", rowNum), u.Nombre)
			f.SetCellValue(hoja, fmt.Sprintf("D%d", rowNum), u.Tipo)

			f.SetCellStyle(hoja, fmt.Sprintf("A%d", rowNum), fmt.Sprintf("B%d", rowNum), styleCentrado)
			f.SetCellStyle(hoja, fmt.Sprintf("C%d", rowNum), fmt.Sprintf("C%d", rowNum), styleDatos)
			f.SetCellStyle(hoja, fmt.Sprintf("D%d", rowNum), fmt.Sprintf("D%d", rowNum), styleCentrado)
		}

		s.ajustarAnchosColumnas(f, hoja, len(cabeceras))

	case "puesto_resumen":
		puestos, err := s.PuestoRepo.ObtenerTodos(tenantID)
		if err != nil {
			return nil, "", fmt.Errorf("error cargando puestos: %w", err)
		}

		cabeceras := []string{"N°", "Código AIRHSP", "Nombre del Puesto", "Unidad Orgánica", "Régimen Laboral", "Sueldo Presupuestado", "Estado"}
		for i, h := range cabeceras {
			colName, _ := excelize.ColumnNumberToName(i + 1)
			f.SetCellValue(hoja, colName+"1", h)
			f.SetCellStyle(hoja, colName+"1", colName+"1", styleCabecera)
		}

		for idx, p := range puestos {
			rowNum := idx + 2
			f.SetRowHeight(hoja, rowNum, 20)

			codAirhsp := "-"
			if p.CodigoAirhsp != nil && *p.CodigoAirhsp != "" {
				codAirhsp = *p.CodigoAirhsp
			}

			f.SetCellValue(hoja, fmt.Sprintf("A%d", rowNum), idx+1)
			f.SetCellValue(hoja, fmt.Sprintf("B%d", rowNum), codAirhsp)
			f.SetCellValue(hoja, fmt.Sprintf("C%d", rowNum), p.Nombre)
			f.SetCellValue(hoja, fmt.Sprintf("D%d", rowNum), p.UnidadOrganicaNombre)
			f.SetCellValue(hoja, fmt.Sprintf("E%d", rowNum), p.RegimenDesc)
			f.SetCellValue(hoja, fmt.Sprintf("F%d", rowNum), p.SueldoPresupuestado)
			f.SetCellValue(hoja, fmt.Sprintf("G%d", rowNum), p.Estado)

			f.SetCellStyle(hoja, fmt.Sprintf("A%d", rowNum), fmt.Sprintf("B%d", rowNum), styleCentrado)
			f.SetCellStyle(hoja, fmt.Sprintf("C%d", rowNum), fmt.Sprintf("E%d", rowNum), styleDatos)
			f.SetCellStyle(hoja, fmt.Sprintf("F%d", rowNum), fmt.Sprintf("F%d", rowNum), styleMonto)
			f.SetCellStyle(hoja, fmt.Sprintf("G%d", rowNum), fmt.Sprintf("G%d", rowNum), styleCentrado)
		}

		s.ajustarAnchosColumnas(f, hoja, len(cabeceras))

	case "puesto_pap":
		puestos, err := s.PuestoRepo.ObtenerTodos(tenantID)
		if err != nil {
			return nil, "", fmt.Errorf("error cargando puestos: %w", err)
		}

		cabeceras := []string{"N°", "Nombre del Puesto", "Unidad Orgánica", "Régimen Laboral", "Sueldo Mensual", "Presupuesto Anual"}
		for i, h := range cabeceras {
			colName, _ := excelize.ColumnNumberToName(i + 1)
			f.SetCellValue(hoja, colName+"1", h)
			f.SetCellStyle(hoja, colName+"1", colName+"1", styleCabecera)
		}

		for idx, p := range puestos {
			rowNum := idx + 2
			f.SetRowHeight(hoja, rowNum, 20)

			f.SetCellValue(hoja, fmt.Sprintf("A%d", rowNum), idx+1)
			f.SetCellValue(hoja, fmt.Sprintf("B%d", rowNum), p.Nombre)
			f.SetCellValue(hoja, fmt.Sprintf("C%d", rowNum), p.UnidadOrganicaNombre)
			f.SetCellValue(hoja, fmt.Sprintf("D%d", rowNum), p.RegimenDesc)
			f.SetCellValue(hoja, fmt.Sprintf("E%d", rowNum), p.SueldoPresupuestado)
			f.SetCellValue(hoja, fmt.Sprintf("F%d", rowNum), p.SueldoPresupuestado*12)

			f.SetCellStyle(hoja, fmt.Sprintf("A%d", rowNum), fmt.Sprintf("A%d", rowNum), styleCentrado)
			f.SetCellStyle(hoja, fmt.Sprintf("B%d", rowNum), fmt.Sprintf("D%d", rowNum), styleDatos)
			f.SetCellStyle(hoja, fmt.Sprintf("E%d", rowNum), fmt.Sprintf("F%d", rowNum), styleMonto)
		}

		s.ajustarAnchosColumnas(f, hoja, len(cabeceras))

	case "concepto_maestro":
		conceptos, err := s.ConceptoTenantRepo.ObtenerTodos(tenantID)
		if err != nil {
			return nil, "", fmt.Errorf("error cargando conceptos: %w", err)
		}

		cabeceras := []string{"N°", "Código", "Concepto Local", "Tipo", "Remunerativo", "Pensionable", "Extraordinario", "Clasificador MEF", "Frecuencia", "Estado"}
		for i, h := range cabeceras {
			colName, _ := excelize.ColumnNumberToName(i + 1)
			f.SetCellValue(hoja, colName+"1", h)
			f.SetCellStyle(hoja, colName+"1", colName+"1", styleCabecera)
		}

		for idx, c := range conceptos {
			rowNum := idx + 2
			f.SetRowHeight(hoja, rowNum, 20)

			activo := "INACTIVO"
			if c.Activo {
				activo = "ACTIVO"
			}
			esRem := "NO"
			if c.EsRemunerativa {
				esRem = "SI"
			}
			esPen := "NO"
			if c.EsPensionable {
				esPen = "SI"
			}
			esExt := "NO"
			if c.EsExtraordinario {
				esExt = "SI"
			}

			f.SetCellValue(hoja, fmt.Sprintf("A%d", rowNum), idx+1)
			f.SetCellValue(hoja, fmt.Sprintf("B%d", rowNum), c.ConceptoCodigo)
			f.SetCellValue(hoja, fmt.Sprintf("C%d", rowNum), c.NombrePersonalizado)
			f.SetCellValue(hoja, fmt.Sprintf("D%d", rowNum), c.ConceptoTipo)
			f.SetCellValue(hoja, fmt.Sprintf("E%d", rowNum), esRem)
			f.SetCellValue(hoja, fmt.Sprintf("F%d", rowNum), esPen)
			f.SetCellValue(hoja, fmt.Sprintf("G%d", rowNum), esExt)
			f.SetCellValue(hoja, fmt.Sprintf("H%d", rowNum), c.ClasificadorCodigo)
			f.SetCellValue(hoja, fmt.Sprintf("I%d", rowNum), fmt.Sprintf("Cada %s Meses", c.FrecuenciaMeses))
			f.SetCellValue(hoja, fmt.Sprintf("J%d", rowNum), activo)

			f.SetCellStyle(hoja, fmt.Sprintf("A%d", rowNum), fmt.Sprintf("B%d", rowNum), styleCentrado)
			f.SetCellStyle(hoja, fmt.Sprintf("C%d", rowNum), fmt.Sprintf("C%d", rowNum), styleDatos)
			f.SetCellStyle(hoja, fmt.Sprintf("D%d", rowNum), fmt.Sprintf("G%d", rowNum), styleCentrado)
			f.SetCellStyle(hoja, fmt.Sprintf("H%d", rowNum), fmt.Sprintf("J%d", rowNum), styleCentrado)
		}

		s.ajustarAnchosColumnas(f, hoja, len(cabeceras))

	case "contrato_vence":
		diasStr := params["dias"]
		dias, err := strconv.Atoi(diasStr)
		if err != nil || dias <= 0 {
			dias = 30
		}

		contratos, err := s.ContratoRepo.ObtenerContratosVencimiento(tenantID, dias)
		if err != nil {
			return nil, "", fmt.Errorf("error cargando contratos: %w", err)
		}

		cabeceras := []string{"N°", "Nro Documento", "Trabajador", "Puesto", "Régimen Laboral", "Tipo Contrato", "Fecha Inicio", "Fecha Vencimiento"}
		for i, h := range cabeceras {
			colName, _ := excelize.ColumnNumberToName(i + 1)
			f.SetCellValue(hoja, colName+"1", h)
			f.SetCellStyle(hoja, colName+"1", colName+"1", styleCabecera)
		}

		for idx, c := range contratos {
			rowNum := idx + 2
			f.SetRowHeight(hoja, rowNum, 20)

			fFin := "-"
			if c.FechaFin != nil {
				fFin = *c.FechaFin
			}

			f.SetCellValue(hoja, fmt.Sprintf("A%d", rowNum), idx+1)
			f.SetCellValue(hoja, fmt.Sprintf("B%d", rowNum), c.TrabajadorDoc)
			f.SetCellValue(hoja, fmt.Sprintf("C%d", rowNum), c.TrabajadorNombre)
			f.SetCellValue(hoja, fmt.Sprintf("D%d", rowNum), c.PuestoNombre)
			f.SetCellValue(hoja, fmt.Sprintf("E%d", rowNum), c.RegimenDesc)
			f.SetCellValue(hoja, fmt.Sprintf("F%d", rowNum), c.TipoContrato)
			f.SetCellValue(hoja, fmt.Sprintf("G%d", rowNum), c.FechaInicio)
			f.SetCellValue(hoja, fmt.Sprintf("H%d", rowNum), fFin)

			f.SetCellStyle(hoja, fmt.Sprintf("A%d", rowNum), fmt.Sprintf("B%d", rowNum), styleCentrado)
			f.SetCellStyle(hoja, fmt.Sprintf("C%d", rowNum), fmt.Sprintf("E%d", rowNum), styleDatos)
			f.SetCellStyle(hoja, fmt.Sprintf("F%d", rowNum), fmt.Sprintf("H%d", rowNum), styleCentrado)
		}

		s.ajustarAnchosColumnas(f, hoja, len(cabeceras))

	default:
		return nil, "", fmt.Errorf("reporte no soportado o inválido: %s", id)
	}

	nombreArchivo := fmt.Sprintf("Reporte_%s_%s.xlsx", id, time.Now().Format("20060102"))
	var buf bytes.Buffer
	err := f.Write(&buf)
	if err != nil {
		return nil, "", fmt.Errorf("error escribiendo salida Excel: %w", err)
	}

	return &buf, nombreArchivo, nil
}

func (s *ReporteService) agregarCabeceraPDF(pdf *gofpdf.Fpdf, tenantID int, titulo string, subtitulo string) {
	tr := pdf.UnicodeTranslatorFromDescriptor("")
	pdf.SetMargins(10, 10, 10)
	
	tenant, _ := s.TenantRepo.ObtenerPorID(tenantID)
	
	if tenant != nil && tenant.LogoURL != nil && *tenant.LogoURL != "" {
		rutaLogo := strings.TrimPrefix(*tenant.LogoURL, "/")
		if _, err := os.Stat(rutaLogo); err == nil {
			pdf.ImageOptions(rutaLogo, 10, 8, 22, 0, false, gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}, 0, "")
		}
	}

	pdf.SetFont("Arial", "B", 13)
	pdf.SetXY(35, 10)
	if tenant != nil {
		pdf.Cell(100, 6, tr(tenant.Nombre))
	} else {
		pdf.Cell(100, 6, tr("MUNICIPALIDAD DISTRITAL"))
	}
	
	pdf.SetFont("Arial", "", 8)
	pdf.SetXY(35, 16)
	if tenant != nil {
		pdf.Cell(100, 4, tr("RUC: "+tenant.Ruc))
	}
	if tenant != nil && tenant.Direccion != nil {
		pdf.SetXY(35, 20)
		pdf.Cell(100, 4, tr(*tenant.Direccion))
	}

	pdf.SetFont("Arial", "I", 8)
	var anchoPagina float64
	_, altoPagina := pdf.GetPageSize()
	if altoPagina > 250 {
		anchoPagina = 210.0
	} else {
		anchoPagina = 297.0
	}
	pdf.SetXY(anchoPagina-65, 10)
	pdf.CellFormat(55, 4, tr("Fecha Impresión: "+time.Now().Format("02/01/2006 15:04")), "", 0, "R", false, 0, "")

	pdf.SetDrawColor(180, 180, 180)
	pdf.Line(10, 27, anchoPagina-10, 27)
	
	pdf.Ln(14)
	pdf.SetFont("Arial", "B", 14)
	pdf.CellFormat(anchoPagina-20, 7, tr(titulo), "", 1, "C", false, 0, "")
	
	if subtitulo != "" {
		pdf.SetFont("Arial", "I", 9)
		pdf.CellFormat(anchoPagina-20, 5, tr(subtitulo), "", 1, "C", false, 0, "")
	}
	pdf.Ln(4)
}

func (s *ReporteService) ajustarAnchosColumnas(f *excelize.File, sheet string, numCols int) {
	for i := 1; i <= numCols; i++ {
		colName, _ := excelize.ColumnNumberToName(i)
		maxLen := 10
		for r := 1; r <= 500; r++ {
			val, err := f.GetCellValue(sheet, fmt.Sprintf("%s%d", colName, r))
			if err == nil && len(val) > maxLen {
				maxLen = len(val)
			}
		}
		f.SetColWidth(sheet, colName, colName, float64(maxLen+3))
	}
}
