package ocr

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Confianza de un dato: alta si se encontró junto a su etiqueta, media si se infirió, ninguna si no
// se encontró.
const (
	Alta    = "alta"
	Media   = "media"
	Ninguna = "ninguna"
)

// Monto es el importe total leído, con la línea donde se leyó.
type Monto struct {
	Valor     float64 `json:"valor"`
	Texto     string  `json:"texto"`
	Confianza string  `json:"confianza"`
}

// Campo es un dato de texto leído (la fecha como aaaa-mm-dd; el tipo como factura, boleta u otro).
type Campo struct {
	Valor     string `json:"valor"`
	Texto     string `json:"texto"`
	Confianza string `json:"confianza"`
}

// Datos es lo que se extrae del texto de un comprobante; se guarda en lectura_comprobante.datos.
type Datos struct {
	Monto Monto    `json:"monto"`
	Fecha Campo    `json:"fecha"`
	Rucs  []string `json:"rucs"`
	Tipo  Campo    `json:"tipo"`
}

// Extraer lee monto, fecha, RUC y tipo de documento de un texto (se normaliza antes).
func Extraer(texto string) Datos {
	lineas := strings.Split(Normalizar(texto), "\n")
	return Datos{Monto: extraerMonto(lineas), Fecha: extraerFecha(lineas), Rucs: extraerRucs(lineas), Tipo: extraerTipo(lineas)}
}

// coincidencias devuelve cada aparición de re en s (índices como FindStringSubmatchIndex) y sigue
// buscando desde el final del último grupo: los bordes (?:^|\D) y (?:\D|$) no se consumen, así dos
// datos separados por un solo espacio se leen los dos.
func coincidencias(re *regexp.Regexp, s string) [][]int {
	var out [][]int
	for desde := 0; desde < len(s); {
		m := re.FindStringSubmatchIndex(s[desde:])
		if m == nil {
			break
		}
		for i := range m {
			if m[i] >= 0 {
				m[i] += desde
			}
		}
		out = append(out, m)
		desde = m[len(m)-1]
	}
	return out
}

var (
	etiquetaTotal = regexp.MustCompile(`(?i)(?:^|\P{L})(importe\s+total|total\s+a\s+pagar|total)(?:\P{L}|$)`)
	// cifraMonto: 2 decimales con coma o punto, con o sin separador de miles; no toma «30.09.2026».
	cifraMonto = regexp.MustCompile(`(?:^|[^\d.,])(\d{1,3}(?:[.,]\d{3})+[.,]\d{2}|\d+[.,]\d{2})(?:[^\d.,]|[.,](?:\D|$)|$)`)
)

// aNumero convierte «1,234.56», «1.234,56» u «85,50»: el último separador es el decimal.
func aNumero(s string) (float64, bool) {
	entero := strings.NewReplacer(".", "", ",", "").Replace(s[:len(s)-3])
	v, err := strconv.ParseFloat(entero+"."+s[len(s)-2:], 64)
	return v, err == nil
}

// extraerMonto toma el mayor número con 2 decimales que sigue a «TOTAL», «IMPORTE TOTAL» o «TOTAL A
// PAGAR» en la misma línea o, si en esa línea no hay ninguno, en la siguiente.
func extraerMonto(lineas []string) Monto {
	mejor := Monto{Confianza: Ninguna}
	for i, l := range lineas {
		et := etiquetaTotal.FindStringSubmatchIndex(l)
		if et == nil {
			continue
		}
		candidatas := []string{l[et[3]:]}
		if i+1 < len(lineas) {
			candidatas = append(candidatas, lineas[i+1])
		}
		for j, texto := range candidatas {
			encontrado := false
			for _, m := range coincidencias(cifraMonto, texto) {
				v, ok := aNumero(texto[m[2]:m[3]])
				if !ok {
					continue
				}
				encontrado = true
				if mejor.Confianza == Ninguna || v > mejor.Valor {
					mejor = Monto{Valor: v, Texto: lineas[i+j], Confianza: Alta}
				}
			}
			if encontrado {
				break
			}
		}
	}
	return mejor
}

var (
	fechaDMA      = regexp.MustCompile(`(?:^|\D)(\d{1,2})[/-](\d{1,2})[/-](\d{4})(?:\D|$)`)
	fechaAMD      = regexp.MustCompile(`(?:^|\D)(\d{4})-(\d{1,2})-(\d{1,2})(?:\D|$)`)
	fechaLarga    = regexp.MustCompile(`(?i)(?:^|\D)(\d{1,2})\s+de\s+(\p{L}+)\s+(?:de|del)\s+(\d{4})(?:\D|$)`)
	etiquetaFecha = regexp.MustCompile(`(?i)fecha`)
	meses         = map[string]int{
		"enero": 1, "febrero": 2, "marzo": 3, "abril": 4, "mayo": 5, "junio": 6, "julio": 7, "agosto": 8,
		"septiembre": 9, "setiembre": 9, "octubre": 10, "noviembre": 11, "diciembre": 12,
	}
)

type fechaEnLinea struct {
	pos   int
	valor string
}

// fechaValida descarta fechas imposibles (31/02, mes 13) comparando con lo que normaliza time.Date.
func fechaValida(a, m, d int) (string, bool) {
	t := time.Date(a, time.Month(m), d, 0, 0, 0, 0, time.UTC)
	if t.Year() != a || int(t.Month()) != m || t.Day() != d {
		return "", false
	}
	return t.Format("2006-01-02"), true
}

func fechasEn(linea string) []fechaEnLinea {
	var out []fechaEnLinea
	agregar := func(pos int, a, m, d int) {
		if v, ok := fechaValida(a, m, d); ok {
			out = append(out, fechaEnLinea{pos, v})
		}
	}
	num := func(s string) int { n, _ := strconv.Atoi(s); return n }
	for _, m := range coincidencias(fechaDMA, linea) {
		agregar(m[2], num(linea[m[6]:m[7]]), num(linea[m[4]:m[5]]), num(linea[m[2]:m[3]]))
	}
	for _, m := range coincidencias(fechaAMD, linea) {
		agregar(m[2], num(linea[m[2]:m[3]]), num(linea[m[4]:m[5]]), num(linea[m[6]:m[7]]))
	}
	for _, m := range coincidencias(fechaLarga, linea) {
		if mes, ok := meses[strings.ToLower(linea[m[4]:m[5]])]; ok {
			agregar(m[2], num(linea[m[6]:m[7]]), mes, num(linea[m[2]:m[3]]))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].pos < out[j].pos })
	return out
}

// extraerFecha prefiere la primera fecha válida que sigue a «FECHA» en su línea (alta); si no hay,
// la primera fecha válida del texto (media).
func extraerFecha(lineas []string) Campo {
	var primera *Campo
	for _, l := range lineas {
		et := etiquetaFecha.FindStringIndex(l)
		var elegida *fechaEnLinea
		for _, f := range fechasEn(l) {
			if et != nil && f.pos >= et[1] && elegida == nil {
				elegida = &f
			}
			if primera == nil {
				primera = &Campo{Valor: f.valor, Texto: l, Confianza: Media}
			}
		}
		if elegida != nil {
			return Campo{Valor: elegida.valor, Texto: l, Confianza: Alta}
		}
	}
	if primera != nil {
		return *primera
	}
	return Campo{Confianza: Ninguna}
}

var reRuc = regexp.MustCompile(`(?:^|\D)((?:10|15|17|20)\d{9})(?:\D|$)`)

// extraerRucs devuelve los RUC (11 cifras que empiezan con 10, 15, 17 o 20) en el orden en que
// aparecen, sin repetir.
func extraerRucs(lineas []string) []string {
	out := []string{}
	vistos := map[string]bool{}
	for _, l := range lineas {
		for _, m := range coincidencias(reRuc, l) {
			if ruc := l[m[2]:m[3]]; !vistos[ruc] {
				vistos[ruc] = true
				out = append(out, ruc)
			}
		}
	}
	return out
}

// reTipo: la primera de estas palabras en el texto decide el tipo; una guía o una nota de crédito o
// de débito que mencione una factura sigue siendo «otro».
var reTipo = regexp.MustCompile(`(?i)(?:^|\P{L})(factura|boleta|gu[ií]a|nota\s+de\s+(?:cr[eé]dito|d[eé]bito))(?:\P{L}|$)`)

func extraerTipo(lineas []string) Campo {
	for _, l := range lineas {
		m := reTipo.FindStringSubmatchIndex(l)
		if m == nil {
			continue
		}
		switch palabra := strings.ToLower(l[m[2]:m[3]]); palabra {
		case "factura", "boleta":
			return Campo{Valor: palabra, Texto: l, Confianza: Alta}
		default:
			return Campo{Valor: "otro", Texto: l, Confianza: Alta}
		}
	}
	return Campo{Valor: "otro", Confianza: Ninguna}
}
