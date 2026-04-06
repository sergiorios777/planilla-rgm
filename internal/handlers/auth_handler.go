package handlers

import (
	"html/template"
	"net/http"
	"planilla-rgm/internal/repository"
	"planilla-rgm/internal/service"
	"time"
)

type AuthHandler struct {
	Repo *repository.UsuarioRepository
}

// MostrarLogin devuelve la página HTML del formulario
func (h *AuthHandler) MostrarLogin(w http.ResponseWriter, r *http.Request) {
	tmpl, _ := template.ParseFiles("ui/templates/login.html")
	tmpl.Execute(w, nil)
}

// ProcesarLogin verifica las credenciales y crea la cookie
func (h *AuthHandler) ProcesarLogin(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	email := r.FormValue("email")
	password := r.FormValue("password")

	// 1. Buscamos al usuario por su correo
	usuario, err := h.Repo.ObtenerPorEmail(email)
	if err != nil {
		// Error genérico por seguridad (no decimos si falló el correo o la clave)
		http.Error(w, "Credenciales inválidas", http.StatusUnauthorized)
		return
	}

	// 2. Verificamos la contraseña
	if !service.CheckPassword(usuario.Password, password) {
		http.Error(w, "Credenciales inválidas", http.StatusUnauthorized)
		return
	}

	// 3. Contraseña correcta: Generamos el JWT
	tokenString, err := service.GenerarJWT(usuario)
	if err != nil {
		http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
		return
	}

	// 4. Creamos la Cookie HttpOnly (El "pasaporte" seguro)
	http.SetCookie(w, &http.Cookie{
		Name:     "rgm_auth_token",
		Value:    tokenString,
		Expires:  time.Now().Add(time.Hour * 24),
		HttpOnly: true,  // Protege contra robo por JavaScript (XSS)
		Secure:   false, // PONER EN TRUE EN PRODUCCIÓN (requiere HTTPS)
		Path:     "/",   // La cookie es válida en toda la aplicación
		SameSite: http.SameSiteLaxMode,
	})

	// 5. Truco de HTMX: Le decimos al navegador que haga una redirección completa al panel
	w.Header().Set("HX-Redirect", "/admin")
	w.WriteHeader(http.StatusOK)
}

// CerrarSesion destruye la cookie y redirige al login
func (h *AuthHandler) CerrarSesion(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "rgm_auth_token",
		Value:    "",
		MaxAge:   -1, // Un valor negativo destruye la cookie inmediatamente
		HttpOnly: true,
		Path:     "/",
	})

	w.Header().Set("HX-Redirect", "/login")
	w.WriteHeader(http.StatusOK)
}
