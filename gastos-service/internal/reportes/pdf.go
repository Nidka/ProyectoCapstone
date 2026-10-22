package reportes

import (
	_ "embed"
	"strconv"
	"time"

	"github.com/signintech/gopdf"
)

// Overpass (SIL Open Font License 1.1, ver fuentes/OFL.txt): la misma letra del frontend.
//
//go:embed fuentes/overpass-400.ttf
var fuenteRegular []byte

//go:embed fuentes/overpass-700.ttf
var fuenteNegrita []byte

type color struct{ r, g, b uint8 }

// Colores de los tokens del frontend (styles/tokens.css).
var (
	colorRojo      = color{225, 29, 72}
	colorRojoTinta = color{180, 18, 58}
	colorTinta     = color{30, 33, 36}
	colorTinta2    = color{74, 80, 86}
	colorTinta3    = color{101, 107, 113}
	colorLinea     = color{226, 227, 224}
	colorPanel2    = color{240, 241, 238}
)

// ordenEstados es el orden del proceso; los colores son los de cada estado en la app.
var ordenEstados = []string{"Registrado", "Pagado", "Sustentado"}

var coloresEstado = map[string]color{
	"Registrado": {138, 144, 150},
	"Pagado":     {37, 99, 235},
	"Sustentado": {21, 128, 61},
}

const (
	margen     = 40.0
	textoVacio = "No hay gastos registrados en este período."
)

// documento envuelve gopdf con un cursor vertical, la cabecera de LIZAR y el pie
// con «Página N de M». Guarda el primer error de dibujo y lo devuelve al final.
type documento struct {
	pdf     *gopdf.GoPdf
	ancho   float64
	alto    float64
	titulo  string
	periodo string
	emision string
	y       float64
	err     error
	// alNuevaPagina dibuja lo que se repite debajo de la cabecera (el encabezado de una tabla).
	alNuevaPagina func()
}

func nuevoDocumento(pagina *gopdf.Rect, titulo string, r *Resumen, ahora time.Time) (*documento, error) {
	pdf := &gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *pagina})
	if err := pdf.AddTTFFontData("overpass", fuenteRegular); err != nil {
		return nil, err
	}
	if err := pdf.AddTTFFontData("overpass-negrita", fuenteNegrita); err != nil {
		return nil, err
	}
	d := &documento{
		pdf:     pdf,
		ancho:   pagina.W,
		alto:    pagina.H,
		titulo:  titulo,
		periodo: "Período: " + fechaTexto(r.Desde) + " – " + fechaTexto(r.Hasta),
		emision: "Emitido: " + FormatoEmision(ahora),
	}
	d.nuevaPagina()
	return d, nil
}

func fechaTexto(iso string) string {
	t, err := time.Parse(formatoISO, iso)
	if err != nil {
		return iso
	}
	return FormatoFecha(t)
}

func (d *documento) anchoUtil() float64 { return d.ancho - 2*margen }

func (d *documento) nuevaPagina() {
	d.pdf.AddPage()
	d.cabecera()
	if d.alNuevaPagina != nil {
		d.alNuevaPagina()
	}
}

// asegurar pasa a una página nueva si lo que sigue no cabe en la actual.
func (d *documento) asegurar(alto float64) {
	if d.y+alto > d.alto-margen {
		d.nuevaPagina()
	}
}

func (d *documento) cabecera() {
	d.rect(0, 0, d.ancho, 6, colorRojo)
	d.texto(margen, 22, d.anchoUtil(), 12, "LIZAR SRL · Fleet Manager Pro", true, 9, colorRojoTinta, gopdf.Left)
	d.texto(margen, 36, d.anchoUtil(), 22, d.titulo, true, 18, colorTinta, gopdf.Left)
	d.texto(margen, 60, d.anchoUtil(), 12, d.periodo+"   ·   "+d.emision, false, 9, colorTinta2, gopdf.Left)
	d.linea(margen, 80, d.ancho-margen, 80, colorLinea)
	d.y = 96
}

func (d *documento) pie(numero, total int) {
	y := d.alto - 28
	d.texto(margen, y, d.anchoUtil(), 10, "LIZAR SRL · Trujillo, Perú", false, 8, colorTinta3, gopdf.Left)
	d.texto(margen, y, d.anchoUtil(), 10, "Página "+strconv.Itoa(numero)+" de "+strconv.Itoa(total), false, 8, colorTinta3, gopdf.Right)
}

// finalizar escribe el pie en cada página (ya se sabe el total) y devuelve el PDF.
func (d *documento) finalizar() ([]byte, error) {
	total := d.pdf.GetNumberOfPages()
	for i := 1; i <= total; i++ {
		if err := d.pdf.SetPage(i); err != nil {
			return nil, err
		}
		d.pie(i, total)
	}
	if d.err != nil {
		return nil, d.err
	}
	return d.pdf.GetBytesPdf(), nil
}

func (d *documento) fuente(negrita bool, tamano float64) {
	familia := "overpass"
	if negrita {
		familia = "overpass-negrita"
	}
	if err := d.pdf.SetFont(familia, "", tamano); err != nil && d.err == nil {
		d.err = err
	}
}

// texto escribe una línea dentro de un recuadro, centrada en vertical y recortada con «…» si no cabe.
func (d *documento) texto(x, y, ancho, alto float64, txt string, negrita bool, tamano float64, c color, alineacion int) {
	d.fuente(negrita, tamano)
	d.pdf.SetTextColor(c.r, c.g, c.b)
	d.pdf.SetXY(x, y)
	if err := d.pdf.CellWithOption(&gopdf.Rect{W: ancho, H: alto}, d.recortar(txt, ancho), gopdf.CellOption{Align: alineacion | gopdf.Middle}); err != nil && d.err == nil {
		d.err = err
	}
}

// tamanoQueCabe baja el tamaño de letra de medio punto en medio punto hasta que el texto
// quepa en el ancho, sin bajar de minimo: las cifras no se recortan con «…».
func (d *documento) tamanoQueCabe(txt string, ancho float64, negrita bool, tamano, minimo float64) float64 {
	for ; tamano > minimo; tamano -= 0.5 {
		d.fuente(negrita, tamano)
		if w, err := d.pdf.MeasureTextWidth(txt); err == nil && w <= ancho {
			return tamano
		}
	}
	return minimo
}

func (d *documento) recortar(txt string, ancho float64) string {
	if w, err := d.pdf.MeasureTextWidth(txt); err != nil || w <= ancho {
		return txt
	}
	r := []rune(txt)
	for len(r) > 0 {
		r = r[:len(r)-1]
		if w, _ := d.pdf.MeasureTextWidth(string(r) + "…"); w <= ancho {
			break
		}
	}
	return string(r) + "…"
}

func (d *documento) rect(x, y, ancho, alto float64, c color) {
	if ancho <= 0 || alto <= 0 {
		return
	}
	d.pdf.SetFillColor(c.r, c.g, c.b)
	d.pdf.RectFromUpperLeftWithStyle(x, y, ancho, alto, "F")
}

func (d *documento) linea(x1, y1, x2, y2 float64, c color) {
	d.pdf.SetStrokeColor(c.r, c.g, c.b)
	d.pdf.SetLineWidth(0.6)
	d.pdf.Line(x1, y1, x2, y2)
}

// tituloSeccion escribe el título de una sección y deja el cursor debajo.
func (d *documento) tituloSeccion(titulo string, altoContenido float64) {
	d.asegurar(22 + altoContenido)
	d.texto(margen, d.y, d.anchoUtil(), 16, titulo, true, 12, colorTinta, gopdf.Left)
	d.y += 22
}

func (d *documento) parrafoVacio() {
	d.texto(margen, d.y, d.anchoUtil(), 16, textoVacio, false, 11, colorTinta2, gopdf.Left)
	d.y += 24
}

// columna de una tabla: título, ancho y alineación (gopdf.Left o gopdf.Right).
type columna struct {
	titulo     string
	ancho      float64
	alineacion int
}

const altoFila = 16.0

func (d *documento) encabezadoTabla(columnas []columna) {
	d.rect(margen, d.y, d.anchoUtil(), altoFila+2, colorPanel2)
	x := margen
	for _, c := range columnas {
		d.texto(x+4, d.y, c.ancho-8, altoFila+2, c.titulo, true, 8, colorTinta2, c.alineacion)
		x += c.ancho
	}
	d.y += altoFila + 2
}

func (d *documento) fila(columnas []columna, valores []string, negrita bool) {
	x := margen
	for i, c := range columnas {
		d.texto(x+4, d.y, c.ancho-8, altoFila, valores[i], negrita, 8.5, colorTinta, c.alineacion)
		x += c.ancho
	}
	d.y += altoFila
}
