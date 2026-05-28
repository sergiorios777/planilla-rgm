package services

import (
	"testing"
)

func TestParseRate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected float64
		wantErr  bool
	}{
		{
			name:     "Porcentaje con %",
			input:    "1.47%",
			expected: 0.0147,
			wantErr:  false,
		},
		{
			name:     "Porcentaje con % y coma",
			input:    "1,55%",
			expected: 0.0155,
			wantErr:  false,
		},
		{
			name:     "Porcentaje sin % pero mayor a 0.30",
			input:    "1.60",
			expected: 0.0160,
			wantErr:  false,
		},
		{
			name:     "Porcentaje sin % con coma y mayor a 0.30",
			input:    "1,69",
			expected: 0.0169,
			wantErr:  false,
		},
		{
			name:     "Valor entero mayor a 0.30",
			input:    "10",
			expected: 0.1000,
			wantErr:  false,
		},
		{
			name:     "Valor entero con %",
			input:    "10%",
			expected: 0.1000,
			wantErr:  false,
		},
		{
			name:     "Formato decimal directo menor a 0.30",
			input:    "0.0137",
			expected: 0.0137,
			wantErr:  false,
		},
		{
			name:     "Formato decimal directo menor a 0.30 con coma",
			input:    "0,0137",
			expected: 0.0137,
			wantErr:  false,
		},
		{
			name:     "Valor cero",
			input:    "0",
			expected: 0.0,
			wantErr:  false,
		},
		{
			name:     "Valor vacío",
			input:    "",
			expected: 0.0,
			wantErr:  false,
		},
		{
			name:     "Guión",
			input:    "-",
			expected: 0.0,
			wantErr:  false,
		},
		{
			name:     "Valor inválido",
			input:    "abc",
			expected: 0.0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseRate(%q) error = %v, wantErr = %v", tt.input, err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.expected {
				t.Errorf("parseRate(%q) = %f, expected = %f", tt.input, got, tt.expected)
			}
		})
	}
}
