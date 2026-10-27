package auth

import (
	"os"
	"strings"
	"testing"
	"time"

	"gastos-service/internal/pruebas"

	"github.com/golang-jwt/jwt/v5"
)

// Credenciales emitidas por la API de Python (python-jose y bcrypt 4.2.1) con la clave por defecto.
const (
	tokenDePython = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJhZG1pbiIsInJvbCI6ImFkbWluIiwiaWQiOjEsImV4cCI6NDEwMjQ0NDgwMH0.xlL_fAko0xnKCrwjzZfEnHLrPSmrUl2BdTaYzo99oZY"
	hashDePython  = "$2b$12$XB21nbr7C1dvI4aRdexRMuKDzyVBhkDhbIQJoPLFAweJzAuQmMWMW" // «admin123»
	hash80xDePy   = "$2b$12$mTvnxMsig9wJW4uhbJ41Pecj8wMh..Ke7In5SDkd6euDbD7YZrQTO" // 80 letras «x»
)

// claveDePython es la clave pública con que la API de Python firmó tokenDePython. Ya no es una clave
// válida del servicio: solo sirve para comprobar el formato del token.
const claveDePython = "lizar-srl-fleet-manager-secret-2025"

func servicioDePrueba() *Servicio {
	return &Servicio{Clave: []byte(claveDePython), Ahora: func() time.Time { return time.Date(2026, 10, 22, 9, 0, 0, 0, time.UTC) }}
}

func TestSeAceptaElHashDePython(t *testing.T) {
	if !Verificar("admin123", hashDePython) {
		t.Fatal("un hash bcrypt de Python debía validar su contraseña")
	}
	if Verificar("otra", hashDePython) {
		t.Fatal("una contraseña distinta no debía validar")
	}
}

func TestContrasenasLargasSeRecortanA72BytesComoEnPython(t *testing.T) {
	if !Verificar(strings.Repeat("x", 80), hash80xDePy) || !Verificar(strings.Repeat("x", 72), hash80xDePy) {
		t.Fatal("bcrypt solo mira 72 bytes: ambas contraseñas debían validar contra el hash de Python")
	}
	h, err := Hash(strings.Repeat("y", 100))
	if err != nil || !Verificar(strings.Repeat("y", 100), h) {
		t.Fatalf("hash propio de una contraseña larga: %v", err)
	}
	if !strings.HasPrefix(h, "$2a$12$") && !strings.HasPrefix(h, "$2b$12$") {
		t.Fatalf("el hash debía usar bcrypt con costo 12: %s", h[:7])
	}
}

func TestSeAceptaElTokenDePython(t *testing.T) {
	usuario, err := servicioDePrueba().UsuarioDeToken(tokenDePython)
	if err != nil || usuario != "admin" {
		t.Fatalf("UsuarioDeToken = %q, %v", usuario, err)
	}
}

func TestTokenPropioLlevaLosMismosDatos(t *testing.T) {
	s := servicioDePrueba()
	token, err := s.CrearToken(Usuario{ID: 3, Username: "ana", Rol: "logistica"})
	if err != nil {
		t.Fatal(err)
	}
	if usuario, err := s.UsuarioDeToken(token); err != nil || usuario != "ana" {
		t.Fatalf("UsuarioDeToken = %q, %v", usuario, err)
	}
}

func TestTokensInvalidos(t *testing.T) {
	s := servicioDePrueba()
	vencido, _ := (&Servicio{Clave: s.Clave, Ahora: func() time.Time { return time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC) }}).CrearToken(Usuario{Username: "ana"})
	otraClave, _ := (&Servicio{Clave: []byte("otra"), Ahora: s.Ahora}).CrearToken(Usuario{Username: "ana"})
	for nombre, tok := range map[string]string{"vacío": "", "basura": "no.es.token", "vencido": vencido, "otra clave": otraClave} {
		if _, err := s.UsuarioDeToken(tok); err == nil {
			t.Errorf("%s: el token debía rechazarse", nombre)
		}
	}
}

func TestTokenSinVencimientoSeRechaza(t *testing.T) {
	s := servicioDePrueba()
	// Un token firmado con la clave correcta pero sin exp valdría para siempre.
	sinExp, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "admin"}).SignedString(s.Clave)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.UsuarioDeToken(sinExp); err == nil {
		t.Fatal("un token sin exp debía rechazarse")
	}
}

func TestSembrarAdminSoloSiNoHayUsuarios(t *testing.T) {
	pool := pruebas.NuevoEsquemaCompleto(t)
	s := &Servicio{DB: pool, Clave: []byte(claveDePython)}
	if err := s.SembrarAdmin(t.Context()); err != nil {
		t.Fatal(err)
	}
	admin, err := s.PorUsername(t.Context(), "admin")
	if err != nil || admin == nil || admin.Rol != "admin" || !admin.Estado || !Verificar("admin123", admin.Hash) {
		t.Fatalf("admin sembrado = %+v, %v", admin, err)
	}
	if !admin.DebeCambiarClave {
		t.Error("el admin sembrado trae una clave conocida: debe quedar obligado a cambiarla")
	}
	if err := s.SembrarAdmin(t.Context()); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := pool.QueryRow(t.Context(), "SELECT count(*) FROM usuario").Scan(&n); err != nil || n != 1 {
		t.Fatalf("usuarios = %d, %v; con uno ya creado no debía sembrar otro", n, err)
	}
}

func TestSembrarAdminQuedaEnLaBitacora(t *testing.T) {
	pool := pruebas.NuevoEsquemaCompleto(t)
	s := &Servicio{DB: pool, Clave: []byte(claveDePython)}
	if err := s.SembrarAdmin(t.Context()); err != nil {
		t.Fatal(err)
	}
	var usuario, accion, entidad string
	var idUsuario *int
	var idEntidad int
	err := pool.QueryRow(t.Context(), "SELECT usuario, accion, entidad, id_usuario, id_entidad FROM bitacora").Scan(&usuario, &accion, &entidad, &idUsuario, &idEntidad)
	if err != nil || usuario != "Sistema" || accion != "usuario.crear" || entidad != "usuario" || idUsuario != nil || idEntidad == 0 {
		t.Fatalf("bitácora = %q %q %q %v %d, %v", usuario, accion, entidad, idUsuario, idEntidad, err)
	}
}

func TestMigracion0006PasaOperadorALogisticaYEsIdempotente(t *testing.T) {
	pool := pruebas.NuevoEsquemaCompleto(t)
	ctx := t.Context()
	// Un usuario de antes de la migración: rol «operador» y el DEFAULT viejo.
	if _, err := pool.Exec(ctx, `INSERT INTO usuario (username, email, hashed_password, nombre, rol)
		VALUES ('vieja', 'vieja@lizar.pe', 'x', 'Usuaria Vieja', 'operador'), ('jefe', 'jefe@lizar.pe', 'x', 'Jefe', 'admin')`); err != nil {
		t.Fatal(err)
	}
	sql, err := os.ReadFile("../../migrations/0006_roles.sql")
	if err != nil {
		t.Fatal(err)
	}
	for vez := 1; vez <= 2; vez++ { // el runner reaplica todas las migraciones en cada arranque
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("aplicación %d: %v", vez, err)
		}
		roles := map[string]string{}
		rows, err := pool.Query(ctx, `SELECT username, rol FROM usuario WHERE username IN ('vieja', 'jefe')`)
		if err != nil {
			t.Fatal(err)
		}
		for rows.Next() {
			var u, r string
			if err := rows.Scan(&u, &r); err != nil {
				t.Fatal(err)
			}
			roles[u] = r
		}
		rows.Close()
		if roles["vieja"] != "logistica" || roles["jefe"] != "admin" {
			t.Fatalf("aplicación %d: roles = %v", vez, roles)
		}
	}
	var rol string
	if _, err := pool.Exec(ctx, `INSERT INTO usuario (username, email, hashed_password, nombre) VALUES ('sinrol', 'sinrol@lizar.pe', 'x', 'Sin Rol')`); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT rol FROM usuario WHERE username = 'sinrol'`).Scan(&rol); err != nil || rol != "logistica" {
		t.Fatalf("rol por defecto = %q, %v", rol, err)
	}
}

func TestMigracion0007AgregaDebeCambiarClaveYEsIdempotente(t *testing.T) {
	pool := pruebas.NuevoEsquemaCompleto(t)
	ctx := t.Context()
	// Estado de antes de la migración: sin la columna y con un usuario ya existente.
	if _, err := pool.Exec(ctx, `ALTER TABLE usuario DROP COLUMN debe_cambiar_clave`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO usuario (username, email, hashed_password, nombre, rol)
		VALUES ('previo', 'previo@lizar.pe', 'x', 'Usuario Previo', 'logistica')`); err != nil {
		t.Fatal(err)
	}
	sql, err := os.ReadFile("../../migrations/0007_usuario_gestion.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, string(sql)); err != nil {
		t.Fatal(err)
	}
	var debe bool
	if err := pool.QueryRow(ctx, `SELECT debe_cambiar_clave FROM usuario WHERE username = 'previo'`).Scan(&debe); err != nil || !debe {
		t.Fatalf("los usuarios existentes quedan obligados a cambiar la clave: %v, %v", debe, err)
	}
	// Quien ya la cambió no vuelve a quedar obligado cuando el runner reaplica la migración.
	if _, err := pool.Exec(ctx, `UPDATE usuario SET debe_cambiar_clave = FALSE WHERE username = 'previo'`); err != nil {
		t.Fatal(err)
	}
	for vez := 2; vez <= 3; vez++ {
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("aplicación %d: %v", vez, err)
		}
		if err := pool.QueryRow(ctx, `SELECT debe_cambiar_clave FROM usuario WHERE username = 'previo'`).Scan(&debe); err != nil || debe {
			t.Fatalf("aplicación %d: reaplicar no debía tocar la marca (%v, %v)", vez, debe, err)
		}
	}
	// Un usuario nuevo nace obligado a cambiarla.
	if _, err := pool.Exec(ctx, `INSERT INTO usuario (username, email, hashed_password, nombre) VALUES ('nuevo', 'nuevo@lizar.pe', 'x', 'Nuevo')`); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT debe_cambiar_clave FROM usuario WHERE username = 'nuevo'`).Scan(&debe); err != nil || !debe {
		t.Fatalf("valor por defecto = %v, %v", debe, err)
	}
}
