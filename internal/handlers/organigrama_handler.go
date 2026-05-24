package handlers

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

type OrganigramaHandler struct {
	Repo       *repository.OrganigramaRepository
	PuestoRepo *repository.PuestoRepository
}

// VistaUI renderiza el esqueleto base de la gestión de organigramas
func (h *OrganigramaHandler) VistaUI(w http.ResponseWriter, r *http.Request) {
	session := obtenerTenantID(r)

	organigramas, err := h.Repo.ObtenerOrganigramasPorTenant(session)
	if err != nil {
		http.Error(w, "Error al obtener organigramas", http.StatusInternalServerError)
		return
	}

	data := struct {
		Organigramas []models.Organigrama
	}{
		Organigramas: organigramas,
	}

	tmpl, err := template.ParseFiles("ui/templates/tenant/organigramas_ui.html")
	if err != nil {
		http.Error(w, "Error al cargar la plantilla: "+err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, data)
}

// ArbolUI es una llamada HTMX que retorna el arbol visual de un organigrama
func (h *OrganigramaHandler) ArbolUI(w http.ResponseWriter, r *http.Request) {
	orgIDStr := r.URL.Query().Get("organigrama_id")
	orgID, err := strconv.Atoi(orgIDStr)
	if err != nil {
		// Intentar buscar el activo
		session := obtenerTenantID(r)
		activo, err := h.Repo.ObtenerOrganigramaActivo(session)
		if err == nil && activo != nil {
			orgID = activo.ID
		} else {
			w.Write([]byte("<p class='error'>No hay un organigrama activo configurado. Por favor, cree o active una versión.</p>"))
			return
		}
	}

	arbol, err := h.Repo.ObtenerArbolNodos(orgID)
	if err != nil {
		http.Error(w, "Error al cargar el árbol", http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFiles("ui/templates/tenant/organigramas_ui.html")
	if err != nil {
		http.Error(w, "Error de plantilla", http.StatusInternalServerError)
		return
	}

	// Ejecuta únicamente el bloque parcial de árbol
	tmpl.ExecuteTemplate(w, "arbol_parcial", arbol)
}

// ClonarVersion duplica el organigrama y re-ubica los puestos físicamente
func (h *OrganigramaHandler) ClonarVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	session := obtenerTenantID(r)

	origenIDStr := r.FormValue("origen_id")
	origenID, _ := strconv.Atoi(origenIDStr)
	documento := r.FormValue("documento_aprobacion")
	descripcion := r.FormValue("descripcion")
	fechaVigenciaStr := r.FormValue("fecha_vigencia")

	fechaVigencia, err := time.Parse("2006-01-02", fechaVigenciaStr)
	if err != nil {
		http.Error(w, "Fecha de vigencia inválida", http.StatusBadRequest)
		return
	}

	// 1. Crear nuevo organigrama vacío
	nuevoOrg := models.Organigrama{
		TenantID:            session,
		DocumentoAprobacion: documento,
		Descripcion:         descripcion,
		FechaVigencia:       fechaVigencia,
		Activo:              false,
	}

	err = h.Repo.CrearOrganigrama(&nuevoOrg)
	if err != nil {
		http.Error(w, "Error al crear nueva versión: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 2. Clonar y mover puestos
	err = h.Repo.ClonarEstructuraYTrasladarPuestos(session, origenID, nuevoOrg.ID)
	if err != nil {
		http.Error(w, "Error en clonación: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Responder con redirección o recarga total de la interfaz del tenant
	w.Header().Set("HX-Refresh", "true")
	w.WriteHeader(http.StatusOK)
}

// AgregarHijoUI renderiza la modal para añadir subunidad
func (h *OrganigramaHandler) AgregarHijoUI(w http.ResponseWriter, r *http.Request) {
	parentIDStr := r.URL.Query().Get("parent_id")
	parentID, _ := strconv.Atoi(parentIDStr)
	orgIDStr := r.URL.Query().Get("organigrama_id")
	orgID, _ := strconv.Atoi(orgIDStr)

	var p *models.UnidadOrganica
	var err error
	if parentID > 0 {
		p, err = h.Repo.ObtenerUnidadPorID(parentID)
		if err != nil {
			http.Error(w, "No se encontró el nodo padre", http.StatusNotFound)
			return
		}
	}

	data := map[string]interface{}{
		"ParentID":      parentID,
		"Parent":        p,
		"OrganigramaID": orgID,
	}

	tmpl, _ := template.ParseFiles("ui/templates/tenant/organigramas_ui.html")
	tmpl.ExecuteTemplate(w, "modal_unidad_crear", data)
}

// EditarUnidadUI renderiza la modal para editar la subunidad
func (h *OrganigramaHandler) EditarUnidadUI(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, _ := strconv.Atoi(idStr)

	u, err := h.Repo.ObtenerUnidadPorID(id)
	if err != nil {
		http.Error(w, "Unidad no encontrada", http.StatusNotFound)
		return
	}

	tmpl, _ := template.ParseFiles("ui/templates/tenant/organigramas_ui.html")
	tmpl.ExecuteTemplate(w, "modal_unidad_editar", u)
}

// GuardarUnidad procesa el guardado/inserción de una unidad orgánica
func (h *OrganigramaHandler) GuardarUnidad(w http.ResponseWriter, r *http.Request) {
	session := obtenerTenantID(r)

	orgIDStr := r.FormValue("organigrama_id")
	orgID, _ := strconv.Atoi(orgIDStr)
	parentIDStr := r.FormValue("parent_id")
	nombre := r.FormValue("nombre")
	tipo := r.FormValue("tipo")
	codigo := r.FormValue("codigo_mef")
	idStr := r.FormValue("id")

	var parentID *int
	if parentIDStr != "" && parentIDStr != "0" {
		pid, _ := strconv.Atoi(parentIDStr)
		parentID = &pid
	}

	if idStr != "" {
		// Editar
		id, _ := strconv.Atoi(idStr)
		u := models.UnidadOrganica{
			ID:        id,
			TenantID:  session,
			ParentID:  parentID,
			Nombre:    nombre,
			Tipo:      tipo,
			CodigoMef: codigo,
		}
		if err := h.Repo.ActualizarUnidad(&u); err != nil {
			log.Printf("[ERROR] ActualizarUnidad falló: %v (tenant_id=%d, organigrama_id=%d, nombre=%s)", err, session, orgID, nombre)
			http.Error(w, "Error al actualizar la unidad: "+err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		// Crear
		u := models.UnidadOrganica{
			TenantID:      session,
			OrganigramaID: orgID,
			ParentID:      parentID,
			Nombre:        nombre,
			Tipo:          tipo,
			CodigoMef:     codigo,
		}
		if err := h.Repo.CrearUnidad(&u); err != nil {
			log.Printf("[ERROR] CrearUnidad falló: %v (tenant_id=%d, organigrama_id=%d, nombre=%s)", err, session, orgID, nombre)
			http.Error(w, "Error al crear la unidad: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Trigger reload
	w.Header().Set("HX-Trigger", "reloadArbol")
	w.WriteHeader(http.StatusOK)
}

// EliminarUnidad elimina el nodo orgánico
func (h *OrganigramaHandler) EliminarUnidad(w http.ResponseWriter, r *http.Request) {
	session := obtenerTenantID(r)
	idStr := r.URL.Query().Get("id")
	id, _ := strconv.Atoi(idStr)

	if err := h.Repo.EliminarUnidad(id, session); err != nil {
		log.Printf("[ERROR] EliminarUnidad falló: %v (id=%d, tenant_id=%d)", err, id, session)
		http.Error(w, "Error al eliminar la unidad: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("HX-Trigger", "reloadArbol")
	w.WriteHeader(http.StatusOK)
}

// DescargarPlantilla genera y envía un archivo Excel base para la importación
func (h *OrganigramaHandler) DescargarPlantilla(w http.ResponseWriter, r *http.Request) {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Estructura_Organica"
	f.SetSheetName("Sheet1", sheet)

	// Cabeceras
	cabeceras := []string{"ID_TEMPORAL", "NOMBRE", "TIPO", "ID_PADRE", "CODIGO_MEF"}
	for i, cabecera := range cabeceras {
		col := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue(sheet, col, cabecera)
	}

	// Datos de ejemplo
	ejemplos := [][]interface{}{
		{1, "Concejo Municipal", "Alta_Dirección", "", "300000"},
		{2, "Alcaldía", "Alta_Dirección", "", "300001"},
		{3, "Gerencia Municipal", "Gerencia", 2, "300002"},
		{4, "Subgerencia de Recursos Humanos", "Subgerencia", 3, "300003"},
	}
	for rIdx, fila := range ejemplos {
		for cIdx, valor := range fila {
			col := fmt.Sprintf("%c%d", 'A'+cIdx, rIdx+2)
			f.SetCellValue(sheet, col, valor)
		}
	}

	// Hoja de instrucciones
	instruccionesSheet := "Instrucciones"
	f.NewSheet(instruccionesSheet)
	f.SetCellValue(instruccionesSheet, "A1", "INSTRUCCIONES DE LLENADO")
	f.SetCellValue(instruccionesSheet, "A3", "ID_TEMPORAL: Número único en esta tabla (ej. 1, 2, 3...)")
	f.SetCellValue(instruccionesSheet, "A4", "NOMBRE: Nombre de la unidad (ej. Gerencia de Administración)")
	f.SetCellValue(instruccionesSheet, "A5", "TIPO: Debe ser uno exacto de: Alta_Dirección, Gerencia, Subgerencia, Oficina, Unidad, Dirección, Departamento")
	f.SetCellValue(instruccionesSheet, "A6", "ID_PADRE: El ID_TEMPORAL de la unidad de la que depende. Dejar vacío si no depende de nadie.")
	f.SetCellValue(instruccionesSheet, "A7", "CODIGO_MEF: Opcional. Código del Ministerio de Economía y Finanzas.")

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=plantilla_organigrama.xlsx")

	if err := f.Write(w); err != nil {
		log.Printf("Error al generar plantilla Excel: %v", err)
	}
}

// ImportarExcel lee el archivo subido y llama al repositorio
func (h *OrganigramaHandler) ImportarExcel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	session := obtenerTenantID(r)
	orgIDStr := r.FormValue("organigrama_id")
	orgID, err := strconv.Atoi(orgIDStr)
	if err != nil {
		http.Error(w, "ID de organigrama inválido", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("archivo_excel")
	if err != nil {
		http.Error(w, "Debe seleccionar un archivo válido", http.StatusBadRequest)
		return
	}
	defer file.Close()

	f, err := excelize.OpenReader(file)
	if err != nil {
		http.Error(w, "Error al leer el archivo Excel", http.StatusBadRequest)
		return
	}
	defer f.Close()

	sheet := f.GetSheetName(0)
	rows, err := f.GetRows(sheet)
	if err != nil || len(rows) < 2 {
		http.Error(w, "El archivo está vacío o no tiene la estructura correcta", http.StatusBadRequest)
		return
	}

	var unidades []repository.UnidadImport

	// Validar y mapear filas
	for i, row := range rows {
		if i == 0 { // Saltar cabecera
			continue
		}

		// Normalizar longitud de la fila a 5 columnas
		for len(row) < 5 {
			row = append(row, "")
		}

		idTempStr := strings.TrimSpace(row[0])
		nombre := strings.TrimSpace(row[1])
		tipo := strings.TrimSpace(row[2])
		idPadreStr := strings.TrimSpace(row[3])
		codigoMef := strings.TrimSpace(row[4])

		if idTempStr == "" || nombre == "" || tipo == "" {
			http.Error(w, fmt.Sprintf("Fila %d: ID_TEMPORAL, NOMBRE y TIPO son obligatorios", i+1), http.StatusBadRequest)
			return
		}

		idTemp, err := strconv.Atoi(idTempStr)
		if err != nil {
			http.Error(w, fmt.Sprintf("Fila %d: ID_TEMPORAL debe ser un número entero", i+1), http.StatusBadRequest)
			return
		}

		var idPadre *int
		if idPadreStr != "" {
			padreTemp, err := strconv.Atoi(idPadreStr)
			if err != nil {
				http.Error(w, fmt.Sprintf("Fila %d: ID_PADRE debe ser un número entero o estar vacío", i+1), http.StatusBadRequest)
				return
			}
			idPadre = &padreTemp
		}

		unidades = append(unidades, repository.UnidadImport{
			IDTemporal: idTemp,
			Nombre:     nombre,
			Tipo:       tipo,
			IDPadre:    idPadre,
			CodigoMef:  codigoMef,
		})
	}

	if err := h.Repo.ImportarUnidadesTransaccional(session, orgID, unidades); err != nil {
		log.Printf("[ERROR] ImportarExcel falló: %v (tenant_id=%d, org_id=%d)", err, session, orgID)
		http.Error(w, `<div class="error" style="color:red; padding:1rem; border:1px solid red;">Error en importación: `+err.Error()+`</div>`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", "reloadArbol")
	w.WriteHeader(http.StatusOK)
}
