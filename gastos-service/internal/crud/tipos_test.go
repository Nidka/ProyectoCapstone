package crud

import (
	"testing"
	"time"
)

func TestFilaSeEscribeEnOrdenYSinEscaparHTML(t *testing.T) {
	r := &Recurso{Campos: []Campo{
		{Nombre: "nombre", Tipo: TTexto}, {Nombre: "monto", Tipo: TDecimal}, {Nombre: "creado", Tipo: TFechaHora},
		{Nombre: "dia", Tipo: TFecha}, {Nombre: "n", Tipo: TEntero}, {Nombre: "nota", Tipo: TTexto},
	}}
	f := Fila{recurso: r, valores: []any{
		"A&B <C>", "12.50", time.Date(2026, 9, 29, 8, 0, 0, 0, time.UTC), time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC), int32(7), nil,
	}}
	directo, err := f.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	esperado := `{"nombre":"A&B <C>","monto":"12.50","creado":"2026-09-29T08:00:00","dia":"2026-09-29","n":7,"nota":null}`
	if string(directo) != esperado {
		t.Fatalf("MarshalJSON = %s\nesperado     %s", directo, esperado)
	}
}

func TestFormatoFechaHoraComoIsoformat(t *testing.T) {
	if got := FormatoFechaHora(time.Date(2026, 9, 29, 8, 5, 3, 0, time.UTC)); got != "2026-09-29T08:05:03" {
		t.Errorf("sin microsegundos: %s", got)
	}
	if got := FormatoFechaHora(time.Date(2026, 9, 29, 8, 5, 3, 120000, time.UTC)); got != "2026-09-29T08:05:03.000120" {
		t.Errorf("con microsegundos: %s", got)
	}
}

func TestLasDefinicionesSonCoherentes(t *testing.T) {
	if len(Recursos) != 19 {
		t.Fatalf("se esperaban 19 recursos, hay %d", len(Recursos))
	}
	rutas := map[string]bool{}
	for _, r := range Recursos {
		if rutas[r.Ruta] {
			t.Errorf("ruta repetida: %s", r.Ruta)
		}
		rutas[r.Ruta] = true
		if c, ok := r.Campo(r.PK); !ok || c.Tipo != TEntero {
			t.Errorf("%s: la clave %q debe ser un campo entero", r.Ruta, r.PK)
		}
		for _, b := range r.Busqueda {
			if _, ok := r.Campo(b); !ok {
				t.Errorf("%s: el campo de búsqueda %q no existe", r.Ruta, b)
			}
		}
		if r.Unico != nil {
			if _, ok := r.Campo(r.Unico.Campo); !ok {
				t.Errorf("%s: el campo único %q no existe", r.Ruta, r.Unico.Campo)
			}
		}
	}
}
