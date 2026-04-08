package models

import "time"

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

// Trabajador representa a un empleado de un inquilino específico.
// Aquí aplicamos estrictamente el tenant_id.
type Trabajador struct {
	ID             int       `json:"id"`
	TenantID       int       `json:"tenant_id"` // A qué entidad pertenece
	Dni            string    `json:"dni"`
	Nombres        string    `json:"nombres"`
	Apellidos      string    `json:"apellidos"`
	RegimenLaboral string    `json:"regimen_laboral"` // Ej: "CAS", "276", "728"
	FechaIngreso   time.Time `json:"fecha_ingreso"`
	Activo         bool      `json:"activo"`
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

// ParametroGlobal define valores anuales que afectan a todas las planillas (UIT, RMV, etc.)
type ParametroGlobal struct {
	ID          int     `json:"id"`
	Clave       string  `json:"clave"`
	Valor       float64 `json:"valor"`
	FechaDesde  string  `json:"fecha_desde"` // Formato YYYY-MM-DD
	FechaHasta  *string `json:"fecha_hasta"` // Puntero para permitir nulos (vigente actualmente)
	Descripcion string  `json:"descripcion"`
}
