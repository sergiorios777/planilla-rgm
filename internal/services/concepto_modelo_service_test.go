package services

import (
	"bytes"
	"github.com/joho/godotenv"
	"planilla-rgm/internal/config"
	"planilla-rgm/internal/repository"
	"testing"
)

func TestImportarDesdeCSV(t *testing.T) {
	// Intentar cargar .env desde la raíz del proyecto para pruebas locales
	_ = godotenv.Load("../../.env")

	// 1. Inicializar base de datos de prueba
	db, err := config.InitDB()
	if err != nil {
		t.Skip("Saltando prueba de integración: Base de datos no disponible o .env faltante")
		return
	}
	defer db.Close()

	// Limpieza selectiva al inicio y fin de la prueba para no afectar datos reales del desarrollador
	limpiarTest := func() {
		_, _ = db.Exec("DELETE FROM regimen_concepto_modelo WHERE concepto_modelo_id IN (SELECT id FROM conceptos_modelo WHERE nombre_personalizado LIKE '% Test')")
		_, _ = db.Exec("DELETE FROM conceptos_modelo WHERE nombre_personalizado LIKE '% Test'")
	}
	limpiarTest()
	defer limpiarTest()

	// Asegurar datos maestros mínimos
	_, _ = db.Exec(`INSERT INTO conceptos_maestros (codigo, descripcion, tipo, activo) 
		VALUES ('0121', 'Remuneración Principal', 'Ingreso', true)
		ON CONFLICT (codigo) DO UPDATE SET activo = true`)
		
	_, _ = db.Exec(`INSERT INTO clasificadores_mef (codigo, codigo_limpio, descripcion, nivel, tipo_transaccion, activo)
		VALUES ('2.1.1 1.1 1', '2.1.1.1.1.1', 'Personal Administrativo Nombrado', 6, 'Gasto', true)
		ON CONFLICT (codigo_limpio) DO NOTHING`)

	// Instanciar repositorios y servicio
	repo := repository.NewConceptoModeloRepository(db)
	service := NewConceptoModeloService(repo, db)

	// A. PRUEBA 1: Carga Masiva Exitosa con UPSERT
	csvValido := `codigo_sunat,nombre_personalizado_unico_,frecuencia_meses,clasificador_codigo,es_extraordinario,requiere_monto,es_pensionable,es_remunerativa,es_base_cts,es_base_beneficios_sociales,es_ocasional,dl_276,dl_728,dl_1057,ley_30057
0121,Remuneración Principal Básica Test,"1,2,3,4,5,6,7,8,9,10,11,12",2.1.1.1.1.1,0,0,1,1,1,1,1,1,1,0,1`

	exitosos, err := service.ImportarDesdeCSV(bytes.NewBufferString(csvValido))
	if err != nil {
		t.Fatalf("Error en importación de CSV válido: %v", err)
	}
	if exitosos != 1 {
		t.Errorf("Se esperaba 1 registro exitoso, se obtuvieron %d", exitosos)
	}

	// Verificar inserción en conceptos_modelo
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM conceptos_modelo WHERE nombre_personalizado = 'Remuneración Principal Básica Test'").Scan(&count)
	if err != nil || count != 1 {
		t.Errorf("El concepto modelo no fue insertado correctamente. Conteo: %d, err: %v", count, err)
	}

	// Verificar que 'es_ocasional' se haya guardado como true
	var esOcasional bool
	err = db.QueryRow("SELECT es_ocasional FROM conceptos_modelo WHERE nombre_personalizado = 'Remuneración Principal Básica Test'").Scan(&esOcasional)
	if err != nil || !esOcasional {
		t.Errorf("No se guardó es_ocasional como true. err: %v", err)
	}

	// Verificar inserción en la tabla pivot (debe tener 3 relaciones: 276, 728 y 30057)
	var pivotCount int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM regimen_concepto_modelo rcm
		JOIN conceptos_modelo cm ON rcm.concepto_modelo_id = cm.id
		WHERE cm.nombre_personalizado = 'Remuneración Principal Básica Test'`).Scan(&pivotCount)
	if err != nil || pivotCount != 3 {
		t.Errorf("Se esperaban 3 relaciones en el pivot, se obtuvieron: %d, err: %v", pivotCount, err)
	}

	// PRUEBA DE UPSERT: Importar el mismo CSV cambiando atributos y regímenes (DL 1057 activado, DL 728 desactivado, y es_ocasional desactivado)
	csvUpsert := `codigo_sunat,nombre_personalizado_unico_,frecuencia_meses,clasificador_codigo,es_extraordinario,requiere_monto,es_pensionable,es_remunerativa,es_base_cts,es_base_beneficios_sociales,es_ocasional,dl_276,dl_728,dl_1057,ley_30057
0121,Remuneración Principal Básica Test,"1,2,3,4,5,6,7,8,9,10,11,12",2.1.1.1.1.1,1,0,1,1,1,1,0,1,0,1,1`

	exitosos, err = service.ImportarDesdeCSV(bytes.NewBufferString(csvUpsert))
	if err != nil {
		t.Fatalf("Error en UPSERT de CSV: %v", err)
	}
	if exitosos != 1 {
		t.Errorf("Se esperaba 1 registro exitoso en UPSERT, se obtuvo %d", exitosos)
	}

	// Verificar que 'es_extraordinario' ahora sea true
	var esExtraordinario bool
	err = db.QueryRow("SELECT es_extraordinario FROM conceptos_modelo WHERE nombre_personalizado = 'Remuneración Principal Básica Test'").Scan(&esExtraordinario)
	if err != nil || !esExtraordinario {
		t.Errorf("El UPSERT no actualizó es_extraordinario a true, err: %v", err)
	}

	// Verificar que 'es_ocasional' se haya actualizado a false en el UPSERT
	err = db.QueryRow("SELECT es_ocasional FROM conceptos_modelo WHERE nombre_personalizado = 'Remuneración Principal Básica Test'").Scan(&esOcasional)
	if err != nil || esOcasional {
		t.Errorf("El UPSERT no actualizó es_ocasional a false, err: %v", err)
	}

	// Verificar las nuevas relaciones en el pivot (siguen siendo 3, pero ahora DL 1057 está y DL 728 no)
	var mappings []string
	rows, err := db.Query(`
		SELECT rl.codigo FROM regimen_concepto_modelo rcm
		JOIN conceptos_modelo cm ON rcm.concepto_modelo_id = cm.id
		JOIN regimenes_laborales rl ON rcm.regimen_id = rl.id
		WHERE cm.nombre_personalizado = 'Remuneración Principal Básica Test'`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var cod string
			rows.Scan(&cod)
			mappings = append(mappings, cod)
		}
	}
	// Debe contener 276, 1057, 30057. No debe contener 728.
	has728 := false
	has1057 := false
	for _, c := range mappings {
		if c == "728" {
			has728 = true
		}
		if c == "1057" {
			has1057 = true
		}
	}
	if has728 || !has1057 {
		t.Errorf("El pivot no fue actualizado correctamente. Mapeos actuales: %v", mappings)
	}

	// B. PRUEBA 2: Transaccionalidad / Rollback en caso de clasificador inválido
	csvInvalido := `codigo_sunat,nombre_personalizado_unico_,frecuencia_meses,clasificador_codigo,es_extraordinario,requiere_monto,es_pensionable,es_remunerativa,es_base_cts,es_base_beneficios_sociales,es_ocasional,dl_276,dl_728,dl_1057,ley_30057
0121,Concepto Valido Test,"1,2,3,4,5,6,7,8,9,10,11,12",2.1.1.1.1.1,0,0,1,1,1,1,0,1,1,0,1
0121,Concepto Invalido Test,"1,2,3,4,5,6,7,8,9,10,11,12",9.9.9.9.9.9,0,0,1,1,1,1,0,1,1,0,1` // Clasificador inexistente

	_, err = service.ImportarDesdeCSV(bytes.NewBufferString(csvInvalido))
	if err == nil {
		t.Fatal("Se esperaba error por clasificador inválido, pero no ocurrió")
	}

	// Verificar transaccionalidad: el primer registro ('Concepto Valido Test') NO debe existir en la base de datos (rollback)
	err = db.QueryRow("SELECT COUNT(*) FROM conceptos_modelo WHERE nombre_personalizado = 'Concepto Valido Test'").Scan(&count)
	if err != nil || count != 0 {
		t.Errorf("Fallo en la transaccionalidad: se insertó 'Concepto Valido Test' a pesar del error posterior. Conteo: %d, err: %v", count, err)
	}

	// C. PRUEBA 3: Importación exitosa con codificación Latin-1/ISO-8859-1 (Excel en español)
	// 'ó' es 0xF3 en Latin-1
	csvLatin1 := []byte("codigo_sunat,nombre_personalizado_unico_,frecuencia_meses,clasificador_codigo,es_extraordinario,requiere_monto,es_pensionable,es_remunerativa,es_base_cts,es_base_beneficios_sociales,es_ocasional,dl_276,dl_728,dl_1057,ley_30057\n" +
		"0121,Remuneraci\xf3n de Integraci\xf3n Test,\"1,2,3,4,5,6,7,8,9,10,11,12\",2.1.1.1.1.1,0,0,1,1,1,1,0,1,1,0,1")

	exitosos, err = service.ImportarDesdeCSV(bytes.NewReader(csvLatin1))
	if err != nil {
		t.Fatalf("Error en importación de CSV en codificación Latin-1: %v", err)
	}
	if exitosos != 1 {
		t.Errorf("Se esperaba 1 registro exitoso en importación Latin-1, se obtuvieron %d", exitosos)
	}

	// Verificar que el nombre con acento 'ó' se haya guardado correctamente en UTF-8 en la base de datos
	var nombreGuardado string
	err = db.QueryRow("SELECT nombre_personalizado FROM conceptos_modelo WHERE nombre_personalizado = 'Remuneración de Integración Test'").Scan(&nombreGuardado)
	if err != nil || nombreGuardado != "Remuneración de Integración Test" {
		t.Errorf("El concepto modelo con caracteres Latin-1 no fue convertido y guardado correctamente en UTF-8. Obtenido: %s, err: %v", nombreGuardado, err)
	}
}
