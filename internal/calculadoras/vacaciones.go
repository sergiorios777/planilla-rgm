package calculadoras

import "math"

// CalculadoraVacacional define la interfaz para calcular los conceptos vacacionales al cese.
type CalculadoraVacacional interface {
	Calcular(remComputable float64, mesesT, diasT, periodosVencidos, periodosNoVencidos int, contratoMenorMes bool) (truncas float64, noGozadas float64, indemnizacion float64)
}

// CalculadoraVacacionalGenerica implementa el cálculo estándar / fallback
type CalculadoraVacacionalGenerica struct{}

func (c *CalculadoraVacacionalGenerica) Calcular(remComputable float64, mesesT, diasT, periodosVencidos, periodosNoVencidos int, contratoMenorMes bool) (float64, float64, float64) {
	truncas := (remComputable/12.0)*float64(mesesT) + (remComputable/360.0)*float64(diasT)
	noGozadas := remComputable * float64(periodosVencidos+periodosNoVencidos)
	return round(truncas), round(noGozadas), 0.0
}

// CalculadoraVacacional276 calcula para el régimen DL 276 (Carrera Administrativa)
type CalculadoraVacacional276 struct{}

func (c *CalculadoraVacacional276) Calcular(remComputable float64, mesesT, diasT, periodosVencidos, periodosNoVencidos int, contratoMenorMes bool) (float64, float64, float64) {
	truncas := (remComputable/12.0)*float64(mesesT) + (remComputable/360.0)*float64(diasT)

	// Límite estricto de acumulación de 2 periodos
	totalPeriodos := periodosVencidos + periodosNoVencidos
	if totalPeriodos > 2 {
		totalPeriodos = 2
	}
	noGozadas := remComputable * float64(totalPeriodos)

	return round(truncas), round(noGozadas), 0.0
}

// CalculadoraVacacionalCAS calcula para el régimen DL 1057 (CAS)
type CalculadoraVacacionalCAS struct{}

func (c *CalculadoraVacacionalCAS) Calcular(remComputable float64, mesesT, diasT, periodosVencidos, periodosNoVencidos int, contratoMenorMes bool) (float64, float64, float64) {
	var truncas float64
	if !contratoMenorMes {
		truncas = (remComputable/12.0)*float64(mesesT) + (remComputable/360.0)*float64(diasT)
	}

	noGozadas := remComputable * float64(periodosVencidos+periodosNoVencidos)
	return round(truncas), round(noGozadas), 0.0
}

// CalculadoraVacacional728 calcula para el régimen DL 728 (Actividad Privada)
type CalculadoraVacacional728 struct{}

func (c *CalculadoraVacacional728) Calcular(remComputable float64, mesesT, diasT, periodosVencidos, periodosNoVencidos int, contratoMenorMes bool) (float64, float64, float64) {
	truncas := (remComputable/12.0)*float64(mesesT) + (remComputable/360.0)*float64(diasT)

	// Vacaciones no gozadas simples para todos los periodos pendientes
	noGozadas := remComputable * float64(periodosVencidos+periodosNoVencidos)

	// Indemnización equivalente a una remuneración por cada periodo ya vencido
	indemnizacion := remComputable * float64(periodosVencidos)

	return round(truncas), round(noGozadas), round(indemnizacion)
}

// CalculadoraVacacionalServir calcula para el régimen Ley 30057 (SERVIR)
type CalculadoraVacacionalServir struct{}

func (c *CalculadoraVacacionalServir) Calcular(remComputable float64, mesesT, diasT, periodosVencidos, periodosNoVencidos int, contratoMenorMes bool) (float64, float64, float64) {
	truncas := (remComputable/12.0)*float64(mesesT) + (remComputable/360.0)*float64(diasT)
	noGozadas := remComputable * float64(periodosVencidos+periodosNoVencidos)
	return round(truncas), round(noGozadas), 0.0
}

// ObtenerCalculadoraVacacional es la fábrica (Factory) que retorna la estrategia adecuada
func ObtenerCalculadoraVacacional(regimen string) CalculadoraVacacional {
	switch regimen {
	case "276":
		return &CalculadoraVacacional276{}
	case "1057", "CAS":
		return &CalculadoraVacacionalCAS{}
	case "728":
		return &CalculadoraVacacional728{}
	case "30057":
		return &CalculadoraVacacionalServir{}
	default:
		return &CalculadoraVacacionalGenerica{}
	}
}

func round(val float64) float64 {
	return math.Round(val*100) / 100
}
