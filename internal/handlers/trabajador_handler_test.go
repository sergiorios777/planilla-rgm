package handlers

import (
	"testing"
)

func TestParseFechaExcel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "Valid YYYY-MM-DD",
			input:    "1990-05-15",
			expected: "1990-05-15",
			wantErr:  false,
		},
		{
			name:     "Valid DD/MM/YYYY",
			input:    "25/12/1985",
			expected: "1985-12-25",
			wantErr:  false,
		},
		{
			name:     "Valid DD-MM-YYYY",
			input:    "01-08-2023",
			expected: "2023-08-01",
			wantErr:  false,
		},
		{
			name:     "Valid YYYY/MM/DD",
			input:    "2010/04/30",
			expected: "2010-04-30",
			wantErr:  false,
		},
		{
			name:     "Empty input",
			input:    "  ",
			expected: "",
			wantErr:  false,
		},
		{
			name:     "Invalid date format",
			input:    "31-31-2020",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "Non-date string",
			input:    "not-a-date",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseFechaExcel(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseFechaExcel(%q) error = %v, wantErr = %v", tt.input, err, tt.wantErr)
			}
			if got != tt.expected {
				t.Errorf("parseFechaExcel(%q) = %q, expected = %q", tt.input, got, tt.expected)
			}
		})
	}
}
