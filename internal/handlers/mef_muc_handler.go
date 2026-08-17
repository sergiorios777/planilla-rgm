package handlers

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"html/template"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
)

type MefMucHandler struct {
	Repo *repository.MefMucRepository
}

func NewMefMucHandler(repo *repository.MefMucRepository) *MefMucHandler {
	return &MefMucHandler{Repo: repo}
}

// VistaUI renderiza la página principal del módulo de valores MUC
func (h *MefMucHandler) VistaUI(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles(
		"ui/templates/admin/mef_muc_ui.html",
		"ui/templates/components/paginacion.html",
	)
	if err != nil {
		http.Error(w, "Error al cargar plantilla MUC: "+err.Error(), http.StatusInternalServerError)
		return
	}

	normas, _ := h.Repo.ObtenerNormasLegales()

	filtros := models.MefMucFiltros{
		Pagina: 1,
		Limite: 15,
	}

	valores, total, err := h.Repo.ListarPaginado(filtros)
	if err != nil {
		http.Error(w, "Error al obtener registros de MUC: "+err.Error(), http.StatusInternalServerError)
		return
	}

	totalPaginas := int(math.Ceil(float64(total) / float64(filtros.Limite)))
	paginacion := models.CalcularPaginacion(
		filtros.Pagina,
		totalPaginas,
		total,
		"/admin/mef-muc/lista",
		"#contenedor-tabla-mef-muc",
		"#form-filtros-mef-muc",
	)

	data := models.MefMucRespuestaDTO{
		Valores:       valores,
		Paginacion:    paginacion,
		Filtros:       filtros,
		NormasLegales: normas,
	}

	tmpl.Execute(w, data)
}

// Listar atiende peticiones HTMX para filtrar y paginar la tabla MUC
func (h *MefMucHandler) Listar(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	pagina, _ := strconv.Atoi(r.FormValue("pagina"))
	if pagina <= 0 {
		pagina = 1
	}

	filtros := models.MefMucFiltros{
		NormaLegal: r.FormValue("norma_legal"),
		FechaNorma: r.FormValue("fecha_norma"),
		Activo:     r.FormValue("activo"),
		Buscar:     r.FormValue("buscar"),
		Pagina:     pagina,
		Limite:     15,
	}

	valores, total, err := h.Repo.ListarPaginado(filtros)
	if err != nil {
		http.Error(w, "Error al obtener registros: "+err.Error(), http.StatusInternalServerError)
		return
	}

	normas, _ := h.Repo.ObtenerNormasLegales()

	totalPaginas := int(math.Ceil(float64(total) / float64(filtros.Limite)))
	paginacion := models.CalcularPaginacion(
		filtros.Pagina,
		totalPaginas,
		total,
		"/admin/mef-muc/lista",
		"#contenedor-tabla-mef-muc",
		"#form-filtros-mef-muc",
	)

	data := models.MefMucRespuestaDTO{
		Valores:       valores,
		Paginacion:    paginacion,
		Filtros:       filtros,
		NormasLegales: normas,
	}

	tmpl, err := template.ParseFiles(
		"ui/templates/admin/mef_muc_ui.html",
		"ui/templates/components/paginacion.html",
	)
	if err != nil {
		http.Error(w, "Error al cargar plantilla: "+err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.ExecuteTemplate(w, "tabla_mef_muc", data)
}

// Crear guarda un nuevo valor histórico MUC
func (h *MefMucHandler) Crear(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	normaLegal := strings.TrimSpace(r.FormValue("norma_legal"))
	fechaNormaStr := strings.TrimSpace(r.FormValue("fecha_norma"))
	grupoOcupacional := strings.TrimSpace(r.FormValue("grupo_ocupacional"))
	nivelRemunerativo := strings.TrimSpace(r.FormValue("nivel_remunerativo"))
	montoStr := strings.TrimSpace(r.FormValue("monto_muc"))
	activo := r.FormValue("activo") == "on" || r.FormValue("activo") == "true" || r.FormValue("activo") == "1"

	if normaLegal == "" || fechaNormaStr == "" || grupoOcupacional == "" || nivelRemunerativo == "" || montoStr == "" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<div class="alert alert-danger mb-md">⚠️ Todos los campos requeridos deben ser completados.</div>`))
		return
	}

	fechaNorma, err := time.Parse("2006-01-02", fechaNormaStr)
	if err != nil {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<div class="alert alert-danger mb-md">⚠️ La fecha ingresada no tiene un formato válido (AAAA-MM-DD).</div>`))
		return
	}

	montoClean := strings.ReplaceAll(strings.ReplaceAll(montoStr, "S/", ""), ",", "")
	monto, err := strconv.ParseFloat(strings.TrimSpace(montoClean), 64)
	if err != nil || monto < 0 {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<div class="alert alert-danger mb-md">⚠️ El monto MUC debe ser un número válido mayor o igual a 0.00.</div>`))
		return
	}

	v := models.MefMucValor{
		NormaLegal:        normaLegal,
		FechaNorma:        fechaNorma,
		GrupoOcupacional:  grupoOcupacional,
		NivelRemunerativo: nivelRemunerativo,
		MontoMuc:          monto,
		Activo:            activo,
	}

	err = h.Repo.Crear(&v)
	if err != nil {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<div class="alert alert-danger mb-md">❌ Error al guardar en la BD: ` + err.Error() + `</div>`))
		return
	}

	w.Header().Set("HX-Trigger", "cerrarModalCrearMuc, recargarTablaMuc")
	h.Listar(w, r)
}

// EditarForm devuelve el contenido HTML para el modal de edición de un registro MUC
func (h *MefMucHandler) EditarForm(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	if id <= 0 {
		http.Error(w, "ID no válido", http.StatusBadRequest)
		return
	}

	valor, err := h.Repo.ObtenerPorID(id)
	if err != nil {
		http.Error(w, "Registro no encontrado: "+err.Error(), http.StatusNotFound)
		return
	}

	tmpl, err := template.ParseFiles("ui/templates/admin/mef_muc_ui.html")
	if err != nil {
		http.Error(w, "Error al cargar plantilla: "+err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.ExecuteTemplate(w, "formulario_editar_muc", valor)
}

// Actualizar edita los datos de un registro MUC
func (h *MefMucHandler) Actualizar(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	id, _ := strconv.Atoi(r.FormValue("id"))
	if id <= 0 {
		http.Error(w, "ID no válido", http.StatusBadRequest)
		return
	}

	normaLegal := strings.TrimSpace(r.FormValue("norma_legal"))
	fechaNormaStr := strings.TrimSpace(r.FormValue("fecha_norma"))
	grupoOcupacional := strings.TrimSpace(r.FormValue("grupo_ocupacional"))
	nivelRemunerativo := strings.TrimSpace(r.FormValue("nivel_remunerativo"))
	montoStr := strings.TrimSpace(r.FormValue("monto_muc"))
	activo := r.FormValue("activo") == "on" || r.FormValue("activo") == "true" || r.FormValue("activo") == "1"

	fechaNorma, err := time.Parse("2006-01-02", fechaNormaStr)
	if err != nil {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<div class="alert alert-danger mb-md">⚠️ La fecha de norma no es válida.</div>`))
		return
	}

	montoClean := strings.ReplaceAll(strings.ReplaceAll(montoStr, "S/", ""), ",", "")
	monto, err := strconv.ParseFloat(strings.TrimSpace(montoClean), 64)
	if err != nil || monto < 0 {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<div class="alert alert-danger mb-md">⚠️ El monto MUC debe ser un valor numérico válido.</div>`))
		return
	}

	v := models.MefMucValor{
		ID:                id,
		NormaLegal:        normaLegal,
		FechaNorma:        fechaNorma,
		GrupoOcupacional:  grupoOcupacional,
		NivelRemunerativo: nivelRemunerativo,
		MontoMuc:          monto,
		Activo:            activo,
	}

	err = h.Repo.Actualizar(&v)
	if err != nil {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<div class="alert alert-danger mb-md">❌ Error al actualizar el registro: ` + err.Error() + `</div>`))
		return
	}

	w.Header().Set("HX-Trigger", "cerrarModalEditarMuc, recargarTablaMuc")
	h.Listar(w, r)
}

// ToggleEstado conmuta la condición de activo / inactivo de un registro MUC
func (h *MefMucHandler) ToggleEstado(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	activoStr := r.URL.Query().Get("activo")
	activo := activoStr == "true" || activoStr == "1"

	if id > 0 {
		_ = h.Repo.CambiarEstado(id, activo)
	}

	h.Listar(w, r)
}

// ImportarCSV procesa un archivo CSV subido con registros MUC
func (h *MefMucHandler) ImportarCSV(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20) // 10MB máximo
	if err != nil {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<div class="alert alert-danger mb-md">❌ Error al procesar la subida del archivo.</div>`))
		return
	}

	file, _, err := r.FormFile("archivo_csv")
	if err != nil {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<div class="alert alert-danger mb-md">⚠️ Por favor seleccione un archivo CSV válido.</div>`))
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	// Leer encabezado
	headers, err := reader.Read()
	if err != nil {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<div class="alert alert-danger mb-md">❌ El archivo CSV está vacío o corrupto.</div>`))
		return
	}

	// Mapeo de columnas
	idxNorma := -1
	idxFecha := -1
	idxGrupo := -1
	idxNivel := -1
	idxMonto := -1
	idxActivo := -1

	for i, hName := range headers {
		cleanHeader := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(hName, "\ufeff")))
		switch {
		case strings.Contains(cleanHeader, "fecha"):
			idxFecha = i
		case strings.Contains(cleanHeader, "norma"):
			idxNorma = i
		case strings.Contains(cleanHeader, "grupo"):
			idxGrupo = i
		case strings.Contains(cleanHeader, "nivel"):
			idxNivel = i
		case strings.Contains(cleanHeader, "monto") || strings.Contains(cleanHeader, "muc"):
			idxMonto = i
		case strings.Contains(cleanHeader, "activo") || strings.Contains(cleanHeader, "estado"):
			idxActivo = i
		}
	}

	// Fallback por orden de columna si algún campo obligatorio no fue reconocido en los encabezados
	if idxNorma < 0 && len(headers) > 0 {
		idxNorma = 0
	}
	if idxFecha < 0 && len(headers) > 1 {
		idxFecha = 1
	}
	if idxGrupo < 0 && len(headers) > 2 {
		idxGrupo = 2
	}
	if idxNivel < 0 && len(headers) > 3 {
		idxNivel = 3
	}
	if idxMonto < 0 && len(headers) > 4 {
		idxMonto = 4
	}
	if idxActivo < 0 && len(headers) > 5 {
		idxActivo = 5
	}

	// Verificar que todos los campos requeridos tengan un índice válido
	if idxNorma < 0 || idxFecha < 0 || idxGrupo < 0 || idxNivel < 0 || idxMonto < 0 {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<div class="alert alert-danger mb-md">❌ El archivo CSV no contiene las columnas requeridas (norma_legal, fecha_norma, grupo_ocupacional, nivel_remunerativo, monto_muc).</div>`))
		return
	}

	maxRequiredIdx := idxNorma
	for _, idx := range []int{idxFecha, idxGrupo, idxNivel, idxMonto} {
		if idx > maxRequiredIdx {
			maxRequiredIdx = idx
		}
	}

	var lote []models.MefMucValor
	linea := 1
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			linea++
			continue
		}
		linea++

		if len(record) <= maxRequiredIdx {
			continue
		}

		norma := strings.TrimSpace(strings.TrimPrefix(record[idxNorma], "\ufeff"))
		fechaStr := strings.TrimSpace(record[idxFecha])
		grupo := strings.ToUpper(strings.TrimSpace(record[idxGrupo]))
		nivel := strings.ToUpper(strings.TrimSpace(record[idxNivel]))
		montoStr := strings.TrimSpace(record[idxMonto])

		if norma == "" || fechaStr == "" || grupo == "" || nivel == "" || montoStr == "" {
			continue
		}

		// Parsear Fecha (probar AAAA-MM-DD y DD/MM/AAAA)
		var fecha time.Time
		if f, err := time.Parse("2006-01-02", fechaStr); err == nil {
			fecha = f
		} else if f, err := time.Parse("02/01/2006", fechaStr); err == nil {
			fecha = f
		} else {
			continue
		}

		// Parsear Monto
		montoClean := strings.ReplaceAll(strings.ReplaceAll(montoStr, "S/", ""), ",", "")
		monto, err := strconv.ParseFloat(strings.TrimSpace(montoClean), 64)
		if err != nil {
			continue
		}

		esActivo := true
		if idxActivo >= 0 && idxActivo < len(record) {
			valAct := strings.ToLower(strings.TrimSpace(record[idxActivo]))
			if valAct == "false" || valAct == "0" || valAct == "inactivo" || valAct == "no" {
				esActivo = false
			}
		}

		lote = append(lote, models.MefMucValor{
			NormaLegal:        norma,
			FechaNorma:        fecha,
			GrupoOcupacional:  grupo,
			NivelRemunerativo: nivel,
			MontoMuc:          monto,
			Activo:            esActivo,
		})
	}

	if len(lote) == 0 {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<div class="alert alert-warning mb-md">⚠️ No se encontraron filas válidas para importar. Revisa el formato del archivo CSV.</div>`))
		return
	}

	insertados, err := h.Repo.ImportarCSVBulk(lote)
	if err != nil {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<div class="alert alert-danger mb-md">❌ Error al realizar la importación masiva: ` + err.Error() + `</div>`))
		return
	}

	w.Header().Set("HX-Trigger", "cerrarModalImportarMuc, recargarTablaMuc")
	w.WriteHeader(http.StatusOK)
	w.Write(fmt.Appendf(nil, `<div class="alert alert-success mb-md">✅ Importación exitosa: Se registraron %d valores de MUC.</div>`, insertados))
}

// DescargarPlantillaCSV genera y envía al navegador un archivo CSV de ejemplo
func (h *MefMucHandler) DescargarPlantillaCSV(w http.ResponseWriter, r *http.Request) {
	buf := new(bytes.Buffer)
	writer := csv.NewWriter(buf)

	// Escribir encabezados
	_ = writer.Write([]string{"norma_legal", "fecha_norma", "grupo_ocupacional", "nivel_remunerativo", "monto_muc", "activo"})
	// Filas de ejemplo
	_ = writer.Write([]string{"D.U. N° 038-2019", "2019-12-31", "FUNCIONARIO", "F-1", "1250.00", "true"})
	_ = writer.Write([]string{"D.U. N° 038-2019", "2019-12-31", "PROFESIONAL", "SPA", "950.00", "true"})
	_ = writer.Write([]string{"D.U. N° 038-2019", "2019-12-31", "TECNICO", "STA", "750.00", "true"})
	_ = writer.Write([]string{"D.U. N° 038-2019", "2019-12-31", "AUXILIAR", "SAA", "650.00", "true"})
	writer.Flush()

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\"plantilla_valores_muc_mef.csv\"")
	w.Write(buf.Bytes())
}
