package models

import (
	"fmt"
	"strings"
	"time"
)

// MefMucValor representa un registro histórico del Monto Único Consolidado (MUC) aprobado por el MEF
type MefMucValor struct {
	ID                int       `json:"id"`
	NormaLegal        string    `json:"norma_legal"`
	FechaNorma        time.Time `json:"fecha_norma"`
	FechaNormaFormato string    `json:"fecha_norma_formato,omitempty"`
	GrupoOcupacional  string    `json:"grupo_ocupacional"`
	NivelRemunerativo string    `json:"nivel_remunerativo"`
	MontoMuc          float64   `json:"monto_muc"`
	MontoMucFormato   string    `json:"monto_muc_formato,omitempty"`
	Activo            bool      `json:"activo"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// FormatearMontoMUC convierte un valor float64 en formato financiero "#,##0.00"
func FormatearMontoMUC(monto float64) string {
	partes := strings.Split(fmt.Sprintf("%.2f", monto), ".")
	entero := partes[0]
	decimal := partes[1]

	var res []string
	n := len(entero)
	for i, c := range entero {
		if i > 0 && (n-i)%3 == 0 {
			res = append(res, ",")
		}
		res = append(res, string(c))
	}
	return strings.Join(res, "") + "." + decimal
}

// MefMucFiltros encapsula las opciones de filtrado y paginación para la lista de MUC
type MefMucFiltros struct {
	NormaLegal string `json:"norma_legal"`
	FechaNorma string `json:"fecha_norma"`
	Activo     string `json:"activo"` // "todos", "activos", "inactivos"
	Buscar     string `json:"buscar"` // Filtra por grupo ocupacional o nivel remunerativo
	Pagina     int    `json:"pagina"`
	Limite     int    `json:"limite"`
}

// MefMucRespuestaDTO estructura la respuesta enviada a la plantilla de vista MUC
type MefMucRespuestaDTO struct {
	Valores        []MefMucValor  `json:"valores"`
	Paginacion     PaginacionDTO  `json:"paginacion"`
	Filtros        MefMucFiltros  `json:"filtros"`
	NormasLegales  []string       `json:"normas_legales"`
}
