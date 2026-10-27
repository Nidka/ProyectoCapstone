package handlers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"gastos-service/internal/auth"
)

// usuarioJSON es la forma pública de un usuario en las respuestas.
type usuarioJSON struct {
	ID               int    `json:"id_usuario"`
	Username         string `json:"username"`
	Email            string `json:"email"`
	Nombre           string `json:"nombre"`
	Rol              string `json:"rol"`
	Estado           bool   `json:"estado"`
	DebeCambiarClave bool   `json:"debe_cambiar_clave"`
}

func usuarioDe(t *testing.T, rec *httptest.ResponseRecorder) usuarioJSON {
	t.Helper()
	var u usuarioJSON
	if err := json.Unmarshal(rec.Body.Bytes(), &u); err != nil {
		t.Fatalf("respuesta no es un usuario: %s", rec.Body)
	}
	return u
}

// sembrarConClave crea un usuario con contraseña real (bcrypt) para probar el login.
func sembrarConClave(t *testing.T, e *entorno, username, nombre, rol, clave string, debeCambiar bool) int {
	t.Helper()
	hash, err := auth.Hash(clave)
	if err != nil {
		t.Fatal(err)
	}
	var id int
	err = e.pool.QueryRow(context.Background(), `
		INSERT INTO usuario (username, email, hashed_password, nombre, rol, estado, debe_cambiar_clave)
		VALUES ($1, $2, $3, $4, $5, TRUE, $6) RETURNING id_usuario`,
		username, username+"@prueba.pe", hash, nombre, rol, debeCambiar).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// iniciarSesion hace POST /auth/login y devuelve el token (vacío si no entró) y la respuesta.
func (e *entorno) iniciarSesion(username, clave string) (string, *httptest.ResponseRecorder) {
	rec := pedir(e.mux, "POST", "/api/v1/auth/login", "", map[string]any{"username": username, "password": clave})
	var r struct {
		AccessToken string `json:"access_token"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &r)
	return r.AccessToken, rec
}

func (e *entorno) idDe(t *testing.T, username string) int {
	t.Helper()
	var id int
	if err := e.pool.QueryRow(context.Background(), `SELECT id_usuario FROM usuario WHERE username = $1`, username).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func cuerpoUsuario(username string) map[string]any {
	return map[string]any{"username": username, "email": username + "@lizar.pe", "password": "secreta123", "nombre": "Persona " + username, "rol": "logistica"}
}

func TestGestionDeUsuariosExigeToken(t *testing.T) {
	e := nuevoEntorno(t)
	rutas := []struct{ metodo, ruta string }{
		{"GET", "/api/v1/usuarios"}, {"POST", "/api/v1/usuarios"}, {"PUT", "/api/v1/usuarios/1"},
		{"POST", "/api/v1/usuarios/1/desactivar"}, {"POST", "/api/v1/usuarios/1/activar"},
		{"POST", "/api/v1/auth/cambiar-clave"},
	}
	for _, r := range rutas {
		if rec := pedir(e.mux, r.metodo, r.ruta, "", nil); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s sin token = %d, quiero 401", r.metodo, r.ruta, rec.Code)
		}
	}
}

func TestGestionDeUsuariosEsSoloDelAdmin(t *testing.T) {
	e := nuevoEntorno(t)
	otro := sembrarConClave(t, e, "otro", "Otro", "logistica", "secreta123", false)
	rutas := []struct {
		metodo, ruta string
		cuerpo       any
	}{
		{"GET", "/api/v1/usuarios", nil},
		{"POST", "/api/v1/usuarios", cuerpoUsuario("nuevo")},
		{"PUT", fmt.Sprintf("/api/v1/usuarios/%d", otro), map[string]any{"nombre": "Cambiado"}},
		{"POST", fmt.Sprintf("/api/v1/usuarios/%d/desactivar", otro), nil},
		{"POST", fmt.Sprintf("/api/v1/usuarios/%d/activar", otro), nil},
	}
	for _, r := range rutas {
		rec := pedir(e.mux, r.metodo, r.ruta, e.tokenLogistica, r.cuerpo)
		if rec.Code != http.StatusForbidden {
			t.Errorf("logistica %s %s = %d, quiero 403 (%s)", r.metodo, r.ruta, rec.Code, rec.Body)
		}
	}
	var n int
	var nombre string
	var estado bool
	_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM usuario WHERE username = 'nuevo'`).Scan(&n)
	_ = e.pool.QueryRow(context.Background(), `SELECT nombre, estado FROM usuario WHERE id_usuario = $1`, otro).Scan(&nombre, &estado)
	if n != 0 || nombre != "Otro" || !estado {
		t.Errorf("el 403 no debía cambiar nada: creados=%d nombre=%q estado=%v", n, nombre, estado)
	}
}

func TestListarUsuariosIncluyeInactivosYNoMuestraElHash(t *testing.T) {
	e := nuevoEntorno(t)
	sembrarUsuario(t, e.pool, "baja", "Usuario de Baja", "logistica", false)
	rec := pedir(e.mux, "GET", "/api/v1/usuarios", e.tokenAdmin, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /usuarios = %d %s", rec.Code, rec.Body)
	}
	var lista []usuarioJSON
	if err := json.Unmarshal(rec.Body.Bytes(), &lista); err != nil {
		t.Fatalf("la respuesta debía ser una lista: %s", rec.Body)
	}
	if len(lista) != 3 {
		t.Fatalf("usuarios = %d, quiero 3 (ana, admin_prueba, baja)", len(lista))
	}
	for i := 1; i < len(lista); i++ {
		if lista[i-1].ID >= lista[i].ID {
			t.Errorf("la lista debía ir ordenada por id: %v", lista)
		}
	}
	vistos := map[string]usuarioJSON{}
	for _, u := range lista {
		vistos[u.Username] = u
	}
	if baja := vistos["baja"]; baja.Estado || baja.Rol != "logistica" || baja.Nombre != "Usuario de Baja" {
		t.Errorf("baja = %+v", baja)
	}
	if strings.Contains(rec.Body.String(), "hashed_password") || strings.Contains(rec.Body.String(), "sin-contraseña") {
		t.Errorf("la lista no debe mostrar contraseñas: %s", rec.Body)
	}
}

func TestListarUsuariosSinNadieDevuelveListaVaciaNoNull(t *testing.T) {
	// Siempre hay al menos el admin que llama, pero la forma debe ser un arreglo JSON.
	e := nuevoEntorno(t)
	rec := pedir(e.mux, "GET", "/api/v1/usuarios", e.tokenAdmin, nil)
	if !strings.HasPrefix(strings.TrimSpace(rec.Body.String()), "[") {
		t.Errorf("cuerpo = %s", rec.Body)
	}
}

func TestCrearUsuario(t *testing.T) {
	e := nuevoEntorno(t)
	rec := pedir(e.mux, "POST", "/api/v1/usuarios", e.tokenAdmin, cuerpoUsuario("luis"))
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /usuarios = %d %s", rec.Code, rec.Body)
	}
	u := usuarioDe(t, rec)
	if u.ID == 0 || u.Username != "luis" || u.Email != "luis@lizar.pe" || u.Nombre != "Persona luis" || u.Rol != "logistica" || !u.Estado || !u.DebeCambiarClave {
		t.Errorf("usuario = %+v", u)
	}
	if strings.Contains(rec.Body.String(), "secreta123") || strings.Contains(rec.Body.String(), "hashed") {
		t.Errorf("la respuesta no debe incluir la contraseña: %s", rec.Body)
	}
	// La contraseña se guardó con hash y sirve para entrar; el usuario nace obligado a cambiarla.
	token, login := e.iniciarSesion("luis", "secreta123")
	if token == "" || !strings.Contains(login.Body.String(), `"debe_cambiar_clave":true`) {
		t.Fatalf("login del usuario nuevo: %d %s", login.Code, login.Body)
	}
}

func TestCrearUsuarioSinRolEsLogistica(t *testing.T) {
	e := nuevoEntorno(t)
	c := cuerpoUsuario("sinrol")
	delete(c, "rol")
	rec := pedir(e.mux, "POST", "/api/v1/usuarios", e.tokenAdmin, c)
	if rec.Code != http.StatusCreated || usuarioDe(t, rec).Rol != "logistica" {
		t.Fatalf("sin rol: %d %s", rec.Code, rec.Body)
	}
	c = cuerpoUsuario("otroadmin")
	c["rol"] = "admin"
	if rec := pedir(e.mux, "POST", "/api/v1/usuarios", e.tokenAdmin, c); rec.Code != http.StatusCreated || usuarioDe(t, rec).Rol != "admin" {
		t.Fatalf("admin: %d %s", rec.Code, rec.Body)
	}
}

func TestCrearUsuarioValida(t *testing.T) {
	e := nuevoEntorno(t)
	casos := map[string]func(map[string]any){
		"clave de 7":       func(c map[string]any) { c["password"] = "1234567" },
		"correo inválido":  func(c map[string]any) { c["email"] = "no-es-correo" },
		"sin nombre":       func(c map[string]any) { delete(c, "nombre") },
		"nombre en blanco": func(c map[string]any) { c["nombre"] = "   " },
		"rol inválido":     func(c map[string]any) { c["rol"] = "gerente" },
		"sin usuario":      func(c map[string]any) { delete(c, "username") },
	}
	for nombre, cambia := range casos {
		c := cuerpoUsuario("valida")
		cambia(c)
		rec := pedir(e.mux, "POST", "/api/v1/usuarios", e.tokenAdmin, c)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("%s = %d, quiero 422 (%s)", nombre, rec.Code, rec.Body)
		}
	}
	var n int
	_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM usuario WHERE username = 'valida'`).Scan(&n)
	if n != 0 {
		t.Error("un 422 no debía crear el usuario")
	}
	// Un cuerpo que no es JSON también es 422.
	if rec := pedir(e.mux, "POST", "/api/v1/usuarios", e.tokenAdmin, "texto"); rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("cuerpo no objeto = %d", rec.Code)
	}
}

func TestCrearUsuarioDuplicadoDa409(t *testing.T) {
	e := nuevoEntorno(t)
	if rec := pedir(e.mux, "POST", "/api/v1/usuarios", e.tokenAdmin, cuerpoUsuario("luis")); rec.Code != http.StatusCreated {
		t.Fatal(rec.Body)
	}
	mismoUsuario := cuerpoUsuario("luis")
	mismoUsuario["email"] = "otro@lizar.pe"
	rec := pedir(e.mux, "POST", "/api/v1/usuarios", e.tokenAdmin, mismoUsuario)
	if rec.Code != http.StatusConflict || !strings.Contains(detalleDe(rec), "luis") {
		t.Errorf("usuario repetido = %d %s", rec.Code, rec.Body)
	}
	mismoCorreo := cuerpoUsuario("maria")
	mismoCorreo["email"] = "luis@lizar.pe"
	rec = pedir(e.mux, "POST", "/api/v1/usuarios", e.tokenAdmin, mismoCorreo)
	if rec.Code != http.StatusConflict || !strings.Contains(detalleDe(rec), "luis@lizar.pe") {
		t.Errorf("correo repetido = %d %s", rec.Code, rec.Body)
	}
	// Los dos intentos fallidos no crearon nada.
	var n int
	_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM usuario WHERE username IN ('luis', 'maria')`).Scan(&n)
	if n != 1 {
		t.Errorf("usuarios luis/maria = %d, quiero 1", n)
	}
}

func TestEditarUsuario(t *testing.T) {
	e := nuevoEntorno(t)
	id := sembrarConClave(t, e, "luis", "Luis", "logistica", "secreta123", false)
	ruta := fmt.Sprintf("/api/v1/usuarios/%d", id)
	rec := pedir(e.mux, "PUT", ruta, e.tokenAdmin, map[string]any{"nombre": "Luis Rojas", "email": "luis.rojas@lizar.pe", "rol": "admin"})
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT = %d %s", rec.Code, rec.Body)
	}
	u := usuarioDe(t, rec)
	if u.ID != id || u.Username != "luis" || u.Nombre != "Luis Rojas" || u.Email != "luis.rojas@lizar.pe" || u.Rol != "admin" || !u.Estado || u.DebeCambiarClave {
		t.Errorf("usuario = %+v", u)
	}
	// Un cambio parcial deja el resto como estaba, y el nombre de usuario nunca cambia.
	rec = pedir(e.mux, "PUT", ruta, e.tokenAdmin, map[string]any{"nombre": "Luis R.", "username": "otro"})
	u = usuarioDe(t, rec)
	if rec.Code != http.StatusOK || u.Nombre != "Luis R." || u.Email != "luis.rojas@lizar.pe" || u.Rol != "admin" || u.Username != "luis" {
		t.Errorf("parcial: %d %+v", rec.Code, u)
	}
	// Guardar el mismo correo que ya tiene no es un duplicado.
	if rec := pedir(e.mux, "PUT", ruta, e.tokenAdmin, map[string]any{"email": "luis.rojas@lizar.pe"}); rec.Code != http.StatusOK {
		t.Errorf("mismo correo propio = %d %s", rec.Code, rec.Body)
	}
}

func TestEditarUsuarioErrores(t *testing.T) {
	e := nuevoEntorno(t)
	id := sembrarConClave(t, e, "luis", "Luis", "logistica", "secreta123", false)
	ruta := fmt.Sprintf("/api/v1/usuarios/%d", id)
	if rec := pedir(e.mux, "PUT", "/api/v1/usuarios/9999", e.tokenAdmin, map[string]any{"nombre": "X"}); rec.Code != http.StatusNotFound {
		t.Errorf("inexistente = %d", rec.Code)
	}
	if rec := pedir(e.mux, "PUT", "/api/v1/usuarios/abc", e.tokenAdmin, map[string]any{"nombre": "X"}); rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("id no numérico = %d", rec.Code)
	}
	// El correo de ana (logistica) ya está tomado.
	if rec := pedir(e.mux, "PUT", ruta, e.tokenAdmin, map[string]any{"email": "ana@prueba.pe"}); rec.Code != http.StatusConflict {
		t.Errorf("correo de otro = %d %s", rec.Code, rec.Body)
	}
	for nombre, cuerpo := range map[string]map[string]any{
		"nombre vacío": {"nombre": ""}, "correo inválido": {"email": "x"}, "rol inválido": {"rol": "root"}, "clave corta": {"password": "corta"},
	} {
		if rec := pedir(e.mux, "PUT", ruta, e.tokenAdmin, cuerpo); rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("%s = %d, quiero 422 (%s)", nombre, rec.Code, rec.Body)
		}
	}
	var nombre, email string
	_ = e.pool.QueryRow(context.Background(), `SELECT nombre, email FROM usuario WHERE id_usuario = $1`, id).Scan(&nombre, &email)
	if nombre != "Luis" || email != "luis@prueba.pe" {
		t.Errorf("los errores no debían cambiar nada: %q %q", nombre, email)
	}
}

func TestRestablecerClaveDesdeEditar(t *testing.T) {
	e := nuevoEntorno(t)
	id := sembrarConClave(t, e, "luis", "Luis", "logistica", "claveVieja1", false)
	rec := pedir(e.mux, "PUT", fmt.Sprintf("/api/v1/usuarios/%d", id), e.tokenAdmin, map[string]any{"password": "claveNueva1"})
	if rec.Code != http.StatusOK || !usuarioDe(t, rec).DebeCambiarClave {
		t.Fatalf("restablecer: %d %s", rec.Code, rec.Body)
	}
	if token, _ := e.iniciarSesion("luis", "claveVieja1"); token != "" {
		t.Error("la clave vieja ya no debía servir")
	}
	if token, _ := e.iniciarSesion("luis", "claveNueva1"); token == "" {
		t.Error("la clave nueva debía servir")
	}
}

func TestNoSePuedeQuitarElRolAlUltimoAdmin(t *testing.T) {
	e := nuevoEntorno(t)
	adminID := e.idDe(t, "admin_prueba")
	rec := pedir(e.mux, "PUT", fmt.Sprintf("/api/v1/usuarios/%d", adminID), e.tokenAdmin, map[string]any{"rol": "logistica"})
	if rec.Code != http.StatusConflict {
		t.Fatalf("degradar al único admin = %d %s", rec.Code, rec.Body)
	}
	var rol string
	_ = e.pool.QueryRow(context.Background(), `SELECT rol FROM usuario WHERE id_usuario = $1`, adminID).Scan(&rol)
	if rol != "admin" {
		t.Errorf("rol = %q, debía seguir admin", rol)
	}
	// Con otro admin activo ya se puede.
	sembrarUsuario(t, e.pool, "admin2", "Segundo Admin", "admin", true)
	if rec := pedir(e.mux, "PUT", fmt.Sprintf("/api/v1/usuarios/%d", adminID), e.tokenAdmin, map[string]any{"rol": "logistica"}); rec.Code != http.StatusOK {
		t.Errorf("degradar con otro admin activo = %d %s", rec.Code, rec.Body)
	}
}

func TestUnAdminInactivoNoCuentaParaElMinimo(t *testing.T) {
	e := nuevoEntorno(t)
	sembrarUsuario(t, e.pool, "admin_baja", "Admin de Baja", "admin", false)
	adminID := e.idDe(t, "admin_prueba")
	if rec := pedir(e.mux, "PUT", fmt.Sprintf("/api/v1/usuarios/%d", adminID), e.tokenAdmin, map[string]any{"rol": "logistica"}); rec.Code != http.StatusConflict {
		t.Errorf("el otro admin está inactivo: %d, quiero 409", rec.Code)
	}
}

func TestDesactivarYActivarUsuario(t *testing.T) {
	e := nuevoEntorno(t)
	id := sembrarConClave(t, e, "luis", "Luis", "logistica", "secreta123", false)
	tokenLuis, _ := e.iniciarSesion("luis", "secreta123")
	if rec := pedir(e.mux, "GET", "/gastos", tokenLuis, nil); rec.Code != http.StatusOK {
		t.Fatalf("antes de desactivar: %d", rec.Code)
	}
	rec := pedir(e.mux, "POST", fmt.Sprintf("/api/v1/usuarios/%d/desactivar", id), e.tokenAdmin, nil)
	if rec.Code != http.StatusOK || usuarioDe(t, rec).Estado {
		t.Fatalf("desactivar: %d %s", rec.Code, rec.Body)
	}
	// El usuario ya no entra ni con el token que tenía ni con su contraseña.
	if rec := pedir(e.mux, "GET", "/gastos", tokenLuis, nil); rec.Code != http.StatusUnauthorized || detalleDe(rec) != "Usuario inactivo" {
		t.Errorf("token de un desactivado: %d %q", rec.Code, detalleDe(rec))
	}
	if token, login := e.iniciarSesion("luis", "secreta123"); token != "" || login.Code != http.StatusUnauthorized {
		t.Errorf("login de un desactivado: %d", login.Code)
	}
	// Repetirlo no falla (queda desactivado).
	if rec := pedir(e.mux, "POST", fmt.Sprintf("/api/v1/usuarios/%d/desactivar", id), e.tokenAdmin, nil); rec.Code != http.StatusOK {
		t.Errorf("desactivar otra vez = %d", rec.Code)
	}
	rec = pedir(e.mux, "POST", fmt.Sprintf("/api/v1/usuarios/%d/activar", id), e.tokenAdmin, nil)
	if rec.Code != http.StatusOK || !usuarioDe(t, rec).Estado {
		t.Fatalf("activar: %d %s", rec.Code, rec.Body)
	}
	if token, _ := e.iniciarSesion("luis", "secreta123"); token == "" {
		t.Error("tras activarlo debía poder entrar")
	}
	if rec := pedir(e.mux, "POST", fmt.Sprintf("/api/v1/usuarios/%d/activar", id), e.tokenAdmin, nil); rec.Code != http.StatusOK {
		t.Errorf("activar otra vez = %d", rec.Code)
	}
}

func TestDesactivarYActivarInexistentesDan404(t *testing.T) {
	e := nuevoEntorno(t)
	for _, accion := range []string{"desactivar", "activar"} {
		if rec := pedir(e.mux, "POST", "/api/v1/usuarios/9999/"+accion, e.tokenAdmin, nil); rec.Code != http.StatusNotFound {
			t.Errorf("%s inexistente = %d", accion, rec.Code)
		}
		if rec := pedir(e.mux, "POST", "/api/v1/usuarios/abc/"+accion, e.tokenAdmin, nil); rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("%s con id no numérico = %d", accion, rec.Code)
		}
	}
}

func TestUnAdminNoPuedeDesactivarseASiMismo(t *testing.T) {
	e := nuevoEntorno(t)
	sembrarUsuario(t, e.pool, "admin2", "Segundo Admin", "admin", true) // aunque haya otro admin
	rec := pedir(e.mux, "POST", fmt.Sprintf("/api/v1/usuarios/%d/desactivar", e.idDe(t, "admin_prueba")), e.tokenAdmin, nil)
	if rec.Code != http.StatusConflict {
		t.Fatalf("desactivarse a sí mismo = %d %s", rec.Code, rec.Body)
	}
	var estado bool
	_ = e.pool.QueryRow(context.Background(), `SELECT estado FROM usuario WHERE username = 'admin_prueba'`).Scan(&estado)
	if !estado {
		t.Error("el admin debía seguir activo")
	}
}

// Dos admins que se desactivan el uno al otro a la vez no pueden dejar el sistema sin admin.
func TestDosAdminsQueSeDesactivanMutuamenteDejanUnoActivo(t *testing.T) {
	for intento := 0; intento < 5; intento++ {
		e := nuevoEntorno(t)
		idA := e.idDe(t, "admin_prueba")
		idB, tokenB := sembrarUsuario(t, e.pool, "admin_b", "Admin B", "admin", true)
		var wg sync.WaitGroup
		codigos := make([]int, 2)
		for i, par := range []struct {
			token string
			id    int
		}{{e.tokenAdmin, idB}, {tokenB, idA}} {
			wg.Add(1)
			go func() {
				defer wg.Done()
				codigos[i] = pedir(e.mux, "POST", fmt.Sprintf("/api/v1/usuarios/%d/desactivar", par.id), par.token, nil).Code
			}()
		}
		wg.Wait()
		var activos int
		_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM usuario WHERE rol = 'admin' AND estado`).Scan(&activos)
		if activos < 1 {
			t.Fatalf("intento %d: no quedó ningún admin activo (códigos %v)", intento, codigos)
		}
	}
}

// La comprobación de «queda otro admin» y el cambio van bajo un bloqueo (advisory) que comparten
// desactivar y editar. Aquí otra transacción lo retiene: la petición debe esperar y terminar al soltarlo.
func TestDesactivarEsperaElBloqueoDeAdmins(t *testing.T) {
	e := nuevoEntorno(t)
	id := sembrarConClave(t, e, "luis", "Luis", "logistica", "secreta123", false)
	ctx := context.Background()
	tx, err := e.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", int64(0x4C495A41)); err != nil { // bloqueoAdmins
		t.Fatal(err)
	}
	terminada := make(chan int, 1)
	go func() {
		terminada <- pedir(e.mux, "POST", fmt.Sprintf("/api/v1/usuarios/%d/desactivar", id), e.tokenAdmin, nil).Code
	}()
	select {
	case codigo := <-terminada:
		t.Fatalf("la petición terminó (%d) con el bloqueo retenido", codigo)
	case <-time.After(400 * time.Millisecond):
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case codigo := <-terminada:
		if codigo != http.StatusOK {
			t.Errorf("al soltar el bloqueo = %d", codigo)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("la petición no terminó tras soltar el bloqueo")
	}
}

func TestNingunUsuarioSeBorra(t *testing.T) {
	e := nuevoEntorno(t)
	id := e.idDe(t, "ana")
	if rec := pedir(e.mux, "DELETE", fmt.Sprintf("/api/v1/usuarios/%d", id), e.tokenAdmin, nil); rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("DELETE /usuarios/{id} = %d, quiero 405", rec.Code)
	}
}

func TestLoginYMeIncluyenDebeCambiarClave(t *testing.T) {
	e := nuevoEntorno(t)
	sembrarConClave(t, e, "nuevo", "Nuevo", "logistica", "inicial123", true)
	sembrarConClave(t, e, "viejo", "Viejo", "logistica", "inicial123", false)
	casos := map[string]bool{"nuevo": true, "viejo": false}
	for username, esperado := range casos {
		token, rec := e.iniciarSesion(username, "inicial123")
		var login struct {
			DebeCambiarClave bool `json:"debe_cambiar_clave"`
			Usuario          struct {
				DebeCambiarClave *bool `json:"debe_cambiar_clave"`
			} `json:"usuario"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &login); err != nil || rec.Code != http.StatusOK {
			t.Fatalf("%s login: %d %s", username, rec.Code, rec.Body)
		}
		if login.DebeCambiarClave != esperado || login.Usuario.DebeCambiarClave == nil || *login.Usuario.DebeCambiarClave != esperado {
			t.Errorf("%s: login trae debe_cambiar_clave = %v / %v, quiero %v", username, login.DebeCambiarClave, login.Usuario.DebeCambiarClave, esperado)
		}
		me := pedir(e.mux, "GET", "/api/v1/auth/me", token, nil)
		if me.Code != http.StatusOK || usuarioDe(t, me).DebeCambiarClave != esperado {
			t.Errorf("%s: /auth/me = %d %s", username, me.Code, me.Body)
		}
	}
}

func TestConLaClavePendienteSoloSePuedeVerElPerfilYCambiarla(t *testing.T) {
	e := nuevoEntorno(t)
	sembrarConClave(t, e, "nuevo", "Nuevo", "admin", "inicial123", true)
	token, _ := e.iniciarSesion("nuevo", "inicial123")
	bloqueadas := []struct{ metodo, ruta string }{
		{"GET", "/gastos"}, {"POST", "/gastos"}, {"GET", "/comprobantes"}, {"GET", "/reportes/resumen"},
		{"GET", "/api/v1/clientes"}, {"GET", "/api/v1/usuarios"}, {"POST", "/api/v1/usuarios"}, {"POST", "/api/v1/auth/register"},
	}
	for _, r := range bloqueadas {
		rec := pedir(e.mux, r.metodo, r.ruta, token, nil)
		if rec.Code != http.StatusForbidden || detalleDe(rec) != "Debes cambiar tu contraseña" {
			t.Errorf("%s %s = %d %q, quiero 403 «Debes cambiar tu contraseña»", r.metodo, r.ruta, rec.Code, detalleDe(rec))
		}
	}
	if rec := pedir(e.mux, "GET", "/api/v1/auth/me", token, nil); rec.Code != http.StatusOK {
		t.Errorf("/auth/me con clave pendiente = %d", rec.Code)
	}
	// Lo público sigue igual.
	if rec := pedir(e.mux, "GET", "/health", "", nil); rec.Code != http.StatusOK {
		t.Errorf("/health = %d", rec.Code)
	}
	// Sin token válido sigue siendo 401, no 403.
	if rec := pedir(e.mux, "GET", "/gastos", "basura", nil); rec.Code != http.StatusUnauthorized {
		t.Errorf("token inválido = %d", rec.Code)
	}
}

func TestCambiarClaveLiberaElAcceso(t *testing.T) {
	e := nuevoEntorno(t)
	sembrarConClave(t, e, "nuevo", "Nuevo", "logistica", "inicial123", true)
	token, _ := e.iniciarSesion("nuevo", "inicial123")
	if rec := pedir(e.mux, "GET", "/gastos", token, nil); rec.Code != http.StatusForbidden {
		t.Fatalf("antes de cambiarla: %d", rec.Code)
	}
	rec := pedir(e.mux, "POST", "/api/v1/auth/cambiar-clave", token, map[string]any{"clave_actual": "inicial123", "clave_nueva": "ClaveSegura9"})
	u := usuarioDe(t, rec)
	if rec.Code != http.StatusOK || u.DebeCambiarClave || u.Username != "nuevo" {
		t.Fatalf("cambiar clave: %d %s", rec.Code, rec.Body)
	}
	// El mismo token, sin volver a entrar, ya abre todo (la marca se lee de la base).
	if rec := pedir(e.mux, "GET", "/gastos", token, nil); rec.Code != http.StatusOK {
		t.Errorf("tras cambiarla: %d %s", rec.Code, rec.Body)
	}
	if t2, _ := e.iniciarSesion("nuevo", "inicial123"); t2 != "" {
		t.Error("la clave inicial ya no debía servir")
	}
	t3, login := e.iniciarSesion("nuevo", "ClaveSegura9")
	if t3 == "" || !strings.Contains(login.Body.String(), `"debe_cambiar_clave":false`) {
		t.Errorf("login con la clave nueva: %d %s", login.Code, login.Body)
	}
}

func TestCambiarClaveRechazaLoQueNoCorresponde(t *testing.T) {
	e := nuevoEntorno(t)
	sembrarConClave(t, e, "nuevo", "Nuevo", "logistica", "inicial123", true)
	token, _ := e.iniciarSesion("nuevo", "inicial123")
	casos := []struct {
		nombre string
		cuerpo any
		codigo int
	}{
		{"clave actual incorrecta", map[string]any{"clave_actual": "mala", "clave_nueva": "ClaveSegura9"}, http.StatusBadRequest},
		{"nueva igual a la actual", map[string]any{"clave_actual": "inicial123", "clave_nueva": "inicial123"}, http.StatusBadRequest},
		{"nueva de 7 caracteres", map[string]any{"clave_actual": "inicial123", "clave_nueva": "1234567"}, http.StatusUnprocessableEntity},
		{"falta la actual", map[string]any{"clave_nueva": "ClaveSegura9"}, http.StatusUnprocessableEntity},
		{"falta la nueva", map[string]any{"clave_actual": "inicial123"}, http.StatusUnprocessableEntity},
		{"cuerpo vacío", map[string]any{}, http.StatusUnprocessableEntity},
	}
	for _, c := range casos {
		rec := pedir(e.mux, "POST", "/api/v1/auth/cambiar-clave", token, c.cuerpo)
		if rec.Code != c.codigo {
			t.Errorf("%s = %d, quiero %d (%s)", c.nombre, rec.Code, c.codigo, rec.Body)
		}
	}
	// Nada de eso tocó la contraseña ni la marca.
	var debe bool
	_ = e.pool.QueryRow(context.Background(), `SELECT debe_cambiar_clave FROM usuario WHERE username = 'nuevo'`).Scan(&debe)
	if !debe {
		t.Error("los rechazos no debían quitar la marca")
	}
	if t2, _ := e.iniciarSesion("nuevo", "inicial123"); t2 == "" {
		t.Error("la clave original debía seguir sirviendo")
	}
}

func TestCambiarClaveTambienLoHaceQuienYaLaCambio(t *testing.T) {
	e := nuevoEntorno(t)
	sembrarConClave(t, e, "viejo", "Viejo", "admin", "inicial123", false)
	token, _ := e.iniciarSesion("viejo", "inicial123")
	rec := pedir(e.mux, "POST", "/api/v1/auth/cambiar-clave", token, map[string]any{"clave_actual": "inicial123", "clave_nueva": "OtraClave99"})
	if rec.Code != http.StatusOK {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	if t2, _ := e.iniciarSesion("viejo", "OtraClave99"); t2 == "" {
		t.Error("la clave nueva debía servir")
	}
}

func TestElAdminNuevoPuedeAdministrarTrasCambiarSuClave(t *testing.T) {
	e := nuevoEntorno(t)
	sembrarConClave(t, e, "jefe", "Jefe", "admin", "inicial123", true)
	token, _ := e.iniciarSesion("jefe", "inicial123")
	if rec := pedir(e.mux, "GET", "/api/v1/usuarios", token, nil); rec.Code != http.StatusForbidden {
		t.Fatalf("antes de cambiarla = %d", rec.Code)
	}
	pedir(e.mux, "POST", "/api/v1/auth/cambiar-clave", token, map[string]any{"clave_actual": "inicial123", "clave_nueva": "ClaveSegura9"})
	if rec := pedir(e.mux, "GET", "/api/v1/usuarios", token, nil); rec.Code != http.StatusOK {
		t.Errorf("después de cambiarla = %d %s", rec.Code, rec.Body)
	}
}
