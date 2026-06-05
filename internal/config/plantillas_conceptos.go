package config

// ConceptosMestrosCTS define los códigos internos de la tabla codigos_maestros para conceptos relacionados a CTS por régimen laboral
var ConceptosMestrosCTS = map[string]map[string][]string{
	"DL 728": {
		"remuneracion":        {"2002"},                                         // Remuneración Principal
		"gratificacion":       {"0406", "GRATI_DIC_DL_728", "GRATI_JUL_DL_728"}, // Gratificación
		"asignacion_familiar": {"ASIG_FAM_DL728"},
	},
	"LEY 30057": {
		"compensacion_economica": {"2001"}, // Remuneración Principal
	},
}

// ConceptosBaseGratificaciones define los códigos internos de la tabla codigos_maestros para conceptos relacionados a Gratificaciones por régimen laboral
var ConceptosBaseGratificaciones = map[string]map[string][]string{
	"DL 728": {
		"remuneracion":                {"2002"},
		"asignacion_familiar":         {"ASIG_FAM_DL728"},
		"gratificacion":               {"0406", "GRATI_DIC_DL_728", "GRATI_JUL_DL_728"},
		"bonificacion_extraordinaria": {"0312", "BON_EXTR_DIC_DL_728", "BON_EXTR_JUL_DL_728"},
	},
}

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

// ClasificadorMefPorContrato mapea [Régimen][Tipo Contrato] -> Código Limpio MEF
var ClasificadorMefPorContrato = map[string]map[string]string{
	"DL 276": {
		"Nombrado":     "2.1.1.1.1.2",
		"A plazo fijo": "2.1.1.1.1.3",
	},
	"Ley 30057": {
		"Alta dirección - Libre designación y remoción": "2.1.1.1.1.7",
		"Alcalde": "2.1.1.1.1.1",
	},
	"DL 1057": {
		"Indeterminado": "2.1.1.13.1.1",
		"Transitorio":   "2.1.1.13.1.2",
	},
	"DL 728": {
		"Permanentes":  "2.1.1.8.1.1",
		"A plazo fijo": "2.1.1.8.2.1",
	},
}

// MapRegimenToKey normaliza los códigos de régimen a las claves de nuestro mapa
func MapRegimenToKey(codigo string) string {
	switch codigo {
	case "276":
		return "DL 276"
	case "30057":
		return "Ley 30057"
	case "1057":
		return "DL 1057"
	case "728":
		return "DL 728"
	default:
		return ""
	}
}
