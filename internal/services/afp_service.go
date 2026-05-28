package services

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
	"strconv"
	"strings"
)

// AFPService expone la lógica de negocio para gestionar AFPs e importaciones
type AFPService struct {
	Repo *repository.AFPRepository
}

// NewAFPService crea un nuevo servicio de AFP
func NewAFPService(repo *repository.AFPRepository) *AFPService {
	return &AFPService{Repo: repo}
}

// ProcesarCSV procesa el archivo subido por el super admin e importa las tasas a la BD
func (s *AFPService) ProcesarCSV(file io.Reader, anio int, mes int) error {
	// 1. Obtener todas las AFPs para validar nombres e IDs
	afps, err := s.Repo.ObtenerTodos("")
	if err != nil {
		return fmt.Errorf("error al leer catálogo de AFPs: %w", err)
	}

	// Mapeamos Nombre -> ID
	mapaAFPs := make(map[string]int)
	for _, a := range afps {
		if a.Activo {
			mapaAFPs[strings.ToUpper(strings.TrimSpace(a.Nombre))] = a.ID
		}
	}

	// 2. Parsear el archivo CSV
	reader := csv.NewReader(file)
	reader.LazyQuotes = true

	filas, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("error al leer archivo CSV: %w", err)
	}

	if len(filas) < 2 {
		return errors.New("el archivo CSV está vacío o no tiene suficientes filas")
	}

	var tasas []models.AFPTasaMensual

	for i, fila := range filas {
		if i == 0 {
			continue // Saltar cabecera
		}

		// Validar columnas
		if len(fila) < 5 {
			return fmt.Errorf("fila %d: formato incorrecto, se requieren al menos 5 columnas (AFP, Comisión Flujo, Comisión Saldo, Prima, Aporte)", i+1)
		}

		nombreAFP := strings.ToUpper(strings.TrimSpace(fila[0]))
		if nombreAFP == "" {
			continue // Ignorar líneas vacías
		}

		afpID, existe := mapaAFPs[nombreAFP]
		if !existe {
			return fmt.Errorf("fila %d: la AFP '%s' no está registrada o no se encuentra activa en el sistema", i+1, fila[0])
		}

		// Parsear tasas individuales
		comisionFlujo, err := parseRate(fila[1])
		if err != nil {
			return fmt.Errorf("fila %d: error en Comisión Flujo '%s': %w", i+1, fila[1], err)
		}

		comisionAnualSaldo, err := parseRate(fila[2])
		if err != nil {
			return fmt.Errorf("fila %d: error en Comisión Anual Saldo '%s': %w", i+1, fila[2], err)
		}

		primaSeguro, err := parseRate(fila[3])
		if err != nil {
			return fmt.Errorf("fila %d: error en Prima Seguro '%s': %w", i+1, fila[3], err)
		}

		aporteObligatorio, err := parseRate(fila[4])
		if err != nil {
			return fmt.Errorf("fila %d: error en Aporte Obligatorio '%s': %w", i+1, fila[4], err)
		}

		// Comisión Mixta Flujo: por defecto 0.0 (regla vigente desde Feb 2023),
		// pero si se proporciona como 6ta columna, la leemos.
		comisionMixtaFlujo := 0.0
		if len(fila) >= 6 && fila[5] != "" {
			comisionMixtaFlujo, err = parseRate(fila[5])
			if err != nil {
				return fmt.Errorf("fila %d: error en Comisión Mixta Flujo '%s': %w", i+1, fila[5], err)
			}
		}

		tasas = append(tasas, models.AFPTasaMensual{
			AfpID:              afpID,
			Anio:               anio,
			Mes:                mes,
			AporteObligatorio:   aporteObligatorio,
			ComisionFlujo:       comisionFlujo,
			ComisionMixtaFlujo: comisionMixtaFlujo,
			PrimaSeguro:         primaSeguro,
			ComisionAnualSaldo: comisionAnualSaldo,
		})
	}

	if len(tasas) == 0 {
		return errors.New("no se encontraron tasas válidas para importar")
	}

	// 3. Persistir en base de datos (con lógica UPSERT en el repositorio)
	if err := s.Repo.GuardarTasasMensuales(tasas); err != nil {
		return fmt.Errorf("error al guardar tasas en la BD: %w", err)
	}

	return nil
}

// parseRate limpia y convierte un string a float64.
// Soporta formatos directos decimales (0.0147), porcentajes numéricos (1.47) y con '%' (1.47%).
func parseRate(raw string) (float64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "-" || raw == "0" {
		return 0.0, nil
	}

	hasPercent := false
	if strings.Contains(raw, "%") {
		hasPercent = true
		raw = strings.ReplaceAll(raw, "%", "")
	}

	// Permitir comas en vez de puntos
	raw = strings.ReplaceAll(raw, ",", ".")

	val, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0.0, err
	}

	// Si se detecta símbolo de porcentaje '%' o si el valor ingresado es > 0.30,
	// se infiere que se ingresó como porcentaje (ej. 1.47 o 10.0 en lugar de 0.0147 o 0.1).
	// Los valores legales en el SPP de Perú no superan el 30% (0.30) para estos conceptos.
	if hasPercent || val > 0.30 {
		val = val / 100.0
	}

	return val, nil
}
