package models

type Reporte struct {
	ID          string `json:"id"`          // Ej: "trab_padron"
	Modulo      string `json:"modulo"`      // Ej: "TRABAJADORES", "PUESTOS"
	Nombre      string `json:"nombre"`      // Ej: "Padrón General de Personal"
	Descripcion string `json:"descripcion"` // Ej: "Lista detallada de todo el personal activo..."
}

type DatosReportePlanilla struct {
	TenantNombre     string
	TenantRUC        string
	TenantLogoURL    string
	PlanillaAnio     int
	PlanillaMes      int
	PlanillaDesc     string
	PlanillaEstado   string
	Boletas          []*BoletaReporte // Usamos punteros para poder modificar los slices internos
	TotalIngresos    float64
	TotalRetenciones float64
	TotalAportes     float64
	TotalNeto        float64
}

type BoletaReporte struct {
	DetalleID          int
	TrabajadorDoc      string
	TrabajadorNombre   string
	Cargo              string
	Regimen            string
	Direccion          string
	Sexo               string
	FechaNacimiento    string
	FechaIngreso       string
	FechaCese          string
	RegimenPensionario string
	AfpNombre          string
	Cuspp              string
	TotalIngresos      float64
	TotalRetenciones   float64
	TotalAportes       float64
	NetoPagar          float64
	Ingresos           []ConceptoReporte
	Retenciones        []ConceptoReporte
	Aportes            []ConceptoReporte
}

type ConceptoReporte struct {
	Nombre string
	Monto  float64
}
