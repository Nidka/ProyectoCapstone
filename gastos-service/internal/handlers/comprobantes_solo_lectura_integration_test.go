package handlers_test

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"

	"os"
)

// RNF-5: el comprobante guardado queda en solo lectura (0444).
func TestComprobanteGuardadoQuedaEnSoloLectura(t *testing.T) {
	e := nuevoEntorno(t)
	id := e.gastoRegistrado(t)
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
	ruta := filepath.Join(e.uploads, filepath.Base(c.ArchivoURL))
	info, err := os.Stat(ruta)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o444 {
		t.Fatalf("modo del comprobante = %v, se esperaba -r--r--r--", info.Mode().Perm())
	}
}
