package services

import (
	"log"
	"planilla-rgm/internal/config"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
)

type PuestoService struct {
	Repo *repository.PuestoRepository
}

// puesto_service.go
func (s *PuestoService) CrearPuestoConPlantilla(nuevoPuesto *models.Puesto) error {
	// 1. Guardamos el Puesto en la BD (nos devuelve el ID generado)
	err := s.Repo.Crear(nuevoPuesto)
	if err != nil {
		return err
	}

	// 2. Buscamos el código del régimen (Ej. '1057') usando el ID generado
	puestoGuardado, err := s.Repo.ObtenerPorID(nuevoPuesto.ID, nuevoPuesto.TenantID)
	if err != nil {
		return err
	}

	// 3. Buscamos qué plantilla le corresponde
	codigosPlantilla, existe := config.ConceptosBasePorRegimen[puestoGuardado.RegimenCodigo]
	if !existe {
		return nil // Régimen sin plantilla por defecto
	}

	// 4. Traducimos los códigos SUNAT a los IDs locales
	idsLocales, err := s.Repo.ObtenerConceptosTenantPorCodigosSUNAT(nuevoPuesto.TenantID, codigosPlantilla)
	if err != nil {
		log.Println("Error al obtener plantilla de conceptos:", err)
		return nil
	}

	// 5. Insertamos los conceptos
	return s.Repo.AsignarConceptosAPuesto(nuevoPuesto.ID, idsLocales, nuevoPuesto.SueldoPresupuestado)
}
