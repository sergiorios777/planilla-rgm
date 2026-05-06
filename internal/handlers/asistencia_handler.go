package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"planilla-rgm/internal/models"
	"strconv"
	"strings"

	"planilla-rgm/internal/repository"

	"github.com/xuri/excelize/v2"
)

type AsistenciaHandler struct {
	Repo *repository.AsistenciaRepository
}

func (h *AsistenciaHandler) VistaUI(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r) // Usamos tu helper de sesión

	// Traemos los contratos para llenar el desplegable de búsqueda
	contratos, _ := h.Repo.ObtenerContratosParaSelect(tenantID)

	tmpl, _ := template.ParseFiles("ui/templates/tenant/asistencia_ui.html")
	tmpl.Execute(w, map[string]interface{}{
		"Contratos": contratos,
	})
}

func (h *AsistenciaHandler) Listar(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	buscar := r.URL.Query().Get("buscar")
	tipo := r.URL.Query().Get("tipo")
	procesado := r.URL.Query().Get("procesado")
	paginaStr := r.URL.Query().Get("pagina")
	limiteStr := r.URL.Query().Get("limite")

	limite, err := strconv.Atoi(limiteStr)
	if err != nil || limite <= 0 {
		limite = 10 // Por defecto mostramos 10
	}

	pagina, err := strconv.Atoi(paginaStr)
	if err != nil || pagina <= 0 {
		pagina = 1 // Por defecto empezamos en la página 1
	}

	offset := (pagina - 1) * limite

	ocurrencias, totalRegistros, err := h.Repo.ListarPaginado(tenantID, buscar, tipo, procesado, limite, offset)
	if err != nil {
		http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
		return
	}

	totalPaginas := (totalRegistros + limite - 1) / limite
	if totalPaginas == 0 {
		totalPaginas = 1
	}

	datosVista := struct {
		Ocurrencias     []models.OcurrenciaVista
		TotalPaginas    int
		PaginaActual    int
		PaginaAnterior  int
		PaginaSiguiente int
	}{
		Ocurrencias:     ocurrencias,
		TotalPaginas:    totalPaginas,
		PaginaActual:    pagina,
		PaginaAnterior:  pagina - 1,
		PaginaSiguiente: pagina + 1,
	}

	tmpl, _ := template.ParseFiles("ui/templates/tenant/asistencia_ui.html")
	tmpl.ExecuteTemplate(w, "tabla_ocurrencias", datosVista)
}

func (h *AsistenciaHandler) Crear(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	contratoID, _ := strconv.Atoi(r.FormValue("contrato_id"))
	cantidad, _ := strconv.ParseFloat(r.FormValue("cantidad"), 64)
	tipo := r.FormValue("tipo")
	fecha := r.FormValue("fecha_ocurrencia")

	// Guardamos en la base de datos
	h.Repo.Crear(contratoID, tipo, fecha, cantidad)

	// Devolvemos la tabla actualizada
	h.Listar(w, r)
}

// ImportarExcel procesa el archivo .xlsx en formato Detallado (Fila por Ocurrencia)
func (h *AsistenciaHandler) ImportarExcel(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)

	// 1. Leer el formulario multipart (máximo 10 MB)
	r.ParseMultipartForm(10 << 20)

	file, _, err := r.FormFile("archivo_excel")
	if err != nil {
		http.Error(w, "Error al leer el archivo: "+err.Error(), 400)
		return
	}
	defer file.Close()

	// 2. Abrir el Excel
	f, err := excelize.OpenReader(file)
	if err != nil {
		http.Error(w, "Error al procesar el Excel: "+err.Error(), 400)
		return
	}
	defer f.Close()

	hoja := f.GetSheetName(0)
	filas, err := f.GetRows(hoja)
	if err != nil {
		http.Error(w, "Error al leer las filas", 500)
		return
	}

	procesados := 0
	errores := 0

	// 3. Iterar sobre las filas (saltando el encabezado)
	for i, fila := range filas {
		// Necesitamos al menos 4 columnas: DNI, Fecha, Tipo, Cantidad
		if i == 0 || len(fila) < 4 {
			continue
		}

		dni := strings.TrimSpace(fila[0])
		fechaRaw := strings.TrimSpace(fila[1]) // Idealmente en formato AAAA-MM-DD
		tipoRaw := strings.ToUpper(strings.TrimSpace(fila[2]))
		cantidadRaw := strings.TrimSpace(fila[3])

		// Buscar contrato
		contratoID, err := h.Repo.ObtenerContratoPorDNI(tenantID, dni)
		if err != nil || dni == "" {
			errores++
			continue
		}

		// Normalizar el Tipo de ocurrencia
		tipoDB := "TARDANZA"
		if tipoRaw == "FALTA" || tipoRaw == "INASISTENCIA" {
			tipoDB = "INASISTENCIA"
		}

		// Validar cantidad
		cantidad, err := strconv.ParseFloat(cantidadRaw, 64)
		if err != nil || cantidad <= 0 {
			errores++
			continue
		}

		// Insertar en base de datos reutilizando tu función
		err = h.Repo.Crear(contratoID, tipoDB, fechaRaw, cantidad)
		if err != nil {
			errores++
			continue
		}

		procesados++
	}

	// 4. Mensaje de éxito
	mensaje := fmt.Sprintf(`
		<article style="background-color: #e8f5e9; color: #1b5e20; padding: 1rem; border-radius: 5px;">
			✅ Importación Detallada Completada.<br>
			<strong>%d</strong> ocurrencias registradas con éxito.<br>
			<strong>%d</strong> filas ignoradas (DNI inactivo o error de formato).
		</article>
	`, procesados, errores)
	w.Write([]byte(mensaje))
}

func (h *AsistenciaHandler) FormularioCrearUI(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	contratos, _ := h.Repo.ObtenerContratosParaSelect(tenantID)

	tmpl, _ := template.ParseFiles("ui/templates/tenant/asistencia_ui.html")
	tmpl.ExecuteTemplate(w, "formulario_crear", map[string]interface{}{
		"Contratos": contratos,
	})
}

func (h *AsistenciaHandler) EditarUI(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	tenantID := obtenerTenantID(r)

	ocurrencia, _ := h.Repo.ObtenerPorID(id, tenantID)

	// Si ya está procesada, no permitimos editar (seguridad extra)
	if ocurrencia.Procesado {
		w.Write([]byte(`<p style="color:red;">⚠️ Esta ocurrencia ya fue procesada y no puede editarse.</p>`))
		return
	}

	tmpl, _ := template.ParseFiles("ui/templates/tenant/asistencia_ui.html")
	tmpl.ExecuteTemplate(w, "formulario_editar", ocurrencia)
}

func (h *AsistenciaHandler) Actualizar(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id, _ := strconv.Atoi(r.FormValue("id"))
	tipo := r.FormValue("tipo")
	fecha := r.FormValue("fecha_ocurrencia")
	cantidad, _ := strconv.ParseFloat(r.FormValue("cantidad"), 64)
	tenantID := obtenerTenantID(r)

	h.Repo.Actualizar(id, tipo, fecha, cantidad, tenantID)

	w.Header().Set("HX-Trigger", "recargarTablaAsistencias")
	h.FormularioCrearUI(w, r)
}
