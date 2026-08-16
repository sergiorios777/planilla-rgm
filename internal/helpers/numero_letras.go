package helpers

import (
	"fmt"
	"math"
	"strings"
)

var unidades = []string{"", "UN", "DOS", "TRES", "CUATRO", "CINCO", "SEIS", "SIETE", "OCHO", "NUEVE"}
var decenas10 = []string{"DIEZ", "ONCE", "DOCE", "TRECE", "CATORCE", "QUINCE", "DIECISEIS", "DIECISIETE", "DIECIOCHO", "DIECINUEVE"}
var decenas = []string{"", "DIEZ", "VEINTE", "TREINTA", "CUARENTA", "CINCUENTA", "SESENTA", "SETENTA", "OCHENTA", "NOVENTA"}
var centenas = []string{"", "CIENTO", "DOSCIENTOS", "TRESCIENTOS", "CUATROCIENTOS", "QUINIENTOS", "SEISCIENTOS", "SETECIENTOS", "OCHOCIENTOS", "NOVECIENTOS"}

// NumeroALetras convierte un monto numérico a su representación en letras en castellano con formato legal peruano.
// Ejemplo: 1585.23 -> "SON: UN MIL QUINIENTOS OCHENTA Y CINCO Y 23/100 SOLES"
func NumeroALetras(monto float64) string {
	if monto < 0 {
		monto = math.Abs(monto)
	}

	entero := int64(monto)
	decimales := int64(math.Round((monto - float64(entero)) * 100))
	if decimales >= 100 {
		entero++
		decimales -= 100
	}

	textoEntero := ""
	if entero == 0 {
		textoEntero = "CERO"
	} else {
		textoEntero = convertirGrupo(entero)
	}

	return fmt.Sprintf("SON: %s Y %02d/100 SOLES", textoEntero, decimales)
}

func convertirGrupo(n int64) string {
	if n == 0 {
		return ""
	}
	if n == 100 {
		return "CIEN"
	}

	if n < 10 {
		return unidades[n]
	}
	if n >= 10 && n < 20 {
		return decenas10[n-10]
	}
	if n >= 20 && n < 30 {
		if n == 20 {
			return "VEINTE"
		}
		return "VEINTI" + unidades[n-20]
	}
	if n >= 30 && n < 100 {
		dec := n / 10
		uni := n % 10
		if uni == 0 {
			return decenas[dec]
		}
		return decenas[dec] + " Y " + unidades[uni]
	}
	if n >= 100 && n < 1000 {
		cent := n / 100
		resto := n % 100
		if resto == 0 {
			if cent == 1 {
				return "CIEN"
			}
			return centenas[cent]
		}
		return centenas[cent] + " " + convertirGrupo(resto)
	}
	if n >= 1000 && n < 1000000 {
		miles := n / 1000
		resto := n % 1000
		strMiles := ""
		if miles == 1 {
			strMiles = "UN MIL"
		} else {
			strMiles = convertirGrupo(miles) + " MIL"
		}

		if resto == 0 {
			return strMiles
		}
		return strMiles + " " + convertirGrupo(resto)
	}
	if n >= 1000000 && n < 1000000000 {
		millones := n / 1000000
		resto := n % 1000000
		strMillones := ""
		if millones == 1 {
			strMillones = "UN MILLON"
		} else {
			strMillones = convertirGrupo(millones) + " MILLONES"
		}

		if resto == 0 {
			return strMillones
		}
		return strMillones + " " + convertirGrupo(resto)
	}

	return strings.TrimSpace(fmt.Sprintf("%d", n))
}
