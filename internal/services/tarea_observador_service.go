package services

import (
	"log"
	"os"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
	"time"
)

// TareaObservadorService implementa el background daemon en Go
type TareaObservadorService struct {
	TareaRepo *repository.AdminTareaRepository
	NotifRepo *repository.NotificacionRepository
	Mailer    Mailer
}

// NewTareaObservadorService crea un nuevo servicio de TareaObservadorService
func NewTareaObservadorService(
	tareaRepo *repository.AdminTareaRepository,
	notifRepo *repository.NotificacionRepository,
	mailer Mailer,
) *TareaObservadorService {
	return &TareaObservadorService{
		TareaRepo: tareaRepo,
		NotifRepo: notifRepo,
		Mailer:    mailer,
	}
}

// Iniciar arranca el observador en segundo plano con una frecuencia determinada
func (s *TareaObservadorService) Iniciar(intervalo time.Duration) {
	ticker := time.NewTicker(intervalo)
	log.Printf("⏰ Daemon observador de tareas iniciado (frecuencia: %v)", intervalo)

	go func() {
		// Procesamiento inicial al arrancar el servidor
		if err := s.ProcesarTareasVencidas(time.Now()); err != nil {
			log.Printf("⚠️ Error inicial en Daemon de tareas: %v", err)
		}

		for range ticker.C {
			if err := s.ProcesarTareasVencidas(time.Now()); err != nil {
				log.Printf("⚠️ Error en Daemon observador de tareas: %v", err)
			}
		}
	}()
}

// ProcesarTareasVencidas consulta la base de datos y despacha avisos de tareas expiradas
func (s *TareaObservadorService) ProcesarTareasVencidas(ahora time.Time) error {
	tareas, err := s.TareaRepo.ObtenerTareasVencidas(ahora)
	if err != nil {
		return err
	}

	for _, t := range tareas {
		log.Printf("⏰ Daemon: Procesando tarea programada vencida [ID: %d] - '%s'", t.ID, t.Titulo)

		// 1. Crear notificación en BD para el panel del Super Admin (tenant_id = nil, usuario_id = nil)
		notif := models.Notificacion{
			TenantID:  nil,
			UsuarioID: nil,
			Titulo:    "⏰ RECORDATORIO: " + t.Titulo,
			Mensaje:   t.Descripcion,
			Tipo:      "ALERTA_SISTEMA",
			Leido:     false,
		}
		if err := s.NotifRepo.Crear(&notif); err != nil {
			log.Printf("Error al registrar alerta de notificación de tarea %d: %v", t.ID, err)
		}

		// 2. Enviar correo usando el Mailer inyectado (SMTP o Mock)
		emailTo := os.Getenv("SMTP_TO")
		if emailTo == "" {
			emailTo = "admin@rgm.com" // Fallback seguro
		}
		asunto := "⏰ RECORDATORIO RGM: " + t.Titulo
		cuerpo := t.Descripcion + "\n\nFecha de Vencimiento: " + t.FechaVencimiento.Format("02/01/2006 15:04")

		if err := s.Mailer.Enviar(emailTo, asunto, cuerpo); err != nil {
			log.Printf("Error al despachar correo electrónico para tarea %d: %v", t.ID, err)
		}

		// 3. Calcular la recurrencia y avanzar el reloj
		var nuevoAviso time.Time
		desactivar := false

		switch t.Recurrencia {
		case "MENSUAL":
			nuevoAviso = t.ProximoAviso.AddDate(0, 1, 0)
		case "TRIMESTRAL":
			nuevoAviso = t.ProximoAviso.AddDate(0, 3, 0)
		default: // 'UNICO' u otros
			nuevoAviso = t.ProximoAviso
			desactivar = true
		}

		if err := s.TareaRepo.ActualizarProximoAviso(t.ID, nuevoAviso, desactivar); err != nil {
			log.Printf("Error al actualizar la programación del próximo aviso de la tarea %d: %v", t.ID, err)
		}
	}

	return nil
}
