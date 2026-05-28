package models

// AFP representa a la administradora de fondo de pensiones (Habitat, Integra, etc.)
type AFP struct {
	ID        int    `json:"id"`
	Nombre    string `json:"nombre"`
	CodigoSBS string `json:"codigo_sbs"`
	Activo    bool   `json:"activo"`
}

// AFPTasaMensual representa la matriz de comisiones y primas vigentes para un mes determinado
type AFPTasaMensual struct {
	ID                 int     `json:"id"`
	AfpID              int     `json:"afp_id"`
	Anio               int     `json:"anio"`
	Mes                int     `json:"mes"`
	AporteObligatorio   float64 `json:"aporte_obligatorio"`
	ComisionFlujo       float64 `json:"comision_flujo"`
	ComisionMixtaFlujo float64 `json:"comision_mixta_flujo"`
	PrimaSeguro         float64 `json:"prima_seguro"`
	ComisionAnualSaldo float64 `json:"comision_anual_saldo"`
}

// AFPTasaVista es un DTO útil para cruzar las tasas con las AFPs en las vistas del panel admin
type AFPTasaVista struct {
	AfpID              int     `json:"afp_id"`
	AfpNombre          string  `json:"afp_nombre"`
	AfpCodigoSBS       string  `json:"afp_codigo_sbs"`
	TasaID             *int    `json:"tasa_id"` // Nulo si no hay tasas registradas para ese mes
	Anio               int     `json:"anio"`
	Mes                int     `json:"mes"`
	AporteObligatorio   float64 `json:"aporte_obligatorio"`
	ComisionFlujo       float64 `json:"comision_flujo"`
	ComisionMixtaFlujo float64 `json:"comision_mixta_flujo"`
	PrimaSeguro         float64 `json:"prima_seguro"`
	ComisionAnualSaldo float64 `json:"comision_anual_saldo"`
	Registrado         bool    `json:"registrado"` // True si hay registro en la BD
}

// Métodos auxiliares para formatear los valores como porcentajes en las plantillas HTML
func (v AFPTasaVista) AportePct() float64      { return v.AporteObligatorio * 100.0 }
func (v AFPTasaVista) FlujoPct() float64       { return v.ComisionFlujo * 100.0 }
func (v AFPTasaVista) MixtaPct() float64       { return v.ComisionMixtaFlujo * 100.0 }
func (v AFPTasaVista) PrimaPct() float64       { return v.PrimaSeguro * 100.0 }
func (v AFPTasaVista) SaldoPct() float64       { return v.ComisionAnualSaldo * 100.0 }
