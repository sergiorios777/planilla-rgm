package services

import (
	"database/sql"
	"fmt"
	"math"
	"planilla-rgm/internal/repository"
	"time"
)

type TipoBeneficio string

const (
	BeneficioCTS           TipoBeneficio = "CTS"
	BeneficioGratificacion TipoBeneficio = "GRATIFICACION"
	BeneficioVacaciones    TipoBeneficio = "VACACIONES"
	BeneficioVacTruncas    TipoBeneficio = "VAC_TRUNCAS"
	BeneficioVacNoGozadas  TipoBeneficio = "VAC_NO_GOZADAS"
	BeneficioAguinaldo276  TipoBeneficio = "AGUINALDO_276"
	BeneficioAsig2530      TipoBeneficio = "ASIG_25_30"
	BeneficioSubsidioSep   TipoBeneficio = "SUBSIDIO_SEPELIO"
)

// ConceptoBaseItem detalla cada concepto individual que compone la base computable para auditoría
type ConceptoBaseItem struct {
	ConceptoTenantID int     `json:"concepto_tenant_id"`
	CodigoSunat      string  `json:"codigo_sunat"`
	NombreConcepto   string  `json:"nombre_concepto"`
	ModalidadEntrega string  `json:"modalidad_entrega"`
	TipoVariable     string  `json:"tipo_variable"` // REMUNERACION_BASICA, ASIGNACION_FAMILIAR, SEXTO_GRATIFICACION, etc.
	MontoOriginal    float64 `json:"monto_original"`
	MontoComputable  float64 `json:"monto_computable"`
}

// DesgloseBaseComputable contiene el total de la base y su composición desglosada
type DesgloseBaseComputable struct {
	TotalComputable float64            `json:"total_computable"`
	SueldoBasico    float64            `json:"sueldo_basico"`
	AsigFamiliar    float64            `json:"asig_familiar"`
	SextoGrati      float64            `json:"sexto_grati"`
	PromedioVar     float64            `json:"promedio_var"`
	Items           []ConceptoBaseItem `json:"items"`
}

// BaseComputableService centraliza la resolución legal de remuneraciones computables
type BaseComputableService struct {
	db              *sql.DB
	BaseRegimenRepo *repository.BaseRegimenRepository
}

func NewBaseComputableService(db *sql.DB) *BaseComputableService {
	return &BaseComputableService{
		db:              db,
		BaseRegimenRepo: repository.NewBaseRegimenRepository(db),
	}
}

// ResolverBaseComputable determina la base de cálculo aplicable según régimen laboral, tipo de beneficio y entidad
func (s *BaseComputableService) ResolverBaseComputable(
	tenantID, contratoID, puestoID, regimenID int,
	regimenCodigo string,
	beneficio TipoBeneficio,
	fechaCorte time.Time,
) (*DesgloseBaseComputable, error) {

	desglose := &DesgloseBaseComputable{
		Items: make([]ConceptoBaseItem, 0),
	}

	// 1. Obtener tipo de entidad del tenant (Gobierno Local, Regional, Nacional)
	var tipoEntidad string
	err := s.db.QueryRow("SELECT COALESCE(tipo_entidad, 'GOBIERNO_LOCAL') FROM tenants WHERE id = $1", tenantID).Scan(&tipoEntidad)
	if err != nil {
		tipoEntidad = "GOBIERNO_LOCAL"
	}
	esGobiernoLocal := (tipoEntidad == "GOBIERNO_LOCAL")

	// 2. Obtener sueldo presupuestado del puesto como salvaguarda (fallback)
	var sueldoPresupuestado float64
	_ = s.db.QueryRow("SELECT COALESCE(sueldo_presupuestado, 0.0) FROM puestos WHERE id = $1", puestoID).Scan(&sueldoPresupuestado)

	// Normalizar código de beneficio para consultas en base_regimen_tenant
	codInternoCalculado := string(beneficio)
	if beneficio == BeneficioVacaciones {
		codInternoCalculado = "VAC_TRUNCAS"
	}

	switch regimenCodigo {
	case "276":
		if esGobiernoLocal {
			// =========================================================================
			// D.L. 276 EN GOBIERNOS LOCALES (Municipalidades):
			// - CTS: Ley 32199 (17/12/2024) -> 100% de conceptos PERMANENTES al cese.
			// - Vacaciones: D.S. 420-2019-EF DT Única -> Ingreso mensual permanente Genérica 2.1 (excl. CAFAE).
			// =========================================================================
			queryMuni := `
				SELECT ct.id, COALESCE(cm.codigo, ''), ct.nombre_personalizado, 
				       COALESCE(ct.modalidad_entrega, 'PERMANENTE'), COALESCE(pc.monto, 0.0)
				FROM puesto_conceptos pc
				INNER JOIN conceptos_tenant ct ON pc.concepto_tenant_id = ct.id
				LEFT JOIN conceptos_maestros cm ON ct.concepto_id = cm.id
				WHERE pc.puesto_id = $1 
				  AND pc.activo = true 
				  AND ct.activo = true 
				  AND ct.tenant_id = $2
				  AND ct.modalidad_entrega = 'PERMANENTE'
			`
			rows, err := s.db.Query(queryMuni, puestoID, tenantID)
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var item ConceptoBaseItem
					if err := rows.Scan(&item.ConceptoTenantID, &item.CodigoSunat, &item.NombreConcepto, &item.ModalidadEntrega, &item.MontoOriginal); err == nil {
						item.TipoVariable = "CONCEPTO_PERMANENTE_MUNI"
						item.MontoComputable = item.MontoOriginal
						desglose.TotalComputable += item.MontoComputable
						desglose.Items = append(desglose.Items, item)
					}
				}
			}

			if desglose.TotalComputable <= 0 && sueldoPresupuestado > 0 {
				desglose.TotalComputable = sueldoPresupuestado
			}

		} else {
			// =========================================================================
			// D.L. 276 EN GOBIERNO NACIONAL / GOBIERNOS REGIONALES (AIRHSP / MUC / BET):
			// - Vacaciones: MUC + BET Fijo (D.S. 420-2019-EF Art. 4).
			// - CTS: Promedio 36 meses MUC + CAFAE (Ley 32199 / D.U. 038-2019).
			// =========================================================================
			if beneficio == BeneficioCTS {
				prom36, err := s.calcularPromedio36MesesMUCyCAFAE(contratoID, fechaCorte)
				if err == nil && prom36 > 0 {
					desglose.TotalComputable = prom36
				} else {
					muc, _ := s.BaseRegimenRepo.ObtenerMontoVariable(tenantID, puestoID, regimenID, "CTS", "MUC")
					bet, _ := s.BaseRegimenRepo.ObtenerMontoVariable(tenantID, puestoID, regimenID, "CTS", "BET")
					if muc+bet > 0 {
						desglose.TotalComputable = muc + bet
					} else {
						desglose.TotalComputable = sueldoPresupuestado
					}
				}
			} else {
				// Vacaciones / Truncas / No Gozadas / Luto
				muc, _ := s.BaseRegimenRepo.ObtenerMontoVariable(tenantID, puestoID, regimenID, codInternoCalculado, "MUC")
				bet, _ := s.BaseRegimenRepo.ObtenerMontoVariable(tenantID, puestoID, regimenID, codInternoCalculado, "BET")
				if muc <= 0 && bet <= 0 {
					// Intento con código CTS si no hay en VAC_TRUNCAS
					muc, _ = s.BaseRegimenRepo.ObtenerMontoVariable(tenantID, puestoID, regimenID, "CTS", "MUC")
					bet, _ = s.BaseRegimenRepo.ObtenerMontoVariable(tenantID, puestoID, regimenID, "CTS", "BET")
				}
				if muc+bet > 0 {
					desglose.TotalComputable = muc + bet
					if muc > 0 {
						desglose.Items = append(desglose.Items, ConceptoBaseItem{NombreConcepto: "MUC", TipoVariable: "MUC", MontoComputable: muc})
					}
					if bet > 0 {
						desglose.Items = append(desglose.Items, ConceptoBaseItem{NombreConcepto: "BET", TipoVariable: "BET", MontoComputable: bet})
					}
				} else {
					desglose.TotalComputable = sueldoPresupuestado
				}
			}
		}

	case "728":
		// =============================================================================
		// D.L. 728 (Actividad Privada):
		// 1. Remuneración Básica + Asignación Familiar
		// 2. Promedio de Variables Regulares (3 de 6 meses)
		// 3. + 1/6 Gratificación anterior EXCLUSIVAMENTE para CTS
		// =============================================================================
		// 1. Fijos (Básico y Asignación Familiar)
		basico, _ := s.BaseRegimenRepo.ObtenerMontoVariable(tenantID, puestoID, regimenID, codInternoCalculado, "REMUNERACION_BASICA")
		if basico <= 0 {
			basico, _ = s.BaseRegimenRepo.ObtenerMontoVariable(tenantID, puestoID, regimenID, "CTS", "REMUNERACION_BASICA")
		}
		if basico <= 0 {
			basico = sueldoPresupuestado
		}
		desglose.SueldoBasico = basico
		desglose.TotalComputable += basico
		desglose.Items = append(desglose.Items, ConceptoBaseItem{
			NombreConcepto:  "Remuneración Básica",
			TipoVariable:    "REMUNERACION_BASICA",
			MontoComputable: basico,
		})

		asigFam, _ := s.BaseRegimenRepo.ObtenerMontoVariable(tenantID, puestoID, regimenID, codInternoCalculado, "ASIGNACION_FAMILIAR")
		if asigFam <= 0 {
			asigFam, _ = s.BaseRegimenRepo.ObtenerMontoVariable(tenantID, puestoID, regimenID, "CTS", "ASIGNACION_FAMILIAR")
		}
		if asigFam > 0 {
			desglose.AsigFamiliar = asigFam
			desglose.TotalComputable += asigFam
			desglose.Items = append(desglose.Items, ConceptoBaseItem{
				NombreConcepto:  "Asignación Familiar",
				TipoVariable:    "ASIGNACION_FAMILIAR",
				MontoComputable: asigFam,
			})
		}

		// 2. Promedio de Variables Regulares (percibidas >= 3 meses en el semestre previo)
		promVar, varItems, _ := s.calcularPromedioVariablesSemestre(contratoID, fechaCorte)
		if promVar > 0 {
			desglose.PromedioVar = promVar
			desglose.TotalComputable += promVar
			desglose.Items = append(desglose.Items, varItems...)
		}

		// 3. Sexto de Gratificación: SOLO para CTS
		if beneficio == BeneficioCTS {
			sextoGrati, gratiItem, _ := s.obtenerUltimoSextoGratificacion(contratoID, fechaCorte, basico+asigFam)
			if sextoGrati > 0 {
				desglose.SextoGrati = sextoGrati
				desglose.TotalComputable += sextoGrati
				if gratiItem != nil {
					desglose.Items = append(desglose.Items, *gratiItem)
				}
			}
		}

	case "1057", "CAS":
		// =============================================================================
		// D.L. 1057 (CAS):
		// Retribución Mensual pactada en contrato / puesto
		// =============================================================================
		retribucion, _ := s.BaseRegimenRepo.ObtenerMontoVariable(tenantID, puestoID, regimenID, codInternoCalculado, "RETRIBUCION_MENSUAL")
		if retribucion <= 0 {
			retribucion, _ = s.BaseRegimenRepo.ObtenerMontoVariable(tenantID, puestoID, regimenID, "CTS", "RETRIBUCION_MENSUAL")
		}
		if retribucion <= 0 {
			retribucion = sueldoPresupuestado
		}
		desglose.TotalComputable = retribucion
		desglose.Items = append(desglose.Items, ConceptoBaseItem{
			NombreConcepto:  "Retribución Mensual CAS",
			TipoVariable:    "RETRIBUCION_MENSUAL",
			MontoComputable: retribucion,
		})

	case "30057":
		// =============================================================================
		// LEY 30057 (SERVIR):
		// - CTS: Promedio de compensaciones de últimos 36 meses efectivamente laborados.
		// - Vacaciones / Gratificaciones: Valorización Principal + Valorización Ajustada.
		// =============================================================================
		if beneficio == BeneficioCTS {
			prom36, err := s.calcularPromedio36MesesServir(contratoID, fechaCorte)
			if err == nil && prom36 > 0 {
				desglose.TotalComputable = prom36
			} else {
				vp, _ := s.BaseRegimenRepo.ObtenerMontoVariable(tenantID, puestoID, regimenID, "VAC_TRUNCAS", "VALORIZACION_PRINCIPAL")
				va, _ := s.BaseRegimenRepo.ObtenerMontoVariable(tenantID, puestoID, regimenID, "VAC_TRUNCAS", "VALORIZACION_AJUSTADA")
				if vp+va > 0 {
					desglose.TotalComputable = vp + va
				} else {
					desglose.TotalComputable = sueldoPresupuestado
				}
			}
		} else {
			vp, _ := s.BaseRegimenRepo.ObtenerMontoVariable(tenantID, puestoID, regimenID, codInternoCalculado, "VALORIZACION_PRINCIPAL")
			va, _ := s.BaseRegimenRepo.ObtenerMontoVariable(tenantID, puestoID, regimenID, codInternoCalculado, "VALORIZACION_AJUSTADA")
			if vp <= 0 && va <= 0 {
				vp, _ = s.BaseRegimenRepo.ObtenerMontoVariable(tenantID, puestoID, regimenID, "VAC_TRUNCAS", "VALORIZACION_PRINCIPAL")
				va, _ = s.BaseRegimenRepo.ObtenerMontoVariable(tenantID, puestoID, regimenID, "VAC_TRUNCAS", "VALORIZACION_AJUSTADA")
			}
			if vp+va > 0 {
				desglose.TotalComputable = vp + va
				if vp > 0 {
					desglose.Items = append(desglose.Items, ConceptoBaseItem{NombreConcepto: "Valorización Principal", TipoVariable: "VALORIZACION_PRINCIPAL", MontoComputable: vp})
				}
				if va > 0 {
					desglose.Items = append(desglose.Items, ConceptoBaseItem{NombreConcepto: "Valorización Ajustada", TipoVariable: "VALORIZACION_AJUSTADA", MontoComputable: va})
				}
			} else {
				desglose.TotalComputable = sueldoPresupuestado
			}
		}

	default:
		desglose.TotalComputable = sueldoPresupuestado
	}

	desglose.TotalComputable = math.Round(desglose.TotalComputable*100) / 100
	return desglose, nil
}

// calcularPromedioVariablesSemestre calcula el promedio de variables que cumplen el principio de regularidad (>= 3 de 6 meses)
func (s *BaseComputableService) calcularPromedioVariablesSemestre(contratoID int, fechaCorte time.Time) (float64, []ConceptoBaseItem, error) {
	fechaDesde := fechaCorte.AddDate(0, -6, 0)

	query := `
		SELECT pc.maestro_id, COALESCE(pc.nombre_en_boleta, 'Concepto Variable'),
		       COUNT(DISTINCT p.id) AS veces_percibido,
		       SUM(pc.monto) AS suma_monto
		FROM planilla_conceptos pc
		INNER JOIN planilla_detalles pd ON pc.planilla_detalle_id = pd.id
		INNER JOIN planillas p ON pd.planilla_id = p.id
		LEFT JOIN conceptos_tenant ct ON pc.concepto_tenant_id = ct.id
		WHERE pd.contrato_id = $1
		  AND p.estado = 'CERRADA'
		  AND pc.tipo_concepto = 'INGRESO'
		  AND make_date(p.anio, p.mes, 1) >= $2
		  AND make_date(p.anio, p.mes, 1) <= $3
		  AND (ct.modalidad_entrega IN ('PERIODICO', 'OCASIONAL') OR ct.es_extraordinario = true OR ct.es_ocasional = true)
		  AND COALESCE(pc.codigo_sunat, '') NOT IN ('2002', '0201', '0406', '0312')
		GROUP BY pc.maestro_id, pc.nombre_en_boleta
		HAVING COUNT(DISTINCT p.id) >= 3
	`

	rows, err := s.db.Query(query, contratoID, fechaDesde, fechaCorte)
	if err != nil {
		return 0.0, nil, err
	}
	defer rows.Close()

	var items []ConceptoBaseItem
	var sumaTotal float64

	for rows.Next() {
		var maestroID, veces int
		var nombre string
		var suma float64
		if err := rows.Scan(&maestroID, &nombre, &veces, &suma); err == nil {
			prom := math.Round((suma/6.0)*100) / 100
			sumaTotal += prom
			items = append(items, ConceptoBaseItem{
				NombreConcepto:  fmt.Sprintf("Promedio %s (%d/6 meses)", nombre, veces),
				TipoVariable:    "REMUNERACION_VARIABLE",
				MontoOriginal:   suma,
				MontoComputable: prom,
			})
		}
	}

	return sumaTotal, items, nil
}

// obtenerUltimoSextoGratificacion busca el 1/6 de la última gratificación ordinaria percibida o calcula el estimado
func (s *BaseComputableService) obtenerUltimoSextoGratificacion(contratoID int, fechaCorte time.Time, baseEstimada float64) (float64, *ConceptoBaseItem, error) {
	// Determinar el periodo de la última gratificación ordinaria
	mesCorte := int(fechaCorte.Month())
	anioCorte := fechaCorte.Year()

	var mesGrati, anioGrati int
	if mesCorte >= 5 && mesCorte <= 10 {
		// Periodo de Mayo a Octubre -> última gratificación fue en Diciembre del año anterior
		mesGrati = 12
		anioGrati = anioCorte - 1
	} else if mesCorte >= 11 {
		// Periodo de Noviembre a Diciembre -> última gratificación fue en Julio del mismo año
		mesGrati = 7
		anioGrati = anioCorte
	} else {
		// Periodo de Enero a Abril -> última gratificación fue en Diciembre del año anterior
		mesGrati = 12
		anioGrati = anioCorte - 1
	}

	query := `
		SELECT COALESCE(pc.monto, 0.0)
		FROM planilla_conceptos pc
		INNER JOIN planilla_detalles pd ON pc.planilla_detalle_id = pd.id
		INNER JOIN planillas p ON pd.planilla_id = p.id
		WHERE pd.contrato_id = $1
		  AND p.anio = $2 AND p.mes = $3
		  AND (pc.codigo_sunat = '0406' OR pc.nombre_en_boleta ILIKE '%gratificac%')
		LIMIT 1
	`
	var montoGrati float64
	err := s.db.QueryRow(query, contratoID, anioGrati, mesGrati).Scan(&montoGrati)
	if err != nil || montoGrati <= 0 {
		// Fallback: Si no tiene boleta registrada de gratificación anterior, estimar 1/6 sobre sueldo base + asignación
		montoGrati = baseEstimada
	}

	sexto := math.Round((montoGrati/6.0)*100) / 100
	item := &ConceptoBaseItem{
		NombreConcepto:  fmt.Sprintf("1/6 Gratificación (%d-%02d)", anioGrati, mesGrati),
		TipoVariable:    "SEXTO_GRATIFICACION",
		MontoOriginal:   montoGrati,
		MontoComputable: sexto,
	}

	return sexto, item, nil
}

// calcularPromedio36MesesMUCyCAFAE calcula el promedio mensual de MUC + CAFAE en los últimos 36 meses (D.L. 276 GN/GR)
func (s *BaseComputableService) calcularPromedio36MesesMUCyCAFAE(contratoID int, fechaCorte time.Time) (float64, error) {
	fechaDesde := fechaCorte.AddDate(-3, 0, 0)
	query := `
		SELECT COALESCE(SUM(pc.monto) / NULLIF(COUNT(DISTINCT p.id), 0), 0.0)
		FROM planilla_conceptos pc
		INNER JOIN planilla_detalles pd ON pc.planilla_detalle_id = pd.id
		INNER JOIN planillas p ON pd.planilla_id = p.id
		LEFT JOIN conceptos_tenant ct ON pc.concepto_tenant_id = ct.id
		WHERE pd.contrato_id = $1
		  AND p.estado = 'CERRADA'
		  AND make_date(p.anio, p.mes, 1) >= $2
		  AND make_date(p.anio, p.mes, 1) <= $3
		  AND (ct.nombre_personalizado ILIKE '%muc%' OR ct.nombre_personalizado ILIKE '%cafae%')
	`
	var prom float64
	err := s.db.QueryRow(query, contratoID, fechaDesde, fechaCorte).Scan(&prom)
	return math.Round(prom*100) / 100, err
}

// calcularPromedio36MesesServir calcula el promedio mensual de compensaciones en los últimos 36 meses (Ley 30057)
func (s *BaseComputableService) calcularPromedio36MesesServir(contratoID int, fechaCorte time.Time) (float64, error) {
	fechaDesde := fechaCorte.AddDate(-3, 0, 0)
	query := `
		SELECT COALESCE(AVG(pd.total_ingresos), 0.0)
		FROM planilla_detalles pd
		INNER JOIN planillas p ON pd.planilla_id = p.id
		WHERE pd.contrato_id = $1
		  AND p.estado = 'CERRADA'
		  AND make_date(p.anio, p.mes, 1) >= $2
		  AND make_date(p.anio, p.mes, 1) <= $3
	`
	var prom float64
	err := s.db.QueryRow(query, contratoID, fechaDesde, fechaCorte).Scan(&prom)
	return math.Round(prom*100) / 100, err
}
