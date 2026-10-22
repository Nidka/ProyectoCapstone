package trazabilidad

import "testing"

func TestCatalogoConservaNumeracionDeLaTabla(t *testing.T) {
	if len(Cambios) != 31 {
		t.Fatalf("len(Cambios) = %d; se esperaban 31", len(Cambios))
	}

	for indice, cambio := range Cambios {
		esperado := indice + 1
		if cambio.Numero != esperado {
			t.Fatalf("Cambios[%d].Numero = %d; se esperaba %d", indice, cambio.Numero, esperado)
		}
		if cambio.Fecha == "" || cambio.Descripcion == "" || cambio.Fuente == "" {
			t.Fatalf("el cambio %d tiene datos de trazabilidad incompletos", cambio.Numero)
		}
	}
}
