package models

// ModalConfirmacionDTO representa la configuración de un modal de confirmación reutilizable
type ModalConfirmacionDTO struct {
	ID            string `json:"id"`
	Titulo        string `json:"titulo"`
	Mensaje       string `json:"mensaje"`
	BadgeTexto    string `json:"badge_texto"`
	BadgeClase    string `json:"badge_clase"`
	BotonCancelar string `json:"boton_cancelar"`
	BotonConfirm  string `json:"boton_confirm"`
	MetodoHTTP    string `json:"metodo_http"` // POST, DELETE, PUT
	AccionURL     string `json:"accion_url"`
	TargetID      string `json:"target_id"`
}

// PaginacionDTO contiene la información necesaria para renderizar la barra de paginación HTMX
type PaginacionDTO struct {
	PaginaActual    int    `json:"pagina_actual"`
	TotalPaginas    int    `json:"total_paginas"`
	PaginaAnterior  int    `json:"pagina_anterior"`
	PaginaSiguiente int    `json:"pagina_siguiente"`
	TotalRegistros  int    `json:"total_registros"`
	RangoPaginas    []int  `json:"rango_paginas"`   // Ej: [1, 2, 3, 4, 5]
	AccionURL       string `json:"accion_url"`     // Ej: "/tenant/conceptos-locales/lista"
	TargetID        string `json:"target_id"`      // Ej: "#lista-conceptos-tenant"
	FormIncludeID   string `json:"form_include_id"` // Ej: "#form-filtros-conceptos-tenant"
}

// CalcularPaginacion genera el struct PaginacionDTO calculando el rango de páginas numeradas
func CalcularPaginacion(paginaActual, totalPaginas, totalRegistros int, accionURL, targetID, formIncludeID string) PaginacionDTO {
	if paginaActual < 1 {
		paginaActual = 1
	}
	if totalPaginas < 1 {
		totalPaginas = 1
	}
	if paginaActual > totalPaginas {
		paginaActual = totalPaginas
	}

	paginaAnt := paginaActual - 1
	paginaSig := paginaActual + 1
	if paginaSig > totalPaginas {
		paginaSig = 0
	}

	maxVisible := 5
	inicio := paginaActual - maxVisible/2
	if inicio < 1 {
		inicio = 1
	}
	fin := inicio + maxVisible - 1
	if fin > totalPaginas {
		fin = totalPaginas
		inicio = fin - maxVisible + 1
		if inicio < 1 {
			inicio = 1
		}
	}

	var rango []int
	for i := inicio; i <= fin; i++ {
		rango = append(rango, i)
	}

	return PaginacionDTO{
		PaginaActual:    paginaActual,
		TotalPaginas:    totalPaginas,
		PaginaAnterior:  paginaAnt,
		PaginaSiguiente: paginaSig,
		TotalRegistros:  totalRegistros,
		RangoPaginas:    rango,
		AccionURL:       accionURL,
		TargetID:        targetID,
		FormIncludeID:   formIncludeID,
	}
}
