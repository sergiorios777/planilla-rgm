package services

import (
	"testing"
)

func TestFormatearMoneda(t *testing.T) {
	tests := []struct {
		name     float64
		input    float64
		expected string
	}{
		{
			input:    0.00,
			expected: "0.00",
		},
		{
			input:    1234.56,
			expected: "1,234.56",
		},
		{
			input:    1234567.89,
			expected: "1,234,567.89",
		},
		{
			input:    99.9,
			expected: "99.90",
		},
	}

	for _, tt := range tests {
		got := formatearMoneda(tt.input)
		if got != tt.expected {
			t.Errorf("formatearMoneda(%f) = %s; want %s", tt.input, got, tt.expected)
		}
	}
}
