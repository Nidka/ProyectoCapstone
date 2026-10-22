package reportes

import (
	"testing"
	"time"
)

func TestFormatoMontoComoElFrontend(t *testing.T) {
	casos := map[float64]string{
		0:          "S/ 0.00",
		85.5:       "S/ 85.50",
		1234.5:     "S/ 1,234.50",
		1234567.89: "S/ 1,234,567.89",
		-12.3:      "S/ -12.30",
	}
	for v, want := range casos {
		if got := FormatoMonto(v); got != want {
			t.Errorf("FormatoMonto(%v) = %q, se esperaba %q", v, got, want)
		}
	}
}

func TestFormatosDeFecha(t *testing.T) {
	if got := FormatoFecha(time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)); got != "30/09/2026" {
		t.Errorf("FormatoFecha = %q", got)
	}
	if got := FormatoEmision(time.Date(2026, 10, 17, 15, 32, 0, 0, time.UTC)); got != "17/10/2026 10:32" {
		t.Errorf("FormatoEmision = %q", got)
	}
	if got := MesCorto("2026-10"); got != "oct 26" {
		t.Errorf("MesCorto = %q", got)
	}
	if got := FormatoPorcentaje(42.26); got != "42.3 %" {
		t.Errorf("FormatoPorcentaje = %q", got)
	}
}
