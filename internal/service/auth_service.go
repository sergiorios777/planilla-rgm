package service

import (
	"os"
	"time"

	"planilla-rgm/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerarJWT crea el "pasaporte" que guardaremos en la cookie del usuario
func GenerarJWT(usuario *models.Usuario) (string, error) {
	// Leemos la llave secreta desde el .env
	secreto := os.Getenv("JWT_SECRET")

	// Definimos qué información (Claims) viajará de forma segura dentro del token
	claims := jwt.MapClaims{
		"usuario_id": usuario.ID,
		"rol":        usuario.Rol,
		// El token expirará en 24 horas
		"exp": time.Now().Add(time.Hour * 24).Unix(),
	}

	// Si el usuario pertenece a una municipalidad, incluimos su TenantID
	if usuario.TenantID != nil {
		claims["tenant_id"] = *usuario.TenantID
	}

	// Firmamos el token con nuestra llave
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secreto))
}
