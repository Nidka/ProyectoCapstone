package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"gastos-service/internal/handlers"
	"gastos-service/internal/pruebas"
)

func servidorCompleto(t *testing.T) http.Handler {
	t.Helper()
	pool := pruebas.NuevoEsquemaCompleto(t)
	servidor, logistica, _ := servidorAutenticado(t, pool, handlers.New(pool, configDePrueba(t.TempDir())))
	return conToken(servidor, logistica)
}

func llamar(t *testing.T, h http.Handler, metodo, ruta string, cuerpo any) (int, map[string]any) {
	t.Helper()
	var b []byte
	if cuerpo != nil {
		b, _ = json.Marshal(cuerpo)
	}
	req := httptest.NewRequest(metodo, ruta, bytes.NewReader(b))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var obj map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &obj)
	return rec.Code, obj
}

// Guías, presupuestos, viajes realizados y facturas cuelgan de un padre que solo admite un hijo:
// el contrato las cubre con su error; aquí se prueba el alta que sí funciona, creando el padre.
func TestAltaDeRecursosConPadreUnico(t *testing.T) {
	h := servidorCompleto(t)
	_, viaje := llamar(t, h, "POST", "/api/v1/viajes", map[string]any{
		"id_tracto": 2, "id_carreta": 1, "id_conductor_unidad": 1, "id_direccion_origen": 1, "id_direccion_destino": 1,
		"fecha_programado": "2026-10-01T08:00:00", "id_cliente": 1,
	})
	idViaje := viaje["id_viaje"]
	if idViaje == nil {
		t.Fatalf("no se creó el viaje: %v", viaje)
	}

	estado, presupuesto := llamar(t, h, "POST", "/api/v1/presupuestos-viaje", map[string]any{"id_viaje": idViaje, "monto_total": 4000, "fecha_registro": "2026-10-01T00:00:00"})
	if estado != 201 || presupuesto["monto_total"] != "4000.00" {
		t.Fatalf("presupuesto: %d %v", estado, presupuesto)
	}
	estado, guia := llamar(t, h, "POST", "/api/v1/guias-transportista", map[string]any{"numero_guia": "GT-NUEVA-1", "id_viaje": idViaje, "peso_total": "11000.5"})
	if estado != 201 || guia["peso_total"] != "11000.50" || guia["guias_remitente"] != nil {
		t.Fatalf("guía: %d %v", estado, guia)
	}
	estado, _ = llamar(t, h, "POST", "/api/v1/guias-transportista", map[string]any{"numero_guia": "GT-NUEVA-1", "id_viaje": idViaje, "peso_total": 1})
	if estado != 409 {
		t.Fatalf("una guía con el mismo número debía dar 409, dio %d", estado)
	}
	estado, realizado := llamar(t, h, "POST", "/api/v1/viajes-realizados", map[string]any{"id_guia_transportista": guia["id_guia_transportista"], "flete": 8000, "fecha_llegada": "2026-10-02T10:00:00"})
	if estado != 201 || realizado["flete"] != "8000.00" {
		t.Fatalf("viaje realizado: %d %v", estado, realizado)
	}
	estado, factura := llamar(t, h, "POST", "/api/v1/facturas", map[string]any{"numero_factura": "F001-1", "id_viaje_realizado": realizado["id_viaje_realizado"], "monto_total": 8200})
	if estado != 201 || factura["estado_factura"] != true {
		t.Fatalf("factura: %d %v", estado, factura)
	}
	// Un viaje con presupuesto, guía y factura no se puede borrar: viola una llave foránea (500 en Python también)
	estado, _ = llamar(t, h, "DELETE", fmt.Sprintf("/api/v1/viajes/%v", idViaje), nil)
	if estado != 500 {
		t.Fatalf("borrar un viaje con hijos daba 500 en Python; dio %d", estado)
	}
}

func TestLaBusquedaNoEsUnaInyeccionSQL(t *testing.T) {
	h := servidorCompleto(t)
	req := httptest.NewRequest("GET", "/api/v1/advanced/clientes?search=%27%3B+DROP+TABLE+cliente%3B+--&filters=ruc%3A%27+OR+1%3D1&order_by=ruc%3B+DROP+TABLE+cliente", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 || rec.Body.String() != "[]\n" {
		t.Fatalf("código %d, cuerpo %q", rec.Code, rec.Body.String())
	}
	if estado, _ := llamar(t, h, "GET", "/api/v1/clientes/1", nil); estado != 200 {
		t.Fatalf("la tabla cliente debía seguir existiendo, dio %d", estado)
	}
}

func TestCORSPermitePUTyDELETEYLaSaludIncluyeElEntorno(t *testing.T) {
	h := servidorCompleto(t)
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/rubros/1", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	metodos := rec.Header().Get("Access-Control-Allow-Methods")
	if rec.Code != http.StatusNoContent || metodos != "GET, POST, PUT, DELETE, OPTIONS" {
		t.Fatalf("preflight: %d %q", rec.Code, metodos)
	}
	estado, salud := llamar(t, h, "GET", "/health", nil)
	if estado != 200 || salud["status"] != "ok" || salud["env"] != "dev" {
		t.Fatalf("health: %d %v", estado, salud)
	}
}
