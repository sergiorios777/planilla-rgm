package services

import (
	"database/sql"
	"fmt"
	"math"
	"planilla-rgm/internal/calculadoras"
	"planilla-rgm/internal/helpers"
	"planilla-rgm/internal/models"
	"time"
)

type LiquidacionService struct {
	db                *sql.DB
	VacacionesService *VacacionesService
	BaseComputable    *BaseComputableService
}

func NewLiquidacionService(db *sql.DB, vacService *VacacionesService) *LiquidacionService {
	return &LiquidacionService{
		db:                db,
		VacacionesService: vacService,
		BaseComputable:    NewBaseComputableService(db),
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

	// 3. Determinar Remuneración Computable para CTS según Régimen usando BaseComputableService
	desgloseCTS, err := s.BaseComputable.ResolverBaseComputable(l.TenantID, contratoID, puestoID, regimenID, l.Regimen, BeneficioCTS, fechaCese)
	if err != nil || desgloseCTS == nil || desgloseCTS.TotalComputable <= 0 {
		desgloseCTS = &DesgloseBaseComputable{TotalComputable: sueldo}
	}
	l.RemuneracionComputable = desgloseCTS.TotalComputable

	switch l.Regimen {
	case "276":
		// DL 276: Cálculo según años completos o fracción mayor a 6 meses
		l.MontoCts = calculadoras.CalcularCtsDL276(desgloseCTS.TotalComputable, mesesTotales)

	case "30057":
		// Ley SERVIR: Promedio de compensaciones de los últimos 36 meses
		l.MontoCts = calculadoras.CalcularCtsLey30057(desgloseCTS.TotalComputable, mesesTotales)

	case "1057", "CAS":
		// CAS (DL 1057): Ley 32563 / DS 142-2026-EF - CTS al cese
		l.MontoCts = calculadoras.CalcularCtsDL1057(desgloseCTS.TotalComputable, fechaCese.Year(), mesesTotales)

		// CAS: Gratificación Trunca
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
		desgloseGratiCAS, _ := s.BaseComputable.ResolverBaseComputable(l.TenantID, contratoID, puestoID, regimenID, l.Regimen, BeneficioGratificacion, fechaCese)
		remGratiCAS := desgloseGratiCAS.TotalComputable
		if remGratiCAS <= 0 {
			remGratiCAS = sueldo
		}
		gratiTrunca, _ := calculadoras.CalcularGratificacionDL1057(remGratiCAS, mesPago, fechaCese.Year(), mesesGrati, diasGrati)
		l.MontoGratiTrunca = gratiTrunca

	default:
		// DL 728: CTS Trunca (meses y días por dozavos y treintavos)
		montoMeses := (desgloseCTS.TotalComputable / 12.0) * float64(mesesTotales)
		montoDias := (desgloseCTS.TotalComputable / 360.0) * float64(dias)
		l.MontoCts = math.Round((montoMeses+montoDias)*100) / 100

		// DL 728: Gratificación Trunca (meses calendario completos laborados en el semestre)
		var semStart, semEnd time.Time
		if fechaCese.Month() <= 6 {
			semStart = time.Date(fechaCese.Year(), time.January, 1, 0, 0, 0, 0, time.UTC)
			semEnd = time.Date(fechaCese.Year(), time.June, 30, 23, 59, 59, 0, time.UTC)
		} else {
			semStart = time.Date(fechaCese.Year(), time.July, 1, 0, 0, 0, 0, time.UTC)
			semEnd = time.Date(fechaCese.Year(), time.December, 31, 23, 59, 59, 0, time.UTC)
		}

		mesesGrati728 := calculadoras.CalcularMesesSemestreGratificacion(fechaInicio, &fechaCese, semStart, semEnd)
		desgloseGrati728, _ := s.BaseComputable.ResolverBaseComputable(l.TenantID, contratoID, puestoID, regimenID, l.Regimen, BeneficioGratificacion, fechaCese)
		baseGrati728 := desgloseGrati728.TotalComputable
		if baseGrati728 <= 0 {
			baseGrati728 = sueldo
		}
		grati728, bonoExt := calculadoras.CalcularGratificacionDL728(baseGrati728, mesesGrati728)
		l.MontoGratiTrunca = math.Round((grati728+bonoExt)*100) / 100
	}

	// 4. Calcular Vacaciones usando BaseComputableService para Vacaciones (SIN 1/6 de gratificación en 728)
	desgloseVac, _ := s.BaseComputable.ResolverBaseComputable(l.TenantID, contratoID, puestoID, regimenID, l.Regimen, BeneficioVacaciones, fechaCese)
	baseVac := desgloseVac.TotalComputable
	if baseVac <= 0 {
		baseVac = sueldo
	}

	truncas, noGozadas, indemnizacion, _, err := s.VacacionesService.CalcularVacacionesCeseConBase(
		baseVac, l.Regimen, fechaInicio, fechaCese, periodosVencidos, periodosNoVencidos,
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
	l.TotalLiquidacion = math.Round(l.TotalLiquidacion*100) / 100
	return &l, nil
}
