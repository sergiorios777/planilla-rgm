package services

import (
	"planilla-rgm/internal/config"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
	"testing"
	"time"
)

// SpyMailer es un espía para verificar que los correos electrónicos se envíen
type SpyMailer struct {
	Enviados []string
}

func (s *SpyMailer) Enviar(para string, asunto string, cuerpo string) error {
	s.Enviados = append(s.Enviados, para+":"+asunto)
	return nil
}

func TestProcesarTareasVencidas(t *testing.T) {
	// 1. Inicializar la base de datos de prueba cargando el .env
	db, err := config.InitDB()
	if err != nil {
		t.Skip("Saltando prueba de integración: Base de datos no disponible o .env faltante")
		return
	}
	defer db.Close()

	// Limpiar tablas antes de la prueba
	_, _ = db.Exec("DELETE FROM notificaciones")
	_, _ = db.Exec("DELETE FROM admin_tareas")

	// 2. Instanciar repositorios y servicio con el SpyMailer
	tareaRepo := repository.NewAdminTareaRepository(db)
	notifRepo := repository.NewNotificacionRepository(db)
	spyMailer := &SpyMailer{}
	service := NewTareaObservadorService(tareaRepo, notifRepo, spyMailer)

	// 3. Crear fixtures
	ahora := time.Now().Truncate(time.Second)

	// Tarea Única (Vencida)
	tUnica := models.AdminTarea{
		Titulo:           "Tarea Única Expirada",
		Descripcion:      "Esta es una tarea única que ya venció",
		Recurrencia:      "UNICO",
		FechaVencimiento: ahora.Add(-1 * time.Hour),
		ProximoAviso:     ahora.Add(-30 * time.Minute),
		Activo:           true,
	}

	// Tarea Mensual (Vencida)
	tMensual := models.AdminTarea{
		Titulo:           "Tarea Mensual Expirada",
		Descripcion:      "Esta es una tarea mensual que ya venció",
		Recurrencia:      "MENSUAL",
		FechaVencimiento: ahora.Add(-5 * time.Hour),
		ProximoAviso:     ahora.Add(-4 * time.Hour),
		Activo:           true,
	}

	// Tarea Futura (Vigente, no debe procesarse)
	tFutura := models.AdminTarea{
		Titulo:           "Tarea Futura Activa",
		Descripcion:      "Esta tarea vencerá mañana",
		Recurrencia:      "UNICO",
		FechaVencimiento: ahora.Add(24 * time.Hour),
		ProximoAviso:     ahora.Add(23 * time.Hour),
		Activo:           true,
	}

	// Insertar fixtures en base de datos
	if err := tareaRepo.Crear(&tUnica); err != nil {
		t.Fatal("Error creando tarea única:", err)
	}
	if err := tareaRepo.Crear(&tMensual); err != nil {
		t.Fatal("Error creando tarea mensual:", err)
	}
	if err := tareaRepo.Crear(&tFutura); err != nil {
		t.Fatal("Error creando tarea futura:", err)
	}

	// 4. Ejecutar el observador del Daemon pasándole la fecha simulada
	err = service.ProcesarTareasVencidas(ahora)
	if err != nil {
		t.Fatal("Error ejecutando ProcesarTareasVencidas:", err)
	}

	// 5. Aseveraciones (Assertions)

	// A. La tarea única vencida debe haberse desactivado
	tUnicaVerif, err := tareaRepo.ObtenerPorID(tUnica.ID)
	if err != nil {
		t.Fatal(err)
	}
	if tUnicaVerif.Activo {
		t.Error("La tarea única expirada debió ser desactivada (activo = false)")
	}

	// B. La tarea mensual vencida debe seguir activa y reprogramar su fecha de aviso a un mes después
	tMensualVerif, err := tareaRepo.ObtenerPorID(tMensual.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !tMensualVerif.Activo {
		t.Error("La tarea mensual debió seguir activa (activo = true)")
	}
	expectedAviso := tMensual.ProximoAviso.AddDate(0, 1, 0)
	if tMensualVerif.ProximoAviso.Format("2006-01-02 15:04") != expectedAviso.Format("2006-01-02 15:04") {
		t.Errorf("Próximo aviso incorrecto. Esperado: %v, Obtenido: %v", expectedAviso, tMensualVerif.ProximoAviso)
	}

	// C. La tarea futura no debió ser procesada ni modificada
	tFuturaVerif, err := tareaRepo.ObtenerPorID(tFutura.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !tFuturaVerif.ProximoAviso.Equal(tFutura.ProximoAviso) {
		t.Error("La tarea futura no debió ser modificada")
	}

	// D. Deben haberse creado exactamente 2 notificaciones en la tabla
	conteoNotif, err := notifRepo.ContarNoLeidas(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if conteoNotif != 2 {
		t.Errorf("Se esperaban 2 notificaciones en base de datos, se obtuvieron: %d", conteoNotif)
	}

	// E. El SpyMailer debe registrar exactamente 2 correos despachados
	if len(spyMailer.Enviados) != 2 {
		t.Errorf("Se esperaban 2 correos electrónicos enviados, se registraron: %d", len(spyMailer.Enviados))
	}
}
