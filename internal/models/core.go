package models

import (
	"time"
)

const (
	ModalidadEntregaPermanente  = "PERMANENTE"
	ModalidadEntregaPeriodico   = "PERIODICO"
	ModalidadEntregaExcepcional = "EXCEPCIONAL"
	ModalidadEntregaOcasional   = "OCASIONAL"
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
	TipoEntidad  string    `json:"tipo_entidad"`
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
	ID            int    `json:"id"`
	ParentID      *int   `json:"parent_id"`   // Vinculación a un nivel superior (ej. 0100)
	Codigo        string `json:"codigo"`      // Ej: "0121" (Puede ser el código PDT/SUNAT si aplica)
	CodigoInterno string `json:"codigo_interno"` // Código de uso interno del motor de cálculos
	Descripcion   string `json:"descripcion"` // Ej: "Sueldo Básico", "AFP Integra - Aporte", "EsSalud"
	Tipo          string `json:"tipo"`        // Ej: "Ingreso", "Retencion", "Aporte"
	Activo        bool   `json:"activo"`
	Origen        string `json:"origen"`      // 'sunat' o 'interno'
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
	ID                       int    `json:"id"`
	TenantID                 int    `json:"tenant_id"`
	ConceptoID               int    `json:"concepto_id"` // Apunta a conceptos_maestros
	ModeloID                 *int   `json:"modelo_id"`   // Apunta a conceptos_modelo
	NombrePersonalizado      string `json:"nombre_personalizado"`
	FrecuenciaMeses          string `json:"frecuencia_meses"`
	ClasificadorID           *int   `json:"clasificador_id"`
	Activo                   bool   `json:"activo"`
	EsExtraordinario         bool   `json:"es_extraordinario"`
	EsPensionable            bool   `json:"es_pensionable"`
	EsRemunerativa           bool   `json:"es_remunerativa"`
	EsBaseCts                bool   `json:"es_base_cts"`
	EsBaseBeneficiosSociales bool   `json:"es_base_beneficios_sociales"`
	EsOcasional              bool     `json:"es_ocasional"`
	EsAfectoCargasSociales   bool     `json:"es_afecto_cargas_sociales"`
	ModalidadEntrega         string   `json:"modalidad_entrega"`
	BaseCalculoPara          []string `json:"base_calculo_para"`

	// Campos Auxiliares (JOINs desde la tabla conceptos_maestros)
	RegimenesIDs       []int  `json:"regimenes_ids"` // Para los checkboxes (POST/PUT)
	RegimenesCodigos   string `json:"regimenes_codigos,omitempty"`
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
	ID                 int    `json:"id"`
	TenantID           int    `json:"tenant_id"` // Clave de seguridad
	TipoDocumento      string `json:"tipo_documento"`
	NumeroDocumento    string `json:"numero_documento"`
	Nombres            string `json:"nombres"`
	ApellidoPaterno    string `json:"apellido_paterno"`
	ApellidoMaterno    string `json:"apellido_materno"`
	FechaNacimiento    string `json:"fecha_nacimiento"`
	FechaIngreso       string `json:"fecha_ingreso"`
	FechaCese          string `json:"fecha_cese"`
	Direccion          string `json:"direccion"`
	Banco              string `json:"banco"`
	Cuenta             string `json:"cuenta"`
	Cci                string `json:"cci"`
	Sexo               string `json:"sexo"`
	Activo             bool   `json:"activo"`
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

// Edad calcula la edad del trabajador basándose en su fecha de nacimiento
func (t *Trabajador) Edad() int {
	if t.FechaNacimiento == "" {
		return 0
	}
	dob, err := time.Parse("2006-01-02", t.FechaNacimiento)
	if err != nil {
		return 0
	}
	now := time.Now()
	years := now.Year() - dob.Year()
	if now.YearDay() < dob.YearDay() {
		years--
	}
	return years
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
	TipoContrato string  `json:"tipo_contrato"`
	Nivel        string  `json:"nivel"`
	MotivoBaja   *string `json:"motivo_baja,omitempty"` // Campo para indicar motivo del cese del contrato

	// Campos auxiliares para la interfaz web (JOINs)
	TrabajadorNombre    string  `json:"trabajador_nombre,omitempty"`
	TrabajadorDoc       string  `json:"trabajador_doc,omitempty"`
	PuestoNombre        string  `json:"puesto_nombre,omitempty"`
	RegimenDesc         string  `json:"regimen_desc,omitempty"`
	SueldoPresupuestado float64 `json:"sueldo_presupuestado,omitempty"`
}

// ContratoPlanilla representa los datos básicos de un trabajador para el cálculo
type ContratoPlanilla struct {
	ID                             int
	TenantID                       int
	TrabajadorID                   int
	PuestoID                       int
	RegimenID                      int
	Regimen                        string
	RegimenPensionario             string
	AfpID                          int
	AfpTipoComision                string
	FechaInicio                    time.Time
	FechaFin                       *time.Time
	TrabajadorNombreCompleto       string
	TrabajadorNumeroDocumento      string
	PuestoNombre                   string
	PuestoCodigoAirhsp             string
	OrganigramaDocumentoAprobacion string
	UnidadOrganicaNombre           string
	UnidadOrganicaTipo             string
	SueldoBasicoHistorico          float64
	MetaID                         *int
	FuenteRubroID                  *int
}

// ConceptoPlanilla representa un rubro de la estructura de costos del puesto
type ConceptoPlanilla struct {
	ID               int
	TenantID         int
	ConceptoTenantID int
	MaestroID        int
	CodigoInterno    string // Código de uso interno del motor de cálculos
	CodigoSunat      string // Código original SUNAT PLAME
	Tipo             string
	Monto            float64
	Frecuencia       string
	ParentID         int
	EsExtraordinario bool

	// Agregado para el PAP
	Nombre string // Nombre personalizado del concepto
}

// PlanillaConcepto es el desglose rubro por rubro
type PlanillaConcepto struct {
	ID                int     `json:"id"`
	PlanillaDetalleID int     `json:"planilla_detalle_id"`
	ConceptoTenantID  *int    `json:"concepto_tenant_id"`
	MetaID            *int    `json:"meta_id"`
	FuenteRubroID     *int    `json:"fuente_rubro_id"`
	TipoConcepto      string  `json:"tipo_concepto"` // INGRESO, RETENCION, APORTE
	Monto             float64 `json:"monto"`
	MaestroID         int     `json:"maestro_id"`
	CodigoSunat       string  `json:"codigo_sunat"`
	NombreEnBoleta    string  `json:"nombre_en_boleta"`

	// Campos auxiliares
	NombrePersonalizado    string `json:"nombre_personalizado,omitempty"`
	MetaCodigo             string `json:"meta_codigo,omitempty"`
	MetaDescripcion        string `json:"meta_descripcion,omitempty"`
	FuenteRubroCodigo      string `json:"fuente_rubro_codigo,omitempty"`
	FuenteRubroDescripcion string `json:"fuente_rubro_descripcion,omitempty"`
	ClasificadorCodigo     string `json:"clasificador_codigo,omitempty"`
}

// ConceptoSunatAgrupado representa el consolidado de un concepto dentro de una planilla para auditoría SUNAT / PLAME
type ConceptoSunatAgrupado struct {
	ConceptoTenantID       *int    `json:"concepto_tenant_id"`
	MaestroID              int     `json:"maestro_id"`
	CodigoSunatActual      string  `json:"codigo_sunat_actual"`
	DescripcionSunatActual string  `json:"descripcion_sunat_actual"`
	NombreConcepto         string  `json:"nombre_concepto"`
	TipoConcepto           string  `json:"tipo_concepto"` // INGRESO, RETENCION, APORTE
	TotalTrabajadores      int     `json:"total_trabajadores"`
	TotalMonto             float64 `json:"total_monto"`
	TotalDevengado         float64 `json:"total_devengado"`
	TotalPagado            float64 `json:"total_pagado"`
	TieneAjustesManuales   bool    `json:"tiene_ajustes_manuales"`
	TieneVacacional        bool    `json:"tiene_vacacional"`
	MaestroIDOriginal      int     `json:"maestro_id_original"`
}

// PlanillaPlameConcepto representa una fila del snapshot tributario planilla_plame_conceptos
type PlanillaPlameConcepto struct {
	ID                   int       `json:"id"`
	PlanillaID           int       `json:"planilla_id"`
	PlanillaDetalleID    int       `json:"planilla_detalle_id"`
	TrabajadorID         int       `json:"trabajador_id"`
	PlanillaConceptoID   *int      `json:"planilla_concepto_id,omitempty"`
	CodigoSunat          string    `json:"codigo_sunat"`
	DescripcionSunat     string    `json:"descripcion_sunat"`
	TipoConcepto         string    `json:"tipo_concepto"` // INGRESO, RETENCION, APORTE
	MontoDevengado       float64   `json:"monto_devengado"`
	MontoPagado          float64   `json:"monto_pagado"`
	EsConceptoVacacional bool      `json:"es_concepto_vacacional"`
	EsAjusteManual       bool      `json:"es_ajuste_manual"`
	ObservacionAjuste    string    `json:"observacion_ajuste,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`

	// Campos auxiliares para renderizado en UI
	TrabajadorNombre      string `json:"trabajador_nombre,omitempty"`
	TrabajadorDocumento   string `json:"trabajador_documento,omitempty"`
	TrabajadorTipoDoc     string `json:"trabajador_tipo_doc,omitempty"`
	RegimenNombre         string `json:"regimen_nombre,omitempty"`
	ConceptoLaboralNombre string `json:"concepto_laboral_nombre,omitempty"`
}

// PlameTrabajadorConceptoItem representa un colaborador asociado a un código SUNAT para la vista detallada
type PlameTrabajadorConceptoItem struct {
	PlanillaPlameConceptoID int     `json:"planilla_plame_concepto_id"`
	PlanillaDetalleID       int     `json:"planilla_detalle_id"`
	TrabajadorID            int     `json:"trabajador_id"`
	TipoDocumento           string  `json:"tipo_documento"`
	NumeroDocumento         string  `json:"numero_documento"`
	NombreCompleto          string  `json:"nombre_completo"`
	RegimenNombre           string  `json:"regimen_nombre"`
	CodigoSunat             string  `json:"codigo_sunat"`
	DescripcionSunat        string  `json:"descripcion_sunat"`
	TipoConcepto            string  `json:"tipo_concepto"`
	MontoDevengado          float64 `json:"monto_devengado"`
	MontoPagado             float64 `json:"monto_pagado"`
	EsConceptoVacacional    bool    `json:"es_concepto_vacacional"`
	EsAjusteManual          bool    `json:"es_ajuste_manual"`
	ObservacionAjuste       string  `json:"observacion_ajuste"`
	ConceptoLaboralNombre   string  `json:"concepto_laboral_nombre"`
}

// PlamePlanillaResumenItem representa una planilla calculada del periodo con su estado de snapshot para el Hub PLAME
type PlamePlanillaResumenItem struct {
	PlanillaID           int     `json:"planilla_id"`
	Anio                 int     `json:"anio"`
	Mes                  int     `json:"mes"`
	TipoPlanilla         string  `json:"tipo_planilla"`
	Descripcion          string  `json:"descripcion"`
	EstadoPlanilla       string  `json:"estado_planilla"` // BORRADOR, CERRADA
	TotalTrabajadores    int     `json:"total_trabajadores"`
	TotalDevengado       float64 `json:"total_devengado"`
	TotalPagado          float64 `json:"total_pagado"`
	TieneSnapshot        bool    `json:"tiene_snapshot"`
	TieneAjustesManuales bool    `json:"tiene_ajustes_manuales"`
	TieneRemVacacional   bool    `json:"tiene_rem_vacacional"`
	TotalConceptosSunat  int     `json:"total_conceptos_sunat"`
}

// PlameTrabajadorPadronItem representa un colaborador en el padrón consolidado de auditoría PLAME
type PlameTrabajadorPadronItem struct {
	PlanillaDetalleID int     `json:"planilla_detalle_id"`
	TrabajadorID      int     `json:"trabajador_id"`
	TipoDocumento     string  `json:"tipo_documento"`
	NumeroDocumento   string  `json:"numero_documento"`
	NombreCompleto    string  `json:"nombre_completo"`
	RegimenNombre     string  `json:"regimen_nombre"`
	TotalDevengado    float64 `json:"total_devengado"`
	TotalPagado       float64 `json:"total_pagado"`
	TotalConceptos    int     `json:"total_conceptos"`
	TieneAjusteManual bool    `json:"tiene_ajuste_manual"`
}

// PlameConceptoNominaItem representa un concepto institucional de nómina y su mapeo al snapshot PLAME
type PlameConceptoNominaItem struct {
	ConceptoTenantID  int     `json:"concepto_tenant_id"`
	ConceptoNombre    string  `json:"concepto_nombre"`
	TipoConcepto      string  `json:"tipo_concepto"`
	CodigoSunat       string  `json:"codigo_sunat"`
	DescripcionSunat  string  `json:"descripcion_sunat"`
	TotalTrabajadores int     `json:"total_trabajadores"`
	TotalDevengado    float64 `json:"total_devengado"`
	TotalPagado       float64 `json:"total_pagado"`
	TieneAjusteManual bool    `json:"tiene_ajuste_manual"`
}

// PlameHubResumen agrupa los KPIs globales del periodo mensual para el Hub PLAME
type PlameHubResumen struct {
	Anio                    int     `json:"anio"`
	Mes                     int     `json:"mes"`
	TotalPlanillas          int     `json:"total_planillas"`
	TotalTrabajadores       int     `json:"total_trabajadores"`
	TotalDevengado          float64 `json:"total_devengado"`
	TotalPagado             float64 `json:"total_pagado"`
	TotalVacaciones         int     `json:"total_vacaciones"`
	TotalLicenciasConGoce   int     `json:"total_licencias_con_goce"`
	TotalLicenciasSinGoce   int     `json:"total_licencias_sin_goce"`
	AlertaVacacionesSin0118 bool    `json:"alerta_vacaciones_sin_0118"`
	PlanillasListas         int     `json:"planillas_listas"`
	PlanillasConAjustes     int     `json:"planillas_con_ajustes"`
}

// ReglaFinanciamientoConcepto representa una regla de excepción de financiamiento por concepto
type ReglaFinanciamientoConcepto struct {
	ID               int       `json:"id"`
	TenantID         int       `json:"tenant_id"`
	ConceptoTenantID int       `json:"concepto_tenant_id"`
	RegimenID        *int      `json:"regimen_id"`
	MetaID           *int      `json:"meta_id"`
	FuenteRubroID    *int      `json:"fuente_rubro_id"`
	Activo           bool      `json:"activo"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`

	// Campos auxiliares para JOINs / UI
	ConceptoNombre         string `json:"concepto_nombre,omitempty"`
	RegimenNombre          string `json:"regimen_nombre,omitempty"`
	MetaCodigo             string `json:"meta_codigo,omitempty"`
	MetaDescripcion        string `json:"meta_descripcion,omitempty"`
	FuenteRubroCodigo      string `json:"fuente_rubro_codigo,omitempty"`
	FuenteRubroDescripcion string `json:"fuente_rubro_descripcion,omitempty"`
}

// EntidadFinanciera representa el catálogo bancario oficial (Tabla 3 SUNAT)
type EntidadFinanciera struct {
	ID        int       `json:"id"`
	Codigo    string    `json:"codigo"`
	Nombre    string    `json:"nombre"`
	Activo    bool      `json:"activo"`
	CreatedAt time.Time `json:"created_at"`
}

// Descuento representa un mandato judicial o descuento convencional/voluntario de un trabajador
type Descuento struct {
	ID                 int        `json:"id"`
	TenantID           int        `json:"tenant_id"`
	TrabajadorID       int        `json:"trabajador_id"`
	ConceptoTenantID   int        `json:"concepto_tenant_id"`
	TipoDescuento      string     `json:"tipo_descuento"`      // 'JUDICIAL', 'SINDICAL', 'PRESTAMO', 'CONVENIO', 'OTROS'
	DocumentoOrdenador string     `json:"documento_ordenador"`  // 'SENTENCIA', 'RESOLUCION', 'CONTRATO', 'OTRO'
	DetalleDocumento   string     `json:"detalle_documento"`   // Ej: "Expediente N° 00234-2024-0-1801-JP-FC-01"
	Descripcion        string     `json:"descripcion"`         // Ej: "Retención de Alimentos a favor de Juanita Pérez"
	TipoCalculo        string     `json:"tipo_calculo"`        // 'PORCENTAJE', 'MONTO_FIJO'
	BaseCalculo        string     `json:"base_calculo"`        // 'NETO_LEY', 'BRUTO_AFECTO'
	Porcentaje         float64    `json:"porcentaje"`
	MontoFijo          float64    `json:"monto_fijo"`
	MontoTotalDeuda    float64    `json:"monto_total_deuda"`
	MontoAcumulado     float64    `json:"monto_acumulado"`
	CuotasTotales      int        `json:"cuotas_totales"`
	CuotaActual        int        `json:"cuota_actual"`
	InicioVigencia     time.Time  `json:"inicio_vigencia"`
	FinVigencia        *time.Time `json:"fin_vigencia"`
	Activo             bool       `json:"activo"`
	MotivoBaja         string     `json:"motivo_baja"`

	// Beneficiario
	BeneficiarioTipoDocumento   string `json:"beneficiario_tipo_documento"`
	BeneficiarioNumeroDocumento string `json:"beneficiario_numero_documento"`
	BeneficiarioNombre          string `json:"beneficiario_nombre"`
	EntidadFinancieraID         *int   `json:"entidad_financiera_id"`
	BeneficiarioCuenta          string `json:"beneficiario_cuenta"`
	BeneficiarioCCI             string `json:"beneficiario_cci"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Campos auxiliares para UI y cálculo (JOINs)
	TrabajadorNombreCompleto  string `json:"trabajador_nombre_completo,omitempty"`
	TrabajadorNumeroDocumento string `json:"trabajador_numero_documento,omitempty"`
	ConceptoNombre            string `json:"concepto_nombre,omitempty"`
	ConceptoCodigoSunat       string `json:"concepto_codigo_sunat,omitempty"`
	ConceptoMaestroID         int    `json:"concepto_maestro_id,omitempty"`
	EntidadFinancieraNombre   string `json:"entidad_financiera_nombre,omitempty"`
	ConceptosAfectosIDs       []int  `json:"conceptos_afectos_ids,omitempty"`
	ConceptosAfectosNombres   string `json:"conceptos_afectos_nombres,omitempty"`
}

// DescuentoConcepto representa un concepto de ingreso que integra la base computable de un descuento
type DescuentoConcepto struct {
	ID               int       `json:"id"`
	DescuentoID      int       `json:"descuento_id"`
	ConceptoTenantID int       `json:"concepto_tenant_id"`
	CreatedAt        time.Time `json:"created_at"`
}

// DescuentoConConceptos agrupa el descuento y sus conceptos de base imponible para el motor de cálculo
type DescuentoConConceptos struct {
	Descuento          Descuento
	ConceptosTenantIDs []int
}

// DescuentoFiltroDTO contiene parámetros de filtrado para el listado de descuentos
type DescuentoFiltroDTO struct {
	TrabajadorID  *int
	TipoDescuento string
	Estado        string // "TODOS", "ACTIVOS", "INACTIVOS"
	Busqueda      string
}

// DescuentoResumenKPI contiene métricas para el encabezado Bento Grid
type DescuentoResumenKPI struct {
	TotalActivos      int
	TotalJudiciales   int
	TotalSindicales   int
	TotalPrestamos    int
	TotalTrabajadores int
}

// InfoTrabajadorPuesto contiene la información contractual, puesto, régimen y conceptos asignados de un colaborador
type InfoTrabajadorPuesto struct {
	TrabajadorID  int
	ContratoID    int
	PuestoID      int
	PuestoNombre  string
	RegimenID     int
	RegimenCodigo string
	RegimenNombre string
	TieneContrato bool
	Conceptos     []ConceptoPuestoDTO
}

// ConceptoPuestoDTO representa un concepto de ingreso asociado al puesto del trabajador
type ConceptoPuestoDTO struct {
	ConceptoTenantID    int
	ConceptoID          int
	ConceptoCodigo      string
	NombrePersonalizado string
	Monto               float64
	Seleccionado        bool
}

func (d Descuento) GetEntidadFinancieraID() int {
	if d.EntidadFinancieraID != nil {
		return *d.EntidadFinancieraID
	}
	return 0
}

func (r ReglaFinanciamientoConcepto) GetMetaID() int {
	if r.MetaID != nil {
		return *r.MetaID
	}
	return 0
}

func (r ReglaFinanciamientoConcepto) GetFuenteRubroID() int {
	if r.FuenteRubroID != nil {
		return *r.FuenteRubroID
	}
	return 0
}

func (r ReglaFinanciamientoConcepto) GetRegimenID() int {
	if r.RegimenID != nil {
		return *r.RegimenID
	}
	return 0
}

// FuenteRubro representa el catálogo del MEF de fuentes de financiamiento
type FuenteRubro struct {
	ID                   int    `json:"id"`
	Anio                 int    `json:"anio"`
	FuenteFinanciamiento string `json:"fuente_financiamiento"`
	Rubro                string `json:"rubro"`
	CodigoFuenteRubro    string `json:"codigo_fuente_rubro"`
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

// Organigrama representa la ordenanza o estructura de oficinas vigente
type Organigrama struct {
	ID                  int       `json:"id"`
	TenantID            int       `json:"tenant_id"`
	DocumentoAprobacion string    `json:"documento_aprobacion"`
	Descripcion         string    `json:"descripcion"`
	FechaVigencia       time.Time `json:"fecha_vigencia"`
	Activo              bool      `json:"activo"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// UnidadOrganica es un nodo de la estructura orgánica en la base de datos
type UnidadOrganica struct {
	ID            int       `json:"id"`
	TenantID      int       `json:"tenant_id"`
	OrganigramaID int       `json:"organigrama_id"`
	ParentID      *int      `json:"parent_id"`
	CodigoMef     string    `json:"codigo_mef"`
	Nombre        string    `json:"nombre"`
	Tipo          string    `json:"tipo"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// UnidadNodo representa un nodo procesado listo para renderizar jerárquicamente en la UI
type UnidadNodo struct {
	ID             int          `json:"id"`
	Nombre         string       `json:"nombre"`
	Tipo           string       `json:"tipo"`
	CodigoMef      string       `json:"codigo_mef"`
	ParentID       *int         `json:"parent_id"`
	TotalPuestos   int          `json:"total_puestos"`
	PuestosPropios int          `json:"puestos_propios"`
	Hijos          []UnidadNodo `json:"hijos"`
}

// Puesto (Plaza) representa una "silla" dentro de la municipalidad, esté ocupada o vacante.
// Es la base para el cálculo del Presupuesto Anual.
type Puesto struct {
	ID                  int     `json:"id"`
	TenantID            int     `json:"tenant_id"`
	MetaID              int     `json:"meta_id"`
	FuenteRubroID       int     `json:"fuente_rubro_id"`
	RegimenID           int     `json:"regimen_id"`
	RegimenCodigo       string  `json:"-"`
	Nombre              string  `json:"nombre"`
	SueldoPresupuestado float64 `json:"sueldo_presupuestado"`
	Estado              string  `json:"estado"` // VACANTE u OCUPADO
	Activo              bool    `json:"activo"`
	EsDietario          bool    `json:"es_dietario"`
	UnidadOrganicaID    *int    `json:"unidad_organica_id,omitempty"`
	CodigoAirhsp        *string `json:"codigo_airhsp,omitempty"`

	// Campos auxiliares para pintar tablas dinámicas sin hacer múltiples consultas
	MetaCodigo           string `json:"meta_codigo,omitempty"`
	FuenteRubroDesc      string `json:"fuente_rubro_desc,omitempty"`
	RegimenDesc          string `json:"regimen_desc,omitempty"`
	UnidadOrganicaNombre string `json:"unidad_organica_nombre,omitempty"`
	RequiereRevision     bool   `json:"-"`
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

// ReglaFinanciamientoModelo representa una regla de financiamiento a nivel SaaS (Conceptos Modelo)
type ReglaFinanciamientoModelo struct {
	ID               int       `json:"id"`
	ConceptoModeloID int       `json:"concepto_modelo_id"`
	RegimenID        *int      `json:"regimen_id"`
	MetaID           *int      `json:"meta_id"`
	FuenteRubroID    *int      `json:"fuente_rubro_id"`
	Activo           bool      `json:"activo"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`

	// Campos auxiliares para JOINs / UI
	ConceptoNombre         string `json:"concepto_nombre,omitempty"`
	RegimenNombre          string `json:"regimen_nombre,omitempty"`
	MetaCodigo             string `json:"meta_codigo,omitempty"`
	MetaDescripcion        string `json:"meta_descripcion,omitempty"`
	FuenteRubroCodigo      string `json:"fuente_rubro_codigo,omitempty"`
	FuenteRubroDescripcion string `json:"fuente_rubro_descripcion,omitempty"`
}

// Planilla representa la cabecera mensual de cálculos
type Planilla struct {
	ID               int     `json:"id"`
	TenantID         int     `json:"tenant_id"`
	Anio             int     `json:"anio"`
	Mes              int     `json:"mes"`
	Descripcion      string  `json:"descripcion"`
	Estado           string  `json:"estado"` // BORRADOR, CERRADA
	EsExtraordinaria bool    `json:"es_extraordinaria"`
	Tipo             string  `json:"tipo"` // ORDINARIA, EXTRAORDINARIA, CTS, CESE
	TotalIngresos    float64 `json:"total_ingresos"`
	TotalAportes     float64 `json:"total_aportes"`
	CostoTotal       float64 `json:"costo_total"`
}

// PlanillaDetalle representa la boleta consolidada de un trabajador
type PlanillaDetalle struct {
	ID                             int     `json:"id"`
	PlanillaID                     int     `json:"planilla_id"`
	ContratoID                     int     `json:"contrato_id"`
	TotalIngresos                  float64 `json:"total_ingresos"`
	TotalRetenciones               float64 `json:"total_retenciones"`
	TotalAportes                   float64 `json:"total_aportes"`
	NetoPagar                      float64 `json:"neto_pagar"`
	TrabajadorNombreCompleto       string  `json:"trabajador_nombre_completo"`
	TrabajadorNumeroDocumento      string  `json:"trabajador_numero_documento"`
	PuestoCodigoAirhsp             string  `json:"puesto_codigo_airhsp"`
	PuestoNombre                   string  `json:"puesto_nombre"`
	OrganigramaDocumentoAprobacion string  `json:"organigrama_documento_aprobacion"`
	UnidadOrganicaNombre           string  `json:"unidad_organica_nombre"`
	UnidadOrganicaTipo             string  `json:"unidad_organica_tipo"`
	SueldoBasicoHistorico          float64 `json:"sueldo_basico_historico"`

	// Campos auxiliares para pintar la boleta (JOINs)
	TrabajadorNombre string             `json:"trabajador_nombre,omitempty"`
	TrabajadorDoc    string             `json:"trabajador_doc,omitempty"`
	RegimenDesc      string                  `json:"regimen_desc,omitempty"`
	Conceptos        []PlanillaConcepto      `json:"conceptos,omitempty"`
	Incidencias      []PersonalIncidenciaMes `json:"incidencias,omitempty"`
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
	ID                       int        `json:"id"`
	ConceptoID               int        `json:"concepto_id"`
	NombrePersonalizado      string     `json:"nombre_personalizado"`
	FrecuenciaMeses          string     `json:"frecuencia_meses"`
	ClasificadorID           *int       `json:"clasificador_id"` // Es puntero porque puede ser nulo en la BD
	EsExtraordinario         bool       `json:"es_extraordinario"`
	RequiereMonto            bool       `json:"requiere_monto"`
	EsPensionable            bool       `json:"es_pensionable"`
	EsRemunerativa           bool       `json:"es_remunerativa"`
	EsBaseCts                bool       `json:"es_base_cts"`
	EsBaseBeneficiosSociales bool       `json:"es_base_beneficios_sociales"`
	EsOcasional              bool       `json:"es_ocasional"`
	EsAfectoCargasSociales   bool       `json:"es_afecto_cargas_sociales"`
	ModalidadEntrega         string     `json:"modalidad_entrega"`
	BaseCalculoPara          []string   `json:"base_calculo_para"`
	CreatedAt                string     `json:"created_at"`
	UpdatedAt                *time.Time `json:"updated_at,omitempty"`

	// Campos "virtuales" obtenidos mediante JOINs para la interfaz de usuario
	RegimenesIDs        []int  `json:"regimenes_ids"` // Para los checkboxes (POST/PUT)
	RegimenesNombres    string `json:"regimenes_nombres,omitempty"`
	ConceptoCodigo      string `json:"concepto_codigo,omitempty"`
	ConceptoDescripcion string `json:"concepto_descripcion,omitempty"`
	ConceptoTipo        string `json:"concepto_tipo,omitempty"`
	ClasificadorCodigo  string `json:"clasificador_codigo,omitempty"`
}

// RegimenConceptoModelo representa la relación Muchos a Muchos entre regimenes_laborales y conceptos_modelo
type RegimenConceptoModelo struct {
	RegimenID        int `json:"regimen_id"`
	ConceptoModeloID int `json:"concepto_modelo_id"`
}

// RegimenConceptoTenant representa la relación Muchos a Muchos entre regimenes_laborales y conceptos_tenant
type RegimenConceptoTenant struct {
	TenantID         int `json:"tenant_id"`
	RegimenID        int `json:"regimen_id"`
	ConceptoTenantID int `json:"concepto_tenant_id"`
}

// ConceptoAsignacion representa la estructura temporal para la vista de
// asignación manual de conceptos a un puesto específico.
type ConceptoAsignacion struct {
	ConceptoTenantID int     `json:"concepto_tenant_id"`
	Nombre           string  `json:"nombre"`
	Tipo             string  `json:"tipo"` // INGRESO, RETENCION, APORTE
	RequiereMonto    bool    `json:"requiere_monto"`
	Asignado         bool    `json:"asignado"` // Define si el switch estará encendido
	Monto            float64 `json:"monto"`    // El valor actual (si aplica)
}

// PlanillaCts representa la cabecera semestral de CTS (DL 728)
type PlanillaCts struct {
	ID           int       `json:"id"`
	TenantID     int       `json:"tenant_id"`
	PlanillaID   *int      `json:"planilla_id,omitempty"`
	Anio         int       `json:"anio"`
	Periodo      string    `json:"periodo"` // 'MAYO' o 'NOVIEMBRE'
	Estado       string    `json:"estado"`  // 'BORRADOR', 'PROCESADA'
	FechaCalculo time.Time `json:"fecha_calculo"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// PlanillaCtsDetalle representa el cálculo de CTS individual
type PlanillaCtsDetalle struct {
	ID                     int       `json:"id"`
	PlanillaCtsID          int       `json:"planilla_cts_id"`
	ContratoID             int       `json:"contrato_id"`
	SueldoBasico           float64   `json:"sueldo_basico"`
	AsignacionFamilia      float64   `json:"asignacion_familiar"`
	SextoGratificacion     float64   `json:"sexto_gratificacion"`
	PromedioVariables      float64   `json:"promedio_variables"`
	RemuneracionComputable float64   `json:"remuneracion_computable"`
	MesesComputables       int       `json:"meses_computables"`
	DiasFaltas             int       `json:"dias_faltas"`
	MontoDescuentoFaltas   float64   `json:"monto_descuento_faltas"`
	MontoCts               float64   `json:"monto_cts"`
	CreatedAt              time.Time `json:"created_at"`

	// Campos auxiliares para UI
	TrabajadorNombre    string `json:"trabajador_nombre,omitempty"`
	TrabajadorDocumento string `json:"trabajador_documento,omitempty"`
}

// LiquidacionCese representa el cálculo definitivo de CTS al cese
type LiquidacionCese struct {
	ID                           int       `json:"id"`
	TenantID                     int       `json:"tenant_id"`
	ContratoID                   int       `json:"contrato_id"`
	FechaInicioComputable        time.Time `json:"fecha_inicio_computable"`
	FechaCese                    time.Time `json:"fecha_cese"`
	Motivo                       string    `json:"motivo"`
	AnosServicios                int       `json:"anos_servicios"`
	MesesServicios               int       `json:"meses_servicios"`
	DiasServicios                int       `json:"dias_servicios"`
	RemuneracionComputable       float64   `json:"remuneracion_computable"`
	MontoCts                     float64   `json:"monto_cts"`
	MontoVacacionesTruncas       float64   `json:"monto_vacaciones_truncas"`
	MontoVacacionesNoGozadas     float64   `json:"monto_vacaciones_no_gozadas"`
	MontoIndemnizacionVacacional float64   `json:"monto_indemnizacion_vacacional"`
	PeriodosVencidosVacaciones   int       `json:"periodos_vencidos_vacaciones"`
	PeriodosNoVencidosVacaciones int       `json:"periodos_no_vencidos_vacaciones"`
	MontoGratiTrunca             float64   `json:"monto_gratificacion_trunca"`
	TotalLiquidacion             float64   `json:"total_liquidacion"`
	Estado                       string    `json:"estado"`
	CreatedAt                    time.Time `json:"created_at"`
	UpdatedAt                    time.Time `json:"updated_at"`

	// Campos auxiliares para UI
	TrabajadorNombre    string `json:"trabajador_nombre,omitempty"`
	TrabajadorDocumento string `json:"trabajador_documento,omitempty"`
	PuestoNombre        string `json:"puesto_nombre,omitempty"`
	Regimen             string `json:"regimen,omitempty"`
}

// PlameJornada representa la jornada laboral calculada para exportación al PLAME
type PlameJornada struct {
	TipoDocumento    string
	NumeroDocumento  string
	DiasInasistencia float64
}

// PlameRemuneracion representa el concepto de remuneración calculado para exportación al PLAME
type PlameRemuneracion struct {
	TipoDocumento   string
	NumeroDocumento string
	CodigoConcepto  string
	Monto           float64
	MontoDevengado  float64
	MontoPagado     float64
}

// PlameConceptoDetalle representa un concepto con sus metadatos de régimen y naturaleza remunerativa para el prorrateo vacacional PLAME
type PlameConceptoDetalle struct {
	TrabajadorID    int
	TipoDocumento   string
	NumeroDocumento string
	CodigoConcepto  string
	TipoConcepto    string // 'INGRESO', 'RETENCION', 'APORTE'
	EsRemunerativa  bool
	RegimenCodigo   string
	Monto           float64
}

// TrabajadorEspecialItem representa un trabajador activo y su plaza para la formulación especial
type TrabajadorEspecialItem struct {
	ContratoID               int     `json:"contrato_id"`
	TrabajadorID             int     `json:"trabajador_id"`
	NumeroDocumento          string  `json:"numero_documento"`
	NombreCompleto           string  `json:"nombre_completo"`
	PuestoNombre             string  `json:"puesto_nombre"`
	RegimenID                int     `json:"regimen_id"`
	RegimenNombre            string  `json:"regimen_nombre"`
	MetaID                   int     `json:"meta_id"`
	MetaCodigo               string  `json:"meta_codigo"`
	MetaDescripcion          string  `json:"meta_descripcion"`
	UnidadOrganicaID         int     `json:"unidad_organica_id"`
	UnidadOrganicaNombre     string  `json:"unidad_organica_nombre"`
	TieneRetencionJudicial   bool    `json:"tiene_retencion_judicial"`
	PorcentajeJudicial       float64 `json:"porcentaje_judicial"`
	DetalleRetencionJudicial string  `json:"detalle_retencion_judicial"`
}

// DatosReporteLiquidacion representa la estructura completa para la generación del PDF de Liquidación
type DatosReporteLiquidacion struct {
	TenantNombre           string
	TenantRUC              string
	TenantLogoURL          string
	Liquidacion            LiquidacionCese
	FechaIngreso           time.Time
	SueldoBasico           float64
	AsignacionFamiliar     float64
	PromedioGratificacion  float64
	RemuneracionComputable float64
	CtsPeriodoInicio       string
	CtsPeriodoFin          string
	CtsMeses               int
	CtsDias                int
	MontoCtsMeses          float64
	MontoCtsDias           float64
	VacacionesMeses        int
	VacacionesDias         int
	MontoVacacionesMeses   float64
	MontoVacacionesDias    float64
	VacacionesBrutas       float64
	DescuentoPensionNombre string
	MontoDescuentoPension  float64
	VacacionesNetas        float64
	GratiSemestreTipo      string
	GratiPeriodoInicio     string
	GratiPeriodoFin        string
	GratiMeses             int
	GratiDias              int
	MontoGratiMeses        float64
	MontoGratiDias         float64
	BonificacionEspecial   float64
	TotalLiquidacion       float64
	MontoEnLetras          string
	FechaEmisionTexto      string
}
