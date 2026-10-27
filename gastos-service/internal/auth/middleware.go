package auth

import (
	"context"
	"net/http"
	"strings"

	"gastos-service/internal/bitacora"
	"gastos-service/internal/crud"
)

// Roles del sistema: Administrador del Sistema y Encargado de Logística.
const (
	RolAdmin     = "admin"
	RolLogistica = "logistica"
)

// RolValido dice si el rol es uno de los dos que existen.
func RolValido(rol string) bool { return rol == RolAdmin || rol == RolLogistica }

// Identidad es el usuario autenticado de una petición, tal como está en la base en ese momento.
type Identidad struct {
	ID       int
	Username string
	Email    string
	Nombre   string
	Rol      string
	// DebeCambiarClave repite la marca del usuario; el middleware ya bloqueó lo que no puede hacer.
	DebeCambiarClave bool
}

// Actor es la identidad vista como autor de una entrada de la bitácora.
func (i Identidad) Actor() bitacora.Actor { return bitacora.Actor{ID: i.ID, Nombre: i.Nombre} }

type claveIdentidad struct{}

// ConIdentidad devuelve un contexto que lleva la identidad de la petición.
func ConIdentidad(ctx context.Context, id Identidad) context.Context {
	return context.WithValue(ctx, claveIdentidad{}, id)
}

// IdentidadDe lee la identidad que dejó el middleware; false si la petición no pasó por él.
func IdentidadDe(ctx context.Context) (Identidad, bool) {
	id, ok := ctx.Value(claveIdentidad{}).(Identidad)
	return id, ok
}

// Autenticar exige Authorization: Bearer <JWT> válido de un usuario activo en toda petición,
// salvo las que esPublica acepte y el preflight OPTIONS de CORS. El token solo dice quién es
// (sub): el rol y el estado se leen de la base en cada petición, así que desactivar un usuario o
// cambiarle el rol surte efecto al instante. La identidad queda en el contexto de la petición.
// Mientras el usuario deba cambiar su contraseña (también leído de la base), solo se atienden
// /auth/me, /auth/cambiar-clave y /auth/logout; todo lo demás responde 403.
func (s *Servicio) Autenticar(next http.Handler, esPublica func(*http.Request) bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions || (esPublica != nil && esPublica(r)) {
			next.ServeHTTP(w, r)
			return
		}
		cabecera := r.Header.Get("Authorization")
		if !strings.HasPrefix(cabecera, "Bearer ") {
			noAutenticado(w, "No autenticado")
			return
		}
		username, err := s.UsuarioDeToken(strings.TrimSpace(cabecera[len("Bearer "):]))
		if err != nil {
			noAutenticado(w, "Token inválido")
			return
		}
		u, err := s.PorUsername(r.Context(), username)
		if err != nil {
			falloInterno(w, err)
			return
		}
		if u == nil {
			noAutenticado(w, "Usuario no encontrado")
			return
		}
		if !u.Estado {
			noAutenticado(w, "Usuario inactivo")
			return
		}
		if u.DebeCambiarClave && !s.permitidaConClavePendiente(r) {
			crud.Detalle(w, http.StatusForbidden, MsgCambiarClave)
			return
		}
		id := Identidad{ID: u.ID, Username: u.Username, Email: u.Email, Nombre: u.Nombre, Rol: u.Rol, DebeCambiarClave: u.DebeCambiarClave}
		next.ServeHTTP(w, r.WithContext(ConIdentidad(r.Context(), id)))
	})
}

// MsgCambiarClave es el detalle del 403 que reciben los usuarios con la contraseña pendiente de cambio.
const MsgCambiarClave = "Debes cambiar tu contraseña"

// permitidaConClavePendiente dice si la petición se atiende aunque el usuario deba cambiar su
// contraseña: ver su perfil, cambiarla y cerrar sesión.
func (s *Servicio) permitidaConClavePendiente(r *http.Request) bool {
	prefijo := s.prefijo
	if prefijo == "" {
		prefijo = "/api/v1"
	}
	switch {
	case r.Method == http.MethodGet && r.URL.Path == prefijo+"/auth/me":
		return true
	case r.Method == http.MethodPost && (r.URL.Path == prefijo+"/auth/cambiar-clave" || r.URL.Path == prefijo+"/auth/logout"):
		return true
	}
	return false
}

func noAutenticado(w http.ResponseWriter, detalle string) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	crud.Detalle(w, http.StatusUnauthorized, detalle)
}
