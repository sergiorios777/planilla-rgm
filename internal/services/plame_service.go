package services

import (
	"archive/zip"
	"bytes"
	"fmt"
	"math"
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

// MapearCodigoSunatVacaciones retorna el código de la Tabla 22 de SUNAT para remuneración vacacional según régimen laboral
func MapearCodigoSunatVacaciones(regimenCodigo string) string {
	reg := strings.TrimSpace(strings.ToUpper(regimenCodigo))
	switch reg {
	case "276", "DL 276", "DL276", "728", "DL 728", "DL728":
		return "2007"
	case "1057", "CAS", "DL 1057", "DL1057":
		return "2043"
	case "30057", "SERVIR", "LEY 30057", "LEY30057":
		return "2049"
	default:
		return "2007" // Por defecto para regímenes públicos generales
	}
}

// TransformarRemuneracionesConVacaciones segrega proporcionalmente los conceptos remunerativos para trabajadores con goce vacacional
func (s *PlameService) TransformarRemuneracionesConVacaciones(
	conceptos []models.PlameConceptoDetalle,
	diasVacacionesPorTrabajador map[int]int,
) []models.PlameRemuneracion {
	type remKey struct {
		TipoDoc string
		NumDoc  string
		Codigo  string
	}

	montosPorClave := make(map[remKey]float64)
	var ordenClaves []remKey

	for _, c := range conceptos {
		diasVac := 0
		if diasVacacionesPorTrabajador != nil {
			diasVac = diasVacacionesPorTrabajador[c.TrabajadorID]
		}
		if diasVac < 0 {
			diasVac = 0
		}
		if diasVac > 30 {
			diasVac = 30
		}

		montoTotal := c.Monto
		if montoTotal <= 0 {
			continue
		}

		var montoOrd float64
		var montoVac float64

		if diasVac == 0 {
			// Sin vacaciones: todo es ordinario
			montoOrd = montoTotal
			montoVac = 0.0
		} else if diasVac == 30 {
			// Mes completo de vacaciones
			if c.EsRemunerativa && strings.ToUpper(strings.TrimSpace(c.TipoConcepto)) == "INGRESO" {
				montoVac = montoTotal
				montoOrd = 0.0
			} else {
				montoOrd = montoTotal
				montoVac = 0.0
			}
		} else {
			// Vacaciones parciales (0 < diasVac < 30)
			if c.EsRemunerativa && strings.ToUpper(strings.TrimSpace(c.TipoConcepto)) == "INGRESO" {
				montoVac = math.Round(montoTotal*float64(diasVac)/30.0*100) / 100
				montoOrd = math.Round((montoTotal-montoVac)*100) / 100
			} else {
				montoOrd = montoTotal
				montoVac = 0.0
			}
		}

		// Acumular porción ordinaria
		if montoOrd > 0 {
			k := remKey{
				TipoDoc: c.TipoDocumento,
				NumDoc:  c.NumeroDocumento,
				Codigo:  c.CodigoConcepto,
			}
			if _, exists := montosPorClave[k]; !exists {
				ordenClaves = append(ordenClaves, k)
			}
			montosPorClave[k] += montoOrd
		}

		// Acumular porción vacacional
		if montoVac > 0 {
			codVac := MapearCodigoSunatVacaciones(c.RegimenCodigo)
			k := remKey{
				TipoDoc: c.TipoDocumento,
				NumDoc:  c.NumeroDocumento,
				Codigo:  codVac,
			}
			if _, exists := montosPorClave[k]; !exists {
				ordenClaves = append(ordenClaves, k)
			}
			montosPorClave[k] += montoVac
		}
	}

	var resultado []models.PlameRemuneracion
	for _, k := range ordenClaves {
		m := math.Round(montosPorClave[k]*100) / 100
		if m > 0 {
			resultado = append(resultado, models.PlameRemuneracion{
				TipoDocumento:   k.TipoDoc,
				NumeroDocumento: k.NumDoc,
				CodigoConcepto:  k.Codigo,
				Monto:           m,
			})
		}
	}

	return resultado
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

// GenerarSuspensionesTexto builds the content for the .snl file (Días no laborados y subsidiados - Tabla 21 SUNAT)
func (s *PlameService) GenerarSuspensionesTexto(incidencias []models.PersonalIncidenciaMes) string {
	var sb strings.Builder
	for _, inc := range incidencias {
		if inc.DiasEnMes <= 0 {
			continue
		}
		tipoDocCode := "01"
		codSusp := fmt.Sprintf("%02s", strings.TrimSpace(inc.CodigoSunatSuspension))
		line := fmt.Sprintf("%s|%s|%s|%d|\r\n", tipoDocCode, inc.TrabajadorDoc, codSusp, inc.DiasEnMes)
		sb.WriteString(line)
	}
	return sb.String()
}

// GenerarZip aggregates files into a zip archive
func (s *PlameService) GenerarZip(jornadaTxt string, remuneracionesTxt string, jorFilename string, remFilename string) ([]byte, error) {
	return s.GenerarZipCompleto(jornadaTxt, remuneracionesTxt, "", jorFilename, remFilename, "")
}

// GenerarZipCompleto aggregates jornada, remuneraciones and optional suspensiones into a zip archive
func (s *PlameService) GenerarZipCompleto(jornadaTxt string, remuneracionesTxt string, suspensionesTxt string, jorFilename string, remFilename string, snlFilename string) ([]byte, error) {
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

	// Add suspensiones file (.snl) if present
	if strings.TrimSpace(suspensionesTxt) != "" && snlFilename != "" {
		fSnl, err := w.Create(snlFilename)
		if err != nil {
			return nil, err
		}
		_, err = fSnl.Write([]byte(suspensionesTxt))
		if err != nil {
			return nil, err
		}
	}

	err = w.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
