# Checklist documental del comprobante — plan de implementación

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Antes de aprobar un comprobante, Logística marca 5 criterios documentales; el servidor exige los 5 para aprobar, guarda el checklist y lo muestra en el historial.

**Architecture:** gastos-service (Go) gana un paquete puro `internal/checklist` con las claves y las validaciones, una tabla `checklist_comprobante` y validación transaccional (`FOR UPDATE`, solo comprobantes Pendiente) en aprobar y rechazar; el listado devuelve el checklist de cada comprobante. En Angular, `core/models/checklist.ts` define criterios y lógica pura; la revisión del comprobante muestra las casillas y el detalle muestra el checklist guardado.

**Tech Stack:** Go 1.25 + pgx v5 + PostgreSQL; Angular 21 (zoneless, signals) + Vitest; Playwright (`playwright-core` con el Chrome del sistema) para la verificación en navegador.

**Spec:** `docs/superpowers/specs/2026-10-10-checklist-documental-design.md`

## Global Constraints

- Se trabaja en `main` sin ramas ni commits (decisión del usuario). Los pasos «Checkpoint» del plan no hacen commit.
- No se modifican `frontend/src/app/core/models/gasto-estados.ts` ni `frontend/src/app/core/utils/error-messages.ts` (copia exacta RNF-1).
- Estos textos existentes no cambian: «Comprobante aprobado. El gasto quedó como "Sustentado".», «Comprobante rechazado. El gasto volvió a estado "Registrado".» y «Debes indicar un motivo para rechazar el comprobante.».
- Claves de criterio, en este orden: `legible`, `tipo_valido`, `a_nombre_empresa`, `monto_coincide`, `fecha_coincide`.
- Estilos solo en `frontend/src/styles/*.css`, con tokens existentes; sin colores nuevos.
- Go: sin dependencias nuevas (biblioteca estándar + pgx).
- Las pruebas de integración de Go usan `TEST_DATABASE_URL` y un esquema temporal propio que se borra al terminar; nunca escriben en las tablas reales.
- El `ng serve` del usuario en el puerto 4200 no se toca; las verificaciones en navegador usan `ng serve --port 4300`.
- Comandos: frontend `cd frontend && npx ng test --watch=false` y `npx ng build`; Go `cd gastos-service && go test ./... -count=1`.

## Review Focus

1. **Dos aprobaciones del mismo comprobante** (doble clic o dos pestañas): la segunda debe recibir 409 y no duplicar filas. Lo fijan `FOR UPDATE` + `UNIQUE (comprobante_id, criterio)` y la prueba `TestRevisarComprobanteYaRevisadoDa409` (Tarea 2).
2. **Llega un comprobante nuevo a la misma revisión** (recarga del store con otro pendiente): el checklist debe volver a vacío; con el mismo comprobante (recarga con el mismo id) debe conservar lo marcado. Lo fija la prueba «el checklist vuelve a vacío solo si cambia el comprobante» (Tarea 4).
3. **Un cliente manda una clave desconocida o le falta una** al rechazar: 422 claro, sin cambios en la base. Lo fijan `TestRechazo*` (Tarea 1) y `TestRechazarConClaveDesconocidaDa422` (Tarea 2).
4. **Rechazo con los 5 criterios marcados** (se rechaza por otra razón): el motivo no se propone y sigue siendo obligatorio. Lo fija la prueba «rechazar exige un motivo…» actualizada (Tarea 4).
5. **Comprobantes revisados antes de este módulo** (sin checklist): el historial no debe mostrar una lista vacía ni «Revisado por». Lo fija la prueba del historial sin checklist (Tarea 5).

---

### Task 1: Paquete `checklist` en Go (claves y validaciones puras)

**Files:**
- Modify: `gastos-service/internal/models/models.go`
- Create: `gastos-service/internal/checklist/checklist.go`
- Test: `gastos-service/internal/checklist/checklist_test.go`

**Interfaces:**
- Consumes: nada.
- Produces:
  - `models.ItemChecklist{ Criterio string; Cumple bool }` (JSON `criterio`, `cumple`) y el campo `models.Comprobante.Checklist []ItemChecklist` (JSON `checklist`).
  - `checklist.Criterios []string` (orden fijo).
  - `checklist.ValidarAprobacion(map[string]bool) error` → `nil`, `checklist.ErrIncompleto` o `checklist.ErrCriterioDesconocido{Clave}`.
  - `checklist.ValidarRechazo(map[string]bool) error` → `nil` (también con mapa nil), `checklist.ErrFaltanCriterios` o `ErrCriterioDesconocido`.
  - `checklist.Items(map[string]bool) []models.ItemChecklist` (en orden) y `checklist.Ordenar([]models.ItemChecklist) []models.ItemChecklist`.

- [ ] **Step 1: Escribir las pruebas que fallan**

`gastos-service/internal/checklist/checklist_test.go`:

```go
package checklist

import (
	"errors"
	"reflect"
	"testing"

	"gastos-service/internal/models"
)

func completo() map[string]bool {
	m := map[string]bool{}
	for _, c := range Criterios {
		m[c] = true
	}
	return m
}

func TestCriteriosEnOrden(t *testing.T) {
	want := []string{"legible", "tipo_valido", "a_nombre_empresa", "monto_coincide", "fecha_coincide"}
	if !reflect.DeepEqual(Criterios, want) {
		t.Fatalf("Criterios = %v, se esperaba %v", Criterios, want)
	}
}

func TestAprobacionCompletaEsValida(t *testing.T) {
	if err := ValidarAprobacion(completo()); err != nil {
		t.Fatalf("se esperaba nil, llegó %v", err)
	}
}

func TestAprobacionSinChecklistOIncompleta(t *testing.T) {
	falta := completo()
	delete(falta, "monto_coincide")
	enFalso := completo()
	enFalso["fecha_coincide"] = false
	for nombre, c := range map[string]map[string]bool{"sin checklist": nil, "falta uno": falta, "uno en false": enFalso} {
		if err := ValidarAprobacion(c); !errors.Is(err, ErrIncompleto) {
			t.Errorf("%s: se esperaba ErrIncompleto, llegó %v", nombre, err)
		}
	}
	if ErrIncompleto.Error() != "para aprobar, el comprobante debe cumplir los 5 criterios del checklist" {
		t.Errorf("mensaje inesperado: %q", ErrIncompleto.Error())
	}
}

func TestCriterioDesconocido(t *testing.T) {
	c := completo()
	c["sello"] = true
	var e ErrCriterioDesconocido
	if err := ValidarAprobacion(c); !errors.As(err, &e) || e.Clave != "sello" {
		t.Fatalf("se esperaba ErrCriterioDesconocido{sello}, llegó %v", err)
	}
	if e.Error() != "criterio desconocido: sello" {
		t.Errorf("mensaje inesperado: %q", e.Error())
	}
}

func TestRechazoSinChecklistOConCriteriosEnFalseEsValido(t *testing.T) {
	if err := ValidarRechazo(nil); err != nil {
		t.Errorf("sin checklist: se esperaba nil, llegó %v", err)
	}
	c := completo()
	c["monto_coincide"] = false
	c["fecha_coincide"] = false
	if err := ValidarRechazo(c); err != nil {
		t.Errorf("con criterios en false: se esperaba nil, llegó %v", err)
	}
}

func TestRechazoConChecklistIncompletoOClaveDesconocida(t *testing.T) {
	falta := completo()
	delete(falta, "legible")
	if err := ValidarRechazo(falta); !errors.Is(err, ErrFaltanCriterios) {
		t.Errorf("falta uno: se esperaba ErrFaltanCriterios, llegó %v", err)
	}
	extra := completo()
	extra["firma"] = false
	var e ErrCriterioDesconocido
	if err := ValidarRechazo(extra); !errors.As(err, &e) {
		t.Errorf("clave desconocida: se esperaba ErrCriterioDesconocido, llegó %v", err)
	}
}

func TestItemsYOrdenar(t *testing.T) {
	c := completo()
	c["tipo_valido"] = false
	items := Items(c)
	if len(items) != 5 || items[0].Criterio != "legible" || items[1].Criterio != "tipo_valido" || items[1].Cumple {
		t.Fatalf("Items en orden inesperado: %v", items)
	}
	desordenados := []models.ItemChecklist{{Criterio: "fecha_coincide", Cumple: true}, {Criterio: "legible", Cumple: true}, {Criterio: "monto_coincide"}}
	got := Ordenar(desordenados)
	want := []string{"legible", "monto_coincide", "fecha_coincide"}
	for i, it := range got {
		if it.Criterio != want[i] {
			t.Fatalf("Ordenar = %v, se esperaba el orden %v", got, want)
		}
	}
}
```

- [ ] **Step 2: Ejecutar y verificar que fallan**

Run: `cd gastos-service && go test ./internal/checklist/ -count=1`
Expected: FAIL de compilación: `undefined: Criterios`, `ValidarAprobacion`, `models.ItemChecklist`…

- [ ] **Step 3: Agregar el tipo al modelo**

En `gastos-service/internal/models/models.go`, reemplaza el struct `Comprobante` completo por:

```go
// Comprobante representa un comprobante de pago adjuntado a un gasto.
type Comprobante struct {
	ID            int             `json:"id"`
	GastoID       int             `json:"gasto_id"`
	ArchivoURL    string          `json:"archivo_url"`
	TipoArchivo   string          `json:"tipo_archivo"`
	Estado        string          `json:"estado"`
	MotivoRechazo *string         `json:"motivo_rechazo,omitempty"`
	AprobadoPor   *string         `json:"aprobado_por,omitempty"`
	AprobadoEn    *time.Time      `json:"aprobado_en,omitempty"`
	CreadoEn      time.Time       `json:"creado_en"`
	Checklist     []ItemChecklist `json:"checklist"`
}

// ItemChecklist es un criterio del checklist documental con su resultado.
type ItemChecklist struct {
	Criterio string `json:"criterio"`
	Cumple   bool   `json:"cumple"`
}
```

- [ ] **Step 4: Implementar el paquete**

`gastos-service/internal/checklist/checklist.go`:

```go
// Package checklist define el checklist documental que Logística revisa
// antes de aprobar un comprobante: las claves de sus criterios y las reglas
// para aprobar o rechazar. No toca la base de datos.
package checklist

import (
	"errors"
	"sort"

	"gastos-service/internal/models"
)

// Criterios del checklist documental, en el orden en que se revisan y se devuelven.
var Criterios = []string{"legible", "tipo_valido", "a_nombre_empresa", "monto_coincide", "fecha_coincide"}

// ErrIncompleto: para aprobar deben venir los 5 criterios y todos cumplidos.
var ErrIncompleto = errors.New("para aprobar, el comprobante debe cumplir los 5 criterios del checklist")

// ErrFaltanCriterios: un checklist enviado al rechazar debe traer los 5 criterios.
var ErrFaltanCriterios = errors.New("el checklist debe incluir los 5 criterios")

// ErrCriterioDesconocido señala una clave que no es un criterio del checklist.
type ErrCriterioDesconocido struct{ Clave string }

func (e ErrCriterioDesconocido) Error() string { return "criterio desconocido: " + e.Clave }

func esCriterio(clave string) bool {
	for _, c := range Criterios {
		if c == clave {
			return true
		}
	}
	return false
}

// validarClaves revisa las claves en orden alfabético para que el error sea siempre el mismo.
func validarClaves(c map[string]bool) error {
	claves := make([]string, 0, len(c))
	for clave := range c {
		claves = append(claves, clave)
	}
	sort.Strings(claves)
	for _, clave := range claves {
		if !esCriterio(clave) {
			return ErrCriterioDesconocido{Clave: clave}
		}
	}
	return nil
}

// ValidarAprobacion exige los 5 criterios presentes y cumplidos.
func ValidarAprobacion(c map[string]bool) error {
	if err := validarClaves(c); err != nil {
		return err
	}
	for _, clave := range Criterios {
		if !c[clave] {
			return ErrIncompleto
		}
	}
	return nil
}

// ValidarRechazo acepta un rechazo sin checklist; si viene, debe traer los 5 criterios (cumplidos o no).
func ValidarRechazo(c map[string]bool) error {
	if c == nil {
		return nil
	}
	if err := validarClaves(c); err != nil {
		return err
	}
	for _, clave := range Criterios {
		if _, ok := c[clave]; !ok {
			return ErrFaltanCriterios
		}
	}
	return nil
}

// Items convierte el checklist recibido en filas, en el orden de Criterios.
func Items(c map[string]bool) []models.ItemChecklist {
	out := make([]models.ItemChecklist, 0, len(Criterios))
	for _, clave := range Criterios {
		if cumple, ok := c[clave]; ok {
			out = append(out, models.ItemChecklist{Criterio: clave, Cumple: cumple})
		}
	}
	return out
}

// Ordenar devuelve las filas en el orden de Criterios.
func Ordenar(items []models.ItemChecklist) []models.ItemChecklist {
	pos := map[string]int{}
	for i, c := range Criterios {
		pos[c] = i
	}
	out := append([]models.ItemChecklist(nil), items...)
	sort.SliceStable(out, func(i, j int) bool { return pos[out[i].Criterio] < pos[out[j].Criterio] })
	return out
}
```

- [ ] **Step 5: Ejecutar y verificar que pasan**

Run: `cd gastos-service && go test ./internal/checklist/ -count=1 -v`
Expected: PASS (7 pruebas). También `go build ./...` sin errores.

- [ ] **Step 6: Checkpoint (sin commit)**

Run: `cd gastos-service && gofmt -l . && go vet ./...`
Expected: sin salida de `gofmt -l` y `go vet` sin hallazgos.

---

### Task 2: Migración, aprobar/rechazar con checklist y listado (Go)

**Files:**
- Create: `gastos-service/migrations/0002_checklist_comprobante.sql`
- Modify: `gastos-service/internal/handlers/comprobantes.go` (imports; `uploadComprobante` antes de `writeJSON`; reemplazo desde `// listComprobantes soporta` hasta el final del archivo)
- Test: `gastos-service/internal/handlers/checklist_integration_test.go`

**Interfaces:**
- Consumes: `checklist.Criterios`, `ValidarAprobacion`, `ValidarRechazo`, `Items`, `Ordenar`, `models.ItemChecklist` (Tarea 1).
- Produces (contrato HTTP, sección 6 de la spec):
  - `POST /comprobantes/{id}/aprobar` con `{"checklist":{…5 en true…}}` → 200 `{"status":"aprobado"}`; 400 cuerpo no JSON; 422 incompleto o clave desconocida; 404 inexistente; 409 no Pendiente.
  - `POST /comprobantes/{id}/rechazar` con `{"motivo_rechazo":"…","checklist":{…}}` (checklist opcional) → 200 `{"status":"rechazado"}`; 400 sin motivo; 422; 404; 409.
  - `GET /comprobantes` → cada comprobante con `checklist` ordenado (lista vacía si no tiene).

- [ ] **Step 1: Escribir las pruebas de integración que fallan**

`gastos-service/internal/handlers/checklist_integration_test.go`:

```go
package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"gastos-service/internal/checklist"
	"gastos-service/internal/config"
	"gastos-service/internal/db"
	"gastos-service/internal/handlers"
	"gastos-service/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type entorno struct {
	pool *pgxpool.Pool
	mux  *http.ServeMux
}

// nuevoEntorno arma un esquema temporal con tablas mínimas de viaje y cliente,
// aplica las migraciones del servicio y borra el esquema al terminar.
func nuevoEntorno(t *testing.T) *entorno {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL no definida: se omiten las pruebas contra PostgreSQL")
	}
	ctx := context.Background()
	esquema := fmt.Sprintf("checklist_prueba_%d", time.Now().UnixNano())

	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+esquema); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec(context.Background(), "DROP SCHEMA "+esquema+" CASCADE")
		admin.Close()
	})

	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = esquema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	if _, err := pool.Exec(ctx, `CREATE TABLE programacionviaje (id_viaje SERIAL PRIMARY KEY);
		CREATE TABLE cliente (id_cliente SERIAL PRIMARY KEY);`); err != nil {
		t.Fatal(err)
	}
	if err := db.RunMigrations(pool, "../../migrations"); err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	handlers.New(pool, config.Config{UploadsDir: t.TempDir()}).Register(mux)
	return &entorno{pool: pool, mux: mux}
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
	req.Header.Set("X-User-Role", "logistica")
	req.Header.Set("X-User-Name", "Ana Quispe")
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

func completo() map[string]bool {
	m := map[string]bool{}
	for _, c := range checklist.Criterios {
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
	req := httptest.NewRequest(http.MethodGet, "/comprobantes", nil)
	rec := httptest.NewRecorder()
	e.mux.ServeHTTP(rec, req)
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
		if it.Criterio != checklist.Criterios[i] || !it.Cumple {
			t.Fatalf("orden o valor inesperado: %v", got)
		}
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"checklist":[]`)) || len(porID[pendiente].Checklist) != 0 {
		t.Fatalf("el pendiente debía traer checklist vacío como lista: %s", rec.Body)
	}
}
```

- [ ] **Step 2: Ejecutar y verificar que fallan**

Run:
```bash
cd gastos-service
TEST_DATABASE_URL="$(sed -n 's/^DATABASE_URL=//p' .env | tr -d "\"'")" go test ./internal/handlers/ -count=1
```
Expected: FAIL. Las pruebas compilan (Tarea 1 ya existe) y fallan por el comportamiento: aprobar incompleto da 200, un segundo rechazo da 200, `relation "checklist_comprobante" does not exist`, el listado no trae `checklist`. Si el error es de permisos al crear el esquema, regístralo como ruling y usa una base donde el usuario pueda crear esquemas.

- [ ] **Step 3: Crear la migración**

`gastos-service/migrations/0002_checklist_comprobante.sql`:

```sql
-- Módulo de checklist documental: lo que Logística verificó en cada comprobante.
-- Quién y cuándo revisó ya viven en comprobante.aprobado_por y comprobante.aprobado_en.

CREATE TABLE IF NOT EXISTS checklist_comprobante (
    id             SERIAL PRIMARY KEY,
    comprobante_id INTEGER NOT NULL REFERENCES comprobante (id),
    criterio       VARCHAR(30) NOT NULL
                   CHECK (criterio IN ('legible', 'tipo_valido', 'a_nombre_empresa', 'monto_coincide', 'fecha_coincide')),
    cumple         BOOLEAN NOT NULL,
    CONSTRAINT uq_checklist_criterio UNIQUE (comprobante_id, criterio)
);

CREATE INDEX IF NOT EXISTS idx_checklist_comprobante_id ON checklist_comprobante (comprobante_id);
```

- [ ] **Step 4: Actualizar los imports y la subida**

En `gastos-service/internal/handlers/comprobantes.go`, reemplaza el bloque `import (…)` por:

```go
import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gastos-service/internal/checklist"
	"gastos-service/internal/models"

	"github.com/jackc/pgx/v5"
)
```

En `uploadComprobante`, reemplaza la última línea `writeJSON(w, http.StatusCreated, c)` por:

```go
	c.Checklist = []models.ItemChecklist{}
	writeJSON(w, http.StatusCreated, c)
```

- [ ] **Step 5: Reemplazar el listado, la aprobación y el rechazo**

En el mismo archivo, reemplaza todo desde la línea `// listComprobantes soporta la pantalla de aprobación (filtrando por` hasta el final del archivo por:

```go
// listComprobantes soporta la pantalla de aprobación (filtrando por
// ?estado=Pendiente) y el historial; cada comprobante trae su checklist.
func (h *Handler) listComprobantes(w http.ResponseWriter, r *http.Request) {
	estado := r.URL.Query().Get("estado")
	ctx := context.Background()

	query := `
		SELECT id, gasto_id, archivo_url, tipo_archivo, estado, motivo_rechazo, aprobado_por, aprobado_en, creado_en
		FROM comprobante`
	args := []any{}
	if estado != "" {
		query += ` WHERE estado = $1`
		args = append(args, estado)
	}
	query += ` ORDER BY creado_en DESC LIMIT 200`

	rows, err := h.DB.Query(ctx, query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo listar los comprobantes")
		return
	}
	defer rows.Close()

	out := []models.Comprobante{}
	ids := []int{}
	for rows.Next() {
		var c models.Comprobante
		if err := rows.Scan(&c.ID, &c.GastoID, &c.ArchivoURL, &c.TipoArchivo, &c.Estado, &c.MotivoRechazo, &c.AprobadoPor, &c.AprobadoEn, &c.CreadoEn); err != nil {
			writeError(w, http.StatusInternalServerError, "error leyendo los comprobantes")
			return
		}
		c.Checklist = []models.ItemChecklist{}
		out = append(out, c)
		ids = append(ids, c.ID)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "error leyendo los comprobantes")
		return
	}
	rows.Close()

	if err := h.adjuntarChecklists(ctx, out, ids); err != nil {
		writeError(w, http.StatusInternalServerError, "error leyendo los checklists")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// adjuntarChecklists carga en una sola consulta el checklist de los comprobantes listados.
func (h *Handler) adjuntarChecklists(ctx context.Context, comprobantes []models.Comprobante, ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	rows, err := h.DB.Query(ctx, `SELECT comprobante_id, criterio, cumple FROM checklist_comprobante WHERE comprobante_id = ANY($1)`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()

	porComprobante := map[int][]models.ItemChecklist{}
	for rows.Next() {
		var id int
		var it models.ItemChecklist
		if err := rows.Scan(&id, &it.Criterio, &it.Cumple); err != nil {
			return err
		}
		porComprobante[id] = append(porComprobante[id], it)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for i := range comprobantes {
		if items, ok := porComprobante[comprobantes[i].ID]; ok {
			comprobantes[i].Checklist = checklist.Ordenar(items)
		}
	}
	return nil
}

var (
	errComprobanteNoEncontrado = errors.New("comprobante no encontrado")
	errYaRevisado              = errors.New("el comprobante ya fue revisado")
)

// bloquearPendiente toma el comprobante con FOR UPDATE y exige que siga Pendiente;
// devuelve el gasto al que pertenece.
func bloquearPendiente(ctx context.Context, tx pgx.Tx, id int) (int, error) {
	var gastoID int
	var estado string
	err := tx.QueryRow(ctx, `SELECT gasto_id, estado FROM comprobante WHERE id = $1 FOR UPDATE`, id).Scan(&gastoID, &estado)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, errComprobanteNoEncontrado
	}
	if err != nil {
		return 0, err
	}
	if estado != "Pendiente" {
		return 0, errYaRevisado
	}
	return gastoID, nil
}

func guardarChecklist(ctx context.Context, tx pgx.Tx, id int, items []models.ItemChecklist) error {
	for _, it := range items {
		if _, err := tx.Exec(ctx, `INSERT INTO checklist_comprobante (comprobante_id, criterio, cumple) VALUES ($1, $2, $3)`, id, it.Criterio, it.Cumple); err != nil {
			return err
		}
	}
	return nil
}

func escribirErrorRevision(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errComprobanteNoEncontrado):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, errYaRevisado):
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "no se pudo completar la revisión")
	}
}

type aprobarRequest struct {
	AprobadoPor string          `json:"aprobado_por"`
	Checklist   map[string]bool `json:"checklist"`
}

// aprobarComprobante implementa RF-7 (aprobar) para el rol Encargado de
// Logística: exige el checklist documental completo y hace avanzar el gasto
// a "Sustentado".
func (h *Handler) aprobarComprobante(w http.ResponseWriter, r *http.Request) {
	nombre, autorizado := requireLogistica(r)
	if !autorizado {
		writeError(w, http.StatusForbidden, "solo el Encargado de Logística puede aprobar comprobantes")
		return
	}

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id inválido")
		return
	}

	var req aprobarRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	if err := checklist.ValidarAprobacion(req.Checklist); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if req.AprobadoPor != "" {
		nombre = req.AprobadoPor
	}

	ctx := context.Background()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo iniciar la operación")
		return
	}
	defer tx.Rollback(ctx)

	gastoID, err := bloquearPendiente(ctx, tx, id)
	if err != nil {
		escribirErrorRevision(w, err)
		return
	}
	if err := guardarChecklist(ctx, tx, id, checklist.Items(req.Checklist)); err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo guardar el checklist")
		return
	}
	if _, err := tx.Exec(ctx, `UPDATE comprobante SET estado = 'Aprobado', aprobado_por = $1, aprobado_en = now() WHERE id = $2`, nombre, id); err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo aprobar el comprobante")
		return
	}
	if _, err := tx.Exec(ctx, `UPDATE gasto SET estado = 'Sustentado' WHERE id = $1`, gastoID); err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo actualizar el gasto")
		return
	}

	if err := tx.Commit(ctx); err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo confirmar la operación")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "aprobado"})
}

type rechazarRequest struct {
	MotivoRechazo string          `json:"motivo_rechazo"`
	AprobadoPor   string          `json:"aprobado_por"`
	Checklist     map[string]bool `json:"checklist"`
}

// rechazarComprobante implementa RF-8 (motivo obligatorio) y RF-9 (el gasto
// vuelve a "Registrado"); guarda el checklist si viene.
func (h *Handler) rechazarComprobante(w http.ResponseWriter, r *http.Request) {
	nombre, autorizado := requireLogistica(r)
	if !autorizado {
		writeError(w, http.StatusForbidden, "solo el Encargado de Logística puede rechazar comprobantes")
		return
	}

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id inválido")
		return
	}

	var req rechazarRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	if strings.TrimSpace(req.MotivoRechazo) == "" {
		writeError(w, http.StatusBadRequest, "motivo_rechazo es obligatorio")
		return
	}
	if err := checklist.ValidarRechazo(req.Checklist); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if req.AprobadoPor != "" {
		nombre = req.AprobadoPor
	}

	ctx := context.Background()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo iniciar la operación")
		return
	}
	defer tx.Rollback(ctx)

	gastoID, err := bloquearPendiente(ctx, tx, id)
	if err != nil {
		escribirErrorRevision(w, err)
		return
	}
	if err := guardarChecklist(ctx, tx, id, checklist.Items(req.Checklist)); err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo guardar el checklist")
		return
	}
	if _, err := tx.Exec(ctx, `UPDATE comprobante SET estado = 'Rechazado', motivo_rechazo = $1, aprobado_por = $2, aprobado_en = now() WHERE id = $3`, req.MotivoRechazo, nombre, id); err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo rechazar el comprobante")
		return
	}
	if _, err := tx.Exec(ctx, `UPDATE gasto SET estado = 'Registrado' WHERE id = $1`, gastoID); err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo actualizar el gasto")
		return
	}

	if err := tx.Commit(ctx); err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo confirmar la operación")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "rechazado"})
}
```

- [ ] **Step 6: Ejecutar y verificar que pasan**

Run:
```bash
cd gastos-service
TEST_DATABASE_URL="$(sed -n 's/^DATABASE_URL=//p' .env | tr -d "\"'")" go test ./... -count=1 -v 2>&1 | tail -30
```
Expected: PASS de las 7 pruebas de `internal/checklist` y las 7 de `internal/handlers` (ninguna en SKIP). Después, comprueba que no quedó ningún esquema de prueba:
```bash
psql "$(sed -n 's/^DATABASE_URL=//p' .env | tr -d "\"'")" -tAc "SELECT count(*) FROM information_schema.schemata WHERE schema_name LIKE 'checklist_prueba_%'"
```
Expected: `0`.

- [ ] **Step 7: Checkpoint (sin commit)**

Run: `cd gastos-service && gofmt -l . && go vet ./... && go build ./...`
Expected: sin salida de `gofmt -l`; vet y build sin errores. El servicio del usuario en :8081 sigue con el binario anterior hasta que lo reinicie; anótalo para el informe final.

---

### Task 3: Modelo del checklist en Angular (criterios y lógica pura)

**Files:**
- Modify: `frontend/src/app/core/models/gasto.model.ts`
- Create: `frontend/src/app/core/models/checklist.ts`
- Test: `frontend/src/app/core/models/checklist.spec.ts`

**Interfaces:**
- Consumes: `formatMonto`, `formatFecha` (`core/utils/format.ts`); `Gasto`.
- Produces:
  - En `gasto.model.ts`: `type CriterioChecklist`, `interface ItemChecklist { criterio: CriterioChecklist; cumple: boolean }` y `Comprobante.checklist?: ItemChecklist[] | null`.
  - En `checklist.ts`: `type Checklist = Record<CriterioChecklist, boolean>`; `CRITERIOS_CHECKLIST: readonly { clave; corto; texto(gasto) }[]`; `TOTAL_CRITERIOS`; `checklistVacio()`; `cumplidos(c)`; `checklistCompleto(c)`; `motivoDesdeChecklist(c)`; `cortoDe(clave)`; `MSG_COMPROBANTE_YA_REVISADO`; `MSG_CHECKLIST_INCOMPLETO`.

- [ ] **Step 1: Escribir la prueba que falla**

`frontend/src/app/core/models/checklist.spec.ts`:

```ts
import { describe, expect, it } from 'vitest';
import {
  CRITERIOS_CHECKLIST, Checklist, checklistCompleto, checklistVacio, cortoDe, cumplidos, motivoDesdeChecklist,
} from './checklist';
import { Gasto } from './gasto.model';

const gasto: Gasto = {
  id: 30, viaje_id: 14, cliente_id: 1, monto: 85.5, fecha: '2026-09-30T00:00:00Z', estado: 'Pagado', creado_por: 'Ana', creado_en: '2026-10-01T13:00:00Z',
};

function con(valores: Partial<Checklist>): Checklist {
  return { ...checklistVacio(), ...valores };
}

describe('checklist documental', () => {
  it('las claves coinciden con gastos-service, en el mismo orden', () => {
    expect(CRITERIOS_CHECKLIST.map(c => c.clave)).toEqual(['legible', 'tipo_valido', 'a_nombre_empresa', 'monto_coincide', 'fecha_coincide']);
  });

  it('los criterios de monto y fecha muestran el dato del gasto', () => {
    const textos = CRITERIOS_CHECKLIST.map(c => c.texto(gasto));
    expect(textos).toEqual([
      'El comprobante es legible y está completo.',
      'Es una boleta o factura válida.',
      'Está emitido a nombre de LIZAR SRL.',
      'El monto coincide con el gasto (S/ 85.50).',
      'La fecha corresponde al gasto (30/09/2026).',
    ]);
  });

  it('cuenta lo marcado y solo está completo con los 5', () => {
    expect(cumplidos(checklistVacio())).toBe(0);
    expect(checklistCompleto(checklistVacio())).toBe(false);
    const cuatro = con({ legible: true, tipo_valido: true, a_nombre_empresa: true, monto_coincide: true });
    expect(cumplidos(cuatro)).toBe(4);
    expect(checklistCompleto(cuatro)).toBe(false);
    expect(checklistCompleto({ ...cuatro, fecha_coincide: true })).toBe(true);
  });

  it('propone el motivo con los criterios sin marcar, y nada si cumple todos', () => {
    expect(motivoDesdeChecklist(con({ legible: true, tipo_valido: true, a_nombre_empresa: true }))).toBe('No cumple: monto coincide; fecha coincide.');
    expect(motivoDesdeChecklist(con({ monto_coincide: true, fecha_coincide: true, legible: true }))).toBe(
      'No cumple: boleta o factura válida; a nombre de LIZAR SRL.',
    );
    const todo = con({ legible: true, tipo_valido: true, a_nombre_empresa: true, monto_coincide: true, fecha_coincide: true });
    expect(motivoDesdeChecklist(todo)).toBe('');
  });

  it('el texto corto sirve para el historial', () => {
    expect(cortoDe('monto_coincide')).toBe('Monto coincide');
    expect(cortoDe('a_nombre_empresa')).toBe('A nombre de LIZAR SRL');
  });
});
```

- [ ] **Step 2: Ejecutar y verificar que falla**

Run: `cd frontend && npx ng test --watch=false --include src/app/core/models/checklist.spec.ts`
Expected: FAIL: no se puede resolver `./checklist`.

- [ ] **Step 3: Agregar los tipos al modelo**

En `frontend/src/app/core/models/gasto.model.ts`, después de `export type EstadoComprobante = …;` agrega:

```ts
/** Criterios del checklist documental (las claves coinciden con gastos-service). */
export type CriterioChecklist = 'legible' | 'tipo_valido' | 'a_nombre_empresa' | 'monto_coincide' | 'fecha_coincide';

export interface ItemChecklist {
  criterio: CriterioChecklist;
  cumple: boolean;
}
```

y dentro de `interface Comprobante`, después de `creado_en: string;`, agrega:

```ts
  /** Lo que Logística verificó al revisarlo; vacío en comprobantes revisados antes del checklist */
  checklist?: ItemChecklist[] | null;
```

- [ ] **Step 4: Implementar el módulo**

`frontend/src/app/core/models/checklist.ts`:

```ts
import { CriterioChecklist, Gasto } from './gasto.model';
import { formatFecha, formatMonto } from '../utils/format';

/** Checklist documental en curso: un valor por criterio. */
export type Checklist = Record<CriterioChecklist, boolean>;

interface DefinicionCriterio {
  clave: CriterioChecklist;
  /** Texto corto del historial */
  corto: string;
  /** Texto de la revisión; el monto y la fecha salen del gasto para comparar */
  texto: (gasto: Gasto) => string;
}

/** Criterios del checklist documental, en el orden en que se revisan. */
export const CRITERIOS_CHECKLIST: readonly DefinicionCriterio[] = [
  { clave: 'legible', corto: 'Legible y completo', texto: () => 'El comprobante es legible y está completo.' },
  { clave: 'tipo_valido', corto: 'Boleta o factura válida', texto: () => 'Es una boleta o factura válida.' },
  { clave: 'a_nombre_empresa', corto: 'A nombre de LIZAR SRL', texto: () => 'Está emitido a nombre de LIZAR SRL.' },
  { clave: 'monto_coincide', corto: 'Monto coincide', texto: g => `El monto coincide con el gasto (${formatMonto(g.monto)}).` },
  { clave: 'fecha_coincide', corto: 'Fecha coincide', texto: g => `La fecha corresponde al gasto (${formatFecha(g.fecha)}).` },
];

export const TOTAL_CRITERIOS = CRITERIOS_CHECKLIST.length;

export const MSG_COMPROBANTE_YA_REVISADO = 'Este comprobante ya fue revisado. Actualizamos la vista.';
export const MSG_CHECKLIST_INCOMPLETO = 'Para aprobar, el comprobante debe cumplir los 5 criterios del checklist.';

export function checklistVacio(): Checklist {
  return { legible: false, tipo_valido: false, a_nombre_empresa: false, monto_coincide: false, fecha_coincide: false };
}

export function cumplidos(checklist: Checklist): number {
  return CRITERIOS_CHECKLIST.filter(c => checklist[c.clave]).length;
}

export function checklistCompleto(checklist: Checklist): boolean {
  return cumplidos(checklist) === TOTAL_CRITERIOS;
}

/** «No cumple: monto coincide; fecha coincide.» con lo que quedó sin marcar; vacío si cumple todo. */
export function motivoDesdeChecklist(checklist: Checklist): string {
  const faltan = CRITERIOS_CHECKLIST.filter(c => !checklist[c.clave]).map(c => c.corto.charAt(0).toLowerCase() + c.corto.slice(1));
  return faltan.length ? `No cumple: ${faltan.join('; ')}.` : '';
}

export function cortoDe(clave: CriterioChecklist): string {
  return CRITERIOS_CHECKLIST.find(c => c.clave === clave)?.corto ?? clave;
}
```

- [ ] **Step 5: Ejecutar y verificar que pasa**

Run: `cd frontend && npx ng test --watch=false --include src/app/core/models/checklist.spec.ts`
Expected: PASS (5 pruebas).

- [ ] **Step 6: Checkpoint (sin commit)**

Run: `cd frontend && npx ng test --watch=false`
Expected: PASS en todas las pruebas (nada más cambió de comportamiento).

---

### Task 4: Revisión del comprobante con checklist (store, componente y estilos)

**Files:**
- Modify: `frontend/src/app/core/state/gastos.store.ts` (`aprobar` y `rechazar`)
- Modify: `frontend/src/app/core/state/gastos.store.spec.ts` (prueba de mutaciones)
- Modify (reemplazo completo): `frontend/src/app/features/gastos/revision-comprobante/revision-comprobante.component.ts`
- Modify (reemplazo completo): `frontend/src/app/features/gastos/revision-comprobante/revision-comprobante.component.html`
- Modify (reemplazo completo): `frontend/src/app/features/gastos/revision-comprobante/revision-comprobante.component.spec.ts`
- Modify: `frontend/src/app/features/gastos/gasto-detalle/gasto-detalle.component.html` (pasar el gasto a la revisión)
- Modify: `frontend/src/styles/gastos.css` (estilos del checklist)

**Interfaces:**
- Consumes: todo lo de la Tarea 3; `HttpErrorResponse`; `mensajeError`.
- Produces: `GastosStore.aprobar(comprobanteId: number, checklist: Checklist)` envía `{ checklist }`; `GastosStore.rechazar(comprobanteId: number, motivo: string, checklist: Checklist)` envía `{ motivo_rechazo, checklist }`. `RevisionComprobanteComponent` con inputs `comprobante` y `gasto` (ambos requeridos).

- [ ] **Step 1: Escribir las pruebas que fallan**

En `frontend/src/app/core/state/gastos.store.spec.ts`, dentro de la prueba `'las mutaciones llaman a la API correcta y recargan'`, reemplaza:

```ts
    store.aprobar(9).subscribe();
    store.rechazar(9, 'Ilegible').subscribe();
```
por:
```ts
    const todo = { legible: true, tipo_valido: true, a_nombre_empresa: true, monto_coincide: true, fecha_coincide: true };
    const parcial = { ...todo, monto_coincide: false };
    store.aprobar(9, todo).subscribe();
    store.rechazar(9, 'Ilegible', parcial).subscribe();
```
y reemplaza:
```ts
    expect(api.post).toHaveBeenCalledWith('/comprobantes/9/aprobar', {});
    expect(api.post).toHaveBeenCalledWith('/comprobantes/9/rechazar', { motivo_rechazo: 'Ilegible' });
```
por:
```ts
    expect(api.post).toHaveBeenCalledWith('/comprobantes/9/aprobar', { checklist: todo });
    expect(api.post).toHaveBeenCalledWith('/comprobantes/9/rechazar', { motivo_rechazo: 'Ilegible', checklist: parcial });
```

Reemplaza completo `frontend/src/app/features/gastos/revision-comprobante/revision-comprobante.component.spec.ts`:

```ts
import { describe, expect, it, vi } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { HttpErrorResponse } from '@angular/common/http';
import { Observable, of, throwError } from 'rxjs';
import { RevisionComprobanteComponent } from './revision-comprobante.component';
import { GastosStore } from '../../../core/state/gastos.store';
import { GastosApiService } from '../../../core/services/gastos-api.service';
import { ToastService } from '../../../core/services/toast.service';
import { Comprobante, Gasto } from '../../../core/models/gasto.model';

const comprobante: Comprobante = {
  id: 9, gasto_id: 7, archivo_url: '/uploads/9.jpg', tipo_archivo: 'imagen', estado: 'Pendiente', creado_en: '2026-10-01T17:00:00Z',
};
const gasto: Gasto = {
  id: 7, viaje_id: 14, cliente_id: 1, monto: 85.5, fecha: '2026-09-30T00:00:00Z', estado: 'Pagado', creado_por: 'Ana', creado_en: '2026-10-01T13:00:00Z',
};
const TODO = { legible: true, tipo_valido: true, a_nombre_empresa: true, monto_coincide: true, fecha_coincide: true };

function montar(respuesta: () => Observable<unknown> = () => of({})) {
  const store = { aprobar: vi.fn(respuesta), rechazar: vi.fn(respuesta), cargar: vi.fn() };
  TestBed.configureTestingModule({
    providers: [
      { provide: GastosStore, useValue: store },
      { provide: GastosApiService, useValue: { fileUrl: (u: string) => `http://localhost:8081${u}` } },
    ],
  });
  const f = TestBed.createComponent(RevisionComprobanteComponent);
  f.componentRef.setInput('comprobante', comprobante);
  f.componentRef.setInput('gasto', gasto);
  f.detectChanges();
  const el = f.nativeElement as HTMLElement;
  const boton = (texto: string) => [...el.querySelectorAll('button')].find(b => b.textContent?.includes(texto)) as HTMLButtonElement;
  const casillas = () => [...el.querySelectorAll<HTMLInputElement>('.checklist input[type="checkbox"]')];
  const marcar = (cuantas: number) => {
    casillas().slice(0, cuantas).forEach(c => c.click());
    f.detectChanges();
  };
  return { f, el, store, boton, casillas, marcar };
}

describe('RevisionComprobanteComponent', () => {
  it('muestra el comprobante para revisarlo', () => {
    const { el } = montar();
    expect(el.querySelector('img')?.getAttribute('src')).toBe('http://localhost:8081/uploads/9.jpg');
  });

  it('muestra los 5 criterios con el monto y la fecha del gasto', () => {
    const { el, casillas } = montar();
    expect(el.querySelector('.checklist legend')?.textContent?.trim()).toBe('Checklist documental');
    expect(casillas()).toHaveLength(5);
    const textos = [...el.querySelectorAll('.checklist__criterio')].map(c => c.textContent?.trim());
    expect(textos).toContain('El monto coincide con el gasto (S/ 85.50).');
    expect(textos).toContain('La fecha corresponde al gasto (30/09/2026).');
  });

  it('Aprobar queda deshabilitado hasta marcar los 5 criterios', () => {
    const { el, boton, marcar } = montar();
    expect(boton('Aprobar').disabled).toBe(true);
    expect(el.querySelector('#checklist-aviso')?.textContent?.trim()).toBe('Marca los 5 criterios para aprobar.');
    expect(boton('Aprobar').getAttribute('aria-describedby')).toBe('checklist-aviso');
    marcar(4);
    expect(el.querySelector('.checklist__conteo')?.textContent?.trim()).toBe('4 de 5 criterios');
    expect(boton('Aprobar').disabled).toBe(true);
  });

  it('con los 5 criterios, aprobar envía el checklist y avisa con el copy exacto', () => {
    const { el, store, boton, marcar } = montar();
    marcar(5);
    expect(boton('Aprobar').disabled).toBe(false);
    expect(el.querySelector('#checklist-aviso')).toBeNull();
    boton('Aprobar').click();
    expect(store.aprobar).toHaveBeenCalledWith(9, TODO);
    expect(TestBed.inject(ToastService).toasts()[0].title).toBe('Comprobante aprobado. El gasto quedó como "Sustentado".');
  });

  it('Aprobar va en verde de guía, Rechazar es neutro y el rojo queda para confirmar el rechazo', () => {
    const { f, boton } = montar();
    expect(boton('Aprobar').classList).toContain('btn--guia');
    expect(boton('Rechazar').classList).toContain('btn--secundario');
    boton('Rechazar').click();
    f.detectChanges();
    expect(boton('Confirmar rechazo').classList).toContain('btn--peligro');
  });

  it('rechazar propone el motivo con los criterios sin marcar y envía el checklist', async () => {
    const { f, el, store, boton, marcar } = montar();
    marcar(3);
    boton('Rechazar').click();
    f.detectChanges();
    // ngModel escribe el valor en el textarea en una microtarea
    await f.whenStable();
    f.detectChanges();
    const area = el.querySelector<HTMLTextAreaElement>('#motivo-rechazo')!;
    expect(area.value).toBe('No cumple: monto coincide; fecha coincide.');
    boton('Confirmar rechazo').click();
    expect(store.rechazar).toHaveBeenCalledWith(9, 'No cumple: monto coincide; fecha coincide.', {
      ...TODO, monto_coincide: false, fecha_coincide: false,
    });
  });

  it('rechazar exige un motivo (con los 5 criterios marcados no se propone ninguno) y lo envía sin espacios sobrantes', () => {
    const { f, el, store, boton, marcar } = montar();
    marcar(5);
    boton('Rechazar').click();
    f.detectChanges();
    boton('Confirmar rechazo').click();
    f.detectChanges();
    expect(el.querySelector('#motivo-error')?.textContent?.trim()).toBe('Debes indicar un motivo para rechazar el comprobante.');
    expect(store.rechazar).not.toHaveBeenCalled();
    const area = el.querySelector<HTMLTextAreaElement>('#motivo-rechazo')!;
    area.value = '  Comprobante ilegible  ';
    area.dispatchEvent(new Event('input'));
    boton('Confirmar rechazo').click();
    expect(store.rechazar).toHaveBeenCalledWith(9, 'Comprobante ilegible', TODO);
    expect(TestBed.inject(ToastService).toasts().at(-1)?.title).toBe('Comprobante rechazado. El gasto volvió a estado "Registrado".');
    expect(TestBed.inject(ToastService).toasts().at(-1)?.type).toBe('info');
  });

  it('si el comprobante ya fue revisado (409), lo avisa y recarga', () => {
    const { store, boton, marcar } = montar(() => throwError(() => new HttpErrorResponse({ status: 409 })));
    marcar(5);
    boton('Aprobar').click();
    expect(TestBed.inject(ToastService).toasts().at(-1)?.title).toBe('Este comprobante ya fue revisado. Actualizamos la vista.');
    expect(store.cargar).toHaveBeenCalled();
  });

  it('si el servidor rechaza el checklist (422), lo explica', () => {
    const { boton, marcar } = montar(() => throwError(() => new HttpErrorResponse({ status: 422 })));
    marcar(5);
    boton('Aprobar').click();
    expect(TestBed.inject(ToastService).toasts().at(-1)?.title).toBe('Para aprobar, el comprobante debe cumplir los 5 criterios del checklist.');
  });

  it('el checklist vuelve a vacío solo si cambia el comprobante', () => {
    const { f, el, marcar } = montar();
    marcar(2);
    f.componentRef.setInput('comprobante', { ...comprobante });
    f.detectChanges();
    expect(el.querySelector('.checklist__conteo')?.textContent?.trim()).toBe('2 de 5 criterios');
    f.componentRef.setInput('comprobante', { ...comprobante, id: 10 });
    f.detectChanges();
    expect(el.querySelector('.checklist__conteo')?.textContent?.trim()).toBe('0 de 5 criterios');
  });
});
```

- [ ] **Step 2: Ejecutar y verificar que fallan**

Run: `cd frontend && npx ng test --watch=false --include src/app/features/gastos/revision-comprobante/revision-comprobante.component.spec.ts --include src/app/core/state/gastos.store.spec.ts`
Expected: FAIL. `NG0303: Can't set value of the 'gasto' input`, no existe `.checklist`, `aprobar` se llama con un solo argumento y el store envía `{}`.

- [ ] **Step 3: Actualizar el store**

En `frontend/src/app/core/state/gastos.store.ts`, agrega a los imports:

```ts
import { Checklist } from '../models/checklist';
```

y reemplaza los métodos `aprobar` y `rechazar` por:

```ts
  aprobar(comprobanteId: number, checklist: Checklist): Observable<unknown> {
    return this.api
      .post(`/comprobantes/${comprobanteId}/aprobar`, { checklist })
      .pipe(timeout(HTTP_TIMEOUT_MS), tap(() => this.cargar()));
  }

  rechazar(comprobanteId: number, motivo: string, checklist: Checklist): Observable<unknown> {
    return this.api
      .post(`/comprobantes/${comprobanteId}/rechazar`, { motivo_rechazo: motivo, checklist })
      .pipe(timeout(HTTP_TIMEOUT_MS), tap(() => this.cargar()));
  }
```

- [ ] **Step 4: Implementar el componente**

Reemplaza `frontend/src/app/features/gastos/revision-comprobante/revision-comprobante.component.ts`:

```ts
import { HttpErrorResponse } from '@angular/common/http';
import {
  ChangeDetectionStrategy, Component, Injector, afterNextRender, computed, inject, input, linkedSignal, signal,
} from '@angular/core';
import { FormsModule } from '@angular/forms';
import {
  CRITERIOS_CHECKLIST, MSG_CHECKLIST_INCOMPLETO, MSG_COMPROBANTE_YA_REVISADO, TOTAL_CRITERIOS,
  checklistCompleto, checklistVacio, cumplidos, motivoDesdeChecklist,
} from '../../../core/models/checklist';
import { Comprobante, CriterioChecklist, Gasto } from '../../../core/models/gasto.model';
import { GastosApiService } from '../../../core/services/gastos-api.service';
import { ToastService } from '../../../core/services/toast.service';
import { GastosStore } from '../../../core/state/gastos.store';
import { mensajeError } from '../../../core/utils/error-messages';
import { formatFechaHora } from '../../../core/utils/format';
import { IconoComponent } from '../../../shared/ui/icono/icono.component';

/**
 * Revisión del comprobante pendiente: el checklist documental y, con él, aprobar o rechazar
 * con motivo, en línea y sin ventana modal.
 */
@Component({
  selector: 'app-revision-comprobante',
  imports: [FormsModule, IconoComponent],
  templateUrl: './revision-comprobante.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class RevisionComprobanteComponent {
  readonly comprobante = input.required<Comprobante>();
  /** El gasto del comprobante: su monto y su fecha aparecen en los criterios para comparar */
  readonly gasto = input.required<Gasto>();

  private readonly store = inject(GastosStore);
  private readonly toast = inject(ToastService);
  private readonly gastosApi = inject(GastosApiService);
  private readonly injector = inject(Injector);

  protected readonly criterios = CRITERIOS_CHECKLIST;
  protected readonly total = TOTAL_CRITERIOS;
  /** Checklist en curso; vuelve a vacío solo cuando llega otro comprobante (no con una recarga del mismo) */
  protected readonly checklist = linkedSignal({ source: () => this.comprobante().id, computation: () => checklistVacio() });
  protected readonly marcados = computed(() => cumplidos(this.checklist()));
  protected readonly completo = computed(() => checklistCompleto(this.checklist()));

  protected readonly rechazando = signal(false);
  protected readonly enviando = signal(false);
  protected readonly motivoError = signal('');
  protected motivo = '';
  protected readonly url = computed(() => this.gastosApi.fileUrl(this.comprobante().archivo_url));
  protected readonly esPdf = computed(() => this.comprobante().tipo_archivo === 'pdf');
  protected readonly formatFechaHora = formatFechaHora;

  protected marcar(clave: CriterioChecklist, cumple: boolean): void {
    this.checklist.update(c => ({ ...c, [clave]: cumple }));
  }

  protected aprobar(): void {
    if (!this.completo()) return;
    this.enviando.set(true);
    this.store.aprobar(this.comprobante().id, this.checklist()).subscribe({
      next: () => {
        this.enviando.set(false);
        this.toast.show('Comprobante aprobado. El gasto quedó como "Sustentado".', '', 'ok');
      },
      error: err => {
        this.enviando.set(false);
        this.avisarError(err);
      },
    });
  }

  protected abrirRechazo(): void {
    this.rechazando.set(true);
    // Lo que quedó sin marcar es el motivo más probable; se puede editar
    this.motivo = motivoDesdeChecklist(this.checklist());
    this.motivoError.set('');
    afterNextRender(() => document.getElementById('motivo-rechazo')?.focus(), { injector: this.injector });
  }

  protected cancelarRechazo(): void {
    this.rechazando.set(false);
    this.motivo = '';
    this.motivoError.set('');
  }

  protected confirmarRechazo(): void {
    const motivo = this.motivo.trim();
    if (!motivo) {
      this.motivoError.set('Debes indicar un motivo para rechazar el comprobante.');
      return;
    }
    this.enviando.set(true);
    this.store.rechazar(this.comprobante().id, motivo, this.checklist()).subscribe({
      next: () => {
        this.enviando.set(false);
        this.toast.show('Comprobante rechazado. El gasto volvió a estado "Registrado".', '', 'info');
        this.cancelarRechazo();
      },
      error: err => {
        this.enviando.set(false);
        this.avisarError(err);
      },
    });
  }

  /** 409 y 422 tienen su propio mensaje; el resto usa el copy exacto de error-messages.ts */
  private avisarError(err: unknown): void {
    if (err instanceof HttpErrorResponse && err.status === 409) {
      this.toast.show(MSG_COMPROBANTE_YA_REVISADO, '', 'warn');
      this.store.cargar();
      return;
    }
    if (err instanceof HttpErrorResponse && err.status === 422) {
      this.toast.show(MSG_CHECKLIST_INCOMPLETO, '', 'err');
      return;
    }
    this.toast.show(mensajeError(err), '', 'err');
  }
}
```

Reemplaza `frontend/src/app/features/gastos/revision-comprobante/revision-comprobante.component.html`:

```html
<div class="revision">
  <figure class="revision__vista">
    @if (esPdf()) {
      <div class="revision__pdf">
        <app-icono nombre="file-type-pdf" [tamano]="40" />
        <span>Comprobante en PDF</span>
      </div>
    } @else {
      <img class="revision__imagen" [src]="url()" alt="Comprobante de pago adjuntado" loading="lazy" />
    }
    <figcaption class="revision__pie">
      <span>Adjuntado el {{ formatFechaHora(comprobante().creado_en) }}</span>
      <a class="enlace" [href]="url()" target="_blank" rel="noopener">{{ esPdf() ? 'Abrir PDF' : 'Ver en tamaño completo' }}</a>
    </figcaption>
  </figure>

  <fieldset class="checklist" [disabled]="enviando()">
    <legend class="checklist__titulo">Checklist documental</legend>
    <ul class="checklist__lista">
      @for (criterio of criterios; track criterio.clave) {
        <li>
          <label class="casilla checklist__criterio">
            <input
              type="checkbox"
              [checked]="checklist()[criterio.clave]"
              (change)="marcar(criterio.clave, $any($event.target).checked)"
            />
            <span>{{ criterio.texto(gasto()) }}</span>
          </label>
        </li>
      }
    </ul>
    <p class="checklist__conteo" aria-live="polite">{{ marcados() }} de {{ total }} criterios</p>
  </fieldset>

  @if (!rechazando()) {
    <div class="formulario__acciones">
      <button
        type="button"
        class="btn btn--guia"
        [disabled]="enviando() || !completo()"
        [attr.aria-describedby]="completo() ? null : 'checklist-aviso'"
        (click)="aprobar()"
      >
        <app-icono nombre="check" [tamano]="18" />Aprobar
      </button>
      <button type="button" class="btn btn--secundario" [disabled]="enviando()" (click)="abrirRechazo()">
        <app-icono nombre="x" [tamano]="18" />Rechazar
      </button>
    </div>
    @if (!completo()) {
      <p class="campo__ayuda" id="checklist-aviso">Marca los 5 criterios para aprobar.</p>
    }
  } @else {
    <div class="revision__rechazo">
      <div class="campo" [class.campo--con-error]="motivoError()">
        <label class="campo__etiqueta" for="motivo-rechazo">Motivo del rechazo <span class="req" aria-hidden="true">*</span></label>
        <textarea
          class="control"
          id="motivo-rechazo"
          name="motivo"
          rows="3"
          [(ngModel)]="motivo"
          placeholder="Explica por qué se rechaza este comprobante (ej. monto no coincide, comprobante ilegible)"
          [attr.aria-invalid]="!!motivoError() || null"
          [attr.aria-describedby]="motivoError() ? 'motivo-error' : null"
        ></textarea>
        @if (motivoError()) {
          <p class="campo__error" id="motivo-error">{{ motivoError() }}</p>
        }
      </div>
      <div class="formulario__acciones">
        <button type="button" class="btn btn--peligro" [disabled]="enviando()" (click)="confirmarRechazo()">Confirmar rechazo</button>
        <button type="button" class="btn btn--fantasma" (click)="cancelarRechazo()">Cancelar</button>
      </div>
    </div>
  }
</div>
```

En `frontend/src/app/features/gastos/gasto-detalle/gasto-detalle.component.html`, reemplaza:

```html
            <app-revision-comprobante [comprobante]="c" />
```
por:
```html
            <app-revision-comprobante [comprobante]="c" [gasto]="g" />
```

- [ ] **Step 5: Estilos del checklist**

Al final de `frontend/src/styles/gastos.css`, agrega:

```css
/* ══ Gastos: checklist documental de la revisión ══ */
.checklist {
  min-width: 0;
  margin: 0;
  padding: var(--e-4) var(--e-5);
  border: 1px solid var(--linea);
  border-radius: var(--r-panel);
  background: var(--panel);
}

.checklist__titulo {
  padding: 0;
  font-size: var(--t-16);
  font-weight: 700;
}

.checklist__lista {
  display: grid;
  gap: 2px;
  margin: var(--e-2) 0 0;
  padding: 0;
  list-style: none;
}

.checklist__criterio {
  min-height: 40px;
  color: var(--tinta);
  font-size: var(--t-15);
}

.checklist__conteo {
  margin-top: var(--e-2);
  color: var(--tinta-2);
  font-size: var(--t-13);
  font-variant-numeric: tabular-nums;
}

@media (max-width: 767px) {
  /* Objetivos táctiles de 44 px */
  .checklist__criterio {
    min-height: 44px;
  }
}
```

- [ ] **Step 6: Ejecutar y verificar que pasan**

Run: `cd frontend && npx ng test --watch=false --include src/app/features/gastos/revision-comprobante/revision-comprobante.component.spec.ts --include src/app/core/state/gastos.store.spec.ts`
Expected: PASS (10 pruebas de la revisión y todas las del store).

- [ ] **Step 7: Checkpoint (sin commit)**

Run: `cd frontend && npx ng test --watch=false && npx ng build`
Expected: PASS en todas las pruebas y build de producción sin errores ni avisos de presupuesto.

---

### Task 5: El checklist en el historial del detalle

**Files:**
- Modify: `frontend/src/app/features/gastos/gasto-detalle/gasto-detalle.component.ts` (exponer `cortoDe`)
- Modify: `frontend/src/app/features/gastos/gasto-detalle/gasto-detalle.component.html` (historial)
- Modify: `frontend/src/app/features/gastos/gasto-detalle/gasto-detalle.component.spec.ts`
- Modify: `frontend/src/styles/gastos.css`

**Interfaces:**
- Consumes: `cortoDe` (Tarea 3); `Comprobante.checklist`, `aprobado_por`, `aprobado_en`.
- Produces: en el historial, `ul.historial__checklist` con 5 `li` (clase `historial__criterio--no` si no cumple) y `span.historial__revisor` «Revisado por … el …», solo para comprobantes con checklist.

- [ ] **Step 1: Escribir las pruebas que fallan**

En `frontend/src/app/features/gastos/gasto-detalle/gasto-detalle.component.spec.ts`, antes de la prueba `'«Gastos» vuelve a la pestaña de origen'`, agrega:

```ts
  it('el historial muestra el checklist guardado y quién lo revisó', () => {
    const rechazado: Comprobante = {
      id: 7, gasto_id: 31, archivo_url: '/u/7.jpg', tipo_archivo: 'imagen', estado: 'Rechazado', creado_en: '2026-10-01T17:00:00Z',
      motivo_rechazo: 'No cumple: monto coincide.', aprobado_por: 'Ana Quispe', aprobado_en: '2026-09-30T15:00:00Z',
      checklist: [
        { criterio: 'legible', cumple: true }, { criterio: 'tipo_valido', cumple: true }, { criterio: 'a_nombre_empresa', cumple: true },
        { criterio: 'monto_coincide', cumple: false }, { criterio: 'fecha_coincide', cumple: true },
      ],
    };
    const f = montar(storeFalso([gasto(31, 'Registrado')], [rechazado]), '31');
    const el: HTMLElement = f.nativeElement;
    const items = [...el.querySelectorAll('.historial__checklist li')];
    expect(items).toHaveLength(5);
    expect(items[3].textContent).toContain('Monto coincide');
    expect(items[3].classList).toContain('historial__criterio--no');
    expect(items[0].classList).not.toContain('historial__criterio--no');
    expect(el.querySelector('.historial__revisor')?.textContent?.trim()).toBe('Revisado por Ana Quispe el 30/09/2026 10:00');
  });

  it('un comprobante revisado antes del checklist no muestra lista ni revisor', () => {
    const antiguo: Comprobante = {
      id: 6, gasto_id: 29, archivo_url: '/u/6.jpg', tipo_archivo: 'imagen', estado: 'Aprobado', creado_en: '2026-09-29T17:00:00Z',
      aprobado_por: 'Logística', aprobado_en: '2026-09-30T15:00:00Z', checklist: [],
    };
    const f = montar(storeFalso([gasto(29, 'Sustentado')], [antiguo]), '29');
    expect(f.nativeElement.querySelector('.historial__checklist')).toBeNull();
    expect(f.nativeElement.querySelector('.historial__revisor')).toBeNull();
  });
```

- [ ] **Step 2: Ejecutar y verificar que fallan**

Run: `cd frontend && npx ng test --watch=false --include src/app/features/gastos/gasto-detalle/gasto-detalle.component.spec.ts`
Expected: FAIL en la primera prueba nueva (no hay `.historial__checklist`); la segunda pasa desde ya y queda como guardia.

- [ ] **Step 3: Implementar**

En `frontend/src/app/features/gastos/gasto-detalle/gasto-detalle.component.ts`, agrega a los imports:

```ts
import { cortoDe } from '../../../core/models/checklist';
```

y, junto a `protected readonly formatFechaHora = formatFechaHora;`, agrega:

```ts
  protected readonly cortoDe = cortoDe;
```

En `gasto-detalle.component.html`, dentro de `<div class="historial__texto">`, reemplaza:

```html
                @if (c.estado === 'Rechazado' && c.motivo_rechazo) {
                  <span class="historial__motivo">Motivo: {{ c.motivo_rechazo }}</span>
                }
```
por:
```html
                @if (c.estado === 'Rechazado' && c.motivo_rechazo) {
                  <span class="historial__motivo">Motivo: {{ c.motivo_rechazo }}</span>
                }
                @if (c.checklist?.length) {
                  <ul class="historial__checklist" aria-label="Checklist documental">
                    @for (item of c.checklist; track item.criterio) {
                      <li [class.historial__criterio--no]="!item.cumple">
                        <app-icono [nombre]="item.cumple ? 'check' : 'x'" [tamano]="14" />{{ cortoDe(item.criterio) }}<span class="oculto-visual">{{ item.cumple ? ': cumple' : ': no cumple' }}</span>
                      </li>
                    }
                  </ul>
                  <span class="historial__revisor">Revisado por {{ c.aprobado_por }} el {{ formatFechaHora(c.aprobado_en) }}</span>
                }
```

Al final de `frontend/src/styles/gastos.css`, agrega:

```css
/* El checklist guardado de cada comprobante revisado */
.historial__checklist {
  display: flex;
  flex-wrap: wrap;
  gap: 2px var(--e-3);
  margin: var(--e-1) 0 0;
  padding: 0;
  list-style: none;
  color: var(--sustentado-tinta);
  font-size: var(--t-13);
}

.historial__checklist li {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.historial__criterio--no {
  color: var(--rojo-tinta);
}

.historial__revisor {
  color: var(--tinta-2);
  font-size: var(--t-13);
}
```

- [ ] **Step 4: Ejecutar y verificar que pasan**

Run: `cd frontend && npx ng test --watch=false --include src/app/features/gastos/gasto-detalle/gasto-detalle.component.spec.ts`
Expected: PASS (todas las del detalle, incluidas las 2 nuevas).

- [ ] **Step 5: Checkpoint (sin commit)**

Run: `cd frontend && npx ng test --watch=false && npx ng build`
Expected: PASS en todas las pruebas y build de producción sin errores.

---

### Task 6: Verificación en navegador, detector y cierre

**Files (fuera del repositorio, en el scratchpad):**
- Create: `$SCRATCH/visual/checklist-navegador.mjs` (usa el `playwright-core` ya instalado en `$SCRATCH/visual/node_modules`)
- Capturas: `$SCRATCH/visual/capturas/checklist-*.png`

`SCRATCH=/private/tmp/claude-501/-Users-helcorp-Downloads-Proyectos-Universidad-Proyecto-Campstone/00d73130-eac8-4a38-b8ba-011054089b50/scratchpad`

**Interfaces:**
- Consumes: la app corriendo con `ng serve --port 4300`; todo lo anterior.
- Produces: evidencia de que el flujo funciona en Chrome de escritorio y celular.

- [ ] **Step 1: Escribir el script de verificación**

`$SCRATCH/visual/checklist-navegador.mjs`:

```js
// Verificación del checklist documental en navegador, con las APIs simuladas (datos sintéticos).
import { chromium } from 'playwright-core';
import { mkdirSync } from 'node:fs';

const BASE = 'http://localhost:4300';
const CHROME = '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
const SALIDA = new URL('./capturas/', import.meta.url).pathname;
mkdirSync(SALIDA, { recursive: true });
const USUARIO = { id_usuario: 1, username: 'aquispe', email: 'ana@lizar.pe', nombre: 'Ana Quispe', rol: 'admin', estado: true };
const PNG = Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII=', 'base64');
const CRITERIOS = ['legible', 'tipo_valido', 'a_nombre_empresa', 'monto_coincide', 'fecha_coincide'];
const CORS = {
  'access-control-allow-origin': BASE,
  'access-control-allow-headers': 'Content-Type, X-User-Role, X-User-Name',
  'access-control-allow-methods': 'GET, POST, OPTIONS',
};

const verificar = (condicion, mensaje) => {
  if (!condicion) throw new Error(`Falló: ${mensaje}`);
  console.log('OK', mensaje);
};

async function esperarServidor() {
  for (let i = 0; i < 90; i++) {
    try {
      if ((await fetch(BASE)).ok) return;
    } catch {
      /* todavía compilando */
    }
    await new Promise(r => setTimeout(r, 2000));
  }
  throw new Error(`ng serve no respondió en ${BASE}`);
}

async function contexto(navegador, opciones) {
  const ctx = await navegador.newContext(opciones);
  const datos = {
    gastos: [{ id: 30, viaje_id: 14, cliente_id: 1, monto: 85.5, fecha: '2026-09-30T00:00:00Z', estado: 'Pagado', creado_por: 'Ana Quispe', creado_en: '2026-10-01T13:00:00Z' }],
    comprobantes: [{ id: 9, gasto_id: 30, archivo_url: '/uploads/9.png', tipo_archivo: 'imagen', estado: 'Pendiente', creado_en: '2026-10-01T15:00:00Z', checklist: [] }],
  };
  const json = (ruta, cuerpo, status = 200) =>
    ruta.fulfill({ status, contentType: 'application/json', headers: CORS, body: JSON.stringify(cuerpo) });
  await ctx.addInitScript(u => {
    localStorage.setItem('lizar_token', 'demo');
    localStorage.setItem('lizar_user', JSON.stringify(u));
  }, USUARIO);
  await ctx.route('http://localhost:8000/**', ruta => json(ruta, []));
  await ctx.route('http://localhost:8081/**', ruta => {
    const req = ruta.request();
    const url = new URL(req.url());
    if (req.method() === 'OPTIONS') return ruta.fulfill({ status: 204, headers: CORS });
    if (url.pathname.startsWith('/uploads/')) return ruta.fulfill({ status: 200, contentType: 'image/png', headers: CORS, body: PNG });
    if (url.pathname === '/gastos') return json(ruta, datos.gastos);
    if (url.pathname === '/comprobantes') {
      const estado = url.searchParams.get('estado');
      return json(ruta, datos.comprobantes.filter(c => !estado || c.estado === estado));
    }
    const m = url.pathname.match(/^\/comprobantes\/(\d+)\/(aprobar|rechazar)$/);
    if (m && req.method() === 'POST') {
      const c = datos.comprobantes.find(x => x.id === Number(m[1]));
      if (!c) return json(ruta, { error: 'comprobante no encontrado' }, 404);
      if (c.estado !== 'Pendiente') return json(ruta, { error: 'el comprobante ya fue revisado' }, 409);
      const cuerpo = req.postDataJSON() ?? {};
      const lista = cuerpo.checklist ?? {};
      if (m[2] === 'aprobar' && !CRITERIOS.every(k => lista[k] === true)) {
        return json(ruta, { error: 'para aprobar, el comprobante debe cumplir los 5 criterios del checklist' }, 422);
      }
      const g = datos.gastos.find(x => x.id === c.gasto_id);
      c.checklist = CRITERIOS.map(k => ({ criterio: k, cumple: !!lista[k] }));
      c.aprobado_por = 'Ana Quispe';
      c.aprobado_en = '2026-10-02T15:00:00Z';
      if (m[2] === 'aprobar') {
        c.estado = 'Aprobado';
        g.estado = 'Sustentado';
      } else {
        c.estado = 'Rechazado';
        c.motivo_rechazo = cuerpo.motivo_rechazo;
        g.estado = 'Registrado';
      }
      return json(ruta, { status: m[2] === 'aprobar' ? 'aprobado' : 'rechazado' });
    }
    return json(ruta, { error: 'no simulado' }, 404);
  });
  return ctx;
}

await esperarServidor();
const navegador = await chromium.launch({ executablePath: CHROME, headless: true });

// 1. Escritorio: aprobar exige los 5 criterios y el checklist queda en el historial
{
  const ctx = await contexto(navegador, { viewport: { width: 1440, height: 900 } });
  const p = await ctx.newPage();
  await p.goto(`${BASE}/app/gastos/30`, { waitUntil: 'networkidle' });
  await p.getByText('Checklist documental').waitFor();
  const aprobar = p.getByRole('button', { name: 'Aprobar' });
  verificar(await aprobar.isDisabled(), 'Aprobar empieza deshabilitado');
  verificar(await p.getByText('La fecha corresponde al gasto (30/09/2026).').isVisible(), 'los criterios muestran la fecha del gasto');
  await p.screenshot({ path: `${SALIDA}checklist-pendiente-escritorio.png`, fullPage: true });
  const casillas = p.locator('.checklist input[type="checkbox"]');
  for (let i = 0; i < 4; i++) await casillas.nth(i).check();
  await p.getByText('4 de 5 criterios').waitFor();
  verificar(await aprobar.isDisabled(), 'con 4 de 5 criterios sigue deshabilitado');
  await casillas.nth(4).check();
  await p.getByText('5 de 5 criterios').waitFor();
  verificar(await aprobar.isEnabled(), 'con los 5 criterios se habilita');
  await aprobar.click();
  await p.getByText('Comprobante aprobado. El gasto quedó como "Sustentado".').waitFor();
  await p.getByRole('heading', { name: 'Gasto sustentado' }).waitFor();
  await p.locator('.historial__checklist').waitFor();
  verificar((await p.locator('.historial__checklist li').count()) === 5, 'el historial muestra los 5 criterios');
  verificar(await p.getByText(/^Revisado por Ana Quispe el /).isVisible(), 'el historial dice quién revisó');
  await p.screenshot({ path: `${SALIDA}checklist-aprobado-escritorio.png`, fullPage: true });
  await ctx.close();
}

// 2. Celular: rechazar propone el motivo con lo que no cumple
{
  const ctx = await contexto(navegador, { viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true, deviceScaleFactor: 2 });
  const p = await ctx.newPage();
  await p.goto(`${BASE}/app/gastos/30`, { waitUntil: 'networkidle' });
  const casillas = p.locator('.checklist input[type="checkbox"]');
  await casillas.first().waitFor();
  for (let i = 0; i < 3; i++) await casillas.nth(i).check();
  const alto = await p.locator('.checklist__criterio').first().evaluate(e => e.getBoundingClientRect().height);
  verificar(alto >= 44, `cada criterio mide al menos 44 px en el celular (mide ${alto})`);
  await p.getByRole('button', { name: 'Rechazar' }).click();
  verificar((await p.locator('#motivo-rechazo').inputValue()) === 'No cumple: monto coincide; fecha coincide.', 'el motivo se propone con lo que no cumple');
  await p.screenshot({ path: `${SALIDA}checklist-rechazo-celular.png`, fullPage: true });
  await p.getByRole('button', { name: 'Confirmar rechazo' }).click();
  await p.getByText('Comprobante rechazado. El gasto volvió a estado "Registrado".').waitFor();
  await p.locator('.historial__criterio--no').first().waitFor();
  verificar((await p.locator('.historial__criterio--no').count()) === 2, 'el historial marca los 2 criterios que no cumplen');
  const desborde = await p.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
  verificar(desborde === 0, 'sin desborde horizontal en el celular');
  await p.screenshot({ path: `${SALIDA}checklist-rechazado-celular.png`, fullPage: true });
  await ctx.close();
}

await navegador.close();
console.log('Checklist verificado en navegador.');
```

- [ ] **Step 2: Levantar la app y ejecutar el script**

En segundo plano, desde `frontend/`: `npx ng serve --port 4300` (no tocar el servidor del puerto 4200).

Run: `node "$SCRATCH/visual/checklist-navegador.mjs"`
Expected: 11 líneas `OK …` y `Checklist verificado en navegador.`

- [ ] **Step 3: Revisar las capturas**

Abre `checklist-pendiente-escritorio.png`, `checklist-aprobado-escritorio.png`, `checklist-rechazo-celular.png` y `checklist-rechazado-celular.png`. Comprueba:
- el checklist queda entre el comprobante y los botones, con la leyenda «Checklist documental», las 5 casillas alineadas y el conteo;
- el aviso «Marca los 5 criterios para aprobar.» aparece bajo los botones mientras falten criterios;
- en el historial, los ✓ en verde de guía, los ✗ en rojo tinta y la línea «Revisado por …»;
- en el celular, nada se desborda y cada criterio es cómodo de tocar.

Corrige lo que encuentres en un solo lote y vuelve a ejecutar el script una vez. Luego detén tu `ng serve` del puerto 4300.

- [ ] **Step 4: Detector de impeccable**

Run: `cd "/Users/helcorp/Downloads/Proyectos/Universidad/Proyecto Campstone" && /Users/helcorp/.claude/skills/impeccable/scripts/impeccable detect --json frontend/src/app/features/gastos frontend/src/styles`
Expected: solo los 2 falsos positivos conocidos (`broken-image` de `<img [src]>` en adjuntar y revisión). Cualquier otro hallazgo se corrige.

- [ ] **Step 5: Verificación final (sin commit)**

Run:
```bash
cd "/Users/helcorp/Downloads/Proyectos/Universidad/Proyecto Campstone/gastos-service" && gofmt -l . && go vet ./... && TEST_DATABASE_URL="$(sed -n 's/^DATABASE_URL=//p' .env | tr -d "\"'")" go test ./... -count=1
cd ../frontend && npx ng test --watch=false && npx ng build
cd .. && git status --short | head -40 && git log --oneline | head -1
```
Expected: Go sin salida de gofmt, vet limpio y tests PASS; frontend con todas las pruebas en verde y build de producción OK; `git log` sigue en `1953886 first commit` (sin commits).

Informa al usuario: qué se agregó, cómo probarlo en la app (gasto en estado Pagado → «Revisar comprobante»), que debe **reiniciar gastos-service** para que tome la migración 0002 y las nuevas reglas, y el comando para correr las pruebas de Go.
