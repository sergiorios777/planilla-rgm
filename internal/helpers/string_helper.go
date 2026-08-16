package helpers

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// FormatearRol transforma un identificador de rol (ej: "operador_planilla") 
// en un texto legible para el usuario (ej: "Operador planilla").
func FormatearRol(rol string) string {
	if rol == "" {
		return "Administrador"
	}
	switch rol {
	case "tenant_admin":
		return "Administrador"
	case "tenant_operator":
		return "Operador"
	case "super_admin":
		return "Súper Admin"
	}

	// Reemplazar guiones bajos por espacios
	formateado := strings.ReplaceAll(rol, "_", " ")
	formateado = strings.TrimSpace(formateado)
	if len(formateado) == 0 {
		return "Administrador"
	}

	// Primera letra de la primera palabra en mayúscula
	r, size := utf8.DecodeRuneInString(formateado)
	return string(unicode.ToUpper(r)) + formateado[size:]
}
