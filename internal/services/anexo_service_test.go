package services

import (
	"math"
	"planilla-rgm/internal/models"
	"testing"
)

func TestCalculoRedondeoSUNATMath(t *testing.T) {
	casos := []struct {
		nombre          string
		montoExacto     float64
		esperadoRedond  float64
		esperadoDiferen float64
	}{
		{
			nombre:          "Redondeo hacia abajo (130.40 -> 130.00, dif -0.40)",
			montoExacto:     130.40,
			esperadoRedond:  130.00,
			esperadoDiferen: -0.40,
		},
		{
			nombre:          "Redondeo hacia arriba (130.60 -> 131.00, dif +0.40)",
			montoExacto:     130.60,
			esperadoRedond:  131.00,
			esperadoDiferen: 0.40,
		},
		{
			nombre:          "Sin ajuste (130.00 -> 130.00, dif 0.00)",
			montoExacto:     130.00,
			esperadoRedond:  130.00,
			esperadoDiferen: 0.00,
		},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			redondeado := math.Round(c.montoExacto)
			diferencia := math.Round((redondeado-c.montoExacto)*100) / 100

			if redondeado != c.esperadoRedond {
				t.Errorf("montoRedondeado esperado %.2f, obtenido %.2f", c.esperadoRedond, redondeado)
			}
			if math.Abs(diferencia-c.esperadoDiferen) > 0.001 {
				t.Errorf("diferencia esperada %.2f, obtenida %.2f", c.esperadoDiferen, diferencia)
			}
		})
	}
}

func TestGenerarAnexo1PDF(t *testing.T) {
	service := NewAnexoService(nil, nil, nil)

	data := &models.DatosAnexo1{
		TenantNombre:   "MUNICIPALIDAD DISTRITAL DE PRUEBA",
		TenantRUC:      "20123456789",
		PlanillaID:     1,
		PlanillaDesc:   "PLANILLA REGULAR CAS - JULIO 2026",
		PlanillaAnio:   2026,
		PlanillaMes:    7,
		PlanillaEstado: "CERRADA",
		Items: []models.ItemCompromisoPresupuestal{
			{
				MetaCodigo:              "0001",
				MetaDescripcion:         "GESTION ADMINISTRATIVA",
				ClasificadorCodigo:      "2.1.1.1.1.1",
				ClasificadorDescripcion: "PERSONAL CON CONTRATO A TIEMPO DETERMINADO",
				MontoTotal:              3000.00,
			},
			{
				MetaCodigo:              "0001",
				MetaDescripcion:         "GESTION ADMINISTRATIVA",
				ClasificadorCodigo:      "2.1.3.1.1.14",
				ClasificadorDescripcion: "PERSONAL CAS",
				MontoTotal:              270.00,
			},
		},
		ResumenMetas: []models.ResumenMetaCompromiso{
			{
				MetaCodigo:      "0001",
				MetaDescripcion: "GESTION ADMINISTRATIVA",
				Items: []models.ItemCompromisoPresupuestal{
					{
						MetaCodigo:              "0001",
						MetaDescripcion:         "GESTION ADMINISTRATIVA",
						ClasificadorCodigo:      "2.1.1.1.1.1",
						ClasificadorDescripcion: "PERSONAL CON CONTRATO A TIEMPO DETERMINADO",
						MontoTotal:              3000.00,
					},
					{
						MetaCodigo:              "0001",
						MetaDescripcion:         "GESTION ADMINISTRATIVA",
						ClasificadorCodigo:      "2.1.3.1.1.14",
						ClasificadorDescripcion: "PERSONAL CAS",
						MontoTotal:              270.00,
					},
				},
				TotalMeta: 3270.00,
			},
		},
		MontoTotal: 3270.00,
	}

	pdfBytes, err := service.GenerarAnexo1PDF(data)
	if err != nil {
		t.Fatalf("error al generar PDF de Anexo 1: %v", err)
	}

	if len(pdfBytes) == 0 {
		t.Errorf("el PDF generado está vacío")
	}
}

func TestGenerarAnexo1Excel(t *testing.T) {
	service := NewAnexoService(nil, nil, nil)

	data := &models.DatosAnexo1{
		TenantNombre:   "MUNICIPALIDAD DISTRITAL DE PRUEBA",
		TenantRUC:      "20123456789",
		PlanillaID:     1,
		PlanillaDesc:   "PLANILLA REGULAR CAS - JULIO 2026",
		PlanillaAnio:   2026,
		PlanillaMes:    7,
		PlanillaEstado: "CERRADA",
		Items: []models.ItemCompromisoPresupuestal{
			{
				MetaCodigo:              "0001",
				MetaDescripcion:         "GESTION ADMINISTRATIVA",
				ClasificadorCodigo:      "2.1.1.1.1.1",
				ClasificadorDescripcion: "PERSONAL CON CONTRATO A TIEMPO DETERMINADO",
				MontoTotal:              3000.00,
			},
		},
		ResumenMetas: []models.ResumenMetaCompromiso{
			{
				MetaCodigo:      "0001",
				MetaDescripcion: "GESTION ADMINISTRATIVA",
				Items: []models.ItemCompromisoPresupuestal{
					{
						MetaCodigo:              "0001",
						MetaDescripcion:         "GESTION ADMINISTRATIVA",
						ClasificadorCodigo:      "2.1.1.1.1.1",
						ClasificadorDescripcion: "PERSONAL CON CONTRATO A TIEMPO DETERMINADO",
						MontoTotal:              3000.00,
					},
				},
				TotalMeta: 3000.00,
			},
		},
		MontoTotal: 3000.00,
	}

	excelBytes, err := service.GenerarAnexo1Excel(data)
	if err != nil {
		t.Fatalf("error al generar Excel de Anexo 1: %v", err)
	}

	if len(excelBytes) == 0 {
		t.Errorf("el archivo Excel generado está vacío")
	}
}

func TestGenerarAnexo1APDF(t *testing.T) {
	service := NewAnexoService(nil, nil, nil)

	data := &models.DatosAnexo1A{
		TenantNombre:   "MUNICIPALIDAD DISTRITAL DE PRUEBA",
		TenantRUC:      "20123456789",
		PlanillaID:     1,
		PlanillaDesc:   "PLANILLA REGULAR CAS - JULIO 2026",
		PlanillaAnio:   2026,
		PlanillaMes:    7,
		PlanillaEstado: "CERRADA",
		Grupos: []models.GrupoResumenConcepto{
			{
				TipoConcepto: "INGRESO",
				Titulo:       "1. INGRESOS Y REMUNERACIONES",
				Items: []models.ItemResumenConcepto{
					{TipoConcepto: "INGRESO", CodigoSunat: "0121", NombreConcepto: "REMUNERACION CAS", MontoTotal: 3000.00},
				},
				TotalGrupo: 3000.00,
			},
			{
				TipoConcepto: "RETENCION",
				Titulo:       "2. RETENCIONES / DESCUENTOS AL TRABAJADOR",
				Items: []models.ItemResumenConcepto{
					{TipoConcepto: "RETENCION", CodigoSunat: "0607", NombreConcepto: "SNP DL 19990 - ONP", MontoTotal: 390.00},
				},
				TotalGrupo: 390.00,
			},
			{
				TipoConcepto: "APORTE",
				Titulo:       "3. APORTES DE LA ENTIDAD / EMPLEADOR",
				Items: []models.ItemResumenConcepto{
					{TipoConcepto: "APORTE", CodigoSunat: "0804", NombreConcepto: "ESSALUD CAS", MontoTotal: 270.00},
				},
				TotalGrupo: 270.00,
			},
		},
		TotalIngresos:    3000.00,
		TotalRetenciones: 390.00,
		TotalAportes:     270.00,
		CostoTotal:       3270.00,
	}

	pdfBytes, err := service.GenerarAnexo1APDF(data)
	if err != nil {
		t.Fatalf("error al generar PDF de Anexo 1A: %v", err)
	}

	if len(pdfBytes) == 0 {
		t.Errorf("el PDF de Anexo 1A generado está vacío")
	}
}

func TestGenerarAnexo1AExcel(t *testing.T) {
	service := NewAnexoService(nil, nil, nil)

	data := &models.DatosAnexo1A{
		TenantNombre:   "MUNICIPALIDAD DISTRITAL DE PRUEBA",
		TenantRUC:      "20123456789",
		PlanillaID:     1,
		PlanillaDesc:   "PLANILLA REGULAR CAS - JULIO 2026",
		PlanillaAnio:   2026,
		PlanillaMes:    7,
		PlanillaEstado: "CERRADA",
		Grupos: []models.GrupoResumenConcepto{
			{
				TipoConcepto: "INGRESO",
				Titulo:       "1. INGRESOS Y REMUNERACIONES",
				Items: []models.ItemResumenConcepto{
					{TipoConcepto: "INGRESO", CodigoSunat: "0121", NombreConcepto: "REMUNERACION CAS", MontoTotal: 3000.00},
				},
				TotalGrupo: 3000.00,
			},
		},
		TotalIngresos:    3000.00,
		TotalRetenciones: 0.00,
		TotalAportes:     270.00,
		CostoTotal:       3270.00,
	}

	excelBytes, err := service.GenerarAnexo1AExcel(data)
	if err != nil {
		t.Fatalf("error al generar Excel de Anexo 1A: %v", err)
	}

	if len(excelBytes) == 0 {
		t.Errorf("el archivo Excel de Anexo 1A generado está vacío")
	}
}

func TestGenerarAnexo2PDF(t *testing.T) {
	service := NewAnexoService(nil, nil, nil)

	data := &models.DatosAnexo2{
		TenantNombre:   "MUNICIPALIDAD DISTRITAL DE PRUEBA",
		TenantRUC:      "20123456789",
		PlanillaID:     1,
		PlanillaDesc:   "PLANILLA REGULAR CAS - JULIO 2026",
		PlanillaAnio:   2026,
		PlanillaMes:    7,
		PlanillaEstado: "CERRADA",
		Items: []models.ItemResumenAFP{
			{AFPNombre: "HABITAT", AporteObligatorio: 900.00, Comision: 0.00, PrimaSeguro: 34.98, TotalAFP: 934.98},
			{AFPNombre: "INTEGRA", AporteObligatorio: 2105.40, Comision: 0.00, PrimaSeguro: 93.36, TotalAFP: 2198.76},
			{AFPNombre: "PRIMA", AporteObligatorio: 360.00, Comision: 18.60, PrimaSeguro: 31.80, TotalAFP: 410.40},
			{AFPNombre: "PROFUTURO", AporteObligatorio: 840.00, Comision: 26.88, PrimaSeguro: 56.85, TotalAFP: 923.73},
		},
		TotalAporteObligatorio: 4205.40,
		TotalComision:          45.48,
		TotalPrimaSeguro:       216.99,
		GranTotal:              4467.87,
	}

	pdfBytes, err := service.GenerarAnexo2PDF(data)
	if err != nil {
		t.Fatalf("error al generar PDF de Anexo 2: %v", err)
	}

	if len(pdfBytes) == 0 {
		t.Errorf("el PDF de Anexo 2 generado está vacío")
	}
}

func TestGenerarAnexo2Excel(t *testing.T) {
	service := NewAnexoService(nil, nil, nil)

	data := &models.DatosAnexo2{
		TenantNombre:   "MUNICIPALIDAD DISTRITAL DE PRUEBA",
		TenantRUC:      "20123456789",
		PlanillaID:     1,
		PlanillaDesc:   "PLANILLA REGULAR CAS - JULIO 2026",
		PlanillaAnio:   2026,
		PlanillaMes:    7,
		PlanillaEstado: "CERRADA",
		Items: []models.ItemResumenAFP{
			{AFPNombre: "HABITAT", AporteObligatorio: 900.00, Comision: 0.00, PrimaSeguro: 34.98, TotalAFP: 934.98},
		},
		TotalAporteObligatorio: 900.00,
		TotalComision:          0.00,
		TotalPrimaSeguro:       34.98,
		GranTotal:              934.98,
	}

	excelBytes, err := service.GenerarAnexo2Excel(data)
	if err != nil {
		t.Fatalf("error al generar Excel de Anexo 2: %v", err)
	}

	if len(excelBytes) == 0 {
		t.Errorf("el archivo Excel de Anexo 2 generado está vacío")
	}
}

func TestGenerarAnexo2APDF(t *testing.T) {
	service := NewAnexoService(nil, nil, nil)

	data := &models.DatosAnexo2A{
		TenantNombre:   "MUNICIPALIDAD DISTRITAL DE PRUEBA",
		TenantRUC:      "20123456789",
		PlanillaID:     1,
		PlanillaDesc:   "PLANILLA REGULAR CAS - JULIO 2026",
		PlanillaAnio:   2026,
		PlanillaMes:    7,
		PlanillaEstado: "CERRADA",
		Grupos: []models.GrupoDevengadoAFP{
			{
				AFPNombre: "HABITAT",
				Items: []models.ItemDevengadoAFP{
					{
						AFPNombre:               "HABITAT",
						MetaCodigo:              "0002",
						ClasificadorCodigo:      "2.1.1.1.1.2",
						ClasificadorDescripcion: "PERSONAL ADMINISTRATIVO NOMBRADO",
						AporteObligatorio:       900.00,
						Comision:                0.00,
						PrimaSeguro:             34.98,
						TotalFila:               934.98,
					},
				},
				TotalAporteObligatorio: 900.00,
				TotalComision:          0.00,
				TotalPrimaSeguro:       34.98,
				TotalGrupo:             934.98,
			},
		},
		TotalAporteObligatorio: 900.00,
		TotalComision:          0.00,
		TotalPrimaSeguro:       34.98,
		GranTotal:              934.98,
	}

	pdfBytes, err := service.GenerarAnexo2APDF(data)
	if err != nil {
		t.Fatalf("error al generar PDF de Anexo 2A: %v", err)
	}

	if len(pdfBytes) == 0 {
		t.Errorf("el PDF de Anexo 2A generado está vacío")
	}
}

func TestGenerarAnexo2AExcel(t *testing.T) {
	service := NewAnexoService(nil, nil, nil)

	data := &models.DatosAnexo2A{
		TenantNombre:   "MUNICIPALIDAD DISTRITAL DE PRUEBA",
		TenantRUC:      "20123456789",
		PlanillaID:     1,
		PlanillaDesc:   "PLANILLA REGULAR CAS - JULIO 2026",
		PlanillaAnio:   2026,
		PlanillaMes:    7,
		PlanillaEstado: "CERRADA",
		Grupos: []models.GrupoDevengadoAFP{
			{
				AFPNombre: "HABITAT",
				Items: []models.ItemDevengadoAFP{
					{
						AFPNombre:               "HABITAT",
						MetaCodigo:              "0002",
						ClasificadorCodigo:      "2.1.1.1.1.2",
						ClasificadorDescripcion: "PERSONAL ADMINISTRATIVO NOMBRADO",
						AporteObligatorio:       900.00,
						Comision:                0.00,
						PrimaSeguro:             34.98,
						TotalFila:               934.98,
					},
				},
				TotalAporteObligatorio: 900.00,
				TotalComision:          0.00,
				TotalPrimaSeguro:       34.98,
				TotalGrupo:             934.98,
			},
		},
		TotalAporteObligatorio: 900.00,
		TotalComision:          0.00,
		TotalPrimaSeguro:       34.98,
		GranTotal:              934.98,
	}

	excelBytes, err := service.GenerarAnexo2AExcel(data)
	if err != nil {
		t.Fatalf("error al generar Excel de Anexo 2A: %v", err)
	}

	if len(excelBytes) == 0 {
		t.Errorf("el archivo Excel de Anexo 2A generado está vacío")
	}
}

func TestGenerarAnexo3PDF(t *testing.T) {
	service := NewAnexoService(nil, nil, nil)

	data := &models.DatosAnexo3{
		TenantNombre:   "MUNICIPALIDAD DISTRITAL DE PRUEBA",
		TenantRUC:      "20123456789",
		PlanillaID:     1,
		PlanillaDesc:   "PLANILLA REGULAR CAS - JULIO 2026",
		PlanillaAnio:   2026,
		PlanillaMes:    7,
		PlanillaEstado: "CERRADA",
		Items: []models.ItemRetencionesSunat{
			{
				MetaCodigo:              "0002",
				ClasificadorCodigo:      "2.1.1.1.1.2",
				ClasificadorDescripcion: "PERSONAL ADMINISTRATIVO NOMBRADO",
				ONP:                     900.00,
				Renta4ta:                0.00,
				Renta5ta:                34.98,
				TotalFila:               934.98,
			},
		},
		TotalONP:      900.00,
		TotalRenta4ta: 0.00,
		TotalRenta5ta: 34.98,
		GranTotal:     934.98,
	}

	pdfBytes, err := service.GenerarAnexo3PDF(data)
	if err != nil {
		t.Fatalf("error al generar PDF de Anexo 3: %v", err)
	}

	if len(pdfBytes) == 0 {
		t.Errorf("el PDF de Anexo 3 generado está vacío")
	}
}

func TestGenerarAnexo3Excel(t *testing.T) {
	service := NewAnexoService(nil, nil, nil)

	data := &models.DatosAnexo3{
		TenantNombre:   "MUNICIPALIDAD DISTRITAL DE PRUEBA",
		TenantRUC:      "20123456789",
		PlanillaID:     1,
		PlanillaDesc:   "PLANILLA REGULAR CAS - JULIO 2026",
		PlanillaAnio:   2026,
		PlanillaMes:    7,
		PlanillaEstado: "CERRADA",
		Items: []models.ItemRetencionesSunat{
			{
				MetaCodigo:              "0002",
				ClasificadorCodigo:      "2.1.1.1.1.2",
				ClasificadorDescripcion: "PERSONAL ADMINISTRATIVO NOMBRADO",
				ONP:                     900.00,
				Renta4ta:                0.00,
				Renta5ta:                34.98,
				TotalFila:               934.98,
			},
		},
		TotalONP:      900.00,
		TotalRenta4ta: 0.00,
		TotalRenta5ta: 34.98,
		GranTotal:     934.98,
	}

	excelBytes, err := service.GenerarAnexo3Excel(data)
	if err != nil {
		t.Fatalf("error al generar Excel de Anexo 3: %v", err)
	}

	if len(excelBytes) == 0 {
		t.Errorf("el archivo Excel de Anexo 3 generado está vacío")
	}
}
