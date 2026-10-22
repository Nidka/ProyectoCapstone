// Package checklist define el checklist documental que Logística revisa
// antes de aprobar un comprobante: las claves de sus criterios y las reglas
// para aprobar o rechazar. No toca la base de datos.
package checklist

import (
	"errors"
	"sort"

	"gastos-service/internal/models"
)

// Criterios del checklist documental, en el orden en que se revisan y se devuelven.
var Criterios = []string{"legible", "tipo_valido", "a_nombre_empresa", "monto_coincide", "fecha_coincide"}

// ErrIncompleto: para aprobar deben venir los 5 criterios y todos cumplidos.
var ErrIncompleto = errors.New("para aprobar, el comprobante debe cumplir los 5 criterios del checklist")

// ErrFaltanCriterios: un checklist enviado al rechazar debe traer los 5 criterios.
var ErrFaltanCriterios = errors.New("el checklist debe incluir los 5 criterios")

// ErrCriterioDesconocido señala una clave que no es un criterio del checklist.
type ErrCriterioDesconocido struct{ Clave string }

func (e ErrCriterioDesconocido) Error() string { return "criterio desconocido: " + e.Clave }

func esCriterio(clave string) bool {
	for _, c := range Criterios {
		if c == clave {
			return true
		}
	}
	return false
}

// validarClaves revisa las claves en orden alfabético para que el error sea siempre el mismo.
func validarClaves(c map[string]bool) error {
	claves := make([]string, 0, len(c))
	for clave := range c {
		claves = append(claves, clave)
	}
	sort.Strings(claves)
	for _, clave := range claves {
		if !esCriterio(clave) {
			return ErrCriterioDesconocido{Clave: clave}
		}
	}
	return nil
}

// ValidarAprobacion exige los 5 criterios presentes y cumplidos.
func ValidarAprobacion(c map[string]bool) error {
	if err := validarClaves(c); err != nil {
		return err
	}
	for _, clave := range Criterios {
		if !c[clave] {
			return ErrIncompleto
		}
	}
	return nil
}

// ValidarRechazo acepta un rechazo sin checklist; si viene, debe traer los 5 criterios (cumplidos o no).
func ValidarRechazo(c map[string]bool) error {
	if c == nil {
		return nil
	}
	if err := validarClaves(c); err != nil {
		return err
	}
	for _, clave := range Criterios {
		if _, ok := c[clave]; !ok {
			return ErrFaltanCriterios
		}
	}
	return nil
}

// Items convierte el checklist recibido en filas, en el orden de Criterios.
func Items(c map[string]bool) []models.ItemChecklist {
	out := make([]models.ItemChecklist, 0, len(Criterios))
	for _, clave := range Criterios {
		if cumple, ok := c[clave]; ok {
			out = append(out, models.ItemChecklist{Criterio: clave, Cumple: cumple})
		}
	}
	return out
}

// Ordenar devuelve las filas en el orden de Criterios.
func Ordenar(items []models.ItemChecklist) []models.ItemChecklist {
	pos := map[string]int{}
	for i, c := range Criterios {
		pos[c] = i
	}
	out := append([]models.ItemChecklist(nil), items...)
	sort.SliceStable(out, func(i, j int) bool { return pos[out[i].Criterio] < pos[out[j].Criterio] })
	return out
}
