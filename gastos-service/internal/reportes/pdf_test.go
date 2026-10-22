package reportes

import (
	"bytes"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/signintech/gopdf"
)

var paginaPDF = regexp.MustCompile(`/Type\s*/Page[^s]`)

func paginas(t *testing.T, pdf []byte) int {
	t.Helper()
	if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatalf("no es un PDF: %q", pdf[:min(len(pdf), 16)])
	}
	return len(paginaPDF.FindAll(pdf, -1))
}

// resumenDePrueba arma un resumen con n gastos y 12 meses de datos.
func resumenDePrueba(n int) *Resumen {
	r := &Resumen{Desde: "2025-11-01", Hasta: "2026-10-31"}
	estados := []string{"Registrado", "Pagado", "Sustentado"}
	for i := 0; i < 12; i++ {
		r.PorMes = append(r.PorMes, Mes{Mes: fmt.Sprintf("%04d-%02d", 2025+(10+i)/12, (10+i)%12+1), Registrado: 100, Pagado: 200, Sustentado: float64(300 + i*50)})
	}
	for _, e := range estados {
		r.PorEstado = append(r.PorEstado, PorEstado{Estado: e, Cantidad: n / 3, Monto: 1000, Porcentaje: 33.3})
	}
	for p := 1; p <= 10; p++ {
		fila := FilaRanking{Puesto: p, Etiqueta: fmt.Sprintf("Cliente con un nombre bastante largo número %d SAC", p), Cantidad: 11 - p, Monto: float64(1100 - p*100), Porcentaje: float64(11 - p)}
		r.PorCliente = append(r.PorCliente, fila)
		r.PorUnidad = append(r.PorUnidad, fila)
		r.PorConductor = append(r.PorConductor, fila)
	}
	for i := 0; i < n; i++ {
		r.Detalle = append(r.Detalle, FilaDetalle{
			ID: i + 1, Fecha: time.Date(2026, 1, 1+i%28, 0, 0, 0, 0, time.UTC), Cliente: "Agroexportadora Norte del Perú SAC",
			ViajeID: i % 7, Unidad: "T1A-123", Conductor: "José Ñique Peña", Estado: estados[i%3], Monto: 85.5,
		})
	}
	r.Indicadores = Indicadores{Total: 3000, Cantidad: n, Promedio: 85.5, Sustentado: 1000, Pendiente: 2000, PorcentajeSustentado: 33.3}
	return r
}

func TestPDFEstadisticoConDatosOcupaDosPaginas(t *testing.T) {
	pdf, err := PDFEstadistico(resumenDePrueba(30), ahoraPrueba)
	if err != nil {
		t.Fatal(err)
	}
	if n := paginas(t, pdf); n != 2 {
		t.Fatalf("páginas = %d, se esperaban 2", n)
	}
}

func TestPDFEstadisticoCon24Meses(t *testing.T) {
	r := resumenDePrueba(30)
	r.PorMes = nil
	for i := 0; i < 24; i++ {
		r.PorMes = append(r.PorMes, Mes{Mes: fmt.Sprintf("%04d-%02d", 2024+i/12, i%12+1), Registrado: 10, Pagado: 20, Sustentado: 30, Total: 60})
	}
	pdf, err := PDFEstadistico(r, ahoraPrueba)
	if err != nil {
		t.Fatal(err)
	}
	if n := paginas(t, pdf); n != 2 {
		t.Fatalf("páginas = %d, se esperaban 2", n)
	}
}

func TestPDFDetalleRepartePorPaginas(t *testing.T) {
	// 27 filas por página: 60 gastos ocupan 3 páginas, con los totales al final de la tercera.
	pdf, err := PDFDetalle(resumenDePrueba(60), ahoraPrueba)
	if err != nil {
		t.Fatal(err)
	}
	if n := paginas(t, pdf); n != 3 {
		t.Fatalf("páginas = %d, se esperaban 3", n)
	}
}

func TestPDFSinGastosEsUnaPagina(t *testing.T) {
	vacio := &Resumen{Desde: "2026-10-01", Hasta: "2026-10-31", PorMes: []Mes{}, PorEstado: []PorEstado{}}
	for nombre, generar := range map[string]func(*Resumen, time.Time) ([]byte, error){"estadístico": PDFEstadistico, "detalle": PDFDetalle} {
		pdf, err := generar(vacio, ahoraPrueba)
		if err != nil {
			t.Fatalf("%s: %v", nombre, err)
		}
		if n := paginas(t, pdf); n != 1 {
			t.Errorf("%s vacío: páginas = %d, se esperaba 1", nombre, n)
		}
	}
}

func TestEscalaAgradable(t *testing.T) {
	casos := []struct{ maximo, tope, paso float64 }{
		{0, 4, 1},
		{7950, 8000, 2000},
		{8100, 10000, 2500},
		{120, 200, 50},
	}
	for _, c := range casos {
		if tope, paso := escalaAgradable(c.maximo); tope != c.tope || paso != c.paso {
			t.Errorf("escalaAgradable(%v) = %v, %v; se esperaba %v, %v", c.maximo, tope, paso, c.tope, c.paso)
		}
	}
}

func TestMaximoMensualUsaLoQueSeDibuja(t *testing.T) {
	// La escala sale de la suma de los segmentos apilados, aunque Total no coincida
	meses := []Mes{{Mes: "2026-01", Registrado: 100, Pagado: 200, Sustentado: 300}, {Mes: "2026-02", Total: 50}}
	if got := maximoMensual(meses); got != 600 {
		t.Fatalf("maximoMensual = %v, se esperaba 600", got)
	}
}

func TestIndicadorGrandeSeAjustaSinRecortarse(t *testing.T) {
	d, err := nuevoDocumento(gopdf.PageSizeA4, "Prueba", &Resumen{Desde: "2026-01-01", Hasta: "2026-01-31"}, ahoraPrueba)
	if err != nil {
		t.Fatal(err)
	}
	// Ancho útil del valor en cada uno de los cuatro recuadros de indicadores
	const ancho = 101.0
	grande := FormatoMonto(12345678.9)
	tamano := d.tamanoQueCabe(grande, ancho, true, 14, 8)
	d.fuente(true, tamano)
	if w, _ := d.pdf.MeasureTextWidth(grande); tamano >= 14 || w > ancho {
		t.Fatalf("«%s» a %v pt mide %v pt; debía achicarse hasta caber en %v pt", grande, tamano, w, ancho)
	}
	if got := d.tamanoQueCabe("S/ 85.50", ancho, true, 14, 8); got != 14 {
		t.Fatalf("un monto corto no debía achicarse: %v", got)
	}
}
