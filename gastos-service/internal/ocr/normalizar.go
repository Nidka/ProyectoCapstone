// Package ocr lee los comprobantes de pago (PDF con texto, PDF escaneados y fotos), extrae monto,
// fecha, RUC y tipo de documento, y sugiere qué criterios del checklist se cumplen. Nunca aprueba ni
// rechaza: solo sugiere ✓, y siempre decide una persona.
package ocr

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// MinAlfanumericos es el mínimo de letras y cifras para considerar que un texto se leyó; con menos,
// un PDF se trata como escaneado y una lectura queda «sin texto».
const MinAlfanumericos = 20

var espacios = regexp.MustCompile(`[\t\f\v \x{00A0}]+`)

// tramoNumerico es una seguidilla de cifras, separadores y las letras que el OCR confunde con 0 y 1.
var tramoNumerico = regexp.MustCompile(`[0-9OolI][0-9OolI.,/-]*[0-9OolI]`)

var aCifra = strings.NewReplacer("O", "0", "o", "0", "l", "1", "I", "1")

// Normalizar deja el texto en UTF-8 válido, sin caracteres de control, con un solo espacio entre
// palabras, sin líneas vacías y con «O», «o», «l» e «I» cambiadas por 0 y 1 solo dentro de números.
func Normalizar(texto string) string {
	texto = strings.ToValidUTF8(texto, "")
	texto = strings.ReplaceAll(texto, "\r\n", "\n")
	texto = strings.ReplaceAll(texto, "\r", "\n")
	texto = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' || r == '\f' || r == '\v' {
			return r
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, texto)
	lineas := []string{}
	for _, l := range strings.Split(texto, "\n") {
		l = strings.TrimSpace(espacios.ReplaceAllString(l, " "))
		if l != "" {
			lineas = append(lineas, corregirCifras(l))
		}
	}
	return strings.Join(lineas, "\n")
}

func corregirCifras(linea string) string {
	var b strings.Builder
	ultimo := 0
	for _, m := range tramoNumerico.FindAllStringIndex(linea, -1) {
		if !esNumero(linea, m[0], m[1]) {
			continue
		}
		b.WriteString(linea[ultimo:m[0]])
		b.WriteString(aCifra.Replace(linea[m[0]:m[1]]))
		ultimo = m[1]
	}
	b.WriteString(linea[ultimo:])
	return b.String()
}

// esNumero: el tramo tiene alguna cifra real y no está pegado a otra letra, así «Folio», «Trujillo»
// o la placa «T046» no cambian.
func esNumero(linea string, ini, fin int) bool {
	if !strings.ContainsAny(linea[ini:fin], "0123456789") {
		return false
	}
	if r, _ := utf8.DecodeLastRuneInString(linea[:ini]); ini > 0 && unicode.IsLetter(r) {
		return false
	}
	if r, _ := utf8.DecodeRuneInString(linea[fin:]); fin < len(linea) && unicode.IsLetter(r) {
		return false
	}
	return true
}

// Alfanumericos cuenta las letras y cifras de un texto.
func Alfanumericos(texto string) int {
	n := 0
	for _, r := range texto {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			n++
		}
	}
	return n
}
