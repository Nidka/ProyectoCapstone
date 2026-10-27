package handlers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

func corregir(e *entorno, token string, id any, cuerpo map[string]any) (int, map[string]any) {
	rec := pedir(e.mux, "POST", fmt.Sprintf("/gastos/%v/correccion", id), token, cuerpo)
	var salida map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &salida)
	return rec.Code, salida
}

func TestCorregirGastoCambiaMontoYFechaYDejaRastro(t *testing.T) {
	e := entornoCompleto(t)
	id := e.registrarGasto(t) // monto 85.50, fecha de hoy
	antes := e.total(t)

	code, g := corregir(e, e.tokenLogistica, id, map[string]any{"monto": 513, "fecha": "2026-09-26", "motivo": "Error de digitación"})
	if code != http.StatusOK {
		t.Fatalf("POST = %d %v", code, g)
	}
	if g["monto"] != 513.0 || g["estado"] != "Registrado" {
		t.Errorf("gasto devuelto = %v", g)
	}

	// Lo guardado es lo que se lee después
	rec := pedir(e.mux, "GET", fmt.Sprintf("/gastos/%d", id), e.tokenLogistica, nil)
	var leido map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &leido)
	if leido["monto"] != 513.0 || fmt.Sprint(leido["fecha"])[:10] != "2026-09-26" {
		t.Errorf("GET /gastos/%d = %v", id, leido)
	}

	a := e.unaNueva(t, antes, "gasto.corregir")
	if a.Usuario != "Ana Quispe" || a.Entidad != "gasto" || a.IDEntidad == nil || *a.IDEntidad != int64(id) {
		t.Errorf("asiento = %+v", a)
	}
	if a.sub("antes", "monto") != 85.5 || a.sub("despues", "monto") != 513.0 || a.sub("despues", "fecha") != "2026-09-26" || a.sub("motivo") != "Error de digitación" {
		t.Errorf("detalle = %v", a.Detalle)
	}
	if a.sub("antes", "fecha") == a.sub("despues", "fecha") {
		t.Errorf("la fecha no cambió en el detalle: %v", a.Detalle)
	}
}

func TestCorregirSoloElMontoDejaLaFecha(t *testing.T) {
	e := entornoCompleto(t)
	id := e.registrarGasto(t)
	code, g := corregir(e, e.tokenLogistica, id, map[string]any{"monto": 99.9, "motivo": "El comprobante dice 99.90"})
	if code != http.StatusOK || g["monto"] != 99.9 {
		t.Fatalf("POST = %d %v", code, g)
	}
	rec := pedir(e.mux, "GET", fmt.Sprintf("/gastos/%d", id), e.tokenLogistica, nil)
	var leido map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &leido)
	if fmt.Sprint(leido["fecha"])[:10] != fmt.Sprint(g["fecha"])[:10] {
		t.Errorf("la fecha cambió sin pedirlo: %v → %v", g["fecha"], leido["fecha"])
	}
}

func TestCorregirGastoValidaYNoDejaRastroSiRechaza(t *testing.T) {
	e := entornoCompleto(t)
	id := e.registrarGasto(t)
	antes := e.total(t)

	casos := []struct {
		nombre string
		cuerpo map[string]any
		code   int
	}{
		{"sin motivo", map[string]any{"monto": 100}, http.StatusBadRequest},
		{"motivo en blanco", map[string]any{"monto": 100, "motivo": "   "}, http.StatusBadRequest},
		{"sin monto ni fecha", map[string]any{"motivo": "x necesito"}, http.StatusBadRequest},
		{"monto cero", map[string]any{"monto": 0, "motivo": "Error de digitación"}, http.StatusBadRequest},
		{"monto negativo", map[string]any{"monto": -5, "motivo": "Error de digitación"}, http.StatusBadRequest},
		{"fecha inválida", map[string]any{"fecha": "26/09/2026", "motivo": "Error de digitación"}, http.StatusBadRequest},
		{"mismo monto: no hay cambios", map[string]any{"monto": 85.5, "motivo": "Error de digitación"}, http.StatusBadRequest},
	}
	for _, c := range casos {
		if code, g := corregir(e, e.tokenLogistica, id, c.cuerpo); code != c.code {
			t.Errorf("%s: POST = %d %v, quiero %d", c.nombre, code, g, c.code)
		}
	}
	if code, _ := corregir(e, e.tokenAdmin, id, map[string]any{"monto": 100, "motivo": "Error de digitación"}); code != http.StatusForbidden {
		t.Errorf("admin = %d, quiero 403", code)
	}
	if code, _ := corregir(e, e.tokenLogistica, 999999, map[string]any{"monto": 100, "motivo": "Error de digitación"}); code != http.StatusNotFound {
		t.Errorf("gasto inexistente = %d, quiero 404", code)
	}
	if code, _ := corregir(e, e.tokenLogistica, "abc", map[string]any{"monto": 100, "motivo": "Error de digitación"}); code != http.StatusBadRequest {
		t.Errorf("id inválido = %d, quiero 400", code)
	}
	if n := e.total(t); n != antes {
		t.Fatalf("lo rechazado dejó %d entradas en la bitácora", n-antes)
	}
	// Y el gasto sigue como estaba
	rec := pedir(e.mux, "GET", fmt.Sprintf("/gastos/%d", id), e.tokenLogistica, nil)
	var leido map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &leido)
	if leido["monto"] != 85.5 {
		t.Errorf("monto = %v, quiero 85.5", leido["monto"])
	}
}

func TestNoSePuedeCorregirUnGastoSustentado(t *testing.T) {
	e := entornoCompleto(t)
	id := e.registrarGasto(t)
	comprobante := e.adjuntar(t, id)
	if rec := pedir(e.mux, "POST", fmt.Sprintf("/comprobantes/%d/aprobar", comprobante), e.tokenLogistica, map[string]any{"checklist": completo()}); rec.Code != http.StatusOK {
		t.Fatalf("aprobar = %d %s", rec.Code, rec.Body)
	}
	antes := e.total(t)
	code, g := corregir(e, e.tokenLogistica, id, map[string]any{"monto": 100, "motivo": "Error de digitación"})
	if code != http.StatusConflict {
		t.Fatalf("POST a un gasto sustentado = %d %v, quiero 409", code, g)
	}
	if e.total(t) != antes {
		t.Error("el intento rechazado dejó una entrada")
	}
}

func TestCorregirUnGastoPagadoConComprobantePendienteSiSePuede(t *testing.T) {
	e := entornoCompleto(t)
	id := e.registrarGasto(t)
	e.adjuntar(t, id) // el gasto pasa a Pagado, comprobante Pendiente
	code, g := corregir(e, e.tokenLogistica, id, map[string]any{"monto": 200, "motivo": "El comprobante dice 200"})
	if code != http.StatusOK || g["estado"] != "Pagado" || g["monto"] != 200.0 {
		t.Fatalf("POST = %d %v", code, g)
	}
}

func TestCorregirLaFechaNoPuedeQuedarDespuesDeLaFechaDeCobro(t *testing.T) {
	e := entornoCompleto(t)
	rec := pedir(e.mux, "POST", "/gastos", e.tokenLogistica, map[string]any{
		"viaje_id": 1, "cliente_id": 1, "monto": 50, "fecha": "2026-09-01", "fecha_cobro": "2026-09-30",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /gastos = %d %s", rec.Code, rec.Body)
	}
	var g struct {
		ID int `json:"id"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &g)
	antes := e.total(t)
	if code, _ := corregir(e, e.tokenLogistica, g.ID, map[string]any{"fecha": "2026-10-05", "motivo": "Error de digitación"}); code != http.StatusUnprocessableEntity {
		t.Fatalf("fecha posterior al cobro = %d, quiero 422", code)
	}
	if e.total(t) != antes {
		t.Error("el intento rechazado dejó una entrada")
	}
	if code, _ := corregir(e, e.tokenLogistica, g.ID, map[string]any{"fecha": "2026-09-15", "motivo": "Error de digitación"}); code != http.StatusOK {
		t.Fatalf("fecha anterior al cobro = %d, quiero 200", code)
	}
}
