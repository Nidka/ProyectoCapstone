package ocr

import (
	"reflect"
	"strings"
	"testing"
)

const boletaGrifo = `GRIFO SAN JUAN E.I.R.L.
RUC: 20512345678
BOLETA DE VENTA ELECTRONICA
B001-000123
Fecha de emisión: 30/09/2026   Hora: 08:15
Cliente: EMPRESA DE TRANSPORTES LIZAR SRL
RUC 20481608750
Diesel B5    10.000 GLN    85.50
OP. GRAVADA        S/ 72.46
IGV                S/ 13.04
IMPORTE TOTAL      S/ 85.50
SON: OCHENTA Y CINCO CON 50/100 SOLES`

const facturaPeaje = `FACTURA ELECTRÓNICA F002-00004567
Concesionaria Vial del Norte S.A.C.    R.U.C. 20601234567
Señor(es): LIZAR SRL   R.U.C. 20481608750
Trujillo, 29 de setiembre de 2026
Subtotal 1.059,32
I.G.V. 18% 190,68
TOTAL A PAGAR S/ 1.250,00`

func TestNormalizarEspaciosYCifrasConfundidas(t *testing.T) {
	entrada := "  TOTAL\t\tS/  85.5O \r\n\r\n RUC 2O48l6O875O \n Folio Trujillo T046 \x00"
	want := "TOTAL S/ 85.50\nRUC 20481608750\nFolio Trujillo T046"
	if got := Normalizar(entrada); got != want {
		t.Fatalf("Normalizar = %q, se esperaba %q", got, want)
	}
	if got := Normalizar("caf\xffé"); got != "café" {
		t.Errorf("UTF-8 inválido: %q", got)
	}
}

func TestExtraerBoleta(t *testing.T) {
	d := Extraer(boletaGrifo)
	if d.Monto != (Monto{Valor: 85.5, Texto: "IMPORTE TOTAL S/ 85.50", Confianza: Alta}) {
		t.Errorf("monto = %+v", d.Monto)
	}
	if d.Fecha != (Campo{Valor: "2026-09-30", Texto: "Fecha de emisión: 30/09/2026 Hora: 08:15", Confianza: Alta}) {
		t.Errorf("fecha = %+v", d.Fecha)
	}
	if want := []string{"20512345678", "20481608750"}; !reflect.DeepEqual(d.Rucs, want) {
		t.Errorf("rucs = %v, se esperaba %v", d.Rucs, want)
	}
	if d.Tipo != (Campo{Valor: "boleta", Texto: "BOLETA DE VENTA ELECTRONICA", Confianza: Alta}) {
		t.Errorf("tipo = %+v", d.Tipo)
	}
}

func TestExtraerFacturaConComaDecimalYFechaEnLetras(t *testing.T) {
	d := Extraer(facturaPeaje)
	if d.Monto.Valor != 1250 || d.Monto.Confianza != Alta || d.Monto.Texto != "TOTAL A PAGAR S/ 1.250,00" {
		t.Errorf("monto = %+v", d.Monto)
	}
	if d.Fecha != (Campo{Valor: "2026-09-29", Texto: "Trujillo, 29 de setiembre de 2026", Confianza: Media}) {
		t.Errorf("fecha = %+v", d.Fecha)
	}
	if want := []string{"20601234567", "20481608750"}; !reflect.DeepEqual(d.Rucs, want) {
		t.Errorf("rucs = %v", d.Rucs)
	}
	if d.Tipo.Valor != "factura" {
		t.Errorf("tipo = %+v", d.Tipo)
	}
}

func TestFormatosDeMonto(t *testing.T) {
	casos := map[string]float64{
		"TOTAL: 85.50":               85.5,
		"TOTAL S/85,50":              85.5,
		"Importe total S/. 1,234.56": 1234.56,
		"TOTAL 1.234,56":             1234.56,
		"TOTAL\nS/ 42.00":            42,
		"TOTAL 85.50.":               85.5,
		"SUBTOTAL 70.00\nIGV 12.60\nTOTAL 82.60\nTOTAL DESCUENTOS 5.00": 82.6,
	}
	for texto, want := range casos {
		if got := Extraer(texto).Monto; got.Valor != want || got.Confianza != Alta {
			t.Errorf("%q: monto = %+v, se esperaba %v", texto, got, want)
		}
	}
	for _, texto := range []string{"SUBTOTAL 70.00", "PESO TOTAL (KGM): 30633", "TOTAL 30.09.2026", "Diesel 85.50"} {
		if got := Extraer(texto).Monto; got.Confianza != Ninguna {
			t.Errorf("%q: no debía leer monto, leyó %+v", texto, got)
		}
	}
}

func TestFormatosDeFecha(t *testing.T) {
	casos := map[string]Campo{
		"FECHA: 30-09-2026":                  {Valor: "2026-09-30", Texto: "FECHA: 30-09-2026", Confianza: Alta},
		"Emitido 2026-09-30":                 {Valor: "2026-09-30", Texto: "Emitido 2026-09-30", Confianza: Media},
		"Lima, 1 de Octubre del 2026":        {Valor: "2026-10-01", Texto: "Lima, 1 de Octubre del 2026", Confianza: Media},
		"Vence 15/10/2026\nFecha 30/09/2026": {Valor: "2026-09-30", Texto: "Fecha 30/09/2026", Confianza: Alta},
		"FECHA 31/02/2026 01/03/2026":        {Valor: "2026-03-01", Texto: "FECHA 31/02/2026 01/03/2026", Confianza: Alta},
	}
	for texto, want := range casos {
		if got := Extraer(texto).Fecha; got != want {
			t.Errorf("%q: fecha = %+v, se esperaba %+v", texto, got, want)
		}
	}
	for _, texto := range []string{"FECHA 31/02/2026", "Fecha 30 de brumario de 2026", "FECHA 13/13/2026"} {
		if got := Extraer(texto).Fecha; got.Confianza != Ninguna {
			t.Errorf("%q: no debía leer fecha, leyó %+v", texto, got)
		}
	}
}

func TestRucs(t *testing.T) {
	d := Extraer("RUC 20481608750 20100190797\nDNI 17955625 cel 987654321\nRUC 10456789012 / 30123456789 / 2048160875012\nR.U.C. 20481608750")
	if want := []string{"20481608750", "20100190797", "10456789012"}; !reflect.DeepEqual(d.Rucs, want) {
		t.Fatalf("rucs = %v, se esperaba %v", d.Rucs, want)
	}
}

func TestTipoDeDocumento(t *testing.T) {
	casos := map[string]string{
		"FACTURA ELECTRÓNICA F001-1":                                     "factura",
		"Boleta de Venta Electrónica":                                    "boleta",
		"GUIA DE REMISIÓN ELECTRÓNICA\nFACTURA F001-1":                   "otro",
		"NOTA DE CRÉDITO ELECTRÓNICA\nDoc. que modifica: FACTURA F001-1": "otro",
		"FACTURA F001-1\nGuía de remisión: T001-9":                       "factura",
		"TICKET 0001\nTOTAL 10.00":                                       "otro",
	}
	for texto, want := range casos {
		if got := Extraer(texto).Tipo.Valor; got != want {
			t.Errorf("%q: tipo = %q, se esperaba %q", texto, got, want)
		}
	}
	if got := Extraer("TICKET").Tipo; got.Confianza != Ninguna || got.Texto != "" {
		t.Errorf("sin tipo: %+v", got)
	}
}

func TestTextoSinDatos(t *testing.T) {
	for _, texto := range []string{"", "   \n\n", "Gracias por su compra", strings.Repeat("x", 500)} {
		d := Extraer(texto)
		want := Datos{Monto: Monto{Confianza: Ninguna}, Fecha: Campo{Confianza: Ninguna}, Rucs: []string{}, Tipo: Campo{Valor: "otro", Confianza: Ninguna}}
		if !reflect.DeepEqual(d, want) {
			t.Errorf("%q: datos = %+v", texto, d)
		}
	}
}
