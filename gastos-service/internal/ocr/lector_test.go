package ocr

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// programa escribe un ejecutable de prueba (sh) que hace las veces de pdftotext, pdftoppm o tesseract.
func programa(t *testing.T, nombre, cuerpo string) string {
	t.Helper()
	ruta := filepath.Join(t.TempDir(), nombre)
	if err := os.WriteFile(ruta, []byte("#!/bin/sh\n"+cuerpo+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return ruta
}

func herramientas() *Herramientas {
	return &Herramientas{PorEjecucion: 5 * time.Second, Total: 10 * time.Second}
}

func TestSinHerramientasNoDisponible(t *testing.T) {
	h := herramientas()
	for _, tipo := range []string{"imagen", "pdf"} {
		if l := h.Leer(t.Context(), "/no/importa", tipo); l.Estado != NoDisponible {
			t.Errorf("%s sin herramientas: %+v", tipo, l)
		}
	}
}

func TestPDFConTextoSeLeeConPdftotextYSeNormaliza(t *testing.T) {
	h := herramientas()
	h.Pdftotext = programa(t, "pdftotext", `[ "$1" = "-layout" ] && [ "$3" = "-" ] || exit 9
printf '%s\n' "FACTURA ELECTRÓNICA F001-77" "R.U.C.   20481608750" "FECHA: 30/09/2026" "TOTAL S/ 85.5O"`)
	l := h.Leer(t.Context(), "/tmp/x.pdf", "pdf")
	if l.Estado != Leido || l.Texto != "FACTURA ELECTRÓNICA F001-77\nR.U.C. 20481608750\nFECHA: 30/09/2026\nTOTAL S/ 85.50" {
		t.Fatalf("lectura = %+v", l)
	}
}

func TestLosNombresDeArchivoNoPasanPorUnaShell(t *testing.T) {
	h := herramientas()
	h.Pdftotext = programa(t, "pdftotext", `printf 'ARCHIVO RECIBIDO: %s\n' "$2"`)
	raro := "/tmp/comprobante; rm -rf ~ $(id) `uname`.pdf"
	l := h.Leer(t.Context(), raro, "pdf")
	if l.Estado != Leido || !strings.Contains(l.Texto, "comprobante; rm -rf ~ $(id) `uname`.pdf") {
		t.Fatalf("el nombre debía llegar intacto como un solo argumento: %+v", l)
	}
}

func TestPDFEscaneadoSinPdftoppmQuedaSinTexto(t *testing.T) {
	h := herramientas()
	h.Pdftotext = programa(t, "pdftotext", `printf '  \f\n'`)
	h.Tesseract = programa(t, "tesseract", `echo nunca`)
	if l := h.Leer(t.Context(), "/tmp/x.pdf", "pdf"); l.Estado != SinTexto {
		t.Fatalf("lectura = %+v", l)
	}
}

func TestPDFEscaneadoSinTesseractNoDisponible(t *testing.T) {
	h := herramientas()
	h.Pdftotext = programa(t, "pdftotext", `printf 'p. 1\n'`)
	h.Pdftoppm = programa(t, "pdftoppm", `exit 0`)
	if l := h.Leer(t.Context(), "/tmp/x.pdf", "pdf"); l.Estado != NoDisponible {
		t.Fatalf("lectura = %+v", l)
	}
}

func TestPDFEscaneadoSeLeePaginaPorPaginaConTesseract(t *testing.T) {
	h := herramientas()
	h.Pdftotext = programa(t, "pdftotext", `exit 0`)
	// pdftoppm -r 200 -png -f 1 -l 2 <pdf> <prefijo>: crea <prefijo>-1.png y <prefijo>-2.png
	h.Pdftoppm = programa(t, "pdftoppm", `[ "$1 $2 $3 $4 $5 $6 $7" = "-r 200 -png -f 1 -l 2" ] || exit 9
: > "$9-1.png"; : > "$9-2.png"`)
	h.Tesseract = programa(t, "tesseract", `[ "$2 $3 $4 $5 $6" = "stdout -l spa --psm 6" ] || exit 9
echo "PAGINA $(basename "$1") FACTURA ELECTRONICA TOTAL S/ 85.50"`)
	l := h.Leer(t.Context(), "/tmp/x.pdf", "pdf")
	want := "PAGINA pagina-1.png FACTURA ELECTRONICA TOTAL S/ 85.50\nPAGINA pagina-2.png FACTURA ELECTRONICA TOTAL S/ 85.50"
	if l.Estado != Leido || l.Texto != want {
		t.Fatalf("lectura = %+v", l)
	}
}

func TestImagenConTesseract(t *testing.T) {
	h := herramientas()
	h.Tesseract = programa(t, "tesseract", `printf '%s\n' "BOLETA DE VENTA" "IMPORTE TOTAL S/ 12.00" "Gracias"`)
	if l := h.Leer(t.Context(), "/tmp/foto.jpg", "imagen"); l.Estado != Leido || !strings.HasPrefix(l.Texto, "BOLETA DE VENTA") {
		t.Fatalf("lectura = %+v", l)
	}
	h.Tesseract = programa(t, "tesseract", `printf '|| ~~\n'`)
	if l := h.Leer(t.Context(), "/tmp/foto.jpg", "imagen"); l.Estado != SinTexto {
		t.Fatalf("foto sin texto: %+v", l)
	}
}

func TestProgramaQueFallaDaError(t *testing.T) {
	h := herramientas()
	h.Pdftotext = programa(t, "pdftotext", `echo "Syntax Error: Couldn't find trailer dictionary" >&2; exit 1`)
	if l := h.Leer(t.Context(), "/tmp/roto.pdf", "pdf"); l.Estado != Error {
		t.Fatalf("lectura = %+v", l)
	}
}

func TestCadaEjecucionTieneTiempoLimite(t *testing.T) {
	h := herramientas()
	h.PorEjecucion = 200 * time.Millisecond
	h.Pdftotext = programa(t, "pdftotext", `exec sleep 5`)
	inicio := time.Now()
	if l := h.Leer(t.Context(), "/tmp/x.pdf", "pdf"); l.Estado != Error {
		t.Fatalf("lectura = %+v", l)
	}
	if d := time.Since(inicio); d > 3*time.Second {
		t.Fatalf("tardó %v: el tiempo límite no cortó la ejecución", d)
	}
}

func TestLaLecturaCompletaTieneTiempoLimite(t *testing.T) {
	h := herramientas()
	h.Total = 300 * time.Millisecond
	h.Pdftotext = programa(t, "pdftotext", `exit 0`)
	h.Pdftoppm = programa(t, "pdftoppm", `: > "$9-1.png"; : > "$9-2.png"`)
	// Cada página tarda menos que el límite por ejecución, pero las dos juntas pasan el total
	h.Tesseract = programa(t, "tesseract", `sleep 0.2; echo FACTURA ELECTRONICA TOTAL S/ 85.50`)
	if l := h.Leer(t.Context(), "/tmp/x.pdf", "pdf"); l.Estado != Error {
		t.Fatalf("lectura = %+v", l)
	}
}

func TestLecturaCanceladaDaError(t *testing.T) {
	h := herramientas()
	h.Tesseract = programa(t, "tesseract", `exec sleep 5`)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if l := h.Leer(ctx, "/tmp/foto.jpg", "imagen"); l.Estado != Error {
		t.Fatalf("lectura = %+v", l)
	}
}

func TestTesseractSoloCuentaConEspanol(t *testing.T) {
	h := herramientas()
	con := programa(t, "tesseract", `printf 'List of available languages in "/opt/tessdata/" (2):\neng\nspa\n' >&2`)
	sin := programa(t, "tesseract", `printf 'List of available languages (1):\neng\n'`)
	if !h.tieneEspanol(con) {
		t.Error("con spa debía contar")
	}
	if h.tieneEspanol(sin) {
		t.Error("sin spa no debía contar")
	}
}

func TestResumen(t *testing.T) {
	h := &Herramientas{Pdftotext: "/bin/pdftotext", Pdftoppm: "/bin/pdftoppm"}
	if got := h.Resumen(); got != "pdftotext sí, pdftoppm sí, tesseract (spa) no" {
		t.Fatalf("Resumen = %q", got)
	}
}
