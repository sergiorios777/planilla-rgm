package services

import (
	"database/sql"
	"fmt"
	"planilla-rgm/internal/calculadoras"
	"planilla-rgm/internal/helpers"
	"planilla-rgm/internal/models"
	"time"
)

type LiquidacionService struct {
	db                *sql.DB
	VacacionesService *VacacionesService
}

func NewLiquidacionService(db *sql.DB, vacService *VacacionesService) *LiquidacionService {
	return &LiquidacionService{
		db:                db,
		VacacionesService: vacService,
	}
}

// CalcularLiquidacion realiza el cálculo en memoria de la liquidación de cese (CTS, Vacaciones e Indemnizaciones)
func (s *LiquidacionService) CalcularLiquidacion(contratoID int, fechaInicio, fechaCese time.Time, motivo string, periodosVencidos, periodosNoVencidos int) (*models.LiquidacionCese, error) {
	// 1. Obtener datos del contrato y puesto
	var l models.LiquidacionCese
	l.ContratoID = contratoID
	l.FechaInicioComputable = fechaInicio
	l.FechaCese = fechaCese
	l.Motivo = motivo
	l.Estado = "BORRADOR"

	// Traer información del trabajador y del puesto
	query := `
		SELECT c.tenant_id, rl.codigo AS regimen, p.regimen_id, c.puesto_id,
		       t.apellido_paterno || ' ' || t.apellido_materno || ', ' || t.nombres AS trabajador_nombre,
		       t.numero_documento, p.nombre AS puesto_nombre, COALESCE(p.sueldo_presupuestado, 0)
		FROM contratos c
		INNER JOIN trabajadores t ON c.trabajador_id = t.id
		INNER JOIN puestos p ON c.puesto_id = p.id
		INNER JOIN regimenes_laborales rl ON p.regimen_id = rl.id
		WHERE c.id = $1
	`
	var sueldo float64
	var puestoID, regimenID int
	err := s.db.QueryRow(query, contratoID).Scan(
		&l.TenantID, &l.Regimen, &regimenID, &puestoID,
		&l.TrabajadorNombre, &l.TrabajadorDocumento, &l.PuestoNombre, &sueldo,
	)
	if err != nil {
		return nil, fmt.Errorf("error al obtener datos del contrato para la liquidación: %w", err)
	}

	// 2. Calcular años, meses y días de servicios usando helpers transversales
	anos, meses, dias := helpers.CalcularTiempoServicioCompleto(fechaInicio, fechaCese)

	// Validar consistencia de periodos vacacionales completos ingresados manualmente
	if anos == 0 && (periodosVencidos > 0 || periodosNoVencidos > 0) {
		return nil, fmt.Errorf("el trabajador tiene menos de 1 año de servicios (%d meses); no puede poseer periodos de vacaciones completos ganados", meses)
	}

	l.AnosServicios = anos
	l.MesesServicios = meses
	l.DiasServicios = dias
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

	case "1057", "CAS":
		// CAS (DL 1057): Ley 32563 / DS 142-2026-EF - CTS al cese
		remTotalCts, _ := s.VacacionesService.BaseRegimenRepo.ObtenerMontoVariable(l.TenantID, puestoID, regimenID, "CTS", "RETRIBUCION_MENSUAL")
		if remTotalCts <= 0 {
			remTotalCts = sueldo
		}
		l.RemuneracionComputable = remTotalCts
		l.MontoCts = calculadoras.CalcularCtsDL1057(remTotalCts, fechaCese.Year(), mesesTotales)

		// CAS (DL 1057): Gratificación Trunca
		var semStart, semEnd time.Time
		mesPago := 7
		if fechaCese.Month() <= 6 {
			semStart = time.Date(fechaCese.Year(), time.January, 1, 0, 0, 0, 0, time.UTC)
			semEnd = time.Date(fechaCese.Year(), time.June, 30, 23, 59, 59, 0, time.UTC)
			mesPago = 7
		} else {
			semStart = time.Date(fechaCese.Year(), time.July, 1, 0, 0, 0, 0, time.UTC)
			semEnd = time.Date(fechaCese.Year(), time.December, 31, 23, 59, 59, 0, time.UTC)
			mesPago = 12
		}

		mesesGrati, diasGrati := helpers.CalcularMesesYDiasSemestreGratificacionCAS(fechaInicio, &fechaCese, semStart, semEnd)
		remTotalGrati, _ := s.VacacionesService.BaseRegimenRepo.ObtenerMontoVariable(l.TenantID, puestoID, regimenID, "GRATIFICACION", "RETRIBUCION_MENSUAL")
		if remTotalGrati <= 0 {
			remTotalGrati = sueldo
		}
		gratiTrunca, _ := calculadoras.CalcularGratificacionDL1057(remTotalGrati, mesPago, fechaCese.Year(), mesesGrati, diasGrati)
		l.MontoGratiTrunca = gratiTrunca

	default:
		// DL 728 u otros
		l.RemuneracionComputable = sueldo
		l.MontoCts = (sueldo / 12.0) * float64(mesesTotales)
	}

	// 4. Calcular Vacaciones usando el nuevo VacacionesService
	truncas, noGozadas, indemnizacion, _, err := s.VacacionesService.CalcularVacacionesCese(
		l.TenantID, puestoID, regimenID, l.Regimen,
		sueldo, fechaInicio, fechaCese, periodosVencidos, periodosNoVencidos,
	)
	if err != nil {
		return nil, fmt.Errorf("error al calcular vacaciones para liquidación: %w", err)
	}

	l.MontoVacacionesTruncas = truncas
	l.MontoVacacionesNoGozadas = noGozadas
	l.MontoIndemnizacionVacacional = indemnizacion
	l.PeriodosVencidosVacaciones = periodosVencidos
	l.PeriodosNoVencidosVacaciones = periodosNoVencidos

	l.TotalLiquidacion = l.MontoCts + l.MontoVacacionesTruncas + l.MontoVacacionesNoGozadas + l.MontoIndemnizacionVacacional + l.MontoGratiTrunca
	return &l, nil
}
