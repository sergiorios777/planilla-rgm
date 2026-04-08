package handlers

import (
	"html/template"
	"net/http"
	"planilla-rgm/internal/models"
	"planilla-rgm/internal/repository"
	"strconv"
	"strings"
)

type ContratoHandler struct {
	Repo           *repository.ContratoRepository
	TrabajadorRepo *repository.TrabajadorRepository // Lo necesitamos para el select
}

func (h *ContratoHandler) VistaUI(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)

	// Preparamos los datos para los menús desplegables
	trabajadores, _ := h.TrabajadorRepo.ObtenerTodos(tenantID)
	regimenes, _ := h.Repo.ObtenerRegimenes()

	datos := map[string]interface{}{
		"Trabajadores": trabajadores,
		"Regimenes":    regimenes,
	}

	tmpl, _ := template.ParseFiles("ui/templates/tenant/contratos_ui.html")
	tmpl.Execute(w, datos)
}

func (h *ContratoHandler) Listar(w http.ResponseWriter, r *http.Request) {
	tenantID := obtenerTenantID(r)
	contratos, _ := h.Repo.ObtenerTodos(tenantID)

	tmpl, _ := template.ParseFiles("ui/templates/tenant/contratos_ui.html")
	tmpl.ExecuteTemplate(w, "tabla_contratos", contratos)
}

func (h *ContratoHandler) Crear(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	tID, _ := strconv.Atoi(r.FormValue("trabajador_id"))
	rID, _ := strconv.Atoi(r.FormValue("regimen_id"))
	sueldo, _ := strconv.ParseFloat(r.FormValue("sueldo_base"), 64)

	fFinStr := r.FormValue("fecha_fin")
	var fFin *string
	if strings.TrimSpace(fFinStr) != "" {
		fFin = &fFinStr
	}

	nuevoContrato := models.Contrato{
		TenantID:     obtenerTenantID(r),
		TrabajadorID: tID,
		RegimenID:    rID,
		Cargo:        r.FormValue("cargo"),
		SueldoBase:   sueldo,
		FechaInicio:  r.FormValue("fecha_inicio"),
		FechaFin:     fFin,
		Activo:       r.FormValue("activo") == "on",
	}

	h.Repo.Crear(&nuevoContrato)
	h.Listar(w, r)
}
