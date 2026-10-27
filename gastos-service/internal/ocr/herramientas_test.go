package ocr

import (
	"image"
	"image/color"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"

	"github.com/signintech/gopdf"
)

// Pruebas con las herramientas reales: cada una se omite si su herramienta no está instalada.

func requerir(t *testing.T, h *Herramientas, pdftotext, pdftoppm, tesseract bool) {
	t.Helper()
	for _, falta := range []struct {
		pide   bool
		ruta   string
		nombre string
	}{{pdftotext, h.Pdftotext, "pdftotext"}, {pdftoppm, h.Pdftoppm, "pdftoppm"}, {tesseract, h.Tesseract, "tesseract con idioma spa"}} {
		if falta.pide && falta.ruta == "" {
			t.Skip(falta.nombre + " no está instalado: se omite la prueba")
		}
	}
}

var lineasFactura = []string{
	"FACTURA ELECTRÓNICA F001-000777",
	"R.U.C. 20481608750",
	"FECHA DE EMISIÓN: 30/09/2026",
	"IMPORTE TOTAL S/ 85.50",
}

// pdfDe arma un PDF A4 con texto grande (o, si img no es nil, solo con esa imagen: un «escaneado»).
func pdfDe(t *testing.T, lineas []string, img image.Image) string {
	t.Helper()
	pdf := &gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
	if err := pdf.AddTTFFont("overpass", filepath.Join("..", "reportes", "fuentes", "overpass-400.ttf")); err != nil {
		t.Fatal(err)
	}
	pdf.AddPage()
	if img != nil {
		if err := pdf.ImageFrom(img, 0, 0, &gopdf.Rect{W: 595, H: 842}); err != nil {
			t.Fatal(err)
		}
	}
	if err := pdf.SetFont("overpass", "", 22); err != nil {
		t.Fatal(err)
	}
	for i, l := range lineas {
		pdf.SetXY(40, 60+float64(i)*40)
		if err := pdf.Text(l); err != nil {
			t.Fatal(err)
		}
	}
	ruta := filepath.Join(t.TempDir(), "comprobante.pdf")
	if err := pdf.WritePdf(ruta); err != nil {
		t.Fatal(err)
	}
	return ruta
}

func blanca() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 400, 560))
	for x := range 400 {
		for y := range 560 {
			img.Set(x, y, color.White)
		}
	}
	return img
}

func comprobarFactura(t *testing.T, l Lectura) {
	t.Helper()
	if l.Estado != Leido {
		t.Fatalf("lectura = %+v", l)
	}
	d := Extraer(l.Texto)
	if d.Monto.Valor != 85.5 || !slices.Contains(d.Rucs, RucEmpresa) || d.Tipo.Valor != "factura" || d.Fecha.Valor != "2026-09-30" {
		t.Fatalf("datos = %+v (texto %q)", d, l.Texto)
	}
}

func TestPdftotextConUnPDFGenerado(t *testing.T) {
	h := NuevoLector()
	requerir(t, h, true, false, false)
	comprobarFactura(t, h.Leer(t.Context(), pdfDe(t, lineasFactura, nil), "pdf"))
}

func TestPdftotextConLasMuestrasDeUploads(t *testing.T) {
	h := NuevoLector()
	requerir(t, h, true, false, false)
	muestras, _ := filepath.Glob(filepath.Join("..", "..", "uploads", "*.pdf"))
	if len(muestras) == 0 {
		t.Skip("no hay PDF de muestra en uploads/")
	}
	for _, m := range muestras {
		if l := h.Leer(t.Context(), m, "pdf"); l.Estado == Error || l.Estado == NoDisponible {
			t.Errorf("%s: %s", filepath.Base(m), l.Estado)
		}
	}
}

func TestPDFEscaneadoRealSinTesseract(t *testing.T) {
	h := NuevoLector()
	requerir(t, h, true, true, false)
	h.Tesseract = ""
	escaneado := pdfDe(t, nil, blanca())
	if l := h.Leer(t.Context(), escaneado, "pdf"); l.Estado != NoDisponible {
		t.Fatalf("sin tesseract: %+v", l)
	}
	h.Pdftoppm = ""
	if l := h.Leer(t.Context(), escaneado, "pdf"); l.Estado != SinTexto {
		t.Fatalf("sin pdftoppm: %+v", l)
	}
}

func TestTesseractConUnaImagenGenerada(t *testing.T) {
	h := NuevoLector()
	requerir(t, h, false, true, true)
	dir := t.TempDir()
	salida, err := exec.Command(h.Pdftoppm, "-r", "200", "-png", "-singlefile", pdfDe(t, lineasFactura, nil), filepath.Join(dir, "foto")).CombinedOutput()
	if err != nil {
		t.Fatalf("pdftoppm: %v %s", err, salida)
	}
	comprobarFactura(t, h.Leer(t.Context(), filepath.Join(dir, "foto.png"), "imagen"))
}
