package handlers

import (
	"encoding/csv"
	"html/template"
	"log"
	"math"
	"net/http"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
	"strconv"
	"strings"
)

type MefHandler struct {
	Repo *repository.MefRepository
}

func (h *MefHandler) ListarClasificadores(w http.ResponseWriter, r *http.Request) {
	// 1. Capturar filtros
	busqueda := r.URL.Query().Get("buscar")
	tipo := r.URL.Query().Get("tipo_clasificador")
	anioStr := r.URL.Query().Get("anio")
	anio, _ := strconv.Atoi(anioStr) // Si falla, queda en 0 y no se filtra

	// 2. Capturar y calcular paginación (con valores por defecto si están vacíos)
	limiteStr := r.URL.Query().Get("limite")
	paginaStr := r.URL.Query().Get("pagina")

	limite, err := strconv.Atoi(limiteStr)
	if err != nil || limite <= 0 {
		limite = 15 // Por defecto mostramos 10
	}

	pagina, err := strconv.Atoi(paginaStr)
	if err != nil || pagina <= 0 {
		pagina = 1 // Por defecto empezamos en la página 1
	}
	offset := (pagina - 1) * limite // Matemática del salto

	// 3. Obtener los datos con los nuevos parámetros
	clasificadores, totalRegistros, err := h.Repo.ObtenerTodos(busqueda, anio, tipo, limite, offset)
	if err != nil {
		http.Error(w, "Error al obtener clasificadores", http.StatusInternalServerError)
		return
	}

	// 4. Crear una estructura "al vuelo" para enviar datos + paginación a la vista
	// math.Ceil redondea hacia arriba, por lo que 4.1 se convierte en 5.
	totalPaginas := int(math.Ceil(float64(totalRegistros) / float64(limite)))

	// Si la búsqueda no arrojó resultados, forzamos la vista a "Página 1 de 1"
	if totalPaginas == 0 {
		totalPaginas = 1
	}

	datosVista := struct {
		Clasificadores  []models.ClasificadorMEF
		PaginaActual    int
		PaginaAnterior  int
		PaginaSiguiente int
		TotalPaginas    int
	}{
		Clasificadores:  clasificadores,
		PaginaActual:    pagina,
		PaginaAnterior:  pagina - 1,
		PaginaSiguiente: pagina + 1,
		TotalPaginas:    totalPaginas,
	}

	tmpl, err := template.ParseFiles("ui/templates/admin/clasificadores.html")
	if err != nil {
		http.Error(w, "Error cargando plantilla: "+err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, datosVista)
}

func (h *MefHandler) CrearClasificador(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	// Conversión de tipos: El formulario envía strings, necesitamos ints
	anio, _ := strconv.Atoi(r.FormValue("anio"))
	nivel, _ := strconv.Atoi(r.FormValue("nivel"))

	codigoOriginal := r.FormValue("codigo")
	// Lógica de limpieza: eliminamos todos los espacios en blanco
	codigoLimpio := strings.ReplaceAll(codigoOriginal, " ", "")

	nuevo := models.ClasificadorMEF{
		Anio:            anio,
		CodigoOriginal:  codigoOriginal,
		CodigoLimpio:    codigoLimpio,
		Descripcion:     r.FormValue("descripcion"),
		Nivel:           nivel,
		TipoTransaccion: r.FormValue("tipo_transaccion"),
		Activo:          true, // Por defecto al crearlos manualmente
	}

	err := h.Repo.Crear(&nuevo)
	if err != nil {
		http.Error(w, "Error al guardar: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", "cerrarModal")
	h.ListarClasificadores(w, r)
}

// VistaUI devuelve la estructura HTML de la página de MEF
func (h *MefHandler) VistaUI(w http.ResponseWriter, r *http.Request) {
	tmpl, _ := template.ParseFiles("ui/templates/admin/mef_ui.html")
	tmpl.Execute(w, nil)
}

// ImportarCSV lee el archivo subido, calcula los campos faltantes y lo guarda
func (h *MefHandler) ImportarCSV(w http.ResponseWriter, r *http.Request) {
	// 1. Recibimos el archivo del formulario HTMX (límite de 10 MB)
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "Error al procesar el formulario", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("archivo_csv")
	if err != nil {
		http.Error(w, "No se encontró el archivo CSV", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 2. Leemos el CSV
	reader := csv.NewReader(file)
	// Como tus datos de ejemplo separan columnas por coma, lo definimos así
	reader.Comma = ','
	reader.LazyQuotes = true // Ayuda si el Excel puso comillas raras

	records, err := reader.ReadAll()
	if err != nil {
		http.Error(w, "Error al leer el contenido del CSV", http.StatusInternalServerError)
		return
	}

	var lista []models.ClasificadorMEF

	// 3. Procesamos cada fila
	for i, row := range records {
		if i == 0 {
			continue // Saltamos la fila 0 porque son los encabezados (ANIO, CLASIFICADOR, DESCRIPCION)
		}
		if len(row) < 3 {
			continue // Fila vacía o inválida, la ignoramos
		}

		anio, _ := strconv.Atoi(row[0])
		codigoOriginal := row[1]
		descripcion := row[2]

		// --- MAGIA DE LIMPIEZA Y CÁLCULO ---
		// Reemplazamos todos los puntos por espacios, y luego separamos por espacios.
		// "1.1.1 1.2" -> "1 1 1 1 2" -> ["1", "1", "1", "1", "2"]
		partes := strings.Fields(strings.ReplaceAll(codigoOriginal, ".", " "))

		nivel := len(partes)
		// Volvemos a unir con puntos para tener un código limpio perfecto
		codigoLimpio := strings.Join(partes, ".")

		// Calculamos el Tipo de Transacción
		tipo := "Gasto"
		if len(partes) > 0 && partes[0] == "1" {
			tipo = "Ingreso"
		}
		// ------------------------------------

		lista = append(lista, models.ClasificadorMEF{
			Anio:            anio,
			CodigoOriginal:  codigoOriginal,
			CodigoLimpio:    codigoLimpio,
			Descripcion:     descripcion,
			Nivel:           nivel,
			TipoTransaccion: tipo,
			Activo:          true,
		})
	}

	// 4. Guardamos todo masivamente en la BD
	err = h.Repo.InsertarMasivo(lista)
	if err != nil {
		http.Error(w, "Error al guardar en BD: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// A. Usamos un mapa para extraer solo los años únicos (sin repetir)
	aniosUnicos := make(map[int]bool)
	for _, clasificador := range lista {
		aniosUnicos[clasificador.Anio] = true
	}

	// B. Ejecutamos el vinculador por cada año encontrado en el mapa
	for anio := range aniosUnicos {
		err = h.Repo.VincularJerarquia(anio)
		if err != nil {
			// Informamos en consola especificando en qué año falló
			log.Printf("Advertencia: No se pudo generar la jerarquía para el año %d: %v", anio, err)
		}
	}

	w.Header().Set("HX-Trigger", "cerrarModal")
	// 5. Devolvemos la tabla actualizada a HTMX
	h.ListarClasificadores(w, r)
}

// VincularJerarquiaManual ejecuta el algoritmo de vinculación mediante un botón en la UI
func (h *MefHandler) VincularJerarquiaManual(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error al leer el formulario", http.StatusBadRequest)
		return
	}

	// Leemos el año que el usuario seleccionó en la interfaz
	anio, err := strconv.Atoi(r.FormValue("anio_vincular"))
	if err != nil {
		http.Error(w, "Año inválido", http.StatusBadRequest)
		return
	}

	// Ejecutamos nuestro algoritmo de 2da pasada (el que ya habíamos escrito)
	err = h.Repo.VincularJerarquia(anio)
	if err != nil {
		http.Error(w, "Error al procesar la jerarquía: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", "cerrarModal")
	// Devolvemos la tabla actualizada
	h.ListarClasificadores(w, r)
}
