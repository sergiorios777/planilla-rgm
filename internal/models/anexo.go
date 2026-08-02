package models

// AjusteRedondeoSunat almacena los valores exactos, redondeados y la diferencia por concepto tributario SUNAT
type AjusteRedondeoSunat struct {
	ConceptoClave            string  `json:"concepto_clave"`              // "ONP", "RENTA_4TA", "RENTA_5TA"
	NombreConcepto           string  `json:"nombre_concepto"`             // Ej: "SNP DL 19990 - ONP"
	MontoExacto              float64 `json:"monto_exacto"`                // Suma exacta de centavos en planilla_conceptos
	MontoRedondeado          float64 `json:"monto_redondeado"`            // Math.Round(MontoExacto)
	Diferencia               float64 `json:"diferencia"`                  // MontoRedondeado - MontoExacto
	MetaCodigoTarget         string  `json:"meta_codigo_target,omitempty"`
	ClasificadorTarget       string  `json:"clasificador_target,omitempty"`
	CodigoSunatIngresoTarget string  `json:"codigo_sunat_ingreso_target,omitempty"`
	NombreIngresoTarget      string  `json:"nombre_ingreso_target,omitempty"`
}

// ItemCompromisoPresupuestal representa una fila agregada por Meta Presupuestal y Clasificador de Gasto MEF (Anexo 1)
type ItemCompromisoPresupuestal struct {
	MetaCodigo              string  `json:"meta_codigo"`
	MetaDescripcion         string  `json:"meta_descripcion"`
	ClasificadorCodigo      string  `json:"clasificador_codigo"`
	ClasificadorDescripcion string  `json:"clasificador_descripcion"`
	MontoTotal              float64 `json:"monto_total"`
}

// ResumenMetaCompromiso agrupa los ítems y el subtotal acumulado por Meta Presupuestal
type ResumenMetaCompromiso struct {
	MetaCodigo      string                       `json:"meta_codigo"`
	MetaDescripcion string                       `json:"meta_descripcion"`
	Items           []ItemCompromisoPresupuestal `json:"items"`
	TotalMeta       float64                      `json:"total_meta"`
}

// DatosAnexo1 representa la estructura consolidada para la vista web, PDF y Excel del Anexo 1
type DatosAnexo1 struct {
	TenantNombre     string                       `json:"tenant_nombre"`
	TenantRUC        string                       `json:"tenant_ruc"`
	PlanillaID       int                          `json:"planilla_id"`
	PlanillaDesc     string                       `json:"planilla_desc"`
	PlanillaAnio     int                          `json:"planilla_anio"`
	PlanillaMes      int                          `json:"planilla_mes"`
	PlanillaEstado   string                       `json:"planilla_estado"`
	Items            []ItemCompromisoPresupuestal `json:"items"`
	ResumenMetas     []ResumenMetaCompromiso      `json:"resumen_metas"`
	AjustesAplicados []AjusteRedondeoSunat        `json:"ajustes_aplicados"`
	MontoTotal       float64                      `json:"monto_total"`
}

// ItemResumenConcepto representa la sumatoria de un concepto en la planilla (Anexo 1A)
type ItemResumenConcepto struct {
	TipoConcepto   string  `json:"tipo_concepto"` // "INGRESO", "RETENCION", "APORTE"
	CodigoSunat    string  `json:"codigo_sunat"`
	NombreConcepto string  `json:"nombre_concepto"`
	MontoTotal     float64 `json:"monto_total"`
}

// GrupoResumenConcepto agrupa los conceptos por su tipo (INGRESO, RETENCION, APORTE)
type GrupoResumenConcepto struct {
	TipoConcepto string                `json:"tipo_concepto"` // "INGRESO", "RETENCION", "APORTE"
	Titulo       string                `json:"titulo"`        // "1. INGRESOS", "2. RETENCIONES / DESCUENTOS", "3. APORTES DEL EMPLEADOR"
	Items        []ItemResumenConcepto `json:"items"`
	TotalGrupo   float64               `json:"total_grupo"`
}

// DatosAnexo1A representa la estructura completa para la vista, PDF y Excel del Anexo 1A
type DatosAnexo1A struct {
	TenantNombre     string                 `json:"tenant_nombre"`
	TenantRUC        string                 `json:"tenant_ruc"`
	PlanillaID       int                    `json:"planilla_id"`
	PlanillaDesc     string                 `json:"planilla_desc"`
	PlanillaAnio     int                    `json:"planilla_anio"`
	PlanillaMes      int                    `json:"planilla_mes"`
	PlanillaEstado   string                 `json:"planilla_estado"`
	Grupos           []GrupoResumenConcepto `json:"grupos"`
	TotalIngresos    float64                `json:"total_ingresos"`
	TotalRetenciones float64                `json:"total_retenciones"`
	TotalAportes     float64                `json:"total_aportes"`
	CostoTotal       float64                `json:"costo_total"` // TotalIngresos + TotalAportes
}

// ItemResumenAFP representa una fila agregada por AFP en el Anexo 2
type ItemResumenAFP struct {
	AFPNombre         string  `json:"afp_nombre"`
	AporteObligatorio float64 `json:"aporte_obligatorio"`
	Comision          float64 `json:"comision"`
	PrimaSeguro       float64 `json:"prima_seguro"`
	TotalAFP          float64 `json:"total_afp"`
}

// DatosAnexo2 representa la estructura consolidada para la vista, PDF y Excel del Anexo 2
type DatosAnexo2 struct {
	TenantNombre           string           `json:"tenant_nombre"`
	TenantRUC              string           `json:"tenant_ruc"`
	PlanillaID             int              `json:"planilla_id"`
	PlanillaDesc           string           `json:"planilla_desc"`
	PlanillaAnio           int              `json:"planilla_anio"`
	PlanillaMes            int              `json:"planilla_mes"`
	PlanillaEstado         string           `json:"planilla_estado"`
	Items                  []ItemResumenAFP `json:"items"`
	TotalAporteObligatorio float64          `json:"total_aporte_obligatorio"`
	TotalComision          float64          `json:"total_comision"`
	TotalPrimaSeguro       float64          `json:"total_prima_seguro"`
	GranTotal              float64          `json:"gran_total"`
}

// ItemDevengadoAFP representa una fila por Meta y Clasificador en el Anexo 2A
type ItemDevengadoAFP struct {
	AFPNombre               string  `json:"afp_nombre"`
	MetaCodigo              string  `json:"meta_codigo"`
	ClasificadorCodigo      string  `json:"clasificador_codigo"`
	ClasificadorDescripcion string  `json:"clasificador_descripcion"`
	AporteObligatorio       float64 `json:"aporte_obligatorio"`
	Comision                float64 `json:"comision"`
	PrimaSeguro             float64 `json:"prima_seguro"`
	TotalFila               float64 `json:"total_fila"`
}

// GrupoDevengadoAFP agrupa las filas de devengado por AFP para la presentación institucional
type GrupoDevengadoAFP struct {
	AFPNombre              string             `json:"afp_nombre"`
	Items                  []ItemDevengadoAFP `json:"items"`
	TotalAporteObligatorio float64            `json:"total_aporte_obligatorio"`
	TotalComision          float64            `json:"total_comision"`
	TotalPrimaSeguro       float64            `json:"total_prima_seguro"`
	TotalGrupo             float64            `json:"total_grupo"`
}

// DatosAnexo2A representa la estructura consolidada para el Anexo 2A (Registro Devengado por Meta/Clasificador)
type DatosAnexo2A struct {
	TenantNombre           string              `json:"tenant_nombre"`
	TenantRUC              string              `json:"tenant_ruc"`
	PlanillaID             int                 `json:"planilla_id"`
	PlanillaDesc           string              `json:"planilla_desc"`
	PlanillaAnio           int                 `json:"planilla_anio"`
	PlanillaMes            int                 `json:"planilla_mes"`
	PlanillaEstado         string              `json:"planilla_estado"`
	Grupos                 []GrupoDevengadoAFP `json:"grupos"`
	TotalAporteObligatorio float64             `json:"total_aporte_obligatorio"`
	TotalComision          float64             `json:"total_comision"`
	TotalPrimaSeguro       float64             `json:"total_prima_seguro"`
	GranTotal              float64             `json:"gran_total"`
}

// ItemRetencionesSunat representa una fila agregada por Meta y Clasificador en el Anexo 3
type ItemRetencionesSunat struct {
	MetaCodigo              string  `json:"meta_codigo"`
	ClasificadorCodigo      string  `json:"clasificador_codigo"`
	ClasificadorDescripcion string  `json:"clasificador_descripcion"`
	ONP                     float64 `json:"onp"`
	Renta4ta                float64 `json:"renta_4ta"`
	Renta5ta                float64 `json:"renta_5ta"`
	TotalFila               float64 `json:"total_fila"`
}

// DatosAnexo3 representa la estructura consolidada para la vista, PDF y Excel del Anexo 3 (Retenciones de SUNAT)
type DatosAnexo3 struct {
	TenantNombre     string                 `json:"tenant_nombre"`
	TenantRUC        string                 `json:"tenant_ruc"`
	PlanillaID       int                    `json:"planilla_id"`
	PlanillaDesc     string                 `json:"planilla_desc"`
	PlanillaAnio     int                    `json:"planilla_anio"`
	PlanillaMes      int                    `json:"planilla_mes"`
	PlanillaEstado   string                 `json:"planilla_estado"`
	Items            []ItemRetencionesSunat `json:"items"`
	TotalONP         float64                `json:"total_onp"`
	TotalRenta4ta    float64                `json:"total_renta_4ta"`
	TotalRenta5ta    float64                `json:"total_renta_5ta"`
	GranTotal        float64                `json:"gran_total"`
}
