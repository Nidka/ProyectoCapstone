package reportes

import (
	"math"
	"strconv"
	"strings"
	"time"
)

var mesesCortos = [...]string{"ene", "feb", "mar", "abr", "may", "jun", "jul", "ago", "sep", "oct", "nov", "dic"}

// FormatoMonto escribe un monto como el frontend (formatMonto): S/ 1,234.50.
func FormatoMonto(v float64) string {
	txt := strconv.FormatFloat(math.Abs(v), 'f', 2, 64)
	entero, decimales, _ := strings.Cut(txt, ".")
	var b strings.Builder
	for i, c := range entero {
		if i > 0 && (len(entero)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	signo := ""
	if v < 0 {
		signo = "-"
	}
	return "S/ " + signo + b.String() + "." + decimales
}

// FormatoFecha escribe una fecha como 30/09/2026.
func FormatoFecha(t time.Time) string { return t.Format("02/01/2006") }

// FormatoEmision escribe el momento de emisión en hora de Lima: 17/10/2026 10:32.
func FormatoEmision(t time.Time) string { return t.In(Lima).Format("02/01/2006 15:04") }

// MesCorto convierte AAAA-MM en «oct 26».
func MesCorto(mes string) string {
	t, err := time.Parse("2006-01", mes)
	if err != nil {
		return mes
	}
	return mesesCortos[t.Month()-1] + " " + t.Format("06")
}

// FormatoPorcentaje escribe 42.5 como «42.5 %».
func FormatoPorcentaje(v float64) string {
	return strconv.FormatFloat(v, 'f', 1, 64) + " %"
}
