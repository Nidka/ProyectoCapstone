package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"gastos-service/internal/handlers"
	"gastos-service/internal/reportes"
)

func (e *entorno) get(ruta string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	handlers.CORS(e.mux).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, ruta, nil))
	return rec
}

func (e *entorno) gastoEn(t *testing.T, fecha, estado string, monto float64) {
	t.Helper()
	ctx := context.Background()
	var viaje, cliente int
	if err := e.pool.QueryRow(ctx, `INSERT INTO programacionviaje DEFAULT VALUES RETURNING id_viaje`).Scan(&viaje); err != nil {
		t.Fatal(err)
	}
	if err := e.pool.QueryRow(ctx, `INSERT INTO cliente (razon_social) VALUES ('Agro Norte SAC') RETURNING id_cliente`).Scan(&cliente); err != nil {
		t.Fatal(err)
	}
	if _, err := e.pool.Exec(ctx, `INSERT INTO gasto (viaje_id, cliente_id, monto, fecha, estado, creado_por) VALUES ($1, $2, $3, $4, $5, 'x')`, viaje, cliente, monto, fecha, estado); err != nil {
		t.Fatal(err)
	}
}

func TestResumenDelReporte(t *testing.T) {
	e := nuevoEntorno(t)
	e.gastoEn(t, "2026-03-10", "Sustentado", 120)
	e.gastoEn(t, "2026-03-11", "Registrado", 80)
	// gastoEn crea un cliente por gasto (mismo nombre): el ranking los separa por id
	rec := e.get("/reportes/resumen?desde=2026-03-01&hasta=2026-03-31")
	if rec.Code != http.StatusOK {
		t.Fatalf("código %d: %s", rec.Code, rec.Body)
	}
	var r reportes.Resumen
	if err := json.Unmarshal(rec.Body.Bytes(), &r); err != nil {
		t.Fatal(err)
	}
	if r.Desde != "2026-03-01" || r.Indicadores.Total != 200 || r.Indicadores.PorcentajeSustentado != 60 || len(r.PorMes) != 1 || len(r.PorCliente) != 2 {
		t.Fatalf("resumen = %+v", r)
	}
	if bytes.Contains(rec.Body.Bytes(), []byte("Detalle")) {
		t.Fatalf("el resumen no debe incluir el detalle: %s", rec.Body)
	}
}

func TestReporteConFiltroInvalidoDa400(t *testing.T) {
	e := nuevoEntorno(t)
	casos := map[string]string{
		"/reportes/resumen?desde=2026-13-01":                          "fecha inválida: use AAAA-MM-DD",
		"/reportes/estadistico.pdf?desde=2026-05-02&hasta=2026-05-01": "la fecha inicial no puede ser posterior a la final",
		"/reportes/detalle.pdf?desde=2020-01-01&hasta=2026-01-01":     "el rango no puede superar 24 meses",
	}
	for ruta, mensaje := range casos {
		rec := e.get(ruta)
		var cuerpo map[string]string
		_ = json.Unmarshal(rec.Body.Bytes(), &cuerpo)
		if rec.Code != http.StatusBadRequest || cuerpo["error"] != mensaje {
			t.Errorf("%s: código %d, cuerpo %s", ruta, rec.Code, rec.Body)
		}
	}
}

func TestDescargaDeLosPDF(t *testing.T) {
	e := nuevoEntorno(t)
	e.gastoEn(t, "2026-03-10", "Pagado", 99.9)
	casos := map[string]string{
		"/reportes/estadistico.pdf?desde=2026-03-01&hasta=2026-03-31": `attachment; filename="reporte-estadistico-2026-03-01_2026-03-31.pdf"`,
		"/reportes/detalle.pdf?desde=2026-03-01&hasta=2026-03-31":     `attachment; filename="detalle-gastos-2026-03-01_2026-03-31.pdf"`,
	}
	for ruta, disposicion := range casos {
		rec := e.get(ruta)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: código %d: %s", ruta, rec.Code, rec.Body)
		}
		if rec.Header().Get("Content-Type") != "application/pdf" || rec.Header().Get("Content-Disposition") != disposicion {
			t.Errorf("%s: cabeceras %v", ruta, rec.Header())
		}
		if rec.Header().Get("Access-Control-Expose-Headers") != "Content-Disposition" {
			t.Errorf("%s: CORS no expone Content-Disposition", ruta)
		}
		if !bytes.HasPrefix(rec.Body.Bytes(), []byte("%PDF-")) {
			t.Errorf("%s: el cuerpo no es un PDF", ruta)
		}
	}
}

func TestReporteConFalloDeBaseDa500YQuedaEnElLog(t *testing.T) {
	e := nuevoEntorno(t)
	var registro bytes.Buffer
	log.SetOutput(&registro)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	e.pool.Close()
	for _, ruta := range []string{"/reportes/resumen", "/reportes/estadistico.pdf"} {
		rec := e.get(ruta)
		var cuerpo map[string]string
		_ = json.Unmarshal(rec.Body.Bytes(), &cuerpo)
		if rec.Code != http.StatusInternalServerError || cuerpo["error"] != "no se pudo generar el reporte" {
			t.Errorf("%s: código %d, cuerpo %s", ruta, rec.Code, rec.Body)
		}
	}
	if !bytes.Contains(registro.Bytes(), []byte("reportes:")) {
		t.Fatalf("el error no quedó en el log: %q", registro.String())
	}
}
