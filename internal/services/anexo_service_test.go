package services

import (
	"math"
	"planilla-rgm/internal/models"
	"testing"
)

func TestCalculoRedondeoSUNATMath(t *testing.T) {
	// Prueba unitaria de la matemática de redondeo SUNAT
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
