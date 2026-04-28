package models

type DatosReportePlanilla struct {
	TenantNombre     string
	TenantRUC        string
	PlanillaAnio     int
	PlanillaMes      int
	PlanillaDesc     string
	Boletas          []*BoletaReporte // Usamos punteros para poder modificar los slices internos
	TotalIngresos    float64
	TotalRetenciones float64
	TotalAportes     float64
	TotalNeto        float64
}

type BoletaReporte struct {
	DetalleID        int
	TrabajadorDoc    string
	TrabajadorNombre string
	Cargo            string
	Regimen          string
	TotalIngresos    float64
	TotalRetenciones float64
	TotalAportes     float64
	NetoPagar        float64
	Ingresos         []ConceptoReporte
	Retenciones      []ConceptoReporte
	Aportes          []ConceptoReporte
}

type ConceptoReporte struct {
	Nombre string
	Monto  float64
}
