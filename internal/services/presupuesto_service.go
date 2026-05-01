package services

import (
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
)

type PresupuestoService struct {
	PresupuestoRepo    *repository.PresupuestoRepository
	PuestoRepo         *repository.PuestoRepository
	PuestoConceptoRepo *repository.PuestoConceptoRepository
	PlanillaSvc        *PlanillaService // Reutilizamos tu servicio de planillas para acceder al simulador
}

func NewPresupuestoService(
	pr *repository.PresupuestoRepository,
	pRepo *repository.PuestoRepository,
	pConceptoRepo *repository.PuestoConceptoRepository,
	pSvc *PlanillaService,
) *PresupuestoService {
	return &PresupuestoService{
		PresupuestoRepo:    pr,
		PuestoRepo:         pRepo,
		PuestoConceptoRepo: pConceptoRepo,
		PlanillaSvc:        pSvc,
	}
}

// GenerarProyeccionPIA crea la proyección de 12 meses evaluando cada puesto
func (s *PresupuestoService) GenerarProyeccionPIA(tenantID int, anio int, parametrosGlobales map[string]float64, mapaCodigos map[string]int, mapaAfectaciones map[int][]int) error {

	// 1. Obtener todos los puestos activos
	puestos, err := s.PuestoRepo.ObtenerPuestosParaPAP(tenantID)
	if err != nil {
		return err
	}

	// Esta estructura nos sirve como "Llave" para agrupar en el mapa
	type AgrupacionKey struct {
		Meta         string
		Fuente       string
		Clasificador string
	}

	// Diccionario para acumular los montos: Llave -> Detalle de la fila
	matriz := make(map[AgrupacionKey]*models.PapDetalle)

	// 2. Simular cada puesto mes a mes
	for _, p := range puestos {
		// Asignamos clasificadores genéricos por ahora (se pueden ajustar por régimen luego)
		clasifIngresos := "2.1.1.1.1"
		descIngresos := "REMUNERACIONES Y DIETAS"
		clasifAportes := "2.1.3.1.1"
		descAportes := "OBLIGACIONES DEL EMPLEADOR (ESSALUD)"

		// Obtenemos los conceptos del puesto inyectando tu repositorio de PuestoConceptos
		// (Asegúrate de agregar PuestoConceptoRepo a la estructura PresupuestoService)
		conceptosPlaza, err := s.PuestoConceptoRepo.ObtenerParaCalculo(p.ID)
		if err != nil {
			// Si hay un error, lo registramos pero continuamos con el siguiente puesto
			continue
		}

		for mes := 1; mes <= 12; mes++ {
			ingresos, aportes, _ := s.PlanillaSvc.SimularCostoMensualPuesto(p.ID, p.RegimenCodigo, conceptosPlaza, anio, mes, parametrosGlobales, mapaCodigos, mapaAfectaciones)

			// Acumular Ingresos
			if ingresos > 0 {
				keyIngreso := AgrupacionKey{p.MetaCodigo, p.FuenteRubroCodigo, clasifIngresos}
				if _, existe := matriz[keyIngreso]; !existe {
					matriz[keyIngreso] = &models.PapDetalle{
						MetaCodigo: p.MetaCodigo, MetaDescripcion: p.MetaDescripcion,
						FuenteRubroCodigo: p.FuenteRubroCodigo, FuenteRubroDescripcion: p.FuenteRubroDescripcion,
						ClasificadorCodigoLimpio: clasifIngresos, ClasificadorDescripcion: descIngresos,
					}
				}
				matriz[keyIngreso].Meses[mes-1] += ingresos
				matriz[keyIngreso].TotalAnual += ingresos
			}

			// Acumular Aportes Patronales
			if aportes > 0 {
				keyAporte := AgrupacionKey{p.MetaCodigo, p.FuenteRubroCodigo, clasifAportes}
				if _, existe := matriz[keyAporte]; !existe {
					matriz[keyAporte] = &models.PapDetalle{
						MetaCodigo: p.MetaCodigo, MetaDescripcion: p.MetaDescripcion,
						FuenteRubroCodigo: p.FuenteRubroCodigo, FuenteRubroDescripcion: p.FuenteRubroDescripcion,
						ClasificadorCodigoLimpio: clasifAportes, ClasificadorDescripcion: descAportes,
					}
				}
				matriz[keyAporte].Meses[mes-1] += aportes
				matriz[keyAporte].TotalAnual += aportes
			}
		}
	}

	// 3. Convertir el diccionario a una lista (Slice)
	var listaDetalles []models.PapDetalle
	for _, fila := range matriz {
		listaDetalles = append(listaDetalles, *fila)
	}

	// 4. Guardar la Versión y los Detalles
	nuevaVersion := models.PapVersion{TenantID: tenantID, Anio: anio, Tipo: "PIA"}
	err = s.PresupuestoRepo.CrearVersion(&nuevaVersion)
	if err != nil {
		return err
	}

	return s.PresupuestoRepo.GuardarDetallesMasivo(nuevaVersion.ID, listaDetalles)
}

// GenerarProyeccionAnioVigente crea la matriz combinando el gasto real ejecutado y la proyección futura
func (s *PresupuestoService) GenerarProyeccionAnioVigente(
	tenantID int, anio int, mesCorte int,
	parametrosGlobales map[string]float64,
	mapaCodigos map[string]int,
	mapaAfectaciones map[int][]int,
) error {

	puestos, err := s.PuestoRepo.ObtenerPuestosParaPAP(tenantID)
	if err != nil {
		return err
	}

	type AgrupacionKey struct {
		Meta         string
		Fuente       string
		Clasificador string
	}
	matriz := make(map[AgrupacionKey]*models.PapDetalle)

	for _, p := range puestos {
		clasifIngresos := "2.1.1.1.1"
		descIngresos := "REMUNERACIONES Y DIETAS"
		clasifAportes := "2.1.3.1.1"
		descAportes := "OBLIGACIONES DEL EMPLEADOR (ESSALUD)"

		conceptosPlaza, err := s.PuestoConceptoRepo.ObtenerParaCalculo(p.ID)
		if err != nil {
			continue
		}

		for mes := 1; mes <= 12; mes++ {
			var ingresos, aportes float64

			if mes <= mesCorte {
				// AQUÍ USAMOS EL GASTO REAL DE LAS PLANILLAS CERRADAS
				// Accedemos al repositorio de planillas a través de tu servicio inyectado
				ingresos, aportes, _ = s.PlanillaSvc.Repo.ObtenerGastoRealPorPuesto(p.ID, anio, mes)
			} else {
				// AQUÍ USAMOS EL SIMULADOR PARA EL FUTURO
				ingresos, aportes, _ = s.PlanillaSvc.SimularCostoMensualPuesto(p.ID, p.RegimenCodigo, conceptosPlaza, anio, mes, parametrosGlobales, mapaCodigos, mapaAfectaciones)
			}

			// Acumular Ingresos
			if ingresos > 0 {
				keyIngreso := AgrupacionKey{p.MetaCodigo, p.FuenteRubroCodigo, clasifIngresos}
				if _, existe := matriz[keyIngreso]; !existe {
					matriz[keyIngreso] = &models.PapDetalle{
						MetaCodigo: p.MetaCodigo, MetaDescripcion: p.MetaDescripcion,
						FuenteRubroCodigo: p.FuenteRubroCodigo, FuenteRubroDescripcion: p.FuenteRubroDescripcion,
						ClasificadorCodigoLimpio: clasifIngresos, ClasificadorDescripcion: descIngresos,
					}
				}
				matriz[keyIngreso].Meses[mes-1] += ingresos
				matriz[keyIngreso].TotalAnual += ingresos
			}

			// Acumular Aportes
			if aportes > 0 {
				keyAporte := AgrupacionKey{p.MetaCodigo, p.FuenteRubroCodigo, clasifAportes}
				if _, existe := matriz[keyAporte]; !existe {
					matriz[keyAporte] = &models.PapDetalle{
						MetaCodigo: p.MetaCodigo, MetaDescripcion: p.MetaDescripcion,
						FuenteRubroCodigo: p.FuenteRubroCodigo, FuenteRubroDescripcion: p.FuenteRubroDescripcion,
						ClasificadorCodigoLimpio: clasifAportes, ClasificadorDescripcion: descAportes,
					}
				}
				matriz[keyAporte].Meses[mes-1] += aportes
				matriz[keyAporte].TotalAnual += aportes
			}
		}
	}

	var listaDetalles []models.PapDetalle
	for _, fila := range matriz {
		listaDetalles = append(listaDetalles, *fila)
	}

	// Guardamos la versión como AÑO VIGENTE
	nuevaVersion := models.PapVersion{TenantID: tenantID, Anio: anio, Tipo: "AÑO VIGENTE"}
	err = s.PresupuestoRepo.CrearVersion(&nuevaVersion)
	if err != nil {
		return err
	}

	return s.PresupuestoRepo.GuardarDetallesMasivo(nuevaVersion.ID, listaDetalles)
}
