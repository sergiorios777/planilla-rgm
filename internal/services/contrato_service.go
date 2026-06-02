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

	// 2. Sincronizar conceptos del puesto
	_, err = s.SincronizarConceptosPuesto(c.TenantID, c.PuestoID)
	return err
}

// SincronizarConceptosPuesto unifica las reglas para poblar la estructura de costos de un puesto.
// Retorna true si el puesto tiene contrato activo (ocupado), false si está vacante, y el error si ocurre alguno.
func (s *ContratoService) SincronizarConceptosPuesto(tenantID, puestoID int) (bool, error) {
	// 1. Obtener detalles del puesto
	puesto, err := s.RepoPuesto.ObtenerPorID(puestoID, tenantID)
	if err != nil {
		return false, err
	}

	// 2. Verificar si existe un contrato activo para este puesto
	contrato, err := s.Repo.ObtenerActivoPorPuesto(puestoID, tenantID)
	if err != nil {
		return false, err
	}

	// Estrategia 2: Vacante (Sin contrato activo)
	if contrato == nil {
		err = s.RepoPuesto.RestaurarPlantillaBase(puestoID, tenantID, puesto.RegimenID)
		if err != nil {
			return false, err
		}
		return false, nil
	}

	// Estrategia 1: Ocupado (Con contrato activo)
	// A. Obtener la plantilla de conceptos base según el régimen (excluyendo pensiones y clasificadores de contratos específicos)
	var excluidos []string
	for _, mappings := range config.ClasificadorMefPorContrato {
		for _, codigoMef := range mappings {
			excluidos = append(excluidos, codigoMef)
		}
	}

	idsLocales, err := s.RepoPuesto.ObtenerConceptosModeloPorRegimen(tenantID, puesto.RegimenID, excluidos)
	if err != nil {
		return true, err
	}

	// B. Mapear régimen a la clave del mapa y buscar clasificador MEF
	key := config.MapRegimenToKey(puesto.RegimenCodigo)
	if key != "" {
		if options, ok := config.ClasificadorMefPorContrato[key]; ok {
			if codigoMef, ok := options[contrato.TipoContrato]; ok {
				// Buscar el ID del concepto local correspondiente a este clasificador MEF
				conceptoID, err := s.RepoPuesto.ObtenerConceptoRemunerativoPorClasificador(tenantID, puesto.RegimenID, codigoMef)
				if err != nil {
					log.Printf("Advertencia: no se encontró el concepto local para el clasificador %s bajo régimen %d: %v", codigoMef, puesto.RegimenID, err)
				} else {
					idsLocales = append(idsLocales, conceptoID)
				}
			}
		}
	}

	// C. Obtener pensiones del trabajador
	trabajador, err := s.RepoTrabajador.ObtenerPorID(contrato.TrabajadorID, tenantID)
	if err != nil {
		return true, err
	}

	codigosPension, existe := config.PensionesBase[trabajador.RegimenPensionario]
	if existe {
		idsPensiones, err := s.RepoPuesto.ObtenerConceptosTenantPorCodigosSUNAT(tenantID, codigosPension)
		if err == nil && len(idsPensiones) > 0 {
			idsLocales = append(idsLocales, idsPensiones...)
		}
	}

	// D. Limpiar todos los conceptos actuales del puesto
	err = s.RepoPuesto.LimpiarConceptosPuesto(puestoID)
	if err != nil {
		return true, err
	}

	// E. Asignar todos los conceptos calculados al puesto
	err = s.RepoPuesto.AsignarConceptosAPuesto(puestoID, idsLocales, puesto.SueldoPresupuestado)
	if err != nil {
		return true, err
	}

	return true, nil
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
