package config

// ConceptosBasePorRegimen define la "Plantilla" estándar usando los Códigos SUNAT
var ConceptosBasePorRegimen = map[string][]string{
	// CAS (1057): Remuneración, Aguinaldo, Vacaciones, CTS, Tardanzas, Faltas, EsSalud, Rta 4ta (S101)
	"1057": {"2039", "2006", "2007", "2010", "0704", "0705", "0804", "S101"},

	// D.L. 276: Remuneración, Aguinaldo, Tardanzas, Faltas, EsSalud, Rta 5ta
	"276": {"2001", "2006", "0704", "0705", "0804", "0605"},

	// Ley Servir (30057): Remuneración, Aguinaldo, CTS, Tardanzas, Faltas, EsSalud, Rta 5ta
	"30057": {"2001", "2006", "2010", "0704", "0705", "0804", "0605"},
}

// PensionesBase define los códigos SUNAT según la elección del trabajador
var PensionesBase = map[string][]string{
	"ONP": {"0607"},
	"AFP": {"0608", "0606", "0601"}, // Aporte, Prima, Comisión Flujo/Mixta
}

// ConceptosQueRequierenMonto son códigos SUNAT que no son calculados por fórmula
var ConceptosQueRequierenMonto = map[string]bool{
	"2001": true, // Remuneración Principal (276)
	"2039": true, // Remuneración CAS
	"2027": true, // Otros ingresos no remunerativos
	"2006": true, // Aguinaldos (aunque se puede automatizar, a veces es variable)
}
