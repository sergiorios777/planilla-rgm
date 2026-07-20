package services

import (
	"planilla-rgm/internal/calculadoras"
	"planilla-rgm/internal/helpers"
	"planilla-rgm/internal/repository"
	"time"
)

type VacacionesService struct {
	BaseRegimenRepo *repository.BaseRegimenRepository
}

func NewVacacionesService(baseRegimenRepo *repository.BaseRegimenRepository) *VacacionesService {
	return &VacacionesService{
		BaseRegimenRepo: baseRegimenRepo,
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

	// Obtener base computable según el régimen laboral
	switch regimenCod {
	case "276":
		// DL 276: MUC + BET
		muc, _ := s.BaseRegimenRepo.ObtenerMontoVariable(tenantID, puestoID, regimenID, "VAC_TRUNCAS", "MUC")
		bet, _ := s.BaseRegimenRepo.ObtenerMontoVariable(tenantID, puestoID, regimenID, "VAC_TRUNCAS", "BET")
		totalBaseVac = muc + bet
	case "30057":
		// Ley SERVIR: Valorización Principal + Valorización Ajustada
		vp, _ := s.BaseRegimenRepo.ObtenerMontoVariable(tenantID, puestoID, regimenID, "VAC_TRUNCAS", "VALORIZACION_PRINCIPAL")
		va, _ := s.BaseRegimenRepo.ObtenerMontoVariable(tenantID, puestoID, regimenID, "VAC_TRUNCAS", "VALORIZACION_AJUSTADA")
		totalBaseVac = vp + va
	case "1057", "CAS":
		// CAS: Retribución Mensual
		totalBaseVac = sueldo
	default:
		// DL 728 / Genérico: Sueldo Básico + Asignación Familiar + Remuneración Variable
		baseVac, _ := s.BaseRegimenRepo.ObtenerMontoVariable(tenantID, puestoID, regimenID, "VAC_TRUNCAS", "SUELDO_BASICO")
		asigFamVac, _ := s.BaseRegimenRepo.ObtenerMontoVariable(tenantID, puestoID, regimenID, "VAC_TRUNCAS", "ASIGNACION_FAMILIAR")
		promVarVac, _ := s.BaseRegimenRepo.ObtenerMontoVariable(tenantID, puestoID, regimenID, "VAC_TRUNCAS", "REMUNERACION_VARIABLE")
		totalBaseVac = baseVac + asigFamVac + promVarVac
	}

	if totalBaseVac <= 0 {
		totalBaseVac = sueldo // Fallback
	}

	mesesTruncos, diasTruncos := helpers.CalcularMesesYDiasTruncas(fechaInicio, fechaCese)

	// Validar si la duración del contrato es menor a 1 mes para la restricción de CAS
	totalMeses, totalAnos := helpers.CalcularMesesYAnosServicio(fechaInicio, fechaCese)
	contratoMenorMes := (totalAnos == 0 && totalMeses == 0)

	calcVac := calculadoras.ObtenerCalculadoraVacacional(regimenCod)
	truncas, noGozadas, indemnizacion = calcVac.Calcular(
		totalBaseVac,
		mesesTruncos,
		diasTruncos,
		periodosVencidos,
		periodosNoVencidos,
		contratoMenorMes,
	)

	return truncas, noGozadas, indemnizacion, totalBaseVac, nil
}
