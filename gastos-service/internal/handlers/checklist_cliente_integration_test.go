package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

type criterioJSON struct {
	Clave string `json:"clave"`
	Texto string `json:"texto"`
}

type checklistJSON struct {
	ClienteID     int            `json:"cliente_id"`
	Personalizado bool           `json:"personalizado"`
	Criterios     []criterioJSON `json:"criterios"`
}

func (e *entorno) clienteDe(t *testing.T, gastoID int) int {
	t.Helper()
	var id int
	if err := e.pool.QueryRow(context.Background(), `SELECT cliente_id FROM gasto WHERE id = $1`, gastoID).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func (e *entorno) checklistDe(t *testing.T, clienteID int) (int, checklistJSON) {
	t.Helper()
	rec := e.get(fmt.Sprintf("/clientes/%d/checklist", clienteID))
	var c checklistJSON
	_ = json.Unmarshal(rec.Body.Bytes(), &c)
	return rec.Code, c
}

// enviar hace una petición con el token del rol dado ("logistica" o "admin"), o sin token si rol es "".
func (e *entorno) enviar(t *testing.T, metodo, ruta, rol string, cuerpo any) *httptest.ResponseRecorder {
	t.Helper()
	var b []byte
	if cuerpo != nil {
		b, _ = json.Marshal(cuerpo)
	}
	req := httptest.NewRequest(metodo, ruta, bytes.NewReader(b))
	if token := e.conRol(rol); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	e.mux.ServeHTTP(rec, req)
	return rec
}

func claves(c checklistJSON) []string {
	out := []string{}
	for _, x := range c.Criterios {
		out = append(out, x.Clave)
	}
	return out
}

func TestUnClienteSinChecklistPropioUsaElEstandar(t *testing.T) {
	e := nuevoEntorno(t)
	gastoID, _ := e.comprobantePendiente(t)
	codigo, c := e.checklistDe(t, e.clienteDe(t, gastoID))
	if codigo != http.StatusOK || c.Personalizado || len(c.Criterios) != 5 {
		t.Fatalf("código %d: %+v", codigo, c)
	}
	for i, esperado := range criteriosEstandar {
		if c.Criterios[i].Clave != esperado || len(c.Criterios[i].Texto) < 10 {
			t.Fatalf("criterio %d = %+v", i, c.Criterios[i])
		}
	}
	if codigo, _ := e.checklistDe(t, 99999); codigo != http.StatusNotFound {
		t.Errorf("un cliente que no existe daba %d", codigo)
	}
}

func TestConfigurarElChecklistDeUnCliente(t *testing.T) {
	e := nuevoEntorno(t)
	gastoID, _ := e.comprobantePendiente(t)
	cliente := e.clienteDe(t, gastoID)
	ruta := fmt.Sprintf("/clientes/%d/checklist", cliente)
	cuerpo := map[string]any{"criterios": []criterioJSON{{Clave: "legible", Texto: "Se lee completo"}, {Texto: "Lleva sello de recepción"}}}

	if rec := e.enviar(t, "PUT", ruta, "admin", cuerpo); rec.Code != http.StatusForbidden {
		t.Fatalf("sin rol de logística: %d", rec.Code)
	}
	rec := e.enviar(t, "PUT", ruta, "logistica", cuerpo)
	var c checklistJSON
	_ = json.Unmarshal(rec.Body.Bytes(), &c)
	if rec.Code != http.StatusOK || !c.Personalizado || len(c.Criterios) != 2 || c.Criterios[0].Clave != "legible" || c.Criterios[0].Texto != "Se lee completo" || c.Criterios[1].Clave == "" {
		t.Fatalf("PUT: %d %s", rec.Code, rec.Body)
	}
	if _, otra := e.checklistDe(t, cliente); !otra.Personalizado || otra.Criterios[1].Clave != c.Criterios[1].Clave {
		t.Fatalf("GET después de guardar: %+v", otra)
	}
	// Otro cliente sigue con el estándar
	otroGasto, _ := e.comprobantePendiente(t)
	if _, otro := e.checklistDe(t, e.clienteDe(t, otroGasto)); otro.Personalizado || len(otro.Criterios) != 5 {
		t.Fatalf("el cambio de un cliente no debe afectar a otro: %+v", otro)
	}

	invalidos := map[string]any{
		"vacía":      map[string]any{"criterios": []criterioJSON{}},
		"repetidos":  map[string]any{"criterios": []criterioJSON{{Texto: "Lleva sello"}, {Texto: "lleva SELLO"}}},
		"muy corto":  map[string]any{"criterios": []criterioJSON{{Texto: "ab"}}},
		"sin cuerpo": nil,
	}
	for nombre, cuerpo := range invalidos {
		if rec := e.enviar(t, "PUT", ruta, "logistica", cuerpo); rec.Code != http.StatusUnprocessableEntity && rec.Code != http.StatusBadRequest {
			t.Errorf("%s: código %d", nombre, rec.Code)
		}
	}
	if rec := e.enviar(t, "PUT", "/clientes/99999/checklist", "logistica", cuerpo); rec.Code != http.StatusNotFound {
		t.Errorf("cliente inexistente: %d", rec.Code)
	}
}

func TestRestablecerDevuelveElEstandar(t *testing.T) {
	e := nuevoEntorno(t)
	gastoID, _ := e.comprobantePendiente(t)
	cliente := e.clienteDe(t, gastoID)
	ruta := fmt.Sprintf("/clientes/%d/checklist", cliente)
	e.enviar(t, "PUT", ruta, "logistica", map[string]any{"criterios": []criterioJSON{{Texto: "Solo este criterio"}}})
	if rec := e.enviar(t, "DELETE", ruta, "admin", nil); rec.Code != http.StatusForbidden {
		t.Fatalf("sin rol: %d", rec.Code)
	}
	if rec := e.enviar(t, "DELETE", ruta, "logistica", nil); rec.Code != http.StatusOK {
		t.Fatalf("restablecer: %d %s", rec.Code, rec.Body)
	}
	if _, c := e.checklistDe(t, cliente); c.Personalizado || len(c.Criterios) != 5 {
		t.Fatalf("después de restablecer: %+v", c)
	}
}

func TestAprobarExigeLosCriteriosDelClienteYGuardaSuTexto(t *testing.T) {
	e := nuevoEntorno(t)
	gastoID, id := e.comprobantePendiente(t)
	cliente := e.clienteDe(t, gastoID)
	rec := e.enviar(t, "PUT", fmt.Sprintf("/clientes/%d/checklist", cliente), "logistica",
		map[string]any{"criterios": []criterioJSON{{Clave: "legible", Texto: "Se lee completo"}, {Texto: "Lleva sello de recepción"}}})
	var c checklistJSON
	_ = json.Unmarshal(rec.Body.Bytes(), &c)
	sello := c.Criterios[1].Clave

	// Los 5 criterios estándar ya no son los de este cliente
	if r := e.post(t, fmt.Sprintf("/comprobantes/%d/aprobar", id), map[string]any{"checklist": completo()}); r.Code != http.StatusUnprocessableEntity {
		t.Fatalf("aprobar con criterios de otro checklist: %d %s", r.Code, r.Body)
	}
	// Falta el nuevo criterio
	if r := e.post(t, fmt.Sprintf("/comprobantes/%d/aprobar", id), map[string]any{"checklist": map[string]bool{"legible": true}}); r.Code != http.StatusUnprocessableEntity {
		t.Fatalf("aprobar sin el criterio nuevo: %d %s", r.Code, r.Body)
	}
	if r := e.post(t, fmt.Sprintf("/comprobantes/%d/aprobar", id), map[string]any{"checklist": map[string]bool{"legible": true, sello: true}}); r.Code != http.StatusOK {
		t.Fatalf("aprobar con los criterios del cliente: %d %s", r.Code, r.Body)
	}

	// La revisión guarda el texto de ese momento: cambiar el checklist después no la altera
	e.enviar(t, "PUT", fmt.Sprintf("/clientes/%d/checklist", cliente), "logistica",
		map[string]any{"criterios": []criterioJSON{{Clave: sello, Texto: "Texto nuevo del sello"}}})
	lista := e.get("/comprobantes")
	var comprobantes []struct {
		ID        int `json:"id"`
		Checklist []struct {
			Criterio string `json:"criterio"`
			Texto    string `json:"texto"`
			Cumple   bool   `json:"cumple"`
		} `json:"checklist"`
	}
	_ = json.Unmarshal(lista.Body.Bytes(), &comprobantes)
	for _, comp := range comprobantes {
		if comp.ID != id {
			continue
		}
		if len(comp.Checklist) != 2 || comp.Checklist[0].Texto != "Se lee completo" || comp.Checklist[1].Texto != "Lleva sello de recepción" || !comp.Checklist[1].Cumple {
			t.Fatalf("checklist guardado = %+v", comp.Checklist)
		}
		return
	}
	t.Fatal("no apareció el comprobante en la lista")
}

func TestRechazarUsaLosCriteriosDelCliente(t *testing.T) {
	e := nuevoEntorno(t)
	gastoID, id := e.comprobantePendiente(t)
	cliente := e.clienteDe(t, gastoID)
	e.enviar(t, "PUT", fmt.Sprintf("/clientes/%d/checklist", cliente), "logistica",
		map[string]any{"criterios": []criterioJSON{{Clave: "legible", Texto: "Se lee completo"}, {Clave: "sello", Texto: "Lleva sello"}}})
	if r := e.post(t, fmt.Sprintf("/comprobantes/%d/rechazar", id), map[string]any{"motivo_rechazo": "x", "checklist": completo()}); r.Code != http.StatusUnprocessableEntity {
		t.Fatalf("rechazar con claves de otro checklist: %d %s", r.Code, r.Body)
	}
	if r := e.post(t, fmt.Sprintf("/comprobantes/%d/rechazar", id), map[string]any{"motivo_rechazo": "Sin sello", "checklist": map[string]bool{"legible": true, "sello": false}}); r.Code != http.StatusOK {
		t.Fatalf("rechazar: %d %s", r.Code, r.Body)
	}
	if n := e.filas(t, id, true); n != 1 {
		t.Fatalf("debía guardar 1 criterio incumplido, hay %d", n)
	}
}
