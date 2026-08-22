package services

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"planilla-rgm/internal/calculadoras"
	"planilla-rgm/internal/helpers"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"

	"github.com/xuri/excelize/v2"
)

type CtsService struct {
	Repo            *repository.CtsRepository
	db              *sql.DB
	BaseRegimenRepo *repository.BaseRegimenRepository
	BaseComputable  *BaseComputableService
}

func NewCtsService(repo *repository.CtsRepository, db *sql.DB) *CtsService {
	return &CtsService{
		Repo:            repo,
		db:              db,
		BaseRegimenRepo: repository.NewBaseRegimenRepository(db),
		BaseComputable:  NewBaseComputableService(db),
	}
}

// ProcesarCtsSemestral crea un borrador de cálculo semestral para todos los trabajadores DL 728
func (s *CtsService) ProcesarCtsSemestral(tenantID int, anio int, periodo string) (int, error) {
	// 1. Determinar el rango de fechas del semestre
	var desde, hasta time.Time

	periodoUpper := strings.ToUpper(periodo)
	switch periodoUpper {
	case "MAYO":
		// Periodo: Noviembre (anio-1) a Abril (anio)
		desde = time.Date(anio-1, time.November, 1, 0, 0, 0, 0, time.UTC)
		hasta = time.Date(anio, time.April, 30, 23, 59, 59, 0, time.UTC)
	case "NOVIEMBRE":
		// Periodo: Mayo (anio) a Octubre (anio)
		desde = time.Date(anio, time.May, 1, 0, 0, 0, 0, time.UTC)
		hasta = time.Date(anio, time.October, 31, 23, 59, 59, 0, time.UTC)
	default:
		return 0, errors.New("periodo no válido, debe ser MAYO o NOVIEMBRE")
	}

	// 2. Determinar mes fiscal para el espejo
	mesFiscal := 5
	if periodoUpper == "NOVIEMBRE" {
		mesFiscal = 11
	}

	// 3. Obtener contratos DL 728 elegibles
	contratos, err := s.Repo.ObtenerContratosCtsEligibles(tenantID, desde, hasta)
	if err != nil {
		return 0, fmt.Errorf("error al obtener trabajadores elegibles de CTS: %w", err)
	}

	if len(contratos) == 0 {
		return 0, errors.New("no se encontraron trabajadores activos bajo el régimen D.L. 728 para el periodo seleccionado")
	}

	var detalles []models.PlanillaCtsDetalle

	// 4. Calcular para cada trabajador usando BaseComputableService
	for _, c := range contratos {
		desglose, err := s.BaseComputable.ResolverBaseComputable(tenantID, c.ID, c.PuestoID, c.RegimenID, "728", BeneficioCTS, hasta)

		var sueldo, familiar, sextoGrati, promVariables, remComputable float64
		if err == nil && desglose != nil && desglose.TotalComputable > 0 {
			sueldo = desglose.SueldoBasico
			familiar = desglose.AsigFamiliar
			sextoGrati = desglose.SextoGrati
			promVariables = desglose.PromedioVar
			remComputable = desglose.TotalComputable
		} else {
			// Fallback de seguridad
			sueldo = c.SueldoBasicoHistorico
			if sueldo <= 0 {
				sueldo = 1025.00
			}
			remComputable = sueldo
		}

		if sueldo <= 0 {
			sueldo = 1025.00
		}

		// Calcular meses laborados en el semestre
		mesesLaborados := helpers.CalcularMesesInterseccion(c.FechaInicio, c.FechaFin, desde, hasta)

		// Obtener inasistencias
		faltas, _ := s.Repo.ObtenerInasistenciasSemestre(c.ID, desde, hasta)

		// Calcular CTS
		_, desc, neto := calculadoras.CalcularCtsSemestralDL728(remComputable, mesesLaborados, faltas)

		d := models.PlanillaCtsDetalle{
			ContratoID:             c.ID,
			SueldoBasico:           sueldo,
			AsignacionFamilia:      familiar,
			SextoGratificacion:     sextoGrati,
			PromedioVariables:      promVariables,
			RemuneracionComputable: remComputable,
			MesesComputables:       mesesLaborados,
			DiasFaltas:             faltas,
			MontoDescuentoFaltas:   desc,
			MontoCts:               neto,
		}
		detalles = append(detalles, d)
	}

	// 5. Guardar cabeceras y detalles atómicamente en una sola transacción
	_, err = s.Repo.CrearYGuardarCtsTransaccional(tenantID, anio, periodoUpper, mesFiscal, detalles, contratos)
	if err != nil {
		return 0, err
	}

	return len(detalles), nil
}

// ProcesarExcelGratificaciones lee un archivo Excel para actualizar el sexto de gratificación en la planilla temporal
func (s *CtsService) ProcesarExcelGratificaciones(planillaCtsID int, file io.Reader) (int, error) {
	// 1. Abrir Excel
	f, err := excelize.OpenReader(file)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	hoja := f.GetSheetName(0)
	filas, err := f.GetRows(hoja)
	if err != nil {
		return 0, err
	}

	// 2. Obtener los detalles actuales de la planilla de CTS
	detalles, err := s.Repo.ObtenerDetallesCts(planillaCtsID)
	if err != nil {
		return 0, err
	}

	// Mapear documento -> detalle
	mapaDetalles := make(map[string]*models.PlanillaCtsDetalle)
	for i := range detalles {
		mapaDetalles[detalles[i].TrabajadorDocumento] = &detalles[i]
	}

	procesados := 0

	// 3. Iterar y procesar filas
	for idx, fila := range filas {
		if idx == 0 || len(fila) < 2 {
			continue // Omitir cabecera o filas cortas
		}

		doc := strings.TrimSpace(fila[0])
		montoStr := strings.TrimSpace(fila[1])
		log.Printf("Procesando fila %d: doc=%s, monto=%s", idx+1, doc, montoStr)

		det, existe := mapaDetalles[doc]
		if !existe {
			continue // No coincide con ningún trabajador DL 728
		}

		montoGrati, err := strconv.ParseFloat(montoStr, 64)
		if err != nil {
			continue // Formato inválido del monto
		}

		// Calcular el sexto
		sexto := montoGrati / 6.0
		det.SextoGratificacion = math.Round(sexto*100) / 100

		// Recalcular
		det.RemuneracionComputable = det.SueldoBasico + det.AsignacionFamilia + det.PromedioVariables + det.SextoGratificacion
		_, desc, neto := calculadoras.CalcularCtsSemestralDL728(det.RemuneracionComputable, det.MesesComputables, det.DiasFaltas)
		det.MontoDescuentoFaltas = math.Round(desc*100) / 100
		det.MontoCts = math.Round(neto*100) / 100

		// Actualizar en base de datos
		err = s.Repo.ActualizarDetalleCts(det)
		if err == nil {
			procesados++
		}
	}

	return procesados, nil
}
