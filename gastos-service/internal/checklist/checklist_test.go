package checklist

import (
	"errors"
	"strings"
	"testing"
)

func estandar() []Criterio {
	return []Criterio{
		{Clave: "legible", Texto: "El comprobante es legible y está completo."},
		{Clave: "monto_coincide", Texto: "El monto coincide con el gasto."},
		{Clave: "c-ab12", Texto: "Tiene sello de recepción."},
	}
}

func marcados(claves ...string) map[string]bool {
	m := map[string]bool{}
	for _, c := range claves {
		m[c] = true
	}
	return m
}

func TestAprobacionExigeTodosLosCriteriosVigentes(t *testing.T) {
	if err := ValidarAprobacion(estandar(), marcados("legible", "monto_coincide", "c-ab12")); err != nil {
		t.Fatalf("con todos marcados se esperaba nil, llegó %v", err)
	}
	casos := map[string]map[string]bool{
		"sin checklist": nil,
		"falta uno":     marcados("legible", "monto_coincide"),
		"uno en false":  {"legible": true, "monto_coincide": true, "c-ab12": false},
	}
	for nombre, c := range casos {
		if err := ValidarAprobacion(estandar(), c); !errors.Is(err, ErrIncompleto) {
			t.Errorf("%s: se esperaba ErrIncompleto, llegó %v", nombre, err)
		}
	}
	if ErrIncompleto.Error() != "para aprobar, el comprobante debe cumplir todos los criterios del checklist" {
		t.Errorf("mensaje inesperado: %q", ErrIncompleto.Error())
	}
}

func TestUnCriterioQueNoEsDelClienteSeRechaza(t *testing.T) {
	c := marcados("legible", "monto_coincide", "c-ab12", "fecha_coincide") // fecha_coincide ya no es de este cliente
	var e ErrCriterioDesconocido
	if err := ValidarAprobacion(estandar(), c); !errors.As(err, &e) || e.Clave != "fecha_coincide" {
		t.Fatalf("se esperaba ErrCriterioDesconocido{fecha_coincide}, llegó %v", err)
	}
	if e.Error() != "criterio desconocido: fecha_coincide" {
		t.Errorf("mensaje inesperado: %q", e.Error())
	}
}

func TestRechazoAceptaCriteriosEnFalseYChecklistAusente(t *testing.T) {
	if err := ValidarRechazo(estandar(), nil); err != nil {
		t.Errorf("sin checklist: %v", err)
	}
	if err := ValidarRechazo(estandar(), map[string]bool{"legible": true, "monto_coincide": false, "c-ab12": false}); err != nil {
		t.Errorf("con criterios en false: %v", err)
	}
	if err := ValidarRechazo(estandar(), marcados("legible")); !errors.Is(err, ErrFaltanCriterios) {
		t.Errorf("checklist incompleto: se esperaba ErrFaltanCriterios, llegó %v", err)
	}
	var e ErrCriterioDesconocido
	if err := ValidarRechazo(estandar(), map[string]bool{"legible": true, "monto_coincide": true, "c-ab12": true, "otro": true}); !errors.As(err, &e) {
		t.Errorf("clave desconocida: %v", err)
	}
}

func TestItemsSiguenElOrdenDeLosCriteriosYGuardanSuTexto(t *testing.T) {
	items := Items(estandar(), map[string]bool{"c-ab12": true, "legible": false, "monto_coincide": true})
	if len(items) != 3 || items[0].Criterio != "legible" || items[0].Cumple || items[2].Criterio != "c-ab12" || !items[2].Cumple {
		t.Fatalf("items = %+v", items)
	}
	if items[2].Texto != "Tiene sello de recepción." {
		t.Errorf("el texto del criterio se guarda con la revisión: %q", items[2].Texto)
	}
}

func TestValidarCriteriosDeUnCliente(t *testing.T) {
	ok := []Entrada{{Texto: "  Tiene sello  "}, {Clave: "legible", Texto: "Es legible"}}
	if err := ValidarEntradas(ok); err != nil {
		t.Fatalf("lista válida: %v", err)
	}
	casos := map[string][]Entrada{
		"vacía":                nil,
		"texto corto":          {{Texto: "ab"}},
		"texto largo":          {{Texto: strings.Repeat("a", 121)}},
		"repetidos":            {{Texto: "Tiene sello"}, {Texto: "tiene SELLO"}},
		"clave con mayúsculas": {{Clave: "Legible", Texto: "Es legible"}},
	}
	for nombre, c := range casos {
		if err := ValidarEntradas(c); !errors.Is(err, ErrCriterios) {
			t.Errorf("%s: se esperaba ErrCriterios, llegó %v", nombre, err)
		}
	}
	quince := make([]Entrada, 16)
	for i := range quince {
		quince[i] = Entrada{Texto: "Criterio número " + strings.Repeat("x", i+1)}
	}
	if err := ValidarEntradas(quince[:15]); err != nil {
		t.Errorf("15 criterios son válidos: %v", err)
	}
	if err := ValidarEntradas(quince); !errors.Is(err, ErrCriterios) {
		t.Errorf("16 criterios son demasiados: %v", err)
	}
}

func TestClavesNuevasSonUnicasYLasExistentesSeConservan(t *testing.T) {
	entradas := []Entrada{{Clave: "legible", Texto: "Es legible"}, {Texto: "Nuevo A"}, {Texto: "Nuevo B"}}
	criterios := AsignarClaves(entradas)
	if criterios[0].Clave != "legible" || criterios[0].Texto != "Es legible" {
		t.Fatalf("la clave existente se conserva: %+v", criterios[0])
	}
	if criterios[1].Clave == "" || criterios[1].Clave == criterios[2].Clave || !strings.HasPrefix(criterios[1].Clave, "c-") {
		t.Fatalf("las claves nuevas deben ser distintas: %+v", criterios)
	}
}
