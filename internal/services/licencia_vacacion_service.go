package services

import (
	"errors"
	"fmt"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
	"strings"
	"time"
)

type LicenciaVacacionService struct {
	Repo *repository.LicenciaVacacionRepository
}

func NewLicenciaVacacionService(repo *repository.LicenciaVacacionRepository) *LicenciaVacacionService {
	return &LicenciaVacacionService{Repo: repo}
}

// ValidarDatos valida la integridad de los campos antes de persistir
func (s *LicenciaVacacionService) ValidarDatos(item *models.LicenciaVacacion) error {
	if item.TrabajadorID <= 0 && (item.ContratoID == nil || *item.ContratoID <= 0) {
		return errors.New("debe seleccionar un trabajador o contrato válido")
	}

	item.DocumentoAprobacion = strings.TrimSpace(item.DocumentoAprobacion)
	if item.DocumentoAprobacion == "" {
		return errors.New("el documento de aprobación es obligatorio (ej. Resolución de Alcaldía, Memorándum)")
	}

	tipoUpper := strings.ToUpper(strings.TrimSpace(item.Tipo))
	if tipoUpper != "VACACION" && tipoUpper != "LICENCIA_CON_GOCE" && tipoUpper != "LICENCIA_SIN_GOCE" {
		return errors.New("el tipo de incidencia debe ser VACACION, LICENCIA_CON_GOCE o LICENCIA_SIN_GOCE")
	}
	item.Tipo = tipoUpper

	// Parseo y validación de fechas
	fInicio, err := time.Parse("2006-01-02", item.FechaInicio)
	if err != nil {
		return fmt.Errorf("formato inválido para fecha de inicio (debe ser YYYY-MM-DD): %v", err)
	}

	fFin, err := time.Parse("2006-01-02", item.FechaFin)
	if err != nil {
		return fmt.Errorf("formato inválido para fecha de fin (debe ser YYYY-MM-DD): %v", err)
	}

	if fFin.Before(fInicio) {
		return errors.New("la fecha de fin no puede ser anterior a la fecha de inicio")
	}

	// Asignación de código SUNAT por defecto si no se especificó
	if strings.TrimSpace(item.CodigoSunatSuspension) == "" {
		switch item.Tipo {
		case "VACACION":
			item.CodigoSunatSuspension = "23" // S.I. DESCANSO VACACIONAL
		case "LICENCIA_CON_GOCE":
			item.CodigoSunatSuspension = "26" // S.I. LICENCIA CON GOCE DE HABER
		case "LICENCIA_SIN_GOCE":
			item.CodigoSunatSuspension = "05" // S.P. PERMISO, LICENCIA U OTROS SIN GOCE
		}
	}

	if strings.TrimSpace(item.Estado) == "" {
		item.Estado = "APROBADO"
	}

	return nil
}

// Crear valida e inserta un nuevo registro de vacación o licencia
func (s *LicenciaVacacionService) Crear(tenantID int, item *models.LicenciaVacacion) error {
	item.TenantID = tenantID

	if err := s.ValidarDatos(item); err != nil {
		return err
	}

	// Si se envió contrato_id pero trabajador_id es 0, resolverlo
	if item.TrabajadorID <= 0 && item.ContratoID != nil && *item.ContratoID > 0 {
		tID, err := s.Repo.ObtenerTrabajadorYContratoID(tenantID, *item.ContratoID)
		if err != nil {
			return fmt.Errorf("no se pudo identificar al trabajador del contrato: %w", err)
		}
		item.TrabajadorID = tID
	}

	// Validar solapamiento de fechas para el mismo trabajador
	solapa, err := s.Repo.ValidarSolapamiento(tenantID, item.TrabajadorID, item.FechaInicio, item.FechaFin, 0)
	if err != nil {
		return fmt.Errorf("error validando solapamiento: %w", err)
	}
	if solapa {
		return errors.New("el trabajador ya cuenta con un registro activo de vacaciones o licencia que solapa con el rango de fechas seleccionado")
	}

	return s.Repo.Crear(item)
}

// Actualizar valida y actualiza un registro existente
func (s *LicenciaVacacionService) Actualizar(tenantID int, item *models.LicenciaVacacion) error {
	item.TenantID = tenantID

	if item.ID <= 0 {
		return errors.New("identificador de registro inválido")
	}

	if err := s.ValidarDatos(item); err != nil {
		return err
	}

	// Validar solapamiento excluyendo el propio ID
	solapa, err := s.Repo.ValidarSolapamiento(tenantID, item.TrabajadorID, item.FechaInicio, item.FechaFin, item.ID)
	if err != nil {
		return fmt.Errorf("error validando solapamiento: %w", err)
	}
	if solapa {
		return errors.New("el nuevo rango de fechas solapa con otro registro activo del trabajador")
	}

	return s.Repo.Actualizar(item)
}

// Eliminar borra un registro
func (s *LicenciaVacacionService) Eliminar(id int, tenantID int) error {
	if id <= 0 {
		return errors.New("ID inválido")
	}
	return s.Repo.Eliminar(id, tenantID)
}

// Listar obtiene los registros filtrados
func (s *LicenciaVacacionService) Listar(tenantID int, buscar string, tipo string, estado string, anio int, mes int) ([]models.LicenciaVacacionVista, error) {
	return s.Repo.Listar(tenantID, buscar, tipo, estado, anio, mes)
}

// ObtenerKPIs calcula las métricas del Bento Grid
func (s *LicenciaVacacionService) ObtenerKPIs(tenantID int, anio int, mes int) (*models.KpisLicenciaVacacion, error) {
	return s.Repo.ObtenerKPIs(tenantID, anio, mes)
}
