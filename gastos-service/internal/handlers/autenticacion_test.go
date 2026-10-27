package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"gastos-service/internal/auth"
	"gastos-service/internal/config"
	"gastos-service/internal/handlers"

	"github.com/jackc/pgx/v5/pgxpool"
)

// claveDePrueba firma los tokens de todas las pruebas de integración.
const claveDePrueba = "prueba"

// sembrarUsuario inserta un usuario en el esquema temporal (sin bcrypt: la contraseña no importa
// porque las pruebas entran con token) y devuelve su id y un token válido de 8 horas. Ya cambió su
// contraseña (debe_cambiar_clave = FALSE); si no, el middleware le cerraría el paso a casi todo.
func sembrarUsuario(t testing.TB, pool *pgxpool.Pool, username, nombre, rol string, activo bool) (int, string) {
	t.Helper()
	var id int
	err := pool.QueryRow(context.Background(), `
		INSERT INTO usuario (username, email, hashed_password, nombre, rol, estado, debe_cambiar_clave)
		VALUES ($1, $2, 'sin-contraseña', $3, $4, $5, FALSE) RETURNING id_usuario`,
		username, username+"@prueba.pe", nombre, rol, activo).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	return id, tokenDe(t, auth.Usuario{ID: id, Username: username, Rol: rol})
}

// tokenDe emite un token firmado con la clave de las pruebas (no toca la base).
func tokenDe(t testing.TB, u auth.Usuario) string {
	t.Helper()
	token, err := (&auth.Servicio{Clave: []byte(claveDePrueba)}).CrearToken(u)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

// servidorAutenticado arma el servicio con su middleware REAL (handlers.Servidor, el mismo que usa
// main.go) y le crea dos usuarios activos: Ana Quispe (logistica) y Admin Prueba (admin).
func servidorAutenticado(t testing.TB, pool *pgxpool.Pool, h *handlers.Handler) (servidor http.Handler, tokenLogistica, tokenAdmin string) {
	t.Helper()
	_, tokenLogistica = sembrarUsuario(t, pool, "ana", "Ana Quispe", "logistica", true)
	_, tokenAdmin = sembrarUsuario(t, pool, "admin_prueba", "Admin Prueba", "admin", true)
	return h.Servidor(), tokenLogistica, tokenAdmin
}

func configDePrueba(uploads string) config.Config {
	return config.Config{UploadsDir: uploads, AppEnv: "dev", JWTSecret: claveDePrueba}
}

// conToken añade el Authorization de un usuario a toda petición que no traiga uno propio. El
// middleware que valida ese token sigue siendo el real.
func conToken(h http.Handler, token string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		h.ServeHTTP(w, r)
	})
}

// pedir hace una petición JSON; token vacío significa sin Authorization.
func pedir(h http.Handler, metodo, ruta, token string, cuerpo any) *httptest.ResponseRecorder {
	var b []byte
	if cuerpo != nil {
		b, _ = json.Marshal(cuerpo)
	}
	req := httptest.NewRequest(metodo, ruta, bytes.NewReader(b))
	if cuerpo != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func (e *entorno) conRol(rol string) string {
	switch rol {
	case "logistica":
		return e.tokenLogistica
	case "admin":
		return e.tokenAdmin
	}
	return ""
}

func detalleDe(rec *httptest.ResponseRecorder) string {
	var cuerpo map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &cuerpo)
	if s, ok := cuerpo["detail"].(string); ok {
		return s
	}
	return fmt.Sprint(cuerpo["error"])
}

// pedirConCabecera hace una petición sin cuerpo con cabeceras sueltas. Si el primer argumento es
// la cabecera Authorization completa (o vacío), el resto son pares nombre/valor.
func pedirConCabecera(h http.Handler, metodo, ruta, authorization string, pares ...string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(metodo, ruta, nil)
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	for i := 0; i+1 < len(pares); i += 2 {
		req.Header.Set(pares[i], pares[i+1])
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}
