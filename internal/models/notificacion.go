package models

import "time"

// Notificacion representa una alerta o aviso para el panel de usuarios
type Notificacion struct {
	ID        int       `json:"id"`
	TenantID  *int      `json:"tenant_id"`  // NULL indica que es una notificación global/SaaS Admin
	UsuarioID *int      `json:"usuario_id"` // NULL indica que es para todos los usuarios de la entidad
	Titulo    string    `json:"titulo"`
	Mensaje   string    `json:"mensaje"`
	Tipo      string    `json:"tipo"` // 'ALERTA_SISTEMA', 'PROCESO_EXITOSO', 'ERROR'
	Leido     bool      `json:"leido"`
	CreatedAt time.Time `json:"created_at"`
}
