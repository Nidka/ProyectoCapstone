package handlers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gastos-service/internal/auth"
	"gastos-service/internal/handlers"
)

// rutasProtegidas son rutas reales de cada módulo: todas exigen un token válido.
var rutasProtegidas = []struct{ metodo, ruta string }{
	{"GET", "/gastos"}, {"POST", "/gastos"}, {"GET", "/gastos/1"}, {"GET", "/gastos/exportar.xlsx"},
	{"POST", "/gastos/1/comprobantes"}, {"GET", "/comprobantes"},
	{"POST", "/comprobantes/1/aprobar"}, {"POST", "/comprobantes/1/rechazar"}, {"GET", "/comprobantes/1/lectura"},
	{"GET", "/clientes/1/checklist"}, {"PUT", "/clientes/1/checklist"}, {"DELETE", "/clientes/1/checklist"},
	{"GET", "/reportes/resumen"}, {"GET", "/reportes/estadistico.pdf"}, {"GET", "/reportes/detalle.pdf"},
	{"GET", "/api/v1/clientes"}, {"POST", "/api/v1/clientes"}, {"DELETE", "/api/v1/clientes/1"},
	{"GET", "/api/v1/auth/me"}, {"POST", "/api/v1/auth/register"},
	{"POST", "/api/v1/auth/logout"}, {"GET", "/gastos/1/historial"},
	{"GET", "/notificaciones"}, {"POST", "/notificaciones/1/leida"},
	{"GET", "/ruta-que-no-existe"},
}

func TestSinTokenTodaRutaProtegidaDa401(t *testing.T) {
	e := nuevoEntorno(t)
	for _, r := range rutasProtegidas {
		rec := pedir(e.mux, r.metodo, r.ruta, "", nil)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s sin token = %d, quiero 401", r.metodo, r.ruta, rec.Code)
		}
	}
	if d := detalleDe(pedir(e.mux, "GET", "/gastos", "", nil)); d != "No autenticado" {
		t.Errorf("detalle = %q", d)
	}
}

func TestTokensInvalidosDan401(t *testing.T) {
	e := nuevoEntorno(t)
	vencido, err := (&auth.Servicio{Clave: []byte(claveDePrueba), Ahora: func() time.Time { return time.Now().Add(-9 * time.Hour) }}).CrearToken(auth.Usuario{ID: 1, Username: "ana", Rol: "logistica"})
	if err != nil {
		t.Fatal(err)
	}
	otraClave, _ := (&auth.Servicio{Clave: []byte("otra-clave")}).CrearToken(auth.Usuario{ID: 1, Username: "ana", Rol: "logistica"})
	casos := map[string]string{"basura": "no.es.token", "vencido": vencido, "firmado con otra clave": otraClave}
	for nombre, token := range casos {
		rec := pedir(e.mux, "GET", "/gastos", token, nil)
		if rec.Code != http.StatusUnauthorized || detalleDe(rec) != "Token inválido" {
			t.Errorf("%s: %d %q", nombre, rec.Code, detalleDe(rec))
		}
	}
	// Una cabecera que no es Bearer tampoco vale.
	req := pedirConCabecera(e.mux, "GET", "/gastos", "Basic YW5hOnNlY3JldGE=")
	if req.Code != http.StatusUnauthorized {
		t.Errorf("Authorization Basic = %d", req.Code)
	}
}

func TestUsuarioDesactivadoOInexistenteDa401(t *testing.T) {
	e := nuevoEntorno(t)
	// Inexistente: el token es válido (firma y vigencia), pero nadie se llama así.
	fantasma := tokenDe(t, auth.Usuario{ID: 999, Username: "fantasma", Rol: "logistica"})
	rec := pedir(e.mux, "GET", "/gastos", fantasma, nil)
	if rec.Code != http.StatusUnauthorized || detalleDe(rec) != "Usuario no encontrado" {
		t.Errorf("inexistente: %d %q", rec.Code, detalleDe(rec))
	}
	// Desactivado desde el inicio.
	_, tokenInactivo := sembrarUsuario(t, e.pool, "baja", "Usuario de Baja", "logistica", false)
	rec = pedir(e.mux, "GET", "/gastos", tokenInactivo, nil)
	if rec.Code != http.StatusUnauthorized || detalleDe(rec) != "Usuario inactivo" {
		t.Errorf("desactivado: %d %q", rec.Code, detalleDe(rec))
	}
	// Desactivado con el token ya emitido: el estado se consulta en la base en cada petición.
	if rec := pedir(e.mux, "GET", "/gastos", e.tokenLogistica, nil); rec.Code != http.StatusOK {
		t.Fatalf("antes de desactivar: %d", rec.Code)
	}
	if _, err := e.pool.Exec(context.Background(), `UPDATE usuario SET estado = FALSE WHERE username = 'ana'`); err != nil {
		t.Fatal(err)
	}
	if rec := pedir(e.mux, "GET", "/gastos", e.tokenLogistica, nil); rec.Code != http.StatusUnauthorized {
		t.Errorf("tras desactivar: %d, quiero 401", rec.Code)
	}
}

func TestElRolSeTomaDeLaBaseNoDelToken(t *testing.T) {
	e := nuevoEntorno(t)
	_, comprobante := e.comprobantePendiente(t)
	// Ana es logistica en la base; un token que afirme «admin» no le quita el permiso...
	tokenFalso := tokenDe(t, auth.Usuario{ID: 1, Username: "ana", Rol: "admin"})
	rec := pedir(e.mux, "POST", fmt.Sprintf("/comprobantes/%d/aprobar", comprobante), tokenFalso, map[string]any{"checklist": completo()})
	if rec.Code != http.StatusOK {
		t.Fatalf("aprobar con rol de token distinto: %d %s", rec.Code, rec.Body)
	}
	// ...y el admin no gana permisos de logística con un token que diga «logistica».
	tokenAdminDisfrazado := tokenDe(t, auth.Usuario{ID: 2, Username: "admin_prueba", Rol: "logistica"})
	rec = pedir(e.mux, "PUT", "/clientes/1/checklist", tokenAdminDisfrazado, map[string]any{"criterios": []criterioJSON{{Texto: "x"}}})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("admin disfrazado: %d", rec.Code)
	}
}

func TestRutasPublicas(t *testing.T) {
	e := nuevoEntorno(t)
	if rec := pedir(e.mux, "GET", "/health", "", nil); rec.Code != http.StatusOK {
		t.Errorf("/health sin token = %d", rec.Code)
	}
	// Login sin token: con credenciales malas responde 401 de credenciales, no de autenticación.
	rec := pedir(e.mux, "POST", "/api/v1/auth/login", "", map[string]any{"username": "ana", "password": "mala"})
	if rec.Code != http.StatusUnauthorized || detalleDe(rec) != "Credenciales inválidas" {
		t.Errorf("login con clave mala: %d %q", rec.Code, detalleDe(rec))
	}
	// Login bueno: el token que entrega abre las rutas protegidas.
	hash, err := auth.Hash("secreta123")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.pool.Exec(context.Background(), `UPDATE usuario SET hashed_password = $1 WHERE username = 'ana'`, hash); err != nil {
		t.Fatal(err)
	}
	rec = pedir(e.mux, "POST", "/api/v1/auth/login", "", map[string]any{"username": "ana", "password": "secreta123"})
	var login struct {
		AccessToken string `json:"access_token"`
		Usuario     struct {
			Rol string `json:"rol"`
		} `json:"usuario"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &login); err != nil || rec.Code != http.StatusOK || login.AccessToken == "" || login.Usuario.Rol != "logistica" {
		t.Fatalf("login: %d %s", rec.Code, rec.Body)
	}
	if rec := pedir(e.mux, "GET", "/gastos", login.AccessToken, nil); rec.Code != http.StatusOK {
		t.Errorf("GET /gastos con el token del login = %d", rec.Code)
	}
	if rec := pedir(e.mux, "GET", "/api/v1/auth/me", login.AccessToken, nil); rec.Code != http.StatusOK {
		t.Errorf("GET /auth/me = %d", rec.Code)
	}
}

func TestOptionsSinTokenResponde204(t *testing.T) {
	e := nuevoEntorno(t)
	for _, ruta := range []string{"/gastos", "/comprobantes/1/aprobar", "/api/v1/auth/register", "/api/v1/clientes/1"} {
		rec := pedir(e.mux, http.MethodOptions, ruta, "", nil)
		if rec.Code != http.StatusNoContent {
			t.Errorf("OPTIONS %s = %d, quiero 204", ruta, rec.Code)
		}
	}
}

func TestLaCabeceraDeCORSYaNoAceptaLasCabecerasDeRolDeAntes(t *testing.T) {
	e := nuevoEntorno(t)
	rec := pedir(e.mux, http.MethodOptions, "/gastos", "", nil)
	permitidas := rec.Header().Get("Access-Control-Allow-Headers")
	if !strings.Contains(permitidas, "Authorization") || strings.Contains(permitidas, "X-User") {
		t.Errorf("Access-Control-Allow-Headers = %q", permitidas)
	}
}

func TestLasCabecerasXUserYaNoDanAcceso(t *testing.T) {
	e := nuevoEntorno(t)
	_, comprobante := e.comprobantePendiente(t)
	req := pedirConCabecera(e.mux, "POST", fmt.Sprintf("/comprobantes/%d/aprobar", comprobante), "", "X-User-Role", "logistica", "X-User-Name", "Ana Quispe")
	if req.Code != http.StatusUnauthorized {
		t.Errorf("X-User-Role sin token = %d, quiero 401", req.Code)
	}
	// Con token de admin, X-User-Role: logistica no cambia el 403.
	rec := pedirConCabecera(e.mux, "POST", fmt.Sprintf("/comprobantes/%d/aprobar", comprobante), "Bearer "+e.tokenAdmin, "X-User-Role", "logistica")
	if rec.Code != http.StatusForbidden {
		t.Errorf("admin con X-User-Role = %d, quiero 403", rec.Code)
	}
}

func TestUploadsSeSirveSinToken(t *testing.T) {
	e := nuevoEntorno(t)
	if err := os.WriteFile(filepath.Join(e.uploads, "boleta.jpg"), []byte("contenido"), 0o644); err != nil {
		t.Fatal(err)
	}
	rec := pedir(e.mux, "GET", "/uploads/boleta.jpg", "", nil)
	if rec.Code != http.StatusOK || rec.Body.String() != "contenido" {
		t.Errorf("/uploads sin token = %d %q", rec.Code, rec.Body)
	}
}

func TestAdminNoPuedeEscribirGastosNiComprobantesNiChecklist(t *testing.T) {
	e := nuevoEntorno(t)
	gasto, comprobante := e.comprobantePendiente(t)
	cliente := e.clienteDe(t, gasto)
	casos := []struct {
		metodo, ruta string
		cuerpo       any
	}{
		{"POST", "/gastos", map[string]any{"viaje_id": 1, "cliente_id": cliente, "monto": 10}},
		{"POST", fmt.Sprintf("/gastos/%d/comprobantes", gasto), nil},
		{"POST", fmt.Sprintf("/comprobantes/%d/aprobar", comprobante), map[string]any{"checklist": completo()}},
		{"POST", fmt.Sprintf("/comprobantes/%d/rechazar", comprobante), map[string]any{"motivo_rechazo": "borroso"}},
		{"PUT", fmt.Sprintf("/clientes/%d/checklist", cliente), map[string]any{"criterios": []criterioJSON{{Texto: "Uno"}}}},
		{"DELETE", fmt.Sprintf("/clientes/%d/checklist", cliente), nil},
	}
	for _, c := range casos {
		if rec := pedir(e.mux, c.metodo, c.ruta, e.tokenAdmin, c.cuerpo); rec.Code != http.StatusForbidden {
			t.Errorf("admin %s %s = %d, quiero 403 (%s)", c.metodo, c.ruta, rec.Code, rec.Body)
		}
	}
	if g, c := e.estados(t, gasto, comprobante); g != "Pagado" || c != "Pendiente" {
		t.Errorf("el 403 no debía cambiar nada: gasto %q, comprobante %q", g, c)
	}
	var n int
	_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM gasto`).Scan(&n)
	if n != 1 {
		t.Errorf("gastos = %d, el admin no debía crear ninguno", n)
	}
}

func TestAdminPuedeLeerTodo(t *testing.T) {
	e := nuevoEntorno(t)
	gasto, comprobante := e.comprobantePendiente(t)
	cliente := e.clienteDe(t, gasto)
	rutas := []string{"/gastos", fmt.Sprintf("/gastos/%d", gasto), "/gastos/exportar.xlsx", "/comprobantes",
		fmt.Sprintf("/comprobantes/%d/lectura", comprobante), fmt.Sprintf("/clientes/%d/checklist", cliente),
		"/reportes/resumen", "/api/v1/auth/me"}
	for _, ruta := range rutas {
		if rec := pedir(e.mux, "GET", ruta, e.tokenAdmin, nil); rec.Code >= 400 {
			t.Errorf("admin GET %s = %d (%s)", ruta, rec.Code, rec.Body)
		}
	}
}

func TestLogisticaRegistraGastosYAdjuntaComprobantes(t *testing.T) {
	e := nuevoEntorno(t)
	gasto, _ := e.comprobantePendiente(t)
	cliente := e.clienteDe(t, gasto)
	var viaje int
	if err := e.pool.QueryRow(context.Background(), `SELECT viaje_id FROM gasto WHERE id = $1`, gasto).Scan(&viaje); err != nil {
		t.Fatal(err)
	}
	// El nombre del gasto sale de la sesión, aunque el cuerpo traiga otro.
	rec := pedir(e.mux, "POST", "/gastos", e.tokenLogistica, map[string]any{"viaje_id": viaje, "cliente_id": cliente, "monto": 10, "creado_por": "Otra Persona"})
	var creado struct {
		ID        int    `json:"id"`
		CreadoPor string `json:"creado_por"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &creado); err != nil || rec.Code != http.StatusCreated || creado.CreadoPor != "Ana Quispe" {
		t.Fatalf("POST /gastos: %d %s", rec.Code, rec.Body)
	}
	if rec := e.subirComprobante(t, creado.ID, "boleta.png", pngDe(2048)); rec.Code != http.StatusCreated {
		t.Errorf("adjuntar como logistica = %d %s", rec.Code, rec.Body)
	}
}

func TestAprobarYRechazarGuardanElNombreDeLaSesion(t *testing.T) {
	e := nuevoEntorno(t)
	_, aprobable := e.comprobantePendiente(t)
	_, rechazable := e.comprobantePendiente(t)
	// El nombre que venga en el cuerpo se ignora.
	if rec := e.post(t, fmt.Sprintf("/comprobantes/%d/aprobar", aprobable), map[string]any{"checklist": completo(), "aprobado_por": "Intruso"}); rec.Code != http.StatusOK {
		t.Fatalf("aprobar: %d %s", rec.Code, rec.Body)
	}
	if rec := e.post(t, fmt.Sprintf("/comprobantes/%d/rechazar", rechazable), map[string]any{"motivo_rechazo": "borroso", "aprobado_por": "Intruso"}); rec.Code != http.StatusOK {
		t.Fatalf("rechazar: %d %s", rec.Code, rec.Body)
	}
	for _, id := range []int{aprobable, rechazable} {
		var revisor string
		if err := e.pool.QueryRow(context.Background(), `SELECT aprobado_por FROM comprobante WHERE id = $1`, id).Scan(&revisor); err != nil || revisor != "Ana Quispe" {
			t.Errorf("comprobante %d: aprobado_por = %q (%v)", id, revisor, err)
		}
	}
}

func TestRegistrarUsuariosEsSoloDelAdmin(t *testing.T) {
	e := nuevoEntorno(t)
	nuevo := func(usuario string, rol any) map[string]any {
		c := map[string]any{"username": usuario, "email": usuario + "@lizar.pe", "password": "secreta1", "nombre": "Persona " + usuario}
		if rol != nil {
			c["rol"] = rol
		}
		return c
	}
	if rec := pedir(e.mux, "POST", "/api/v1/auth/register", e.tokenLogistica, nuevo("uno", "logistica")); rec.Code != http.StatusForbidden {
		t.Errorf("logistica registrando = %d, quiero 403 (%s)", rec.Code, rec.Body)
	}
	var n int
	_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM usuario WHERE username = 'uno'`).Scan(&n)
	if n != 0 {
		t.Error("el 403 no debía crear el usuario")
	}
	// Por defecto el rol es logistica.
	rec := pedir(e.mux, "POST", "/api/v1/auth/register", e.tokenAdmin, nuevo("dos", nil))
	var creado struct {
		Rol string `json:"rol"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &creado); err != nil || rec.Code != http.StatusCreated || creado.Rol != "logistica" {
		t.Fatalf("admin registrando sin rol: %d %s", rec.Code, rec.Body)
	}
	if rec := pedir(e.mux, "POST", "/api/v1/auth/register", e.tokenAdmin, nuevo("tres", "admin")); rec.Code != http.StatusCreated {
		t.Errorf("admin creando admin = %d", rec.Code)
	}
	// Los roles válidos son solo admin y logistica.
	for _, rol := range []string{"operador", "gerente"} {
		rec := pedir(e.mux, "POST", "/api/v1/auth/register", e.tokenAdmin, nuevo("cuatro", rol))
		if rec.Code != http.StatusUnprocessableEntity || !strings.Contains(rec.Body.String(), "'admin'") || !strings.Contains(rec.Body.String(), "'logistica'") {
			t.Errorf("rol %q = %d, quiero 422 que nombre admin y logistica: %s", rol, rec.Code, rec.Body)
		}
	}
	// Sin token: 401 antes que 403.
	if rec := pedir(e.mux, "POST", "/api/v1/auth/register", "", nuevo("cinco", nil)); rec.Code != http.StatusUnauthorized {
		t.Errorf("sin token = %d", rec.Code)
	}
}

// El middleware que usa main.go es el mismo que arma Servidor: sin él, Register deja todo abierto.
func TestServidorEnvuelveElMuxConElMiddleware(t *testing.T) {
	e := nuevoEntorno(t)
	h := handlers.New(e.pool, configDePrueba(t.TempDir()))
	if rec := pedir(h.Servidor(), "GET", "/gastos", "", nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("Servidor() sin token = %d", rec.Code)
	}
}
