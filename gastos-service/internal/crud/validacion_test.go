package crud

import (
	"testing"
	"time"
)

func TestCoercionarEntero(t *testing.T) {
	c := Campo{Nombre: "n", Tipo: TEntero}
	for _, v := range []any{float64(12), "12", " 12 ", float64(12.0)} {
		if got, e := Coercionar(c, v); e != nil || got != int64(12) {
			t.Errorf("Coercionar(%v) = %v, %v; se esperaba 12", v, got, e)
		}
	}
	casos := map[string]any{"int_parsing": "abc", "int_from_float": 1.5, "int_type": true}
	for tipo, v := range casos {
		if _, e := Coercionar(c, v); e == nil || e.Tipo != tipo {
			t.Errorf("Coercionar(%v): error %v, se esperaba %s", v, e, tipo)
		}
	}
	if _, e := Coercionar(c, nil); e == nil || e.Tipo != "int_type" {
		t.Errorf("null en un campo no nulo: %v", e)
	}
	if got, e := Coercionar(Campo{Tipo: TEntero, Nulo: true}, nil); e != nil || got != nil {
		t.Errorf("null en un campo nulo: %v %v", got, e)
	}
}

func TestCoercionarTextoConLargos(t *testing.T) {
	c := Campo{Tipo: TTexto, Min: 3, Max: 5}
	if got, e := Coercionar(c, "abcde"); e != nil || got != "abcde" {
		t.Fatalf("abcde: %v %v", got, e)
	}
	if _, e := Coercionar(c, "abcdef"); e == nil || e.Tipo != "string_too_long" || e.Msg != "String should have at most 5 characters" || e.Ctx["max_length"] != 5 {
		t.Errorf("largo máximo: %+v", e)
	}
	if _, e := Coercionar(c, "ab"); e == nil || e.Tipo != "string_too_short" {
		t.Errorf("largo mínimo: %+v", e)
	}
	if _, e := Coercionar(c, float64(12345)); e == nil || e.Tipo != "string_type" {
		t.Errorf("un número no es texto: %+v", e)
	}
	// Se cuentan caracteres, no bytes: «ñandú» son 5 caracteres
	if _, e := Coercionar(Campo{Tipo: TTexto, Max: 5}, "ñandú"); e != nil {
		t.Errorf("ñandú debía caber en 5: %+v", e)
	}
	if _, e := Coercionar(Campo{Tipo: TTexto, Email: true}, "no-es-correo"); e == nil {
		t.Error("un correo sin arroba ni dominio debía rechazarse")
	}
}

func TestCoercionarBoolDecimalYFechas(t *testing.T) {
	if got, e := Coercionar(Campo{Tipo: TBool}, "true"); e != nil || got != true {
		t.Errorf("bool desde texto: %v %v", got, e)
	}
	if _, e := Coercionar(Campo{Tipo: TBool}, "quizás"); e == nil || e.Tipo != "bool_parsing" {
		t.Errorf("bool inválido: %v", e)
	}
	if got, e := Coercionar(Campo{Tipo: TDecimal}, float64(8000.5)); e != nil || got != "8000.5" {
		t.Errorf("decimal desde número: %v %v", got, e)
	}
	if got, e := Coercionar(Campo{Tipo: TDecimal}, "8000.50"); e != nil || got != "8000.50" {
		t.Errorf("decimal desde texto: %v %v", got, e)
	}
	if _, e := Coercionar(Campo{Tipo: TDecimal}, "mucho"); e == nil || e.Tipo != "decimal_parsing" {
		t.Errorf("decimal inválido: %v", e)
	}
	esperado := time.Date(2026, 9, 29, 8, 30, 0, 0, time.UTC)
	for _, v := range []string{"2026-09-29T08:30:00", "2026-09-29 08:30:00", "2026-09-29T08:30:00Z"} {
		if got, e := Coercionar(Campo{Tipo: TFechaHora}, v); e != nil || !got.(time.Time).Equal(esperado) {
			t.Errorf("fecha-hora %q: %v %v", v, got, e)
		}
	}
	if got, e := Coercionar(Campo{Tipo: TFecha}, "2026-09-29"); e != nil || !got.(time.Time).Equal(time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("fecha: %v %v", got, e)
	}
	if _, e := Coercionar(Campo{Tipo: TFecha}, "29/09/2026"); e == nil {
		t.Error("una fecha dd/mm/aaaa debía rechazarse")
	}
}

func TestValidarCreacionAplicaDefectosYJuntaLosErrores(t *testing.T) {
	r := &Recurso{Campos: []Campo{
		{Nombre: "nombre", Tipo: TTexto, Obligatorio: true, Max: 10},
		{Nombre: "detalle", Tipo: TTexto, Nulo: true},
		{Nombre: "estado", Tipo: TBool, Defecto: true},
		{Nombre: "id", Tipo: TEntero, SoloLectura: true},
	}}
	valores, errs := ValidarCreacion(r, map[string]any{"nombre": "Agro"})
	if len(errs) != 0 || valores["estado"] != true || valores["detalle"] != nil {
		t.Fatalf("valores = %v, errores = %v", valores, errs)
	}
	if _, ok := valores["id"]; ok {
		t.Error("la clave generada no se envía a la base")
	}
	_, errs = ValidarCreacion(r, map[string]any{"detalle": 5.0, "estado": "talvez"})
	if len(errs) != 3 || errs[0].Tipo != "missing" || errs[0].Loc[1] != "nombre" || errs[1].Tipo != "string_type" || errs[2].Tipo != "bool_parsing" {
		t.Fatalf("errores = %+v", errs)
	}
}

func TestValidarCambiosIgnoraLoDesconocidoYLaClave(t *testing.T) {
	r := &Recurso{PK: "id", Campos: []Campo{{Nombre: "nombre", Tipo: TTexto}, {Nombre: "id", Tipo: TEntero}}}
	valores, errs := ValidarCambios(r, map[string]any{"nombre": "Nuevo", "id": 99.0, "inventado": "x"})
	if len(errs) != 0 || len(valores) != 1 || valores["nombre"] != "Nuevo" {
		t.Fatalf("valores = %v, errores = %v", valores, errs)
	}
}
