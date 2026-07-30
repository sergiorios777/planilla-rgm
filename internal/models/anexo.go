package models

// AjusteRedondeoSunat almacena los valores exactos, redondeados y la diferencia por concepto tributario SUNAT
type AjusteRedondeoSunat struct {
	ConceptoClave     string  `json:"concepto_clave"`     // "ONP", "RENTA_4TA", "RENTA_5TA"
	NombreConcepto    string  `json:"nombre_concepto"`    // Ej: "SNP DL 19990 - ONP"
	MontoExacto       float64 `json:"monto_exacto"`       // Suma exacta de centavos en planilla_conceptos
	MontoRedondeado   float64 `json:"monto_redondeado"`   // Math.Round(MontoExacto)
	Diferencia        float64 `json:"diferencia"`         // MontoRedondeado - MontoExacto
	MetaCodigoTarget  string  `json:"meta_codigo_target,omitempty"`
	ClasificadorTarget string `json:"clasificador_target,omitempty"`
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
