package calculadoras

import (
	"math"
	"testing"
)

func TestCalculadoraVacacionalStrategy(t *testing.T) {
	tests := []struct {
		name              string
		regimen           string
		remComputable     float64
		mesesT            int
		diasT             int
		periodosVencidos  int
		periodosNoVencid  int
		contratoMenorMes  bool
		wantTruncas       float64
		wantNoGozadas     float64
		wantIndemnist     float64
	}{
		// 1. DL 276
		{
			name:              "DL 276 - Standard calculation and VNG limit of 2 periods (accumulated 1)",
			regimen:           "276",
			remComputable:     1200.00,
			mesesT:            6,
			diasT:             0,
			periodosVencidos:  1,
			periodosNoVencid:  0,
			contratoMenorMes:  false,
			wantTruncas:       600.00,
			wantNoGozadas:     1200.00,
			wantIndemnist:     0.0,
		},
		{
			name:              "DL 276 - VNG limited to 2 periods when accumulating 3",
			regimen:           "276",
			remComputable:     1200.00,
			mesesT:            6,
			diasT:             0,
			periodosVencidos:  2,
			periodosNoVencid:  1, // Total 3
			contratoMenorMes:  false,
			wantTruncas:       600.00,
			wantNoGozadas:     2400.00, // limited to 2 * 1200
			wantIndemnist:     0.0,
		},
		// 2. DL 1057 (CAS)
		{
			name:              "CAS - Standard calculation >= 1 month",
			regimen:           "1057",
			remComputable:     1500.00,
			mesesT:            3,
			diasT:             12,
			periodosVencidos:  0,
			periodosNoVencid:  0,
			contratoMenorMes:  false,
			wantTruncas:       425.00, // (1500/12)*3 + (1500/360)*12 = 375 + 50 = 425
			wantNoGozadas:     0.0,
			wantIndemnist:     0.0,
		},
		{
			name:              "CAS - Truncas is 0 if total time < 1 month",
			regimen:           "CAS",
			remComputable:     1500.00,
			mesesT:            0,
			diasT:             20,
			periodosVencidos:  0,
			periodosNoVencid:  0,
			contratoMenorMes:  true,
			wantTruncas:       0.0,
			wantNoGozadas:     0.0,
			wantIndemnist:     0.0,
		},
		// 3. DL 728
		{
			name:              "DL 728 - Standard calculation with VNG and indemnification",
			regimen:           "728",
			remComputable:     2000.00,
			mesesT:            6,
			diasT:             15,
			periodosVencidos:  1, // 1 expired period (gets 1 VNG + 1 Indemnization)
			periodosNoVencid:  1, // 1 simple pending period (gets 1 VNG)
			contratoMenorMes:  false,
			wantTruncas:       1083.33, // (2000/12)*6 + (2000/360)*15 = 1000 + 83.33 = 1083.33
			wantNoGozadas:     4000.00, // 2 periods * 2000
			wantIndemnist:     2000.00, // 1 expired period * 2000
		},
		// 4. SERVIR (Ley 30057)
		{
			name:              "SERVIR - Standard calculation",
			regimen:           "30057",
			remComputable:     3000.00,
			mesesT:            4,
			diasT:             18,
			periodosVencidos:  1,
			periodosNoVencid:  1,
			contratoMenorMes:  false,
			wantTruncas:       1150.00, // (3000/12)*4 + (3000/360)*18 = 1000 + 150 = 1150
			wantNoGozadas:     6000.00,
			wantIndemnist:     0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calc := ObtenerCalculadoraVacacional(tt.regimen)
			truncas, noGozadas, indem := calc.Calcular(tt.remComputable, tt.mesesT, tt.diasT, tt.periodosVencidos, tt.periodosNoVencid, tt.contratoMenorMes)

			if math.Abs(truncas-tt.wantTruncas) > 0.01 {
				t.Errorf("Calcular() truncas = %v, want %v", truncas, tt.wantTruncas)
			}
			if math.Abs(noGozadas-tt.wantNoGozadas) > 0.01 {
				t.Errorf("Calcular() noGozadas = %v, want %v", noGozadas, tt.wantNoGozadas)
			}
			if math.Abs(indem-tt.wantIndemnist) > 0.01 {
				t.Errorf("Calcular() indemnizacion = %v, want %v", indem, tt.wantIndemnist)
			}
		})
	}
}
