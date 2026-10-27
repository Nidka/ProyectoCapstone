package registro

import (
	"bytes"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// guardarSalida devuelve el destino actual del log estándar y lo restaura al terminar la prueba.
func guardarSalida(t *testing.T) {
	t.Helper()
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
}

func TestConfigurarCreaElDirectorioYEscribeEnArchivoYConsola(t *testing.T) {
	guardarSalida(t)
	ruta := filepath.Join(t.TempDir(), "logs", "interno", "errores.log")
	var consola bytes.Buffer

	cerrar, err := Configurar(ruta, &consola)
	if err != nil {
		t.Fatal(err)
	}
	log.Printf("crud: %v", io.ErrUnexpectedEOF)
	if err := cerrar(); err != nil {
		t.Fatal(err)
	}

	contenido, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(contenido), "crud: unexpected EOF") {
		t.Fatalf("el archivo no tiene el error: %q", contenido)
	}
	if !strings.Contains(consola.String(), "crud: unexpected EOF") {
		t.Fatalf("la consola no tiene el error: %q", consola.String())
	}
}

func TestConfigurarAgregaSinBorrarLoAnterior(t *testing.T) {
	guardarSalida(t)
	ruta := filepath.Join(t.TempDir(), "errores.log")
	if err := os.WriteFile(ruta, []byte("anterior\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cerrar, err := Configurar(ruta, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	log.Print("nuevo")
	_ = cerrar()

	contenido, _ := os.ReadFile(ruta)
	if !strings.HasPrefix(string(contenido), "anterior\n") || !strings.Contains(string(contenido), "nuevo") {
		t.Fatalf("contenido inesperado: %q", contenido)
	}
}

func TestConfigurarSinRutaSoloUsaLaConsola(t *testing.T) {
	guardarSalida(t)
	var consola bytes.Buffer
	cerrar, err := Configurar("", &consola)
	if err != nil {
		t.Fatal(err)
	}
	log.Print("solo consola")
	if err := cerrar(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(consola.String(), "solo consola") {
		t.Fatalf("consola: %q", consola.String())
	}
}

func TestConfigurarFallaSiNoPuedeCrearElArchivo(t *testing.T) {
	guardarSalida(t)
	// Un archivo común donde debería haber un directorio.
	archivo := filepath.Join(t.TempDir(), "plano")
	if err := os.WriteFile(archivo, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Configurar(filepath.Join(archivo, "errores.log"), io.Discard); err == nil {
		t.Fatal("se esperaba un error")
	}
}
