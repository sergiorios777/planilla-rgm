package services

import (
	"planilla-rgm/internal/config"
	"planilla-rgm/internal/repository"
)

type ContratoService struct {
	Repo           *repository.ContratoRepository
	RepoTrabajador *repository.TrabajadorRepository
	RepoPuesto     *repository.PuestoRepository // Lo necesitamos para actualizar el puesto
}

// AsignarPensionesAutomaticas inyecta los conceptos de pensión a la plaza ocupada
func (s *ContratoService) AsignarPensionesAutomaticas(puestoID int, trabajadorID int, tenantID int) error {
	// 1. Obtenemos al trabajador para saber qué eligió
	trabajador, err := s.RepoTrabajador.ObtenerPorID(trabajadorID, tenantID)
	if err != nil {
		return err
	}

	// 2. Buscamos los códigos SUNAT que le corresponden (ONP o AFP)
	codigosPension, existe := config.PensionesBase[trabajador.RegimenPensionario]
	if !existe {
		return nil
	} // Prevención de errores si el dato está corrupto

	// 3. Traducimos esos códigos SUNAT a los IDs locales de la municipalidad (Tenant)
	// (Reutilizamos la función que creamos en el PuestoRepository)
	idsLocales, err := s.RepoPuesto.ObtenerConceptosTenantPorCodigosSUNAT(tenantID, codigosPension)
	if err != nil || len(idsLocales) == 0 {
		return err
	}

	// 4. Asignamos los conceptos al Puesto con monto 0.00
	return s.RepoPuesto.AsignarConceptosAPuesto(puestoID, idsLocales, 0.00)
}
