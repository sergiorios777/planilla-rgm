package handlers

import (
	"encoding/csv"
	"html/template"
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

func (h *ConceptoHandler) ImportarCSV(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(10 << 20)
	file, _, err := r.FormFile("archivo_csv")
	if err != nil {
		http.Error(w, "Archivo no encontrado", http.StatusBadRequest)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = ','
	reader.LazyQuotes = true
	records, _ := reader.ReadAll()

	var lista []models.ConceptoMaestro
	afectaciones := make(map[string][]string)

	for i, row := range records {
		if i == 0 || len(row) < 3 {
			continue // Saltamos encabezados o filas vacías
		}

		codigo := strings.TrimSpace(row[0])
		descripcion := strings.TrimSpace(row[1])
		tipo := strings.TrimSpace(row[2])

		lista = append(lista, models.ConceptoMaestro{
			Codigo:      codigo,
			Descripcion: descripcion,
			Tipo:        tipo,
			Activo:      true,
		})

		// Si existe la 4ta columna y tiene datos, la procesamos
		if len(row) >= 4 && strings.TrimSpace(row[3]) != "" {
			codigosDerivados := strings.Split(row[3], ",")
			afectaciones[codigo] = codigosDerivados
		}
	}

	h.Repo.ProcesarImportacion(lista, afectaciones)
	w.Header().Set("HX-Trigger", "cerrarModal")
	h.ListarConceptos(w, r)
}
