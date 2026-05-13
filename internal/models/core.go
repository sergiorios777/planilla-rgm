package models

import (
	"time"
)

// Tenant representa a las entidades/inquilinos que usan tu SaaS
type Tenant struct {
	ID           int       `json:"id"`
	Nombre       string    `json:"nombre"`
	Ruc          string    `json:"ruc"`
	Direccion    *string   `json:"direccion"`
	FraseGestion *string   `json:"frase_gestion"`
	LogoURL      *string   `json:"logo_url"`
	Slug         *string   `json:"slug"`
	Activo       bool      `json:"activo"`
	CreatedAt    time.Time `json:"created_at"`
}

// Usuario representa a las personas que inician sesión.
// El campo TenantID es un puntero (*int) para permitir valores "null" en la base de datos.
// Si TenantID es null, significa que es un usuario "Súper Admin" del SaaS.
type Usuario struct {
	ID           int    `json:"id"`
	TenantID     *int   `json:"tenant_id"` // Clave foránea hacia Tenant
	Nombre       string `json:"nombre"`
	Email        string `json:"email"`
	Password     string `json:"-"`   // El guion evita que la contraseña se envíe en respuestas JSON
	Rol          string `json:"rol"` // Ej: "admin_saas", "rrhh_local", "planillero"
	Activo       bool   `json:"activo"`
	TenantNombre string `json:"tenant_nombre,omitempty"`
}

// ClasificadorMEF representa el catálogo universal gestionado por el administrador SaaS.
// Nota: No tiene tenant_id porque es global y visible para todos los inquilinos.
type ClasificadorMEF struct {
	ID              int    `json:"id"`
	Anio            int    `json:"anio"`          // Ej: 2026
	CodigoOriginal  string `json:"codigo"`        // Ej: "2.0. 1  1. 1  1"
	CodigoLimpio    string `json:"codigo_limpio"` // Ej: "2.0.1.1.1.1"
	Descripcion     string `json:"descripcion"`
	Nivel           int    `json:"nivel"`            // 1 al 6
	TipoTransaccion string `json:"tipo_transaccion"` // "Gasto" o "Ingreso" (Para filtrar rápidamente)
	Activo          bool   `json:"activo"`           // Para variaciones intra-anuales
	ParentID        *int   `json:"parent_id"`
}

// ConceptoMaestro representa el catálogo global de ingresos, retenciones y aportes.
type ConceptoMaestro struct {
	ID          int    `json:"id"`
	ParentID    *int   `json:"parent_id"`   // Vinculación a un nivel superior (ej. 0100)
	Codigo      string `json:"codigo"`      // Ej: "0121" (Puede ser el código PDT/SUNAT si aplica)
	Descripcion string `json:"descripcion"` // Ej: "Sueldo Básico", "AFP Integra - Aporte", "EsSalud"
	Tipo        string `json:"tipo"`        // Ej: "Ingreso", "Retencion", "Aporte"
	Activo      bool   `json:"activo"`
}

// ConceptoAfectacion representa la tabla intermedia de relaciones.
// Ej: El Sueldo Básico (Base) está afecto a EsSalud (Derivado).
type ConceptoAfectacion struct {
	ID                 int `json:"id"`
	ConceptoBaseID     int `json:"concepto_base_id"`
	ConceptoDerivadoID int `json:"concepto_derivado_id"`
}

// ConceptoTenant es la configuración local (por municipalidad) de un concepto maestro
type ConceptoTenant struct {
	ID                  int    `json:"id"`
	TenantID            int    `json:"tenant_id"`
	ConceptoID          int    `json:"concepto_id"` // Apunta a conceptos_maestros
	NombrePersonalizado string `json:"nombre_personalizado"`
	FrecuenciaMeses     string `json:"frecuencia_meses"`
	ClasificadorID      *int   `json:"clasificador_id"`
	Activo              bool   `json:"activo"`
	EsExtraordinario    bool   `json:"es_extraordinario"`

	// Campos Auxiliares (JOINs desde la tabla conceptos_maestros)
	ConceptoCodigo     string `json:"concepto_codigo,omitempty"`
	ConceptoNombre     string `json:"concepto_nombre,omitempty"`
	ConceptoTipo       string `json:"concepto_tipo,omitempty"`
	ClasificadorCodigo string `json:"clasificador_codigo,omitempty"`
}

// ParametroGlobal define valores anuales que afectan a todas las planillas (UIT, RMV, etc.)
type ParametroGlobal struct {
	ID          int     `json:"id"`
	Clave       string  `json:"clave"`
	Valor       float64 `json:"valor"`
	FechaDesde  string  `json:"fecha_desde"` // Formato YYYY-MM-DD
	FechaHasta  *string `json:"fecha_hasta"` // Puntero para permitir nulos (vigente actualmente)
	Descripcion string  `json:"descripcion"`
}

// Trabajador representa a un empleado registrado en una municipalidad (Inquilino)
type Trabajador struct {
	ID              int    `json:"id"`
	TenantID        int    `json:"tenant_id"` // Clave de seguridad
	TipoDocumento   string `json:"tipo_documento"`
	NumeroDocumento string `json:"numero_documento"`
	Nombres         string `json:"nombres"`
	ApellidoPaterno string `json:"apellido_paterno"`
	ApellidoMaterno string `json:"apellido_materno"`
	FechaNacimiento string `json:"fecha_nacimiento"`
	Sexo            string `json:"sexo"`
	Activo          bool   `json:"activo"`
	// PENSIONES
	RegimenPensionario string `json:"regimen_pensionario"`
	AfpID              int    `json:"afp_id"`
	AfpTipoComision    string `json:"afp_tipo_comision"`
	Cuspp              string `json:"cuspp"` // NUEVO: Obligatorio para PLAME
}

// NombreCompleto es una función auxiliar útil para mostrar en la interfaz
func (t *Trabajador) NombreCompleto() string {
	return t.ApellidoPaterno + " " + t.ApellidoMaterno + ", " + t.Nombres
}

// RegimenLaboral representa el catálogo nacional de regímenes (276, CAS, etc.)
type RegimenLaboral struct {
	ID          int    `json:"id"`
	Codigo      string `json:"codigo"`
	Descripcion string `json:"descripcion"`
}

// Contrato vincula a un trabajador con un régimen y establece sus condiciones económicas
type Contrato struct {
	ID           int     `json:"id"`
	TenantID     int     `json:"tenant_id"`
	TrabajadorID int     `json:"trabajador_id"`
	PuestoID     int     `json:"puesto_id"`
	FechaInicio  string  `json:"fecha_inicio"`
	FechaFin     *string `json:"fecha_fin"` // Puntero para permitir nulos
	Activo       bool    `json:"activo"`

	// Campos auxiliares para la interfaz web (JOINs)
	TrabajadorNombre    string  `json:"trabajador_nombre,omitempty"`
	TrabajadorDoc       string  `json:"trabajador_doc,omitempty"`
	PuestoNombre        string  `json:"puesto_nombre,omitempty"`
	RegimenDesc         string  `json:"regimen_desc,omitempty"`
	SueldoPresupuestado float64 `json:"sueldo_presupuestado,omitempty"`
}

// ContratoPlanilla representa los datos básicos de un trabajador para el cálculo
type ContratoPlanilla struct {
	ID                 int
	PuestoID           int
	Regimen            string
	RegimenPensionario string
	AfpID              int
	AfpTipoComision    string
	FechaInicio        time.Time
	FechaFin           *time.Time
}

// ConceptoPlanilla representa un rubro de la estructura de costos del puesto
type ConceptoPlanilla struct {
	ID               int
	TenantID         int
	MaestroID        int
	MaestroCodigo    string
	Tipo             string
	Monto            float64
	Frecuencia       string
	ParentID         int
	EsExtraordinario bool

	// Agregado para el PAP
	Nombre string // Nombre personalizado del concepto
}

// FuenteRubro representa el catálogo del MEF de fuentes de financiamiento
type FuenteRubro struct {
	ID                   int    `json:"id"`
	Anio                 int    `json:"anio"`
	FuenteFinanciamiento string `json:"fuente_financiamiento"`
	Rubro                string `json:"rubro"`
	Activo               bool   `json:"activo"`
}

// MetaPresupuestal representa un proyecto o actividad del año fiscal
type MetaPresupuestal struct {
	ID          int    `json:"id"`
	TenantID    int    `json:"tenant_id"`
	Anio        int    `json:"anio"`
	Codigo      string `json:"codigo"`
	Descripcion string `json:"descripcion"`
	Activo      bool   `json:"activo"`
}

// Puesto (Plaza) representa una "silla" dentro de la municipalidad, esté ocupada o vacante.
// Es la base para el cálculo del Presupuesto Anual.
type Puesto struct {
	ID                  int `json:"id"`
	TenantID            int `json:"tenant_id"`
	MetaID              int `json:"meta_id"`
	FuenteRubroID       int `json:"fuente_rubro_id"`
	RegimenID           int `json:"regimen_id"`
	RegimenCodigo       string
	Nombre              string  `json:"nombre"`
	SueldoPresupuestado float64 `json:"sueldo_presupuestado"`
	Estado              string  `json:"estado"` // VACANTE u OCUPADO
	Activo              bool    `json:"activo"`
	EsDietario          bool    `json:"es_dietario"`

	// Campos auxiliares para pintar tablas dinámicas sin hacer múltiples consultas
	MetaCodigo       string `json:"meta_codigo,omitempty"`
	FuenteRubroDesc  string `json:"fuente_rubro_desc,omitempty"`
	RegimenDesc      string `json:"regimen_desc,omitempty"`
	RequiereRevision bool
}

// PuestoConcepto es el detalle de qué conceptos arman el costo de una Plaza específica
type PuestoConcepto struct {
	ID               int      `json:"id"`
	PuestoID         int      `json:"puesto_id"`
	ConceptoTenantID int      `json:"concepto_tenant_id"`
	Monto            *float64 `json:"monto"`
	Activo           bool     `json:"activo"`

	// Campos Auxiliares para pintar la UI
	NombrePersonalizado string `json:"nombre_personalizado,omitempty"`
	ConceptoTipo        string `json:"concepto_tipo,omitempty"`
	Clasificador        string `json:"clasificador,omitempty"`
	MaestroCodigo       string `json:"maestro_codigo,omitempty"`
	RequiereMontoManual bool   `json:"requiere_monto_manual,omitempty"`
	MontoIngresado      bool   `json:"monto_ingresado,omitempty"`
}

// Planilla representa la cabecera mensual de cálculos
type Planilla struct {
	ID          int    `json:"id"`
	TenantID    int    `json:"tenant_id"`
	Anio        int    `json:"anio"`
	Mes         int    `json:"mes"`
	Descripcion string `json:"descripcion"`
	Estado      string `json:"estado"` // BORRADOR, CERRADA
}

// PlanillaDetalle representa la boleta consolidada de un trabajador
type PlanillaDetalle struct {
	ID               int     `json:"id"`
	PlanillaID       int     `json:"planilla_id"`
	ContratoID       int     `json:"contrato_id"`
	TotalIngresos    float64 `json:"total_ingresos"`
	TotalRetenciones float64 `json:"total_retenciones"`
	TotalAportes     float64 `json:"total_aportes"`
	NetoPagar        float64 `json:"neto_pagar"`

	// Campos auxiliares para pintar la boleta (JOINs)
	TrabajadorNombre string `json:"trabajador_nombre,omitempty"`
	TrabajadorDoc    string `json:"trabajador_doc,omitempty"`
	PuestoNombre     string `json:"puesto_nombre,omitempty"`
	RegimenDesc      string `json:"regimen_desc,omitempty"`
}

// PlanillaConcepto es el desglose rubro por rubro
type PlanillaConcepto struct {
	ID                int     `json:"id"`
	PlanillaDetalleID int     `json:"planilla_detalle_id"`
	ConceptoTenantID  int     `json:"concepto_tenant_id"`
	TipoConcepto      string  `json:"tipo_concepto"` // INGRESO, RETENCION, APORTE
	Monto             float64 `json:"monto"`
	MaestroID         int     `json:"maestro_id"`

	// Campo auxiliar
	NombrePersonalizado string `json:"nombre_personalizado,omitempty"`
}

// PuestoPAP es un DTO exclusivo para extraer los datos descriptivos del reporte
type PuestoPAP struct {
	ID                     int
	RegimenCodigo          string
	MetaCodigo             string
	MetaDescripcion        string
	FuenteRubroCodigo      string
	FuenteRubroDescripcion string
	TotalIngresosAnual     float64
	TotalAportesAnual      float64
}

// PapVersion representa la cabecera de la proyección anual
type PapVersion struct {
	ID              int
	TenantID        int
	Anio            int
	Tipo            string
	FechaGeneracion string
	Estado          string
}

// PapDetalle representa una fila de la matriz de proyección (Meta x Fuente x Clasificador)
type PapDetalle struct {
	ID                       int
	VersionID                int
	MetaCodigo               string
	MetaDescripcion          string
	FuenteRubroCodigo        string
	FuenteRubroDescripcion   string
	ClasificadorCodigoLimpio string
	ClasificadorDescripcion  string
	Meses                    [12]float64
	TotalAnual               float64
}

// ConceptoModelo representa la plantilla base para cada régimen laboral
type ConceptoModelo struct {
	ID                  int    `json:"id"`
	ConceptoID          int    `json:"concepto_id"`
	NombrePersonalizado string `json:"nombre_personalizado"`
	FrecuenciaMeses     string `json:"frecuencia_meses"`
	ClasificadorID      *int   `json:"clasificador_id"` // Es puntero porque puede ser nulo en la BD
	EsExtraordinario    bool   `json:"es_extraordinario"`
	RequiereMonto       bool   `json:"requiere_monto"`
	CreatedAt           string `json:"created_at"`

	// Campos "virtuales" obtenidos mediante JOINs para la interfaz de usuario
	RegimenesIDs        []int  `json:"regimenes_ids"` // Para los checkboxes (POST/PUT)
	RegimenesNombres    string `json:"regimenes_nombres,omitempty"`
	ConceptoCodigo      string `json:"concepto_codigo,omitempty"`
	ConceptoDescripcion string `json:"concepto_descripcion,omitempty"`
	ClasificadorCodigo  string `json:"clasificador_codigo,omitempty"`
}

// ConceptoAsignacion representa la estructura temporal para la vista de 
// asignación manual de conceptos a un puesto específico.
type ConceptoAsignacion struct {
	ConceptoTenantID int     `json:"concepto_tenant_id"`
	Nombre           string  `json:"nombre"`
	Tipo             string  `json:"tipo"`           // INGRESO, RETENCION, APORTE
	RequiereMonto    bool    `json:"requiere_monto"`
	Asignado         bool    `json:"asignado"`       // Define si el switch estará encendido
	Monto            float64 `json:"monto"`          // El valor actual (si aplica)
}