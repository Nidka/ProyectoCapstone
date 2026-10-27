package handlers_test

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// /uploads se sirve sin token (el navegador carga los comprobantes con <img> y <a href>), así que
// al menos no debe ofrecer el listado de la carpeta a quien no tiene cuenta.
func TestUploadsNoListaLaCarpeta(t *testing.T) {
	e := nuevoEntorno(t)
	if err := os.WriteFile(filepath.Join(e.uploads, "gasto-34-secreto.pdf"), []byte("contenido"), 0o444); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(e.uploads, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, ruta := range []string{"/uploads/", "/uploads/sub/", "/uploads/sub"} {
		for _, token := range []string{"", e.tokenLogistica} {
			rec := pedir(e.mux, "GET", ruta, token, nil)
			if rec.Code != http.StatusNotFound {
				t.Errorf("GET %s = %d, quiero 404", ruta, rec.Code)
			}
			if strings.Contains(rec.Body.String(), "gasto-34-secreto.pdf") {
				t.Errorf("GET %s lista los archivos: %s", ruta, rec.Body)
			}
		}
	}
	// El archivo pedido por su nombre sigue sirviéndose.
	if rec := pedir(e.mux, "GET", "/uploads/gasto-34-secreto.pdf", "", nil); rec.Code != http.StatusOK || rec.Body.String() != "contenido" {
		t.Errorf("el archivo debía servirse: %d %q", rec.Code, rec.Body)
	}
}

// El nombre del comprobante lleva una parte aleatoria: el id del gasto y la hora no bastan para
// deducirlo.
func TestElNombreDelComprobanteNoSeDeduce(t *testing.T) {
	e := nuevoEntorno(t)
	id := e.gastoRegistrado(t)
	nombres := map[string]bool{}
	for i := 0; i < 2; i++ {
		rec := e.subirComprobante(t, id, "boleta.png", pngDe(2048))
		if rec.Code != http.StatusCreated {
			t.Fatalf("código %d: %s", rec.Code, rec.Body)
		}
		var c struct {
			ArchivoURL string `json:"archivo_url"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &c); err != nil {
			t.Fatal(err)
		}
		nombre := filepath.Base(c.ArchivoURL)
		if !regexp.MustCompile(`^gasto-\d+-[0-9a-f]{32}\.png$`).MatchString(nombre) {
			t.Fatalf("nombre = %q, quiero gasto-<id>-<32 hex>.png", nombre)
		}
		nombres[nombre] = true
	}
	if len(nombres) != 2 {
		t.Fatalf("dos subidas dieron el mismo nombre: %v", nombres)
	}
}

// Un token firmado con la clave pública de versiones anteriores no abre nada, ni siquiera para el
// usuario admin: el rol sale de la base, pero solo si el token lo firmó quien conoce JWT_SECRET.
func TestUnTokenFirmadoConLaClavePublicaNoAbreNada(t *testing.T) {
	e := nuevoEntorno(t)
	forjado, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "admin_prueba", "exp": time.Now().Add(time.Minute).Unix(),
	}).SignedString([]byte("lizar-srl-fleet-manager-secret-2025"))
	if err != nil {
		t.Fatal(err)
	}
	if rec := pedir(e.mux, "GET", "/api/v1/auth/me", forjado, nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("token forjado = %d, quiero 401", rec.Code)
	}
	sinExp, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "admin_prueba"}).SignedString([]byte(claveDePrueba))
	if rec := pedir(e.mux, "GET", "/api/v1/auth/me", sinExp, nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("token sin vencimiento = %d, quiero 401", rec.Code)
	}
}
