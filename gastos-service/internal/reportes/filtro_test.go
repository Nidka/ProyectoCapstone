package reportes

import (
	"errors"
	"testing"
	"time"
)

// 17/10/2026 02:00 en UTC todavía es 16/10/2026 en Lima.
var ahoraPrueba = time.Date(2026, 10, 17, 2, 0, 0, 0, time.UTC)

func TestFiltroPorDefectoSonLosUltimos12MesesHastaHoyEnLima(t *testing.T) {
	f, err := ParseFiltro("", "", ahoraPrueba)
	if err != nil {
		t.Fatal(err)
	}
	if f.DesdeISO() != "2025-11-01" || f.HastaISO() != "2026-10-16" {
		t.Fatalf("rango = %s .. %s", f.DesdeISO(), f.HastaISO())
	}
}

func TestFiltroSinDesdeCuentaDesdeElHastaElegido(t *testing.T) {
	f, err := ParseFiltro("", "2026-03-15", ahoraPrueba)
	if err != nil {
		t.Fatal(err)
	}
	if f.DesdeISO() != "2025-04-01" || f.HastaISO() != "2026-03-15" {
		t.Fatalf("rango = %s .. %s", f.DesdeISO(), f.HastaISO())
	}
}

func TestFiltroExplicito(t *testing.T) {
	f, err := ParseFiltro("2026-01-01", "2026-01-31", ahoraPrueba)
	if err != nil || f.DesdeISO() != "2026-01-01" || f.HastaISO() != "2026-01-31" {
		t.Fatalf("rango = %v, err = %v", f, err)
	}
}

func TestFiltroInvalido(t *testing.T) {
	casos := []struct{ desde, hasta, mensaje string }{
		{"2026-13-01", "", "fecha inválida: use AAAA-MM-DD"},
		{"", "17/10/2026", "fecha inválida: use AAAA-MM-DD"},
		{"2026-02-30", "", "fecha inválida: use AAAA-MM-DD"},
		{"2026-05-02", "2026-05-01", "la fecha inicial no puede ser posterior a la final"},
		{"2024-01-01", "2026-01-01", "el rango no puede superar 24 meses"},
	}
	for _, c := range casos {
		_, err := ParseFiltro(c.desde, c.hasta, ahoraPrueba)
		var ef ErrFiltro
		if !errors.As(err, &ef) || ef.Mensaje != c.mensaje {
			t.Errorf("ParseFiltro(%q, %q) = %v, se esperaba %q", c.desde, c.hasta, err, c.mensaje)
		}
	}
}

func TestFiltroAceptaExactamente731Dias(t *testing.T) {
	if _, err := ParseFiltro("2024-01-01", "2025-12-31", ahoraPrueba); err != nil {
		t.Fatalf("731 días debían aceptarse: %v", err)
	}
}
