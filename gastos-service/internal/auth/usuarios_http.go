package auth

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"gastos-service/internal/crud"
)

func falloInterno(w http.ResponseWriter, err error) {
	log.Printf("auth: %v", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

// soloAdmin exige un administrador autenticado y devuelve quién es; si no, ya respondió 401 o 403.
func soloAdmin(w http.ResponseWriter, req *http.Request) (Identidad, bool) {
	quien, autenticado := IdentidadDe(req.Context())
	if !autenticado {
		noAutenticado(w, "No autenticado")
		return quien, false
	}
	if quien.Rol != RolAdmin {
		crud.Detalle(w, http.StatusForbidden, "Solo el administrador puede gestionar usuarios")
		return quien, false
	}
	return quien, true
}

// idDeRuta lee {id}; responde 422 como FastAPI si no es un entero.
func idDeRuta(w http.ResponseWriter, req *http.Request) (int, bool) {
	txt := req.PathValue("id")
	n, err := strconv.Atoi(txt)
	if err != nil {
		crud.EscribirJSON(w, http.StatusUnprocessableEntity, map[string]any{"detail": []crud.ErrorCampo{
			{Tipo: "int_parsing", Loc: []any{"path", "id"}, Msg: "Input should be a valid integer, unable to parse string as an integer", Input: txt}}})
		return 0, false
	}
	return n, true
}

// errorDeUsuarios responde con el código que corresponde al error de negocio; true si lo manejó.
func errorDeUsuarios(w http.ResponseWriter, err error, datos datosUsuario) bool {
	switch {
	case errors.Is(err, ErrUsuarioNoEncontrado):
		crud.Detalle(w, http.StatusNotFound, "Usuario no encontrado")
	case errors.Is(err, ErrUsuarioRepetido):
		crud.Detalle(w, http.StatusConflict, "El usuario '"+datos.Username+"' ya existe")
	case errors.Is(err, ErrCorreoRepetido):
		crud.Detalle(w, http.StatusConflict, "El email '"+datos.Email+"' ya está registrado")
	case errors.Is(err, ErrUltimoAdmin):
		crud.Detalle(w, http.StatusConflict, "Debe quedar al menos un administrador activo")
	case errors.Is(err, ErrAutodesactivacion):
		crud.Detalle(w, http.StatusConflict, "No puedes desactivar tu propio usuario")
	default:
		return false
	}
	return true
}

func (s *Servicio) listarUsuarios(w http.ResponseWriter, req *http.Request) {
	if _, ok := soloAdmin(w, req); !ok {
		return
	}
	lista, err := s.Listar(req.Context())
	if err != nil {
		falloInterno(w, err)
		return
	}
	salidas := make([]usuarioOut, len(lista))
	for i := range lista {
		salidas[i] = salida(&lista[i])
	}
	crud.EscribirJSON(w, http.StatusOK, salidas)
}

func (s *Servicio) crearUsuario(w http.ResponseWriter, req *http.Request) {
	quien, ok := soloAdmin(w, req)
	if !ok {
		return
	}
	cuerpo, ok := leerCuerpo(w, req)
	if !ok {
		return
	}
	datos, errs := validarCreacion(cuerpo)
	if len(errs) > 0 {
		crud.EscribirJSON(w, http.StatusUnprocessableEntity, map[string]any{"detail": errs})
		return
	}
	u, err := s.CrearUsuario(req.Context(), quien.Actor(), datos)
	if err != nil {
		if !errorDeUsuarios(w, err, datos) {
			falloInterno(w, err)
		}
		return
	}
	crud.EscribirJSON(w, http.StatusCreated, salida(u))
}

func (s *Servicio) editarUsuario(w http.ResponseWriter, req *http.Request) {
	quien, ok := soloAdmin(w, req)
	if !ok {
		return
	}
	id, ok := idDeRuta(w, req)
	if !ok {
		return
	}
	cuerpo, ok := leerCuerpo(w, req)
	if !ok {
		return
	}
	cambios, errs := validarEdicion(cuerpo)
	if len(errs) > 0 {
		crud.EscribirJSON(w, http.StatusUnprocessableEntity, map[string]any{"detail": errs})
		return
	}
	u, err := s.Editar(req.Context(), quien.Actor(), id, cambios)
	if err != nil {
		datos := datosUsuario{}
		if cambios.Email != nil {
			datos.Email = *cambios.Email
		}
		if !errorDeUsuarios(w, err, datos) {
			falloInterno(w, err)
		}
		return
	}
	crud.EscribirJSON(w, http.StatusOK, salida(u))
}

func (s *Servicio) desactivarUsuario(w http.ResponseWriter, req *http.Request) {
	quien, ok := soloAdmin(w, req)
	if !ok {
		return
	}
	id, ok := idDeRuta(w, req)
	if !ok {
		return
	}
	u, err := s.Desactivar(req.Context(), quien.Actor(), id)
	if err != nil {
		if !errorDeUsuarios(w, err, datosUsuario{}) {
			falloInterno(w, err)
		}
		return
	}
	crud.EscribirJSON(w, http.StatusOK, salida(u))
}

func (s *Servicio) activarUsuario(w http.ResponseWriter, req *http.Request) {
	quien, ok := soloAdmin(w, req)
	if !ok {
		return
	}
	id, ok := idDeRuta(w, req)
	if !ok {
		return
	}
	u, err := s.Activar(req.Context(), quien.Actor(), id)
	if err != nil {
		if !errorDeUsuarios(w, err, datosUsuario{}) {
			falloInterno(w, err)
		}
		return
	}
	crud.EscribirJSON(w, http.StatusOK, salida(u))
}

// cambiarClave deja que cualquier usuario autenticado cambie su propia contraseña. Los fallos de
// contenido (clave actual incorrecta, nueva igual a la actual) son 400 y no 401 a propósito: el
// token sigue siendo bueno y el frontend no debe cerrar la sesión por esto.
func (s *Servicio) cambiarClave(w http.ResponseWriter, req *http.Request) {
	quien, autenticado := IdentidadDe(req.Context())
	if !autenticado {
		noAutenticado(w, "No autenticado")
		return
	}
	cuerpo, ok := leerCuerpo(w, req)
	if !ok {
		return
	}
	actual, nueva, errs := validarCambioClave(cuerpo)
	if len(errs) > 0 {
		crud.EscribirJSON(w, http.StatusUnprocessableEntity, map[string]any{"detail": errs})
		return
	}
	u, err := s.PorID(req.Context(), quien.ID)
	if err != nil {
		falloInterno(w, err)
		return
	}
	if u == nil {
		noAutenticado(w, "Usuario no encontrado")
		return
	}
	if !Verificar(actual, u.Hash) {
		crud.Detalle(w, http.StatusBadRequest, "La contraseña actual no es correcta")
		return
	}
	if nueva == actual {
		crud.Detalle(w, http.StatusBadRequest, "La contraseña nueva debe ser distinta de la actual")
		return
	}
	u, err = s.CambiarClave(req.Context(), quien.Actor(), u.ID, nueva)
	if err != nil {
		falloInterno(w, err)
		return
	}
	crud.EscribirJSON(w, http.StatusOK, salida(u))
}
