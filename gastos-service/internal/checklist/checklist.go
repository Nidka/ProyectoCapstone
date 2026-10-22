// Package checklist define las reglas del checklist documental que Logística revisa antes de
// aprobar un comprobante. Los criterios son configurables por cliente: aquí no hay ninguno fijo,
// solo las reglas para validarlos. No toca la base de datos.
package checklist

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"gastos-service/internal/models"
)

const (
	MaxCriterios = 15
	MinTexto     = 3
	MaxTexto     = 120
)

// Criterio es un punto del checklist de un cliente. La clave es estable (no cambia al renombrar el
// texto) y es lo que viaja en la API y se guarda con cada revisión.
type Criterio struct {
	Clave string `json:"clave"`
	Texto string `json:"texto"`
}

// Entrada es un criterio tal como lo envía quien configura el checklist: sin clave si es nuevo.
type Entrada struct {
	Clave string `json:"clave"`
	Texto string `json:"texto"`
}

// ErrIncompleto: para aprobar deben venir todos los criterios vigentes del cliente y cumplidos.
var ErrIncompleto = errors.New("para aprobar, el comprobante debe cumplir todos los criterios del checklist")

// ErrFaltanCriterios: un checklist enviado al rechazar debe traer todos los criterios vigentes.
var ErrFaltanCriterios = errors.New("el checklist debe incluir todos los criterios del cliente")

// ErrCriterios agrupa los errores al configurar el checklist de un cliente (se responde 422).
var ErrCriterios = errors.New("checklist inválido")

// ErrCriterioDesconocido señala una clave que no es un criterio vigente del cliente.
type ErrCriterioDesconocido struct{ Clave string }

func (e ErrCriterioDesconocido) Error() string { return "criterio desconocido: " + e.Clave }

func errCriterios(mensaje string) error { return errors.Join(ErrCriterios, errors.New(mensaje)) }

// Mensaje es el texto legible de un error de ErrCriterios, sin el prefijo genérico.
func Mensaje(err error) string {
	return strings.TrimPrefix(strings.TrimPrefix(err.Error(), ErrCriterios.Error()), "\n")
}

// validarClaves revisa las claves en orden alfabético para que el error sea siempre el mismo.
func validarClaves(vigentes []Criterio, c map[string]bool) error {
	conocidas := map[string]bool{}
	for _, v := range vigentes {
		conocidas[v.Clave] = true
	}
	claves := make([]string, 0, len(c))
	for clave := range c {
		claves = append(claves, clave)
	}
	sort.Strings(claves)
	for _, clave := range claves {
		if !conocidas[clave] {
			return ErrCriterioDesconocido{Clave: clave}
		}
	}
	return nil
}

// ValidarAprobacion exige todos los criterios vigentes presentes y cumplidos.
func ValidarAprobacion(vigentes []Criterio, c map[string]bool) error {
	if err := validarClaves(vigentes, c); err != nil {
		return err
	}
	for _, v := range vigentes {
		if !c[v.Clave] {
			return ErrIncompleto
		}
	}
	return nil
}

// ValidarRechazo acepta un rechazo sin checklist; si viene, debe traer todos los criterios vigentes
// (cumplidos o no).
func ValidarRechazo(vigentes []Criterio, c map[string]bool) error {
	if c == nil {
		return nil
	}
	if err := validarClaves(vigentes, c); err != nil {
		return err
	}
	for _, v := range vigentes {
		if _, ok := c[v.Clave]; !ok {
			return ErrFaltanCriterios
		}
	}
	return nil
}

// Items convierte el checklist recibido en filas, en el orden de los criterios vigentes y con su
// texto de ese momento.
func Items(vigentes []Criterio, c map[string]bool) []models.ItemChecklist {
	out := make([]models.ItemChecklist, 0, len(vigentes))
	for _, v := range vigentes {
		if cumple, ok := c[v.Clave]; ok {
			out = append(out, models.ItemChecklist{Criterio: v.Clave, Texto: v.Texto, Cumple: cumple})
		}
	}
	return out
}

var reClave = regexp.MustCompile(`^[a-z0-9_-]{1,40}$`)

// ValidarEntradas valida la lista que envía quien configura el checklist de un cliente: de 1 a 15
// criterios, con texto de 3 a 120 caracteres, sin repetidos y con claves bien formadas.
func ValidarEntradas(entradas []Entrada) error {
	if len(entradas) == 0 {
		return errCriterios("el checklist debe tener al menos un criterio")
	}
	if len(entradas) > MaxCriterios {
		return errCriterios("el checklist admite como máximo 15 criterios")
	}
	vistos := map[string]bool{}
	claves := map[string]bool{}
	for _, e := range entradas {
		texto := strings.TrimSpace(e.Texto)
		if n := utf8.RuneCountInString(texto); n < MinTexto || n > MaxTexto {
			return errCriterios("cada criterio debe tener entre 3 y 120 caracteres")
		}
		clave := strings.ToLower(texto)
		if vistos[clave] {
			return errCriterios("hay criterios repetidos: «" + texto + "»")
		}
		vistos[clave] = true
		if e.Clave != "" {
			if !reClave.MatchString(e.Clave) {
				return errCriterios("clave de criterio inválida: " + e.Clave)
			}
			if claves[e.Clave] {
				return errCriterios("clave de criterio repetida: " + e.Clave)
			}
			claves[e.Clave] = true
		}
	}
	return nil
}

// AsignarClaves devuelve los criterios listos para guardar: los que ya tenían clave la conservan
// y los nuevos reciben una única (c- y seis caracteres al azar).
func AsignarClaves(entradas []Entrada) []Criterio {
	usadas := map[string]bool{}
	for _, e := range entradas {
		if e.Clave != "" {
			usadas[e.Clave] = true
		}
	}
	out := make([]Criterio, len(entradas))
	for i, e := range entradas {
		clave := e.Clave
		for clave == "" {
			b := make([]byte, 3)
			_, _ = rand.Read(b)
			candidata := "c-" + hex.EncodeToString(b)
			if !usadas[candidata] {
				clave = candidata
				usadas[clave] = true
			}
		}
		out[i] = Criterio{Clave: clave, Texto: strings.TrimSpace(e.Texto)}
	}
	return out
}
