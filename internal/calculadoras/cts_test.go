package calculadoras

import (
	"math"
	"testing"
)

func TestCalcularCtsSemestralDL728(t *testing.T) {
	tests := []struct {
		name             string
		remComputable    float64
		mesesLaborados   int
		diasFaltas       int
		wantBruto        float64
		wantDescuento    float64
		wantNeto         float64
	}{
		{
			name:           "Semestre completo sin faltas",
			remComputable:  3000.00,
			mesesLaborados: 6,
			diasFaltas:     0,
			wantBruto:      1500.00,
			wantDescuento:  0.00,
			wantNeto:       1500.00,
		},
		{
			name:           "Semestre completo con faltas",
			remComputable:  3000.00,
			mesesLaborados: 6,
			diasFaltas:     3,
			wantBruto:      1500.00,
			wantDescuento:  25.00, // (250 / 30) * 3 = 25.00
			wantNeto:       1475.00,
		},
		{
			name:           "Menos de un mes de labor",
			remComputable:  3000.00,
			mesesLaborados: 0,
			diasFaltas:     0,
			wantBruto:      0.00,
			wantDescuento:  0.00,
			wantNeto:       0.00,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bruto, desc, neto := CalcularCtsSemestralDL728(tt.remComputable, tt.mesesLaborados, tt.diasFaltas)
			if math.Abs(bruto-tt.wantBruto) > 0.001 {
				t.Errorf("CalcularCtsSemestralDL728() bruto = %v, want %v", bruto, tt.wantBruto)
			}
			if math.Abs(desc-tt.wantDescuento) > 0.001 {
				t.Errorf("CalcularCtsSemestralDL728() desc = %v, want %v", desc, tt.wantDescuento)
			}
			if math.Abs(neto-tt.wantNeto) > 0.001 {
				t.Errorf("CalcularCtsSemestralDL728() neto = %v, want %v", neto, tt.wantNeto)
			}
		})
	}
}

func TestCalcularCtsLey30057(t *testing.T) {
	tests := []struct {
		name                 string
		promedioValorizaciones float64
		mesesTotales         int
		want                 float64
	}{
		{
			name:                   "Dos años completos",
			promedioValorizaciones: 4000.00,
			mesesTotales:           24,
			want:                   8000.00,
		},
		{
			name:                   "Fracción de medio año",
			promedioValorizaciones: 3000.00,
			mesesTotales:           6,
			want:                   1500.00,
		},
		{
			name:                   "Cero meses",
			promedioValorizaciones: 3000.00,
			mesesTotales:           0,
			want:                   0.00,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalcularCtsLey30057(tt.promedioValorizaciones, tt.mesesTotales)
			if math.Abs(got-tt.want) > 0.001 {
				t.Errorf("CalcularCtsLey30057() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalcularCtsDL276(t *testing.T) {
	tests := []struct {
		name             string
		remuneracionTotal float64
		mesesTotales     int
		want             float64
	}{
		{
			name:              "Año y 6 meses (se queda en 1 año)",
			remuneracionTotal: 2500.00,
			mesesTotales:     18,
			want:              2500.00,
		},
		{
			name:              "Año y 7 meses (se redondea a 2 años)",
			remuneracionTotal: 2500.00,
			mesesTotales:     19,
			want:              5000.00,
		},
		{
			name:              "Menos de 6 meses (cero años)",
			remuneracionTotal: 2500.00,
			mesesTotales:     5,
			want:              0.00,
		},
		{
			name:              "Justo 6 meses (cero años por no ser mayor a 6 meses)",
			remuneracionTotal: 2500.00,
			mesesTotales:     6,
			want:              0.00,
		},
		{
			name:              "Justo 7 meses (mayor a 6 meses, redondea a 1 año)",
			remuneracionTotal: 2500.00,
			mesesTotales:     7,
			want:              2500.00,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalcularCtsDL276(tt.remuneracionTotal, tt.mesesTotales)
			if math.Abs(got-tt.want) > 0.001 {
				t.Errorf("CalcularCtsDL276() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalcularCtsDL1057(t *testing.T) {
	tests := []struct {
		name                string
		remuneracionMensual float64
		anioCese            int
		mesesTotales        int
		want                float64
	}{
		{
			name:                "Caso 1: Año 2026, 17 meses (1a 5m -> 1 año, 10% de S/2500 = S/250)",
			remuneracionMensual: 2500.00,
			anioCese:            2026,
			mesesTotales:        17,
			want:                250.00,
		},
		{
			name:                "Caso 2: Año 2026, 20 meses (1a 8m -> 2 años por fracción >6m, 10% de S/2500*2 = S/500)",
			remuneracionMensual: 2500.00,
			anioCese:            2026,
			mesesTotales:        20,
			want:                500.00,
		},
		{
			name:                "Caso 3: Año 2026, 5 meses (<6m -> 0 años)",
			remuneracionMensual: 3000.00,
			anioCese:            2026,
			mesesTotales:        5,
			want:                0.00,
		},
		{
			name:                "Caso 4: Año 2027, 20 meses (1a 8m -> 2 años, 20% de S/3000*2 = S/1200)",
			remuneracionMensual: 3000.00,
			anioCese:            2027,
			mesesTotales:        20,
			want:                1200.00,
		},
		{
			name:                "Caso 5: Año 2030, 60 meses (5 años, 100% de S/4000*5 = S/20000)",
			remuneracionMensual: 4000.00,
			anioCese:            2030,
			mesesTotales:        60,
			want:                20000.00,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalcularCtsDL1057(tt.remuneracionMensual, tt.anioCese, tt.mesesTotales)
			if math.Abs(got-tt.want) > 0.001 {
				t.Errorf("CalcularCtsDL1057() = %v, want %v", got, tt.want)
			}
		})
	}
}
