package ocr

import (
	"context"
	"strings"
	"testing"
	"time"
)

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
	h.Pdftotext = programa(t, "pdftotext", "factura")
	l := h.Leer(t.Context(), "/tmp/x.pdf", "pdf")
	if l.Estado != Leido || l.Texto != "FACTURA ELECTRÓNICA F001-77\nR.U.C. 20481608750\nFECHA: 30/09/2026\nTOTAL S/ 85.50" {
		t.Fatalf("lectura = %+v", l)
	}
}

func TestLosNombresDeArchivoNoPasanPorUnaShell(t *testing.T) {
	h := herramientas()
	h.Pdftotext = programa(t, "pdftotext", "argumento")
	raro := "/tmp/comprobante; rm -rf ~ $(id) `uname`.pdf"
	l := h.Leer(t.Context(), raro, "pdf")
	if l.Estado != Leido || !strings.Contains(l.Texto, "comprobante; rm -rf ~ $(id) `uname`.pdf") {
		t.Fatalf("el nombre debía llegar intacto como un solo argumento: %+v", l)
	}
}

func TestPDFEscaneadoSinPdftoppmQuedaSinTexto(t *testing.T) {
	h := herramientas()
	h.Pdftotext = programa(t, "pdftotext", "blanco")
	h.Tesseract = programa(t, "tesseract", "vacio")
	if l := h.Leer(t.Context(), "/tmp/x.pdf", "pdf"); l.Estado != SinTexto {
		t.Fatalf("lectura = %+v", l)
	}
}

func TestPDFEscaneadoSinTesseractNoDisponible(t *testing.T) {
	h := herramientas()
	h.Pdftotext = programa(t, "pdftotext", "blanco")
	h.Pdftoppm = programa(t, "pdftoppm", "vacio")
	if l := h.Leer(t.Context(), "/tmp/x.pdf", "pdf"); l.Estado != NoDisponible {
		t.Fatalf("lectura = %+v", l)
	}
}

func TestPDFEscaneadoSeLeePaginaPorPaginaConTesseract(t *testing.T) {
	h := herramientas()
	h.Pdftotext = programa(t, "pdftotext", "vacio")
	// pdftoppm -r 200 -png -f 1 -l 2 <pdf> <prefijo>: crea <prefijo>-1.png y <prefijo>-2.png
	h.Pdftoppm = programa(t, "pdftoppm", "paginas")
	h.Tesseract = programa(t, "tesseract", "pagina")
	l := h.Leer(t.Context(), "/tmp/x.pdf", "pdf")
	want := "PAGINA pagina-1.png FACTURA ELECTRONICA TOTAL S/ 85.50\nPAGINA pagina-2.png FACTURA ELECTRONICA TOTAL S/ 85.50"
	if l.Estado != Leido || l.Texto != want {
		t.Fatalf("lectura = %+v", l)
	}
}

func TestImagenConTesseract(t *testing.T) {
	h := herramientas()
	h.Tesseract = programa(t, "tesseract", "boleta")
	if l := h.Leer(t.Context(), "/tmp/foto.jpg", "imagen"); l.Estado != Leido || !strings.HasPrefix(l.Texto, "BOLETA DE VENTA") {
		t.Fatalf("lectura = %+v", l)
	}
	h.Tesseract = programa(t, "tesseract", "ruido")
	if l := h.Leer(t.Context(), "/tmp/foto.jpg", "imagen"); l.Estado != SinTexto {
		t.Fatalf("foto sin texto: %+v", l)
	}
}

func TestProgramaQueFallaDaError(t *testing.T) {
	h := herramientas()
	h.Pdftotext = programa(t, "pdftotext", "error")
	if l := h.Leer(t.Context(), "/tmp/roto.pdf", "pdf"); l.Estado != Error {
		t.Fatalf("lectura = %+v", l)
	}
}

func TestCadaEjecucionTieneTiempoLimite(t *testing.T) {
	h := herramientas()
	h.PorEjecucion = 200 * time.Millisecond
	h.Pdftotext = programa(t, "pdftotext", "lento")
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
	h.Pdftotext = programa(t, "pdftotext", "vacio")
	h.Pdftoppm = programa(t, "pdftoppm", "paginas")
	// Cada página tarda menos que el límite por ejecución, pero las dos juntas pasan el total
	h.Tesseract = programa(t, "tesseract", "pagina-lenta")
	if l := h.Leer(t.Context(), "/tmp/x.pdf", "pdf"); l.Estado != Error {
		t.Fatalf("lectura = %+v", l)
	}
}

func TestLecturaCanceladaDaError(t *testing.T) {
	h := herramientas()
	h.Tesseract = programa(t, "tesseract", "lento")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if l := h.Leer(ctx, "/tmp/foto.jpg", "imagen"); l.Estado != Error {
		t.Fatalf("lectura = %+v", l)
	}
}

func TestTesseractSoloCuentaConEspanol(t *testing.T) {
	h := herramientas()
	con := programa(t, "tesseract", "idiomas-spa")
	sin := programa(t, "tesseract", "idiomas-eng")
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
