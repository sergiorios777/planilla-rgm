package services

import (
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
)

type PuestoService struct {
	Repo *repository.PuestoRepository
}

func (s *PuestoService) CrearPuestoConPlantilla(nuevoPuesto *models.Puesto) error {
	// 1. Guardamos el Puesto en la BD
	return s.Repo.Crear(nuevoPuesto)
}
