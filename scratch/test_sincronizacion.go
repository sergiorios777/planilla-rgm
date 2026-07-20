//go:build ignore

package main

import (
	"database/sql"
	"fmt"
	"log"
	"planilla-rgm/internal/config"
	"planilla-rgm/internal/repository"
	"planilla-rgm/internal/services"

	_ "github.com/lib/pq"
)

func main() {
	db, err := sql.Open("postgres", "host=localhost port=5432 user=postgres password=Admin001 dbname=planilla_rgm sslmode=disable")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	defer db.Close()

	repoContrato := repository.NewContratoRepository(db)
	repoTrabajador := repository.NewTrabajadorRepository(db)
	repoPuesto := repository.NewPuestoRepository(db)

	service := &services.ContratoService{
		Repo:           repoContrato,
		RepoTrabajador: repoTrabajador,
		RepoPuesto:     repoPuesto,
	}

	// Probamos para tenant 1 y puesto 52
	tenantID := 1
	puestoID := 52

	fmt.Println("--- INICIANDO PRUEBA DE SINCRONIZACIÓN ---")

	// 1. Obtener detalles del puesto
	puesto, err := repoPuesto.ObtenerPorID(puestoID, tenantID)
	if err != nil {
		log.Fatalf("Error puesto: %v", err)
	}
	fmt.Printf("Puesto ID: %d, Nombre: %s, Regimen ID: %d, RegimenCodigo: %s\n",
		puesto.ID, puesto.Nombre, puesto.RegimenID, puesto.RegimenCodigo)

	// 2. Verificar si existe un contrato activo para este puesto
	contrato, err := repoContrato.ObtenerActivoPorPuesto(puestoID, tenantID)
	if err != nil {
		log.Fatalf("Error contrato: %v", err)
	}

	if contrato == nil {
		fmt.Println("No se encontró contrato activo.")
		return
	}
	fmt.Printf("Contrato ID: %d, TipoContrato: %q, Nivel: %q, Activo: %t\n",
		contrato.ID, contrato.TipoContrato, contrato.Nivel, contrato.Activo)

	// 3. Obtener la plantilla de conceptos base según el régimen
	var excluidos []string
	for _, mappings := range config.ClasificadorMefPorContrato {
		for _, codigoMef := range mappings {
			excluidos = append(excluidos, codigoMef)
		}
	}

	idsLocales, err := repoPuesto.ObtenerConceptosModeloPorRegimen(tenantID, puesto.RegimenID, excluidos)
	if err != nil {
		log.Fatalf("Error base: %v", err)
	}
	fmt.Printf("Conceptos base obtenidos (IDs): %v\n", idsLocales)

	// 4. Mapear régimen y buscar clasificador MEF
	key := config.MapRegimenToKey(puesto.RegimenCodigo)
	fmt.Printf("RegimenCodigo: %q mapped to Key: %q\n", puesto.RegimenCodigo, key)
	if key != "" {
		if options, ok := config.ClasificadorMefPorContrato[key]; ok {
			fmt.Printf("Options found for key %q: %v\n", key, options)
			if codigoMef, ok := options[contrato.TipoContrato]; ok {
				fmt.Printf("MEF code found for %q: %q\n", contrato.TipoContrato, codigoMef)
				conceptoID, err := repoPuesto.ObtenerConceptoRemunerativoPorClasificador(tenantID, puesto.RegimenID, codigoMef)
				if err != nil {
					fmt.Printf("Advertencia/Error al buscar concepto por clasificador %s: %v\n", codigoMef, err)
				} else {
					fmt.Printf("Concepto local ID encontrado para clasificador %s: %d\n", codigoMef, conceptoID)
					idsLocales = append(idsLocales, conceptoID)
				}
			} else {
				fmt.Printf("No MEF code found for TipoContrato %q in options\n", contrato.TipoContrato)
			}
		} else {
			fmt.Printf("No options found for key %q in ClasificadorMefPorContrato\n", key)
		}
	}

	// 5. Trabajador y pensión
	trabajador, err := repoTrabajador.ObtenerPorID(contrato.TrabajadorID, tenantID)
	if err != nil {
		log.Fatalf("Error trabajador: %v", err)
	}
	fmt.Printf("Trabajador ID: %d, Pension: %q\n", trabajador.ID, trabajador.RegimenPensionario)

	codigosPension, existe := config.PensionesBase[trabajador.RegimenPensionario]
	if existe {
		idsPensiones, err := repoPuesto.ObtenerConceptosTenantPorCodigosSUNAT(tenantID, codigosPension)
		if err != nil {
			fmt.Printf("Error pensiones: %v\n", err)
		} else {
			fmt.Printf("Pensiones locales IDs: %v\n", idsPensiones)
			idsLocales = append(idsLocales, idsPensiones...)
		}
	}

	fmt.Printf("Lista final de IDs a asignar: %v\n", idsLocales)

	// Ejecutar la sincronización real
	success, err := service.SincronizarConceptosPuesto(tenantID, puestoID)
	if err != nil {
		log.Fatalf("Error real sync: %v", err)
	}
	fmt.Printf("Resultado real sync: success = %t\n", success)
}
