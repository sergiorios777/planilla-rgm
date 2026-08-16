package services

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
	"strings"
	"unicode/utf8"
)

type ConceptoModeloService struct {
	Repo *repository.ConceptoModeloRepository
	Db   *sql.DB
}

func NewConceptoModeloService(repo *repository.ConceptoModeloRepository, db *sql.DB) *ConceptoModeloService {
	return &ConceptoModeloService{Repo: repo, Db: db}
}

func (s *ConceptoModeloService) ImportarDesdeCSV(file io.Reader) (exitosos int, err error) {
	// A. Cargar los tres mapas en memoria RAM
	mapaMaestros, err := s.Repo.ObtenerMapaMaestros()
	if err != nil {
		return 0, fmt.Errorf("error al cargar mapa de conceptos maestros: %w", err)
	}

	mapaClasificadores, err := s.Repo.ObtenerMapaClasificadores()
	if err != nil {
		return 0, fmt.Errorf("error al cargar mapa de clasificadores: %w", err)
	}

	mapaRegimenes, err := s.Repo.ObtenerMapaRegimenes()
	if err != nil {
		return 0, fmt.Errorf("error al cargar mapa de regímenes: %w", err)
	}

	// B. Iniciar la transacción
	tx, err := s.Db.Begin()
	if err != nil {
		return 0, fmt.Errorf("error al iniciar transacción: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// C. Leer todo el contenido a memoria para validar/convertir codificación
	content, err := io.ReadAll(file)
	if err != nil {
		return 0, fmt.Errorf("error al leer archivo de entrada: %w", err)
	}

	// Detectar si no es UTF-8 válido (como codificaciones ANSI/Windows-1252 de Excel en español)
	if !utf8.Valid(content) {
		content = convertLatin1ToUTF8(content)
	}

	// Inicializar csv.Reader
	r := csv.NewReader(bytes.NewReader(content))
	r.LazyQuotes = true
	r.FieldsPerRecord = -1 // Permitimos flexibilidad en número de columnas (mínimo 14)

	filas, err := r.ReadAll()
	if err != nil {
		return 0, fmt.Errorf("error al leer archivo CSV: %w", err)
	}

	if len(filas) < 2 {
		return 0, errors.New("el archivo CSV está vacío o no contiene suficientes filas (se requiere cabecera y al menos una fila de datos)")
	}

	// D. Iterar por cada fila del CSV
	for i, fila := range filas {
		if i == 0 {
			// Validar que la cabecera tenga al menos 16 columnas
			if len(fila) < 16 {
				return 0, fmt.Errorf("cabecera incorrecta: se requieren al menos 16 columnas, se obtuvieron %d", len(fila))
			}
			// Validar nombres específicos de columnas clave
			col0 := strings.ToLower(strings.TrimSpace(fila[0]))
			col1 := strings.ToLower(strings.TrimSpace(fila[1]))
			col15 := strings.ToLower(strings.TrimSpace(fila[15]))

			if col0 != "codigo_sunat" {
				return 0, fmt.Errorf("cabecera incorrecta: la primera columna debe ser 'codigo_sunat', se obtuvo '%s'", fila[0])
			}
			if col1 != "nombre_personalizado_unico_" {
				return 0, fmt.Errorf("cabecera incorrecta: la segunda columna debe ser 'nombre_personalizado_unico_', se obtuvo '%s'", fila[1])
			}
			if col15 != "ley_30057" {
				return 0, fmt.Errorf("cabecera incorrecta: la columna 16 debe ser 'ley_30057', se obtuvo '%s'", fila[15])
			}
			continue // Saltar cabecera
		}

		// Validar que la fila de datos tenga al menos 16 columnas
		if len(fila) < 16 {
			return 0, fmt.Errorf("fila %d: formato incorrecto, se requieren 16 columnas (codigo_sunat, nombre_personalizado_unico_, frecuencia_meses, clasificador_codigo, es_extraordinario, requiere_monto, es_pensionable, es_remunerativa, es_base_cts, es_base_beneficios_sociales, es_ocasional, es_afecto_cargas_sociales, dl_276, dl_728, dl_1057, ley_30057)", i+1)
		}

		conceptoCodigo := strings.TrimSpace(fila[0])
		if conceptoCodigo == "" {
			continue // Omitir filas vacías
		}

		nombrePersonalizado := strings.TrimSpace(fila[1])
		if nombrePersonalizado == "" {
			return 0, fmt.Errorf("fila %d: el nombre personalizado ('nombre_personalizado_unico_') no puede estar vacío", i+1)
		}

		frecuencia := strings.TrimSpace(fila[2])
		if frecuencia == "" {
			frecuencia = "1,2,3,4,5,6,7,8,9,10,11,12"
		}

		clasificadorCodigo := strings.TrimSpace(fila[3])

		// Buscar IDs correspondientes
		conceptoID, existe := mapaMaestros[conceptoCodigo]
		if !existe {
			return 0, fmt.Errorf("fila %d: el código de concepto maestro '%s' no está registrado o no se encuentra activo", i+1, conceptoCodigo)
		}

		var clasificadorID *int
		if clasificadorCodigo != "" {
			cID, existe := mapaClasificadores[clasificadorCodigo]
			if !existe {
				return 0, fmt.Errorf("fila %d: el clasificador MEF '%s' no está registrado en el catálogo", i+1, clasificadorCodigo)
			}
			clasificadorID = &cID
		}

		esOcasional := parseBoolHelper(fila[10])
		esExtraordinario := parseBoolHelper(fila[4])
		modalidad := models.ModalidadEntregaPermanente
		if esOcasional {
			modalidad = models.ModalidadEntregaOcasional
		} else if esExtraordinario {
			modalidad = models.ModalidadEntregaExcepcional
		} else if frecuencia != "1,2,3,4,5,6,7,8,9,10,11,12" && frecuencia != "" {
			modalidad = models.ModalidadEntregaPeriodico
		}

		// Construir modelo
		modelo := models.ConceptoModelo{
			ConceptoID:               conceptoID,
			NombrePersonalizado:      nombrePersonalizado,
			FrecuenciaMeses:          frecuencia,
			ClasificadorID:           clasificadorID,
			EsExtraordinario:         esExtraordinario,
			RequiereMonto:            parseBoolHelper(fila[5]),
			EsPensionable:            parseBoolHelper(fila[6]),
			EsRemunerativa:           parseBoolHelper(fila[7]),
			EsBaseCts:                parseBoolHelper(fila[8]),
			EsBaseBeneficiosSociales: parseBoolHelper(fila[9]),
			EsOcasional:              esOcasional,
			EsAfectoCargasSociales:   parseBoolHelper(fila[11]),
			ModalidadEntrega:         modalidad,
		}

		// Mapeo de regímenes
		var regimenesAAfectar []int
		regimenesValores := []struct {
			colName string
			val     string
		}{
			{"DL 276", fila[12]},
			{"DL 728", fila[13]},
			{"DL 1057", fila[14]},
			{"LEY SERVIR", fila[15]},
		}

		for _, rv := range regimenesValores {
			if parseBoolHelper(rv.val) {
				regID, existe := mapaRegimenes[rv.colName]
				if !existe {
					return 0, fmt.Errorf("fila %d: el régimen '%s' no está registrado en el catálogo", i+1, rv.colName)
				}
				regimenesAAfectar = append(regimenesAAfectar, regID)
			}
		}

		// Guardar utilizando el método transaccional del repositorio
		err = s.Repo.GuardarConceptoModeloImportado(tx, &modelo, regimenesAAfectar)
		if err != nil {
			return 0, fmt.Errorf("fila %d: error al guardar en la base de datos: %w", i+1, err)
		}

		exitosos++
	}

	// E. Confirmar la transacción
	if err = tx.Commit(); err != nil {
		return 0, fmt.Errorf("error al confirmar transacción: %w", err)
	}

	return exitosos, nil
}

func parseBoolHelper(val string) bool {
	val = strings.ToLower(strings.TrimSpace(val))
	return val == "1" || val == "true" || val == "si" || val == "sí" || val == "yes" || val == "y"
}

func convertLatin1ToUTF8(latin1 []byte) []byte {
	runes := make([]rune, len(latin1))
	for i, b := range latin1 {
		runes[i] = rune(b)
	}
	return []byte(string(runes))
}

// SembrarBaseRegimenTenant copia la configuración global de cálculo hacia el Tenant, resolviendo los IDs espejo.
// Utiliza ON CONFLICT DO NOTHING para respetar los cambios manuales y el flag 'activo' del tenant.
func (s *ConceptoModeloService) SembrarBaseRegimenTenant(tenantID int) error {
	query := `
		INSERT INTO base_regimen_tenant (tenant_id, concepto_calculado_id, regimen_id, concepto_tenant_id, variable_calculo)
		SELECT $1, brd.concepto_calculado_id, brd.regimen_id, ct.id, brd.variable_calculo
		FROM base_regimen_default brd
		INNER JOIN conceptos_tenant ct ON brd.concepto_modelo_id = ct.modelo_id
		WHERE ct.tenant_id = $1
		ON CONFLICT (tenant_id, concepto_calculado_id, regimen_id, concepto_tenant_id, variable_calculo) DO NOTHING;
	`
	_, err := s.Db.Exec(query, tenantID)
	return err
}

