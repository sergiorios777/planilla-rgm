package handlers

import (
	"encoding/csv"
	"html/template"
	"net/http"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
	"strings"
)

type ConceptoHandler struct {
	Repo *repository.ConceptoRepository
}

func (h *ConceptoHandler) VistaUI(w http.ResponseWriter, r *http.Request) {
	tmpl, _ := template.ParseFiles("ui/templates/admin/conceptos_ui.html")
	tmpl.Execute(w, nil)
}

func (h *ConceptoHandler) ListarConceptos(w http.ResponseWriter, r *http.Request) {
	conceptos, _ := h.Repo.ObtenerTodos()
	tmpl, _ := template.ParseFiles("ui/templates/admin/conceptos_ui.html")
	// Usamos un bloque específico dentro de la plantilla para solo recargar la tabla
	tmpl.ExecuteTemplate(w, "tabla_conceptos", conceptos)
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
	h.ListarConceptos(w, r)
}
