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

		if f.Name == "file.jor" {
			if string(contentBytes) != jorText {
				t.Errorf("file.jor expected content %q, got %q", jorText, string(contentBytes))
			}
		} else if f.Name == "file.rem" {
			if string(contentBytes) != remText {
				t.Errorf("file.rem expected content %q, got %q", remText, string(contentBytes))
			}
		} else {
			t.Errorf("unexpected file in zip: %s", f.Name)
		}
	}
}
