package ocr

import (
	"math"
	"regexp"
	"slices"
	"time"

	"gastos-service/internal/reportes"
)

// RucEmpresa es el RUC de LIZAR SRL.
const RucEmpresa = "20481608750"

// Sugerencia de un criterio del checklist: el sistema solo sugiere ✓ (Sugerido true); Detalle dice
// lo que se leyó, también cuando no coincide.
type Sugerencia struct {
	Sugerido bool   `json:"sugerido"`
	Detalle  string `json:"detalle"`
}

var palabraLizar = regexp.MustCompile(`(?i)(?:^|\P{L})lizar(?:\P{L}|$)`)

// Sugerir compara lo leído con el gasto y devuelve una sugerencia para monto_coincide,
// fecha_coincide, a_nombre_empresa y tipo_valido. Nunca para legible (es un juicio humano) y nunca
// ✗: no encontrar un dato no prueba que el comprobante esté mal.
func Sugerir(d Datos, texto string, montoGasto float64, fechaGasto time.Time) map[string]Sugerencia {
	return map[string]Sugerencia{
		"monto_coincide":   sugerirMonto(d.Monto, montoGasto),
		"fecha_coincide":   sugerirFecha(d.Fecha, fechaGasto),
		"a_nombre_empresa": sugerirEmpresa(d.Rucs, texto),
		"tipo_valido":      sugerirTipo(d.Tipo),
	}
}

func sugerirMonto(m Monto, montoGasto float64) Sugerencia {
	if m.Confianza == Ninguna {
		return Sugerencia{}
	}
	return Sugerencia{Sugerido: math.Abs(m.Valor-montoGasto) < 0.005, Detalle: "Leído " + reportes.FormatoMonto(m.Valor)}
}

func sugerirFecha(f Campo, fechaGasto time.Time) Sugerencia {
	leida, err := time.Parse("2006-01-02", f.Valor)
	if f.Confianza == Ninguna || err != nil {
		return Sugerencia{}
	}
	return Sugerencia{Sugerido: f.Valor == fechaGasto.Format("2006-01-02"), Detalle: "Leída " + reportes.FormatoFecha(leida)}
}

func sugerirEmpresa(rucs []string, texto string) Sugerencia {
	switch {
	case slices.Contains(rucs, RucEmpresa):
		return Sugerencia{Sugerido: true, Detalle: "RUC " + RucEmpresa}
	case palabraLizar.MatchString(texto):
		return Sugerencia{Sugerido: true, Detalle: "Menciona LIZAR"}
	default:
		return Sugerencia{}
	}
}

func sugerirTipo(t Campo) Sugerencia {
	switch t.Valor {
	case "factura":
		return Sugerencia{Sugerido: true, Detalle: "Factura"}
	case "boleta":
		return Sugerencia{Sugerido: true, Detalle: "Boleta"}
	default:
		return Sugerencia{}
	}
}
