package models

import "time"

// SunatTipoSuspension representa un registro oficial de la Tabla 21 de SUNAT
type SunatTipoSuspension struct {
	ID                   int       `json:"id"`
	Codigo               string    `json:"codigo"`
	Descripcion          string    `json:"descripcion"`
	DescripcionAbreviada string    `json:"descripcion_abreviada"`
	TipoSuspension       string    `json:"tipo_suspension"` // PERFECTA o IMPERFECTA
	Activo               bool      `json:"activo"`
	CreatedAt            time.Time `json:"created_at"`
}

// LicenciaVacacion representa el registro individual de vacaciones o licencias de un trabajador
type LicenciaVacacion struct {
	ID                     int        `json:"id"`
	TenantID               int        `json:"tenant_id"`
	TrabajadorID           int        `json:"trabajador_id"`
	ContratoID             *int       `json:"contrato_id,omitempty"`
	Tipo                   string     `json:"tipo"` // VACACION, LICENCIA_CON_GOCE, LICENCIA_SIN_GOCE
	Subtipo                string     `json:"subtipo,omitempty"`
	CodigoSunatSuspension  string     `json:"codigo_sunat_suspension"`
	FechaInicio            string     `json:"fecha_inicio"`
	FechaFin               string     `json:"fecha_fin"`
	DiasCalendario         int        `json:"dias_calendario"`
	DocumentoAprobacion    string     `json:"documento_aprobacion"`
	FechaAprobacion        *string    `json:"fecha_aprobacion,omitempty"`
	Observaciones          string     `json:"observaciones,omitempty"`
	Estado                 string     `json:"estado"` // PROGRAMADO, APROBADO, EJECUTADO, CANCELADO
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

// LicenciaVacacionVista es un DTO enriquecido para la grilla HTMX
type LicenciaVacacionVista struct {
	ID                       int     `json:"id"`
	TenantID                 int     `json:"tenant_id"`
	TrabajadorID             int     `json:"trabajador_id"`
	ContratoID               *int    `json:"contrato_id,omitempty"`
	TrabajadorNombre         string  `json:"trabajador_nombre"`
	TrabajadorDoc            string  `json:"trabajador_doc"`
	PuestoNombre             string  `json:"puesto_nombre"`
	RegimenCodigo            string  `json:"regimen_codigo"`
	Tipo                     string  `json:"tipo"`
	Subtipo                  string  `json:"subtipo"`
	CodigoSunatSuspension    string  `json:"codigo_sunat_suspension"`
	SunatDescripcionAbrev    string  `json:"sunat_descripcion_abrev"`
	SunatTipoSuspension      string  `json:"sunat_tipo_suspension"` // PERFECTA o IMPERFECTA
	FechaInicio              string  `json:"fecha_inicio"`
	FechaFin                 string  `json:"fecha_fin"`
	DiasCalendario           int     `json:"dias_calendario"`
	DocumentoAprobacion      string  `json:"documento_aprobacion"`
	FechaAprobacion          *string `json:"fecha_aprobacion,omitempty"`
	Observaciones            string  `json:"observaciones"`
	Estado                   string  `json:"estado"`
}

// PersonalIncidenciaMes representa una vacación o licencia que solapa con el periodo de una planilla mensual
type PersonalIncidenciaMes struct {
	IncidenciaID          int    `json:"incidencia_id"`
	TrabajadorID          int    `json:"trabajador_id"`
	ContratoID            int    `json:"contrato_id"`
	TrabajadorNombre      string `json:"trabajador_nombre"`
	TrabajadorDoc         string `json:"trabajador_doc"`
	PuestoNombre          string `json:"puesto_nombre"`
	RegimenCodigo         string `json:"regimen_codigo"`
	Tipo                  string `json:"tipo"` // VACACION, LICENCIA_CON_GOCE, LICENCIA_SIN_GOCE
	Subtipo               string `json:"subtipo"`
	CodigoSunatSuspension string `json:"codigo_sunat_suspension"` // Tabla 21 SUNAT
	FechaInicio           string `json:"fecha_inicio"`
	FechaFin              string `json:"fecha_fin"`
	DiasEnMes             int    `json:"dias_en_mes"` // Días que caen dentro del mes de cálculo
	DocumentoAprobacion   string `json:"documento_aprobacion"`
	Observaciones         string `json:"observaciones"`
}

// KpisLicenciaVacacion agrupa las métricas para el Bento Grid
type KpisLicenciaVacacion struct {
	TotalEnVacacionesMes      int `json:"total_en_vacaciones_mes"`
	TotalLicenciasConGoceMes  int `json:"total_licencias_con_goce_mes"`
	TotalLicenciasSinGoceMes  int `json:"total_licencias_sin_goce_mes"`
	TotalHistorico            int `json:"total_historico"`
}
