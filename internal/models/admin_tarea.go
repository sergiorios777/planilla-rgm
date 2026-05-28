package models

import "time"

// AdminTarea representa una tarea programada o recordatorio administrado por el Super Admin
type AdminTarea struct {
	ID               int       `json:"id"`
	Titulo           string    `json:"titulo"`
	Descripcion      string    `json:"descripcion"`
	Recurrencia      string    `json:"recurrencia"` // 'UNICO', 'MENSUAL', 'TRIMESTRAL'
	FechaVencimiento time.Time `json:"fecha_vencimiento"`
	ProximoAviso     time.Time `json:"proximo_aviso"`
	NotificadoEmail  bool      `json:"notificado_email"`
	Activo           bool      `json:"activo"`
	CreatedAt        time.Time `json:"created_at"`
}

// Métodos auxiliares para formatear las fechas en los inputs HTML de tipo datetime-local
func (t AdminTarea) FechaVencimientoFormatoInput() string {
	return t.FechaVencimiento.Format("2006-01-02T15:04")
}

func (t AdminTarea) ProximoAvisoFormatoInput() string {
	return t.ProximoAviso.Format("2006-01-02T15:04")
}

func (t AdminTarea) FechaVencimientoLegible() string {
	return t.FechaVencimiento.Format("02/01/2006 15:04")
}

func (t AdminTarea) ProximoAvisoLegible() string {
	return t.ProximoAviso.Format("02/01/2006 15:04")
}
