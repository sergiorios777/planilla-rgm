package calculadoras

import (
	"testing"
)

func TestCalcularAsignacionFamiliar(t *testing.T) {
	// RMV = 1025.00 -> 10% = 102.50
	m1 := CalcularAsignacionFamiliar(1025.00)
	if m1 != 102.50 {
		t.Errorf("got %v; want 102.50", m1)
	}

	// RMV = 1130.00 -> 10% = 113.00
	m2 := CalcularAsignacionFamiliar(1130.00)
	if m2 != 113.00 {
		t.Errorf("got %v; want 113.00", m2)
	}
}
