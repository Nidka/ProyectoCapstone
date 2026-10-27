package auth

import (
	"net/http/httptest"
	"strings"
	"testing"

	"gastos-service/internal/crud"
)

// camposConError devuelve el nombre del campo (último elemento de loc) de cada error.
func camposConError(errs []crud.ErrorCampo) []string {
	var out []string
	for _, e := range errs {
		out = append(out, e.Loc[len(e.Loc)-1].(string)+":"+e.Tipo)
	}
	return out
}

func tieneError(errs []crud.ErrorCampo, campo, tipo string) bool {
	for _, c := range camposConError(errs) {
		if c == campo+":"+tipo {
			return true
		}
	}
	return false
}

func cuerpoValido() map[string]any {
	return map[string]any{"username": "luis", "email": "luis@lizar.pe", "password": "secreta12", "nombre": "Luis Rojas", "rol": "admin"}
}

func TestValidarCreacionAceptaUnCuerpoCompleto(t *testing.T) {
	d, errs := validarCreacion(cuerpoValido())
	if len(errs) != 0 {
		t.Fatalf("errores inesperados: %v", camposConError(errs))
	}
	if d.Username != "luis" || d.Email != "luis@lizar.pe" || d.Nombre != "Luis Rojas" || d.Rol != RolAdmin || d.Password != "secreta12" {
		t.Errorf("datos = %+v", d)
	}
}

func TestValidarCreacionUsaLogisticaPorDefectoYRecortaEspacios(t *testing.T) {
	c := cuerpoValido()
	delete(c, "rol")
	c["username"] = "  luis  "
	c["nombre"] = "  Luis Rojas "
	c["email"] = " luis@lizar.pe "
	d, errs := validarCreacion(c)
	if len(errs) != 0 {
		t.Fatalf("errores inesperados: %v", camposConError(errs))
	}
	if d.Rol != RolLogistica || d.Username != "luis" || d.Nombre != "Luis Rojas" || d.Email != "luis@lizar.pe" {
		t.Errorf("datos = %+v", d)
	}
}

func TestValidarCreacionRechazaCamposInvalidos(t *testing.T) {
	casos := []struct {
		nombre string
		cambia func(map[string]any)
		campo  string
		tipo   string
	}{
		{"falta username", func(c map[string]any) { delete(c, "username") }, "username", "missing"},
		{"username vacío", func(c map[string]any) { c["username"] = "   " }, "username", "string_too_short"},
		{"username muy largo", func(c map[string]any) { c["username"] = strings.Repeat("a", 51) }, "username", "string_too_long"},
		{"username con espacios", func(c map[string]any) { c["username"] = "luis rojas" }, "username", "value_error"},
		{"falta email", func(c map[string]any) { delete(c, "email") }, "email", "missing"},
		{"email sin arroba", func(c map[string]any) { c["email"] = "no-es-correo" }, "email", "value_error"},
		{"email muy largo", func(c map[string]any) { c["email"] = strings.Repeat("a", 95) + "@lizar.pe" }, "email", "string_too_long"},
		{"falta nombre", func(c map[string]any) { delete(c, "nombre") }, "nombre", "missing"},
		{"nombre vacío", func(c map[string]any) { c["nombre"] = "  " }, "nombre", "string_too_short"},
		{"nombre muy largo", func(c map[string]any) { c["nombre"] = strings.Repeat("a", 101) }, "nombre", "string_too_long"},
		{"falta password", func(c map[string]any) { delete(c, "password") }, "password", "missing"},
		{"password de 7", func(c map[string]any) { c["password"] = "1234567" }, "password", "string_too_short"},
		{"password no es texto", func(c map[string]any) { c["password"] = 12345678 }, "password", "string_type"},
		{"rol desconocido", func(c map[string]any) { c["rol"] = "gerente" }, "rol", "value_error"},
		{"rol operador", func(c map[string]any) { c["rol"] = "operador" }, "rol", "value_error"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			cuerpo := cuerpoValido()
			c.cambia(cuerpo)
			_, errs := validarCreacion(cuerpo)
			if !tieneError(errs, c.campo, c.tipo) {
				t.Errorf("errores = %v, quiero %s:%s", camposConError(errs), c.campo, c.tipo)
			}
		})
	}
}

func TestValidarCreacionAceptaClaveDeOchoCaracteresConAcentos(t *testing.T) {
	c := cuerpoValido()
	c["password"] = "áéíóúñüü" // 8 caracteres, 16 bytes
	if _, errs := validarCreacion(c); len(errs) != 0 {
		t.Fatalf("8 caracteres debían bastar: %v", camposConError(errs))
	}
}

func TestValidarEdicionSoloCambiaLoQueLlega(t *testing.T) {
	cambios, errs := validarEdicion(map[string]any{"nombre": " Luis R. "})
	if len(errs) != 0 {
		t.Fatalf("errores: %v", camposConError(errs))
	}
	if cambios.Nombre == nil || *cambios.Nombre != "Luis R." || cambios.Email != nil || cambios.Rol != nil || cambios.Password != nil {
		t.Errorf("cambios = %+v", cambios)
	}
	cambios, errs = validarEdicion(map[string]any{"nombre": "A", "email": "a@lizar.pe", "rol": "admin", "password": "otraclave1"})
	if len(errs) != 0 || cambios.Email == nil || cambios.Rol == nil || *cambios.Rol != RolAdmin || cambios.Password == nil {
		t.Errorf("cambios = %+v, errs = %v", cambios, camposConError(errs))
	}
}

func TestValidarEdicionRechazaCamposInvalidos(t *testing.T) {
	casos := []struct {
		nombre string
		cuerpo map[string]any
		campo  string
		tipo   string
	}{
		{"nombre vacío", map[string]any{"nombre": ""}, "nombre", "string_too_short"},
		{"nombre es número", map[string]any{"nombre": 3}, "nombre", "string_type"},
		{"email inválido", map[string]any{"email": "x"}, "email", "value_error"},
		{"rol inválido", map[string]any{"rol": "root"}, "rol", "value_error"},
		{"password corta", map[string]any{"password": "corta"}, "password", "string_too_short"},
	}
	for _, c := range casos {
		if _, errs := validarEdicion(c.cuerpo); !tieneError(errs, c.campo, c.tipo) {
			t.Errorf("%s: errores = %v, quiero %s:%s", c.nombre, camposConError(errs), c.campo, c.tipo)
		}
	}
}

func TestValidarCambioDeClave(t *testing.T) {
	actual, nueva, errs := validarCambioClave(map[string]any{"clave_actual": "admin123", "clave_nueva": "NuevaClave9"})
	if len(errs) != 0 || actual != "admin123" || nueva != "NuevaClave9" {
		t.Fatalf("válido: %q %q %v", actual, nueva, camposConError(errs))
	}
	casos := []struct {
		nombre string
		cuerpo map[string]any
		campo  string
		tipo   string
	}{
		{"falta la actual", map[string]any{"clave_nueva": "NuevaClave9"}, "clave_actual", "missing"},
		{"falta la nueva", map[string]any{"clave_actual": "x"}, "clave_nueva", "missing"},
		{"nueva de 7", map[string]any{"clave_actual": "x", "clave_nueva": "1234567"}, "clave_nueva", "string_too_short"},
		{"nueva no es texto", map[string]any{"clave_actual": "x", "clave_nueva": 12345678}, "clave_nueva", "string_type"},
	}
	for _, c := range casos {
		if _, _, errs := validarCambioClave(c.cuerpo); !tieneError(errs, c.campo, c.tipo) {
			t.Errorf("%s: errores = %v, quiero %s:%s", c.nombre, camposConError(errs), c.campo, c.tipo)
		}
	}
}

func TestRutasQueSePermitenConLaClavePendiente(t *testing.T) {
	s := &Servicio{}
	casos := []struct {
		metodo, ruta string
		permitida    bool
	}{
		{"GET", "/api/v1/auth/me", true},
		{"POST", "/api/v1/auth/cambiar-clave", true},
		{"POST", "/api/v1/auth/logout", true},
		{"GET", "/api/v1/usuarios", false},
		{"GET", "/gastos", false},
		{"POST", "/gastos", false},
		{"GET", "/api/v1/clientes", false},
		{"POST", "/api/v1/auth/register", false},
		{"GET", "/api/v1/auth/cambiar-clave", false},
		{"POST", "/api/v1/auth/me", false},
		{"GET", "/api/v1/auth/me/otra", false},
	}
	for _, c := range casos {
		r := httptest.NewRequest(c.metodo, c.ruta, nil)
		if got := s.permitidaConClavePendiente(r); got != c.permitida {
			t.Errorf("%s %s = %v, quiero %v", c.metodo, c.ruta, got, c.permitida)
		}
	}
}
