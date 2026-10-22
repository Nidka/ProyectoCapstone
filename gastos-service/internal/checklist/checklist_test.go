package checklist

import (
	"errors"
	"reflect"
	"testing"

	"gastos-service/internal/models"
)

func completo() map[string]bool {
	m := map[string]bool{}
	for _, c := range Criterios {
		m[c] = true
	}
	return m
}

func TestCriteriosEnOrden(t *testing.T) {
	want := []string{"legible", "tipo_valido", "a_nombre_empresa", "monto_coincide", "fecha_coincide"}
	if !reflect.DeepEqual(Criterios, want) {
		t.Fatalf("Criterios = %v, se esperaba %v", Criterios, want)
	}
}

func TestAprobacionCompletaEsValida(t *testing.T) {
	if err := ValidarAprobacion(completo()); err != nil {
		t.Fatalf("se esperaba nil, llegó %v", err)
	}
}

func TestAprobacionSinChecklistOIncompleta(t *testing.T) {
	falta := completo()
	delete(falta, "monto_coincide")
	enFalso := completo()
	enFalso["fecha_coincide"] = false
	for nombre, c := range map[string]map[string]bool{"sin checklist": nil, "falta uno": falta, "uno en false": enFalso} {
		if err := ValidarAprobacion(c); !errors.Is(err, ErrIncompleto) {
			t.Errorf("%s: se esperaba ErrIncompleto, llegó %v", nombre, err)
		}
	}
	if ErrIncompleto.Error() != "para aprobar, el comprobante debe cumplir los 5 criterios del checklist" {
		t.Errorf("mensaje inesperado: %q", ErrIncompleto.Error())
	}
}

func TestCriterioDesconocido(t *testing.T) {
	c := completo()
	c["sello"] = true
	var e ErrCriterioDesconocido
	if err := ValidarAprobacion(c); !errors.As(err, &e) || e.Clave != "sello" {
		t.Fatalf("se esperaba ErrCriterioDesconocido{sello}, llegó %v", err)
	}
	if e.Error() != "criterio desconocido: sello" {
		t.Errorf("mensaje inesperado: %q", e.Error())
	}
}

func TestRechazoSinChecklistOConCriteriosEnFalseEsValido(t *testing.T) {
	if err := ValidarRechazo(nil); err != nil {
		t.Errorf("sin checklist: se esperaba nil, llegó %v", err)
	}
	c := completo()
	c["monto_coincide"] = false
	c["fecha_coincide"] = false
	if err := ValidarRechazo(c); err != nil {
		t.Errorf("con criterios en false: se esperaba nil, llegó %v", err)
	}
}

func TestRechazoConChecklistIncompletoOClaveDesconocida(t *testing.T) {
	falta := completo()
	delete(falta, "legible")
	if err := ValidarRechazo(falta); !errors.Is(err, ErrFaltanCriterios) {
		t.Errorf("falta uno: se esperaba ErrFaltanCriterios, llegó %v", err)
	}
	extra := completo()
	extra["firma"] = false
	var e ErrCriterioDesconocido
	if err := ValidarRechazo(extra); !errors.As(err, &e) {
		t.Errorf("clave desconocida: se esperaba ErrCriterioDesconocido, llegó %v", err)
	}
}

func TestItemsYOrdenar(t *testing.T) {
	c := completo()
	c["tipo_valido"] = false
	items := Items(c)
	if len(items) != 5 || items[0].Criterio != "legible" || items[1].Criterio != "tipo_valido" || items[1].Cumple {
		t.Fatalf("Items en orden inesperado: %v", items)
	}
	desordenados := []models.ItemChecklist{{Criterio: "fecha_coincide", Cumple: true}, {Criterio: "legible", Cumple: true}, {Criterio: "monto_coincide"}}
	got := Ordenar(desordenados)
	want := []string{"legible", "monto_coincide", "fecha_coincide"}
	for i, it := range got {
		if it.Criterio != want[i] {
			t.Fatalf("Ordenar = %v, se esperaba el orden %v", got, want)
		}
	}
}
