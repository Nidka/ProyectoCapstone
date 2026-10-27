package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"gastos-service/internal/handlers"
	"gastos-service/internal/models"
	"gastos-service/internal/pruebas"

	"github.com/jackc/pgx/v5/pgxpool"
)

type entorno struct {
	pool *pgxpool.Pool
	// mux es el servicio completo, con CORS y el middleware de autenticación reales.
	mux     http.Handler
	uploads string
	// Tokens de Ana Quispe (logistica) y de Admin Prueba (admin), ambos activos.
	tokenLogistica, tokenAdmin string
}

// nuevoEntorno arma un esquema temporal (ver internal/pruebas) y el servicio con sus dos usuarios.
func nuevoEntorno(t *testing.T) *entorno {
	t.Helper()
	pool := pruebas.NuevoEsquema(t)
	return armarEntorno(t, pool, handlers.New(pool, configDePrueba(t.TempDir())))
}

func armarEntorno(t *testing.T, pool *pgxpool.Pool, h *handlers.Handler) *entorno {
	t.Helper()
	servidor, logistica, admin := servidorAutenticado(t, pool, h)
	return &entorno{pool: pool, mux: servidor, uploads: h.Cfg.UploadsDir, tokenLogistica: logistica, tokenAdmin: admin}
}

// comprobantePendiente crea un viaje, un cliente, un gasto Pagado y su comprobante Pendiente.
func (e *entorno) comprobantePendiente(t *testing.T) (gastoID, comprobanteID int) {
	t.Helper()
	ctx := context.Background()
	var viaje, cliente int
	if err := e.pool.QueryRow(ctx, `INSERT INTO programacionviaje DEFAULT VALUES RETURNING id_viaje`).Scan(&viaje); err != nil {
		t.Fatal(err)
	}
	if err := e.pool.QueryRow(ctx, `INSERT INTO cliente DEFAULT VALUES RETURNING id_cliente`).Scan(&cliente); err != nil {
		t.Fatal(err)
	}
	if err := e.pool.QueryRow(ctx, `INSERT INTO gasto (viaje_id, cliente_id, monto, fecha, estado, creado_por)
		VALUES ($1, $2, 85.50, '2026-09-30', 'Pagado', 'Ana Quispe') RETURNING id`, viaje, cliente).Scan(&gastoID); err != nil {
		t.Fatal(err)
	}
	if err := e.pool.QueryRow(ctx, `INSERT INTO comprobante (gasto_id, archivo_url, tipo_archivo)
		VALUES ($1, '/uploads/prueba.jpg', 'imagen') RETURNING id`, gastoID).Scan(&comprobanteID); err != nil {
		t.Fatal(err)
	}
	return gastoID, comprobanteID
}

func (e *entorno) post(t *testing.T, ruta string, cuerpo any) *httptest.ResponseRecorder {
	t.Helper()
	b, err := json.Marshal(cuerpo)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, ruta, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.tokenLogistica)
	rec := httptest.NewRecorder()
	e.mux.ServeHTTP(rec, req)
	return rec
}

func (e *entorno) estados(t *testing.T, gastoID, comprobanteID int) (gasto, comprobante string) {
	t.Helper()
	ctx := context.Background()
	if err := e.pool.QueryRow(ctx, `SELECT estado FROM gasto WHERE id = $1`, gastoID).Scan(&gasto); err != nil {
		t.Fatal(err)
	}
	if err := e.pool.QueryRow(ctx, `SELECT estado FROM comprobante WHERE id = $1`, comprobanteID).Scan(&comprobante); err != nil {
		t.Fatal(err)
	}
	return gasto, comprobante
}

func (e *entorno) filas(t *testing.T, comprobanteID int, soloIncumplidos bool) int {
	t.Helper()
	q := `SELECT count(*) FROM checklist_comprobante WHERE comprobante_id = $1`
	if soloIncumplidos {
		q += ` AND NOT cumple`
	}
	var n int
	if err := e.pool.QueryRow(context.Background(), q, comprobanteID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// criteriosEstandar son los criterios con que parte todo cliente (los siembra la migración 0003).
var criteriosEstandar = []string{"legible", "tipo_valido", "a_nombre_empresa", "monto_coincide", "fecha_coincide"}

func completo() map[string]bool {
	m := map[string]bool{}
	for _, c := range criteriosEstandar {
		m[c] = true
	}
	return m
}

func TestAprobarConChecklistCompletoGuardaYSustenta(t *testing.T) {
	e := nuevoEntorno(t)
	gastoID, id := e.comprobantePendiente(t)
	rec := e.post(t, fmt.Sprintf("/comprobantes/%d/aprobar", id), map[string]any{"checklist": completo()})
	if rec.Code != http.StatusOK {
		t.Fatalf("código %d: %s", rec.Code, rec.Body)
	}
	if g, c := e.estados(t, gastoID, id); g != "Sustentado" || c != "Aprobado" {
		t.Fatalf("estados gasto=%s comprobante=%s", g, c)
	}
	if n := e.filas(t, id, false); n != 5 {
		t.Fatalf("se esperaban 5 filas de checklist, hay %d", n)
	}
	var revisor string
	if err := e.pool.QueryRow(context.Background(), `SELECT aprobado_por FROM comprobante WHERE id = $1`, id).Scan(&revisor); err != nil || revisor != "Ana Quispe" {
		t.Fatalf("aprobado_por = %q (%v)", revisor, err)
	}
}

func TestAprobarIncompletoDa422SinCambios(t *testing.T) {
	e := nuevoEntorno(t)
	gastoID, id := e.comprobantePendiente(t)
	c := completo()
	c["monto_coincide"] = false
	rec := e.post(t, fmt.Sprintf("/comprobantes/%d/aprobar", id), map[string]any{"checklist": c})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("código %d: %s", rec.Code, rec.Body)
	}
	if g, cp := e.estados(t, gastoID, id); g != "Pagado" || cp != "Pendiente" {
		t.Fatalf("estados gasto=%s comprobante=%s", g, cp)
	}
	if n := e.filas(t, id, false); n != 0 {
		t.Fatalf("no debía guardar filas, hay %d", n)
	}
	sinCuerpo := e.post(t, fmt.Sprintf("/comprobantes/%d/aprobar", id), map[string]any{})
	if sinCuerpo.Code != http.StatusUnprocessableEntity {
		t.Fatalf("sin checklist: código %d", sinCuerpo.Code)
	}
}

func TestRevisarComprobanteYaRevisadoDa409(t *testing.T) {
	e := nuevoEntorno(t)
	gastoID, id := e.comprobantePendiente(t)
	if rec := e.post(t, fmt.Sprintf("/comprobantes/%d/aprobar", id), map[string]any{"checklist": completo()}); rec.Code != http.StatusOK {
		t.Fatalf("primera aprobación: %d", rec.Code)
	}
	otraVez := e.post(t, fmt.Sprintf("/comprobantes/%d/aprobar", id), map[string]any{"checklist": completo()})
	rechazo := e.post(t, fmt.Sprintf("/comprobantes/%d/rechazar", id), map[string]any{"motivo_rechazo": "tarde", "checklist": completo()})
	if otraVez.Code != http.StatusConflict || rechazo.Code != http.StatusConflict {
		t.Fatalf("se esperaba 409 y 409, llegaron %d y %d", otraVez.Code, rechazo.Code)
	}
	if g, c := e.estados(t, gastoID, id); g != "Sustentado" || c != "Aprobado" {
		t.Fatalf("una revisión posterior cambió los estados: gasto=%s comprobante=%s", g, c)
	}
	if n := e.filas(t, id, false); n != 5 {
		t.Fatalf("se esperaban 5 filas, hay %d", n)
	}
}

func TestRechazarGuardaChecklistYVuelveARegistrado(t *testing.T) {
	e := nuevoEntorno(t)
	gastoID, id := e.comprobantePendiente(t)
	c := completo()
	c["monto_coincide"] = false
	c["fecha_coincide"] = false
	rec := e.post(t, fmt.Sprintf("/comprobantes/%d/rechazar", id), map[string]any{
		"motivo_rechazo": "No cumple: monto coincide; fecha coincide.", "checklist": c,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("código %d: %s", rec.Code, rec.Body)
	}
	if g, cp := e.estados(t, gastoID, id); g != "Registrado" || cp != "Rechazado" {
		t.Fatalf("estados gasto=%s comprobante=%s", g, cp)
	}
	if total, no := e.filas(t, id, false), e.filas(t, id, true); total != 5 || no != 2 {
		t.Fatalf("filas total=%d incumplidas=%d", total, no)
	}
}

func TestRechazarConClaveDesconocidaDa422(t *testing.T) {
	e := nuevoEntorno(t)
	gastoID, id := e.comprobantePendiente(t)
	c := completo()
	c["sello"] = true
	rec := e.post(t, fmt.Sprintf("/comprobantes/%d/rechazar", id), map[string]any{"motivo_rechazo": "x", "checklist": c})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("código %d: %s", rec.Code, rec.Body)
	}
	if g, cp := e.estados(t, gastoID, id); g != "Pagado" || cp != "Pendiente" {
		t.Fatalf("estados gasto=%s comprobante=%s", g, cp)
	}
}

func TestAprobarComprobanteInexistenteDa404(t *testing.T) {
	e := nuevoEntorno(t)
	if rec := e.post(t, "/comprobantes/999999/aprobar", map[string]any{"checklist": completo()}); rec.Code != http.StatusNotFound {
		t.Fatalf("código %d: %s", rec.Code, rec.Body)
	}
}

func TestListarDevuelveChecklistOrdenado(t *testing.T) {
	e := nuevoEntorno(t)
	_, revisado := e.comprobantePendiente(t)
	_, pendiente := e.comprobantePendiente(t)
	if rec := e.post(t, fmt.Sprintf("/comprobantes/%d/aprobar", revisado), map[string]any{"checklist": completo()}); rec.Code != http.StatusOK {
		t.Fatalf("aprobación: %d", rec.Code)
	}
	rec := e.get("/comprobantes")
	var lista []models.Comprobante
	if err := json.Unmarshal(rec.Body.Bytes(), &lista); err != nil {
		t.Fatal(err)
	}
	porID := map[int]models.Comprobante{}
	for _, c := range lista {
		porID[c.ID] = c
	}
	got := porID[revisado].Checklist
	if len(got) != 5 {
		t.Fatalf("checklist del revisado: %v", got)
	}
	for i, it := range got {
		if it.Criterio != criteriosEstandar[i] || !it.Cumple || it.Texto == "" {
			t.Fatalf("orden o valor inesperado: %v", got)
		}
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"checklist":[]`)) || len(porID[pendiente].Checklist) != 0 {
		t.Fatalf("el pendiente debía traer checklist vacío como lista: %s", rec.Body)
	}
}
