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
	TrabajadorID       int
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
	IncidenciasTexto   string
	Incidencias        []PersonalIncidenciaMes
}

type ConceptoReporte struct {
	Nombre string
	Monto  float64
}

type ConceptoFormulacionEspecial struct {
	ID                  int     `json:"id"`
	NombrePersonalizado string  `json:"nombre_personalizado"`
	CodigoSunat         string  `json:"codigo_sunat"`
	ClasificadorCodigo  string  `json:"clasificador_codigo"`
	EsOcasional         bool    `json:"es_ocasional"`
	EsExtraordinario    bool    `json:"es_extraordinario"`
	ModalidadEntrega    string  `json:"modalidad_entrega"`
	EsPensionable       bool    `json:"es_pensionable"`
	EsRemunerativa      bool    `json:"es_remunerativa"`
	MontoBase           float64 `json:"monto_base"`
}

type TrabajadorFormulacionEspecial struct {
	ContratoID               int                `json:"contrato_id"`
	NombreCompleto           string             `json:"nombre_completo"`
	NumeroDocumento          string             `json:"numero_documento"`
	PuestoNombre             string             `json:"puesto_nombre"`
	UnidadOrganicaNombre     string             `json:"unidad_organica_nombre"`
	RegimenNombre            string             `json:"regimen_nombre"`
	MetaCodigo               string             `json:"meta_codigo"`
	MetaDescripcion          string             `json:"meta_descripcion"`
	MontosCustom             map[string]float64 `json:"montos_custom"`
	MontoTotal               float64            `json:"monto_total"`
	TieneRetencionJudicial   bool               `json:"tiene_retencion_judicial"`
	PorcentajeJudicial       float64            `json:"porcentaje_judicial"`
	DetalleRetencionJudicial string             `json:"detalle_retencion_judicial"`
	MontoRetencionJudicial   float64            `json:"monto_retencion_judicial"`
	MontoNetoEstimado        float64            `json:"monto_neto_estimado"`
}
