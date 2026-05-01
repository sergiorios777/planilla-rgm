package middleware

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/golang-jwt/jwt/v5"
)

// Creamos un tipo específico para las llaves del contexto (buenas prácticas en Go)
type contextKey string

const UsuarioIDKey contextKey = "usuario_id"
const RolKey contextKey = "rol"

// RequireAuth es nuestro Middleware Guardián
func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Intentamos leer la cookie
		cookie, err := r.Cookie("rgm_auth_token")
		if err != nil {
			rechazarAcceso(w, r)
			return
		}

		// 2. Extraemos el JWT y la llave secreta
		tokenStr := cookie.Value
		secreto := os.Getenv("JWT_SECRET")

		// 3. Verificamos la validez y la firma criptográfica del token
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			// Verificamos que el algoritmo de firma sea el correcto (HMAC)
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("método de firma inesperado")
			}
			return []byte(secreto), nil
		})

		// Si el token es inválido, expiró o fue alterado
		if err != nil || !token.Valid {
			rechazarAcceso(w, r)
			return
		}

		// 4. Token válido: Extraemos los datos (Claims) y los pasamos a la siguiente ruta
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			ctx := context.WithValue(r.Context(), UsuarioIDKey, claims["usuario_id"])
			ctx = context.WithValue(ctx, RolKey, claims["rol"])

			if tid, exists := claims["tenant_id"]; exists {
				ctx = context.WithValue(ctx, "tenant_id", tid)
			}

			// Cabeceras de Seguridad Anti-Caché
			// Obligamos al navegador a destruir la página al salir
			w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")

			// Le decimos al guardia: "Todo en orden, déjalo pasar"
			next.ServeHTTP(w, r.WithContext(ctx))
		} else {
			rechazarAcceso(w, r)
		}
	}
}

// rechazarAcceso expulsa al usuario devolviéndolo al login
func rechazarAcceso(w http.ResponseWriter, r *http.Request) {
	// Borramos la cookie por si quedó alguna inválida
	http.SetCookie(w, &http.Cookie{
		Name:     "rgm_auth_token",
		Value:    "",
		MaxAge:   -1,
		HttpOnly: true,
		Path:     "/",
	})

	// Si la petición vino de HTMX, respondemos con la cabecera especial de redirección
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/login")
		w.WriteHeader(http.StatusUnauthorized)
	} else {
		// Si fue una petición normal (recargar la página), redireccionamos normalmente
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

// RequireRole verifica que el usuario autenticado tenga un rol específico antes de dejarlo pasar
func RequireRole(rolRequerido string, next http.HandlerFunc) http.HandlerFunc {
	// Primero usamos RequireAuth para garantizar que el usuario tiene un JWT válido
	return RequireAuth(func(w http.ResponseWriter, r *http.Request) {

		// Extraemos el rol que RequireAuth guardó previamente en el contexto
		rolActual := r.Context().Value(RolKey)

		// Comparamos el rol actual con el rol que exige esta ruta
		if rolActual == nil || rolActual.(string) != rolRequerido {
			// Si no coinciden, bloqueamos el acceso
			http.Error(w, "Acceso denegado: Privilegios insuficientes", http.StatusForbidden)
			return
		}

		// Si el rol es correcto, permitimos que la petición continúe
		next.ServeHTTP(w, r)
	})
}
