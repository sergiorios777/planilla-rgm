package services

import (
	"log"
	"planilla-rgm/internal/config"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
)

type ContratoService struct {
	Repo           *repository.ContratoRepository
	RepoTrabajador *repository.TrabajadorRepository
	RepoPuesto     *repository.PuestoRepository // Lo necesitamos para actualizar el puesto
}

// CrearContrato registra el contrato y realiza la inyección automática de conceptos correspondientes
func (s *ContratoService) CrearContrato(c *models.Contrato) error {
	// 1. Guardar el contrato en BD
	err := s.Repo.Crear(c)
	if err != nil {
		return err
	}

	// 2. Obtener los detalles del puesto ocupado
	puesto, err := s.RepoPuesto.ObtenerPorID(c.PuestoID, c.TenantID)
	if err != nil {
		return err
	}

	// 3. Obtener la plantilla de conceptos base según el régimen (excluyendo pensiones)
	idsLocales, err := s.RepoPuesto.ObtenerConceptosModeloPorRegimen(c.TenantID, puesto.RegimenID)
	if err != nil {
		return err
	}

	// 4. Mapear régimen a la clave del mapa y buscar clasificador MEF
	key := config.MapRegimenToKey(puesto.RegimenCodigo)
	if key != "" {
		if options, ok := config.ClasificadorMefPorContrato[key]; ok {
			if codigoMef, ok := options[c.TipoContrato]; ok {
				// Buscar el ID del concepto local correspondiente a este clasificador MEF
				conceptoID, err := s.RepoPuesto.ObtenerConceptoRemunerativoPorClasificador(c.TenantID, puesto.RegimenID, codigoMef)
				if err != nil {
					log.Printf("Advertencia: no se encontró el concepto local para el clasificador %s bajo régimen %d: %v", codigoMef, puesto.RegimenID, err)
				} else {
					idsLocales = append(idsLocales, conceptoID)
				}
			}
		}
	}

	// 5. Obtener pensiones del trabajador
	trabajador, err := s.RepoTrabajador.ObtenerPorID(c.TrabajadorID, c.TenantID)
	if err != nil {
		return err
	}

	codigosPension, existe := config.PensionesBase[trabajador.RegimenPensionario]
	if existe {
		idsPensiones, err := s.RepoPuesto.ObtenerConceptosTenantPorCodigosSUNAT(c.TenantID, codigosPension)
		if err == nil && len(idsPensiones) > 0 {
			idsLocales = append(idsLocales, idsPensiones...)
		}
	}

	// 6. Asignar todos los conceptos calculados al puesto
	return s.RepoPuesto.AsignarConceptosAPuesto(c.PuestoID, idsLocales, puesto.SueldoPresupuestado)
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
