package services

import (
	"log"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
)

type PuestoService struct {
	Repo *repository.PuestoRepository
}

func (s *PuestoService) CrearPuestoConPlantilla(nuevoPuesto *models.Puesto) error {
	// 1. Guardamos el Puesto en la BD
	err := s.Repo.Crear(nuevoPuesto)
	if err != nil {
		return err
	}

	// 2. MAGIA SAAS: Consultamos directamente a la BD la plantilla según el régimen
	// (Esta consulta ya excluye automáticamente ONP/AFP según nuestra regla de negocio)
	idsLocales, err := s.Repo.ObtenerConceptosModeloPorRegimen(nuevoPuesto.TenantID, nuevoPuesto.RegimenID)
	if err != nil {
		log.Println("Error al obtener plantilla de conceptos desde el modelo:", err)
		return nil // No detenemos la creación del puesto si falla la asignación
	}

	// 3. Insertamos los conceptos
	return s.Repo.AsignarConceptosAPuesto(nuevoPuesto.ID, idsLocales, nuevoPuesto.SueldoPresupuestado)
}
