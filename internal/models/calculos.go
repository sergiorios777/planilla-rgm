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
	IngresosPrevios             float64 // Suma de ingresos de meses anteriores

	// 6. Pensiones
	RegimenPensionario string
	TasaAfpAporte      float64
	TasaAfpPrima       float64
	TasaAfpComision    float64
}

// BoletaResultado almacena el cálculo final de un trabajador en memoria antes de ir a la BD
type BoletaResultado struct {
	ContratoID       int
	TotalIngresos    float64
	TotalRetenciones float64
	TotalAportes     float64
	NetoPagar        float64
	// Aquí guardamos el detalle de rubros (Sueldo, EsSalud, AFP, etc.)
	LineasConceptos       []PlanillaConcepto
	OcurrenciasProcesadas []int
	TrabajadorNombreCompleto       string
	TrabajadorNumeroDocumento      string
	PuestoCodigoAirhsp             string
	PuestoNombre                   string
	OrganigramaDocumentoAprobacion string
	UnidadOrganicaNombre           string
	UnidadOrganicaTipo             string
	SueldoBasicoHistorico          float64
}

// OcurrenciaAsistencia representa una falta o tardanza extraída de la BD
type OcurrenciaAsistencia struct {
	ID       int
	Tipo     string
	Cantidad float64
}

// OcurrenciaVista es un DTO para mostrar las faltas en la tabla de la interfaz de usuario
type OcurrenciaVista struct {
	ID               int
	ContratoID       int
	TrabajadorNombre string // Apellidos y Nombres unidos
	Tipo             string
	FechaOcurrencia  string
	Cantidad         float64
	Procesado        bool
}

// ContratoSelect es un DTO ligero para llenar el desplegable <select> del formulario
type ContratoSelect struct {
	ID               int
	TrabajadorNombre string
	NumeroDocumento  string
}

// 2. Crea esta nueva estructura para almacenar la "fotografía" del mes
type TasasAFP struct {
	Aporte float64
	Flujo  float64
	Mixta  float64
	Prima  float64
}

// JobPlanilla es el "expediente" completo que recibe una Goroutine
type JobPlanilla struct {
	Contrato               ContratoPlanilla
	ConceptosPlaza         []ConceptoPlanilla // Pre-cargados
	Ocurrencias            []OcurrenciaAsistencia
	TasasAFP               TasasAFP
	Retenciones5taPrevias  float64 // Pre-cargado
	IngresosPrevios        float64
	MesActual              int
	Anio                   int
	ParametrosGlobales     map[string]float64
	MapaCodigos            map[string]int
	MapaAfectacionesGlobal map[int][]int
	ReglasFinanciamiento   []ReglaFinanciamientoConcepto
	DescuentosTrabajador   []DescuentoConConceptos
	Incidencias            []PersonalIncidenciaMes
}

// ResultPlanilla es lo que la Goroutine devuelve al terminar
type ResultPlanilla struct {
	Boleta BoletaResultado
	Error  error
}

type ConceptoCalculado struct {
	ID            int    `json:"id"`
	Nombre        string `json:"nombre"`
	Tipo          string `json:"tipo"`
	CodigoInterno string `json:"codigo_interno"`
}

type BaseRegimenDefault struct {
	ID                  int    `json:"id"`
	ConceptoCalculadoID int    `json:"concepto_calculado_id"`
	RegimenID           int    `json:"regimen_id"`
	ConceptoModeloID    int    `json:"concepto_modelo_id"`
	VariableCalculo     string `json:"variable_calculo"`
}

type BaseRegimenTenant struct {
	ID                  int    `json:"id"`
	TenantID            int    `json:"tenant_id"`
	ConceptoCalculadoID int    `json:"concepto_calculado_id"`
	RegimenID           int    `json:"regimen_id"`
	ConceptoTenantID    int    `json:"concepto_tenant_id"`
	VariableCalculo     string `json:"variable_calculo"`
	Activo              bool   `json:"activo"`
}
type BaseRegimenDefaultDTO struct {
	ID                  int    `json:"id"`
	ConceptoCalculadoID int    `json:"concepto_calculado_id"`
	RegimenID           int    `json:"regimen_id"`
	RegimenCodigo       string `json:"regimen_codigo"`
	RegimenDesc         string `json:"regimen_desc"`
	ConceptoModeloID    int    `json:"concepto_modelo_id"`
	ConceptoModeloDesc  string `json:"concepto_modelo_desc"`
	VariableCalculo     string `json:"variable_calculo"`
}

type ConceptoModeloDTO struct {
	ID                  int    `json:"id"`
	NombrePersonalizado string `json:"nombre_personalizado"`
}
