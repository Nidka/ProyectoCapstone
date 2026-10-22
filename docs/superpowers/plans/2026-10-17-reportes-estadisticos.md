# Reportes estadísticos de gastos — plan de implementación

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Gerencia ve en «Reportes» las estadísticas de gastos de un rango de fechas con gráficos interactivos y descarga dos PDF generados por gastos-service con las mismas cifras.

**Architecture:** gastos-service (Go) gana el paquete `internal/reportes`: filtro de fechas, consultas de agregación en una sola transacción `REPEATABLE READ, READ ONLY` y dos PDF dibujados con `signintech/gopdf` y la fuente Overpass embebida; tres endpoints los exponen. El frontend agrega la ruta diferida `/app/reportes`: el rango vive en la URL, `rxResource` consulta el resumen, Chart.js dibuja cinco gráficos de barras detrás de una fábrica inyectable y los botones descargan los PDF como blob.

**Tech Stack:** Go 1.25 + pgx v5 + PostgreSQL 17 + `github.com/signintech/gopdf` v0.38.1; Angular 21.2 (zoneless, signals, `rxResource`, `afterRenderEffect`) + Vitest; `chart.js` 4.5.1; Playwright (`playwright-core` con el Chrome del sistema) para la verificación en navegador.

**Spec:** `docs/superpowers/specs/2026-10-17-reportes-estadisticos-design.md`

**Nota:** todo el código de este plan se probó antes en una copia del proyecto en el scratchpad: Go 37 pruebas, Vitest 181 pruebas, build de producción y recorrido en Chrome contra el servicio real con datos DEMO.

## Global Constraints

- Se trabaja en `main` sin ramas ni commits (decisión del usuario). Los pasos «Checkpoint» no hacen commit.
- No se modifican `frontend/src/app/core/models/gasto-estados.ts` ni `frontend/src/app/core/utils/error-messages.ts` (RNF-1). Los errores de descarga usan `mensajeError(err)`.
- Dependencias nuevas, exactamente: Go `github.com/signintech/gopdf v0.38.1` (publicada el 12/09/2026); npm `chart.js@4.5.1` exacta (`--save-exact`). Ninguna otra.
- Fuente Overpass Regular y Bold (TTF del subconjunto latino de Fontsource) con su `OFL.txt`, embebidas con `go:embed`.
- Mensajes de filtro del servidor, idénticos: `fecha inválida: use AAAA-MM-DD`, `la fecha inicial no puede ser posterior a la final`, `el rango no puede superar 24 meses`; cualquier otro error: `no se pudo generar el reporte`. Rango máximo: 731 días contando ambos extremos.
- Nombres de archivo: `reporte-estadistico-AAAA-MM-DD_AAAA-MM-DD.pdf` y `detalle-gastos-AAAA-MM-DD_AAAA-MM-DD.pdf`.
- Colores de estado (app y PDF): Registrado `#8a9096`, Pagado `#2563eb`, Sustentado `#15803d`. El validador de dataviz marca el gris por poco croma: se mantiene por coherencia con la app y se compensa con leyenda siempre visible, huecos de 2 px entre segmentos, tooltips con el nombre del estado y la tabla «Ver datos».
- Estilos solo en `frontend/src/styles/*.css`, con tokens existentes; la hoja nueva va al final de `styles.css`.
- Datos DEMO: RUC `990000000xx`, placas `DEM-`, DNI `990000xx`, direcciones con referencia `DEMO`, gastos con `creado_por = 'DEMO'`. Los scripts no van en `migrations/`.
- Las pruebas de integración de Go usan `TEST_DATABASE_URL` y un esquema temporal `prueba_<n>` que se borra al terminar. La verificación en navegador usa una base temporal `lizar_reportes_prueba` que se borra al final. Nunca se escriben datos en la base real.
- El `ng serve` del usuario en el puerto 4200 y su gastos-service en 8081 no se tocan; la verificación usa 4300 y 8091.
- Comandos: Go `cd gastos-service && TEST_DATABASE_URL="$(sed -n 's/^DATABASE_URL=//p' .env | tr -d "\"'")" go test ./... -count=1`; frontend `cd frontend && npx ng test --watch=false` y `npx ng build`. Si el chequeo de seguridad rechaza el comando Go en línea, se usa el script `scratchpad/go-tests.sh` de la sesión anterior (mismo comando).

## Review Focus

1. **Doble clic en «Descargar»:** una sola petición; el botón dice «Generando…» y ambos botones quedan deshabilitados hasta que termina. Lo fija la prueba «descargar muestra «Generando…», evita el doble clic y guarda el archivo» (Tarea 8).
2. **Cambiar «Desde» y luego «Hasta» rápido** (la URL todavía no tiene el primer cambio): se navega con las dos fechas visibles, sin perder la primera. Lo fija «cambiar las dos fechas seguidas usa ambas…» (Tarea 8). Este caso apareció en la prueba en navegador del prototipo.
3. **Cambiar una fecha a mano después de elegir un mes en el gráfico:** el aviso «Viendo…» desaparece y «Quitar» no vuelve a un rango viejo. Lo fija «cambiar una fecha a mano después de elegir un mes quita el aviso» (Tarea 8).
4. **El límite de 731 días coincide en ambos lados:** 2024-01-01..2025-12-31 pasa y 2024-01-01..2026-01-01 da el mismo mensaje en el frontend y en el servidor. Lo fijan `TestFiltroAceptaExactamente731Dias`/`TestFiltroInvalido` (Tarea 1) y «los errores coinciden con gastos-service…» (Tarea 6).
5. **Nombres largos y 24 meses:** los nombres se recortan con «…» en el eje y en el PDF y salen completos en el tooltip y en la tabla; un rango de 24 meses mantiene una etiqueta por mes en el gráfico y no rompe el PDF. Lo fijan «ranking: recorta nombres largos…», «24 meses siguen teniendo una etiqueta por mes» (Tarea 7) y `TestPDFEstadisticoCon24Meses` (Tarea 3).

## Mapa de archivos

| Archivo | Responsabilidad |
|---|---|
| `gastos-service/internal/reportes/filtro.go` | Rango de fechas: valores por defecto, validación, `ErrFiltro` |
| `gastos-service/internal/reportes/modelo.go` | Tipos del resumen (JSON) y del detalle |
| `gastos-service/internal/reportes/formato.go` | Montos, fechas y meses con el formato del frontend |
| `gastos-service/internal/reportes/consultas.go` | SQL de agregación en una transacción de solo lectura |
| `gastos-service/internal/reportes/pdf.go` | Documento base: fuentes, cabecera, pie «Página N de M», tablas |
| `gastos-service/internal/reportes/pdf_estadistico.go` | PDF estadístico (A4 vertical) |
| `gastos-service/internal/reportes/pdf_detalle.go` | PDF de detalle (A4 horizontal) |
| `gastos-service/internal/reportes/fuentes/` | Overpass 400/700 + OFL.txt |
| `gastos-service/internal/pruebas/esquema.go` | Esquema temporal para pruebas de integración (checklist y reportes) |
| `gastos-service/internal/handlers/reportes.go` | Tres endpoints de reportes |
| `gastos-service/demo/reportes_demo*.sql` | Datos de demostración y su limpieza |
| `frontend/src/app/core/models/reporte.model.ts` | Tipos de la respuesta y del filtro |
| `frontend/src/app/core/services/reportes.service.ts` | Resumen y descarga de PDF |
| `frontend/src/app/core/utils/descarga.ts` | Guardar un Blob como archivo |
| `frontend/src/app/features/reportes/filtro-reporte.ts` | Lógica pura del rango (atajos, mes, validación) |
| `frontend/src/app/features/reportes/graficos.ts` | Configuraciones de Chart.js y descripciones accesibles |
| `frontend/src/app/features/reportes/grafico/` | Envoltorio de Chart.js con fábrica inyectable |
| `frontend/src/app/features/reportes/reportes.component.*` | Página del dashboard |
| `frontend/src/styles/reportes.css` | Estilos del dashboard |

---

### Task 1: Filtro, modelo y formatos (Go)

**Files:**
- Create: `gastos-service/internal/reportes/filtro.go`, `modelo.go`, `formato.go`
- Test: `gastos-service/internal/reportes/filtro_test.go`, `formato_test.go`

**Interfaces:**
- Consumes: nada.
- Produces:
  - `reportes.Lima` (zona UTC−5), `reportes.Filtro{Desde, Hasta time.Time}` con `DesdeISO()`/`HastaISO()`, `reportes.ErrFiltro{Mensaje}`, `reportes.ParseFiltro(desde, hasta string, ahora time.Time) (Filtro, error)`; constante interna `formatoISO`.
  - Tipos `Resumen`, `Indicadores`, `Mes`, `PorEstado`, `FilaRanking`, `FilaDetalle` (JSON en snake_case; `Detalle` no viaja en el JSON).
  - `FormatoMonto(float64) string` («S/ 1,234.50»), `FormatoFecha(time.Time)`, `FormatoEmision(time.Time)` (hora de Lima), `MesCorto("2026-10") == "oct 26"`, `FormatoPorcentaje(42.26) == "42.3 %"`.

- [ ] **Step 1: Escribir las pruebas que fallan**

`gastos-service/internal/reportes/filtro_test.go`:

```go
package reportes

import (
	"errors"
	"testing"
	"time"
)

// 17/10/2026 02:00 en UTC todavía es 16/10/2026 en Lima.
var ahoraPrueba = time.Date(2026, 10, 17, 2, 0, 0, 0, time.UTC)

func TestFiltroPorDefectoSonLosUltimos12MesesHastaHoyEnLima(t *testing.T) {
	f, err := ParseFiltro("", "", ahoraPrueba)
	if err != nil {
		t.Fatal(err)
	}
	if f.DesdeISO() != "2025-11-01" || f.HastaISO() != "2026-10-16" {
		t.Fatalf("rango = %s .. %s", f.DesdeISO(), f.HastaISO())
	}
}

func TestFiltroSinDesdeCuentaDesdeElHastaElegido(t *testing.T) {
	f, err := ParseFiltro("", "2026-03-15", ahoraPrueba)
	if err != nil {
		t.Fatal(err)
	}
	if f.DesdeISO() != "2025-04-01" || f.HastaISO() != "2026-03-15" {
		t.Fatalf("rango = %s .. %s", f.DesdeISO(), f.HastaISO())
	}
}

func TestFiltroExplicito(t *testing.T) {
	f, err := ParseFiltro("2026-01-01", "2026-01-31", ahoraPrueba)
	if err != nil || f.DesdeISO() != "2026-01-01" || f.HastaISO() != "2026-01-31" {
		t.Fatalf("rango = %v, err = %v", f, err)
	}
}

func TestFiltroInvalido(t *testing.T) {
	casos := []struct{ desde, hasta, mensaje string }{
		{"2026-13-01", "", "fecha inválida: use AAAA-MM-DD"},
		{"", "17/10/2026", "fecha inválida: use AAAA-MM-DD"},
		{"2026-02-30", "", "fecha inválida: use AAAA-MM-DD"},
		{"2026-05-02", "2026-05-01", "la fecha inicial no puede ser posterior a la final"},
		{"2024-01-01", "2026-01-01", "el rango no puede superar 24 meses"},
	}
	for _, c := range casos {
		_, err := ParseFiltro(c.desde, c.hasta, ahoraPrueba)
		var ef ErrFiltro
		if !errors.As(err, &ef) || ef.Mensaje != c.mensaje {
			t.Errorf("ParseFiltro(%q, %q) = %v, se esperaba %q", c.desde, c.hasta, err, c.mensaje)
		}
	}
}

func TestFiltroAceptaExactamente731Dias(t *testing.T) {
	if _, err := ParseFiltro("2024-01-01", "2025-12-31", ahoraPrueba); err != nil {
		t.Fatalf("731 días debían aceptarse: %v", err)
	}
}
```

`gastos-service/internal/reportes/formato_test.go`:

```go
package reportes

import (
	"testing"
	"time"
)

func TestFormatoMontoComoElFrontend(t *testing.T) {
	casos := map[float64]string{
		0:          "S/ 0.00",
		85.5:       "S/ 85.50",
		1234.5:     "S/ 1,234.50",
		1234567.89: "S/ 1,234,567.89",
		-12.3:      "S/ -12.30",
	}
	for v, want := range casos {
		if got := FormatoMonto(v); got != want {
			t.Errorf("FormatoMonto(%v) = %q, se esperaba %q", v, got, want)
		}
	}
}

func TestFormatosDeFecha(t *testing.T) {
	if got := FormatoFecha(time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)); got != "30/09/2026" {
		t.Errorf("FormatoFecha = %q", got)
	}
	if got := FormatoEmision(time.Date(2026, 10, 17, 15, 32, 0, 0, time.UTC)); got != "17/10/2026 10:32" {
		t.Errorf("FormatoEmision = %q", got)
	}
	if got := MesCorto("2026-10"); got != "oct 26" {
		t.Errorf("MesCorto = %q", got)
	}
	if got := FormatoPorcentaje(42.26); got != "42.3 %" {
		t.Errorf("FormatoPorcentaje = %q", got)
	}
}
```

- [ ] **Step 2: Ejecutar y verificar que fallan**

Run: `cd gastos-service && go test ./internal/reportes/ -count=1`
Expected: FAIL de compilación: `undefined: ParseFiltro`, `undefined: FormatoMonto`…

- [ ] **Step 3: Implementar**

`gastos-service/internal/reportes/filtro.go`:

```go
// Package reportes calcula las estadísticas de gastos de un rango de fechas
// y genera los PDF que se descargan desde el dashboard gerencial.
package reportes

import "time"

// Lima es la zona horaria de LIZAR (Perú no tiene horario de verano).
var Lima = time.FixedZone("America/Lima", -5*60*60)

const formatoISO = "2006-01-02"

// rangoMaximoDias es el rango más largo que se acepta, contando ambos extremos.
const rangoMaximoDias = 731

// Filtro es el rango de fechas del gasto, con ambos extremos incluidos.
type Filtro struct {
	Desde time.Time
	Hasta time.Time
}

// ErrFiltro es un error del usuario al elegir el rango: se responde con 400.
type ErrFiltro struct{ Mensaje string }

func (e ErrFiltro) Error() string { return e.Mensaje }

// ParseFiltro resuelve desde y hasta (AAAA-MM-DD). Sin hasta, es hoy en Lima;
// sin desde, el primer día del mes 11 meses antes del mes de hasta.
func ParseFiltro(desdeTxt, hastaTxt string, ahora time.Time) (Filtro, error) {
	hoy := ahora.In(Lima)
	hasta := time.Date(hoy.Year(), hoy.Month(), hoy.Day(), 0, 0, 0, 0, time.UTC)
	if hastaTxt != "" {
		h, err := parseFecha(hastaTxt)
		if err != nil {
			return Filtro{}, err
		}
		hasta = h
	}
	desde := time.Date(hasta.Year(), hasta.Month()-11, 1, 0, 0, 0, 0, time.UTC)
	if desdeTxt != "" {
		d, err := parseFecha(desdeTxt)
		if err != nil {
			return Filtro{}, err
		}
		desde = d
	}
	if desde.After(hasta) {
		return Filtro{}, ErrFiltro{"la fecha inicial no puede ser posterior a la final"}
	}
	if int(hasta.Sub(desde).Hours()/24)+1 > rangoMaximoDias {
		return Filtro{}, ErrFiltro{"el rango no puede superar 24 meses"}
	}
	return Filtro{Desde: desde, Hasta: hasta}, nil
}

func parseFecha(txt string) (time.Time, error) {
	t, err := time.Parse(formatoISO, txt)
	if err != nil {
		return time.Time{}, ErrFiltro{"fecha inválida: use AAAA-MM-DD"}
	}
	return t, nil
}

// DesdeISO devuelve el inicio del rango como AAAA-MM-DD.
func (f Filtro) DesdeISO() string { return f.Desde.Format(formatoISO) }

// HastaISO devuelve el fin del rango como AAAA-MM-DD.
func (f Filtro) HastaISO() string { return f.Hasta.Format(formatoISO) }
```

`gastos-service/internal/reportes/modelo.go`:

```go
package reportes

import "time"

// Resumen es lo que muestran el dashboard y los PDF para un rango de fechas.
type Resumen struct {
	Desde        string        `json:"desde"`
	Hasta        string        `json:"hasta"`
	Indicadores  Indicadores   `json:"indicadores"`
	PorMes       []Mes         `json:"por_mes"`
	PorEstado    []PorEstado   `json:"por_estado"`
	PorCliente   []FilaRanking `json:"por_cliente"`
	PorUnidad    []FilaRanking `json:"por_unidad"`
	PorConductor []FilaRanking `json:"por_conductor"`
	// Detalle solo se carga para el PDF de detalle.
	Detalle []FilaDetalle `json:"-"`
}

// Indicadores son las cifras principales del período.
type Indicadores struct {
	Total                float64 `json:"total"`
	Cantidad             int     `json:"cantidad"`
	Promedio             float64 `json:"promedio"`
	Sustentado           float64 `json:"sustentado"`
	Pendiente            float64 `json:"pendiente"`
	PorcentajeSustentado float64 `json:"porcentaje_sustentado"`
}

// Mes es el gasto de un mes (AAAA-MM) separado por estado.
type Mes struct {
	Mes        string  `json:"mes"`
	Registrado float64 `json:"registrado"`
	Pagado     float64 `json:"pagado"`
	Sustentado float64 `json:"sustentado"`
	Total      float64 `json:"total"`
}

// PorEstado es la participación de un estado en el período.
type PorEstado struct {
	Estado     string  `json:"estado"`
	Cantidad   int     `json:"cantidad"`
	Monto      float64 `json:"monto"`
	Porcentaje float64 `json:"porcentaje"`
}

// FilaRanking es un puesto de un ranking (cliente, unidad o conductor).
type FilaRanking struct {
	Puesto     int     `json:"puesto"`
	Etiqueta   string  `json:"etiqueta"`
	Cantidad   int     `json:"cantidad"`
	Monto      float64 `json:"monto"`
	Porcentaje float64 `json:"porcentaje"`
}

// FilaDetalle es un gasto del período con los datos de su viaje.
type FilaDetalle struct {
	ID        int
	Fecha     time.Time
	Cliente   string
	ViajeID   int
	Unidad    string
	Conductor string
	Estado    string
	Monto     float64
}
```

`gastos-service/internal/reportes/formato.go`:

```go
package reportes

import (
	"math"
	"strconv"
	"strings"
	"time"
)

var mesesCortos = [...]string{"ene", "feb", "mar", "abr", "may", "jun", "jul", "ago", "sep", "oct", "nov", "dic"}

// FormatoMonto escribe un monto como el frontend (formatMonto): S/ 1,234.50.
func FormatoMonto(v float64) string {
	txt := strconv.FormatFloat(math.Abs(v), 'f', 2, 64)
	entero, decimales, _ := strings.Cut(txt, ".")
	var b strings.Builder
	for i, c := range entero {
		if i > 0 && (len(entero)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	signo := ""
	if v < 0 {
		signo = "-"
	}
	return "S/ " + signo + b.String() + "." + decimales
}

// FormatoFecha escribe una fecha como 30/09/2026.
func FormatoFecha(t time.Time) string { return t.Format("02/01/2006") }

// FormatoEmision escribe el momento de emisión en hora de Lima: 17/10/2026 10:32.
func FormatoEmision(t time.Time) string { return t.In(Lima).Format("02/01/2006 15:04") }

// MesCorto convierte AAAA-MM en «oct 26».
func MesCorto(mes string) string {
	t, err := time.Parse("2006-01", mes)
	if err != nil {
		return mes
	}
	return mesesCortos[t.Month()-1] + " " + t.Format("06")
}

// FormatoPorcentaje escribe 42.5 como «42.5 %».
func FormatoPorcentaje(v float64) string {
	return strconv.FormatFloat(v, 'f', 1, 64) + " %"
}
```

- [ ] **Step 4: Ejecutar y verificar que pasan**

Run: `cd gastos-service && go test ./internal/reportes/ -count=1 -v 2>&1 | grep -E "^(--- |ok|FAIL)"`
Expected: 7 `--- PASS` y `ok`.

- [ ] **Step 5: Checkpoint (sin commit)**

Run: `cd gastos-service && gofmt -l . && go vet ./...`
Expected: sin salida de `gofmt -l`; `go vet` limpio.

---

### Task 2: Consultas SQL en una transacción y esquema de pruebas compartido (Go)

**Files:**
- Create: `gastos-service/internal/pruebas/esquema.go`
- Modify: `gastos-service/internal/handlers/checklist_integration_test.go` (el armado del esquema pasa a `internal/pruebas`)
- Create: `gastos-service/internal/reportes/consultas.go`
- Test: `gastos-service/internal/reportes/consultas_test.go`

**Interfaces:**
- Consumes: `Filtro`, `ParseFiltro`, tipos del modelo (Tarea 1).
- Produces:
  - `pruebas.NuevoEsquema(t testing.TB) *pgxpool.Pool`: esquema temporal con tablas mínimas `cliente(id_cliente, razon_social)`, `unidad(id_unidad, placa)`, `conductor(id_conductor, nombre1, apellido1, apellido2)`, `conductorunidad(id_conductor_unidad, id_conductor)`, `programacionviaje(id_viaje, id_tracto, id_conductor_unidad)` (columnas opcionales) y las migraciones del servicio; se omite sin `TEST_DATABASE_URL`.
  - `reportes.Iniciador` (lo cumple `*pgxpool.Pool`) y `reportes.Consultar(ctx, db Iniciador, f Filtro, conDetalle bool) (*Resumen, error)`: listas nunca `nil`; `Detalle` solo con `conDetalle`.

- [ ] **Step 1: Crear el esquema de pruebas compartido**

`gastos-service/internal/pruebas/esquema.go`:

```go
// Package pruebas prepara una base de datos aislada para las pruebas de integración.
// Solo lo importan archivos _test.go.
package pruebas

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"gastos-service/internal/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

// tablasMinimas imita, con solo las columnas que usa el servicio, las tablas del
// sistema principal a las que apuntan gasto y los reportes.
const tablasMinimas = `
	CREATE TABLE cliente (id_cliente SERIAL PRIMARY KEY, razon_social VARCHAR(150));
	CREATE TABLE unidad (id_unidad SERIAL PRIMARY KEY, placa VARCHAR(20));
	CREATE TABLE conductor (
		id_conductor SERIAL PRIMARY KEY,
		nombre1 VARCHAR(50), apellido1 VARCHAR(50), apellido2 VARCHAR(50)
	);
	CREATE TABLE conductorunidad (id_conductor_unidad SERIAL PRIMARY KEY, id_conductor INTEGER REFERENCES conductor (id_conductor));
	CREATE TABLE programacionviaje (
		id_viaje SERIAL PRIMARY KEY,
		id_tracto INTEGER REFERENCES unidad (id_unidad),
		id_conductor_unidad INTEGER REFERENCES conductorunidad (id_conductor_unidad)
	);`

// NuevoEsquema crea un esquema temporal con las tablas mínimas y las migraciones
// del servicio, y lo borra al terminar la prueba. Sin TEST_DATABASE_URL, la prueba se omite.
func NuevoEsquema(t testing.TB) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL no definida: se omiten las pruebas contra PostgreSQL")
	}
	ctx := context.Background()
	esquema := fmt.Sprintf("prueba_%d", time.Now().UnixNano())

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

	if _, err := pool.Exec(ctx, tablasMinimas); err != nil {
		t.Fatal(err)
	}
	if err := db.RunMigrations(pool, directorioMigraciones()); err != nil {
		t.Fatal(err)
	}
	return pool
}

// directorioMigraciones ubica gastos-service/migrations a partir de este archivo,
// para que funcione desde cualquier paquete de prueba.
func directorioMigraciones() string {
	_, archivo, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(archivo), "..", "..", "migrations")
}
```

- [ ] **Step 2: Usarlo en las pruebas del checklist (refactor, sin cambio de comportamiento)**

En `gastos-service/internal/handlers/checklist_integration_test.go`, reemplaza todo desde la primera línea (`package handlers_test`) hasta la línea anterior a `// comprobantePendiente crea un viaje…` por:

```go
package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"gastos-service/internal/checklist"
	"gastos-service/internal/config"
	"gastos-service/internal/handlers"
	"gastos-service/internal/models"
	"gastos-service/internal/pruebas"

	"github.com/jackc/pgx/v5/pgxpool"
)

type entorno struct {
	pool *pgxpool.Pool
	mux  *http.ServeMux
}

// nuevoEntorno arma un esquema temporal (ver internal/pruebas) y el mux del servicio.
func nuevoEntorno(t *testing.T) *entorno {
	t.Helper()
	pool := pruebas.NuevoEsquema(t)
	mux := http.NewServeMux()
	handlers.New(pool, config.Config{UploadsDir: t.TempDir()}).Register(mux)
	return &entorno{pool: pool, mux: mux}
}
```

Run: `cd gastos-service && TEST_DATABASE_URL="$(sed -n 's/^DATABASE_URL=//p' .env | tr -d "\"'")" go test ./internal/handlers/ -count=1 -v 2>&1 | grep -E "^(--- |ok|FAIL)"`
Expected: las 7 pruebas del checklist en PASS (ninguna SKIP).

- [ ] **Step 3: Escribir las pruebas de las consultas que fallan**

`gastos-service/internal/reportes/consultas_test.go`:

```go
package reportes_test

import (
	"context"
	"testing"
	"time"

	"gastos-service/internal/pruebas"
	"gastos-service/internal/reportes"

	"github.com/jackc/pgx/v5/pgxpool"
)

// datos crea dos clientes, dos tractos, dos conductores y tres viajes (uno sin tracto ni
// conductor), y estos gastos:
//
//	fecha       cliente  viaje            estado      monto
//	2026-01-31  Agro     1 (T-1, Ana)     Sustentado  100.00
//	2026-01-15  Agro     1 (T-1, Ana)     Pagado       50.25
//	2026-03-01  Minera   2 (T-2, Luis)    Registrado  200.00
//	2026-03-31  Minera   3 (sin asignar)  Sustentado   49.75
//	2025-12-31  Agro     1                Sustentado  999.00   fuera del rango
//	2026-04-01  Minera   2                Pagado      999.00   fuera del rango
func datos(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO cliente (id_cliente, razon_social) VALUES (1, 'Agro Norte SAC'), (2, 'Minera Sur SA');
		INSERT INTO unidad (id_unidad, placa) VALUES (1, 'T-1'), (2, 'T-2');
		INSERT INTO conductor (id_conductor, nombre1, apellido1, apellido2) VALUES (1, 'Ana', 'Quispe', NULL), (2, ' Luis ', 'Rojas', 'Peña');
		INSERT INTO conductorunidad (id_conductor_unidad, id_conductor) VALUES (1, 1), (2, 2);
		INSERT INTO programacionviaje (id_viaje, id_tracto, id_conductor_unidad) VALUES (1, 1, 1), (2, 2, 2), (3, NULL, NULL);
		INSERT INTO gasto (viaje_id, cliente_id, monto, fecha, estado, creado_por) VALUES
			(1, 1, 100.00, '2026-01-31', 'Sustentado', 'x'),
			(1, 1,  50.25, '2026-01-15', 'Pagado',     'x'),
			(2, 2, 200.00, '2026-03-01', 'Registrado', 'x'),
			(3, 2,  49.75, '2026-03-31', 'Sustentado', 'x'),
			(1, 1, 999.00, '2025-12-31', 'Sustentado', 'x'),
			(2, 2, 999.00, '2026-04-01', 'Pagado',     'x');`)
	if err != nil {
		t.Fatal(err)
	}
}

func consultar(t *testing.T, pool *pgxpool.Pool, desde, hasta string) *reportes.Resumen {
	t.Helper()
	f, err := reportes.ParseFiltro(desde, hasta, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	r, err := reportes.Consultar(context.Background(), pool, f, true)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestIndicadoresDelRangoInclusivo(t *testing.T) {
	pool := pruebas.NuevoEsquema(t)
	datos(t, pool)
	r := consultar(t, pool, "2026-01-01", "2026-03-31")
	want := reportes.Indicadores{Total: 400, Cantidad: 4, Promedio: 100, Sustentado: 149.75, Pendiente: 250.25, PorcentajeSustentado: 37.4}
	if r.Indicadores != want {
		t.Fatalf("indicadores = %+v, se esperaba %+v", r.Indicadores, want)
	}
	if r.Desde != "2026-01-01" || r.Hasta != "2026-03-31" {
		t.Fatalf("rango = %s .. %s", r.Desde, r.Hasta)
	}
}

func TestPorMesIncluyeMesesSinGastos(t *testing.T) {
	pool := pruebas.NuevoEsquema(t)
	datos(t, pool)
	r := consultar(t, pool, "2026-01-01", "2026-03-31")
	want := []reportes.Mes{
		{Mes: "2026-01", Registrado: 0, Pagado: 50.25, Sustentado: 100, Total: 150.25},
		{Mes: "2026-02"},
		{Mes: "2026-03", Registrado: 200, Pagado: 0, Sustentado: 49.75, Total: 249.75},
	}
	if len(r.PorMes) != len(want) {
		t.Fatalf("por mes = %+v", r.PorMes)
	}
	for i := range want {
		if r.PorMes[i] != want[i] {
			t.Errorf("mes %d = %+v, se esperaba %+v", i, r.PorMes[i], want[i])
		}
	}
}

func TestPorEstadoSiempreTraeLosTresEnOrden(t *testing.T) {
	pool := pruebas.NuevoEsquema(t)
	datos(t, pool)
	r := consultar(t, pool, "2026-02-01", "2026-02-28")
	if len(r.PorEstado) != 3 || r.PorEstado[0].Estado != "Registrado" || r.PorEstado[1].Estado != "Pagado" || r.PorEstado[2].Estado != "Sustentado" {
		t.Fatalf("por estado = %+v", r.PorEstado)
	}
	for _, e := range r.PorEstado {
		if e.Cantidad != 0 || e.Monto != 0 || e.Porcentaje != 0 {
			t.Errorf("un mes sin gastos debía dar ceros: %+v", e)
		}
	}
	r = consultar(t, pool, "2026-01-01", "2026-03-31")
	if r.PorEstado[0] != (reportes.PorEstado{Estado: "Registrado", Cantidad: 1, Monto: 200, Porcentaje: 50}) {
		t.Errorf("registrado = %+v", r.PorEstado[0])
	}
}

func TestRankingsConSinAsignarYNombresLimpios(t *testing.T) {
	pool := pruebas.NuevoEsquema(t)
	datos(t, pool)
	r := consultar(t, pool, "2026-01-01", "2026-03-31")
	if got := r.PorCliente; len(got) != 2 || got[0] != (reportes.FilaRanking{Puesto: 1, Etiqueta: "Minera Sur SA", Cantidad: 2, Monto: 249.75, Porcentaje: 62.4}) || got[1].Etiqueta != "Agro Norte SAC" || got[1].Puesto != 2 {
		t.Errorf("por cliente = %+v", got)
	}
	unidades := map[string]float64{}
	for _, f := range r.PorUnidad {
		unidades[f.Etiqueta] = f.Monto
	}
	if unidades["T-2"] != 200 || unidades["T-1"] != 150.25 || unidades["Sin asignar"] != 49.75 {
		t.Errorf("por unidad = %+v", r.PorUnidad)
	}
	conductores := map[string]bool{}
	for _, f := range r.PorConductor {
		conductores[f.Etiqueta] = true
	}
	if !conductores["Luis Rojas Peña"] || !conductores["Ana Quispe"] || !conductores["Sin asignar"] {
		t.Errorf("por conductor = %+v", r.PorConductor)
	}
}

func TestRankingEmpatadoCompartePuestoYOrdenaPorNombre(t *testing.T) {
	pool := pruebas.NuevoEsquema(t)
	_, err := pool.Exec(context.Background(), `
		INSERT INTO cliente (id_cliente, razon_social) VALUES (1, 'Zeta SAC'), (2, 'Alfa SAC'), (3, 'Beta SAC');
		INSERT INTO programacionviaje (id_viaje) VALUES (1);
		INSERT INTO gasto (viaje_id, cliente_id, monto, fecha, estado, creado_por) VALUES
			(1, 1, 80, '2026-05-10', 'Registrado', 'x'),
			(1, 2, 80, '2026-05-11', 'Registrado', 'x'),
			(1, 3, 40, '2026-05-12', 'Registrado', 'x');`)
	if err != nil {
		t.Fatal(err)
	}
	got := consultar(t, pool, "2026-05-01", "2026-05-31").PorCliente
	if len(got) != 3 || got[0].Etiqueta != "Alfa SAC" || got[0].Puesto != 1 || got[1].Etiqueta != "Zeta SAC" || got[1].Puesto != 1 || got[2].Puesto != 3 {
		t.Fatalf("ranking con empate = %+v", got)
	}
}

func TestRankingMuestraHastaDiez(t *testing.T) {
	pool := pruebas.NuevoEsquema(t)
	_, err := pool.Exec(context.Background(), `
		INSERT INTO cliente (id_cliente, razon_social) SELECT n, 'Cliente ' || n FROM generate_series(1, 12) n;
		INSERT INTO programacionviaje (id_viaje) VALUES (1);
		INSERT INTO gasto (viaje_id, cliente_id, monto, fecha, estado, creado_por)
			SELECT 1, n, n * 10, '2026-05-10', 'Registrado', 'x' FROM generate_series(1, 12) n;`)
	if err != nil {
		t.Fatal(err)
	}
	got := consultar(t, pool, "2026-05-01", "2026-05-31").PorCliente
	if len(got) != 10 || got[0].Etiqueta != "Cliente 12" || got[9].Etiqueta != "Cliente 3" {
		t.Fatalf("top 10 = %+v", got)
	}
}

func TestDetalleOrdenadoPorFechaConDatosDelViaje(t *testing.T) {
	pool := pruebas.NuevoEsquema(t)
	datos(t, pool)
	d := consultar(t, pool, "2026-01-01", "2026-03-31").Detalle
	if len(d) != 4 {
		t.Fatalf("detalle = %+v", d)
	}
	if d[0].Fecha.Format("2006-01-02") != "2026-01-15" || d[0].Cliente != "Agro Norte SAC" || d[0].Unidad != "T-1" || d[0].Conductor != "Ana Quispe" || d[0].Monto != 50.25 {
		t.Errorf("primera fila = %+v", d[0])
	}
	if d[3].Unidad != "Sin asignar" || d[3].Conductor != "Sin asignar" {
		t.Errorf("última fila = %+v", d[3])
	}
}

func TestSinGastosDevuelveListasVacias(t *testing.T) {
	pool := pruebas.NuevoEsquema(t)
	r := consultar(t, pool, "2026-01-01", "2026-01-31")
	if r.Indicadores != (reportes.Indicadores{}) || len(r.PorMes) != 1 || r.PorCliente == nil || len(r.PorCliente) != 0 || r.Detalle == nil {
		t.Fatalf("resumen vacío = %+v", r)
	}
}
```

- [ ] **Step 4: Ejecutar y verificar que fallan**

Run: `cd gastos-service && TEST_DATABASE_URL="$(sed -n 's/^DATABASE_URL=//p' .env | tr -d "\"'")" go test ./internal/reportes/ -count=1`
Expected: FAIL de compilación: `undefined: reportes.Consultar`.

- [ ] **Step 5: Implementar las consultas**

`gastos-service/internal/reportes/consultas.go`:

```go
package reportes

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Iniciador abre transacciones (lo cumple *pgxpool.Pool).
type Iniciador interface {
	BeginTx(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error)
}

// Uniones de un gasto con su cliente, su viaje, la unidad (tracto) y el conductor.
// Todas son LEFT JOIN: lo que no se pueda unir se muestra como «Sin asignar».
const desdeGastos = `
	FROM gasto g
	LEFT JOIN cliente c ON c.id_cliente = g.cliente_id
	LEFT JOIN programacionviaje v ON v.id_viaje = g.viaje_id
	LEFT JOIN unidad u ON u.id_unidad = v.id_tracto
	LEFT JOIN conductorunidad cu ON cu.id_conductor_unidad = v.id_conductor_unidad
	LEFT JOIN conductor d ON d.id_conductor = cu.id_conductor
	WHERE g.fecha BETWEEN $1::date AND $2::date`

const (
	etiquetaCliente   = `COALESCE(NULLIF(trim(c.razon_social), ''), 'Sin asignar')`
	etiquetaUnidad    = `COALESCE(NULLIF(trim(u.placa), ''), 'Sin asignar')`
	etiquetaConductor = `COALESCE(NULLIF(regexp_replace(trim(concat_ws(' ', d.nombre1, d.apellido1, d.apellido2)), '\s+', ' ', 'g'), ''), 'Sin asignar')`
)

const sqlIndicadores = `
	SELECT count(*)::int,
	       COALESCE(round(sum(monto), 2), 0)::float8,
	       COALESCE(round(avg(monto), 2), 0)::float8,
	       COALESCE(round(sum(monto) FILTER (WHERE estado = 'Sustentado'), 2), 0)::float8,
	       COALESCE(round(sum(monto) FILTER (WHERE estado IN ('Registrado', 'Pagado')), 2), 0)::float8,
	       COALESCE(round(100 * sum(monto) FILTER (WHERE estado = 'Sustentado') / NULLIF(sum(monto), 0), 1), 0)::float8
	FROM gasto
	WHERE fecha BETWEEN $1::date AND $2::date`

const sqlPorMes = `
	WITH meses AS (
		SELECT generate_series(date_trunc('month', $1::date), date_trunc('month', $2::date), interval '1 month')::date AS mes
	),
	gastos_mes AS (
		SELECT date_trunc('month', fecha)::date AS mes, estado, monto
		FROM gasto
		WHERE fecha BETWEEN $1::date AND $2::date
	)
	SELECT to_char(m.mes, 'YYYY-MM'),
	       COALESCE(round(sum(g.monto) FILTER (WHERE g.estado = 'Registrado'), 2), 0)::float8,
	       COALESCE(round(sum(g.monto) FILTER (WHERE g.estado = 'Pagado'), 2), 0)::float8,
	       COALESCE(round(sum(g.monto) FILTER (WHERE g.estado = 'Sustentado'), 2), 0)::float8,
	       COALESCE(round(sum(g.monto), 2), 0)::float8
	FROM meses m
	LEFT JOIN gastos_mes g ON g.mes = m.mes
	GROUP BY m.mes
	ORDER BY m.mes`

const sqlPorEstado = `
	WITH estados (estado, orden) AS (
		VALUES ('Registrado', 1), ('Pagado', 2), ('Sustentado', 3)
	),
	por_estado AS (
		SELECT e.estado, e.orden, count(g.id) AS cantidad, COALESCE(sum(g.monto), 0) AS monto
		FROM estados e
		LEFT JOIN gasto g ON g.estado = e.estado AND g.fecha BETWEEN $1::date AND $2::date
		GROUP BY e.estado, e.orden
	)
	SELECT estado,
	       cantidad::int,
	       round(monto, 2)::float8,
	       COALESCE(round(100 * monto / NULLIF(sum(monto) OVER (), 0), 1), 0)::float8
	FROM por_estado
	ORDER BY orden`

// sqlRanking agrupa por la etiqueta de una dimensión y numera con RANK().
// %s es una de las constantes etiqueta*, nunca un dato del usuario.
const sqlRanking = `
	WITH base AS (
		SELECT g.monto, %s AS etiqueta` + desdeGastos + `
	),
	agrupado AS (
		SELECT etiqueta, count(*) AS cantidad, sum(monto) AS monto
		FROM base
		GROUP BY etiqueta
	),
	ranqueado AS (
		SELECT etiqueta, cantidad, monto,
		       rank() OVER (ORDER BY monto DESC) AS puesto,
		       100 * monto / NULLIF(sum(monto) OVER (), 0) AS porcentaje
		FROM agrupado
	)
	SELECT puesto::int, etiqueta, cantidad::int, round(monto, 2)::float8, COALESCE(round(porcentaje, 1), 0)::float8
	FROM ranqueado
	ORDER BY monto DESC, etiqueta ASC
	LIMIT 10`

var sqlDetalle = `
	SELECT g.id, g.fecha, ` + etiquetaCliente + `, g.viaje_id, ` + etiquetaUnidad + `, ` + etiquetaConductor + `, g.estado, round(g.monto, 2)::float8` +
	desdeGastos + `
	ORDER BY g.fecha, g.id`

// Consultar calcula el resumen del filtro en una sola transacción de solo lectura
// (REPEATABLE READ): todas las cifras salen de la misma foto de los datos.
func Consultar(ctx context.Context, db Iniciador, f Filtro, conDetalle bool) (*Resumen, error) {
	tx, err := db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	r := &Resumen{Desde: f.DesdeISO(), Hasta: f.HastaISO()}
	args := []any{f.Desde, f.Hasta}

	i := &r.Indicadores
	if err := tx.QueryRow(ctx, sqlIndicadores, args...).Scan(&i.Cantidad, &i.Total, &i.Promedio, &i.Sustentado, &i.Pendiente, &i.PorcentajeSustentado); err != nil {
		return nil, fmt.Errorf("indicadores: %w", err)
	}
	if r.PorMes, err = consultarFilas(ctx, tx, sqlPorMes, args, func(row pgx.CollectableRow) (Mes, error) {
		var m Mes
		err := row.Scan(&m.Mes, &m.Registrado, &m.Pagado, &m.Sustentado, &m.Total)
		return m, err
	}); err != nil {
		return nil, fmt.Errorf("por mes: %w", err)
	}
	if r.PorEstado, err = consultarFilas(ctx, tx, sqlPorEstado, args, func(row pgx.CollectableRow) (PorEstado, error) {
		var e PorEstado
		err := row.Scan(&e.Estado, &e.Cantidad, &e.Monto, &e.Porcentaje)
		return e, err
	}); err != nil {
		return nil, fmt.Errorf("por estado: %w", err)
	}
	rankings := []struct {
		destino  *[]FilaRanking
		etiqueta string
	}{
		{&r.PorCliente, etiquetaCliente},
		{&r.PorUnidad, etiquetaUnidad},
		{&r.PorConductor, etiquetaConductor},
	}
	for _, rk := range rankings {
		if *rk.destino, err = consultarFilas(ctx, tx, fmt.Sprintf(sqlRanking, rk.etiqueta), args, func(row pgx.CollectableRow) (FilaRanking, error) {
			var fr FilaRanking
			err := row.Scan(&fr.Puesto, &fr.Etiqueta, &fr.Cantidad, &fr.Monto, &fr.Porcentaje)
			return fr, err
		}); err != nil {
			return nil, fmt.Errorf("ranking: %w", err)
		}
	}
	if conDetalle {
		if r.Detalle, err = consultarFilas(ctx, tx, sqlDetalle, args, func(row pgx.CollectableRow) (FilaDetalle, error) {
			var d FilaDetalle
			err := row.Scan(&d.ID, &d.Fecha, &d.Cliente, &d.ViajeID, &d.Unidad, &d.Conductor, &d.Estado, &d.Monto)
			return d, err
		}); err != nil {
			return nil, fmt.Errorf("detalle: %w", err)
		}
	}
	return r, tx.Commit(ctx)
}

func consultarFilas[T any](ctx context.Context, tx pgx.Tx, sql string, args []any, leer pgx.RowToFunc[T]) ([]T, error) {
	rows, err := tx.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	filas, err := pgx.CollectRows(rows, leer)
	if err != nil {
		return nil, err
	}
	if filas == nil {
		filas = []T{}
	}
	return filas, nil
}
```

- [ ] **Step 6: Ejecutar y verificar que pasan**

Run: `cd gastos-service && TEST_DATABASE_URL="$(sed -n 's/^DATABASE_URL=//p' .env | tr -d "\"'")" go test ./internal/reportes/ ./internal/handlers/ -count=1 -v 2>&1 | grep -E "^(--- |ok|FAIL)|SKIP"`
Expected: 15 PASS en `internal/reportes` (7 de la Tarea 1 + 8 nuevas) y 7 en `internal/handlers`; ninguna SKIP.

- [ ] **Step 7: Checkpoint (sin commit)**

Run:
```bash
cd gastos-service && gofmt -l . && go vet ./...
psql "$(sed -n 's/^DATABASE_URL=//p' .env | tr -d "\"'")" -tAc "SELECT count(*) FROM information_schema.schemata WHERE schema_name LIKE 'prueba_%' OR schema_name LIKE 'checklist_prueba_%'"
```
Expected: sin salida de gofmt, vet limpio y `0` esquemas sobrantes.

---

### Task 3: PDF estadístico y de detalle (Go + gopdf)

**Files:**
- Modify: `gastos-service/go.mod`, `gastos-service/go.sum` (dependencia `github.com/signintech/gopdf v0.38.1`)
- Create: `gastos-service/internal/reportes/fuentes/overpass-400.ttf`, `overpass-700.ttf`, `OFL.txt`
- Create: `gastos-service/internal/reportes/pdf.go`, `pdf_estadistico.go`, `pdf_detalle.go`
- Test: `gastos-service/internal/reportes/pdf_test.go`

**Interfaces:**
- Consumes: modelo y formatos (Tarea 1).
- Produces: `reportes.PDFEstadistico(r *Resumen, ahora time.Time) ([]byte, error)` y `reportes.PDFDetalle(r *Resumen, ahora time.Time) ([]byte, error)` (este usa `r.Detalle`, `r.PorEstado` y `r.Indicadores`); función interna `escalaAgradable(maximo) (tope, paso)`.

- [ ] **Step 1: Agregar la dependencia y las fuentes**

Run:
```bash
cd gastos-service && go get github.com/signintech/gopdf@v0.38.1
mkdir -p internal/reportes/fuentes && python3 - <<'EOF'
import urllib.request
base = 'https://cdn.jsdelivr.net/fontsource/fonts/overpass@latest/'
for peso in ('400', '700'):
    urllib.request.urlretrieve(base + f'latin-{peso}-normal.ttf', f'internal/reportes/fuentes/overpass-{peso}.ttf')
urllib.request.urlretrieve('https://raw.githubusercontent.com/google/fonts/main/ofl/overpass/OFL.txt', 'internal/reportes/fuentes/OFL.txt')
EOF
file internal/reportes/fuentes/*.ttf && head -1 internal/reportes/fuentes/OFL.txt
```
Expected: `go: added github.com/signintech/gopdf v0.38.1`; ambos archivos `TrueType Font data`; `Copyright 2021 The Overpass Project Authors`.

- [ ] **Step 2: Escribir las pruebas que fallan**

`gastos-service/internal/reportes/pdf_test.go`:

```go
package reportes

import (
	"bytes"
	"fmt"
	"regexp"
	"testing"
	"time"
)

var paginaPDF = regexp.MustCompile(`/Type\s*/Page[^s]`)

func paginas(t *testing.T, pdf []byte) int {
	t.Helper()
	if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatalf("no es un PDF: %q", pdf[:min(len(pdf), 16)])
	}
	return len(paginaPDF.FindAll(pdf, -1))
}

// resumenDePrueba arma un resumen con n gastos y 12 meses de datos.
func resumenDePrueba(n int) *Resumen {
	r := &Resumen{Desde: "2025-11-01", Hasta: "2026-10-31"}
	estados := []string{"Registrado", "Pagado", "Sustentado"}
	for i := 0; i < 12; i++ {
		r.PorMes = append(r.PorMes, Mes{Mes: fmt.Sprintf("%04d-%02d", 2025+(10+i)/12, (10+i)%12+1), Registrado: 100, Pagado: 200, Sustentado: float64(300 + i*50)})
	}
	for _, e := range estados {
		r.PorEstado = append(r.PorEstado, PorEstado{Estado: e, Cantidad: n / 3, Monto: 1000, Porcentaje: 33.3})
	}
	for p := 1; p <= 10; p++ {
		fila := FilaRanking{Puesto: p, Etiqueta: fmt.Sprintf("Cliente con un nombre bastante largo número %d SAC", p), Cantidad: 11 - p, Monto: float64(1100 - p*100), Porcentaje: float64(11 - p)}
		r.PorCliente = append(r.PorCliente, fila)
		r.PorUnidad = append(r.PorUnidad, fila)
		r.PorConductor = append(r.PorConductor, fila)
	}
	for i := 0; i < n; i++ {
		r.Detalle = append(r.Detalle, FilaDetalle{
			ID: i + 1, Fecha: time.Date(2026, 1, 1+i%28, 0, 0, 0, 0, time.UTC), Cliente: "Agroexportadora Norte del Perú SAC",
			ViajeID: i % 7, Unidad: "T1A-123", Conductor: "José Ñique Peña", Estado: estados[i%3], Monto: 85.5,
		})
	}
	r.Indicadores = Indicadores{Total: 3000, Cantidad: n, Promedio: 85.5, Sustentado: 1000, Pendiente: 2000, PorcentajeSustentado: 33.3}
	return r
}

func TestPDFEstadisticoConDatosOcupaDosPaginas(t *testing.T) {
	pdf, err := PDFEstadistico(resumenDePrueba(30), ahoraPrueba)
	if err != nil {
		t.Fatal(err)
	}
	if n := paginas(t, pdf); n != 2 {
		t.Fatalf("páginas = %d, se esperaban 2", n)
	}
}

func TestPDFEstadisticoCon24Meses(t *testing.T) {
	r := resumenDePrueba(30)
	r.PorMes = nil
	for i := 0; i < 24; i++ {
		r.PorMes = append(r.PorMes, Mes{Mes: fmt.Sprintf("%04d-%02d", 2024+i/12, i%12+1), Registrado: 10, Pagado: 20, Sustentado: 30, Total: 60})
	}
	pdf, err := PDFEstadistico(r, ahoraPrueba)
	if err != nil {
		t.Fatal(err)
	}
	if n := paginas(t, pdf); n != 2 {
		t.Fatalf("páginas = %d, se esperaban 2", n)
	}
}

func TestPDFDetalleRepartePorPaginas(t *testing.T) {
	// 27 filas por página: 60 gastos ocupan 3 páginas, con los totales al final de la tercera.
	pdf, err := PDFDetalle(resumenDePrueba(60), ahoraPrueba)
	if err != nil {
		t.Fatal(err)
	}
	if n := paginas(t, pdf); n != 3 {
		t.Fatalf("páginas = %d, se esperaban 3", n)
	}
}

func TestPDFSinGastosEsUnaPagina(t *testing.T) {
	vacio := &Resumen{Desde: "2026-10-01", Hasta: "2026-10-31", PorMes: []Mes{}, PorEstado: []PorEstado{}}
	for nombre, generar := range map[string]func(*Resumen, time.Time) ([]byte, error){"estadístico": PDFEstadistico, "detalle": PDFDetalle} {
		pdf, err := generar(vacio, ahoraPrueba)
		if err != nil {
			t.Fatalf("%s: %v", nombre, err)
		}
		if n := paginas(t, pdf); n != 1 {
			t.Errorf("%s vacío: páginas = %d, se esperaba 1", nombre, n)
		}
	}
}

func TestEscalaAgradable(t *testing.T) {
	casos := []struct{ maximo, tope, paso float64 }{
		{0, 4, 1},
		{7950, 8000, 2000},
		{8100, 10000, 2500},
		{120, 200, 50},
	}
	for _, c := range casos {
		if tope, paso := escalaAgradable(c.maximo); tope != c.tope || paso != c.paso {
			t.Errorf("escalaAgradable(%v) = %v, %v; se esperaba %v, %v", c.maximo, tope, paso, c.tope, c.paso)
		}
	}
}
```

- [ ] **Step 3: Ejecutar y verificar que fallan**

Run: `cd gastos-service && go test ./internal/reportes/ -count=1 -run 'PDF|Escala'`
Expected: FAIL de compilación: `undefined: PDFEstadistico`, `undefined: escalaAgradable`…

- [ ] **Step 4: Implementar el documento base**

`gastos-service/internal/reportes/pdf.go`:

```go
package reportes

import (
	_ "embed"
	"strconv"
	"time"

	"github.com/signintech/gopdf"
)

// Overpass (SIL Open Font License 1.1, ver fuentes/OFL.txt): la misma letra del frontend.
//
//go:embed fuentes/overpass-400.ttf
var fuenteRegular []byte

//go:embed fuentes/overpass-700.ttf
var fuenteNegrita []byte

type color struct{ r, g, b uint8 }

// Colores de los tokens del frontend (styles/tokens.css).
var (
	colorRojo      = color{225, 29, 72}
	colorRojoTinta = color{180, 18, 58}
	colorTinta     = color{30, 33, 36}
	colorTinta2    = color{74, 80, 86}
	colorTinta3    = color{101, 107, 113}
	colorLinea     = color{226, 227, 224}
	colorPanel2    = color{240, 241, 238}
)

// ordenEstados es el orden del proceso; los colores son los de cada estado en la app.
var ordenEstados = []string{"Registrado", "Pagado", "Sustentado"}

var coloresEstado = map[string]color{
	"Registrado": {138, 144, 150},
	"Pagado":     {37, 99, 235},
	"Sustentado": {21, 128, 61},
}

const (
	margen     = 40.0
	textoVacio = "No hay gastos registrados en este período."
)

// documento envuelve gopdf con un cursor vertical, la cabecera de LIZAR y el pie
// con «Página N de M». Guarda el primer error de dibujo y lo devuelve al final.
type documento struct {
	pdf     *gopdf.GoPdf
	ancho   float64
	alto    float64
	titulo  string
	periodo string
	emision string
	y       float64
	err     error
	// alNuevaPagina dibuja lo que se repite debajo de la cabecera (el encabezado de una tabla).
	alNuevaPagina func()
}

func nuevoDocumento(pagina *gopdf.Rect, titulo string, r *Resumen, ahora time.Time) (*documento, error) {
	pdf := &gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *pagina})
	if err := pdf.AddTTFFontData("overpass", fuenteRegular); err != nil {
		return nil, err
	}
	if err := pdf.AddTTFFontData("overpass-negrita", fuenteNegrita); err != nil {
		return nil, err
	}
	d := &documento{
		pdf:     pdf,
		ancho:   pagina.W,
		alto:    pagina.H,
		titulo:  titulo,
		periodo: "Período: " + fechaTexto(r.Desde) + " – " + fechaTexto(r.Hasta),
		emision: "Emitido: " + FormatoEmision(ahora),
	}
	d.nuevaPagina()
	return d, nil
}

func fechaTexto(iso string) string {
	t, err := time.Parse(formatoISO, iso)
	if err != nil {
		return iso
	}
	return FormatoFecha(t)
}

func (d *documento) anchoUtil() float64 { return d.ancho - 2*margen }

func (d *documento) nuevaPagina() {
	d.pdf.AddPage()
	d.cabecera()
	if d.alNuevaPagina != nil {
		d.alNuevaPagina()
	}
}

// asegurar pasa a una página nueva si lo que sigue no cabe en la actual.
func (d *documento) asegurar(alto float64) {
	if d.y+alto > d.alto-margen {
		d.nuevaPagina()
	}
}

func (d *documento) cabecera() {
	d.rect(0, 0, d.ancho, 6, colorRojo)
	d.texto(margen, 22, d.anchoUtil(), 12, "LIZAR SRL · Fleet Manager Pro", true, 9, colorRojoTinta, gopdf.Left)
	d.texto(margen, 36, d.anchoUtil(), 22, d.titulo, true, 18, colorTinta, gopdf.Left)
	d.texto(margen, 60, d.anchoUtil(), 12, d.periodo+"   ·   "+d.emision, false, 9, colorTinta2, gopdf.Left)
	d.linea(margen, 80, d.ancho-margen, 80, colorLinea)
	d.y = 96
}

func (d *documento) pie(numero, total int) {
	y := d.alto - 28
	d.texto(margen, y, d.anchoUtil(), 10, "LIZAR SRL · Trujillo, Perú", false, 8, colorTinta3, gopdf.Left)
	d.texto(margen, y, d.anchoUtil(), 10, "Página "+strconv.Itoa(numero)+" de "+strconv.Itoa(total), false, 8, colorTinta3, gopdf.Right)
}

// finalizar escribe el pie en cada página (ya se sabe el total) y devuelve el PDF.
func (d *documento) finalizar() ([]byte, error) {
	total := d.pdf.GetNumberOfPages()
	for i := 1; i <= total; i++ {
		if err := d.pdf.SetPage(i); err != nil {
			return nil, err
		}
		d.pie(i, total)
	}
	if d.err != nil {
		return nil, d.err
	}
	return d.pdf.GetBytesPdf(), nil
}

func (d *documento) fuente(negrita bool, tamano float64) {
	familia := "overpass"
	if negrita {
		familia = "overpass-negrita"
	}
	if err := d.pdf.SetFont(familia, "", tamano); err != nil && d.err == nil {
		d.err = err
	}
}

// texto escribe una línea dentro de un recuadro, centrada en vertical y recortada con «…» si no cabe.
func (d *documento) texto(x, y, ancho, alto float64, txt string, negrita bool, tamano float64, c color, alineacion int) {
	d.fuente(negrita, tamano)
	d.pdf.SetTextColor(c.r, c.g, c.b)
	d.pdf.SetXY(x, y)
	if err := d.pdf.CellWithOption(&gopdf.Rect{W: ancho, H: alto}, d.recortar(txt, ancho), gopdf.CellOption{Align: alineacion | gopdf.Middle}); err != nil && d.err == nil {
		d.err = err
	}
}

func (d *documento) recortar(txt string, ancho float64) string {
	if w, err := d.pdf.MeasureTextWidth(txt); err != nil || w <= ancho {
		return txt
	}
	r := []rune(txt)
	for len(r) > 0 {
		r = r[:len(r)-1]
		if w, _ := d.pdf.MeasureTextWidth(string(r) + "…"); w <= ancho {
			break
		}
	}
	return string(r) + "…"
}

func (d *documento) rect(x, y, ancho, alto float64, c color) {
	if ancho <= 0 || alto <= 0 {
		return
	}
	d.pdf.SetFillColor(c.r, c.g, c.b)
	d.pdf.RectFromUpperLeftWithStyle(x, y, ancho, alto, "F")
}

func (d *documento) linea(x1, y1, x2, y2 float64, c color) {
	d.pdf.SetStrokeColor(c.r, c.g, c.b)
	d.pdf.SetLineWidth(0.6)
	d.pdf.Line(x1, y1, x2, y2)
}

// tituloSeccion escribe el título de una sección y deja el cursor debajo.
func (d *documento) tituloSeccion(titulo string, altoContenido float64) {
	d.asegurar(22 + altoContenido)
	d.texto(margen, d.y, d.anchoUtil(), 16, titulo, true, 12, colorTinta, gopdf.Left)
	d.y += 22
}

func (d *documento) parrafoVacio() {
	d.texto(margen, d.y, d.anchoUtil(), 16, textoVacio, false, 11, colorTinta2, gopdf.Left)
	d.y += 24
}

// columna de una tabla: título, ancho y alineación (gopdf.Left o gopdf.Right).
type columna struct {
	titulo     string
	ancho      float64
	alineacion int
}

const altoFila = 16.0

func (d *documento) encabezadoTabla(columnas []columna) {
	d.rect(margen, d.y, d.anchoUtil(), altoFila+2, colorPanel2)
	x := margen
	for _, c := range columnas {
		d.texto(x+4, d.y, c.ancho-8, altoFila+2, c.titulo, true, 8, colorTinta2, c.alineacion)
		x += c.ancho
	}
	d.y += altoFila + 2
}

func (d *documento) fila(columnas []columna, valores []string, negrita bool) {
	x := margen
	for i, c := range columnas {
		d.texto(x+4, d.y, c.ancho-8, altoFila, valores[i], negrita, 8.5, colorTinta, c.alineacion)
		x += c.ancho
	}
	d.y += altoFila
}
```

- [ ] **Step 5: Implementar los dos PDF**

`gastos-service/internal/reportes/pdf_estadistico.go`:

```go
package reportes

import (
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/signintech/gopdf"
)

// PDFEstadistico genera el reporte estadístico (A4 vertical): indicadores, gasto por mes,
// distribución por estado y los rankings por cliente, unidad y conductor.
func PDFEstadistico(r *Resumen, ahora time.Time) ([]byte, error) {
	d, err := nuevoDocumento(gopdf.PageSizeA4, "Reporte estadístico de gastos", r, ahora)
	if err != nil {
		return nil, err
	}
	d.indicadores(r.Indicadores)
	if r.Indicadores.Cantidad == 0 {
		d.parrafoVacio()
		return d.finalizar()
	}
	d.graficoPorMes(r.PorMes)
	d.distribucionEstados(r.PorEstado)
	d.ranking("Gasto por cliente", "Cliente", r.PorCliente)
	d.ranking("Gasto por unidad", "Unidad (placa)", r.PorUnidad)
	d.ranking("Gasto por conductor", "Conductor", r.PorConductor)
	return d.finalizar()
}

func (d *documento) indicadores(i Indicadores) {
	const separacion, altoRecuadro = 10.0, 58.0
	ancho := (d.anchoUtil() - 3*separacion) / 4
	recuadros := []struct{ etiqueta, valor, nota string }{
		{"Total gastado", FormatoMonto(i.Total), ""},
		{"Gastos registrados", strconv.Itoa(i.Cantidad), ""},
		{"Gasto promedio", FormatoMonto(i.Promedio), ""},
		{"Sustentado", FormatoPorcentaje(i.PorcentajeSustentado), "Pendiente: " + FormatoMonto(i.Pendiente)},
	}
	for n, rc := range recuadros {
		x := margen + float64(n)*(ancho+separacion)
		d.rect(x, d.y, ancho, altoRecuadro, colorPanel2)
		d.texto(x+10, d.y+8, ancho-20, 10, rc.etiqueta, false, 8, colorTinta2, gopdf.Left)
		d.texto(x+10, d.y+21, ancho-20, 18, rc.valor, true, 14, colorTinta, gopdf.Left)
		if rc.nota != "" {
			d.texto(x+10, d.y+41, ancho-20, 10, rc.nota, false, 7.5, colorTinta3, gopdf.Left)
		}
	}
	d.y += altoRecuadro + 22
}

// escalaAgradable redondea el máximo hacia arriba a 4 pasos de 1, 2, 2.5 o 5 por potencia de 10.
func escalaAgradable(maximo float64) (tope, paso float64) {
	if maximo <= 0 {
		return 4, 1
	}
	bruto := maximo / 4
	potencia := math.Pow(10, math.Floor(math.Log10(bruto)))
	for _, m := range []float64{1, 2, 2.5, 5, 10} {
		if m*potencia >= bruto {
			paso = m * potencia
			break
		}
	}
	return 4 * paso, paso
}

func montoEje(v float64) string { return strings.TrimSuffix(FormatoMonto(v), ".00") }

func (d *documento) graficoPorMes(meses []Mes) {
	const altoGrafico, anchoEje = 150.0, 58.0
	d.tituloSeccion("Gasto por mes", altoGrafico+50)
	maximo := 0.0
	for _, m := range meses {
		maximo = math.Max(maximo, m.Total)
	}
	tope, paso := escalaAgradable(maximo)
	x0, y0 := margen+anchoEje, d.y+altoGrafico
	anchoPlot := d.anchoUtil() - anchoEje
	for k := 0; k <= 4; k++ {
		y := y0 - float64(k)*paso/tope*altoGrafico
		d.linea(x0, y, x0+anchoPlot, y, colorLinea)
		d.texto(margen, y-5, anchoEje-6, 10, montoEje(float64(k)*paso), false, 7, colorTinta3, gopdf.Right)
	}
	if len(meses) > 0 {
		hueco := anchoPlot / float64(len(meses))
		anchoBarra := math.Min(hueco*0.6, 26)
		saltoEtiqueta := 1
		if len(meses) > 12 {
			saltoEtiqueta = 2
		}
		for n, m := range meses {
			x := x0 + float64(n)*hueco + (hueco-anchoBarra)/2
			y := y0
			for _, estado := range ordenEstados {
				alto := montoDeEstado(m, estado) / tope * altoGrafico
				d.rect(x, y-alto, anchoBarra, alto, coloresEstado[estado])
				y -= alto
			}
			if n%saltoEtiqueta == 0 {
				anchoEtiqueta := hueco * float64(saltoEtiqueta)
				centro := x0 + float64(n)*hueco + hueco/2
				d.texto(centro-anchoEtiqueta/2, y0+3, anchoEtiqueta, 10, MesCorto(m.Mes), false, 7, colorTinta2, gopdf.Center)
			}
		}
	}
	d.y = y0 + 18
	d.leyenda()
	d.y += 20
}

func montoDeEstado(m Mes, estado string) float64 {
	switch estado {
	case "Registrado":
		return m.Registrado
	case "Pagado":
		return m.Pagado
	default:
		return m.Sustentado
	}
}

func (d *documento) leyenda() {
	x := margen + 58
	for _, estado := range ordenEstados {
		d.rect(x, d.y+2, 8, 8, coloresEstado[estado])
		d.texto(x+12, d.y, 80, 12, estado, false, 8, colorTinta2, gopdf.Left)
		x += 90
	}
	d.y += 14
}

func (d *documento) distribucionEstados(estados []PorEstado) {
	d.tituloSeccion("Distribución por estado", altoFila*float64(len(estados)+1))
	columnas := []columna{{"Estado", 120, gopdf.Left}, {"Gastos", 70, gopdf.Right}, {"Monto", 110, gopdf.Right}, {"Participación", d.anchoUtil() - 300, gopdf.Left}}
	d.encabezadoTabla(columnas)
	for _, e := range estados {
		d.rect(margen+4, d.y+4, 8, 8, coloresEstado[e.Estado])
		d.texto(margen+16, d.y, 100, altoFila, e.Estado, false, 8.5, colorTinta, gopdf.Left)
		d.texto(margen+120, d.y, 66, altoFila, strconv.Itoa(e.Cantidad), false, 8.5, colorTinta, gopdf.Right)
		d.texto(margen+190, d.y, 106, altoFila, FormatoMonto(e.Monto), false, 8.5, colorTinta, gopdf.Right)
		d.barraPorcentaje(margen+300, columnas[3].ancho, e.Porcentaje, coloresEstado[e.Estado])
		d.y += altoFila
	}
	d.y += 18
}

// barraPorcentaje dibuja una barra proporcional y el porcentaje a su derecha.
func (d *documento) barraPorcentaje(x, ancho, porcentaje float64, c color) {
	anchoBarra := ancho - 56
	d.rect(x+4, d.y+5, anchoBarra, 6, colorPanel2)
	d.rect(x+4, d.y+5, anchoBarra*math.Min(porcentaje, 100)/100, 6, c)
	d.texto(x+anchoBarra+8, d.y, 44, altoFila, FormatoPorcentaje(porcentaje), false, 8, colorTinta2, gopdf.Right)
}

func (d *documento) ranking(titulo, nombreColumna string, filas []FilaRanking) {
	d.tituloSeccion(titulo, altoFila*float64(len(filas)+2))
	columnas := []columna{{"Puesto", 44, gopdf.Left}, {nombreColumna, 190, gopdf.Left}, {"Gastos", 54, gopdf.Right}, {"Monto", 90, gopdf.Right}, {"%", d.anchoUtil() - 378, gopdf.Left}}
	d.encabezadoTabla(columnas)
	for _, f := range filas {
		d.fila(columnas[:4], []string{strconv.Itoa(f.Puesto), f.Etiqueta, strconv.Itoa(f.Cantidad), FormatoMonto(f.Monto)}, false)
		d.y -= altoFila
		d.barraPorcentaje(margen+378, columnas[4].ancho, f.Porcentaje, colorTinta2)
		d.y += altoFila
	}
	d.y += 18
}
```

`gastos-service/internal/reportes/pdf_detalle.go`:

```go
package reportes

import (
	"strconv"
	"time"

	"github.com/signintech/gopdf"
)

// PDFDetalle genera el listado de gastos del período (A4 horizontal) con los totales por estado.
func PDFDetalle(r *Resumen, ahora time.Time) ([]byte, error) {
	d, err := nuevoDocumento(gopdf.PageSizeA4Landscape, "Detalle de gastos", r, ahora)
	if err != nil {
		return nil, err
	}
	if len(r.Detalle) == 0 {
		d.parrafoVacio()
		return d.finalizar()
	}
	columnas := []columna{
		{"N°", 40, gopdf.Left}, {"Fecha", 62, gopdf.Left}, {"Cliente", 175, gopdf.Left}, {"Viaje", 45, gopdf.Right},
		{"Unidad", 70, gopdf.Left}, {"Conductor", 165, gopdf.Left}, {"Estado", 80, gopdf.Left},
		{"Monto", d.anchoUtil() - 637, gopdf.Right},
	}
	d.encabezadoTabla(columnas)
	d.alNuevaPagina = func() { d.encabezadoTabla(columnas) }
	for n, g := range r.Detalle {
		d.asegurar(altoFila)
		if n%2 == 1 {
			d.rect(margen, d.y, d.anchoUtil(), altoFila, colorPanel2)
		}
		d.fila(columnas, []string{
			strconv.Itoa(g.ID), FormatoFecha(g.Fecha), g.Cliente, strconv.Itoa(g.ViajeID),
			g.Unidad, g.Conductor, g.Estado, FormatoMonto(g.Monto),
		}, false)
	}
	d.alNuevaPagina = nil

	d.asegurar(altoFila*float64(len(r.PorEstado)+1) + 10)
	d.y += 6
	d.linea(margen, d.y, d.ancho-margen, d.y, colorLinea)
	totales := []columna{{"", d.anchoUtil() - 160, gopdf.Right}, {"", 160, gopdf.Right}}
	for _, e := range r.PorEstado {
		d.fila(totales, []string{"Total " + e.Estado + " (" + strconv.Itoa(e.Cantidad) + " gastos)", FormatoMonto(e.Monto)}, false)
	}
	d.fila(totales, []string{"Total general (" + strconv.Itoa(r.Indicadores.Cantidad) + " gastos)", FormatoMonto(r.Indicadores.Total)}, true)
	return d.finalizar()
}
```

- [ ] **Step 6: Ejecutar y verificar que pasan**

Run: `cd gastos-service && go test ./internal/reportes/ -count=1 -run 'PDF|Escala' -v 2>&1 | grep -E "^(--- |ok|FAIL)"`
Expected: 5 PASS (estadístico en 2 páginas, 24 meses en 2 páginas, detalle de 60 gastos en 3 páginas, vacíos en 1 página, escala del eje).

- [ ] **Step 7: Mirar un PDF de muestra**

Agrega temporalmente, al final de `TestPDFEstadisticoConDatosOcupaDosPaginas`, la línea `_ = os.WriteFile("/private/tmp/claude-501/-Users-helcorp-Downloads-Proyectos-Universidad-Proyecto-Campstone/00d73130-eac8-4a38-b8ba-011054089b50/scratchpad/muestra-reporte.pdf", pdf, 0o644)` (con `"os"` en los imports) y ejecuta:

```bash
cd gastos-service && go test ./internal/reportes/ -count=1 -run TestPDFEstadisticoConDatosOcupaDosPaginas
sips -s format png /private/tmp/claude-501/-Users-helcorp-Downloads-Proyectos-Universidad-Proyecto-Campstone/00d73130-eac8-4a38-b8ba-011054089b50/scratchpad/muestra-reporte.pdf --out /private/tmp/claude-501/-Users-helcorp-Downloads-Proyectos-Universidad-Proyecto-Campstone/00d73130-eac8-4a38-b8ba-011054089b50/scratchpad/muestra-reporte.png
```

Mira la imagen: cabecera con banda roja, título y período; 4 recuadros; barras apiladas con eje y leyenda; tabla por estado con barras; primer ranking. Después **quita la línea temporal y el import**, borra `/private/tmp/claude-501/-Users-helcorp-Downloads-Proyectos-Universidad-Proyecto-Campstone/00d73130-eac8-4a38-b8ba-011054089b50/scratchpad/muestra-reporte.*` y vuelve a correr el Step 6.

- [ ] **Step 8: Checkpoint (sin commit)**

Run: `cd gastos-service && gofmt -l . && go vet ./... && go build ./...`
Expected: sin salida de gofmt; vet y build limpios.

---

### Task 4: Endpoints de reportes (Go)

**Files:**
- Create: `gastos-service/internal/handlers/reportes.go`
- Modify: `gastos-service/internal/handlers/handlers.go` (rutas), `gastos-service/internal/handlers/middleware.go` (CORS)
- Test: `gastos-service/internal/handlers/reportes_integration_test.go`

**Interfaces:**
- Consumes: `ParseFiltro`, `ErrFiltro`, `Consultar`, `PDFEstadistico`, `PDFDetalle` (Tareas 1–3); `nuevoEntorno`/`entorno` de `checklist_integration_test.go` (Tarea 2).
- Produces: `GET /reportes/resumen`, `GET /reportes/estadistico.pdf`, `GET /reportes/detalle.pdf` (contrato de la sección 5 de la spec); CORS con `Access-Control-Expose-Headers: Content-Disposition`.

- [ ] **Step 1: Escribir las pruebas que fallan**

`gastos-service/internal/handlers/reportes_integration_test.go`:

```go
package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	rec := e.get("/reportes/resumen?desde=2026-03-01&hasta=2026-03-31")
	if rec.Code != http.StatusOK {
		t.Fatalf("código %d: %s", rec.Code, rec.Body)
	}
	var r reportes.Resumen
	if err := json.Unmarshal(rec.Body.Bytes(), &r); err != nil {
		t.Fatal(err)
	}
	if r.Desde != "2026-03-01" || r.Indicadores.Total != 200 || r.Indicadores.PorcentajeSustentado != 60 || len(r.PorMes) != 1 || len(r.PorCliente) != 1 {
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
```

- [ ] **Step 2: Ejecutar y verificar que fallan**

Run: `cd gastos-service && TEST_DATABASE_URL="$(sed -n 's/^DATABASE_URL=//p' .env | tr -d "\"'")" go test ./internal/handlers/ -count=1 -run 'Reporte|Resumen|Descarga'`
Expected: FAIL: códigos 404 en lugar de 200/400 y sin `Access-Control-Expose-Headers`.

- [ ] **Step 3: Implementar los handlers**

`gastos-service/internal/handlers/reportes.go`:

```go
package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"gastos-service/internal/reportes"
)

// filtroReporte lee ?desde=&hasta=; si son inválidos, ya respondió 400.
func filtroReporte(w http.ResponseWriter, r *http.Request) (reportes.Filtro, bool) {
	f, err := reportes.ParseFiltro(r.URL.Query().Get("desde"), r.URL.Query().Get("hasta"), time.Now())
	var ef reportes.ErrFiltro
	if errors.As(err, &ef) {
		writeError(w, http.StatusBadRequest, ef.Mensaje)
		return f, false
	}
	return f, err == nil
}

// resumenReporte devuelve las estadísticas del dashboard gerencial.
func (h *Handler) resumenReporte(w http.ResponseWriter, r *http.Request) {
	f, ok := filtroReporte(w, r)
	if !ok {
		return
	}
	resumen, err := reportes.Consultar(r.Context(), h.DB, f, false)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo generar el reporte")
		return
	}
	writeJSON(w, http.StatusOK, resumen)
}

// pdfEstadistico descarga el reporte estadístico del rango.
func (h *Handler) pdfEstadistico(w http.ResponseWriter, r *http.Request) {
	h.enviarPDF(w, r, "reporte-estadistico", false, reportes.PDFEstadistico)
}

// pdfDetalle descarga el listado de gastos del rango.
func (h *Handler) pdfDetalle(w http.ResponseWriter, r *http.Request) {
	h.enviarPDF(w, r, "detalle-gastos", true, reportes.PDFDetalle)
}

func (h *Handler) enviarPDF(w http.ResponseWriter, r *http.Request, nombre string, conDetalle bool, generar func(*reportes.Resumen, time.Time) ([]byte, error)) {
	f, ok := filtroReporte(w, r)
	if !ok {
		return
	}
	resumen, err := reportes.Consultar(r.Context(), h.DB, f, conDetalle)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo generar el reporte")
		return
	}
	pdf, err := generar(resumen, time.Now())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo generar el reporte")
		return
	}
	archivo := nombre + "-" + f.DesdeISO() + "_" + f.HastaISO() + ".pdf"
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="`+archivo+`"`)
	w.Header().Set("Content-Length", strconv.Itoa(len(pdf)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdf)
}
```

En `gastos-service/internal/handlers/handlers.go`, dentro de `Register`, inserta antes de `	// Archivos de comprobantes servidos estáticamente`:

```go
	// Reportes estadísticos (dashboard gerencial y PDF)
	mux.HandleFunc("GET /reportes/resumen", h.resumenReporte)
	mux.HandleFunc("GET /reportes/estadistico.pdf", h.pdfEstadistico)
	mux.HandleFunc("GET /reportes/detalle.pdf", h.pdfDetalle)

```

En `gastos-service/internal/handlers/middleware.go`, después de la línea `w.Header().Set("Access-Control-Allow-Headers", …)`, agrega:

```go
		// El frontend lee el nombre del PDF descargado
		w.Header().Set("Access-Control-Expose-Headers", "Content-Disposition")
```

- [ ] **Step 4: Ejecutar y verificar que pasan**

Run: `cd gastos-service && TEST_DATABASE_URL="$(sed -n 's/^DATABASE_URL=//p' .env | tr -d "\"'")" go test ./... -count=1 -v 2>&1 | grep -cE "^--- PASS"`
Expected: `37` (checklist 7 + 7, reportes 20, endpoints 3), sin FAIL ni SKIP.

- [ ] **Step 5: Checkpoint (sin commit)**

Run: `cd gastos-service && gofmt -l . && go vet ./... && go build ./...`
Expected: limpio.

---

### Task 5: Datos de demostración (SQL)

**Files:**
- Create: `gastos-service/demo/reportes_demo_limpiar.sql`, `gastos-service/demo/reportes_demo.sql`

**Interfaces:**
- Consumes: el esquema de `db.sql` y las migraciones.
- Produces: 4 clientes, 5 tractos y 5 carretas, 5 conductores, 24 viajes y 150 gastos DEMO en los 12 meses anteriores a hoy; los mismos datos en cada ejecución.

- [ ] **Step 1: Escribir los scripts**

`gastos-service/demo/reportes_demo_limpiar.sql`:

```sql
-- Borra los datos de demostración de los reportes: todo lo marcado DEMO
-- (RUC 990000000xx, placas DEM-, DNI 990000xx, direcciones con referencia DEMO
-- y gastos creados por DEMO), más lo que se haya registrado sobre ellos.
--
--   psql "$DATABASE_URL" -f gastos-service/demo/reportes_demo_limpiar.sql

\set ON_ERROR_STOP on
BEGIN;

CREATE TEMP TABLE demo_viajes ON COMMIT DROP AS
SELECT id_viaje
FROM programacionviaje
WHERE id_cliente IN (SELECT id_cliente FROM cliente WHERE ruc LIKE '990000000%')
   OR id_tracto IN (SELECT id_unidad FROM unidad WHERE placa LIKE 'DEM-%')
   OR id_carreta IN (SELECT id_unidad FROM unidad WHERE placa LIKE 'DEM-%')
   OR id_conductor_unidad IN (
        SELECT cu.id_conductor_unidad
        FROM conductorunidad cu
        JOIN conductor d ON d.id_conductor = cu.id_conductor
        WHERE d.dni LIKE '990000%')
   OR id_direccion_origen IN (SELECT id_direccion FROM direccion WHERE referencia = 'DEMO')
   OR id_direccion_destino IN (SELECT id_direccion FROM direccion WHERE referencia = 'DEMO');

CREATE TEMP TABLE demo_gastos ON COMMIT DROP AS
SELECT id
FROM gasto
WHERE creado_por = 'DEMO'
   OR viaje_id IN (SELECT id_viaje FROM demo_viajes)
   OR cliente_id IN (SELECT id_cliente FROM cliente WHERE ruc LIKE '990000000%');

DELETE FROM checklist_comprobante
WHERE comprobante_id IN (SELECT id FROM comprobante WHERE gasto_id IN (SELECT id FROM demo_gastos));
DELETE FROM comprobante WHERE gasto_id IN (SELECT id FROM demo_gastos);
DELETE FROM gasto WHERE id IN (SELECT id FROM demo_gastos);

DELETE FROM detallepresupuesto
WHERE id_presupuesto IN (SELECT id_presupuesto FROM presupuestoviaje WHERE id_viaje IN (SELECT id_viaje FROM demo_viajes));
DELETE FROM presupuestoviaje WHERE id_viaje IN (SELECT id_viaje FROM demo_viajes);
DELETE FROM factura
WHERE id_viaje_realizado IN (
    SELECT vr.id_viaje_realizado
    FROM viajerealizado vr
    JOIN guiatransportista g ON g.id_guia_transportista = vr.id_guia_transportista
    WHERE g.id_viaje IN (SELECT id_viaje FROM demo_viajes));
DELETE FROM viajerealizado
WHERE id_guia_transportista IN (SELECT id_guia_transportista FROM guiatransportista WHERE id_viaje IN (SELECT id_viaje FROM demo_viajes));
DELETE FROM productotransportado
WHERE id_guia_transportista IN (SELECT id_guia_transportista FROM guiatransportista WHERE id_viaje IN (SELECT id_viaje FROM demo_viajes));
DELETE FROM guiatransportista WHERE id_viaje IN (SELECT id_viaje FROM demo_viajes);
DELETE FROM programacionviaje WHERE id_viaje IN (SELECT id_viaje FROM demo_viajes);

DELETE FROM conductorunidad
WHERE id_conductor IN (SELECT id_conductor FROM conductor WHERE dni LIKE '990000%')
   OR id_unidad IN (SELECT id_unidad FROM unidad WHERE placa LIKE 'DEM-%');
DELETE FROM conductor WHERE dni LIKE '990000%';
DELETE FROM unidad WHERE placa LIKE 'DEM-%';
DELETE FROM cliente WHERE ruc LIKE '990000000%';
DELETE FROM direccion WHERE referencia = 'DEMO';

COMMIT;
```

`gastos-service/demo/reportes_demo.sql`:

```sql
-- Datos de demostración para los reportes estadísticos (Cuarto Informe de Estado).
--
-- Crea 4 clientes, 5 tractos con sus carretas, 5 conductores con sus asignaciones,
-- 24 viajes y unos 150 gastos repartidos en los 12 meses anteriores a hoy, todos
-- marcados como DEMO. Los gastos no tienen comprobantes adjuntos: sirven para los
-- reportes. Primero borra los datos DEMO que ya existan, así que se puede repetir.
-- Necesita los catálogos de db.sql (rubro, tipounidad, marca, ubigeo).
--
--   psql "$DATABASE_URL" -f gastos-service/demo/reportes_demo.sql
--
-- Para quitarlos: reportes_demo_limpiar.sql. No es una migración: el servicio nunca lo aplica.

\set ON_ERROR_STOP on
\ir reportes_demo_limpiar.sql

BEGIN;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM rubro) OR NOT EXISTS (SELECT 1 FROM tipounidad)
       OR NOT EXISTS (SELECT 1 FROM marca) OR NOT EXISTS (SELECT 1 FROM ubigeo) THEN
        RAISE EXCEPTION 'Faltan catálogos (rubro, tipounidad, marca, ubigeo): carga db.sql antes de los datos de demostración';
    END IF;
END $$;

-- Mismos datos en cada ejecución
SELECT setseed(0.17);

INSERT INTO direccion (codigo_ubigeo, direccion_detallada, referencia, descripcion, estado_direccion)
SELECT (SELECT min(codigo_ubigeo) FROM ubigeo), d, 'DEMO', 'DEMO', TRUE
FROM unnest(ARRAY[
    'Planta Chao, Panamericana Norte km 512',
    'Puerto de Paita, muelle 2',
    'Almacén Callao, Av. Néstor Gambetta 1200'
]) AS d;

INSERT INTO cliente (razon_social, ruc, contacto, id_rubro, estado_cliente)
SELECT nombre || ' (DEMO)', ruc, 'DEMO', (SELECT min(id_rubro) FROM rubro), TRUE
FROM (VALUES
    ('Agroexportadora Norte SAC', '99000000001'),
    ('Minera Andina del Sur SA', '99000000002'),
    ('Distribuidora Costa Verde SAC', '99000000003'),
    ('Pesquera Bahía Azul SAC', '99000000004')
) AS c (nombre, ruc);

-- Tractos DEM-101..105 y carretas DEM-201..205
INSERT INTO unidad (placa, id_tipo_unidad, id_subtipo_unidad, id_marca, modelo, anio, estado_unidad)
SELECT 'DEM-' || (100 * t.grupo + n), t.id_tipo,
       (SELECT min(id_subtipo_unidad) FROM subtipounidad WHERE id_tipo_unidad = t.id_tipo),
       (SELECT min(id_marca) FROM marca), 'DEMO', 2020 + n % 4, TRUE
FROM generate_series(1, 5) AS n
CROSS JOIN (
    SELECT 1 AS grupo, COALESCE((SELECT min(id_tipo_unidad) FROM tipounidad WHERE nombre ILIKE 'tracto%'), (SELECT min(id_tipo_unidad) FROM tipounidad)) AS id_tipo
    UNION ALL
    SELECT 2, COALESCE((SELECT min(id_tipo_unidad) FROM tipounidad WHERE nombre ILIKE 'carreta%'), (SELECT max(id_tipo_unidad) FROM tipounidad))
) AS t;

INSERT INTO conductor (dni, nombre1, nombre2, apellido1, apellido2, licencia_conducir, estado_conductor)
SELECT '9900000' || n, nombre, 'DEMO', ap1, ap2, 'DEMO-000' || n, TRUE
FROM (VALUES
    (1, 'Carlos', 'Quispe', 'Rojas'),
    (2, 'María', 'Huamán', 'Torres'),
    (3, 'José', 'Ñique', 'Paredes'),
    (4, 'Luis', 'Castillo', 'Vega'),
    (5, 'Rosa', 'Mendoza', 'Alva')
) AS c (n, nombre, ap1, ap2);

-- Cada conductor maneja el tracto de su mismo número
INSERT INTO conductorunidad (id_conductor, id_unidad, fecha_asignacion, estado_asignacion)
SELECT d.id_conductor, u.id_unidad, CURRENT_DATE - interval '1 year', TRUE
FROM conductor d
JOIN unidad u ON u.placa = 'DEM-10' || right(d.dni, 1)
WHERE d.dni LIKE '990000%';

-- 24 viajes: dos por mes (días 3 y 17) en los últimos 12 meses
WITH ids AS (
    SELECT
        (SELECT array_agg(id_cliente ORDER BY ruc) FROM cliente WHERE ruc LIKE '990000000%') AS clientes,
        (SELECT array_agg(id_unidad ORDER BY placa) FROM unidad WHERE placa LIKE 'DEM-1%') AS tractos,
        (SELECT array_agg(id_unidad ORDER BY placa) FROM unidad WHERE placa LIKE 'DEM-2%') AS carretas,
        (SELECT array_agg(cu.id_conductor_unidad ORDER BY d.dni)
           FROM conductorunidad cu JOIN conductor d ON d.id_conductor = cu.id_conductor
          WHERE d.dni LIKE '990000%') AS asignaciones,
        (SELECT array_agg(id_direccion ORDER BY id_direccion) FROM direccion WHERE referencia = 'DEMO') AS direcciones
)
INSERT INTO programacionviaje (id_tracto, id_carreta, id_conductor_unidad, id_direccion_origen, id_direccion_destino,
                               fecha_programado, id_cliente, estado_viaje)
SELECT ids.tractos[n % 5 + 1], ids.carretas[n % 5 + 1], ids.asignaciones[n % 5 + 1],
       ids.direcciones[n % 3 + 1], ids.direcciones[(n + 1) % 3 + 1],
       date_trunc('month', CURRENT_DATE) - make_interval(months => n / 2) + make_interval(days => 2 + (n % 2) * 14),
       ids.clientes[n % 4 + 1], TRUE
FROM ids, generate_series(0, 23) AS n;

-- 150 gastos: los meses antiguos casi todos Sustentado; los recientes, con más Registrado y Pagado
WITH viajes AS (
    SELECT v.id_viaje, v.id_cliente, v.fecha_programado::date AS dia,
           row_number() OVER (ORDER BY v.fecha_programado, v.id_viaje) - 1 AS orden
    FROM programacionviaje v
    JOIN cliente c ON c.id_cliente = v.id_cliente
    WHERE c.ruc LIKE '990000000%'
),
sorteo AS (
    SELECT n, floor(random() * 24)::int AS orden, random() AS r_dia, random() AS r_estado, random() AS r_monto
    FROM generate_series(1, 150) AS n
),
gastos AS (
    SELECT v.id_viaje, v.id_cliente, LEAST(v.dia + floor(s.r_dia * 12)::int, CURRENT_DATE) AS fecha, s.r_estado, s.r_monto
    FROM sorteo s
    JOIN viajes v ON v.orden = s.orden
)
INSERT INTO gasto (viaje_id, cliente_id, monto, fecha, estado, creado_por, creado_en)
SELECT id_viaje, id_cliente,
       round((25 + r_monto * r_monto * 875)::numeric, 2),
       fecha,
       CASE
           WHEN fecha < date_trunc('month', CURRENT_DATE) - interval '2 months' THEN
               CASE WHEN r_estado < 0.85 THEN 'Sustentado' WHEN r_estado < 0.95 THEN 'Pagado' ELSE 'Registrado' END
           WHEN fecha < date_trunc('month', CURRENT_DATE) THEN
               CASE WHEN r_estado < 0.5 THEN 'Sustentado' WHEN r_estado < 0.8 THEN 'Pagado' ELSE 'Registrado' END
           ELSE
               CASE WHEN r_estado < 0.2 THEN 'Sustentado' WHEN r_estado < 0.55 THEN 'Pagado' ELSE 'Registrado' END
       END,
       'DEMO', fecha + time '09:00'
FROM gastos;

COMMIT;
```

- [ ] **Step 2: Probarlos en una base temporal (nunca en la real)**

Run:
```bash
cd gastos-service
BASE="$(sed -n 's/^DATABASE_URL=//p' .env | tr -d "\"'")"
TMP=$(python3 -c "import sys,urllib.parse as u; p=u.urlsplit(sys.argv[1]); print(u.urlunsplit(p._replace(path='/lizar_reportes_prueba')))" "$BASE")
psql "$BASE" -qc "DROP DATABASE IF EXISTS lizar_reportes_prueba" && psql "$BASE" -qc "CREATE DATABASE lizar_reportes_prueba"
psql "$TMP" -q -v ON_ERROR_STOP=1 -f ../db.sql > /dev/null
for m in migrations/*.sql; do psql "$TMP" -q -v ON_ERROR_STOP=1 -f "$m"; done
psql "$TMP" -q -f demo/reportes_demo.sql > /dev/null && psql "$TMP" -tAc "SELECT count(*), round(sum(monto)), count(DISTINCT to_char(fecha,'YYYY-MM')) FROM gasto WHERE creado_por='DEMO'"
psql "$TMP" -q -f demo/reportes_demo.sql > /dev/null && psql "$TMP" -tAc "SELECT count(*), round(sum(monto)) FROM gasto WHERE creado_por='DEMO'"
psql "$TMP" -q -f demo/reportes_demo_limpiar.sql && psql "$TMP" -tAc "SELECT (SELECT count(*) FROM gasto WHERE creado_por='DEMO') + (SELECT count(*) FROM cliente WHERE ruc LIKE '990000000%') + (SELECT count(*) FROM unidad WHERE placa LIKE 'DEM-%') + (SELECT count(*) FROM conductor WHERE dni LIKE '990000%') + (SELECT count(*) FROM direccion WHERE referencia='DEMO')"
psql "$BASE" -qc "DROP DATABASE lizar_reportes_prueba"
```
Expected: `150|<suma>|12`; la segunda carga da la misma suma (se puede repetir); la limpieza deja `0`; la base temporal se borra.

- [ ] **Step 3: Checkpoint (sin commit)**

Nada que formatear. Anota en el ledger la suma obtenida (sirve para el informe).

---

### Task 6: Modelo, servicio, descarga y filtro (frontend)

**Files:**
- Create: `frontend/src/app/core/models/reporte.model.ts`
- Modify: `frontend/src/app/core/services/gastos-api.service.ts` (método `getBlob`)
- Create: `frontend/src/app/core/services/reportes.service.ts` + `.spec.ts`
- Create: `frontend/src/app/core/utils/descarga.ts` + `.spec.ts`
- Create: `frontend/src/app/features/reportes/filtro-reporte.ts` + `.spec.ts`

**Interfaces:**
- Consumes: `GastosApiService.get`, `HTTP_TIMEOUT_MS`, `EstadoGasto`.
- Produces:
  - Tipos `FiltroReporte`, `TipoPdf = 'estadistico' | 'detalle'`, `ResumenReporte`, `IndicadoresReporte`, `MesReporte`, `EstadoReporte`, `FilaRanking`, `ArchivoPdf {blob, nombre}`.
  - `GastosApiService.getBlob(path): Observable<HttpResponse<Blob>>`.
  - `ReportesService.resumen(filtro)`, `ReportesService.descargar(tipo, filtro): Observable<ArchivoPdf>`, `nombreDeArchivo(disposicion, respaldo)`.
  - `guardarArchivo(blob, nombre, doc?)`.
  - `ATAJOS`, `Atajo`, `rangoDeAtajo`, `atajoActivo`, `filtroDesdeConsulta`, `fechaValida`, `errorDeFiltro`, `rangoDelMes`, `mesCompleto`, `nombreMes`, `MSG_FECHA_INVALIDA`, `MSG_RANGO_INVERTIDO`, `MSG_RANGO_LARGO`.

- [ ] **Step 1: Escribir las pruebas que fallan**

`frontend/src/app/core/services/reportes.service.spec.ts`:

```ts
import { describe, expect, it, vi } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { HttpHeaders, HttpResponse } from '@angular/common/http';
import { of } from 'rxjs';
import { ReportesService, nombreDeArchivo } from './reportes.service';
import { GastosApiService } from './gastos-api.service';

function montar(cabeceras: Record<string, string> = {}) {
  const pdf = new Blob(['%PDF-1.7'], { type: 'application/pdf' });
  const api = {
    get: vi.fn(() => of({ desde: '2026-01-01' })),
    getBlob: vi.fn(() => of(new HttpResponse({ body: pdf, headers: new HttpHeaders(cabeceras) }))),
  };
  TestBed.configureTestingModule({ providers: [{ provide: GastosApiService, useValue: api }] });
  return { servicio: TestBed.inject(ReportesService), api, pdf };
}

const filtro = { desde: '2025-11-01', hasta: '2026-10-17' };

describe('ReportesService', () => {
  it('pide el resumen del rango', () => {
    const { servicio, api } = montar();
    servicio.resumen(filtro).subscribe();
    expect(api.get).toHaveBeenCalledWith('/reportes/resumen?desde=2025-11-01&hasta=2026-10-17');
  });

  it('descarga el PDF con el nombre que manda el servidor', () => {
    const { servicio, api, pdf } = montar({ 'Content-Disposition': 'attachment; filename="reporte-estadistico-2025-11-01_2026-10-17.pdf"' });
    let archivo: { blob: Blob; nombre: string } | undefined;
    servicio.descargar('estadistico', filtro).subscribe(a => (archivo = a));
    expect(api.getBlob).toHaveBeenCalledWith('/reportes/estadistico.pdf?desde=2025-11-01&hasta=2026-10-17');
    expect(archivo).toEqual({ blob: pdf, nombre: 'reporte-estadistico-2025-11-01_2026-10-17.pdf' });
  });

  it('sin Content-Disposition, arma el nombre con el rango', () => {
    const { servicio } = montar();
    let nombre = '';
    servicio.descargar('detalle', filtro).subscribe(a => (nombre = a.nombre));
    expect(nombre).toBe('detalle-gastos-2025-11-01_2026-10-17.pdf');
  });

  it('nombreDeArchivo lee la cabecera', () => {
    expect(nombreDeArchivo('attachment; filename="a.pdf"', 'b.pdf')).toBe('a.pdf');
    expect(nombreDeArchivo(null, 'b.pdf')).toBe('b.pdf');
  });
});
```

`frontend/src/app/core/utils/descarga.spec.ts`:

```ts
import { afterEach, describe, expect, it, vi } from 'vitest';
import { guardarArchivo } from './descarga';

describe('guardarArchivo', () => {
  afterEach(() => vi.restoreAllMocks());

  it('crea un enlace temporal con el nombre del archivo y lo pulsa', () => {
    // jsdom no implementa URL.createObjectURL
    const crear = vi.fn(() => 'blob:prueba');
    const revocar = vi.fn();
    Object.assign(URL, { createObjectURL: crear, revokeObjectURL: revocar });
    vi.useFakeTimers();
    let pulsado: HTMLAnchorElement | undefined;
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (this: HTMLAnchorElement) {
      pulsado = this;
    });
    const blob = new Blob(['%PDF'], { type: 'application/pdf' });

    guardarArchivo(blob, 'reporte.pdf');

    expect(crear).toHaveBeenCalledWith(blob);
    expect(pulsado?.download).toBe('reporte.pdf');
    expect(pulsado?.href).toBe('blob:prueba');
    expect(document.querySelector('a[download]')).toBeNull();
    vi.runAllTimers();
    expect(revocar).toHaveBeenCalledWith('blob:prueba');
    vi.useRealTimers();
  });
});
```

`frontend/src/app/features/reportes/filtro-reporte.spec.ts`:

```ts
import { describe, expect, it } from 'vitest';
import {
  MSG_FECHA_INVALIDA, MSG_RANGO_INVERTIDO, MSG_RANGO_LARGO, atajoActivo, errorDeFiltro, fechaValida, filtroDesdeConsulta,
  mesCompleto, nombreMes, rangoDeAtajo, rangoDelMes,
} from './filtro-reporte';

const HOY = '2026-10-17';

describe('filtro de reportes', () => {
  it('los atajos calculan su rango desde hoy', () => {
    expect(rangoDeAtajo('ultimos-12', HOY)).toEqual({ desde: '2025-11-01', hasta: HOY });
    expect(rangoDeAtajo('este-anio', HOY)).toEqual({ desde: '2026-01-01', hasta: HOY });
    expect(rangoDeAtajo('este-mes', HOY)).toEqual({ desde: '2026-10-01', hasta: HOY });
    expect(rangoDeAtajo('ultimos-12', '2026-01-05')).toEqual({ desde: '2025-02-01', hasta: '2026-01-05' });
  });

  it('reconoce el atajo activo', () => {
    expect(atajoActivo({ desde: '2026-10-01', hasta: HOY }, HOY)).toBe('este-mes');
    expect(atajoActivo({ desde: '2026-03-01', hasta: '2026-03-31' }, HOY)).toBeNull();
  });

  it('sin parámetros usa los mismos valores por defecto que el servidor', () => {
    expect(filtroDesdeConsulta(null, null, HOY)).toEqual({ desde: '2025-11-01', hasta: HOY });
    expect(filtroDesdeConsulta(undefined, '2026-03-15', HOY)).toEqual({ desde: '2025-04-01', hasta: '2026-03-15' });
    expect(filtroDesdeConsulta('2026-01-01', null, HOY)).toEqual({ desde: '2026-01-01', hasta: HOY });
  });

  it('valida fechas reales', () => {
    expect(fechaValida('2026-02-28')).toBe(true);
    expect(fechaValida('2026-02-30')).toBe(false);
    expect(fechaValida('17/10/2026')).toBe(false);
  });

  it('los errores coinciden con gastos-service, incluido el límite de 731 días', () => {
    expect(errorDeFiltro({ desde: '2026-13-01', hasta: HOY })).toBe(MSG_FECHA_INVALIDA);
    expect(errorDeFiltro({ desde: '2026-05-02', hasta: '2026-05-01' })).toBe(MSG_RANGO_INVERTIDO);
    expect(errorDeFiltro({ desde: '2024-01-01', hasta: '2025-12-31' })).toBeNull();
    expect(errorDeFiltro({ desde: '2024-01-01', hasta: '2026-01-01' })).toBe(MSG_RANGO_LARGO);
  });

  it('el rango de un mes va del día 1 al último día', () => {
    expect(rangoDelMes('2026-02')).toEqual({ desde: '2026-02-01', hasta: '2026-02-28' });
    expect(rangoDelMes('2025-12')).toEqual({ desde: '2025-12-01', hasta: '2025-12-31' });
    expect(mesCompleto({ desde: '2026-02-01', hasta: '2026-02-28' })).toBe('2026-02');
    expect(mesCompleto({ desde: '2026-02-01', hasta: '2026-02-27' })).toBeNull();
    expect(nombreMes('2026-10')).toBe('octubre 2026');
  });
});
```

- [ ] **Step 2: Ejecutar y verificar que fallan**

Run: `cd frontend && npx ng test --watch=false --include src/app/core/services/reportes.service.spec.ts --include src/app/core/utils/descarga.spec.ts --include src/app/features/reportes/filtro-reporte.spec.ts`
Expected: FAIL: no se pueden resolver `./reportes.service`, `./descarga`, `./filtro-reporte`.

- [ ] **Step 3: Implementar**

`frontend/src/app/core/models/reporte.model.ts`:

```ts
import { EstadoGasto } from './gasto.model';

/** Rango de fechas del gasto (AAAA-MM-DD), con ambos extremos incluidos. */
export interface FiltroReporte {
  desde: string;
  hasta: string;
}

export type TipoPdf = 'estadistico' | 'detalle';

export interface IndicadoresReporte {
  total: number;
  cantidad: number;
  promedio: number;
  sustentado: number;
  pendiente: number;
  porcentaje_sustentado: number;
}

/** Gasto de un mes (AAAA-MM) separado por estado. */
export interface MesReporte {
  mes: string;
  registrado: number;
  pagado: number;
  sustentado: number;
  total: number;
}

export interface EstadoReporte {
  estado: EstadoGasto;
  cantidad: number;
  monto: number;
  porcentaje: number;
}

export interface FilaRanking {
  puesto: number;
  etiqueta: string;
  cantidad: number;
  monto: number;
  porcentaje: number;
}

/** Respuesta de GET /reportes/resumen (gastos-service). */
export interface ResumenReporte {
  desde: string;
  hasta: string;
  indicadores: IndicadoresReporte;
  por_mes: MesReporte[];
  por_estado: EstadoReporte[];
  por_cliente: FilaRanking[];
  por_unidad: FilaRanking[];
  por_conductor: FilaRanking[];
}

export interface ArchivoPdf {
  blob: Blob;
  nombre: string;
}
```

En `frontend/src/app/core/services/gastos-api.service.ts`, cambia el import de `@angular/common/http` a:

```ts
import { HttpClient, HttpHeaders, HttpResponse } from '@angular/common/http';
```

y agrega, justo antes de `post<T>(path: string, body: unknown)`:

```ts
  /** Descarga un archivo con sus cabeceras (para leer el nombre en Content-Disposition). */
  getBlob(path: string): Observable<HttpResponse<Blob>> {
    return this.http.get(`${this._base()}${path}`, { headers: this.headers(), responseType: 'blob', observe: 'response' });
  }

```

`frontend/src/app/core/services/reportes.service.ts`:

```ts
import { Injectable, inject } from '@angular/core';
import { Observable, map, timeout } from 'rxjs';
import { ArchivoPdf, FiltroReporte, ResumenReporte, TipoPdf } from '../models/reporte.model';
import { HTTP_TIMEOUT_MS } from '../utils/error-messages';
import { GastosApiService } from './gastos-api.service';

const PREFIJO_PDF: Record<TipoPdf, string> = { estadistico: 'reporte-estadistico', detalle: 'detalle-gastos' };

/** Lee el nombre de archivo de «attachment; filename="…"»; si no viene, usa el respaldo. */
export function nombreDeArchivo(disposicion: string | null, respaldo: string): string {
  return /filename="([^"]+)"/.exec(disposicion ?? '')?.[1] ?? respaldo;
}

function consulta(filtro: FiltroReporte): string {
  return `desde=${filtro.desde}&hasta=${filtro.hasta}`;
}

/** Estadísticas y PDF de gastos-service para el dashboard gerencial. */
@Injectable({ providedIn: 'root' })
export class ReportesService {
  private readonly api = inject(GastosApiService);

  resumen(filtro: FiltroReporte): Observable<ResumenReporte> {
    return this.api.get<ResumenReporte>(`/reportes/resumen?${consulta(filtro)}`).pipe(timeout(HTTP_TIMEOUT_MS));
  }

  descargar(tipo: TipoPdf, filtro: FiltroReporte): Observable<ArchivoPdf> {
    const respaldo = `${PREFIJO_PDF[tipo]}-${filtro.desde}_${filtro.hasta}.pdf`;
    return this.api.getBlob(`/reportes/${tipo}.pdf?${consulta(filtro)}`).pipe(
      timeout(HTTP_TIMEOUT_MS),
      map(res => ({ blob: res.body ?? new Blob([], { type: 'application/pdf' }), nombre: nombreDeArchivo(res.headers.get('Content-Disposition'), respaldo) })),
    );
  }
}
```

`frontend/src/app/core/utils/descarga.ts`:

```ts
/** Hace que el navegador guarde un archivo recibido como Blob con el nombre indicado. */
export function guardarArchivo(blob: Blob, nombre: string, doc: Document = document): void {
  const url = URL.createObjectURL(blob);
  const enlace = doc.createElement('a');
  enlace.href = url;
  enlace.download = nombre;
  enlace.style.display = 'none';
  doc.body.appendChild(enlace);
  enlace.click();
  enlace.remove();
  // Firefox necesita que la URL viva hasta después del clic
  setTimeout(() => URL.revokeObjectURL(url), 0);
}
```

`frontend/src/app/features/reportes/filtro-reporte.ts`:

```ts
import { FiltroReporte } from '../../core/models/reporte.model';

/** Atajos de rango del dashboard; el primero es el rango por defecto. */
export type Atajo = 'ultimos-12' | 'este-anio' | 'este-mes';

export const ATAJOS: readonly { id: Atajo; etiqueta: string }[] = [
  { id: 'ultimos-12', etiqueta: 'Últimos 12 meses' },
  { id: 'este-anio', etiqueta: 'Este año' },
  { id: 'este-mes', etiqueta: 'Este mes' },
];

/** Los mismos mensajes que devuelve gastos-service (sección 3 de la especificación). */
export const MSG_FECHA_INVALIDA = 'Fecha inválida: use AAAA-MM-DD.';
export const MSG_RANGO_INVERTIDO = 'La fecha inicial no puede ser posterior a la final.';
export const MSG_RANGO_LARGO = 'El rango no puede superar 24 meses.';

const RANGO_MAXIMO_DIAS = 731;
const ISO = /^(\d{4})-(\d{2})-(\d{2})$/;
const MESES = ['enero', 'febrero', 'marzo', 'abril', 'mayo', 'junio', 'julio', 'agosto', 'septiembre', 'octubre', 'noviembre', 'diciembre'];

/** AAAA-MM-DD de un día; Date.UTC corrige meses fuera de rango (mes 0 = diciembre del año anterior). */
function iso(anio: number, mes: number, dia: number): string {
  return new Date(Date.UTC(anio, mes - 1, dia)).toISOString().slice(0, 10);
}

function anioMes(fecha: string): [number, number] {
  const [anio, mes] = fecha.split('-').map(Number);
  return [anio, mes];
}

export function rangoDeAtajo(atajo: Atajo, hoy: string): FiltroReporte {
  const [anio, mes] = anioMes(hoy);
  switch (atajo) {
    case 'ultimos-12':
      return { desde: iso(anio, mes - 11, 1), hasta: hoy };
    case 'este-anio':
      return { desde: iso(anio, 1, 1), hasta: hoy };
    case 'este-mes':
      return { desde: iso(anio, mes, 1), hasta: hoy };
  }
}

export function atajoActivo(filtro: FiltroReporte, hoy: string): Atajo | null {
  return ATAJOS.find(a => {
    const r = rangoDeAtajo(a.id, hoy);
    return r.desde === filtro.desde && r.hasta === filtro.hasta;
  })?.id ?? null;
}

/** El rango de la URL, con los mismos valores por defecto que gastos-service. */
export function filtroDesdeConsulta(desde: string | null | undefined, hasta: string | null | undefined, hoy: string): FiltroReporte {
  const fin = hasta || hoy;
  const [anio, mes] = anioMes(fechaValida(fin) ? fin : hoy);
  return { desde: desde || iso(anio, mes - 11, 1), hasta: fin };
}

export function fechaValida(fecha: string): boolean {
  const m = ISO.exec(fecha);
  return !!m && iso(+m[1], +m[2], +m[3]) === fecha;
}

/** Mensaje de error del rango, o null si es válido. */
export function errorDeFiltro(filtro: FiltroReporte): string | null {
  if (!fechaValida(filtro.desde) || !fechaValida(filtro.hasta)) return MSG_FECHA_INVALIDA;
  if (filtro.desde > filtro.hasta) return MSG_RANGO_INVERTIDO;
  const dias = (Date.parse(filtro.hasta) - Date.parse(filtro.desde)) / 86_400_000 + 1;
  return dias > RANGO_MAXIMO_DIAS ? MSG_RANGO_LARGO : null;
}

/** El mes completo AAAA-MM: del día 1 al último día. */
export function rangoDelMes(mes: string): FiltroReporte {
  const [anio, m] = anioMes(mes);
  return { desde: iso(anio, m, 1), hasta: iso(anio, m + 1, 0) };
}

/** AAAA-MM si el rango es exactamente un mes completo; si no, null. */
export function mesCompleto(filtro: FiltroReporte): string | null {
  const mes = filtro.desde.slice(0, 7);
  const r = rangoDelMes(mes);
  return r.desde === filtro.desde && r.hasta === filtro.hasta ? mes : null;
}

/** «octubre 2026» */
export function nombreMes(mes: string): string {
  const [anio, m] = anioMes(mes);
  return `${MESES[m - 1]} ${anio}`;
}
```

- [ ] **Step 4: Ejecutar y verificar que pasan**

Run: el mismo comando del Step 2.
Expected: PASS, 11 pruebas (servicio 4, descarga 1, filtro 6).

- [ ] **Step 5: Checkpoint (sin commit)**

Run: `cd frontend && npx ng test --watch=false`
Expected: PASS en toda la suite (146 + 11 = 157).

---

### Task 7: Configuración de los gráficos y envoltorio de Chart.js (frontend)

**Files:**
- Modify: `frontend/package.json`, `frontend/package-lock.json` (`chart.js` 4.5.1 exacta)
- Create: `frontend/src/app/features/reportes/graficos.ts` + `.spec.ts`
- Create: `frontend/src/app/features/reportes/grafico/chart-js.ts`
- Create: `frontend/src/app/features/reportes/grafico/grafico.component.ts` + `.spec.ts`

**Interfaces:**
- Consumes: tipos de la Tarea 6, `nombreMes` (Tarea 6), `formatMonto`.
- Produces:
  - `ConfigGrafico = ChartConfiguration<'bar'>`, `COLOR_ESTADO`, `ESTADOS`, `mesCorto`, `montoEje`, `montoCompacto`, `porcentaje`, `gastos`, `recortar` (20 caracteres), `configPorMes(meses, alElegirMes)`, `configPorEstado(estados)`, `configRanking(filas)`, `descripcionPorMes`, `descripcionPorEstado`, `descripcionRanking(titulo, filas)`.
  - `InstanciaGrafico {actualizar, destruir}`, `FabricaGrafico`, `crearGraficoChartJs`, `FABRICA_GRAFICO` (token con fábrica por defecto), `GraficoComponent` (`app-grafico`, inputs `config`, `descripcion`, `alto`).

- [ ] **Step 1: Instalar Chart.js**

Run: `cd frontend && npm install --save-exact --no-audit --no-fund chart.js@4.5.1 && node -p "require('./package.json').dependencies['chart.js']"`
Expected: `4.5.1`.

- [ ] **Step 2: Escribir las pruebas que fallan**

`frontend/src/app/features/reportes/graficos.spec.ts`:

```ts
import { describe, expect, it, vi } from 'vitest';
import type { ActiveElement, ChartEvent, TooltipItem } from 'chart.js';
import { EstadoReporte, FilaRanking, MesReporte } from '../../core/models/reporte.model';
import {
  COLOR_ESTADO, configPorEstado, configPorMes, configRanking, descripcionPorEstado, descripcionPorMes, descripcionRanking,
  mesCorto, montoCompacto, montoEje, recortar,
} from './graficos';

const meses: MesReporte[] = [
  { mes: '2025-11', registrado: 100, pagado: 200, sustentado: 300, total: 600 },
  { mes: '2025-12', registrado: 0, pagado: 0, sustentado: 0, total: 0 },
  { mes: '2026-01', registrado: 50.5, pagado: 0, sustentado: 949.5, total: 1000 },
];
const estados: EstadoReporte[] = [
  { estado: 'Registrado', cantidad: 2, monto: 150.5, porcentaje: 9.4 },
  { estado: 'Pagado', cantidad: 1, monto: 200, porcentaje: 12.5 },
  { estado: 'Sustentado', cantidad: 9, monto: 1249.5, porcentaje: 78.1 },
];
const filas: FilaRanking[] = [
  { puesto: 1, etiqueta: 'Agroexportadora Norte del Perú SAC', cantidad: 7, monto: 900, porcentaje: 56.3 },
  { puesto: 2, etiqueta: 'Minera Sur SA', cantidad: 1, monto: 700, porcentaje: 43.7 },
];

// Los callbacks de Chart.js solo leen dataIndex, parsed y dataset de cada elemento
const item = (dataIndex: number, extra: object = {}) => ({ dataIndex, ...extra }) as unknown as TooltipItem<'bar'>;

describe('configuración de los gráficos', () => {
  it('gasto por mes: una serie por estado, en orden y con su color', () => {
    const c = configPorMes(meses, () => {});
    expect(c.data.labels).toEqual(['nov 25', 'dic 25', 'ene 26']);
    expect(c.data.datasets.map(d => d.label)).toEqual(['Registrado', 'Pagado', 'Sustentado']);
    expect(c.data.datasets.map(d => d.backgroundColor)).toEqual([COLOR_ESTADO.Registrado, COLOR_ESTADO.Pagado, COLOR_ESTADO.Sustentado]);
    expect(c.data.datasets[2].data).toEqual([300, 0, 949.5]);
    expect(c.options?.scales?.['x']?.stacked).toBe(true);
  });

  it('gasto por mes: el tooltip da el monto por estado y el total del mes', () => {
    const tooltip = configPorMes(meses, () => {}).options!.plugins!.tooltip!.callbacks!;
    const items = [item(2, { dataset: { label: 'Sustentado' }, parsed: { y: 949.5 } })];
    expect((tooltip.title as (i: TooltipItem<'bar'>[]) => string)(items)).toBe('enero 2026');
    expect((tooltip.label as (i: TooltipItem<'bar'>) => string)(items[0])).toBe('Sustentado: S/ 949.50');
    expect((tooltip.footer as (i: TooltipItem<'bar'>[]) => string)(items)).toBe('Total: S/ 1,000.00');
  });

  it('gasto por mes: un clic en una columna elige ese mes; en el vacío no hace nada', () => {
    const elegir = vi.fn();
    const onClick = configPorMes(meses, elegir).options!.onClick as (e: ChartEvent, el: ActiveElement[]) => void;
    onClick({} as ChartEvent, [{ index: 1 } as ActiveElement]);
    onClick({} as ChartEvent, []);
    expect(elegir).toHaveBeenCalledTimes(1);
    expect(elegir).toHaveBeenCalledWith('2025-12');
  });

  it('distribución por estado: barras horizontales con el color de cada estado', () => {
    const c = configPorEstado(estados);
    expect(c.options?.indexAxis).toBe('y');
    expect(c.data.datasets[0].backgroundColor).toEqual([COLOR_ESTADO.Registrado, COLOR_ESTADO.Pagado, COLOR_ESTADO.Sustentado]);
    const label = c.options!.plugins!.tooltip!.callbacks!.label as (i: TooltipItem<'bar'>) => string;
    expect(label(item(0, { parsed: { x: 150.5 } }))).toBe('S/ 150.50 · 2 gastos · 9.4 %');
  });

  it('ranking: recorta nombres largos en el eje y los muestra completos en el tooltip', () => {
    const c = configRanking(filas);
    expect(c.data.labels).toEqual(['Agroexportadora Nor…', 'Minera Sur SA']);
    const title = c.options!.plugins!.tooltip!.callbacks!.title as (i: TooltipItem<'bar'>[]) => string;
    const label = c.options!.plugins!.tooltip!.callbacks!.label as (i: TooltipItem<'bar'>) => string;
    expect(title([item(0)])).toBe('1. Agroexportadora Norte del Perú SAC');
    expect(label(item(1, { parsed: { x: 700 } }))).toBe('S/ 700.00 · 1 gasto · 43.7 %');
    expect(c.options?.plugins?.legend?.display).toBe(false);
  });

  it('24 meses siguen teniendo una etiqueta por mes', () => {
    const largos = Array.from({ length: 24 }, (_, i) => ({ mes: `${2025 + Math.floor(i / 12)}-${String((i % 12) + 1).padStart(2, '0')}`, registrado: 1, pagado: 1, sustentado: 1, total: 3 }));
    expect(configPorMes(largos, () => {}).data.labels).toHaveLength(24);
  });

  it('formatos del eje y etiquetas', () => {
    expect(mesCorto('2026-10')).toBe('oct 26');
    expect(montoEje(2000)).toBe('S/ 2,000');
    expect(montoEje(2500.5)).toBe('S/ 2,500.50');
    expect(recortar('corto')).toBe('corto');
    expect(montoCompacto(40000)).toBe('S/ 40 mil');
    expect(montoCompacto(2500)).toBe('S/ 2.5 mil');
    expect(montoCompacto(800)).toBe('S/ 800');
  });

  it('describe cada gráfico para los lectores de pantalla', () => {
    expect(descripcionPorMes(meses)).toBe('Gasto por mes de noviembre 2025 a enero 2026. El mes con más gasto fue enero 2026, con S/ 1,000.00.');
    expect(descripcionPorEstado(estados)).toBe('Distribución por estado: Registrado 9.4 %, Pagado 12.5 %, Sustentado 78.1 %.');
    expect(descripcionRanking('Gasto por cliente', filas)).toBe('Gasto por cliente: primero Agroexportadora Norte del Perú SAC, con S/ 900.00 (56.3 % del total).');
    expect(descripcionRanking('Gasto por unidad', [])).toBe('Gasto por unidad: sin datos.');
  });
});
```

`frontend/src/app/features/reportes/grafico/grafico.component.spec.ts`:

```ts
import { describe, expect, it, vi } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { FABRICA_GRAFICO, GraficoComponent } from './grafico.component';
import type { ConfigGrafico } from '../graficos';

const config = (n: number): ConfigGrafico => ({ type: 'bar', data: { labels: ['a'], datasets: [{ data: [n] }] } });

function montar() {
  const instancia = { actualizar: vi.fn(), destruir: vi.fn() };
  const fabrica = vi.fn(() => instancia);
  TestBed.configureTestingModule({ providers: [{ provide: FABRICA_GRAFICO, useValue: fabrica }] });
  const f = TestBed.createComponent(GraficoComponent);
  f.componentRef.setInput('config', config(1));
  f.componentRef.setInput('descripcion', 'Gasto por mes de enero a marzo.');
  f.componentRef.setInput('alto', 300);
  f.detectChanges();
  return { f, fabrica, instancia };
}

describe('GraficoComponent', () => {
  it('crea el gráfico sobre su lienzo, con su descripción accesible y su alto', () => {
    const { f, fabrica } = montar();
    const lienzo: HTMLCanvasElement = f.nativeElement.querySelector('canvas');
    expect(fabrica).toHaveBeenCalledWith(lienzo, config(1));
    expect(lienzo.getAttribute('role')).toBe('img');
    expect(lienzo.getAttribute('aria-label')).toBe('Gasto por mes de enero a marzo.');
    expect((f.nativeElement.querySelector('.grafico') as HTMLElement).style.height).toBe('300px');
  });

  it('con otra configuración actualiza el mismo gráfico en vez de crear otro', () => {
    const { f, fabrica, instancia } = montar();
    f.componentRef.setInput('config', config(2));
    f.detectChanges();
    expect(fabrica).toHaveBeenCalledTimes(1);
    expect(instancia.actualizar).toHaveBeenCalledWith(config(2));
  });

  it('destruye el gráfico al salir', () => {
    const { f, instancia } = montar();
    f.destroy();
    expect(instancia.destruir).toHaveBeenCalled();
  });
});
```

- [ ] **Step 3: Ejecutar y verificar que fallan**

Run: `cd frontend && npx ng test --watch=false --include src/app/features/reportes/graficos.spec.ts --include src/app/features/reportes/grafico/grafico.component.spec.ts`
Expected: FAIL: no se pueden resolver `./graficos` y `./grafico.component`.

- [ ] **Step 4: Implementar**

`frontend/src/app/features/reportes/graficos.ts`:

```ts
import type { ChartConfiguration, TooltipItem } from 'chart.js';
import { EstadoGasto } from '../../core/models/gasto.model';
import { EstadoReporte, FilaRanking, MesReporte } from '../../core/models/reporte.model';
import { formatMonto } from '../../core/utils/format';
import { nombreMes } from './filtro-reporte';

export type ConfigGrafico = ChartConfiguration<'bar'>;

/**
 * Colores de estado de la app (--registrado, --pagado, --sustentado en styles/tokens.css).
 * El gris de Registrado tiene poco croma a propósito: por eso la leyenda, los huecos
 * entre segmentos, los tooltips y la tabla «Ver datos» nombran siempre el estado.
 */
export const COLOR_ESTADO: Record<EstadoGasto, string> = {
  Registrado: '#8a9096',
  Pagado: '#2563eb',
  Sustentado: '#15803d',
};

export const ESTADOS: readonly EstadoGasto[] = ['Registrado', 'Pagado', 'Sustentado'];

const COLOR_RANKING = '#4a5056'; // --tinta-2: una sola serie, un solo tono
const COLOR_LINEA = '#e2e3e0'; // --linea: rejilla discreta
const COLOR_SUPERFICIE = '#fdfdfc'; // --panel: huecos de 2 px entre segmentos
const LARGO_ETIQUETA = 20;
const MESES_CORTOS = ['ene', 'feb', 'mar', 'abr', 'may', 'jun', 'jul', 'ago', 'sep', 'oct', 'nov', 'dic'];

/** «oct 26» */
export function mesCorto(mes: string): string {
  const [anio, m] = mes.split('-');
  return `${MESES_CORTOS[Number(m) - 1]} ${anio.slice(2)}`;
}

/** Monto del eje sin centavos: «S/ 2,000». */
export function montoEje(valor: number | string): string {
  return formatMonto(valor).replace(/\.00$/, '');
}

/** Monto corto para ejes angostos: «S/ 40 mil», «S/ 2.5 mil», «S/ 800». */
export function montoCompacto(valor: number | string): string {
  const n = Number(valor);
  if (Math.abs(n) < 1000) return `S/ ${Math.round(n)}`;
  return `S/ ${(n / 1000).toFixed(1).replace(/\.0$/, '')} mil`;
}

export function porcentaje(valor: number): string {
  return `${valor.toFixed(1)} %`;
}

export function gastos(cantidad: number): string {
  return `${cantidad} ${cantidad === 1 ? 'gasto' : 'gastos'}`;
}

/** Recorta nombres largos para el eje; el nombre completo va en el tooltip y en la tabla. */
export function recortar(texto: string, largo = LARGO_ETIQUETA): string {
  return texto.length > largo ? `${texto.slice(0, largo - 1)}…` : texto;
}

function montoDe(mes: MesReporte, estado: EstadoGasto): number {
  return estado === 'Registrado' ? mes.registrado : estado === 'Pagado' ? mes.pagado : mes.sustentado;
}

/** Gasto por mes, apilado por estado. Un clic en cualquier punto de la columna elige ese mes. */
export function configPorMes(meses: MesReporte[], alElegirMes: (mes: string) => void): ConfigGrafico {
  return {
    type: 'bar',
    data: {
      labels: meses.map(m => mesCorto(m.mes)),
      datasets: ESTADOS.map(estado => ({
        label: estado,
        data: meses.map(m => montoDe(m, estado)),
        backgroundColor: COLOR_ESTADO[estado],
        borderColor: COLOR_SUPERFICIE,
        borderWidth: { top: 2 },
        borderRadius: 4,
        maxBarThickness: 40,
      })),
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      interaction: { mode: 'index', intersect: false },
      scales: {
        x: { stacked: true, grid: { display: false }, ticks: { maxRotation: 0, autoSkip: true } },
        y: { stacked: true, beginAtZero: true, grid: { color: COLOR_LINEA }, border: { display: false }, ticks: { callback: v => montoEje(v) } },
      },
      plugins: {
        legend: { position: 'bottom', labels: { boxWidth: 12, boxHeight: 12, useBorderRadius: true, borderRadius: 2 } },
        tooltip: {
          callbacks: {
            title: items => nombreMes(meses[items[0]?.dataIndex ?? 0]?.mes ?? ''),
            label: (c: TooltipItem<'bar'>) => `${c.dataset.label}: ${formatMonto(c.parsed.y)}`,
            footer: items => `Total: ${formatMonto(meses[items[0]?.dataIndex ?? 0]?.total ?? 0)}`,
          },
        },
      },
      onClick: (_evento, elementos) => {
        const i = elementos[0]?.index;
        if (i !== undefined) alElegirMes(meses[i].mes);
      },
      onHover: (evento, elementos) => {
        const lienzo = evento.native?.target as HTMLElement | undefined;
        if (lienzo) lienzo.style.cursor = elementos.length ? 'pointer' : 'default';
      },
    },
  };
}

/** Barras horizontales comunes a la distribución por estado y a los rankings. */
function barrasHorizontales(etiquetas: string[], montos: number[], colores: string | string[], detalle: (i: number) => string, titulo: (i: number) => string): ConfigGrafico {
  return {
    type: 'bar',
    data: {
      labels: etiquetas,
      datasets: [{ data: montos, backgroundColor: colores, borderRadius: 4, maxBarThickness: 22 }],
    },
    options: {
      indexAxis: 'y',
      responsive: true,
      maintainAspectRatio: false,
      interaction: { mode: 'nearest', axis: 'y', intersect: false },
      scales: {
        x: { beginAtZero: true, grid: { color: COLOR_LINEA }, border: { display: false }, ticks: { callback: v => montoCompacto(v), maxTicksLimit: 5, maxRotation: 0 } },
        y: { grid: { display: false } },
      },
      plugins: {
        legend: { display: false },
        tooltip: {
          callbacks: {
            title: items => titulo(items[0]?.dataIndex ?? 0),
            label: (c: TooltipItem<'bar'>) => `${formatMonto(c.parsed.x)} · ${detalle(c.dataIndex)}`,
          },
        },
      },
    },
  };
}

export function configPorEstado(estados: EstadoReporte[]): ConfigGrafico {
  return barrasHorizontales(
    estados.map(e => e.estado),
    estados.map(e => e.monto),
    estados.map(e => COLOR_ESTADO[e.estado]),
    i => `${gastos(estados[i].cantidad)} · ${porcentaje(estados[i].porcentaje)}`,
    i => estados[i].estado,
  );
}

export function configRanking(filas: FilaRanking[]): ConfigGrafico {
  return barrasHorizontales(
    filas.map(f => recortar(f.etiqueta)),
    filas.map(f => f.monto),
    COLOR_RANKING,
    i => `${gastos(filas[i].cantidad)} · ${porcentaje(filas[i].porcentaje)}`,
    i => `${filas[i].puesto}. ${filas[i].etiqueta}`,
  );
}

/** Resúmenes para aria-label: lo que una persona con lector de pantalla necesita del gráfico. */
export function descripcionPorMes(meses: MesReporte[]): string {
  if (meses.length === 0) return 'Gasto por mes: sin datos.';
  const mayor = meses.reduce((a, b) => (b.total > a.total ? b : a));
  return `Gasto por mes de ${nombreMes(meses[0].mes)} a ${nombreMes(meses[meses.length - 1].mes)}. ` +
    `El mes con más gasto fue ${nombreMes(mayor.mes)}, con ${formatMonto(mayor.total)}.`;
}

export function descripcionPorEstado(estados: EstadoReporte[]): string {
  return `Distribución por estado: ${estados.map(e => `${e.estado} ${porcentaje(e.porcentaje)}`).join(', ')}.`;
}

export function descripcionRanking(titulo: string, filas: FilaRanking[]): string {
  if (filas.length === 0) return `${titulo}: sin datos.`;
  return `${titulo}: primero ${filas[0].etiqueta}, con ${formatMonto(filas[0].monto)} (${porcentaje(filas[0].porcentaje)} del total).`;
}
```

`frontend/src/app/features/reportes/grafico/chart-js.ts`:

```ts
import { BarController, BarElement, CategoryScale, Chart, Legend, LinearScale, Tooltip } from 'chart.js';
import type { ConfigGrafico } from '../graficos';

// Solo lo que usan los gráficos de barras: Chart.js queda en el chunk de Reportes.
Chart.register(BarController, BarElement, CategoryScale, LinearScale, Tooltip, Legend);
Chart.defaults.font.family = "'Overpass Variable', 'Overpass', system-ui, sans-serif";
Chart.defaults.color = '#4a5056'; // --tinta-2

/** Lo que el componente necesita de un gráfico; en las pruebas se reemplaza (jsdom no tiene canvas). */
export interface InstanciaGrafico {
  actualizar(config: ConfigGrafico): void;
  destruir(): void;
}

export type FabricaGrafico = (lienzo: HTMLCanvasElement, config: ConfigGrafico) => InstanciaGrafico;

function conMovimiento(config: ConfigGrafico): ConfigGrafico {
  const reducido = typeof matchMedia === 'function' && matchMedia('(prefers-reduced-motion: reduce)').matches;
  return { ...config, options: { ...config.options, animation: reducido ? false : { duration: 400 } } };
}

export function crearGraficoChartJs(lienzo: HTMLCanvasElement, config: ConfigGrafico): InstanciaGrafico {
  const grafico = new Chart(lienzo, conMovimiento(config));
  return {
    actualizar(nueva) {
      const conAnimacion = conMovimiento(nueva);
      grafico.data = conAnimacion.data;
      grafico.options = conAnimacion.options ?? {};
      grafico.update();
    },
    destruir() {
      grafico.destroy();
    },
  };
}
```

`frontend/src/app/features/reportes/grafico/grafico.component.ts`:

```ts
import {
  ChangeDetectionStrategy, Component, DestroyRef, ElementRef, InjectionToken, afterRenderEffect, inject, input, viewChild,
} from '@angular/core';
import type { ConfigGrafico } from '../graficos';
import { FabricaGrafico, InstanciaGrafico, crearGraficoChartJs } from './chart-js';

export const FABRICA_GRAFICO = new InjectionToken<FabricaGrafico>('FABRICA_GRAFICO', {
  providedIn: 'root',
  factory: () => crearGraficoChartJs,
});

/** Lienzo de Chart.js: lo crea con la primera configuración, lo actualiza y lo destruye con el componente. */
@Component({
  selector: 'app-grafico',
  template: `
    <div class="grafico" [style.height.px]="alto()">
      <canvas #lienzo role="img" [attr.aria-label]="descripcion()"></canvas>
    </div>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class GraficoComponent {
  readonly config = input.required<ConfigGrafico>();
  /** Resumen del gráfico para lectores de pantalla */
  readonly descripcion = input.required<string>();
  readonly alto = input(280);

  private readonly lienzo = viewChild.required<ElementRef<HTMLCanvasElement>>('lienzo');
  private readonly fabrica = inject(FABRICA_GRAFICO);
  private instancia: InstanciaGrafico | null = null;

  constructor() {
    afterRenderEffect(() => {
      const config = this.config();
      if (this.instancia) {
        this.instancia.actualizar(config);
      } else {
        this.instancia = this.fabrica(this.lienzo().nativeElement, config);
      }
    });
    inject(DestroyRef).onDestroy(() => this.instancia?.destruir());
  }
}
```

- [ ] **Step 5: Ejecutar y verificar que pasan**

Run: el mismo comando del Step 3.
Expected: PASS, 11 pruebas (configuración 8, componente 3).

- [ ] **Step 6: Checkpoint (sin commit)**

Run: `cd frontend && npx ng test --watch=false`
Expected: PASS (168).

---

### Task 8: Página «Reportes», estilos e iconos (frontend)

**Files:**
- Modify: `frontend/src/app/shared/ui/icono/iconos.ts` (iconos `chart-bar` y `download`)
- Create: `frontend/src/app/features/reportes/reportes.component.ts`, `.html`, `.spec.ts`
- Create: `frontend/src/styles/reportes.css`
- Modify: `frontend/src/styles.css` (importa la hoja nueva al final)

**Interfaces:**
- Consumes: todo lo de las Tareas 6 y 7; `ToastService`, `mensajeError`, `formatFecha`, `formatMonto`, `hoyIso`, `EstadoVacioComponent`, `IconoComponent`.
- Produces: `ReportesComponent` (`app-reportes`) con inputs `desde` y `hasta` (enlazados desde la URL por `withComponentInputBinding`).

- [ ] **Step 1: Escribir las pruebas que fallan**

`frontend/src/app/features/reportes/reportes.component.spec.ts`:

```ts
import { afterEach, describe, expect, it, vi } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { HttpErrorResponse } from '@angular/common/http';
import { Router, provideRouter } from '@angular/router';
import { Observable, Subject, of, throwError } from 'rxjs';
import type { ActiveElement, ChartEvent } from 'chart.js';
import { ReportesComponent } from './reportes.component';
import { FABRICA_GRAFICO } from './grafico/grafico.component';
import type { ConfigGrafico } from './graficos';
import { rangoDeAtajo } from './filtro-reporte';
import { ReportesService } from '../../core/services/reportes.service';
import { ToastService } from '../../core/services/toast.service';
import { ArchivoPdf, ResumenReporte } from '../../core/models/reporte.model';
import { mensajeError } from '../../core/utils/error-messages';
import { hoyIso } from '../../core/utils/format';

const fila = (puesto: number, etiqueta: string) => ({ puesto, etiqueta, cantidad: 3, monto: 300 - puesto * 10, porcentaje: 30 });
const RESUMEN: ResumenReporte = {
  desde: '2025-11-01',
  hasta: '2026-01-31',
  indicadores: { total: 1600, cantidad: 12, promedio: 133.33, sustentado: 1249.5, pendiente: 350.5, porcentaje_sustentado: 78.1 },
  por_mes: [
    { mes: '2025-11', registrado: 100, pagado: 200, sustentado: 300, total: 600 },
    { mes: '2025-12', registrado: 0, pagado: 0, sustentado: 0, total: 0 },
    { mes: '2026-01', registrado: 50.5, pagado: 0, sustentado: 949.5, total: 1000 },
  ],
  por_estado: [
    { estado: 'Registrado', cantidad: 2, monto: 150.5, porcentaje: 9.4 },
    { estado: 'Pagado', cantidad: 1, monto: 200, porcentaje: 12.5 },
    { estado: 'Sustentado', cantidad: 9, monto: 1249.5, porcentaje: 78.1 },
  ],
  por_cliente: [fila(1, 'Agro Norte SAC'), fila(2, 'Minera Sur SA')],
  por_unidad: [fila(1, 'T1A-123')],
  por_conductor: [],
};
const VACIO: ResumenReporte = {
  ...RESUMEN,
  indicadores: { total: 0, cantidad: 0, promedio: 0, sustentado: 0, pendiente: 0, porcentaje_sustentado: 0 },
  por_cliente: [], por_unidad: [], por_conductor: [],
};

interface Opciones {
  resumen?: () => Observable<ResumenReporte>;
  descargar?: () => Observable<ArchivoPdf>;
  desde?: string;
  hasta?: string;
}

async function montar(op: Opciones = {}) {
  const configs: ConfigGrafico[] = [];
  const servicio = {
    resumen: vi.fn(op.resumen ?? (() => of(RESUMEN))),
    descargar: vi.fn(op.descargar ?? (() => of({ blob: new Blob(['%PDF']), nombre: 'reporte.pdf' }))),
  };
  TestBed.configureTestingModule({
    providers: [
      provideRouter([]),
      { provide: ReportesService, useValue: servicio },
      { provide: FABRICA_GRAFICO, useValue: (_l: HTMLCanvasElement, c: ConfigGrafico) => (configs.push(c), { actualizar: vi.fn(), destruir: vi.fn() }) },
    ],
  });
  const navegar = vi.spyOn(TestBed.inject(Router), 'navigate').mockResolvedValue(true);
  const f = TestBed.createComponent(ReportesComponent);
  if (op.desde !== undefined) f.componentRef.setInput('desde', op.desde);
  if (op.hasta !== undefined) f.componentRef.setInput('hasta', op.hasta);
  f.detectChanges();
  await f.whenStable();
  f.detectChanges();
  const el = f.nativeElement as HTMLElement;
  const boton = (texto: string) => [...el.querySelectorAll('button')].find(b => b.textContent?.includes(texto)) as HTMLButtonElement;
  const toasts = () => TestBed.inject(ToastService).toasts();
  return { f, el, servicio, navegar, configs, boton, toasts };
}

// El gráfico por mes es el único con una serie por estado
const clicEnMes = (configs: ConfigGrafico[], indice: number) => {
  const porMes = configs.find(c => c.data.datasets.length === 3)!;
  (porMes.options!.onClick as (e: ChartEvent, el: ActiveElement[]) => void)({} as ChartEvent, [{ index: indice } as ActiveElement]);
};

const rangoNavegado = (navegar: ReturnType<typeof vi.fn>) => (navegar.mock.lastCall?.[1] as { queryParams: object }).queryParams;

describe('ReportesComponent', () => {
  afterEach(() => vi.restoreAllMocks());

  it('sin rango en la URL consulta los últimos 12 meses', async () => {
    const { servicio, boton } = await montar();
    expect(servicio.resumen).toHaveBeenCalledWith(rangoDeAtajo('ultimos-12', hoyIso()));
    expect(boton('Últimos 12 meses').getAttribute('aria-pressed')).toBe('true');
  });

  it('muestra los indicadores, los gráficos y la tabla de datos del rango de la URL', async () => {
    const { el, servicio, configs } = await montar({ desde: '2025-11-01', hasta: '2026-01-31' });
    expect(servicio.resumen).toHaveBeenCalledWith({ desde: '2025-11-01', hasta: '2026-01-31' });
    const valores = [...el.querySelectorAll('.indicador__valor')].map(v => v.textContent?.trim());
    expect(valores).toEqual(['S/ 1,600.00', '12', 'S/ 133.33', '78.1 %']);
    expect(el.querySelector('.indicador__nota')?.textContent?.trim()).toBe('Pendiente: S/ 350.50');
    // por mes, por estado, cliente y unidad; el conductor no tiene datos
    expect(configs).toHaveLength(4);
    expect(el.textContent).toContain('Sin datos en este período.');
    expect(el.querySelector('canvas')?.getAttribute('aria-label')).toContain('El mes con más gasto fue enero 2026');
    expect([...el.querySelectorAll('.reportes__datos th[scope="row"]')].map(t => t.textContent?.trim())).toContain('diciembre 2025');
  });

  it('un atajo cambia el rango de la URL', async () => {
    const { boton, navegar } = await montar();
    boton('Este año').click();
    expect(rangoNavegado(navegar)).toEqual(rangoDeAtajo('este-anio', hoyIso()));
  });

  it('un rango inválido se explica junto a las fechas, no se consulta y no se descarga', async () => {
    const { el, servicio, boton } = await montar({ desde: '2026-05-02', hasta: '2026-05-01' });
    expect(el.querySelector('#reporte-error')?.textContent?.trim()).toBe('La fecha inicial no puede ser posterior a la final.');
    expect(el.querySelector('#reporte-desde')?.getAttribute('aria-invalid')).toBe('true');
    expect(servicio.resumen).not.toHaveBeenCalled();
    expect(boton('Descargar reporte estadístico').disabled).toBe(true);
  });

  it('un clic en un mes filtra a ese mes, avisa y «Quitar» vuelve al rango anterior', async () => {
    const { f, el, navegar, configs, boton } = await montar({ desde: '2025-11-01', hasta: '2026-01-31' });
    clicEnMes(configs, 1);
    expect(rangoNavegado(navegar)).toEqual({ desde: '2025-12-01', hasta: '2025-12-31' });
    f.componentRef.setInput('desde', '2025-12-01');
    f.componentRef.setInput('hasta', '2025-12-31');
    f.detectChanges();
    expect(el.querySelector('.reportes__aviso')?.textContent).toContain('Viendo: diciembre 2025');
    boton('Quitar').click();
    expect(rangoNavegado(navegar)).toEqual({ desde: '2025-11-01', hasta: '2026-01-31' });
  });

  it('cambiar una fecha a mano después de elegir un mes quita el aviso', async () => {
    const { f, el, configs } = await montar({ desde: '2025-11-01', hasta: '2026-01-31' });
    clicEnMes(configs, 1);
    f.componentRef.setInput('desde', '2025-12-01');
    f.componentRef.setInput('hasta', '2025-12-31');
    f.detectChanges();
    const desde = el.querySelector<HTMLInputElement>('#reporte-desde')!;
    desde.value = '2025-12-05';
    desde.dispatchEvent(new Event('change'));
    f.detectChanges();
    expect(el.querySelector('.reportes__aviso')).toBeNull();
  });

  it('cambiar las dos fechas seguidas usa ambas, aunque la URL no se haya actualizado', async () => {
    const { el, navegar } = await montar({ desde: '2025-11-01', hasta: '2026-01-31' });
    const desde = el.querySelector<HTMLInputElement>('#reporte-desde')!;
    const hasta = el.querySelector<HTMLInputElement>('#reporte-hasta')!;
    desde.value = '2026-05-02';
    desde.dispatchEvent(new Event('change'));
    hasta.value = '2026-05-01';
    hasta.dispatchEvent(new Event('change'));
    expect(rangoNavegado(navegar)).toEqual({ desde: '2026-05-02', hasta: '2026-05-01' });
  });

  it('sin gastos en el rango muestra el estado vacío y deja descargar', async () => {
    const { el, boton } = await montar({ resumen: () => of(VACIO) });
    expect(el.textContent).toContain('No hay gastos en este período.');
    expect(el.querySelector('.indicadores')).toBeNull();
    expect(boton('Descargar detalle de gastos').disabled).toBe(false);
  });

  it('si falla la consulta ofrece reintentar', async () => {
    let fallar = true;
    const { f, el, servicio, boton } = await montar({ resumen: () => (fallar ? throwError(() => new HttpErrorResponse({ status: 0 })) : of(RESUMEN)) });
    expect(el.textContent).toContain('No se pudo cargar el reporte.');
    fallar = false;
    boton('Reintentar').click();
    await f.whenStable();
    f.detectChanges();
    expect(servicio.resumen).toHaveBeenCalledTimes(2);
    expect(el.querySelector('.indicadores')).not.toBeNull();
  });

  it('descargar muestra «Generando…», evita el doble clic y guarda el archivo', async () => {
    const pendiente = new Subject<ArchivoPdf>();
    const { f, servicio, boton } = await montar({ descargar: () => pendiente });
    Object.assign(URL, { createObjectURL: vi.fn(() => 'blob:x'), revokeObjectURL: vi.fn() });
    const clic = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {});
    boton('Descargar reporte estadístico').click();
    f.detectChanges();
    expect(boton('Generando…').disabled).toBe(true);
    expect(boton('Descargar detalle de gastos').disabled).toBe(true);
    boton('Generando…').click();
    expect(servicio.descargar).toHaveBeenCalledTimes(1);
    pendiente.next({ blob: new Blob(['%PDF']), nombre: 'reporte-estadistico.pdf' });
    pendiente.complete();
    f.detectChanges();
    expect(clic).toHaveBeenCalledTimes(1);
    expect(boton('Descargar reporte estadístico').disabled).toBe(false);
  });

  it('si la descarga falla, avisa con el mensaje de error de la app', async () => {
    const err = new HttpErrorResponse({ status: 500 });
    const { f, toasts, boton } = await montar({ descargar: () => throwError(() => err) });
    boton('Descargar detalle de gastos').click();
    f.detectChanges();
    expect(toasts().at(-1)?.title).toBe(mensajeError(err));
    expect(toasts().at(-1)?.type).toBe('err');
    expect(boton('Descargar detalle de gastos').disabled).toBe(false);
  });
});
```

- [ ] **Step 2: Ejecutar y verificar que fallan**

Run: `cd frontend && npx ng test --watch=false --include src/app/features/reportes/reportes.component.spec.ts`
Expected: FAIL: no se puede resolver `./reportes.component`.

- [ ] **Step 3: Agregar los iconos**

En `frontend/src/app/shared/ui/icono/iconos.ts`, dentro de `ICONOS` y antes de la línea `'receipt': [`, agrega (Tabler Icons 3.48.0, outline):

```ts
  'chart-bar': ['M3 13a1 1 0 0 1 1 -1h4a1 1 0 0 1 1 1v6a1 1 0 0 1 -1 1h-4a1 1 0 0 1 -1 -1l0 -6', 'M15 9a1 1 0 0 1 1 -1h4a1 1 0 0 1 1 1v10a1 1 0 0 1 -1 1h-4a1 1 0 0 1 -1 -1l0 -10', 'M9 5a1 1 0 0 1 1 -1h4a1 1 0 0 1 1 1v14a1 1 0 0 1 -1 1h-4a1 1 0 0 1 -1 -1l0 -14', 'M4 20h14'],
  'download': ['M4 17v2a2 2 0 0 0 2 2h12a2 2 0 0 0 2 -2v-2', 'M7 11l5 5l5 -5', 'M12 4l0 12'],
```

- [ ] **Step 4: Implementar la página**

`frontend/src/app/features/reportes/reportes.component.ts`:

```ts
import { ChangeDetectionStrategy, Component, computed, inject, input, signal } from '@angular/core';
import { rxResource } from '@angular/core/rxjs-interop';
import { ActivatedRoute, Router } from '@angular/router';
import { FiltroReporte, FilaRanking, TipoPdf } from '../../core/models/reporte.model';
import { ReportesService } from '../../core/services/reportes.service';
import { ToastService } from '../../core/services/toast.service';
import { guardarArchivo } from '../../core/utils/descarga';
import { mensajeError } from '../../core/utils/error-messages';
import { formatFecha, formatMonto, hoyIso } from '../../core/utils/format';
import { EstadoVacioComponent } from '../../shared/ui/estado-vacio/estado-vacio.component';
import { IconoComponent } from '../../shared/ui/icono/icono.component';
import {
  ATAJOS, Atajo, atajoActivo, errorDeFiltro, filtroDesdeConsulta, mesCompleto, nombreMes, rangoDeAtajo, rangoDelMes,
} from './filtro-reporte';
import { GraficoComponent } from './grafico/grafico.component';
import {
  COLOR_ESTADO, ConfigGrafico, configPorEstado, configPorMes, configRanking, descripcionPorEstado, descripcionPorMes,
  descripcionRanking, porcentaje,
} from './graficos';

interface Ranking {
  id: string;
  titulo: string;
  columna: string;
  filas: FilaRanking[];
  config: ConfigGrafico;
  descripcion: string;
}

/**
 * Dashboard gerencial: estadísticas de gastos de un rango de fechas, con gráficos
 * interactivos y descarga de los dos PDF. El rango vive en la URL (?desde=&hasta=).
 */
@Component({
  selector: 'app-reportes',
  imports: [EstadoVacioComponent, GraficoComponent, IconoComponent],
  templateUrl: './reportes.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ReportesComponent {
  /** Parámetros de la URL, enlazados por withComponentInputBinding */
  readonly desde = input<string>();
  readonly hasta = input<string>();

  private readonly reportes = inject(ReportesService);
  private readonly router = inject(Router);
  private readonly ruta = inject(ActivatedRoute);
  private readonly toast = inject(ToastService);

  protected readonly hoy = hoyIso();
  protected readonly atajos = ATAJOS;
  protected readonly filtro = computed(() => filtroDesdeConsulta(this.desde(), this.hasta(), this.hoy));
  protected readonly errorFiltro = computed(() => errorDeFiltro(this.filtro()));
  protected readonly atajo = computed(() => atajoActivo(this.filtro(), this.hoy));
  /** Rango que había antes de elegir un mes en el gráfico; «Quitar» vuelve a él */
  private readonly rangoPrevio = signal<FiltroReporte | null>(null);
  protected readonly mesVisto = computed(() => (this.rangoPrevio() ? mesCompleto(this.filtro()) : null));
  protected readonly descargando = signal<TipoPdf | null>(null);

  protected readonly resumen = rxResource({
    params: () => (this.errorFiltro() ? undefined : this.filtro()),
    stream: ({ params }) => this.reportes.resumen(params),
  });
  protected readonly datos = computed(() => (this.resumen.hasValue() ? this.resumen.value() : null));
  protected readonly configMes = computed(() => {
    const d = this.datos();
    return d ? configPorMes(d.por_mes, mes => this.elegirMes(mes)) : null;
  });
  protected readonly configEstado = computed(() => {
    const d = this.datos();
    return d ? configPorEstado(d.por_estado) : null;
  });
  protected readonly rankings = computed<Ranking[]>(() => {
    const d = this.datos();
    if (!d) return [];
    const ranking = (id: string, titulo: string, columna: string, filas: FilaRanking[]): Ranking => ({
      id, titulo, columna, filas, config: configRanking(filas), descripcion: descripcionRanking(titulo, filas),
    });
    return [
      ranking('cliente', 'Gasto por cliente', 'Cliente', d.por_cliente),
      ranking('unidad', 'Gasto por unidad', 'Unidad (placa)', d.por_unidad),
      ranking('conductor', 'Gasto por conductor', 'Conductor', d.por_conductor),
    ];
  });

  protected readonly formatMonto = formatMonto;
  protected readonly formatFecha = formatFecha;
  protected readonly porcentaje = porcentaje;
  protected readonly nombreMes = nombreMes;
  protected readonly colorEstado = COLOR_ESTADO;
  protected readonly descripcionPorMes = descripcionPorMes;
  protected readonly descripcionPorEstado = descripcionPorEstado;

  protected elegirAtajo(atajo: Atajo): void {
    this.rangoPrevio.set(null);
    this.irA(rangoDeAtajo(atajo, this.hoy));
  }

  /** Usa las dos fechas visibles: si se cambian seguidas, la URL todavía no tiene la primera */
  protected cambiarRango(desde: string, hasta: string): void {
    this.rangoPrevio.set(null);
    this.irA({ desde, hasta });
  }

  /** Clic en una columna del gráfico por mes: todo el dashboard pasa a ese mes */
  protected elegirMes(mes: string): void {
    if (!this.rangoPrevio()) this.rangoPrevio.set(this.filtro());
    this.irA(rangoDelMes(mes));
  }

  protected quitarMes(): void {
    const previo = this.rangoPrevio();
    this.rangoPrevio.set(null);
    if (previo) this.irA(previo);
  }

  protected descargar(tipo: TipoPdf): void {
    if (this.descargando() || this.errorFiltro()) return;
    this.descargando.set(tipo);
    this.reportes.descargar(tipo, this.filtro()).subscribe({
      next: ({ blob, nombre }) => {
        this.descargando.set(null);
        guardarArchivo(blob, nombre);
      },
      error: err => {
        this.descargando.set(null);
        this.toast.show(mensajeError(err), '', 'err');
      },
    });
  }

  private irA(filtro: FiltroReporte): void {
    void this.router.navigate([], { relativeTo: this.ruta, queryParams: { desde: filtro.desde, hasta: filtro.hasta } });
  }
}
```

`frontend/src/app/features/reportes/reportes.component.html`:

```html
<div class="pagina">
  <header class="pagina__encabezado">
    <div>
      <h1 class="pagina__titulo">Reportes</h1>
      <p class="pagina__bajada">Gastos adicionales del {{ formatFecha(filtro().desde) }} al {{ formatFecha(filtro().hasta) }}.</p>
    </div>
    <div class="pagina__acciones">
      <button type="button" class="btn btn--primario" [disabled]="!!errorFiltro() || !!descargando()" (click)="descargar('estadistico')">
        <app-icono nombre="download" />{{ descargando() === 'estadistico' ? 'Generando…' : 'Descargar reporte estadístico (PDF)' }}
      </button>
      <button type="button" class="btn btn--secundario" [disabled]="!!errorFiltro() || !!descargando()" (click)="descargar('detalle')">
        <app-icono nombre="download" />{{ descargando() === 'detalle' ? 'Generando…' : 'Descargar detalle de gastos (PDF)' }}
      </button>
    </div>
  </header>

  <section class="reportes__filtro" aria-label="Rango de fechas">
    <div class="pestanas reportes__atajos" role="group" aria-label="Atajos de rango">
      @for (a of atajos; track a.id) {
        <button
          type="button"
          class="pestanas__item"
          [class.pestanas__item--activa]="atajo() === a.id"
          [attr.aria-pressed]="atajo() === a.id"
          (click)="elegirAtajo(a.id)"
        >
          {{ a.etiqueta }}
        </button>
      }
    </div>
    <div class="reportes__fechas">
      <div class="campo" [class.campo--con-error]="errorFiltro()">
        <label class="campo__etiqueta" for="reporte-desde">Desde</label>
        <input
          class="control"
          id="reporte-desde"
          #campoDesde
          type="date"
          [value]="filtro().desde"
          (change)="cambiarRango(campoDesde.value, campoHasta.value)"
          [attr.aria-invalid]="!!errorFiltro() || null"
          [attr.aria-describedby]="errorFiltro() ? 'reporte-error' : null"
        />
      </div>
      <div class="campo" [class.campo--con-error]="errorFiltro()">
        <label class="campo__etiqueta" for="reporte-hasta">Hasta</label>
        <input
          class="control"
          id="reporte-hasta"
          #campoHasta
          type="date"
          [value]="filtro().hasta"
          (change)="cambiarRango(campoDesde.value, campoHasta.value)"
          [attr.aria-invalid]="!!errorFiltro() || null"
          [attr.aria-describedby]="errorFiltro() ? 'reporte-error' : null"
        />
      </div>
    </div>
    @if (errorFiltro(); as error) {
      <p class="campo__error" id="reporte-error">{{ error }}</p>
    }
  </section>

  @if (mesVisto(); as mes) {
    <p class="alerta alerta--via reportes__aviso">
      <app-icono nombre="calendar" [tamano]="18" />
      <span>Viendo: {{ nombreMes(mes) }}</span>
      <button type="button" class="enlace" (click)="quitarMes()">Quitar</button>
    </p>
  }

  @if (errorFiltro()) {
    <!-- Con el rango inválido no se consulta: el mensaje ya está junto a las fechas -->
  } @else if (resumen.isLoading()) {
    <div class="reportes__esqueleto" aria-busy="true">
      <div class="esqueleto esqueleto--franja"></div>
      <div class="esqueleto esqueleto--bloque"></div>
      <div class="esqueleto esqueleto--bloque"></div>
    </div>
  } @else if (resumen.error()) {
    <app-estado-vacio icono="cloud-off" titulo="No se pudo cargar el reporte." texto="Revisa tu conexión e inténtalo de nuevo.">
      <button type="button" class="btn btn--secundario" (click)="resumen.reload()">Reintentar</button>
    </app-estado-vacio>
  } @else if (datos(); as r) {
    @if (r.indicadores.cantidad === 0) {
      <app-estado-vacio icono="chart-bar" titulo="No hay gastos en este período." texto="Prueba con otro rango de fechas." />
    } @else {
      <section class="indicadores" aria-label="Indicadores del período">
        <div class="indicador">
          <span class="indicador__etiqueta">Total gastado</span>
          <span class="indicador__valor">{{ formatMonto(r.indicadores.total) }}</span>
        </div>
        <div class="indicador">
          <span class="indicador__etiqueta">Gastos registrados</span>
          <span class="indicador__valor">{{ r.indicadores.cantidad }}</span>
        </div>
        <div class="indicador">
          <span class="indicador__etiqueta">Gasto promedio</span>
          <span class="indicador__valor">{{ formatMonto(r.indicadores.promedio) }}</span>
        </div>
        <div class="indicador">
          <span class="indicador__etiqueta">Sustentado</span>
          <span class="indicador__valor">{{ porcentaje(r.indicadores.porcentaje_sustentado) }}</span>
          <span class="indicador__nota">Pendiente: {{ formatMonto(r.indicadores.pendiente) }}</span>
        </div>
      </section>

      <section class="panel reportes__tarjeta" aria-labelledby="t-por-mes">
        <h2 id="t-por-mes" class="seccion-titulo">Gasto por mes</h2>
        <p class="reportes__ayuda">Haz clic en un mes para ver solo ese mes.</p>
        <app-grafico [config]="configMes()!" [descripcion]="descripcionPorMes(r.por_mes)" [alto]="300" />
        <details class="reportes__datos">
          <summary>Ver datos</summary>
          <div class="tabla-marco">
            <table class="tabla">
              <thead>
                <tr><th scope="col">Mes</th><th scope="col">Registrado</th><th scope="col">Pagado</th><th scope="col">Sustentado</th><th scope="col">Total</th></tr>
              </thead>
              <tbody>
                @for (m of r.por_mes; track m.mes) {
                  <tr>
                    <th scope="row">{{ nombreMes(m.mes) }}</th>
                    <td>{{ formatMonto(m.registrado) }}</td>
                    <td>{{ formatMonto(m.pagado) }}</td>
                    <td>{{ formatMonto(m.sustentado) }}</td>
                    <td>{{ formatMonto(m.total) }}</td>
                  </tr>
                }
              </tbody>
            </table>
          </div>
        </details>
      </section>

      <section class="panel reportes__tarjeta" aria-labelledby="t-por-estado">
        <h2 id="t-por-estado" class="seccion-titulo">Distribución por estado</h2>
        <app-grafico [config]="configEstado()!" [descripcion]="descripcionPorEstado(r.por_estado)" [alto]="150" />
        <details class="reportes__datos">
          <summary>Ver datos</summary>
          <div class="tabla-marco">
            <table class="tabla">
              <thead>
                <tr><th scope="col">Estado</th><th scope="col">Gastos</th><th scope="col">Monto</th><th scope="col">%</th></tr>
              </thead>
              <tbody>
                @for (e of r.por_estado; track e.estado) {
                  <tr>
                    <th scope="row"><span class="reportes__muestra" [style.background]="colorEstado[e.estado]" aria-hidden="true"></span>{{ e.estado }}</th>
                    <td>{{ e.cantidad }}</td>
                    <td>{{ formatMonto(e.monto) }}</td>
                    <td>{{ porcentaje(e.porcentaje) }}</td>
                  </tr>
                }
              </tbody>
            </table>
          </div>
        </details>
      </section>

      <div class="reportes__rankings">
        @for (k of rankings(); track k.id) {
          <section class="panel reportes__tarjeta" [attr.aria-labelledby]="'t-' + k.id">
            <h2 [id]="'t-' + k.id" class="seccion-titulo">{{ k.titulo }}</h2>
            @if (k.filas.length === 0) {
              <p class="reportes__ayuda">Sin datos en este período.</p>
            } @else {
              <app-grafico [config]="k.config" [descripcion]="k.descripcion" [alto]="60 + k.filas.length * 28" />
              <details class="reportes__datos">
                <summary>Ver datos</summary>
                <div class="tabla-marco">
                  <table class="tabla">
                    <thead>
                      <tr><th scope="col">Puesto</th><th scope="col">{{ k.columna }}</th><th scope="col">Gastos</th><th scope="col">Monto</th><th scope="col">%</th></tr>
                    </thead>
                    <tbody>
                      @for (f of k.filas; track f.etiqueta) {
                        <tr>
                          <td>{{ f.puesto }}</td>
                          <th scope="row">{{ f.etiqueta }}</th>
                          <td>{{ f.cantidad }}</td>
                          <td>{{ formatMonto(f.monto) }}</td>
                          <td>{{ porcentaje(f.porcentaje) }}</td>
                        </tr>
                      }
                    </tbody>
                  </table>
                </div>
              </details>
            }
          </section>
        }
      </div>
    }
  }
</div>
```

- [ ] **Step 5: Estilos**

`frontend/src/styles/reportes.css`:

```css
/* ══ Reportes: dashboard gerencial ══ */
.reportes__filtro {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-end;
  justify-content: space-between;
  gap: var(--e-3) var(--e-6);
  margin-bottom: var(--e-5);
}

.reportes__atajos {
  margin-bottom: 0;
}

/* Los atajos son botones con el aspecto de las pestañas */
.reportes__atajos .pestanas__item {
  border: 0;
  background: none;
  font-family: inherit;
  cursor: pointer;
}

.reportes__fechas {
  display: flex;
  gap: var(--e-3);
}

.reportes__fechas .campo {
  margin: 0;
}

.reportes__filtro .campo__error {
  flex-basis: 100%;
}

.reportes__aviso {
  align-items: center;
  margin-bottom: var(--e-5);
}

.reportes__aviso .enlace {
  margin-left: auto;
  border: 0;
  background: none;
  font: inherit;
  font-weight: 600;
  cursor: pointer;
}

.reportes__esqueleto {
  display: grid;
  gap: var(--e-4);
}

/* ── Indicadores ── */
.indicadores {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: var(--e-4);
  margin-bottom: var(--e-5);
}

.indicador {
  display: grid;
  gap: var(--e-1);
  padding: var(--e-4) var(--e-5);
  border: 1px solid var(--linea);
  border-radius: var(--r-panel);
  background: var(--panel);
}

.indicador__etiqueta {
  color: var(--tinta-2);
  font-size: var(--t-13);
  font-weight: 600;
}

.indicador__valor {
  font-size: var(--t-24);
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  letter-spacing: -0.01em;
}

.indicador__nota {
  color: var(--tinta-3);
  font-size: var(--t-13);
  font-variant-numeric: tabular-nums;
}

/* ── Tarjetas de gráficos ── */
.reportes__tarjeta {
  display: grid;
  gap: var(--e-3);
  min-width: 0;
  margin-bottom: var(--e-5);
}

.reportes__ayuda {
  margin-top: calc(-1 * var(--e-2));
  color: var(--tinta-3);
  font-size: var(--t-13);
}

.grafico {
  position: relative;
  width: 100%;
}

.reportes__datos summary {
  width: fit-content;
  color: var(--tinta-2);
  font-size: var(--t-14);
  font-weight: 600;
  cursor: pointer;
}

.reportes__datos .tabla-marco {
  margin-top: var(--e-3);
}

.reportes__datos td {
  font-variant-numeric: tabular-nums;
}

.reportes__muestra {
  display: inline-block;
  width: 10px;
  height: 10px;
  margin-right: var(--e-2);
  border-radius: 2px;
  vertical-align: baseline;
}

.reportes__rankings {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0 var(--e-5);
}

/* Los nombres de cliente son los más largos: su ranking va a todo el ancho */
.reportes__rankings > :first-child {
  grid-column: 1 / -1;
}

@media (max-width: 1023px) {
  .indicadores {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 767px) {
  .reportes__fechas {
    width: 100%;
  }

  .reportes__fechas .campo {
    flex: 1;
    min-width: 0;
  }

  .indicador {
    padding: var(--e-3) var(--e-4);
  }

  .indicador__valor {
    font-size: var(--t-20);
  }

  .reportes__rankings {
    grid-template-columns: minmax(0, 1fr);
  }
}
```

Al final de `frontend/src/styles.css` agrega:

```css

@import './styles/reportes.css';
```

- [ ] **Step 6: Ejecutar y verificar que pasan**

Run: el mismo comando del Step 2.
Expected: PASS, 11 pruebas.

- [ ] **Step 7: Checkpoint (sin commit)**

Run: `cd frontend && npx ng test --watch=false`
Expected: PASS (179).

---

### Task 9: Ruta y navegación (frontend)

**Files:**
- Modify: `frontend/src/app/app.routes.ts` + `app.routes.spec.ts`
- Modify: `frontend/src/app/core/navigation/navegacion.ts` + `navegacion.spec.ts`
- Modify: `frontend/src/app/shell/paleta-comandos/paleta.logic.spec.ts`
- Modify: `frontend/src/app/shell/barra-movil/barra-movil.component.ts`, `.html`
- Create: `frontend/src/app/shell/barra-movil/barra-movil.component.spec.ts`

**Interfaces:**
- Consumes: `ReportesComponent` (Tarea 8), icono `chart-bar` (Tarea 8).
- Produces: ruta hija `reportes` con `title: 'Reportes · LIZAR'` y carga diferida; «Reportes» en `ENLACES_PRINCIPALES` (entre Gastos y Operación) y en `destinosPaleta()` (`id: 'ir-reportes'`); grupo «Gerencia» con «Reportes» en la hoja «Más» del celular; `enMas` reemplaza a `enCatalogos`.

- [ ] **Step 1: Escribir las pruebas que fallan**

En `frontend/src/app/app.routes.spec.ts`, en la lista esperada de rutas hijas, cambia `'gastos/:id', 'operacion',` por `'gastos/:id', 'reportes', 'operacion',`.

En `frontend/src/app/core/navigation/navegacion.spec.ts`, cambia `['/app/inicio', '/app/gastos', '/app/operacion', '/app/catalogos']` por `['/app/inicio', '/app/gastos', '/app/reportes', '/app/operacion', '/app/catalogos']`, y reemplaza `expect(destinosPaleta()).toHaveLength(2 + MODULOS.length);` por:

```ts
    expect(destinosPaleta()).toHaveLength(3 + MODULOS.length);
    expect(destinosPaleta().find(d => d.etiqueta === 'Reportes')?.ruta).toBe('/app/reportes');
```

En `frontend/src/app/shell/paleta-comandos/paleta.logic.spec.ts`, cambia `expect(r.filter(e => e.grupo === 'Ir a')).toHaveLength(13);` por `toHaveLength(14)`.

`frontend/src/app/shell/barra-movil/barra-movil.component.spec.ts`:

```ts
import { describe, expect, it, vi } from 'vitest';
import { Component, signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { Router, provideRouter } from '@angular/router';
import { BarraMovilComponent } from './barra-movil.component';
import { AuthService } from '../../core/services/auth.service';
import { GastosStore } from '../../core/state/gastos.store';

@Component({ template: '' })
class Vacio {}

async function montar(url: string) {
  TestBed.configureTestingModule({
    providers: [
      provideRouter([{ path: '**', component: Vacio }]),
      { provide: AuthService, useValue: { usuario: signal({ nombre: 'Ana Quispe' }), logout: vi.fn() } },
      { provide: GastosStore, useValue: { pendientes: signal([]) } },
    ],
  });
  await TestBed.inject(Router).navigateByUrl(url);
  const f = TestBed.createComponent(BarraMovilComponent);
  f.detectChanges();
  return f.nativeElement as HTMLElement;
}

const botonMas = (el: HTMLElement) => [...el.querySelectorAll('button')].find(b => b.textContent?.includes('Más'))!;

describe('BarraMovilComponent', () => {
  it('la hoja «Más» lleva a Reportes', async () => {
    const el = await montar('/app/inicio');
    const enlace = [...el.querySelectorAll<HTMLAnchorElement>('dialog a')].find(a => a.textContent?.includes('Reportes'));
    expect(enlace?.getAttribute('href')).toBe('/app/reportes');
    expect(botonMas(el).classList).not.toContain('tabbar__item--activo');
  });

  it('en Reportes y en Catálogos se marca «Más» como activo', async () => {
    expect(botonMas(await montar('/app/reportes?desde=2026-01-01')).classList).toContain('tabbar__item--activo');
    TestBed.resetTestingModule();
    expect(botonMas(await montar('/app/catalogos/ubigeo')).classList).toContain('tabbar__item--activo');
  });
});
```

- [ ] **Step 2: Ejecutar y verificar que fallan**

Run: `cd frontend && npx ng test --watch=false --include src/app/app.routes.spec.ts --include src/app/core/navigation/navegacion.spec.ts --include src/app/shell/paleta-comandos/paleta.logic.spec.ts --include src/app/shell/barra-movil/barra-movil.component.spec.ts`
Expected: FAIL en las cuatro: falta la ruta `reportes`, el enlace, el destino de la paleta y el enlace de la hoja «Más».

- [ ] **Step 3: Implementar**

En `frontend/src/app/app.routes.ts`, antes de `{ path: 'operacion', pathMatch: 'full', redirectTo: 'operacion/viajes' },`, agrega:

```ts
      {
        path: 'reportes',
        title: 'Reportes · LIZAR',
        loadComponent: () => import('./features/reportes/reportes.component').then(m => m.ReportesComponent),
      },
```

En `frontend/src/app/core/navigation/navegacion.ts`:
- en `ENLACES_PRINCIPALES`, después de la línea de Gastos, agrega `  { etiqueta: 'Reportes', ruta: '/app/reportes', icono: 'chart-bar' },`;
- en `destinosPaleta()`, después de la entrada `ir-gastos`, agrega `    { id: 'ir-reportes', grupo: 'Ir a', etiqueta: 'Reportes', icono: 'chart-bar', ruta: '/app/reportes' },`.

En `frontend/src/app/shell/barra-movil/barra-movil.component.ts`:
- cambia el comentario de la clase a `/** Barra inferior del celular: Inicio, Gastos, Operación y Más (Reportes, Catálogos y cuenta). */`;
- reemplaza la línea `protected readonly enCatalogos = computed(() => this.url().startsWith('/app/catalogos'));` por:

```ts
  /** «Más» se marca activo en las secciones que viven dentro de la hoja */
  protected readonly enMas = computed(() => ['/app/catalogos', '/app/reportes'].some(r => this.url().startsWith(r)));
```

En `frontend/src/app/shell/barra-movil/barra-movil.component.html`:
- cambia `[class.tabbar__item--activo]="enCatalogos()"` por `[class.tabbar__item--activo]="enMas()"`;
- reemplaza `      <p class="hoja__grupo">Catálogos</p>` (la primera línea dentro de `hoja__cuerpo`) por:

```html
      <p class="hoja__grupo">Gerencia</p>
      <ul class="lista-enlaces">
        <li>
          <a class="lista-enlaces__item" routerLink="/app/reportes" (click)="cerrarMas()">
            <app-icono nombre="chart-bar" [tamano]="20" />Reportes
            <app-icono class="lista-enlaces__flecha" nombre="chevron-right" [tamano]="16" />
          </a>
        </li>
      </ul>
      <p class="hoja__grupo">Catálogos</p>
```

- [ ] **Step 4: Ejecutar y verificar que pasan**

Run: el mismo comando del Step 2.
Expected: PASS en los cuatro archivos.

- [ ] **Step 5: Checkpoint (sin commit)**

Run: `cd frontend && npx ng test --watch=false && npx ng build 2>&1 | grep -E "WARNING|ERROR|Initial total|reportes-component"`
Expected: PASS (181 pruebas, 40 archivos); build sin avisos; «Initial total» cerca de 360 kB y un chunk diferido `reportes-component` (~55 kB transferidos) que contiene Chart.js.

---

### Task 10: Verificación en navegador contra el servicio real y cierre

**Files (fuera del repositorio, en el scratchpad):**
- `$SCRATCH/visual/reportes-navegador.mjs` (con `playwright-core` instalado en `$SCRATCH/visual`)
- Capturas y PDF descargados en `$SCRATCH/visual/capturas/`

`SCRATCH=/private/tmp/claude-501/-Users-helcorp-Downloads-Proyectos-Universidad-Proyecto-Campstone/00d73130-eac8-4a38-b8ba-011054089b50/scratchpad`

**Interfaces:**
- Consumes: todo lo anterior; `db.sql`; los scripts DEMO.
- Produces: evidencia de punta a punta (filtro, tooltip, clic en un mes, ambas descargas, rango inválido, celular sin desborde).

- [ ] **Step 1: Base temporal con datos DEMO y servicio en 8091**

Run:
```bash
cd gastos-service
BASE="$(sed -n 's/^DATABASE_URL=//p' .env | tr -d "\"'")"
TMP=$(python3 -c "import sys,urllib.parse as u; p=u.urlsplit(sys.argv[1]); print(u.urlunsplit(p._replace(path='/lizar_reportes_prueba')))" "$BASE")
psql "$BASE" -qc "DROP DATABASE IF EXISTS lizar_reportes_prueba" && psql "$BASE" -qc "CREATE DATABASE lizar_reportes_prueba"
psql "$TMP" -q -v ON_ERROR_STOP=1 -f ../db.sql > /dev/null
for m in migrations/*.sql; do psql "$TMP" -q -v ON_ERROR_STOP=1 -f "$m"; done
psql "$TMP" -q -f demo/reportes_demo.sql > /dev/null
go build -o "$SCRATCH/gastos-service-prueba" . && mkdir -p "$SCRATCH/uploads-prueba"
```
En segundo plano, **desde `gastos-service/`** (las migraciones se leen de `./migrations`): `PORT=8091 DATABASE_URL="$TMP" UPLOADS_DIR="$SCRATCH/uploads-prueba" "$SCRATCH/gastos-service-prueba"`.
En segundo plano, desde `frontend/`: `npx ng serve --port 4300`.

- [ ] **Step 2: Script de verificación**

`$SCRATCH/visual/reportes-navegador.mjs`:

```js
// Verificación de Reportes en navegador contra gastos-service real (base temporal con datos DEMO).
// Uso: node reportes-navegador.mjs <url-frontend> <url-gastos-service>
import { chromium } from 'playwright-core';
import { mkdirSync, readFileSync } from 'node:fs';

const BASE = process.argv[2] ?? 'http://localhost:4300';
const GASTOS = process.argv[3] ?? 'http://localhost:8091';
const CHROME = '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
const SALIDA = new URL('./capturas/', import.meta.url).pathname;
mkdirSync(SALIDA, { recursive: true });
const USUARIO = { id_usuario: 1, username: 'aquispe', email: 'ana@lizar.pe', nombre: 'Ana Quispe', rol: 'admin', estado: true };

const verificar = (condicion, mensaje) => {
  if (!condicion) throw new Error(`Falló: ${mensaje}`);
  console.log('OK', mensaje);
};

async function esperar(url) {
  for (let i = 0; i < 90; i++) {
    try {
      if ((await fetch(url)).ok) return;
    } catch {
      /* todavía arrancando */
    }
    await new Promise(r => setTimeout(r, 2000));
  }
  throw new Error(`${url} no respondió`);
}

async function contexto(navegador, opciones) {
  const ctx = await navegador.newContext({ ...opciones, acceptDownloads: true });
  await ctx.addInitScript(([u, gastos]) => {
    localStorage.setItem('lizar_token', 'demo');
    localStorage.setItem('lizar_user', JSON.stringify(u));
    localStorage.setItem('lizar_gastos_api_base', gastos);
  }, [USUARIO, GASTOS]);
  // La API principal (FastAPI) no participa en Reportes: responde vacío
  await ctx.route('http://localhost:8000/**', r => r.fulfill({ status: 200, contentType: 'application/json', body: '[]' }));
  return ctx;
}

async function descargarYVerificar(p, texto, prefijo) {
  const [descarga] = await Promise.all([p.waitForEvent('download'), p.getByRole('button', { name: texto }).click()]);
  const ruta = `${SALIDA}${descarga.suggestedFilename()}`;
  await descarga.saveAs(ruta);
  const cabecera = readFileSync(ruta).subarray(0, 5).toString();
  verificar(descarga.suggestedFilename().startsWith(prefijo) && cabecera === '%PDF-', `descarga «${descarga.suggestedFilename()}» es un PDF`);
}

await esperar(BASE);
await esperar(`${GASTOS}/health`);
const navegador = await chromium.launch({ executablePath: CHROME, headless: true });

// 1. Escritorio: indicadores, gráficos, tooltip, clic en un mes y descargas
{
  const ctx = await contexto(navegador, { viewport: { width: 1440, height: 900 } });
  const p = await ctx.newPage();
  await p.goto(`${BASE}/app/reportes`, { waitUntil: 'networkidle' });
  await p.locator('.indicador__valor').first().waitFor();
  verificar((await p.locator('.indicador').count()) === 4, 'cuatro indicadores');
  verificar((await p.locator('canvas').count()) === 5, 'cinco gráficos (mes, estado y tres rankings)');
  verificar(await p.getByRole('button', { name: 'Últimos 12 meses' }).getAttribute('aria-pressed') === 'true', 'atajo por defecto activo');
  await p.waitForTimeout(600);
  await p.screenshot({ path: `${SALIDA}reportes-escritorio.png`, fullPage: true });

  const lienzo = p.locator('canvas').first();
  const caja = await lienzo.boundingBox();
  await p.mouse.move(caja.x + caja.width - 30, caja.y + caja.height * 0.6);
  await p.waitForTimeout(400);
  await lienzo.screenshot({ path: `${SALIDA}reportes-tooltip.png` });
  await p.mouse.click(caja.x + caja.width - 30, caja.y + caja.height * 0.6);
  await p.locator('.reportes__aviso').waitFor();
  verificar((await p.locator('.reportes__aviso').textContent()).includes('Viendo:'), 'clic en el último mes filtra y avisa');
  verificar(new URL(p.url()).searchParams.get('desde')?.endsWith('-01'), 'la URL lleva el mes elegido');
  await p.getByRole('button', { name: 'Quitar' }).click();
  await p.waitForFunction(() => !document.querySelector('.reportes__aviso'));
  verificar(true, '«Quitar» vuelve al rango anterior');

  await descargarYVerificar(p, 'Descargar reporte estadístico (PDF)', 'reporte-estadistico-');
  await descargarYVerificar(p, 'Descargar detalle de gastos (PDF)', 'detalle-gastos-');

  await p.locator('#reporte-desde').fill('2026-05-02');
  await p.locator('#reporte-hasta').fill('2026-05-01');
  await p.locator('#reporte-hasta').dispatchEvent('change');
  await p.locator('#reporte-error').waitFor();
  verificar(await p.getByRole('button', { name: 'Descargar reporte estadístico (PDF)' }).isDisabled(), 'rango inválido: no se descarga');
  await ctx.close();
}

// 2. Celular: sin desborde y Reportes desde «Más»
{
  const ctx = await contexto(navegador, { viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true, deviceScaleFactor: 2 });
  const p = await ctx.newPage();
  await p.goto(`${BASE}/app/inicio`, { waitUntil: 'networkidle' });
  await p.getByRole('button', { name: 'Más' }).click();
  await p.getByRole('link', { name: 'Reportes' }).click();
  await p.locator('.indicador__valor').first().waitFor();
  await p.waitForTimeout(600);
  const desborde = await p.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
  verificar(desborde === 0, `sin desborde horizontal en el celular (${desborde})`);
  await p.screenshot({ path: `${SALIDA}reportes-celular.png`, fullPage: true });
  await ctx.close();
}

await navegador.close();
console.log('Reportes verificado en navegador.');
```

Si `$SCRATCH/visual/node_modules/playwright-core/index.js` no existe: `cd "$SCRATCH/visual" && (test -f package.json || npm init -y) && npm install --no-audit --no-fund playwright-core@1.48.2`.

Run: `node "$SCRATCH/visual/reportes-navegador.mjs" http://localhost:4300 http://localhost:8091`
Expected: 10 líneas `OK …` y `Reportes verificado en navegador.`

- [ ] **Step 3: Revisar las capturas y los PDF**

Mira `reportes-escritorio.png`, `reportes-tooltip.png` y `reportes-celular.png`, y convierte los dos PDF descargados con `sips -s format png <pdf> --out <png>` para mirarlos. Comprueba:
- filtro y botones en una fila en escritorio, 4 indicadores, barras apiladas con huecos de 2 px y leyenda, cliente a todo el ancho y unidad/conductor lado a lado, etiquetas horizontales sin cortes;
- tooltip con el mes completo, los tres estados y el total;
- celular: indicadores 2×2, botones a lo ancho, ejes con «S/ 10 mil», nombres recortados con «…», sin desborde (la barra inferior a media página en una captura `fullPage` es un artefacto conocido);
- PDF: cabecera, período, «Página N de M», cifras iguales a las del dashboard.
Corrige en un solo lote lo que encuentres y vuelve a ejecutar el script una vez.

- [ ] **Step 4: Detener y limpiar**

Detén tu `ng serve` del puerto 4300 y el servicio del 8091 (verifica con `lsof` que el 4200 del usuario sigue), y borra la base temporal: `psql "$BASE" -qc "DROP DATABASE lizar_reportes_prueba"`.

- [ ] **Step 5: Detector de impeccable**

Run: `cd "/Users/helcorp/Downloads/Proyectos/Universidad/Proyecto Campstone" && /Users/helcorp/.claude/skills/impeccable/scripts/impeccable detect --json frontend/src/app/features frontend/src/styles`
Expected: solo los 2 falsos positivos conocidos (`<img [src]>` en adjuntar y revisión).

- [ ] **Step 6: Verificación final (sin commit)**

Run:
```bash
cd "/Users/helcorp/Downloads/Proyectos/Universidad/Proyecto Campstone/gastos-service" && gofmt -l . && go vet ./... && TEST_DATABASE_URL="$(sed -n 's/^DATABASE_URL=//p' .env | tr -d "\"'")" go test ./... -count=1
psql "$(sed -n 's/^DATABASE_URL=//p' .env | tr -d "\"'")" -tAc "SELECT count(*) FROM information_schema.schemata WHERE schema_name LIKE 'prueba_%'"
cd ../frontend && npx ng test --watch=false && npx ng build
cd .. && git status --short | grep -v "^ M\|^ D" | head -40 && git log --oneline | head -1
```
Expected: Go limpio y en PASS; `0` esquemas sobrantes; frontend 181/181 y build OK; `git log` sigue en `1953886 first commit`.

Informa al usuario: qué se agregó; que debe **reiniciar gastos-service** (dependencia nueva y endpoints nuevos); cómo cargar y borrar los datos de demostración con `psql`; cómo correr las pruebas de Go; y las cifras de la demo.
