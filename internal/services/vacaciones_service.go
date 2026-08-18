package services

import (
	"database/sql"
	"planilla-rgm/internal/calculadoras"
	"planilla-rgm/internal/helpers"
	"planilla-rgm/internal/repository"
	"time"
)

type VacacionesService struct {
	BaseRegimenRepo *repository.BaseRegimenRepository
	BaseComputable  *BaseComputableService
}

func NewVacacionesService(baseRegimenRepo *repository.BaseRegimenRepository) *VacacionesService {
	return &VacacionesService{
		BaseRegimenRepo: baseRegimenRepo,
	}
}

func NewVacacionesServiceWithDB(db *sql.DB) *VacacionesService {
	repo := repository.NewBaseRegimenRepository(db)
	return &VacacionesService{
		BaseRegimenRepo: repo,
		BaseComputable:  NewBaseComputableService(db),
	}
}

// CalcularVacacionesCese calcula los montos de vacaciones truncas, no gozadas e indemnización vacacional.
func (s *VacacionesService) CalcularVacacionesCese(
	tenantID, puestoID, regimenID int,
	regimenCod string,
	sueldo float64,
	fechaInicio, fechaCese time.Time,
	periodosVencidos, periodosNoVencidos int,
) (truncas, noGozadas, indemnizacion, baseComputable float64, err error) {
	var totalBaseVac float64

	// Si tenemos BaseComputableService configurado, resolverlo con el motor declarativo
	if s.BaseComputable != nil {
		desglose, err := s.BaseComputable.ResolverBaseComputable(tenantID, 0, puestoID, regimenID, regimenCod, BeneficioVacaciones, fechaCese)
		if err == nil && desglose.TotalComputable > 0 {
			totalBaseVac = desglose.TotalComputable
		}
	}

	// Fallback con BaseRegimenRepo
	if totalBaseVac <= 0 && s.BaseRegimenRepo != nil {
		switch regimenCod {
		case "276":
			muc, _ := s.BaseRegimenRepo.ObtenerMontoVariable(tenantID, puestoID, regimenID, "VAC_TRUNCAS", "MUC")
			bet, _ := s.BaseRegimenRepo.ObtenerMontoVariable(tenantID, puestoID, regimenID, "VAC_TRUNCAS", "BET")
			totalBaseVac = muc + bet
		case "30057":
			vp, _ := s.BaseRegimenRepo.ObtenerMontoVariable(tenantID, puestoID, regimenID, "VAC_TRUNCAS", "VALORIZACION_PRINCIPAL")
			va, _ := s.BaseRegimenRepo.ObtenerMontoVariable(tenantID, puestoID, regimenID, "VAC_TRUNCAS", "VALORIZACION_AJUSTADA")
			totalBaseVac = vp + va
		case "1057", "CAS":
			retribucionMensual, _ := s.BaseRegimenRepo.ObtenerMontoVariable(tenantID, puestoID, regimenID, "VAC_TRUNCAS", "RETRIBUCION_MENSUAL")
			totalBaseVac = retribucionMensual
		default:
			baseVac, _ := s.BaseRegimenRepo.ObtenerMontoVariable(tenantID, puestoID, regimenID, "VAC_TRUNCAS", "REMUNERACION_BASICA")
			asigFamVac, _ := s.BaseRegimenRepo.ObtenerMontoVariable(tenantID, puestoID, regimenID, "VAC_TRUNCAS", "ASIGNACION_FAMILIAR")
			promVarVac, _ := s.BaseRegimenRepo.ObtenerMontoVariable(tenantID, puestoID, regimenID, "VAC_TRUNCAS", "REMUNERACION_VARIABLE")
			totalBaseVac = baseVac + asigFamVac + promVarVac
		}
	}

	if totalBaseVac <= 0 {
		totalBaseVac = sueldo // Fallback final
	}

	return s.CalcularVacacionesCeseConBase(totalBaseVac, regimenCod, fechaInicio, fechaCese, periodosVencidos, periodosNoVencidos)
}

// CalcularVacacionesCeseConBase calcula directamente con una base previamente determinada
func (s *VacacionesService) CalcularVacacionesCeseConBase(
	baseComputable float64,
	regimenCod string,
	fechaInicio, fechaCese time.Time,
	periodosVencidos, periodosNoVencidos int,
) (truncas, noGozadas, indemnizacion, baseTotal float64, err error) {
	mesesTruncos, diasTruncos := helpers.CalcularMesesYDiasTruncas(fechaInicio, fechaCese)

	// Validar si la duración del contrato es menor a 1 mes para la restricción de CAS
	totalMeses, totalAnos := helpers.CalcularMesesYAnosServicio(fechaInicio, fechaCese)
	contratoMenorMes := (totalAnos == 0 && totalMeses == 0)

	calcVac := calculadoras.ObtenerCalculadoraVacacional(regimenCod)
	truncas, noGozadas, indemnizacion = calcVac.Calcular(
		baseComputable,
		mesesTruncos,
		diasTruncos,
		periodosVencidos,
		periodosNoVencidos,
		contratoMenorMes,
	)

	return truncas, noGozadas, indemnizacion, baseComputable, nil
}
