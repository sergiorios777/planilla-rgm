package services

import (
	"archive/zip"
	"bytes"
	"fmt"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
	"strings"
)

type PlameService struct {
	Repo *repository.PlanillaRepository
}

func NewPlameService(repo *repository.PlanillaRepository) *PlameService {
	return &PlameService{Repo: repo}
}

// Helper to map document types to SUNAT codes (01=DNI, 04=CE, 07=Pasaporte)
func mapTipoDocumento(tipo string) string {
	tipo = strings.ToUpper(strings.TrimSpace(tipo))
	switch tipo {
	case "DNI":
		return "01"
	case "CE", "CARNET DE EXTRANJERIA", "CARNÉ DE EXTRANJERÍA", "CARNÉ EXTRANJERÍA", "CARNET EXTRANJERIA", "CARNET DE EXTRANJERÍA":
		return "04"
	case "PASAPORTE":
		return "07"
	default:
		return "01" // fallback to DNI
	}
}

// GenerarJornadaTexto builds the content for the .jor file
func (s *PlameService) GenerarJornadaTexto(datos []models.PlameJornada) string {
	var sb strings.Builder
	for _, j := range datos {
		tipoDocCode := mapTipoDocumento(j.TipoDocumento)
		horasOrd := 240 - int(j.DiasInasistencia)*8
		if horasOrd < 0 {
			horasOrd = 0
		}
		// Col 1: Tipo de Documento
		// Col 2: Número de Documento
		// Col 3: Horas Ordinarias (max 240)
		// Col 4: Minutos Ordinarios (00)
		// Col 5: Horas Sobretiempo (000)
		// Col 6: Minutos Sobretiempo (00)
		// Col 7: Días No Laborados / Inasistencias
		line := fmt.Sprintf("%s|%s|%d|00|000|00|%02d|\r\n", tipoDocCode, j.NumeroDocumento, horasOrd, int(j.DiasInasistencia))
		sb.WriteString(line)
	}
	return sb.String()
}

// GenerarRemuneracionesTexto builds the content for the .rem file
func (s *PlameService) GenerarRemuneracionesTexto(datos []models.PlameRemuneracion) string {
	var sb strings.Builder
	for _, r := range datos {
		tipoDocCode := mapTipoDocumento(r.TipoDocumento)
		// Col 1: Tipo de Documento
		// Col 2: Número de Documento
		// Col 3: Código de Concepto
		// Col 4: Monto Devengado
		// Col 5: Monto Pagado
		line := fmt.Sprintf("%s|%s|%s|%.2f|%.2f|\r\n", tipoDocCode, r.NumeroDocumento, r.CodigoConcepto, r.Monto, r.Monto)
		sb.WriteString(line)
	}
	return sb.String()
}

// GenerarZip aggregates both files into a zip archive
func (s *PlameService) GenerarZip(jornadaTxt string, remuneracionesTxt string, jorFilename string, remFilename string) ([]byte, error) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	// Add jornada file
	fJor, err := w.Create(jorFilename)
	if err != nil {
		return nil, err
	}
	_, err = fJor.Write([]byte(jornadaTxt))
	if err != nil {
		return nil, err
	}

	// Add remuneraciones file
	fRem, err := w.Create(remFilename)
	if err != nil {
		return nil, err
	}
	_, err = fRem.Write([]byte(remuneracionesTxt))
	if err != nil {
		return nil, err
	}

	err = w.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
