package handlers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

// gasto.creado_en, comprobante.creado_en y comprobante.aprobado_en son TIMESTAMP sin zona
// y guardan la hora de pared del servidor de base de datos (now()). La API debe devolverlos
// como el instante real, no como esa hora de pared marcada como UTC: el frontend los
// muestra en hora de Lima y, si no, el detalle del gasto marcaría 5 horas de menos.
func TestLosMomentosSinZonaSalenComoInstantesReales(t *testing.T) {
	e := nuevoEntorno(t)
	gastoID, comprobanteID := e.comprobantePendiente(t)
	ctx := context.Background()
	if _, err := e.pool.Exec(ctx, `UPDATE gasto SET creado_en = '2026-10-22 18:55:00' WHERE id = $1`, gastoID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.pool.Exec(ctx, `UPDATE comprobante SET creado_en = '2026-10-22 18:56:00', aprobado_en = '2026-10-22 18:57:00' WHERE id = $1`, comprobanteID); err != nil {
		t.Fatal(err)
	}
	esperado := func(pared string) time.Time {
		var t0 time.Time
		if err := e.pool.QueryRow(ctx, `SELECT $1::timestamp AT TIME ZONE current_setting('TimeZone')`, pared).Scan(&t0); err != nil {
			t.Fatal(err)
		}
		return t0
	}
	mismoInstante := func(nombre, texto string, quiere time.Time) {
		t.Helper()
		got, err := time.Parse(time.RFC3339Nano, texto)
		if err != nil {
			t.Fatalf("%s = %q no es RFC3339: %v", nombre, texto, err)
		}
		if !got.Equal(quiere) {
			t.Errorf("%s = %s, se esperaba el instante %s", nombre, got.UTC().Format(time.RFC3339), quiere.UTC().Format(time.RFC3339))
		}
	}

	rec := e.get(fmt.Sprintf("/gastos/%d", gastoID))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /gastos/{id}: %d %s", rec.Code, rec.Body)
	}
	var g struct {
		CreadoEn string `json:"creado_en"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &g)
	mismoInstante("GET /gastos/{id} creado_en", g.CreadoEn, esperado("2026-10-22 18:55:00"))

	rec = e.get("/gastos")
	var lista []struct {
		CreadoEn string `json:"creado_en"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &lista)
	if len(lista) != 1 {
		t.Fatalf("GET /gastos: %d filas", len(lista))
	}
	mismoInstante("GET /gastos creado_en", lista[0].CreadoEn, esperado("2026-10-22 18:55:00"))

	rec = e.get("/comprobantes")
	var comps []struct {
		CreadoEn   string `json:"creado_en"`
		AprobadoEn string `json:"aprobado_en"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &comps)
	if len(comps) != 1 {
		t.Fatalf("GET /comprobantes: %d filas (%s)", len(comps), rec.Body)
	}
	mismoInstante("GET /comprobantes creado_en", comps[0].CreadoEn, esperado("2026-10-22 18:56:00"))
	mismoInstante("GET /comprobantes aprobado_en", comps[0].AprobadoEn, esperado("2026-10-22 18:57:00"))
}
