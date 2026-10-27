package ocr

import (
	"reflect"
	"testing"
	"time"
)

var fechaGasto = time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)

func TestSugerenciasCuandoTodoCoincide(t *testing.T) {
	got := Sugerir(Extraer(boletaGrifo), boletaGrifo, 85.5, fechaGasto)
	want := map[string]Sugerencia{
		"monto_coincide":   {Sugerido: true, Detalle: "Leído S/ 85.50"},
		"fecha_coincide":   {Sugerido: true, Detalle: "Leída 30/09/2026"},
		"a_nombre_empresa": {Sugerido: true, Detalle: "RUC 20481608750"},
		"tipo_valido":      {Sugerido: true, Detalle: "Boleta"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sugerencias = %+v", got)
	}
	if _, ok := got["legible"]; ok {
		t.Error("legible nunca se sugiere")
	}
}

func TestSugerenciasCuandoDifiereSoloMuestranLoLeido(t *testing.T) {
	got := Sugerir(Extraer(facturaPeaje), facturaPeaje, 1250.004, time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC))
	if s := got["monto_coincide"]; !s.Sugerido || s.Detalle != "Leído S/ 1,250.00" {
		t.Errorf("diferencia menor a medio céntimo: %+v", s)
	}
	got = Sugerir(Extraer(facturaPeaje), facturaPeaje, 1250.01, fechaGasto)
	if s := got["monto_coincide"]; s.Sugerido || s.Detalle != "Leído S/ 1,250.00" {
		t.Errorf("monto distinto: %+v", s)
	}
	if s := got["fecha_coincide"]; s.Sugerido || s.Detalle != "Leída 29/09/2026" {
		t.Errorf("fecha distinta: %+v", s)
	}
	if s := got["tipo_valido"]; !s.Sugerido || s.Detalle != "Factura" {
		t.Errorf("tipo: %+v", s)
	}
}

func TestSinDatosNoHaySugerencias(t *testing.T) {
	got := Sugerir(Extraer("Gracias por su compra"), "Gracias por su compra", 85.5, fechaGasto)
	for clave, s := range got {
		if s != (Sugerencia{}) {
			t.Errorf("%s: se esperaba sin sugerencia, llegó %+v", clave, s)
		}
	}
	if len(got) != 4 {
		t.Errorf("se esperaban 4 criterios, llegaron %d", len(got))
	}
}

func TestEmpresaPorNombreYNoPorCorreo(t *testing.T) {
	if s := Sugerir(Extraer(""), "Cliente: Transportes Lizar S.R.L.", 1, fechaGasto)["a_nombre_empresa"]; !s.Sugerido || s.Detalle != "Menciona LIZAR" {
		t.Errorf("por nombre: %+v", s)
	}
	if s := Sugerir(Extraer(""), "coordinador@transporteslizar.com ineslizarraga@hotmail.com", 1, fechaGasto)["a_nombre_empresa"]; s.Sugerido {
		t.Errorf("un correo no es el nombre de la empresa: %+v", s)
	}
}

func TestGuiaNoSugiereTipo(t *testing.T) {
	texto := "GUIA DE REMISIÓN ELECTRÓNICA\nR.U.C. 20481608750"
	got := Sugerir(Extraer(texto), texto, 1, fechaGasto)
	if got["tipo_valido"].Sugerido {
		t.Error("una guía no es boleta ni factura")
	}
	if !got["a_nombre_empresa"].Sugerido {
		t.Error("el RUC de LIZAR sí se sugiere")
	}
}
