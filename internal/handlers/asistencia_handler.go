package handlers

import (
	"fmt"
	"html/template"
	"net/http"
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
	ocurrencias, _ := h.Repo.ListarHistorial(tenantID)

	tmpl, _ := template.ParseFiles("ui/templates/tenant/asistencia_ui.html")
	tmpl.ExecuteTemplate(w, "tabla_ocurrencias", ocurrencias)
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
