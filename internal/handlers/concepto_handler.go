package handlers

import (
	"bytes"
	"encoding/csv"
	"html/template"
	"io"
	"math"
	"net/http"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
	"strconv"
	"strings"
)

type ConceptoHandler struct {
	Repo *repository.ConceptoRepository
}

func (h *ConceptoHandler) VistaUI(w http.ResponseWriter, r *http.Request) {
	padres, err := h.Repo.ObtenerPadres()
	if err != nil {
		http.Error(w, "Error al obtener conceptos padres", http.StatusInternalServerError)
		return
	}

	datos := struct {
		Padres []models.ConceptoMaestro
	}{
		Padres: padres,
	}

	tmpl, err := template.ParseFiles("ui/templates/admin/conceptos_ui.html")
	if err != nil {
		http.Error(w, "Error al parsear plantilla", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, datos)
}

func (h *ConceptoHandler) ListarConceptos(w http.ResponseWriter, r *http.Request) {
	busqueda := r.URL.Query().Get("buscar")
	tipo := r.URL.Query().Get("tipo")
	parentIDStr := r.URL.Query().Get("parent_id")
	limiteStr := r.URL.Query().Get("limite")
	paginaStr := r.URL.Query().Get("pagina")

	limite, err := strconv.Atoi(limiteStr)
	if err != nil || limite <= 0 {
		limite = 15 // Por defecto mostramos 15
	}

	pagina, err := strconv.Atoi(paginaStr)
	if err != nil || pagina <= 0 {
		pagina = 1 // Por defecto empezamos en la página 1
	}
	offset := (pagina - 1) * limite

	conceptos, totalRegistros, err := h.Repo.ObtenerTodos(busqueda, tipo, parentIDStr, limite, offset)
	if err != nil {
		http.Error(w, "Error al listar conceptos", http.StatusInternalServerError)
		return
	}

	totalPaginas := int(math.Ceil(float64(totalRegistros) / float64(limite)))

	if totalPaginas == 0 {
		totalPaginas = 1
	}

	datos := struct {
		Conceptos       []models.ConceptoMaestro
		TotalPaginas    int
		PaginaActual    int
		PaginaAnterior  int
		PaginaSiguiente int
	}{
		Conceptos:       conceptos,
		TotalPaginas:    totalPaginas,
		PaginaActual:    pagina,
		PaginaAnterior:  pagina - 1,
		PaginaSiguiente: pagina + 1,
	}

	tmpl, err := template.ParseFiles("ui/templates/admin/conceptos_ui.html")
	if err != nil {
		http.Error(w, "Error al parsear plantilla", http.StatusInternalServerError)
		return
	}

	tmpl.ExecuteTemplate(w, "tabla_conceptos", datos)
}

// DescargarPlantillaCSV entrega la plantilla CSV de ejemplo para conceptos maestros SUNAT
func (h *ConceptoHandler) DescargarPlantillaCSV(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="plantilla_conceptos_sunat.csv"`)

	contenido := "CODIGO,DESCRIPCION,TIPO,ORIGEN,AFECTA_A\n" +
		"0121,REMUNERACION ESPECIFICA,Ingreso,sunat,\n" +
		"0105,HORAS EXTRAS 25%,Ingreso,sunat,\"0701,0804\"\n" +
		"1001,ASIGNACION POR CUMPLIR 25 ANOS DE SERVICIOS,Ingreso,sunat,\n" +
		"2101,BONO DE PRODUCTIVIDAD SECTOR PUBLICO,Ingreso,sunat,\n" +
		"0601,TARDANZAS,Retencion,sunat,\n" +
		"0701,SNP - ONP,Aporte,sunat,\n" +
		"9901,BONO INTERNO MUNICIPAL,Ingreso,interno,\n"

	w.Write([]byte(contenido))
}

func (h *ConceptoHandler) ImportarCSV(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(10 << 20)
	file, _, err := r.FormFile("archivo_csv")
	if err != nil {
		http.Error(w, "Archivo no encontrado", http.StatusBadRequest)
		return
	}
	defer file.Close()

	bodyBytes, err := io.ReadAll(file)
	if err != nil || len(bodyBytes) == 0 {
		http.Error(w, "El archivo CSV está vacío o no se puede leer", http.StatusBadRequest)
		return
	}

	// Auto-detección de delimitador (, o ;)
	comma := rune(',')
	firstLine := ""
	if idx := bytes.IndexByte(bodyBytes, '\n'); idx != -1 {
		firstLine = string(bodyBytes[:idx])
	} else {
		firstLine = string(bodyBytes)
	}
	if strings.Count(firstLine, ";") > strings.Count(firstLine, ",") {
		comma = ';'
	}

	reader := csv.NewReader(bytes.NewReader(bodyBytes))
	reader.Comma = comma
	reader.LazyQuotes = true
	records, err := reader.ReadAll()
	if err != nil || len(records) == 0 {
		http.Error(w, "Error al procesar las líneas del archivo CSV", http.StatusBadRequest)
		return
	}

	// Mapeo dinámico de cabeceras
	idxCodigo := 0
	idxDescripcion := 1
	idxTipo := 2
	idxOrigen := -1
	idxAfecta := -1
	hasHeaders := false

	headerRow := records[0]
	for i, col := range headerRow {
		cleanCol := strings.ToUpper(strings.TrimSpace(col))
		switch cleanCol {
		case "CODIGO", "CÓDIGO":
			idxCodigo = i
			hasHeaders = true
		case "DESCRIPCION", "DESCRIPCIÓN":
			idxDescripcion = i
			hasHeaders = true
		case "TIPO":
			idxTipo = i
			hasHeaders = true
		case "ORIGEN":
			idxOrigen = i
			hasHeaders = true
		case "AFECTA_A", "AFECTACION", "AFECTACIONES":
			idxAfecta = i
			hasHeaders = true
		}
	}

	// Fallback para archivos sin encabezados explícitos reconocidos
	if !hasHeaders {
		idxCodigo = 0
		idxDescripcion = 1
		idxTipo = 2
		if len(headerRow) == 4 {
			val3 := strings.ToLower(strings.TrimSpace(headerRow[3]))
			if val3 == "sunat" || val3 == "interno" {
				idxOrigen = 3
			} else {
				idxAfecta = 3
			}
		} else if len(headerRow) >= 5 {
			idxOrigen = 3
			idxAfecta = 4
		}
	}

	startRow := 0
	if hasHeaders {
		startRow = 1
	}

	var lista []models.ConceptoMaestro
	afectaciones := make(map[string][]string)

	for i := startRow; i < len(records); i++ {
		row := records[i]
		if len(row) <= idxCodigo || len(row) <= idxDescripcion || len(row) <= idxTipo {
			continue // Saltamos filas incompletas
		}

		codigo := strings.TrimSpace(row[idxCodigo])
		descripcion := strings.TrimSpace(row[idxDescripcion])
		tipo := strings.TrimSpace(row[idxTipo])

		if codigo == "" || descripcion == "" {
			continue
		}

		origen := "sunat"
		if idxOrigen != -1 && idxOrigen < len(row) {
			valOrigen := strings.ToLower(strings.TrimSpace(row[idxOrigen]))
			if valOrigen == "interno" || valOrigen == "sunat" {
				origen = valOrigen
			}
		}

		lista = append(lista, models.ConceptoMaestro{
			Codigo:        codigo,
			CodigoInterno: codigo,
			Descripcion:   descripcion,
			Tipo:          tipo,
			Activo:        true,
			Origen:        origen,
		})

		if idxAfecta != -1 && idxAfecta < len(row) && strings.TrimSpace(row[idxAfecta]) != "" {
			codigosDerivados := strings.Split(row[idxAfecta], ",")
			afectaciones[codigo] = codigosDerivados
		}
	}

	h.Repo.ProcesarImportacion(lista, afectaciones)
	w.Header().Set("HX-Trigger", "cerrarModal")
	h.ListarConceptos(w, r)
}

