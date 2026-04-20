package models

// ContextoCalculo representa el "maletín" de información temporal que viaja hacia las calculadoras
type ContextoCalculo struct {
	// 1. El Entorno
	ParametrosGlobales map[string]float64 // Ej: ["UIT"] = 5150.00, ["TASA_ESSALUD"] = 0.09

	// 2. El Trabajador
	RegimenCodigo   string // Ej: "CAS", "276", "728"
	TieneSuspension bool   // Para futura renta de 4ta

	// 3. El Dinero (Pasada 1)
	// Guardamos los ingresos ya calculados usando como llave el Código SUNAT
	IngresosProcesados map[string]float64 // Ej: ["0121"] = 2500.00, ["0105"] = 500.00

	// 4. El Tiempo
	MesActual int // Ej: 4 (Abril)

	// 5. El Historial (Para Renta 5ta)
	Retenciones5taPrevias       float64 // Suma de lo retenido en Ene-Mar
	RemuneracionNoMensual       float64 // Suma de gratificaciones, bonos, etc.
	IngresoExtraordinarioDelMes float64 // Bonos puntuales del mes
}

// BoletaResultado almacena el cálculo final de un trabajador en memoria antes de ir a la BD
type BoletaResultado struct {
	ContratoID       int
	TotalIngresos    float64
	TotalRetenciones float64
	TotalAportes     float64
	NetoPagar        float64
	// Aquí guardamos el detalle de rubros (Sueldo, EsSalud, AFP, etc.)
	LineasConceptos []PlanillaConcepto
}
