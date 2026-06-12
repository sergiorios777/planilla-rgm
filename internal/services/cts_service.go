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
	"planilla-rgm/internal/config"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"

	"github.com/xuri/excelize/v2"
)

type CtsService struct {
	Repo *repository.CtsRepository
	db   *sql.DB
}

func NewCtsService(repo *repository.CtsRepository, db *sql.DB) *CtsService {
	return &CtsService{
		Repo: repo,
		db:   db,
	}
}

// ProcesarCtsSemestral crea un borrador de cálculo semestral para todos los trabajadores DL 728
func (s *CtsService) ProcesarCtsSemestral(tenantID int, anio int, periodo string) (int, error) {
	// 1. Determinar el rango de fechas del semestre
	var desde, hasta time.Time
	var anio1, anio2 int
	var meses1, meses2 []int

	periodoUpper := strings.ToUpper(periodo)
	switch periodoUpper {
	case "MAYO":
		// Periodo: Noviembre (anio-1) a Abril (anio)
		desde = time.Date(anio-1, time.November, 1, 0, 0, 0, 0, time.UTC)
		hasta = time.Date(anio, time.April, 30, 23, 59, 59, 0, time.UTC)
		anio1 = anio - 1
		meses1 = []int{11, 12}
		anio2 = anio
		meses2 = []int{1, 2, 3, 4}
	case "NOVIEMBRE":
		// Periodo: Mayo (anio) a Octubre (anio)
		desde = time.Date(anio, time.May, 1, 0, 0, 0, 0, time.UTC)
		hasta = time.Date(anio, time.October, 31, 23, 59, 59, 0, time.UTC)
		anio1 = anio
		meses1 = []int{5, 6, 7, 8, 9, 10}
		anio2 = anio // Se busca en el mismo año
		meses2 = []int{}
	default:
		return 0, errors.New("periodo no válido, debe ser MAYO o NOVIEMBRE")
	}

	// 2. Crear la planilla cabecera
	p := &models.PlanillaCts{
		TenantID: tenantID,
		Anio:     anio,
		Periodo:  periodoUpper,
		Estado:   "BORRADOR",
	}
	err := s.Repo.CrearPlanillaCts(p)
	if err != nil {
		return 0, fmt.Errorf("error al crear planilla cabecera de CTS: %w", err)
	}

	// 3. Obtener contratos DL 728 elegibles
	contratos, err := s.Repo.ObtenerContratosCtsEligibles(tenantID, desde, hasta)
	if err != nil {
		return 0, fmt.Errorf("error al obtener trabajadores elegibles de CTS: %w", err)
	}

	var detalles []models.PlanillaCtsDetalle

	// Cargar llaves y conceptos del catálogo
	codigosSueldo := config.ConceptosMestrosCTS["DL 728"]["remuneracion"]
	codigosAsigFam := config.ConceptosMestrosCTS["DL 728"]["asignacion_familiar"]
	codigosGrati := config.ConceptosMestrosCTS["DL 728"]["gratificacion"]

	// Preparar exclusión para variables
	var codigosExcluidos []string
	codigosExcluidos = append(codigosExcluidos, codigosSueldo...)
	codigosExcluidos = append(codigosExcluidos, codigosAsigFam...)

	// 4. Calcular para cada trabajador
	for _, c := range contratos {
		// Obtener sueldo básico oficial de puesto_conceptos o fallback al presupuestado
		sueldo, err := s.Repo.ObtenerSueldoBasicoActivo(c.PuestoID, codigosSueldo)
		if err != nil || sueldo <= 0 {
			sueldo = c.SueldoBasicoHistorico
		}
		if sueldo <= 0 {
			sueldo = 1025.00 // Sueldo mínimo referencial
		}

		// Asignación familiar dinámica basada en config
		familiar, _ := s.Repo.ObtenerRemuneracionFamiliarActiva(c.PuestoID, codigosAsigFam)

		// 1/6 de Gratificación anterior basada en config
		var grati float64
		if periodoUpper == "MAYO" {
			grati, _ = s.Repo.ObtenerGratificacionHistorica(c.ID, anio-1, 12, codigosGrati) // Diciembre
		} else {
			grati, _ = s.Repo.ObtenerGratificacionHistorica(c.ID, anio, 7, codigosGrati) // Julio
		}
		sextoGrati := grati / 6.0

		// Promedio de variables excluyendo sueldo y asignación familiar
		promVariables := 0.0
		vars, err := s.Repo.ObtenerVariablesSemestre(c.ID, anio1, meses1, anio2, meses2, codigosExcluidos)
		if err == nil && len(vars) > 0 {
			conceptosSum := make(map[int]float64)
			conceptosCount := make(map[int]int)
			for _, v := range vars {
				conceptosSum[v.MaestroID] += v.Monto
				conceptosCount[v.MaestroID]++
			}
			for maestroID, count := range conceptosCount {
				if count >= 3 { // Regla de regularidad
					promVariables += conceptosSum[maestroID] / 6.0
				}
			}
		}

		// Calcular meses laborados en el semestre
		mesesLaborados := calcularMesesInterseccion(c.FechaInicio, c.FechaFin, desde, hasta)

		// Obtener inasistencias
		faltas, _ := s.Repo.ObtenerInasistenciasSemestre(c.ID, desde, hasta)

		// Calcular CTS
		remComputable := sueldo + familiar + sextoGrati + promVariables
		_, desc, neto := calculadoras.CalcularCtsSemestralDL728(remComputable, mesesLaborados, faltas)

		d := models.PlanillaCtsDetalle{
			PlanillaCtsID:          p.ID,
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

	// 5. Guardar los detalles en la BD
	if len(detalles) > 0 {
		err = s.Repo.GuardarDetallesCts(detalles)
		if err != nil {
			return 0, fmt.Errorf("error al guardar los detalles de la CTS: %w", err)
		}
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

// CalcularLiquidacion realiza el cálculo en memoria de la liquidación de cese (CTS)
func (s *CtsService) CalcularLiquidacion(contratoID int, fechaInicio, fechaCese time.Time, motivo string) (*models.LiquidacionCese, error) {
	// 1. Obtener datos del contrato y puesto
	var l models.LiquidacionCese
	l.ContratoID = contratoID
	l.FechaInicioComputable = fechaInicio
	l.FechaCese = fechaCese
	l.Motivo = motivo
	l.Estado = "BORRADOR"

	// Traer información del trabajador y del puesto
	query := `
		SELECT c.tenant_id, rl.codigo AS regimen, c.puesto_id,
		       t.apellido_paterno || ' ' || t.apellido_materno || ', ' || t.nombres AS trabajador_nombre,
		       t.numero_documento, p.nombre AS puesto_nombre, COALESCE(p.sueldo_presupuestado, 0)
		FROM contratos c
		INNER JOIN trabajadores t ON c.trabajador_id = t.id
		INNER JOIN puestos p ON c.puesto_id = p.id
		INNER JOIN regimenes_laborales rl ON p.regimen_id = rl.id
		WHERE c.id = $1
	`
	var sueldo float64
	var puestoID int
	err := s.db.QueryRow(query, contratoID).Scan(
		&l.TenantID, &l.Regimen, &puestoID,
		&l.TrabajadorNombre, &l.TrabajadorDocumento, &l.PuestoNombre, &sueldo,
	)
	if err != nil {
		return nil, fmt.Errorf("error al obtener datos del contrato para la liquidación: %w", err)
	}

	// 2. Calcular años y meses de servicios
	anos, meses := s.calcularMesesYAnosServicio(fechaInicio, fechaCese)
	l.AnosServicios = anos
	l.MesesServicios = meses
	mesesTotales := anos*12 + meses

	// 3. Determinar Remuneración Computable según Régimen
	switch l.Regimen {
	case "276":
		// DL 276: Conceptos que perciba permanentemente 12 meses del año al momento del cese
		queryConceptos276 := `
			SELECT COALESCE(SUM(pc.monto), 0)
			FROM puesto_conceptos pc
			INNER JOIN conceptos_tenant ct ON pc.concepto_tenant_id = ct.id
			WHERE pc.puesto_id = $1 AND pc.activo = true AND ct.activo = true
			  AND ct.frecuencia_meses = '1,2,3,4,5,6,7,8,9,10,11,12'
			  AND ct.es_extraordinario = false
		`
		var remTotal float64
		s.db.QueryRow(queryConceptos276, puestoID).Scan(&remTotal)
		if remTotal <= 0 {
			remTotal = sueldo // Salva por si no tiene asignado nada en puesto_conceptos
		}
		l.RemuneracionComputable = remTotal
		l.MontoCts = calculadoras.CalcularCtsDL276(remTotal, mesesTotales)

	case "30057":
		// Ley SERVIR: Promedio de los últimos 36 meses efectivamente laborados
		querySERVIR := `
			SELECT COALESCE(AVG(pd.total_ingresos), 0)
			FROM planilla_detalles pd
			INNER JOIN planillas p ON pd.planilla_id = p.id
			WHERE pd.contrato_id = $1
		`
		var prom36 float64
		s.db.QueryRow(querySERVIR, contratoID).Scan(&prom36)
		if prom36 <= 0 {
			prom36 = sueldo // Por defecto
		}
		l.RemuneracionComputable = prom36
		l.MontoCts = calculadoras.CalcularCtsLey30057(prom36, mesesTotales)

	default:
		// DL 728 / CAS u otros
		l.RemuneracionComputable = sueldo
		l.MontoCts = (sueldo / 12.0) * float64(mesesTotales)
	}

	l.TotalLiquidacion = l.MontoCts + l.MontoVacacionesTruncas + l.MontoGratiTrunca
	return &l, nil
}

// Helpers locales

func (s *CtsService) calcularMesesYAnosServicio(start, end time.Time) (int, int) {
	if start.After(end) {
		return 0, 0
	}
	years := end.Year() - start.Year()
	months := int(end.Month() - start.Month())
	days := end.Day() - start.Day()

	if days < 0 {
		months--
	}
	totalMonths := years*12 + months
	if totalMonths < 0 {
		totalMonths = 0
	}
	return totalMonths / 12, totalMonths % 12
}

func calcularMesesInterseccion(start time.Time, end *time.Time, desde, hasta time.Time) int {
	s := start
	var e time.Time
	if end == nil {
		e = hasta
	} else {
		e = *end
	}

	if s.Before(desde) {
		s = desde
	}
	if e.After(hasta) {
		e = hasta
	}
	if s.After(e) {
		return 0
	}

	months := 0
	curr := time.Date(s.Year(), s.Month(), 1, 0, 0, 0, 0, time.UTC)
	for curr.Before(e) || curr.Equal(e) {
		mStart := curr
		mEnd := time.Date(curr.Year(), curr.Month()+1, 1, 0, 0, 0, 0, time.UTC).Add(-time.Second)

		// Si el rango de labores [s, e] cubre todo el mes de calendario, lo consideramos completo
		if (s.Before(mStart) || s.Equal(mStart)) && (e.After(mEnd) || e.Equal(mEnd)) {
			months++
		}
		curr = curr.AddDate(0, 1, 0)
	}
	return months
}
