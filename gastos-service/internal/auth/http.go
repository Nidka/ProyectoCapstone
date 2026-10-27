package auth

import (
	"encoding/json"
	"net/http"
	"regexp"

	"gastos-service/internal/bitacora"
	"gastos-service/internal/crud"
)

var reCorreo = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

// usuarioOut es la respuesta pública de un usuario (sin el hash).
type usuarioOut struct {
	ID       int    `json:"id_usuario"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Nombre   string `json:"nombre"`
	Rol      string `json:"rol"`
	Estado   bool   `json:"estado"`
	// DebeCambiarClave: el frontend lleva a «Cambiar contraseña» mientras sea verdadera.
	DebeCambiarClave bool `json:"debe_cambiar_clave"`
}

func salida(u *Usuario) usuarioOut {
	return usuarioOut{u.ID, u.Username, u.Email, u.Nombre, u.Rol, u.Estado, u.DebeCambiarClave}
}

// Registrar expone bajo el prefijo /auth/login, /auth/logout, /auth/register, /auth/me, /auth/cambiar-clave y la
// gestión de /usuarios. Solo login es público (lo decide el middleware, ver Autenticar); register y
// /usuarios exigen ser admin.
func (s *Servicio) Registrar(mux *http.ServeMux, prefijo string) {
	s.prefijo = prefijo
	mux.HandleFunc("POST "+prefijo+"/auth/login", s.login)
	mux.HandleFunc("POST "+prefijo+"/auth/register", s.registro)
	mux.HandleFunc("GET "+prefijo+"/auth/me", s.me)
	mux.HandleFunc("POST "+prefijo+"/auth/logout", s.logout)
	mux.HandleFunc("POST "+prefijo+"/auth/cambiar-clave", s.cambiarClave)

	mux.HandleFunc("GET "+prefijo+"/usuarios", s.listarUsuarios)
	mux.HandleFunc("POST "+prefijo+"/usuarios", s.crearUsuario)
	mux.HandleFunc("PUT "+prefijo+"/usuarios/{id}", s.editarUsuario)
	mux.HandleFunc("POST "+prefijo+"/usuarios/{id}/desactivar", s.desactivarUsuario)
	mux.HandleFunc("POST "+prefijo+"/usuarios/{id}/activar", s.activarUsuario)
}

func faltante(campo string, cuerpo any) crud.ErrorCampo {
	return crud.ErrorCampo{Tipo: "missing", Loc: []any{"body", campo}, Msg: "Field required", Input: cuerpo}
}

// texto lee un campo de texto obligatorio; añade el error y devuelve false si falta o no es texto.
func texto(cuerpo map[string]any, campo string, errs *[]crud.ErrorCampo) (string, bool) {
	v, ok := cuerpo[campo]
	if !ok {
		*errs = append(*errs, faltante(campo, cuerpo))
		return "", false
	}
	s, ok := v.(string)
	if !ok {
		*errs = append(*errs, crud.ErrorCampo{Tipo: "string_type", Loc: []any{"body", campo}, Msg: "Input should be a valid string", Input: v})
		return "", false
	}
	return s, true
}

func leerCuerpo(w http.ResponseWriter, req *http.Request) (map[string]any, bool) {
	var obj map[string]any
	if err := json.NewDecoder(http.MaxBytesReader(w, req.Body, 1<<20)).Decode(&obj); err != nil || obj == nil {
		crud.EscribirJSON(w, http.StatusUnprocessableEntity, map[string]any{"detail": []crud.ErrorCampo{
			{Tipo: "json_invalid", Loc: []any{"body", 0}, Msg: "JSON decode error", Input: map[string]any{}}}})
		return nil, false
	}
	return obj, true
}

func (s *Servicio) login(w http.ResponseWriter, req *http.Request) {
	cuerpo, ok := leerCuerpo(w, req)
	if !ok {
		return
	}
	var errs []crud.ErrorCampo
	username, _ := texto(cuerpo, "username", &errs)
	contrasena, _ := texto(cuerpo, "password", &errs)
	if len(errs) > 0 {
		crud.EscribirJSON(w, http.StatusUnprocessableEntity, map[string]any{"detail": errs})
		return
	}
	u, err := s.PorUsername(req.Context(), username)
	if err != nil {
		falloInterno(w, err)
		return
	}
	if u == nil || !Verificar(contrasena, u.Hash) {
		crud.Detalle(w, http.StatusUnauthorized, "Credenciales inválidas")
		return
	}
	if !u.Estado {
		crud.Detalle(w, http.StatusUnauthorized, "Usuario inactivo")
		return
	}
	// RNF-11: el inicio de sesión queda registrado; si no se puede registrar, no se entrega el token.
	err = s.registrar(req.Context(), bitacora.Entrada{
		Actor: bitacora.Actor{ID: u.ID, Nombre: u.Nombre}, Accion: "sesion.inicio", Entidad: "usuario", IDEntidad: int64(u.ID),
	})
	if err != nil {
		falloInterno(w, err)
		return
	}
	token, err := s.CrearToken(*u)
	if err != nil {
		falloInterno(w, err)
		return
	}
	crud.EscribirJSON(w, http.StatusOK, struct {
		AccessToken string     `json:"access_token"`
		TokenType   string     `json:"token_type"`
		Usuario     usuarioOut `json:"usuario"`
		// Repite usuario.debe_cambiar_clave para decidir la pantalla sin abrir el objeto.
		DebeCambiarClave bool `json:"debe_cambiar_clave"`
	}{token, "bearer", salida(u), u.DebeCambiarClave})
}

func (s *Servicio) registro(w http.ResponseWriter, req *http.Request) {
	quien, autenticado := IdentidadDe(req.Context())
	if !autenticado {
		noAutenticado(w, "No autenticado")
		return
	}
	if quien.Rol != RolAdmin {
		crud.Detalle(w, http.StatusForbidden, "Solo el administrador puede registrar usuarios")
		return
	}
	cuerpo, ok := leerCuerpo(w, req)
	if !ok {
		return
	}
	var errs []crud.ErrorCampo
	username, _ := texto(cuerpo, "username", &errs)
	email, okEmail := texto(cuerpo, "email", &errs)
	if okEmail && !reCorreo.MatchString(email) {
		errs = append(errs, crud.ErrorCampo{Tipo: "value_error", Loc: []any{"body", "email"}, Msg: "value is not a valid email address", Input: email})
	}
	contrasena, _ := texto(cuerpo, "password", &errs)
	nombre, _ := texto(cuerpo, "nombre", &errs)
	rol := RolLogistica
	if v, presente := cuerpo["rol"]; presente {
		if r, ok := texto(cuerpo, "rol", &errs); ok {
			rol = r
			if !RolValido(rol) {
				errs = append(errs, crud.ErrorCampo{Tipo: "value_error", Loc: []any{"body", "rol"}, Msg: "Value error, Rol debe ser 'admin' o 'logistica'", Input: v})
			}
		}
	}
	if len(errs) > 0 {
		crud.EscribirJSON(w, http.StatusUnprocessableEntity, map[string]any{"detail": errs})
		return
	}
	if existe, err := s.PorUsername(req.Context(), username); err != nil {
		falloInterno(w, err)
		return
	} else if existe != nil {
		crud.Detalle(w, http.StatusConflict, "El usuario '"+username+"' ya existe")
		return
	}
	if existe, err := s.PorEmail(req.Context(), email); err != nil {
		falloInterno(w, err)
		return
	} else if existe != nil {
		crud.Detalle(w, http.StatusConflict, "El email '"+email+"' ya está registrado")
		return
	}
	u, err := s.Crear(req.Context(), quien.Actor(), Usuario{Username: username, Email: email, Nombre: nombre, Rol: rol}, contrasena)
	if err != nil {
		falloInterno(w, err)
		return
	}
	crud.EscribirJSON(w, http.StatusCreated, salida(u))
}

// me devuelve al usuario autenticado; el middleware ya validó el token y que siga activo.
func (s *Servicio) me(w http.ResponseWriter, req *http.Request) {
	id, ok := IdentidadDe(req.Context())
	if !ok {
		noAutenticado(w, "No autenticado")
		return
	}
	crud.EscribirJSON(w, http.StatusOK, usuarioOut{id.ID, id.Username, id.Email, id.Nombre, id.Rol, true, id.DebeCambiarClave})
}

// logout registra el cierre de sesión (RNF-12). El token es un JWT sin estado en el servidor, así
// que sigue siendo válido hasta que vence; quien cierra sesión debe descartarlo. Se atiende aunque
// el usuario tenga la contraseña pendiente de cambio (ver Autenticar).
func (s *Servicio) logout(w http.ResponseWriter, req *http.Request) {
	id, ok := IdentidadDe(req.Context())
	if !ok {
		noAutenticado(w, "No autenticado")
		return
	}
	err := s.registrar(req.Context(), bitacora.Entrada{Actor: id.Actor(), Accion: "sesion.cierre", Entidad: "usuario", IDEntidad: int64(id.ID)})
	if err != nil {
		falloInterno(w, err)
		return
	}
	crud.Detalle(w, http.StatusOK, "Sesión cerrada")
}
