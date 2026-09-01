package services

import (
	"archive/zip"
	"bytes"
	"io"
	"planilla-rgm/internal/models"
	"testing"
)

func TestGenerarJornadaTexto(t *testing.T) {
	service := NewPlameService(nil)

	datos := []models.PlameJornada{
		{
			TipoDocumento:    "DNI",
			NumeroDocumento:  "44556677",
			DiasInasistencia: 0,
		},
		{
			TipoDocumento:    "CE",
			NumeroDocumento:  "88776655",
			DiasInasistencia: 6,
		},
	}

	expected := "01|44556677|240|00|000|00|00|\r\n" +
		"04|88776655|192|00|000|00|06|\r\n"

	result := service.GenerarJornadaTexto(datos)
	if result != expected {
		t.Errorf("expected:\n%q\ngot:\n%q", expected, result)
	}
}

func TestGenerarRemuneracionesTexto(t *testing.T) {
	service := NewPlameService(nil)

	datos := []models.PlameRemuneracion{
		{
			TipoDocumento:   "DNI",
			NumeroDocumento: "44556677",
			CodigoConcepto:  "0121",
			Monto:           1500.00,
		},
		{
			TipoDocumento:   "DNI",
			NumeroDocumento: "44556677",
			CodigoConcepto:  "0201",
			Monto:           102.50,
		},
	}

	expected := "01|44556677|0121|1500.00|1500.00|\r\n" +
		"01|44556677|0201|102.50|102.50|\r\n"

	result := service.GenerarRemuneracionesTexto(datos)
	if result != expected {
		t.Errorf("expected:\n%q\ngot:\n%q", expected, result)
	}
}

func TestGenerarZip(t *testing.T) {
	service := NewPlameService(nil)

	jorText := "jor_content"
	remText := "rem_content"

	zipBytes, err := service.GenerarZip(jorText, remText, "file.jor", "file.rem")
	if err != nil {
		t.Fatalf("failed to generate zip: %v", err)
	}

	// Read zip back
	r, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		t.Fatalf("failed to read zip: %v", err)
	}

	if len(r.File) != 2 {
		t.Errorf("expected 2 files in zip, got %d", len(r.File))
	}

	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("failed to open file %s: %v", f.Name, err)
		}
		defer rc.Close()

		contentBytes, err := io.ReadAll(rc)
		if err != nil {
			t.Fatalf("failed to read file %s: %v", f.Name, err)
		}

		switch f.Name {
		case "file.jor":
			if string(contentBytes) != jorText {
				t.Errorf("file.jor expected content %q, got %q", jorText, string(contentBytes))
			}
		case "file.rem":
			if string(contentBytes) != remText {
				t.Errorf("file.rem expected content %q, got %q", remText, string(contentBytes))
			}
		default:
			t.Errorf("unexpected file in zip: %s", f.Name)
		}
	}
}

func TestGenerarSuspensionesTexto(t *testing.T) {
	service := NewPlameService(nil)

	incidencias := []models.PersonalIncidenciaMes{
		{
			TrabajadorDoc:         "12345678",
			CodigoSunatSuspension: "23",
			DiasEnMes:             15,
		},
		{
			TrabajadorDoc:         "87654321",
			CodigoSunatSuspension: "05",
			DiasEnMes:             5,
		},
	}

	expected := "01|12345678|23|15|\r\n" +
		"01|87654321|05|5|\r\n"

	result := service.GenerarSuspensionesTexto(incidencias)
	if result != expected {
		t.Errorf("expected:\n%q\ngot:\n%q", expected, result)
	}
}

func TestGenerarZipCompleto(t *testing.T) {
	service := NewPlameService(nil)

	jorText := "jor_content"
	remText := "rem_content"
	snlText := "01|12345678|23|15|\r\n"

	zipBytes, err := service.GenerarZipCompleto(jorText, remText, snlText, "file.jor", "file.rem", "file.snl")
	if err != nil {
		t.Fatalf("failed to generate zip: %v", err)
	}

	r, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		t.Fatalf("failed to read zip: %v", err)
	}

	if len(r.File) != 3 {
		t.Errorf("expected 3 files in zip, got %d", len(r.File))
	}
}

func TestMapearCodigoSunatVacaciones(t *testing.T) {
	tests := []struct {
		regimen string
		want    string
	}{
		{"276", "2007"},
		{"DL 276", "2007"},
		{"728", "2007"},
		{"DL 728", "2007"},
		{"1057", "2043"},
		{"CAS", "2043"},
		{"30057", "2049"},
		{"SERVIR", "2049"},
		{"OTRO", "2007"},
	}

	for _, tt := range tests {
		got := MapearCodigoSunatVacaciones(tt.regimen)
		if got != tt.want {
			t.Errorf("MapearCodigoSunatVacaciones(%q) = %q; want %q", tt.regimen, got, tt.want)
		}
	}
}

func TestTransformarRemuneraciones_SinVacaciones(t *testing.T) {
	service := NewPlameService(nil)

	conceptos := []models.PlameConceptoDetalle{
		{
			TrabajadorID:    1,
			TipoDocumento:   "DNI",
			NumeroDocumento: "10203040",
			CodigoConcepto:  "0121",
			TipoConcepto:    "INGRESO",
			EsRemunerativa:  true,
			RegimenCodigo:   "1057",
			Monto:           2500.00,
		},
	}

	diasVacMap := map[int]int{
		1: 0,
	}

	res := service.TransformarRemuneracionesConVacaciones(conceptos, diasVacMap)
	if len(res) != 1 {
		t.Fatalf("expected 1 record, got %d", len(res))
	}
	if res[0].CodigoConcepto != "0121" || res[0].Monto != 2500.00 {
		t.Errorf("expected 0121: 2500.00, got %s: %.2f", res[0].CodigoConcepto, res[0].Monto)
	}
}

func TestTransformarRemuneraciones_VacacionesParciales_CAS_1057(t *testing.T) {
	service := NewPlameService(nil)

	// Trabajador CAS con 15 días de vacaciones
	// Básico S/ 3000.00 (Remunerativo) -> 15 días ord = S/ 1500.00 (0121), 15 días vac = S/ 1500.00 (2043)
	// Asig Fam S/ 102.50 (Remunerativo) -> 15 días ord = S/ 51.25 (0201), 15 días vac = S/ 51.25 (2043)
	conceptos := []models.PlameConceptoDetalle{
		{
			TrabajadorID:    1,
			TipoDocumento:   "DNI",
			NumeroDocumento: "10203040",
			CodigoConcepto:  "0121",
			TipoConcepto:    "INGRESO",
			EsRemunerativa:  true,
			RegimenCodigo:   "1057",
			Monto:           3000.00,
		},
		{
			TrabajadorID:    1,
			TipoDocumento:   "DNI",
			NumeroDocumento: "10203040",
			CodigoConcepto:  "0201",
			TipoConcepto:    "INGRESO",
			EsRemunerativa:  true,
			RegimenCodigo:   "1057",
			Monto:           102.50,
		},
	}

	diasVacMap := map[int]int{
		1: 15,
	}

	res := service.TransformarRemuneracionesConVacaciones(conceptos, diasVacMap)
	// Debe consolidar: 0121 = 1500.00, 0201 = 51.25, 2043 = 1551.25
	if len(res) != 3 {
		t.Fatalf("expected 3 records, got %d: %+v", len(res), res)
	}

	mapaResultados := make(map[string]float64)
	for _, r := range res {
		mapaResultados[r.CodigoConcepto] = r.Monto
	}

	if mapaResultados["0121"] != 1500.00 {
		t.Errorf("expected 0121 = 1500.00, got %.2f", mapaResultados["0121"])
	}
	if mapaResultados["0201"] != 51.25 {
		t.Errorf("expected 0201 = 51.25, got %.2f", mapaResultados["0201"])
	}
	if mapaResultados["2043"] != 1551.25 {
		t.Errorf("expected 2043 = 1551.25, got %.2f", mapaResultados["2043"])
	}
}

func TestTransformarRemuneraciones_VacacionesParciales_276_ConNoRemunerativo(t *testing.T) {
	service := NewPlameService(nil)

	// Trabajador 276 con 10 días de vacaciones
	// Básico S/ 1000.00 (Remunerativo) -> 10/30 = 333.33 vac (2007), 666.67 ord (0121)
	// Movilidad S/ 200.00 (No Remunerativo) -> 200.00 ord (0909) intacto
	// Aporte EsSalud S/ 90.00 (Aporte) -> 90.00 (0804) intacto
	conceptos := []models.PlameConceptoDetalle{
		{
			TrabajadorID:    2,
			TipoDocumento:   "DNI",
			NumeroDocumento: "20304050",
			CodigoConcepto:  "0121",
			TipoConcepto:    "INGRESO",
			EsRemunerativa:  true,
			RegimenCodigo:   "276",
			Monto:           1000.00,
		},
		{
			TrabajadorID:    2,
			TipoDocumento:   "DNI",
			NumeroDocumento: "20304050",
			CodigoConcepto:  "0909",
			TipoConcepto:    "INGRESO",
			EsRemunerativa:  false,
			RegimenCodigo:   "276",
			Monto:           200.00,
		},
		{
			TrabajadorID:    2,
			TipoDocumento:   "DNI",
			NumeroDocumento: "20304050",
			CodigoConcepto:  "0804",
			TipoConcepto:    "APORTE",
			EsRemunerativa:  false,
			RegimenCodigo:   "276",
			Monto:           90.00,
		},
	}

	diasVacMap := map[int]int{
		2: 10,
	}

	res := service.TransformarRemuneracionesConVacaciones(conceptos, diasVacMap)
	mapaResultados := make(map[string]float64)
	for _, r := range res {
		mapaResultados[r.CodigoConcepto] = r.Monto
	}

	if mapaResultados["0121"] != 666.67 {
		t.Errorf("expected 0121 = 666.67, got %.2f", mapaResultados["0121"])
	}
	if mapaResultados["2007"] != 333.33 {
		t.Errorf("expected 2007 = 333.33, got %.2f", mapaResultados["2007"])
	}
	if mapaResultados["0909"] != 200.00 {
		t.Errorf("expected 0909 = 200.00, got %.2f", mapaResultados["0909"])
	}
	if mapaResultados["0804"] != 90.00 {
		t.Errorf("expected 0804 = 90.00, got %.2f", mapaResultados["0804"])
	}
}

func TestTransformarRemuneraciones_MesCompleto_Ley30057(t *testing.T) {
	service := NewPlameService(nil)

	// Trabajador Servir con 30 días de vacaciones
	// Básico S/ 4000.00 (Remunerativo) -> 100% vacacional a código 2049
	conceptos := []models.PlameConceptoDetalle{
		{
			TrabajadorID:    3,
			TipoDocumento:   "DNI",
			NumeroDocumento: "30405060",
			CodigoConcepto:  "0121",
			TipoConcepto:    "INGRESO",
			EsRemunerativa:  true,
			RegimenCodigo:   "30057",
			Monto:           4000.00,
		},
	}

	diasVacMap := map[int]int{
		3: 30,
	}

	res := service.TransformarRemuneracionesConVacaciones(conceptos, diasVacMap)
	if len(res) != 1 {
		t.Fatalf("expected 1 record, got %d: %+v", len(res), res)
	}
	if res[0].CodigoConcepto != "2049" || res[0].Monto != 4000.00 {
		t.Errorf("expected 2049 = 4000.00, got %s = %.2f", res[0].CodigoConcepto, res[0].Monto)
	}
}

