package frank

import "testing"

func TestAportesMantienenOrdenYDatos(t *testing.T) {
	if len(Aportes) != 17 {
		t.Fatalf("len(Aportes) = %d; se esperaban 17", len(Aportes))
	}
	for indice, aporte := range Aportes {
		esperado := indice + 1
		if aporte.Numero != esperado {
			t.Fatalf("Aportes[%d].Numero = %d; se esperaba %d", indice, aporte.Numero, esperado)
		}
		if aporte.Fecha == "" || aporte.Descripcion == "" || aporte.Fuente == "" || len(aporte.Rutas) == 0 {
			t.Fatalf("el aporte %d tiene datos incompletos", aporte.Numero)
		}
	}
}
