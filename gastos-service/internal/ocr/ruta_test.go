package ocr

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
)

func TestRutaSeguraDentroDeUploads(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "gasto-1.pdf"), []byte("%PDF"), 0o644); err != nil {
		t.Fatal(err)
	}
	ruta, err := RutaSegura(dir, "/uploads/gasto-1.pdf")
	if err != nil {
		t.Fatal(err)
	}
	real, _ := filepath.EvalSymlinks(filepath.Join(dir, "gasto-1.pdf"))
	if ruta != real {
		t.Fatalf("ruta = %q, se esperaba %q", ruta, real)
	}
}

func TestRutaSeguraRechazaLoQueSaleDeUploads(t *testing.T) {
	raiz := t.TempDir()
	dir := filepath.Join(raiz, "uploads")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	secreto := filepath.Join(raiz, "secreto.txt")
	if err := os.WriteFile(secreto, []byte("clave"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, url := range []string{
		"/uploads/../secreto.txt",
		"/uploads/../../../../etc/passwd",
		"/uploads/",
		"/uploads/..",
		"/otra/secreto.txt",
		secreto,
		"http://otro.servidor/uploads/x.pdf",
	} {
		if _, err := RutaSegura(dir, url); !errors.Is(err, ErrRutaFuera) {
			t.Errorf("%q: se esperaba ErrRutaFuera, llegó %v", url, err)
		}
	}
}

func TestRutaSeguraRechazaEnlaceFueraDeUploads(t *testing.T) {
	raiz := t.TempDir()
	dir := filepath.Join(raiz, "uploads")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	secreto := filepath.Join(raiz, "secreto.txt")
	if err := os.WriteFile(secreto, []byte("clave"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secreto, filepath.Join(dir, "enlace.pdf")); err != nil {
		// ERROR_PRIVILEGE_NOT_HELD: Windows requiere modo desarrollador o
		// el privilegio de crear enlaces. Los otros rechazos se prueban aparte.
		if runtime.GOOS == "windows" && errors.Is(err, syscall.Errno(1314)) {
			t.Skip("Windows no permite crear enlaces simbólicos sin el privilegio correspondiente")
		}
		t.Fatal(err)
	}
	if _, err := RutaSegura(dir, "/uploads/enlace.pdf"); !errors.Is(err, ErrRutaFuera) {
		t.Fatalf("se esperaba ErrRutaFuera, llegó %v", err)
	}
}

func TestRutaSeguraArchivoInexistenteOCarpeta(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "carpeta"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, url := range []string{"/uploads/no-existe.pdf", "/uploads/carpeta"} {
		if _, err := RutaSegura(dir, url); err == nil || errors.Is(err, ErrRutaFuera) {
			t.Errorf("%q: se esperaba un error de archivo, llegó %v", url, err)
		}
	}
}
