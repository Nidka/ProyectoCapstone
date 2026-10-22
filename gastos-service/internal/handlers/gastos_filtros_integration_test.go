package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// gastosConDatos crea dos viajes, dos clientes y cuatro gastos:
//
//	id  viaje  cliente  estado
//	a   1      1        Registrado
//	b   1      2        Pagado
//	c   2      1        Sustentado
//	d   2      2        Sustentado
func (e *entorno) gastosConDatos(t *testing.T) {
	t.Helper()
	_, err := e.pool.Exec(context.Background(), `
		INSERT INTO programacionviaje (id_viaje) VALUES (1), (2);
		INSERT INTO cliente (id_cliente, razon_social) VALUES (1, 'Agro SAC'), (2, 'Minera SA');
		INSERT INTO gasto (viaje_id, cliente_id, monto, fecha, estado, creado_por) VALUES
			(1, 1, 10, '2026-05-01', 'Registrado', 'x'),
			(1, 2, 20, '2026-05-02', 'Pagado', 'x'),
			(2, 1, 30, '2026-05-03', 'Sustentado', 'x'),
			(2, 2, 40, '2026-05-04', 'Sustentado', 'x');`)
	if err != nil {
		t.Fatal(err)
	}
}

func (e *entorno) montos(t *testing.T, ruta string) (int, []float64, string) {
	t.Helper()
	rec := e.get(ruta)
	var lista []struct {
		Monto float64 `json:"monto"`
	}
	var cuerpo map[string]string
	if rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), &lista); err != nil {
			t.Fatal(err)
		}
	} else {
		_ = json.Unmarshal(rec.Body.Bytes(), &cuerpo)
	}
	montos := []float64{}
	for _, g := range lista {
		montos = append(montos, g.Monto)
	}
	return rec.Code, montos, cuerpo["error"]
}

func TestListaDeGastosFiltraPorViajeClienteYEstado(t *testing.T) {
	e := nuevoEntorno(t)
	e.gastosConDatos(t)
	casos := map[string][]float64{
		"/gastos":                         {40, 30, 20, 10},
		"/gastos?viaje_id=1":              {20, 10},
		"/gastos?cliente_id=1":            {30, 10},
		"/gastos?estado=Sustentado":       {40, 30},
		"/gastos?viaje_id=2&cliente_id=2": {40},
		"/gastos?viaje_id=2&estado=Sustentado&cliente_id=1": {30},
		"/gastos?viaje_id=1&estado=Sustentado":              {},
		"/gastos?viaje_id=99":                               {},
	}
	for ruta, esperado := range casos {
		codigo, montos, _ := e.montos(t, ruta)
		if codigo != http.StatusOK || len(montos) != len(esperado) {
			t.Errorf("%s: código %d, montos %v, se esperaba %v", ruta, codigo, montos, esperado)
			continue
		}
		for i := range esperado {
			if montos[i] != esperado[i] {
				t.Errorf("%s: montos %v, se esperaba %v", ruta, montos, esperado)
			}
		}
	}
}

func TestListaDeGastosRechazaFiltrosInvalidos(t *testing.T) {
	e := nuevoEntorno(t)
	casos := map[string]string{
		"/gastos?viaje_id=abc":      "viaje_id debe ser un número entero positivo",
		"/gastos?cliente_id=0":      "cliente_id debe ser un número entero positivo",
		"/gastos?cliente_id=-3":     "cliente_id debe ser un número entero positivo",
		"/gastos?estado=Inventado":  "estado debe ser Registrado, Pagado o Sustentado",
		"/gastos?estado=sustentado": "estado debe ser Registrado, Pagado o Sustentado",
	}
	for ruta, mensaje := range casos {
		codigo, _, error := e.montos(t, ruta)
		if codigo != http.StatusBadRequest || error != mensaje {
			t.Errorf("%s: código %d, error %q", ruta, codigo, error)
		}
	}
}

func abrirXLSX(t *testing.T, e *entorno, ruta string) (*httptest.ResponseRecorder, *excelize.File) {
	t.Helper()
	rec := e.get(ruta)
	if rec.Code != http.StatusOK {
		return rec, nil
	}
	f, err := excelize.OpenReader(bytes.NewReader(rec.Body.Bytes()))
	if err != nil {
		t.Fatalf("%s: no es un .xlsx válido: %v", ruta, err)
	}
	t.Cleanup(func() { _ = f.Close() })
	return rec, f
}

func TestExportarExcelAplicaLosMismosFiltros(t *testing.T) {
	e := nuevoEntorno(t)
	e.gastosConDatos(t)
	rec, f := abrirXLSX(t, e, "/gastos/exportar.xlsx?viaje_id=2&estado=Sustentado")
	if rec.Code != http.StatusOK {
		t.Fatalf("código %d: %s", rec.Code, rec.Body)
	}
	if rec.Header().Get("Content-Type") != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		t.Errorf("Content-Type = %q", rec.Header().Get("Content-Type"))
	}
	if d := rec.Header().Get("Content-Disposition"); !strings.HasPrefix(d, `attachment; filename="gastos-`) || !strings.HasSuffix(d, `.xlsx"`) {
		t.Errorf("Content-Disposition = %q", d)
	}
	filas, _ := f.GetRows("Gastos")
	// encabezado + 2 gastos (viaje 2, sustentados: 30 y 40) + fila de total
	if len(filas) != 4 || filas[1][2] != "Agro SAC" || filas[2][2] != "Minera SA" {
		t.Fatalf("filas = %v", filas)
	}
	if v, _ := f.GetCellValue("Resumen", "B3"); v != "Viaje N° 2 · Estado: Sustentado" {
		t.Errorf("filtros en el resumen = %q", v)
	}
}

func TestExportarExcelAcotaPorFechasYValidaLosParametros(t *testing.T) {
	e := nuevoEntorno(t)
	e.gastosConDatos(t)
	_, f := abrirXLSX(t, e, "/gastos/exportar.xlsx?desde=2026-05-02&hasta=2026-05-03")
	if filas, _ := f.GetRows("Gastos"); len(filas) != 4 {
		t.Fatalf("con dos días se esperaban 2 gastos (4 filas con encabezado y total): %v", filas)
	}
	casos := map[string]string{
		"/gastos/exportar.xlsx?desde=2026-13-01":                  "desde debe tener el formato AAAA-MM-DD",
		"/gastos/exportar.xlsx?hasta=ayer":                        "hasta debe tener el formato AAAA-MM-DD",
		"/gastos/exportar.xlsx?desde=2026-06-01&hasta=2026-05-01": "la fecha inicial no puede ser posterior a la final",
		"/gastos/exportar.xlsx?estado=Raro":                       "estado debe ser Registrado, Pagado o Sustentado",
		"/gastos/exportar.xlsx?viaje_id=x":                        "viaje_id debe ser un número entero positivo",
	}
	for ruta, mensaje := range casos {
		codigo, _, error := e.montos(t, ruta)
		if codigo != http.StatusBadRequest || error != mensaje {
			t.Errorf("%s: código %d, error %q", ruta, codigo, error)
		}
	}
}

func TestExportarExcelSinGastosDevuelveUnLibroValido(t *testing.T) {
	e := nuevoEntorno(t)
	rec, f := abrirXLSX(t, e, "/gastos/exportar.xlsx")
	if rec.Code != http.StatusOK || f == nil {
		t.Fatalf("código %d", rec.Code)
	}
	if filas, _ := f.GetRows("Gastos"); len(filas) < 1 || filas[0][0] != "N°" {
		t.Fatalf("filas = %v", filas)
	}
}
