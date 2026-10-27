# OCR de comprobantes — plan de implementación

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Al abrir la revisión de un comprobante Pendiente, gastos-service lo lee (pdftotext / Tesseract), extrae monto, fecha, RUC y tipo, y la pantalla de revisión muestra «Sugerido ✓» junto a las casillas que puede respaldar, con «Aplicar sugerencias»; el OCR nunca aprueba ni rechaza.

**Architecture:** Un paquete nuevo `gastos-service/internal/ocr` con piezas pequeñas: normalización y extracción (funciones puras), sugerencias contra el gasto (puras), validación de la ruta dentro de `UPLOADS_DIR` y un lector que invoca `pdftotext`, `pdftoppm` y `tesseract` con `exec.CommandContext` y tiempos límite. Un endpoint `GET /comprobantes/{id}/lectura` guarda la lectura en `lectura_comprobante` (migración 0005) y recalcula las sugerencias en cada consulta; el `Handler` recibe el lector como interfaz para que las pruebas usen uno falso. En Angular, `LecturaService` (30 s) y `RevisionComprobanteComponent` piden la lectura con `rxResource` y la muestran sin bloquear la revisión.

**Tech Stack:** Go 1.25.4 + pgx v5 + PostgreSQL (biblioteca estándar: `os/exec`, `regexp`, `encoding/json`); poppler (`pdftotext`, `pdftoppm`) y Tesseract con `spa` como programas externos; Angular 21 (standalone, signals, zoneless) + Vitest; Playwright (`playwright-core` con el Chrome del sistema, fuera del repositorio) para la verificación en navegador.

**Spec:** `docs/superpowers/specs/2026-10-22-ocr-comprobantes-design.md`

**Nota:** el código de las Tareas 1 a 6 se probó antes en una copia del proyecto (sin `.env`, `uploads/` ni `db.sql`): 31 pruebas del paquete `ocr` en verde (la de Tesseract real se omite), las 7 pruebas de integración del endpoint en verde contra esquemas temporales, y el frontend con 242 pruebas en verde y build de producción sin avisos.

## Global Constraints

- Se trabaja en `main` **sin ramas ni commits** (decisión del usuario). Donde la skill pide «Commit», este plan tiene un paso «Checkpoint (sin commit)»: correr un comando y comprobar su salida. `git log --oneline | head -1` debe seguir mostrando `1b81dbe feat: checklist documental configurable por cliente (REQ-15, REQ-16)` al terminar.
- La migración es `gastos-service/migrations/0005_lectura_comprobante.sql` (ya existen `0003_criterio_checklist.sql` y `0004_usuario.sql`; la spec dice 0004, pero ese número ya está tomado). Su contenido es el de la spec §7: tabla `lectura_comprobante (comprobante_id INTEGER PRIMARY KEY REFERENCES comprobante (id), estado VARCHAR(20) NOT NULL CHECK (estado IN ('leido', 'sin_texto', 'no_disponible', 'error')), texto TEXT NOT NULL DEFAULT '', datos JSONB NOT NULL DEFAULT '{}', creado_en TIMESTAMP NOT NULL DEFAULT now())`, idempotente.
- Go: **sin dependencias nuevas** (solo biblioteca estándar —`os/exec`, `regexp`, `encoding/json`— y lo que ya está en `go.mod`); la directiva `go 1.25.4` no cambia. Frontend: **sin dependencias npm nuevas**.
- No se modifican `frontend/src/app/core/models/gasto-estados.ts` ni `frontend/src/app/core/utils/error-messages.ts` (RNF-1).
- Textos de la interfaz, exactamente como en la spec: «Leyendo comprobante…», «Sugerido ✓ · Leído S/ 85.50» (forma: `Sugerido ✓ · <detalle>`), «Aplicar sugerencias», «No se pudo leer el comprobante automáticamente». Los errores de red usan `mensajeError`.
- Programas externos: siempre `exec.CommandContext` con argumentos separados, **nunca una shell**; 20 s por ejecución y 30 s por lectura completa; la ruta del archivo debe resolver dentro de `UPLOADS_DIR` (con `..` o con un enlace simbólico que salga, se rechaza).
- En esta máquina hay `pdftotext` y `pdftoppm` (`/opt/homebrew/bin`), pero **no `tesseract`**: las pruebas que necesitan una herramienta real se omiten con `t.Skip` si falta, y el lector responde `no_disponible` sin fallar.
- El OCR solo sugiere ✓ para `monto_coincide`, `fecha_coincide`, `a_nombre_empresa` y `tipo_valido`; nunca `legible` y nunca ✗. RUC de LIZAR SRL: `20481608750`. Solo se lee un comprobante Pendiente (otro estado: 409 `el comprobante ya fue revisado`). El endpoint no exige rol.
- Se guardan solo las lecturas `leido` y `sin_texto`; `no_disponible` y `error` no se guardan (se reintentan).
- Estilos solo en `frontend/src/styles/*.css`, con tokens existentes; sin colores nuevos.
- Las pruebas de integración usan `TEST_DATABASE_URL` y un esquema temporal `prueba_<n>` que se borra al terminar; nunca escriben en las tablas reales. La verificación en navegador usa un esquema temporal propio (`prueba_ocr_navegador`) que se borra al final.
- No se tocan los servicios del usuario en los puertos **8000, 8081 y 4200**; las comprobaciones manuales usan **8093** (gastos-service de prueba) y **4300** (`ng serve`).
- Comandos: Go `cd gastos-service && TEST_DATABASE_URL="$(sed -n 's/^DATABASE_URL=//p' .env | tr -d "\"'")" go test ./... -count=1`; frontend `cd frontend && npx ng test --watch=false` y `npx ng build`. Si el chequeo de seguridad rechaza un comando con comillas anidadas, ponerlo en un script dentro de `$SCRATCH` y ejecutarlo.
- `SCRATCH=/private/tmp/lizar-ocr` (fuera del repositorio) para binarios, scripts y capturas de la Tarea 7.

## Review Focus

1. **El archivo del comprobante no está donde dice** (borrado, o `archivo_url` manipulado con `..`, una ruta absoluta o un enlace simbólico): la consulta responde 200 con `estado: "error"`, no ejecuta ningún programa sobre ese archivo y no guarda nada. Lo fijan `TestRutaSeguraRechazaLoQueSaleDeUploads` y `TestRutaSeguraArchivoInexistenteOCarpeta` (Tarea 3) y `TestLecturaDeUnArchivoFueraDeUploadsDaErrorSinLeer` (Tarea 4).
2. **Una herramienta se cuelga o tarda** (PDF enorme, foto de 5 MB, Tesseract lento): cada programa se corta a los 20 s, la lectura entera a los 30 s, el resultado es `error` (no se guarda) y el frontend espera exactamente 30 s antes de avisar. Lo fijan `TestCadaEjecucionTieneTiempoLimite`, `TestLaLecturaCompletaTieneTiempoLimite` y `TestLecturaCanceladaDaError` (Tarea 3) y «espera hasta 30 s, no los 8 s de las demás operaciones» (Tarea 5).
3. **El OCR devuelve bytes que PostgreSQL no acepta** (NUL, UTF-8 inválido, caracteres de control): el texto se limpia al normalizar y la lectura se guarda igual, sin un 500. Lo fijan `TestNormalizarEspaciosYCifrasConfundidas` (Tarea 1) y `TestLecturaConTextoQuePostgresNoAceptaSeGuardaIgual` (Tarea 4).
4. **Documentos que engañan a una regla simple** (una guía de remisión o una nota de crédito que mencionan «FACTURA», un correo `…lizar…@…`, «SUBTOTAL», una fecha 30.09.2026 en la línea del total): no se sugiere tipo, empresa ni monto por error. Lo fijan `TestTipoDeDocumento` y `TestFormatosDeMonto` (Tarea 1) y `TestEmpresaPorNombreYNoPorCorreo` y `TestGuiaNoSugiereTipo` (Tarea 2).
5. **La pantalla cambia de comprobante, o el cliente tiene su propio checklist**: no deben quedar sugerencias del comprobante anterior, ni sugerirse ni aplicarse criterios que el cliente no revisa. Lo fijan «con otro comprobante no quedan las sugerencias del anterior» y «las sugerencias de criterios que el cliente no revisa no se muestran ni se aplican» (Tarea 6).

## Mapa de archivos

| Archivo | Responsabilidad |
|---|---|
| `gastos-service/internal/ocr/normalizar.go` (nuevo) | Doc del paquete; `Normalizar` (espacios, saltos, control, UTF-8, «O»/«l» dentro de números) y `Alfanumericos`. |
| `gastos-service/internal/ocr/extraer.go` (nuevo) | Tipos `Datos`, `Monto`, `Campo`; `Extraer` y las reglas de monto, fecha, RUC y tipo. |
| `gastos-service/internal/ocr/sugerir.go` (nuevo) | `Sugerencia`, `RucEmpresa`, `Sugerir` (4 criterios, solo ✓). |
| `gastos-service/internal/ocr/ruta.go` (nuevo) | `RutaSegura`: `archivo_url` → ruta real dentro de `UPLOADS_DIR`. |
| `gastos-service/internal/ocr/lector.go` (nuevo) | Estados, `Lectura`, interfaz `Lector`, `Herramientas` (pdftotext, pdftoppm, tesseract con tiempos límite), `NuevoLector`, `Resumen`. |
| `gastos-service/internal/ocr/*_test.go` (nuevos) | `extraer_test.go`, `sugerir_test.go`, `ruta_test.go`, `lector_test.go` (programas falsos en `sh`), `herramientas_test.go` (herramientas reales, se omiten si faltan). |
| `gastos-service/migrations/0005_lectura_comprobante.sql` (nuevo) | Tabla `lectura_comprobante`. |
| `gastos-service/internal/handlers/lectura.go` (nuevo) | `GET /comprobantes/{id}/lectura`: valida, reusa o lee, guarda, sugiere. |
| `gastos-service/internal/handlers/handlers.go` (modificar) | Campo `Lector ocr.Lector`, `New` lo crea, ruta nueva. |
| `gastos-service/main.go` (modificar) | Registra al arrancar qué herramientas encontró. |
| `gastos-service/internal/handlers/lectura_integration_test.go` (nuevo) | Endpoint con un lector falso sobre un esquema temporal. |
| `gastos-service/Dockerfile` (modificar) | Instala poppler y Tesseract con español en la imagen de producción. |
| `frontend/src/app/core/models/lectura.ts` (+ `.spec.ts`, nuevos) | Tipos de la respuesta, `MSG_LECTURA_FALLIDA`, `textoSugerencia`, `clavesSugeridas`. |
| `frontend/src/app/core/services/lectura.service.ts` (+ `.spec.ts`, nuevos) | `LecturaService.lectura(id)` con `timeout` de 30 s. |
| `frontend/src/app/features/gastos/revision-comprobante/revision-comprobante.component.{ts,html,spec.ts}` (modificar) | Pide la lectura, muestra sugerencias, «Aplicar sugerencias», avisos. |
| `frontend/src/styles/gastos.css` (modificar) | Estilos de la lectura junto al checklist. |

---

### Task 1: Normalización y extracción de datos (Go, funciones puras)

**Files:**
- Create: `gastos-service/internal/ocr/normalizar.go`
- Create: `gastos-service/internal/ocr/extraer.go`
- Test: `gastos-service/internal/ocr/extraer_test.go`

**Interfaces:**
- Consumes: nada.
- Produces:
  - `ocr.Normalizar(texto string) string` y `ocr.Alfanumericos(texto string) int`; `ocr.MinAlfanumericos = 20`.
  - Constantes de confianza `ocr.Alta = "alta"`, `ocr.Media = "media"`, `ocr.Ninguna = "ninguna"`.
  - `ocr.Monto{Valor float64; Texto string; Confianza string}` (JSON `valor`, `texto`, `confianza`).
  - `ocr.Campo{Valor string; Texto string; Confianza string}` (mismo JSON; fecha como `aaaa-mm-dd`, tipo como `factura` | `boleta` | `otro`).
  - `ocr.Datos{Monto Monto; Fecha Campo; Rucs []string; Tipo Campo}` (JSON `monto`, `fecha`, `rucs`, `tipo`; `Rucs` nunca es nil).
  - `ocr.Extraer(texto string) ocr.Datos` (normaliza antes de extraer; `Extraer("")` es el valor «sin datos»).

- [ ] **Step 1: Escribir las pruebas que fallan**

`gastos-service/internal/ocr/extraer_test.go`:

```go
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
```

- [ ] **Step 2: Correr las pruebas y ver que fallan**

Run: `cd gastos-service && go test ./internal/ocr -count=1`
Expected: FAIL de compilación: `undefined: Normalizar`, `undefined: Extraer`, `undefined: Monto`…

- [ ] **Step 3: Implementar la normalización**

`gastos-service/internal/ocr/normalizar.go`:

```go
// Package ocr lee los comprobantes de pago (PDF con texto, PDF escaneados y fotos), extrae monto,
// fecha, RUC y tipo de documento, y sugiere qué criterios del checklist se cumplen. Nunca aprueba ni
// rechaza: solo sugiere ✓, y siempre decide una persona.
package ocr

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// MinAlfanumericos es el mínimo de letras y cifras para considerar que un texto se leyó; con menos,
// un PDF se trata como escaneado y una lectura queda «sin texto».
const MinAlfanumericos = 20

var espacios = regexp.MustCompile(`[\t\f\v \x{00A0}]+`)

// tramoNumerico es una seguidilla de cifras, separadores y las letras que el OCR confunde con 0 y 1.
var tramoNumerico = regexp.MustCompile(`[0-9OolI][0-9OolI.,/-]*[0-9OolI]`)

var aCifra = strings.NewReplacer("O", "0", "o", "0", "l", "1", "I", "1")

// Normalizar deja el texto en UTF-8 válido, sin caracteres de control, con un solo espacio entre
// palabras, sin líneas vacías y con «O», «o», «l» e «I» cambiadas por 0 y 1 solo dentro de números.
func Normalizar(texto string) string {
	texto = strings.ToValidUTF8(texto, "")
	texto = strings.ReplaceAll(texto, "\r\n", "\n")
	texto = strings.ReplaceAll(texto, "\r", "\n")
	texto = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' || r == '\f' || r == '\v' {
			return r
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, texto)
	lineas := []string{}
	for _, l := range strings.Split(texto, "\n") {
		l = strings.TrimSpace(espacios.ReplaceAllString(l, " "))
		if l != "" {
			lineas = append(lineas, corregirCifras(l))
		}
	}
	return strings.Join(lineas, "\n")
}

func corregirCifras(linea string) string {
	var b strings.Builder
	ultimo := 0
	for _, m := range tramoNumerico.FindAllStringIndex(linea, -1) {
		if !esNumero(linea, m[0], m[1]) {
			continue
		}
		b.WriteString(linea[ultimo:m[0]])
		b.WriteString(aCifra.Replace(linea[m[0]:m[1]]))
		ultimo = m[1]
	}
	b.WriteString(linea[ultimo:])
	return b.String()
}

// esNumero: el tramo tiene alguna cifra real y no está pegado a otra letra, así «Folio», «Trujillo»
// o la placa «T046» no cambian.
func esNumero(linea string, ini, fin int) bool {
	if !strings.ContainsAny(linea[ini:fin], "0123456789") {
		return false
	}
	if r, _ := utf8.DecodeLastRuneInString(linea[:ini]); ini > 0 && unicode.IsLetter(r) {
		return false
	}
	if r, _ := utf8.DecodeRuneInString(linea[fin:]); fin < len(linea) && unicode.IsLetter(r) {
		return false
	}
	return true
}

// Alfanumericos cuenta las letras y cifras de un texto.
func Alfanumericos(texto string) int {
	n := 0
	for _, r := range texto {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			n++
		}
	}
	return n
}
```

- [ ] **Step 4: Implementar la extracción**

`gastos-service/internal/ocr/extraer.go`:

```go
package ocr

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Confianza de un dato: alta si se encontró junto a su etiqueta, media si se infirió, ninguna si no
// se encontró.
const (
	Alta    = "alta"
	Media   = "media"
	Ninguna = "ninguna"
)

// Monto es el importe total leído, con la línea donde se leyó.
type Monto struct {
	Valor     float64 `json:"valor"`
	Texto     string  `json:"texto"`
	Confianza string  `json:"confianza"`
}

// Campo es un dato de texto leído (la fecha como aaaa-mm-dd; el tipo como factura, boleta u otro).
type Campo struct {
	Valor     string `json:"valor"`
	Texto     string `json:"texto"`
	Confianza string `json:"confianza"`
}

// Datos es lo que se extrae del texto de un comprobante; se guarda en lectura_comprobante.datos.
type Datos struct {
	Monto Monto    `json:"monto"`
	Fecha Campo    `json:"fecha"`
	Rucs  []string `json:"rucs"`
	Tipo  Campo    `json:"tipo"`
}

// Extraer lee monto, fecha, RUC y tipo de documento de un texto (se normaliza antes).
func Extraer(texto string) Datos {
	lineas := strings.Split(Normalizar(texto), "\n")
	return Datos{Monto: extraerMonto(lineas), Fecha: extraerFecha(lineas), Rucs: extraerRucs(lineas), Tipo: extraerTipo(lineas)}
}

// coincidencias devuelve cada aparición de re en s (índices como FindStringSubmatchIndex) y sigue
// buscando desde el final del último grupo: los bordes (?:^|\D) y (?:\D|$) no se consumen, así dos
// datos separados por un solo espacio se leen los dos.
func coincidencias(re *regexp.Regexp, s string) [][]int {
	var out [][]int
	for desde := 0; desde < len(s); {
		m := re.FindStringSubmatchIndex(s[desde:])
		if m == nil {
			break
		}
		for i := range m {
			if m[i] >= 0 {
				m[i] += desde
			}
		}
		out = append(out, m)
		desde = m[len(m)-1]
	}
	return out
}

var (
	etiquetaTotal = regexp.MustCompile(`(?i)(?:^|\P{L})(importe\s+total|total\s+a\s+pagar|total)(?:\P{L}|$)`)
	// cifraMonto: 2 decimales con coma o punto, con o sin separador de miles; no toma «30.09.2026».
	cifraMonto = regexp.MustCompile(`(?:^|[^\d.,])(\d{1,3}(?:[.,]\d{3})+[.,]\d{2}|\d+[.,]\d{2})(?:[^\d.,]|[.,](?:\D|$)|$)`)
)

// aNumero convierte «1,234.56», «1.234,56» u «85,50»: el último separador es el decimal.
func aNumero(s string) (float64, bool) {
	entero := strings.NewReplacer(".", "", ",", "").Replace(s[:len(s)-3])
	v, err := strconv.ParseFloat(entero+"."+s[len(s)-2:], 64)
	return v, err == nil
}

// extraerMonto toma el mayor número con 2 decimales que sigue a «TOTAL», «IMPORTE TOTAL» o «TOTAL A
// PAGAR» en la misma línea o, si en esa línea no hay ninguno, en la siguiente.
func extraerMonto(lineas []string) Monto {
	mejor := Monto{Confianza: Ninguna}
	for i, l := range lineas {
		et := etiquetaTotal.FindStringSubmatchIndex(l)
		if et == nil {
			continue
		}
		candidatas := []string{l[et[3]:]}
		if i+1 < len(lineas) {
			candidatas = append(candidatas, lineas[i+1])
		}
		for j, texto := range candidatas {
			encontrado := false
			for _, m := range coincidencias(cifraMonto, texto) {
				v, ok := aNumero(texto[m[2]:m[3]])
				if !ok {
					continue
				}
				encontrado = true
				if mejor.Confianza == Ninguna || v > mejor.Valor {
					mejor = Monto{Valor: v, Texto: lineas[i+j], Confianza: Alta}
				}
			}
			if encontrado {
				break
			}
		}
	}
	return mejor
}

var (
	fechaDMA      = regexp.MustCompile(`(?:^|\D)(\d{1,2})[/-](\d{1,2})[/-](\d{4})(?:\D|$)`)
	fechaAMD      = regexp.MustCompile(`(?:^|\D)(\d{4})-(\d{1,2})-(\d{1,2})(?:\D|$)`)
	fechaLarga    = regexp.MustCompile(`(?i)(?:^|\D)(\d{1,2})\s+de\s+(\p{L}+)\s+(?:de|del)\s+(\d{4})(?:\D|$)`)
	etiquetaFecha = regexp.MustCompile(`(?i)fecha`)
	meses         = map[string]int{
		"enero": 1, "febrero": 2, "marzo": 3, "abril": 4, "mayo": 5, "junio": 6, "julio": 7, "agosto": 8,
		"septiembre": 9, "setiembre": 9, "octubre": 10, "noviembre": 11, "diciembre": 12,
	}
)

type fechaEnLinea struct {
	pos   int
	valor string
}

// fechaValida descarta fechas imposibles (31/02, mes 13) comparando con lo que normaliza time.Date.
func fechaValida(a, m, d int) (string, bool) {
	t := time.Date(a, time.Month(m), d, 0, 0, 0, 0, time.UTC)
	if t.Year() != a || int(t.Month()) != m || t.Day() != d {
		return "", false
	}
	return t.Format("2006-01-02"), true
}

func fechasEn(linea string) []fechaEnLinea {
	var out []fechaEnLinea
	agregar := func(pos int, a, m, d int) {
		if v, ok := fechaValida(a, m, d); ok {
			out = append(out, fechaEnLinea{pos, v})
		}
	}
	num := func(s string) int { n, _ := strconv.Atoi(s); return n }
	for _, m := range coincidencias(fechaDMA, linea) {
		agregar(m[2], num(linea[m[6]:m[7]]), num(linea[m[4]:m[5]]), num(linea[m[2]:m[3]]))
	}
	for _, m := range coincidencias(fechaAMD, linea) {
		agregar(m[2], num(linea[m[2]:m[3]]), num(linea[m[4]:m[5]]), num(linea[m[6]:m[7]]))
	}
	for _, m := range coincidencias(fechaLarga, linea) {
		if mes, ok := meses[strings.ToLower(linea[m[4]:m[5]])]; ok {
			agregar(m[2], num(linea[m[6]:m[7]]), mes, num(linea[m[2]:m[3]]))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].pos < out[j].pos })
	return out
}

// extraerFecha prefiere la primera fecha válida que sigue a «FECHA» en su línea (alta); si no hay,
// la primera fecha válida del texto (media).
func extraerFecha(lineas []string) Campo {
	var primera *Campo
	for _, l := range lineas {
		et := etiquetaFecha.FindStringIndex(l)
		var elegida *fechaEnLinea
		for _, f := range fechasEn(l) {
			if et != nil && f.pos >= et[1] && elegida == nil {
				elegida = &f
			}
			if primera == nil {
				primera = &Campo{Valor: f.valor, Texto: l, Confianza: Media}
			}
		}
		if elegida != nil {
			return Campo{Valor: elegida.valor, Texto: l, Confianza: Alta}
		}
	}
	if primera != nil {
		return *primera
	}
	return Campo{Confianza: Ninguna}
}

var reRuc = regexp.MustCompile(`(?:^|\D)((?:10|15|17|20)\d{9})(?:\D|$)`)

// extraerRucs devuelve los RUC (11 cifras que empiezan con 10, 15, 17 o 20) en el orden en que
// aparecen, sin repetir.
func extraerRucs(lineas []string) []string {
	out := []string{}
	vistos := map[string]bool{}
	for _, l := range lineas {
		for _, m := range coincidencias(reRuc, l) {
			if ruc := l[m[2]:m[3]]; !vistos[ruc] {
				vistos[ruc] = true
				out = append(out, ruc)
			}
		}
	}
	return out
}

// reTipo: la primera de estas palabras en el texto decide el tipo; una guía o una nota de crédito o
// de débito que mencione una factura sigue siendo «otro».
var reTipo = regexp.MustCompile(`(?i)(?:^|\P{L})(factura|boleta|gu[ií]a|nota\s+de\s+(?:cr[eé]dito|d[eé]bito))(?:\P{L}|$)`)

func extraerTipo(lineas []string) Campo {
	for _, l := range lineas {
		m := reTipo.FindStringSubmatchIndex(l)
		if m == nil {
			continue
		}
		switch palabra := strings.ToLower(l[m[2]:m[3]]); palabra {
		case "factura", "boleta":
			return Campo{Valor: palabra, Texto: l, Confianza: Alta}
		default:
			return Campo{Valor: "otro", Texto: l, Confianza: Alta}
		}
	}
	return Campo{Valor: "otro", Confianza: Ninguna}
}
```

Reglas que fija este archivo (y que las pruebas comprueban): el monto solo se toma junto a «TOTAL», «IMPORTE TOTAL» o «TOTAL A PAGAR» como palabras sueltas (no «SUBTOTAL»), en la misma línea o, si allí no hay cifras, en la siguiente, y gana el mayor; la fecha que sigue a «FECHA» es `alta` y cualquier otra fecha válida es `media`; los RUC se devuelven en orden sin repetir; el tipo lo decide la **primera** aparición de «FACTURA», «BOLETA», «GUÍA» o «NOTA DE CRÉDITO/DÉBITO» (una guía o una nota que citan una factura quedan como `otro`).

- [ ] **Step 5: Correr las pruebas y ver que pasan**

Run: `cd gastos-service && go test ./internal/ocr -count=1 -v`
Expected: PASS en `TestNormalizarEspaciosYCifrasConfundidas`, `TestExtraerBoleta`, `TestExtraerFacturaConComaDecimalYFechaEnLetras`, `TestFormatosDeMonto`, `TestFormatosDeFecha`, `TestRucs`, `TestTipoDeDocumento` y `TestTextoSinDatos`.

- [ ] **Step 6: Checkpoint (sin commit)**

Run: `cd gastos-service && gofmt -l internal/ocr && go vet ./internal/ocr`
Expected: sin salida de `gofmt -l` y `go vet` sin hallazgos.

---

### Task 2: Sugerencias del checklist (Go, funciones puras)

**Files:**
- Create: `gastos-service/internal/ocr/sugerir.go`
- Test: `gastos-service/internal/ocr/sugerir_test.go`

**Interfaces:**
- Consumes: `ocr.Datos`, `ocr.Monto`, `ocr.Campo`, `ocr.Extraer`, `ocr.Ninguna` (Tarea 1); `reportes.FormatoMonto(float64) string` («S/ 1,234.50», igual que `formatMonto` del frontend) y `reportes.FormatoFecha(time.Time) string` («30/09/2026»), que ya existen en `gastos-service/internal/reportes/formato.go`.
- Produces:
  - `ocr.RucEmpresa = "20481608750"`.
  - `ocr.Sugerencia{Sugerido bool; Detalle string}` (JSON `sugerido`, `detalle`).
  - `ocr.Sugerir(d ocr.Datos, texto string, montoGasto float64, fechaGasto time.Time) map[string]ocr.Sugerencia` con exactamente las claves `monto_coincide`, `fecha_coincide`, `a_nombre_empresa`, `tipo_valido`. Detalles: `Leído S/ 85.50`, `Leída 30/09/2026`, `RUC 20481608750` o `Menciona LIZAR`, `Factura` o `Boleta`; vacío si no se leyó el dato.

- [ ] **Step 1: Escribir las pruebas que fallan**

`gastos-service/internal/ocr/sugerir_test.go` (usa `boletaGrifo` y `facturaPeaje` de `extraer_test.go`):

```go
package ocr

import (
	"reflect"
	"testing"
	"time"
)

var fechaGasto = time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)

func TestSugerenciasCuandoTodoCoincide(t *testing.T) {
	got := Sugerir(Extraer(boletaGrifo), boletaGrifo, 85.5, fechaGasto)
	want := map[string]Sugerencia{
		"monto_coincide":   {Sugerido: true, Detalle: "Leído S/ 85.50"},
		"fecha_coincide":   {Sugerido: true, Detalle: "Leída 30/09/2026"},
		"a_nombre_empresa": {Sugerido: true, Detalle: "RUC 20481608750"},
		"tipo_valido":      {Sugerido: true, Detalle: "Boleta"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sugerencias = %+v", got)
	}
	if _, ok := got["legible"]; ok {
		t.Error("legible nunca se sugiere")
	}
}

func TestSugerenciasCuandoDifiereSoloMuestranLoLeido(t *testing.T) {
	got := Sugerir(Extraer(facturaPeaje), facturaPeaje, 1250.004, time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC))
	if s := got["monto_coincide"]; !s.Sugerido || s.Detalle != "Leído S/ 1,250.00" {
		t.Errorf("diferencia menor a medio céntimo: %+v", s)
	}
	got = Sugerir(Extraer(facturaPeaje), facturaPeaje, 1250.01, fechaGasto)
	if s := got["monto_coincide"]; s.Sugerido || s.Detalle != "Leído S/ 1,250.00" {
		t.Errorf("monto distinto: %+v", s)
	}
	if s := got["fecha_coincide"]; s.Sugerido || s.Detalle != "Leída 29/09/2026" {
		t.Errorf("fecha distinta: %+v", s)
	}
	if s := got["tipo_valido"]; !s.Sugerido || s.Detalle != "Factura" {
		t.Errorf("tipo: %+v", s)
	}
}

func TestSinDatosNoHaySugerencias(t *testing.T) {
	got := Sugerir(Extraer("Gracias por su compra"), "Gracias por su compra", 85.5, fechaGasto)
	for clave, s := range got {
		if s != (Sugerencia{}) {
			t.Errorf("%s: se esperaba sin sugerencia, llegó %+v", clave, s)
		}
	}
	if len(got) != 4 {
		t.Errorf("se esperaban 4 criterios, llegaron %d", len(got))
	}
}

func TestEmpresaPorNombreYNoPorCorreo(t *testing.T) {
	if s := Sugerir(Extraer(""), "Cliente: Transportes Lizar S.R.L.", 1, fechaGasto)["a_nombre_empresa"]; !s.Sugerido || s.Detalle != "Menciona LIZAR" {
		t.Errorf("por nombre: %+v", s)
	}
	if s := Sugerir(Extraer(""), "coordinador@transporteslizar.com ineslizarraga@hotmail.com", 1, fechaGasto)["a_nombre_empresa"]; s.Sugerido {
		t.Errorf("un correo no es el nombre de la empresa: %+v", s)
	}
}

func TestGuiaNoSugiereTipo(t *testing.T) {
	texto := "GUIA DE REMISIÓN ELECTRÓNICA\nR.U.C. 20481608750"
	got := Sugerir(Extraer(texto), texto, 1, fechaGasto)
	if got["tipo_valido"].Sugerido {
		t.Error("una guía no es boleta ni factura")
	}
	if !got["a_nombre_empresa"].Sugerido {
		t.Error("el RUC de LIZAR sí se sugiere")
	}
}
```

- [ ] **Step 2: Correr las pruebas y ver que fallan**

Run: `cd gastos-service && go test ./internal/ocr -count=1`
Expected: FAIL de compilación: `undefined: Sugerir`, `undefined: Sugerencia`.

- [ ] **Step 3: Implementar las sugerencias**

`gastos-service/internal/ocr/sugerir.go`:

```go
package ocr

import (
	"math"
	"regexp"
	"slices"
	"time"

	"gastos-service/internal/reportes"
)

// RucEmpresa es el RUC de LIZAR SRL.
const RucEmpresa = "20481608750"

// Sugerencia de un criterio del checklist: el sistema solo sugiere ✓ (Sugerido true); Detalle dice
// lo que se leyó, también cuando no coincide.
type Sugerencia struct {
	Sugerido bool   `json:"sugerido"`
	Detalle  string `json:"detalle"`
}

var palabraLizar = regexp.MustCompile(`(?i)(?:^|\P{L})lizar(?:\P{L}|$)`)

// Sugerir compara lo leído con el gasto y devuelve una sugerencia para monto_coincide,
// fecha_coincide, a_nombre_empresa y tipo_valido. Nunca para legible (es un juicio humano) y nunca
// ✗: no encontrar un dato no prueba que el comprobante esté mal.
func Sugerir(d Datos, texto string, montoGasto float64, fechaGasto time.Time) map[string]Sugerencia {
	return map[string]Sugerencia{
		"monto_coincide":   sugerirMonto(d.Monto, montoGasto),
		"fecha_coincide":   sugerirFecha(d.Fecha, fechaGasto),
		"a_nombre_empresa": sugerirEmpresa(d.Rucs, texto),
		"tipo_valido":      sugerirTipo(d.Tipo),
	}
}

func sugerirMonto(m Monto, montoGasto float64) Sugerencia {
	if m.Confianza == Ninguna {
		return Sugerencia{}
	}
	return Sugerencia{Sugerido: math.Abs(m.Valor-montoGasto) < 0.005, Detalle: "Leído " + reportes.FormatoMonto(m.Valor)}
}

func sugerirFecha(f Campo, fechaGasto time.Time) Sugerencia {
	leida, err := time.Parse("2006-01-02", f.Valor)
	if f.Confianza == Ninguna || err != nil {
		return Sugerencia{}
	}
	return Sugerencia{Sugerido: f.Valor == fechaGasto.Format("2006-01-02"), Detalle: "Leída " + reportes.FormatoFecha(leida)}
}

func sugerirEmpresa(rucs []string, texto string) Sugerencia {
	switch {
	case slices.Contains(rucs, RucEmpresa):
		return Sugerencia{Sugerido: true, Detalle: "RUC " + RucEmpresa}
	case palabraLizar.MatchString(texto):
		return Sugerencia{Sugerido: true, Detalle: "Menciona LIZAR"}
	default:
		return Sugerencia{}
	}
}

func sugerirTipo(t Campo) Sugerencia {
	switch t.Valor {
	case "factura":
		return Sugerencia{Sugerido: true, Detalle: "Factura"}
	case "boleta":
		return Sugerencia{Sugerido: true, Detalle: "Boleta"}
	default:
		return Sugerencia{}
	}
}
```

- [ ] **Step 4: Correr las pruebas y ver que pasan**

Run: `cd gastos-service && go test ./internal/ocr -count=1 -v -run 'Sugerencias|SinDatos|Empresa|Guia'`
Expected: PASS en `TestSugerenciasCuandoTodoCoincide`, `TestSugerenciasCuandoDifiereSoloMuestranLoLeido`, `TestSinDatosNoHaySugerencias`, `TestEmpresaPorNombreYNoPorCorreo` y `TestGuiaNoSugiereTipo`.

- [ ] **Step 5: Checkpoint (sin commit)**

Run: `cd gastos-service && gofmt -l internal/ocr && go vet ./internal/ocr && go test ./internal/ocr -count=1`
Expected: sin salida de `gofmt -l`, vet limpio, `ok gastos-service/internal/ocr`.

---

### Task 3: Ruta segura y lector con pdftotext, pdftoppm y Tesseract

**Files:**
- Create: `gastos-service/internal/ocr/ruta.go`
- Create: `gastos-service/internal/ocr/lector.go`
- Test: `gastos-service/internal/ocr/ruta_test.go`
- Test: `gastos-service/internal/ocr/lector_test.go` (programas falsos escritos en `sh` en un directorio temporal)
- Test: `gastos-service/internal/ocr/herramientas_test.go` (herramientas reales; cada prueba se omite si su herramienta falta)

**Interfaces:**
- Consumes: `ocr.Normalizar`, `ocr.Alfanumericos`, `ocr.MinAlfanumericos`, `ocr.Extraer`, `ocr.RucEmpresa` (Tareas 1 y 2); `github.com/signintech/gopdf` (ya en `go.mod`, solo en pruebas, para generar PDF de muestra) y la fuente `gastos-service/internal/reportes/fuentes/overpass-400.ttf`.
- Produces:
  - `ocr.ErrRutaFuera` y `ocr.RutaSegura(uploadsDir, archivoURL string) (string, error)`: `archivoURL` debe ser `/uploads/<nombre>`; devuelve la ruta real (enlaces resueltos) o `ErrRutaFuera` si sale de la carpeta; otro error si no existe o no es un archivo regular.
  - Estados `ocr.Leido = "leido"`, `ocr.SinTexto = "sin_texto"`, `ocr.NoDisponible = "no_disponible"`, `ocr.Error = "error"`.
  - `ocr.Lectura{Estado string; Texto string}` (texto ya normalizado).
  - `type ocr.Lector interface { Leer(ctx context.Context, ruta, tipoArchivo string) ocr.Lectura }` (`tipoArchivo` es `pdf` o `imagen`).
  - `ocr.Herramientas{Pdftotext, Pdftoppm, Tesseract string; PorEjecucion, Total time.Duration}` que implementa `Lector`; `ocr.NuevoLector() *ocr.Herramientas` (busca en el PATH; 20 s y 30 s); `(*Herramientas).Resumen() string` → `pdftotext sí, pdftoppm sí, tesseract (spa) no`.

- [ ] **Step 1: Escribir las pruebas de la ruta**

`gastos-service/internal/ocr/ruta_test.go`:

```go
package ocr

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRutaSeguraDentroDeUploads(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "gasto-1.pdf"), []byte("%PDF"), 0o644); err != nil {
		t.Fatal(err)
	}
	ruta, err := RutaSegura(dir, "/uploads/gasto-1.pdf")
	if err != nil {
		t.Fatal(err)
	}
	real, _ := filepath.EvalSymlinks(filepath.Join(dir, "gasto-1.pdf"))
	if ruta != real {
		t.Fatalf("ruta = %q, se esperaba %q", ruta, real)
	}
}

func TestRutaSeguraRechazaLoQueSaleDeUploads(t *testing.T) {
	raiz := t.TempDir()
	dir := filepath.Join(raiz, "uploads")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	secreto := filepath.Join(raiz, "secreto.txt")
	if err := os.WriteFile(secreto, []byte("clave"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secreto, filepath.Join(dir, "enlace.pdf")); err != nil {
		t.Fatal(err)
	}
	for _, url := range []string{
		"/uploads/../secreto.txt",
		"/uploads/../../../../etc/passwd",
		"/uploads/enlace.pdf",
		"/uploads/",
		"/uploads/..",
		"/otra/secreto.txt",
		secreto,
		"http://otro.servidor/uploads/x.pdf",
	} {
		if _, err := RutaSegura(dir, url); !errors.Is(err, ErrRutaFuera) {
			t.Errorf("%q: se esperaba ErrRutaFuera, llegó %v", url, err)
		}
	}
}

func TestRutaSeguraArchivoInexistenteOCarpeta(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "carpeta"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, url := range []string{"/uploads/no-existe.pdf", "/uploads/carpeta"} {
		if _, err := RutaSegura(dir, url); err == nil || errors.Is(err, ErrRutaFuera) {
			t.Errorf("%q: se esperaba un error de archivo, llegó %v", url, err)
		}
	}
}
```

- [ ] **Step 2: Escribir las pruebas del lector con programas falsos**

`gastos-service/internal/ocr/lector_test.go`:

```go
package ocr

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// programa escribe un ejecutable de prueba (sh) que hace las veces de pdftotext, pdftoppm o tesseract.
func programa(t *testing.T, nombre, cuerpo string) string {
	t.Helper()
	ruta := filepath.Join(t.TempDir(), nombre)
	if err := os.WriteFile(ruta, []byte("#!/bin/sh\n"+cuerpo+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return ruta
}

func herramientas() *Herramientas {
	return &Herramientas{PorEjecucion: 5 * time.Second, Total: 10 * time.Second}
}

func TestSinHerramientasNoDisponible(t *testing.T) {
	h := herramientas()
	for _, tipo := range []string{"imagen", "pdf"} {
		if l := h.Leer(t.Context(), "/no/importa", tipo); l.Estado != NoDisponible {
			t.Errorf("%s sin herramientas: %+v", tipo, l)
		}
	}
}

func TestPDFConTextoSeLeeConPdftotextYSeNormaliza(t *testing.T) {
	h := herramientas()
	h.Pdftotext = programa(t, "pdftotext", `[ "$1" = "-layout" ] && [ "$3" = "-" ] || exit 9
printf '%s\n' "FACTURA ELECTRÓNICA F001-77" "R.U.C.   20481608750" "FECHA: 30/09/2026" "TOTAL S/ 85.5O"`)
	l := h.Leer(t.Context(), "/tmp/x.pdf", "pdf")
	if l.Estado != Leido || l.Texto != "FACTURA ELECTRÓNICA F001-77\nR.U.C. 20481608750\nFECHA: 30/09/2026\nTOTAL S/ 85.50" {
		t.Fatalf("lectura = %+v", l)
	}
}

func TestLosNombresDeArchivoNoPasanPorUnaShell(t *testing.T) {
	h := herramientas()
	h.Pdftotext = programa(t, "pdftotext", `printf 'ARCHIVO RECIBIDO: %s\n' "$2"`)
	raro := "/tmp/comprobante; rm -rf ~ $(id) `uname`.pdf"
	l := h.Leer(t.Context(), raro, "pdf")
	if l.Estado != Leido || !strings.Contains(l.Texto, "comprobante; rm -rf ~ $(id) `uname`.pdf") {
		t.Fatalf("el nombre debía llegar intacto como un solo argumento: %+v", l)
	}
}

func TestPDFEscaneadoSinPdftoppmQuedaSinTexto(t *testing.T) {
	h := herramientas()
	h.Pdftotext = programa(t, "pdftotext", `printf '  \f\n'`)
	h.Tesseract = programa(t, "tesseract", `echo nunca`)
	if l := h.Leer(t.Context(), "/tmp/x.pdf", "pdf"); l.Estado != SinTexto {
		t.Fatalf("lectura = %+v", l)
	}
}

func TestPDFEscaneadoSinTesseractNoDisponible(t *testing.T) {
	h := herramientas()
	h.Pdftotext = programa(t, "pdftotext", `printf 'p. 1\n'`)
	h.Pdftoppm = programa(t, "pdftoppm", `exit 0`)
	if l := h.Leer(t.Context(), "/tmp/x.pdf", "pdf"); l.Estado != NoDisponible {
		t.Fatalf("lectura = %+v", l)
	}
}

func TestPDFEscaneadoSeLeePaginaPorPaginaConTesseract(t *testing.T) {
	h := herramientas()
	h.Pdftotext = programa(t, "pdftotext", `exit 0`)
	// pdftoppm -r 200 -png -f 1 -l 2 <pdf> <prefijo>: crea <prefijo>-1.png y <prefijo>-2.png
	h.Pdftoppm = programa(t, "pdftoppm", `[ "$1 $2 $3 $4 $5 $6 $7" = "-r 200 -png -f 1 -l 2" ] || exit 9
: > "$9-1.png"; : > "$9-2.png"`)
	h.Tesseract = programa(t, "tesseract", `[ "$2 $3 $4 $5 $6" = "stdout -l spa --psm 6" ] || exit 9
echo "PAGINA $(basename "$1") FACTURA ELECTRONICA TOTAL S/ 85.50"`)
	l := h.Leer(t.Context(), "/tmp/x.pdf", "pdf")
	want := "PAGINA pagina-1.png FACTURA ELECTRONICA TOTAL S/ 85.50\nPAGINA pagina-2.png FACTURA ELECTRONICA TOTAL S/ 85.50"
	if l.Estado != Leido || l.Texto != want {
		t.Fatalf("lectura = %+v", l)
	}
}

func TestImagenConTesseract(t *testing.T) {
	h := herramientas()
	h.Tesseract = programa(t, "tesseract", `printf '%s\n' "BOLETA DE VENTA" "IMPORTE TOTAL S/ 12.00" "Gracias"`)
	if l := h.Leer(t.Context(), "/tmp/foto.jpg", "imagen"); l.Estado != Leido || !strings.HasPrefix(l.Texto, "BOLETA DE VENTA") {
		t.Fatalf("lectura = %+v", l)
	}
	h.Tesseract = programa(t, "tesseract", `printf '|| ~~\n'`)
	if l := h.Leer(t.Context(), "/tmp/foto.jpg", "imagen"); l.Estado != SinTexto {
		t.Fatalf("foto sin texto: %+v", l)
	}
}

func TestProgramaQueFallaDaError(t *testing.T) {
	h := herramientas()
	h.Pdftotext = programa(t, "pdftotext", `echo "Syntax Error: Couldn't find trailer dictionary" >&2; exit 1`)
	if l := h.Leer(t.Context(), "/tmp/roto.pdf", "pdf"); l.Estado != Error {
		t.Fatalf("lectura = %+v", l)
	}
}

func TestCadaEjecucionTieneTiempoLimite(t *testing.T) {
	h := herramientas()
	h.PorEjecucion = 200 * time.Millisecond
	h.Pdftotext = programa(t, "pdftotext", `exec sleep 5`)
	inicio := time.Now()
	if l := h.Leer(t.Context(), "/tmp/x.pdf", "pdf"); l.Estado != Error {
		t.Fatalf("lectura = %+v", l)
	}
	if d := time.Since(inicio); d > 3*time.Second {
		t.Fatalf("tardó %v: el tiempo límite no cortó la ejecución", d)
	}
}

func TestLaLecturaCompletaTieneTiempoLimite(t *testing.T) {
	h := herramientas()
	h.Total = 300 * time.Millisecond
	h.Pdftotext = programa(t, "pdftotext", `exit 0`)
	h.Pdftoppm = programa(t, "pdftoppm", `: > "$9-1.png"; : > "$9-2.png"`)
	// Cada página tarda menos que el límite por ejecución, pero las dos juntas pasan el total
	h.Tesseract = programa(t, "tesseract", `sleep 0.2; echo FACTURA ELECTRONICA TOTAL S/ 85.50`)
	if l := h.Leer(t.Context(), "/tmp/x.pdf", "pdf"); l.Estado != Error {
		t.Fatalf("lectura = %+v", l)
	}
}

func TestLecturaCanceladaDaError(t *testing.T) {
	h := herramientas()
	h.Tesseract = programa(t, "tesseract", `exec sleep 5`)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if l := h.Leer(ctx, "/tmp/foto.jpg", "imagen"); l.Estado != Error {
		t.Fatalf("lectura = %+v", l)
	}
}

func TestTesseractSoloCuentaConEspanol(t *testing.T) {
	h := herramientas()
	con := programa(t, "tesseract", `printf 'List of available languages in "/opt/tessdata/" (2):\neng\nspa\n' >&2`)
	sin := programa(t, "tesseract", `printf 'List of available languages (1):\neng\n'`)
	if !h.tieneEspanol(con) {
		t.Error("con spa debía contar")
	}
	if h.tieneEspanol(sin) {
		t.Error("sin spa no debía contar")
	}
}

func TestResumen(t *testing.T) {
	h := &Herramientas{Pdftotext: "/bin/pdftotext", Pdftoppm: "/bin/pdftoppm"}
	if got := h.Resumen(); got != "pdftotext sí, pdftoppm sí, tesseract (spa) no" {
		t.Fatalf("Resumen = %q", got)
	}
}
```

- [ ] **Step 3: Escribir las pruebas con las herramientas reales**

`gastos-service/internal/ocr/herramientas_test.go`:

```go
package ocr

import (
	"image"
	"image/color"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"

	"github.com/signintech/gopdf"
)

// Pruebas con las herramientas reales: cada una se omite si su herramienta no está instalada.

func requerir(t *testing.T, h *Herramientas, pdftotext, pdftoppm, tesseract bool) {
	t.Helper()
	for _, falta := range []struct {
		pide   bool
		ruta   string
		nombre string
	}{{pdftotext, h.Pdftotext, "pdftotext"}, {pdftoppm, h.Pdftoppm, "pdftoppm"}, {tesseract, h.Tesseract, "tesseract con idioma spa"}} {
		if falta.pide && falta.ruta == "" {
			t.Skip(falta.nombre + " no está instalado: se omite la prueba")
		}
	}
}

var lineasFactura = []string{
	"FACTURA ELECTRÓNICA F001-000777",
	"R.U.C. 20481608750",
	"FECHA DE EMISIÓN: 30/09/2026",
	"IMPORTE TOTAL S/ 85.50",
}

// pdfDe arma un PDF A4 con texto grande (o, si img no es nil, solo con esa imagen: un «escaneado»).
func pdfDe(t *testing.T, lineas []string, img image.Image) string {
	t.Helper()
	pdf := &gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
	if err := pdf.AddTTFFont("overpass", filepath.Join("..", "reportes", "fuentes", "overpass-400.ttf")); err != nil {
		t.Fatal(err)
	}
	pdf.AddPage()
	if img != nil {
		if err := pdf.ImageFrom(img, 0, 0, &gopdf.Rect{W: 595, H: 842}); err != nil {
			t.Fatal(err)
		}
	}
	if err := pdf.SetFont("overpass", "", 22); err != nil {
		t.Fatal(err)
	}
	for i, l := range lineas {
		pdf.SetXY(40, 60+float64(i)*40)
		if err := pdf.Text(l); err != nil {
			t.Fatal(err)
		}
	}
	ruta := filepath.Join(t.TempDir(), "comprobante.pdf")
	if err := pdf.WritePdf(ruta); err != nil {
		t.Fatal(err)
	}
	return ruta
}

func blanca() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 400, 560))
	for x := range 400 {
		for y := range 560 {
			img.Set(x, y, color.White)
		}
	}
	return img
}

func comprobarFactura(t *testing.T, l Lectura) {
	t.Helper()
	if l.Estado != Leido {
		t.Fatalf("lectura = %+v", l)
	}
	d := Extraer(l.Texto)
	if d.Monto.Valor != 85.5 || !slices.Contains(d.Rucs, RucEmpresa) || d.Tipo.Valor != "factura" || d.Fecha.Valor != "2026-09-30" {
		t.Fatalf("datos = %+v (texto %q)", d, l.Texto)
	}
}

func TestPdftotextConUnPDFGenerado(t *testing.T) {
	h := NuevoLector()
	requerir(t, h, true, false, false)
	comprobarFactura(t, h.Leer(t.Context(), pdfDe(t, lineasFactura, nil), "pdf"))
}

func TestPdftotextConLasMuestrasDeUploads(t *testing.T) {
	h := NuevoLector()
	requerir(t, h, true, false, false)
	muestras, _ := filepath.Glob(filepath.Join("..", "..", "uploads", "*.pdf"))
	if len(muestras) == 0 {
		t.Skip("no hay PDF de muestra en uploads/")
	}
	for _, m := range muestras {
		if l := h.Leer(t.Context(), m, "pdf"); l.Estado == Error || l.Estado == NoDisponible {
			t.Errorf("%s: %s", filepath.Base(m), l.Estado)
		}
	}
}

func TestPDFEscaneadoRealSinTesseract(t *testing.T) {
	h := NuevoLector()
	requerir(t, h, true, true, false)
	h.Tesseract = ""
	escaneado := pdfDe(t, nil, blanca())
	if l := h.Leer(t.Context(), escaneado, "pdf"); l.Estado != NoDisponible {
		t.Fatalf("sin tesseract: %+v", l)
	}
	h.Pdftoppm = ""
	if l := h.Leer(t.Context(), escaneado, "pdf"); l.Estado != SinTexto {
		t.Fatalf("sin pdftoppm: %+v", l)
	}
}

func TestTesseractConUnaImagenGenerada(t *testing.T) {
	h := NuevoLector()
	requerir(t, h, false, true, true)
	dir := t.TempDir()
	salida, err := exec.Command(h.Pdftoppm, "-r", "200", "-png", "-singlefile", pdfDe(t, lineasFactura, nil), filepath.Join(dir, "foto")).CombinedOutput()
	if err != nil {
		t.Fatalf("pdftoppm: %v %s", err, salida)
	}
	comprobarFactura(t, h.Leer(t.Context(), filepath.Join(dir, "foto.png"), "imagen"))
}
```

- [ ] **Step 4: Correr las pruebas y ver que fallan**

Run: `cd gastos-service && go test ./internal/ocr -count=1`
Expected: FAIL de compilación: `undefined: RutaSegura`, `undefined: ErrRutaFuera`, `undefined: Herramientas`, `undefined: Lectura`, `undefined: NuevoLector`…

- [ ] **Step 5: Implementar la ruta segura**

`gastos-service/internal/ocr/ruta.go`:

```go
package ocr

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrRutaFuera: el archivo del comprobante no está dentro de UPLOADS_DIR.
var ErrRutaFuera = errors.New("el archivo del comprobante está fuera de la carpeta de comprobantes")

// RutaSegura convierte el archivo_url de un comprobante («/uploads/<nombre>») en la ruta absoluta del
// archivo, y la rechaza si sale de uploadsDir (con «..» o con un enlace simbólico) o si no es un
// archivo regular.
func RutaSegura(uploadsDir, archivoURL string) (string, error) {
	nombre, ok := strings.CutPrefix(archivoURL, "/uploads/")
	if !ok || nombre == "" {
		return "", ErrRutaFuera
	}
	base, err := filepath.Abs(uploadsDir)
	if err != nil {
		return "", err
	}
	if real, err := filepath.EvalSymlinks(base); err == nil {
		base = real
	}
	ruta := filepath.Join(base, filepath.FromSlash(nombre))
	if !dentro(base, ruta) {
		return "", ErrRutaFuera
	}
	real, err := filepath.EvalSymlinks(ruta)
	if err != nil {
		return "", fmt.Errorf("archivo del comprobante: %w", err)
	}
	if !dentro(base, real) {
		return "", ErrRutaFuera
	}
	info, err := os.Stat(real)
	if err != nil {
		return "", fmt.Errorf("archivo del comprobante: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("archivo del comprobante: %s no es un archivo", nombre)
	}
	return real, nil
}

func dentro(base, ruta string) bool {
	rel, err := filepath.Rel(base, ruta)
	return err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
```

- [ ] **Step 6: Implementar el lector**

`gastos-service/internal/ocr/lector.go`:

```go
package ocr

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Estados de una lectura. Solo leido y sin_texto se guardan; no_disponible y error se reintentan.
const (
	Leido        = "leido"
	SinTexto     = "sin_texto"
	NoDisponible = "no_disponible"
	Error        = "error"
)

// Lectura es el resultado de leer un archivo: el estado y el texto ya normalizado.
type Lectura struct {
	Estado string
	Texto  string
}

// Lector lee el texto de un comprobante. tipoArchivo es «pdf» o «imagen» (comprobante.tipo_archivo).
type Lector interface {
	Leer(ctx context.Context, ruta, tipoArchivo string) Lectura
}

// Herramientas lee con pdftotext, pdftoppm y tesseract. Una ruta vacía es una herramienta que no está
// instalada.
type Herramientas struct {
	Pdftotext, Pdftoppm, Tesseract string
	// PorEjecucion limita cada programa; Total, la lectura completa de un comprobante.
	PorEjecucion, Total time.Duration
}

// NuevoLector busca las herramientas en el PATH (tesseract solo cuenta si tiene el idioma spa). El
// servicio arranca igual si falta alguna.
func NuevoLector() *Herramientas {
	h := &Herramientas{Pdftotext: buscar("pdftotext"), Pdftoppm: buscar("pdftoppm"), PorEjecucion: 20 * time.Second, Total: 30 * time.Second}
	if t := buscar("tesseract"); t != "" && h.tieneEspanol(t) {
		h.Tesseract = t
	}
	return h
}

func buscar(nombre string) string {
	ruta, err := exec.LookPath(nombre)
	if err != nil {
		return ""
	}
	return ruta
}

func (h *Herramientas) tieneEspanol(tesseract string) bool {
	salida, err := h.ejecutar(context.Background(), tesseract, true, "--list-langs")
	if err != nil {
		return false
	}
	for _, l := range strings.Split(salida, "\n") {
		if strings.TrimSpace(l) == "spa" {
			return true
		}
	}
	return false
}

// Resumen dice qué herramientas encontró, para el registro de arranque.
func (h *Herramientas) Resumen() string {
	si := func(ruta string) string {
		if ruta == "" {
			return "no"
		}
		return "sí"
	}
	return "pdftotext " + si(h.Pdftotext) + ", pdftoppm " + si(h.Pdftoppm) + ", tesseract (spa) " + si(h.Tesseract)
}

// ejecutar corre un programa con argumentos separados (nunca por una shell) y con tiempo límite.
// conErrores suma la salida de error (tesseract --list-langs escribe ahí en algunas versiones).
func (h *Herramientas) ejecutar(ctx context.Context, programa string, conErrores bool, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, h.PorEjecucion)
	defer cancel()
	cmd := exec.CommandContext(ctx, programa, args...)
	var salida bytes.Buffer
	cmd.Stdout = &salida
	if conErrores {
		cmd.Stderr = &salida
	}
	// Si un hijo del programa hereda la salida, no se espera más que esto tras el tiempo límite
	cmd.WaitDelay = time.Second
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return salida.String(), nil
}

// Leer lee un PDF (texto con pdftotext; si casi no tiene texto, como escaneado con pdftoppm y
// tesseract) o una imagen (tesseract), en no más de h.Total.
func (h *Herramientas) Leer(ctx context.Context, ruta, tipoArchivo string) Lectura {
	ctx, cancel := context.WithTimeout(ctx, h.Total)
	defer cancel()
	if tipoArchivo == "pdf" {
		return h.leerPDF(ctx, ruta)
	}
	if h.Tesseract == "" {
		return Lectura{Estado: NoDisponible}
	}
	texto, err := h.reconocer(ctx, ruta)
	if err != nil {
		return Lectura{Estado: Error}
	}
	return conTexto(texto)
}

func (h *Herramientas) leerPDF(ctx context.Context, ruta string) Lectura {
	texto := ""
	if h.Pdftotext != "" {
		var err error
		if texto, err = h.ejecutar(ctx, h.Pdftotext, false, "-layout", ruta, "-"); err != nil {
			return Lectura{Estado: Error}
		}
		if Alfanumericos(texto) >= MinAlfanumericos {
			return conTexto(texto)
		}
	}
	// Escaneado (o sin pdftotext): se pasa a imagen y se lee con tesseract
	switch {
	case h.Pdftoppm == "" && h.Pdftotext == "":
		return Lectura{Estado: NoDisponible}
	case h.Pdftoppm == "":
		return Lectura{Estado: SinTexto, Texto: Normalizar(texto)}
	case h.Tesseract == "":
		return Lectura{Estado: NoDisponible}
	}
	dir, err := os.MkdirTemp("", "lectura-comprobante-")
	if err != nil {
		return Lectura{Estado: Error}
	}
	defer os.RemoveAll(dir)
	if _, err := h.ejecutar(ctx, h.Pdftoppm, false, "-r", "200", "-png", "-f", "1", "-l", "2", ruta, filepath.Join(dir, "pagina")); err != nil {
		return Lectura{Estado: Error}
	}
	paginas, _ := filepath.Glob(filepath.Join(dir, "pagina*.png"))
	sort.Strings(paginas)
	partes := []string{}
	for _, p := range paginas {
		t, err := h.reconocer(ctx, p)
		if err != nil {
			return Lectura{Estado: Error}
		}
		partes = append(partes, t)
	}
	return conTexto(strings.Join(partes, "\n"))
}

func (h *Herramientas) reconocer(ctx context.Context, imagen string) (string, error) {
	return h.ejecutar(ctx, h.Tesseract, false, imagen, "stdout", "-l", "spa", "--psm", "6")
}

func conTexto(texto string) Lectura {
	texto = Normalizar(texto)
	if Alfanumericos(texto) < MinAlfanumericos {
		return Lectura{Estado: SinTexto, Texto: texto}
	}
	return Lectura{Estado: Leido, Texto: texto}
}
```

Decisiones que fija este archivo: Tesseract solo cuenta si `tesseract --list-langs` lista `spa`; un PDF con menos de 20 letras y cifras se trata como escaneado; si es escaneado y falta `pdftoppm`, queda `sin_texto`; si falta Tesseract, `no_disponible`; un PDF sin `pdftotext` ni `pdftoppm`, `no_disponible`; cualquier programa que falle o pase su tiempo, `error`. `cmd.WaitDelay` evita esperar a un hijo que herede la salida tras el tiempo límite. Las rutas que llegan aquí son absolutas (salen de `RutaSegura`), así que nunca empiezan con `-` y no se confunden con opciones.

- [ ] **Step 7: Correr las pruebas y ver que pasan**

Run: `cd gastos-service && go test ./internal/ocr -count=1 -v`
Expected: `ok gastos-service/internal/ocr`, con PASS en las pruebas de ruta y del lector, PASS en `TestPdftotextConUnPDFGenerado`, `TestPdftotextConLasMuestrasDeUploads` y `TestPDFEscaneadoRealSinTesseract`, y `SKIP: TestTesseractConUnaImagenGenerada` con «tesseract con idioma spa no está instalado: se omite la prueba» (en esta máquina no hay Tesseract). Las pruebas con programas falsos tardan unas décimas cada una por el arranque de `sh`.

- [ ] **Step 8: Checkpoint (sin commit)**

Run: `cd gastos-service && gofmt -l internal/ocr && go vet ./internal/ocr && go build ./...`
Expected: sin salida de `gofmt -l`; vet y build sin errores.

---

### Task 4: Endpoint `GET /comprobantes/{id}/lectura`, migración 0005 y cableado

**Files:**
- Create: `gastos-service/migrations/0005_lectura_comprobante.sql`
- Create: `gastos-service/internal/handlers/lectura.go`
- Modify: `gastos-service/internal/handlers/handlers.go` (import, struct `Handler`, `New`, `Register`)
- Modify: `gastos-service/main.go` (import y registro de herramientas al arrancar)
- Test: `gastos-service/internal/handlers/lectura_integration_test.go`

**Interfaces:**
- Consumes: `ocr.Lector`, `ocr.Lectura`, `ocr.NuevoLector`, `ocr.Herramientas.Resumen`, `ocr.RutaSegura`, `ocr.Extraer`, `ocr.Sugerir`, `ocr.Datos`, `ocr.Sugerencia`, los estados `ocr.Leido`/`ocr.SinTexto`/`ocr.NoDisponible`/`ocr.Error` y `ocr.Normalizar` (Tareas 1–3); `errComprobanteNoEncontrado` («comprobante no encontrado») y `errYaRevisado` («el comprobante ya fue revisado») de `internal/handlers/comprobantes.go`; en las pruebas, `entorno`, `comprobantePendiente` (gasto de S/ 85.50 del 30/09/2026 con comprobante `/uploads/prueba.jpg`), `post` y `completo()` de `checklist_integration_test.go`.
- Produces:
  - Campo `Handler.Lector ocr.Lector` (exportado: las pruebas lo reemplazan antes de `Register`).
  - `GET /comprobantes/{id}/lectura` → 200 `{"estado": "...", "datos": {"monto": {...}, "fecha": {...}, "rucs": [...], "tipo": {...}}, "sugerencias": {"monto_coincide": {"sugerido": true, "detalle": "Leído S/ 85.50"}, ...}}`; `sugerencias` es `{}` si el estado no es `leido`. Errores: 400 `id inválido`, 404 `comprobante no encontrado`, 409 `el comprobante ya fue revisado`, 500 si falla la base.

- [ ] **Step 1: Escribir las pruebas de integración que fallan**

`gastos-service/internal/handlers/lectura_integration_test.go`:

```go
package handlers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	"gastos-service/internal/config"
	"gastos-service/internal/handlers"
	"gastos-service/internal/ocr"
	"gastos-service/internal/pruebas"
)

// lectorFalso devuelve siempre la misma lectura y cuenta cuántas veces se le pidió leer.
type lectorFalso struct {
	mu      sync.Mutex
	lectura ocr.Lectura
	rutas   []string
}

func (l *lectorFalso) Leer(_ context.Context, ruta, _ string) ocr.Lectura {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.rutas = append(l.rutas, ruta)
	return l.lectura
}

func (l *lectorFalso) veces() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.rutas)
}

// textoBoleta es lo que «lee» el lector falso: coincide con el gasto de comprobantePendiente
// (S/ 85.50 del 30/09/2026).
const textoBoleta = "BOLETA DE VENTA ELECTRÓNICA B001-123\nCliente: LIZAR SRL RUC 20481608750\nFECHA: 30/09/2026\nIMPORTE TOTAL S/ 85.50"

// nuevoEntornoConLector es como nuevoEntorno, con un lector falso y el archivo del comprobante
// (/uploads/prueba.jpg) presente en la carpeta de comprobantes.
func nuevoEntornoConLector(t *testing.T, lectura ocr.Lectura) (*entorno, *lectorFalso) {
	t.Helper()
	pool := pruebas.NuevoEsquema(t)
	uploads := t.TempDir()
	if err := os.WriteFile(filepath.Join(uploads, "prueba.jpg"), []byte("jpg"), 0o644); err != nil {
		t.Fatal(err)
	}
	lector := &lectorFalso{lectura: lectura}
	h := handlers.New(pool, config.Config{UploadsDir: uploads})
	h.Lector = lector
	mux := http.NewServeMux()
	h.Register(mux)
	return &entorno{pool: pool, mux: mux}, lector
}

type lecturaJSON struct {
	Estado      string                    `json:"estado"`
	Datos       ocr.Datos                 `json:"datos"`
	Sugerencias map[string]ocr.Sugerencia `json:"sugerencias"`
}

// pedirLectura hace el GET sin cabeceras de rol: la lectura no exige ser de Logística.
func (e *entorno) pedirLectura(t *testing.T, id any) (int, lecturaJSON, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	e.mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/comprobantes/%v/lectura", id), nil))
	var cuerpo lecturaJSON
	if rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), &cuerpo); err != nil {
			t.Fatal(err)
		}
	}
	return rec.Code, cuerpo, rec.Body.String()
}

func (e *entorno) lecturasGuardadas(t *testing.T, id int) int {
	t.Helper()
	var n int
	if err := e.pool.QueryRow(context.Background(), `SELECT count(*) FROM lectura_comprobante WHERE comprobante_id = $1`, id).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestLecturaLeidaSugiereLosCuatroCriteriosYSeGuarda(t *testing.T) {
	e, lector := nuevoEntornoConLector(t, ocr.Lectura{Estado: ocr.Leido, Texto: textoBoleta})
	_, id := e.comprobantePendiente(t)
	codigo, l, cuerpo := e.pedirLectura(t, id)
	if codigo != http.StatusOK || l.Estado != "leido" {
		t.Fatalf("código %d: %s", codigo, cuerpo)
	}
	want := map[string]ocr.Sugerencia{
		"monto_coincide":   {Sugerido: true, Detalle: "Leído S/ 85.50"},
		"fecha_coincide":   {Sugerido: true, Detalle: "Leída 30/09/2026"},
		"a_nombre_empresa": {Sugerido: true, Detalle: "RUC 20481608750"},
		"tipo_valido":      {Sugerido: true, Detalle: "Boleta"},
	}
	if !reflect.DeepEqual(l.Sugerencias, want) {
		t.Fatalf("sugerencias = %+v", l.Sugerencias)
	}
	if l.Datos.Monto.Valor != 85.5 || l.Datos.Monto.Texto != "IMPORTE TOTAL S/ 85.50" || l.Datos.Tipo.Valor != "boleta" {
		t.Fatalf("datos = %+v", l.Datos)
	}
	if n := e.lecturasGuardadas(t, id); n != 1 {
		t.Fatalf("se esperaba 1 lectura guardada, hay %d", n)
	}
	if real, _ := filepath.EvalSymlinks(lector.rutas[0]); filepath.Base(real) != "prueba.jpg" {
		t.Fatalf("el lector recibió %q", lector.rutas[0])
	}
}

func TestLaLecturaGuardadaSeReusaYLasSugerenciasSiguenAlGasto(t *testing.T) {
	e, lector := nuevoEntornoConLector(t, ocr.Lectura{Estado: ocr.Leido, Texto: textoBoleta})
	gastoID, id := e.comprobantePendiente(t)
	e.pedirLectura(t, id)
	if _, err := e.pool.Exec(context.Background(), `UPDATE gasto SET monto = 90, fecha = '2026-10-01' WHERE id = $1`, gastoID); err != nil {
		t.Fatal(err)
	}
	_, l, _ := e.pedirLectura(t, id)
	if lector.veces() != 1 {
		t.Fatalf("el comprobante se leyó %d veces; la segunda consulta debía reusar la lectura", lector.veces())
	}
	if s := l.Sugerencias["monto_coincide"]; s.Sugerido || s.Detalle != "Leído S/ 85.50" {
		t.Errorf("monto tras cambiar el gasto: %+v", s)
	}
	if s := l.Sugerencias["fecha_coincide"]; s.Sugerido || s.Detalle != "Leída 30/09/2026" {
		t.Errorf("fecha tras cambiar el gasto: %+v", s)
	}
	if !l.Sugerencias["a_nombre_empresa"].Sugerido {
		t.Error("la sugerencia de empresa se recalcula desde el texto guardado")
	}
}

func TestLecturaNoDisponibleOConErrorNoSeGuardaYSeReintenta(t *testing.T) {
	for _, estado := range []string{ocr.NoDisponible, ocr.Error} {
		t.Run(estado, func(t *testing.T) {
			e, lector := nuevoEntornoConLector(t, ocr.Lectura{Estado: estado})
			_, id := e.comprobantePendiente(t)
			for range 2 {
				codigo, l, cuerpo := e.pedirLectura(t, id)
				if codigo != http.StatusOK || l.Estado != estado || len(l.Sugerencias) != 0 {
					t.Fatalf("código %d: %s", codigo, cuerpo)
				}
			}
			if n := e.lecturasGuardadas(t, id); n != 0 {
				t.Fatalf("%s no debía guardarse; hay %d", estado, n)
			}
			if lector.veces() != 2 {
				t.Fatalf("se esperaba reintentar la lectura: %d", lector.veces())
			}
		})
	}
}

func TestLecturaSinTextoSeGuardaSinSugerencias(t *testing.T) {
	e, lector := nuevoEntornoConLector(t, ocr.Lectura{Estado: ocr.SinTexto, Texto: "p. 1"})
	_, id := e.comprobantePendiente(t)
	_, l, _ := e.pedirLectura(t, id)
	_, l2, _ := e.pedirLectura(t, id)
	if l.Estado != "sin_texto" || len(l.Sugerencias) != 0 || l2.Estado != "sin_texto" {
		t.Fatalf("lecturas = %+v / %+v", l, l2)
	}
	if e.lecturasGuardadas(t, id) != 1 || lector.veces() != 1 {
		t.Fatalf("sin_texto se guarda y no se vuelve a leer (lecturas %d)", lector.veces())
	}
}

func TestLecturaRechazaIdInvalidoInexistenteYRevisado(t *testing.T) {
	e, lector := nuevoEntornoConLector(t, ocr.Lectura{Estado: ocr.Leido, Texto: textoBoleta})
	if codigo, _, _ := e.pedirLectura(t, "abc"); codigo != http.StatusBadRequest {
		t.Errorf("id inválido: %d", codigo)
	}
	if codigo, _, cuerpo := e.pedirLectura(t, 999999); codigo != http.StatusNotFound || cuerpo != "{\"error\":\"comprobante no encontrado\"}\n" {
		t.Errorf("inexistente: %d %s", codigo, cuerpo)
	}
	_, id := e.comprobantePendiente(t)
	if rec := e.post(t, fmt.Sprintf("/comprobantes/%d/aprobar", id), map[string]any{"checklist": completo()}); rec.Code != http.StatusOK {
		t.Fatalf("aprobar: %d %s", rec.Code, rec.Body)
	}
	if codigo, _, cuerpo := e.pedirLectura(t, id); codigo != http.StatusConflict || cuerpo != "{\"error\":\"el comprobante ya fue revisado\"}\n" {
		t.Errorf("revisado: %d %s", codigo, cuerpo)
	}
	if lector.veces() != 0 {
		t.Errorf("no debía leer ningún archivo: %d", lector.veces())
	}
}

func TestLecturaDeUnArchivoFueraDeUploadsDaErrorSinLeer(t *testing.T) {
	e, lector := nuevoEntornoConLector(t, ocr.Lectura{Estado: ocr.Leido, Texto: textoBoleta})
	_, id := e.comprobantePendiente(t)
	for _, url := range []string{"/uploads/../../../etc/passwd", "/uploads/no-existe.jpg"} {
		if _, err := e.pool.Exec(context.Background(), `UPDATE comprobante SET archivo_url = $1 WHERE id = $2`, url, id); err != nil {
			t.Fatal(err)
		}
		if codigo, l, cuerpo := e.pedirLectura(t, id); codigo != http.StatusOK || l.Estado != "error" || len(l.Sugerencias) != 0 {
			t.Fatalf("%s: código %d: %s", url, codigo, cuerpo)
		}
	}
	if lector.veces() != 0 || e.lecturasGuardadas(t, id) != 0 {
		t.Fatalf("no debía leer ni guardar (lecturas %d)", lector.veces())
	}
}

func TestLecturaConTextoQuePostgresNoAceptaSeGuardaIgual(t *testing.T) {
	// Un lector real devuelve el texto ya normalizado; aquí se prueba que lo normalizado se guarda.
	e, _ := nuevoEntornoConLector(t, ocr.Lectura{Estado: ocr.Leido, Texto: ocr.Normalizar("BOLETA\x00 \xff TOTAL S/ 85.50 RUC 20481608750")})
	_, id := e.comprobantePendiente(t)
	if codigo, l, cuerpo := e.pedirLectura(t, id); codigo != http.StatusOK || l.Estado != "leido" {
		t.Fatalf("código %d: %s", codigo, cuerpo)
	}
	if e.lecturasGuardadas(t, id) != 1 {
		t.Fatal("la lectura debía guardarse")
	}
}
```

- [ ] **Step 2: Correr las pruebas y ver que fallan**

Run: `cd gastos-service && TEST_DATABASE_URL="$(sed -n 's/^DATABASE_URL=//p' .env | tr -d "\"'")" go test ./internal/handlers -run Lectura -count=1`
Expected: FAIL de compilación: `h.Lector undefined (type *handlers.Handler has no field or method Lector)`.

- [ ] **Step 3: Crear la migración**

`gastos-service/migrations/0005_lectura_comprobante.sql`:

```sql
-- Lectura automática (OCR) de comprobantes: el texto y los datos leídos de cada comprobante, para no
-- leerlo dos veces. Las sugerencias del checklist no se guardan: se recalculan contra el gasto en
-- cada consulta. Solo se guardan los estados leido y sin_texto. Idempotente.

CREATE TABLE IF NOT EXISTS lectura_comprobante (
    comprobante_id INTEGER PRIMARY KEY REFERENCES comprobante (id),
    estado         VARCHAR(20) NOT NULL CHECK (estado IN ('leido', 'sin_texto', 'no_disponible', 'error')),
    texto          TEXT NOT NULL DEFAULT '',
    datos          JSONB NOT NULL DEFAULT '{}',
    creado_en      TIMESTAMP NOT NULL DEFAULT now()
);
```

- [ ] **Step 4: Implementar el endpoint**

`gastos-service/internal/handlers/lectura.go`:

```go
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"gastos-service/internal/ocr"

	"github.com/jackc/pgx/v5"
)

type respuestaLectura struct {
	Estado      string                    `json:"estado"`
	Datos       ocr.Datos                 `json:"datos"`
	Sugerencias map[string]ocr.Sugerencia `json:"sugerencias"`
}

// lecturaComprobante lee un comprobante Pendiente (o reusa su lectura guardada) y sugiere qué
// criterios del checklist cumple, comparando con el monto y la fecha actuales del gasto. Es de
// lectura: no exige el rol de Logística.
func (h *Handler) lecturaComprobante(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "id inválido")
		return
	}
	ctx := r.Context()

	var archivoURL, tipoArchivo, estado string
	var monto float64
	var fecha time.Time
	err = h.DB.QueryRow(ctx, `
		SELECT c.archivo_url, c.tipo_archivo, c.estado, g.monto::float8, g.fecha
		FROM comprobante c JOIN gasto g ON g.id = c.gasto_id
		WHERE c.id = $1`, id).Scan(&archivoURL, &tipoArchivo, &estado, &monto, &fecha)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, errComprobanteNoEncontrado.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo leer el comprobante")
		return
	}
	if estado != "Pendiente" {
		writeError(w, http.StatusConflict, errYaRevisado.Error())
		return
	}

	lectura, datos, err := h.lecturaDe(ctx, id, archivoURL, tipoArchivo)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo guardar la lectura del comprobante")
		return
	}
	resp := respuestaLectura{Estado: lectura.Estado, Datos: datos, Sugerencias: map[string]ocr.Sugerencia{}}
	if lectura.Estado == ocr.Leido {
		resp.Sugerencias = ocr.Sugerir(datos, lectura.Texto, monto, fecha)
	}
	writeJSON(w, http.StatusOK, resp)
}

// lecturaDe devuelve la lectura guardada del comprobante o, si no hay, lo lee. Guarda solo las
// lecturas definitivas (leido, sin_texto): no_disponible y error se reintentan en la próxima consulta.
func (h *Handler) lecturaDe(ctx context.Context, id int, archivoURL, tipoArchivo string) (ocr.Lectura, ocr.Datos, error) {
	var l ocr.Lectura
	var crudo []byte
	err := h.DB.QueryRow(ctx, `SELECT estado, texto, datos FROM lectura_comprobante WHERE comprobante_id = $1`, id).Scan(&l.Estado, &l.Texto, &crudo)
	if err == nil {
		datos := ocr.Extraer("")
		if err := json.Unmarshal(crudo, &datos); err != nil {
			return l, datos, err
		}
		return l, datos, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return l, ocr.Datos{}, err
	}

	ruta, err := ocr.RutaSegura(h.Cfg.UploadsDir, archivoURL)
	if err != nil {
		return ocr.Lectura{Estado: ocr.Error}, ocr.Extraer(""), nil
	}
	l = h.Lector.Leer(ctx, ruta, tipoArchivo)
	datos := ocr.Extraer(l.Texto)
	if l.Estado != ocr.Leido && l.Estado != ocr.SinTexto {
		return l, datos, nil
	}
	crudo, err = json.Marshal(datos)
	if err != nil {
		return l, datos, err
	}
	_, err = h.DB.Exec(ctx, `
		INSERT INTO lectura_comprobante (comprobante_id, estado, texto, datos) VALUES ($1, $2, $3, $4)
		ON CONFLICT (comprobante_id) DO NOTHING`, id, l.Estado, l.Texto, crudo)
	return l, datos, err
}
```

La consulta usa `r.Context()`: si quien pidió la lectura se va, los programas se cortan y el resultado (`error`) no se guarda. `ON CONFLICT DO NOTHING` cubre a dos personas que abren el mismo comprobante a la vez. `g.monto::float8` evita depender de cómo pgx escanea `NUMERIC`.

- [ ] **Step 5: Cablear el lector y la ruta en `handlers.go`**

En `gastos-service/internal/handlers/handlers.go`, agregar el import:

```go
	"gastos-service/internal/crud"
	"gastos-service/internal/ocr"
```

Reemplazar el struct y `New`:

```go
// Handler agrupa las dependencias compartidas por todos los endpoints.
type Handler struct {
	DB   *pgxpool.Pool
	Cfg  config.Config
	Auth *auth.Servicio
	// Lector lee los comprobantes para sugerir el checklist; las pruebas lo reemplazan por uno falso.
	Lector ocr.Lector
}

func New(pool *pgxpool.Pool, cfg config.Config) *Handler {
	return &Handler{DB: pool, Cfg: cfg, Auth: &auth.Servicio{DB: pool, Clave: []byte(cfg.JWTSecret)}, Lector: ocr.NuevoLector()}
}
```

Y en `Register`, justo después de la ruta de rechazar:

```go
	mux.HandleFunc("POST /comprobantes/{id}/rechazar", h.rechazarComprobante)

	// Lectura automática del comprobante para sugerir el checklist (OCR)
	mux.HandleFunc("GET /comprobantes/{id}/lectura", h.lecturaComprobante)
```

- [ ] **Step 6: Registrar las herramientas al arrancar en `main.go`**

En `gastos-service/main.go`, agregar el import `"gastos-service/internal/ocr"` después de `"gastos-service/internal/handlers"`, y reemplazar `handler := handlers.New(pool, cfg)` por:

```go
	handler := handlers.New(pool, cfg)
	if lector, ok := handler.Lector.(*ocr.Herramientas); ok {
		log.Printf("lectura de comprobantes: %s", lector.Resumen())
	}
```

- [ ] **Step 7: Correr las pruebas de integración y ver que pasan**

Run: `cd gastos-service && TEST_DATABASE_URL="$(sed -n 's/^DATABASE_URL=//p' .env | tr -d "\"'")" go test ./internal/handlers -run Lectura -count=1 -v`
Expected: PASS en `TestLecturaLeidaSugiereLosCuatroCriteriosYSeGuarda`, `TestLaLecturaGuardadaSeReusaYLasSugerenciasSiguenAlGasto`, `TestLecturaNoDisponibleOConErrorNoSeGuardaYSeReintenta` (con sus subpruebas `no_disponible` y `error`), `TestLecturaSinTextoSeGuardaSinSugerencias`, `TestLecturaRechazaIdInvalidoInexistenteYRevisado`, `TestLecturaDeUnArchivoFueraDeUploadsDaErrorSinLeer` y `TestLecturaConTextoQuePostgresNoAceptaSeGuardaIgual`.

- [ ] **Step 8: Checkpoint (sin commit)**

Run: `cd gastos-service && gofmt -l . && go vet ./... && TEST_DATABASE_URL="$(sed -n 's/^DATABASE_URL=//p' .env | tr -d "\"'")" go test ./... -count=1`
Expected: sin salida de `gofmt -l`; vet limpio; `ok` en `internal/auth`, `internal/checklist`, `internal/crud`, `internal/handlers`, `internal/ocr` y `internal/reportes` (incluido el contrato con la API de Python, que no cambia porque la ruta nueva no está bajo `/api/v1`). El gastos-service del usuario en :8081 sigue con el binario anterior hasta que lo reinicie: anótalo para el informe final.

---

### Task 5: Modelo y servicio de la lectura (Angular)

**Files:**
- Create: `frontend/src/app/core/models/lectura.ts`
- Create: `frontend/src/app/core/services/lectura.service.ts`
- Test: `frontend/src/app/core/models/lectura.spec.ts`
- Test: `frontend/src/app/core/services/lectura.service.spec.ts`

**Interfaces:**
- Consumes: `CriterioCliente` de `core/models/checklist.ts`; `GastosApiService.get<T>(path)` de `core/services/gastos-api.service.ts`.
- Produces:
  - `type EstadoLectura = 'leido' | 'sin_texto' | 'no_disponible' | 'error'`; `interface DatoLeido<T> { valor: T; texto: string; confianza: 'alta' | 'media' | 'ninguna' }`; `interface Sugerencia { sugerido: boolean; detalle: string }`; `interface LecturaComprobante { estado: EstadoLectura; datos: { monto: DatoLeido<number>; fecha: DatoLeido<string>; rucs: string[]; tipo: DatoLeido<string> }; sugerencias: Record<string, Sugerencia> }`.
  - `MSG_LECTURA_FALLIDA = 'No se pudo leer el comprobante automáticamente'`.
  - `textoSugerencia(s: Sugerencia | undefined): string` y `clavesSugeridas(criterios: CriterioCliente[], lectura: LecturaComprobante | undefined): string[]`.
  - `LECTURA_TIMEOUT_MS = 30_000` y `LecturaService.lectura(comprobanteId: number): Observable<LecturaComprobante>` (GET `/comprobantes/{id}/lectura` con `timeout(30 s)`; `HTTP_TIMEOUT_MS` de 8 s no sirve aquí y `error-messages.ts` no se toca).

- [ ] **Step 1: Escribir las pruebas que fallan**

`frontend/src/app/core/models/lectura.spec.ts`:

```ts
import { describe, expect, it } from 'vitest';
import { LecturaComprobante, clavesSugeridas, textoSugerencia } from './lectura';

const lectura = (estado: LecturaComprobante['estado'], sugerencias: LecturaComprobante['sugerencias']): LecturaComprobante => ({
  estado,
  datos: {
    monto: { valor: 0, texto: '', confianza: 'ninguna' },
    fecha: { valor: '', texto: '', confianza: 'ninguna' },
    rucs: [],
    tipo: { valor: 'otro', texto: '', confianza: 'ninguna' },
  },
  sugerencias,
});

describe('lectura del comprobante', () => {
  it('arma el texto de cada sugerencia', () => {
    expect(textoSugerencia({ sugerido: true, detalle: 'Leído S/ 85.50' })).toBe('Sugerido ✓ · Leído S/ 85.50');
    expect(textoSugerencia({ sugerido: true, detalle: '' })).toBe('Sugerido ✓');
    expect(textoSugerencia({ sugerido: false, detalle: 'Leída 29/09/2026' })).toBe('Leída 29/09/2026');
    expect(textoSugerencia({ sugerido: false, detalle: '' })).toBe('');
    expect(textoSugerencia(undefined)).toBe('');
  });

  it('sugiere solo criterios del cliente, en su orden, y solo con una lectura leída', () => {
    const criterios = [
      { clave: 'fecha_coincide', texto: 'Fecha' },
      { clave: 'legible', texto: 'Legible' },
      { clave: 'monto_coincide', texto: 'Monto' },
    ];
    const sugerencias = {
      monto_coincide: { sugerido: true, detalle: 'Leído S/ 85.50' },
      fecha_coincide: { sugerido: true, detalle: 'Leída 30/09/2026' },
      tipo_valido: { sugerido: true, detalle: 'Boleta' },
    };
    expect(clavesSugeridas(criterios, lectura('leido', sugerencias))).toEqual(['fecha_coincide', 'monto_coincide']);
    expect(clavesSugeridas(criterios, lectura('sin_texto', sugerencias))).toEqual([]);
    expect(clavesSugeridas(criterios, undefined)).toEqual([]);
  });
});
```

`frontend/src/app/core/services/lectura.service.spec.ts`:

```ts
import { afterEach, describe, expect, it, vi } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { NEVER, TimeoutError, of } from 'rxjs';
import { LecturaService } from './lectura.service';
import { GastosApiService } from './gastos-api.service';

function montar(respuesta: unknown) {
  const api = { get: vi.fn(() => respuesta) };
  TestBed.configureTestingModule({ providers: [{ provide: GastosApiService, useValue: api }] });
  return { servicio: TestBed.inject(LecturaService), api };
}

describe('LecturaService', () => {
  afterEach(() => vi.useRealTimers());

  it('pide la lectura del comprobante', () => {
    const lectura = { estado: 'no_disponible', sugerencias: {} };
    const { servicio, api } = montar(of(lectura));
    let recibido: unknown;
    servicio.lectura(9).subscribe(r => (recibido = r));
    expect(api.get).toHaveBeenCalledWith('/comprobantes/9/lectura');
    expect(recibido).toEqual(lectura);
  });

  it('espera hasta 30 s, no los 8 s de las demás operaciones', () => {
    vi.useFakeTimers();
    const { servicio } = montar(NEVER);
    let error: unknown;
    servicio.lectura(9).subscribe({ error: e => (error = e) });
    vi.advanceTimersByTime(29_999);
    expect(error).toBeUndefined();
    vi.advanceTimersByTime(1);
    expect(error).toBeInstanceOf(TimeoutError);
  });
});
```

- [ ] **Step 2: Correr las pruebas y ver que fallan**

Run: `cd frontend && npx ng test --watch=false`
Expected: FAIL al compilar los dos archivos nuevos: no se encuentran `./lectura` ni `./lectura.service`. Las demás pruebas siguen en verde.

- [ ] **Step 3: Implementar el modelo**

`frontend/src/app/core/models/lectura.ts`:

```ts
import { CriterioCliente } from './checklist';

/** leido = hay sugerencias; los demás estados se muestran como un aviso y la revisión sigue igual. */
export type EstadoLectura = 'leido' | 'sin_texto' | 'no_disponible' | 'error';

export interface DatoLeido<T> {
  valor: T;
  /** La línea del comprobante donde se leyó */
  texto: string;
  confianza: 'alta' | 'media' | 'ninguna';
}

/** El sistema solo sugiere ✓; el detalle dice lo leído, también cuando no coincide con el gasto. */
export interface Sugerencia {
  sugerido: boolean;
  detalle: string;
}

/** Lo que devuelve GET /comprobantes/{id}/lectura. */
export interface LecturaComprobante {
  estado: EstadoLectura;
  datos: { monto: DatoLeido<number>; fecha: DatoLeido<string>; rucs: string[]; tipo: DatoLeido<string> };
  /** Por clave de criterio; vacío si el estado no es leido */
  sugerencias: Record<string, Sugerencia>;
}

export const MSG_LECTURA_FALLIDA = 'No se pudo leer el comprobante automáticamente';

/** «Sugerido ✓ · Leído S/ 85.50», solo «Sugerido ✓», solo lo leído si no coincide, o vacío. */
export function textoSugerencia(s: Sugerencia | undefined): string {
  if (!s) return '';
  if (s.sugerido) return s.detalle ? `Sugerido ✓ · ${s.detalle}` : 'Sugerido ✓';
  return s.detalle;
}

/** Claves que la lectura sugiere marcar, solo entre los criterios del cliente y en su orden. */
export function clavesSugeridas(criterios: CriterioCliente[], lectura: LecturaComprobante | undefined): string[] {
  if (lectura?.estado !== 'leido') return [];
  return criterios.filter(c => lectura.sugerencias[c.clave]?.sugerido).map(c => c.clave);
}
```

- [ ] **Step 4: Implementar el servicio**

`frontend/src/app/core/services/lectura.service.ts`:

```ts
import { Injectable, inject } from '@angular/core';
import { Observable, timeout } from 'rxjs';
import { LecturaComprobante } from '../models/lectura';
import { GastosApiService } from './gastos-api.service';

/** Leer un comprobante puede tardar (OCR): el servidor corta a los 30 s y aquí se espera lo mismo. */
export const LECTURA_TIMEOUT_MS = 30_000;

/** Lectura automática (OCR) del comprobante pendiente, para sugerir el checklist. */
@Injectable({ providedIn: 'root' })
export class LecturaService {
  private readonly api = inject(GastosApiService);

  lectura(comprobanteId: number): Observable<LecturaComprobante> {
    return this.api.get<LecturaComprobante>(`/comprobantes/${comprobanteId}/lectura`).pipe(timeout(LECTURA_TIMEOUT_MS));
  }
}
```

- [ ] **Step 5: Correr las pruebas y ver que pasan**

Run: `cd frontend && npx ng test --watch=false`
Expected: PASS en todas, incluidas «arma el texto de cada sugerencia», «sugiere solo criterios del cliente, en su orden, y solo con una lectura leída», «pide la lectura del comprobante» y «espera hasta 30 s, no los 8 s de las demás operaciones».

- [ ] **Step 6: Checkpoint (sin commit)**

Run: `cd frontend && npx ng build`
Expected: build de producción sin errores ni avisos de presupuesto.

---

### Task 6: Sugerencias en la revisión del comprobante (Angular)

**Files:**
- Modify: `frontend/src/app/features/gastos/revision-comprobante/revision-comprobante.component.ts`
- Modify: `frontend/src/app/features/gastos/revision-comprobante/revision-comprobante.component.html`
- Modify: `frontend/src/styles/gastos.css` (después de la regla `.checklist__conteo`)
- Test: `frontend/src/app/features/gastos/revision-comprobante/revision-comprobante.component.spec.ts`

**Interfaces:**
- Consumes: `LecturaService.lectura`, `LecturaComprobante`, `MSG_LECTURA_FALLIDA`, `textoSugerencia`, `clavesSugeridas` (Tarea 5); `mensajeError`, `MSG_ERROR_RED`, `MSG_TIMEOUT` de `error-messages.ts` (sin modificarlo); `motivoDesdeChecklist` y el resto de `core/models/checklist.ts`.
- Produces: en la plantilla, `.checklist__lectura` (bajo la leyenda «Checklist documental»: «Leyendo comprobante…», el aviso `.checklist__aviso` o el botón «Aplicar sugerencias») y `.checklist__sugerencia` (con `.checklist__sugerencia--si` si es ✓) dentro de la etiqueta de cada criterio. Nada cambia para quien no tenga lectura: aprobar y rechazar funcionan igual.

- [ ] **Step 1: Ajustar el montaje de las pruebas y escribir las pruebas que fallan**

En `revision-comprobante.component.spec.ts`:

Cambiar el import de rxjs:

```ts
import { Observable, Subject, TimeoutError, of, throwError } from 'rxjs';
```

Agregar después del import de `ChecklistService`:

```ts
import { LecturaService } from '../../../core/services/lectura.service';
```

Reemplazar los imports de `gasto.model` y `error-messages` por:

```ts
import { Comprobante, Gasto } from '../../../core/models/gasto.model';
import { LecturaComprobante } from '../../../core/models/lectura';
import { MSG_ERROR_RED, MSG_TIMEOUT, mensajeError } from '../../../core/utils/error-messages';
```

Agregar después de la constante `TODO`:

```ts
function lecturaCon(estado: LecturaComprobante['estado'], sugerencias: LecturaComprobante['sugerencias'] = {}): LecturaComprobante {
  return {
    estado,
    datos: {
      monto: { valor: 0, texto: '', confianza: 'ninguna' },
      fecha: { valor: '', texto: '', confianza: 'ninguna' },
      rucs: [],
      tipo: { valor: 'otro', texto: '', confianza: 'ninguna' },
    },
    sugerencias,
  };
}
/** Leída: sugiere monto y empresa; la fecha leída no coincide y el tipo no se encontró */
const LEIDA = lecturaCon('leido', {
  monto_coincide: { sugerido: true, detalle: 'Leído S/ 85.50' },
  fecha_coincide: { sugerido: false, detalle: 'Leída 29/09/2026' },
  a_nombre_empresa: { sugerido: true, detalle: 'RUC 20481608750' },
  tipo_valido: { sugerido: false, detalle: '' },
});
```

En `interface Opciones`, después de `checklist?: …`:

```ts
  /** Por defecto, una lectura sin sugerencias (la revisión se ve como antes del OCR) */
  lectura?: (id: number) => Observable<LecturaComprobante>;
```

En `montar`, reemplazar desde `const checklists = …` hasta el proveedor `{ provide: ChecklistService, useValue: checklists },` (inclusive) por:

```ts
  const checklists = { deCliente: vi.fn(op.checklist ?? (() => of(ESTANDAR))) };
  const lecturas = { lectura: vi.fn(op.lectura ?? (() => of(lecturaCon('leido')))) };
  TestBed.configureTestingModule({
    providers: [
      { provide: GastosStore, useValue: store },
      { provide: ChecklistService, useValue: checklists },
      { provide: LecturaService, useValue: lecturas },
```

Y reemplazar `return { f, el, store, checklists, boton, casillas, marcar, refrescar };` por:

```ts
  const sugerencias = () =>
    Object.fromEntries(
      [...el.querySelectorAll('.checklist__lista li')]
        .filter(li => li.querySelector('.checklist__sugerencia'))
        .map(li => [casillas().indexOf(li.querySelector('input')!), li.querySelector('.checklist__sugerencia')!.textContent?.trim()]),
    );
  const lecturaTexto = () => el.querySelector('.checklist__lectura')?.textContent?.trim() ?? '';
  return { f, el, store, checklists, lecturas, boton, casillas, marcar, refrescar, sugerencias, lecturaTexto };
```

Al final, antes del `});` que cierra `describe('RevisionComprobanteComponent', …)`, agregar:

```ts
  describe('lectura automática del comprobante', () => {
    it('pide la lectura del comprobante y muestra lo sugerido y lo leído junto a cada casilla', async () => {
      const { lecturas, sugerencias, boton } = await montar({ lectura: () => of(LEIDA) });
      expect(lecturas.lectura).toHaveBeenCalledWith(9);
      // Orden estándar: legible, tipo_valido, a_nombre_empresa, monto_coincide, fecha_coincide
      expect(sugerencias()).toEqual({
        2: 'Sugerido ✓ · RUC 20481608750',
        3: 'Sugerido ✓ · Leído S/ 85.50',
        4: 'Leída 29/09/2026',
      });
      expect(boton('Aplicar sugerencias')).toBeDefined();
    });

    it('«Aplicar sugerencias» marca solo las sugeridas y no toca las demás', async () => {
      const { el, store, boton, casillas, refrescar } = await montar({ lectura: () => of(LEIDA) });
      casillas()[0].click();
      casillas()[4].click();
      await refrescar();
      boton('Aplicar sugerencias').click();
      await refrescar();
      expect(casillas().map(c => c.checked)).toEqual([true, false, true, true, true]);
      expect(el.querySelector('.checklist__conteo')?.textContent?.trim()).toBe('4 de 5 criterios');
      expect(boton('Aprobar').disabled).toBe(true);
      expect(store.aprobar).not.toHaveBeenCalled();
      casillas()[1].click();
      await refrescar();
      boton('Aprobar').click();
      expect(store.aprobar).toHaveBeenCalledWith(9, TODO);
    });

    it('con el rechazo abierto, aplicar sugerencias actualiza el motivo propuesto como un clic', async () => {
      const { el, boton, refrescar } = await montar({ lectura: () => of(LEIDA) });
      boton('Rechazar').click();
      await refrescar();
      boton('Aplicar sugerencias').click();
      await refrescar();
      expect(el.querySelector<HTMLTextAreaElement>('#motivo-rechazo')!.value).toBe(
        'No cumple: el comprobante es legible y está completo; es una boleta o factura válida; la fecha corresponde al gasto.',
      );
    });

    it('sin sugerencias no hay «Aplicar sugerencias» ni notas junto a las casillas', async () => {
      const { boton, sugerencias, lecturaTexto } = await montar();
      expect(boton('Aplicar sugerencias')).toBeUndefined();
      expect(sugerencias()).toEqual({});
      expect(lecturaTexto()).toBe('');
    });

    it('las sugerencias de criterios que el cliente no revisa no se muestran ni se aplican', async () => {
      const propio: ChecklistCliente = {
        cliente_id: 1, personalizado: true,
        criterios: [{ clave: 'legible', texto: 'Se lee completo' }, { clave: 'c-9f2a11', texto: 'Lleva sello de recepción' }],
      };
      const { boton, sugerencias } = await montar({ checklist: () => of(propio), lectura: () => of(LEIDA) });
      expect(sugerencias()).toEqual({});
      expect(boton('Aplicar sugerencias')).toBeUndefined();
    });

    it('mientras lee dice «Leyendo comprobante…» y se puede marcar y aprobar sin esperar', async () => {
      const pendiente = new Subject<LecturaComprobante>();
      const { store, boton, casillas, lecturaTexto, refrescar } = await montar({ lectura: () => pendiente, esperar: false });
      await refrescar();
      expect(lecturaTexto()).toBe('Leyendo comprobante…');
      casillas().forEach(c => c.click());
      await refrescar();
      boton('Aprobar').click();
      expect(store.aprobar).toHaveBeenCalledWith(9, TODO);
    });

    it.each(['no_disponible', 'error', 'sin_texto'] as const)('si la lectura queda «%s», avisa con discreción y todo lo demás sigue igual', async estado => {
      const { el, store, boton, marcar, sugerencias, lecturaTexto } = await montar({ lectura: () => of(lecturaCon(estado)) });
      expect(lecturaTexto()).toBe('No se pudo leer el comprobante automáticamente');
      expect(el.querySelector('.checklist__lectura [role="alert"], .checklist__lectura .alerta')).toBeNull();
      expect(sugerencias()).toEqual({});
      expect(boton('Aplicar sugerencias')).toBeUndefined();
      await marcar(5);
      boton('Aprobar').click();
      expect(store.aprobar).toHaveBeenCalledWith(9, TODO);
    });

    it('si falla la red al leer, el aviso usa el mensaje general de la app', async () => {
      const { lecturaTexto } = await montar({ lectura: () => throwError(() => new HttpErrorResponse({ status: 0 })) });
      expect(lecturaTexto()).toBe(MSG_ERROR_RED);
      TestBed.resetTestingModule();
      const lenta = await montar({ lectura: () => throwError(() => new TimeoutError()) });
      expect(lenta.lecturaTexto()).toBe(MSG_TIMEOUT);
    });

    it('con otro comprobante no quedan las sugerencias del anterior', async () => {
      const otra = new Subject<LecturaComprobante>();
      const { f, lecturas, boton, sugerencias, lecturaTexto } = await montar({
        lectura: id => (id === 9 ? of(LEIDA) : otra),
      });
      expect(boton('Aplicar sugerencias')).toBeDefined();
      f.componentRef.setInput('comprobante', { ...comprobante, id: 10 });
      f.detectChanges();
      f.detectChanges();
      expect(lecturas.lectura).toHaveBeenLastCalledWith(10);
      expect(lecturaTexto()).toBe('Leyendo comprobante…');
      expect(sugerencias()).toEqual({});
      expect(boton('Aplicar sugerencias')).toBeUndefined();
    });
  });
```

- [ ] **Step 2: Correr las pruebas y ver que fallan**

Run: `cd frontend && npx ng test --watch=false`
Expected: las 15 pruebas anteriores de `RevisionComprobanteComponent` siguen en verde; fallan las del bloque «lectura automática del comprobante» que esperan ver algo (`expected {} to deeply equal { '2': 'Sugerido ✓ · RUC 20481608750', … }`, `expected '' to be 'Leyendo comprobante…'`, `expected '' to be 'No se pudo leer el comprobante automáticamente'`, botón «Aplicar sugerencias» `undefined`), y `lecturas.lectura` no se llama.

- [ ] **Step 3: Pedir la lectura y aplicar sugerencias en el componente**

En `revision-comprobante.component.ts`:

Reemplazar las tres líneas de import de `gasto.model`, `checklist.service` y `gastos-api.service` por:

```ts
import { Comprobante, Gasto } from '../../../core/models/gasto.model';
import { MSG_LECTURA_FALLIDA, clavesSugeridas, textoSugerencia } from '../../../core/models/lectura';
import { ChecklistService } from '../../../core/services/checklist.service';
import { GastosApiService } from '../../../core/services/gastos-api.service';
import { LecturaService } from '../../../core/services/lectura.service';
```

Reemplazar el comentario de la clase por:

```ts
/**
 * Revisión del comprobante pendiente: el checklist documental del cliente y, con él, aprobar o
 * rechazar con motivo, en línea y sin ventana modal. La lectura automática del comprobante solo
 * sugiere casillas: no bloquea nada y siempre decide una persona.
 */
```

Después de `private readonly checklists = inject(ChecklistService);`:

```ts
  private readonly lecturas = inject(LecturaService);
```

Antes de `protected readonly marcados = …`:

```ts
  /** Lectura automática del comprobante (OCR); con otro comprobante se vuelve a pedir */
  protected readonly lectura = rxResource({
    params: () => this.comprobante().id,
    stream: ({ params }) => this.lecturas.lectura(params),
  });
  private readonly lecturaLista = computed(() => (this.lectura.hasValue() ? this.lectura.value() : undefined));
  protected readonly leyendo = computed(() => this.lectura.isLoading());
  /** Claves con «Sugerido ✓» entre los criterios del cliente */
  protected readonly sugeridas = computed(() => clavesSugeridas(this.criterios(), this.lecturaLista()));
  /** Aviso discreto cuando no hay lectura útil; un fallo de red usa el mensaje general de la app */
  protected readonly avisoLectura = computed(() => {
    if (this.lectura.isLoading()) return '';
    if (this.lectura.error()) return mensajeError(this.lectura.error());
    const l = this.lecturaLista();
    return l && l.estado !== 'leido' ? MSG_LECTURA_FALLIDA : '';
  });
```

Reemplazar desde `protected textoDe(criterio: CriterioCliente): string {` hasta el cierre de `marcar` (justo antes de `protected reintentar(): void {`) por lo siguiente; `marcar` pasa a usar `seguirMotivo`, y se agregan `sugerenciaDe` y `aplicarSugerencias`:

```ts
  protected textoDe(criterio: CriterioCliente): string {
    return textoConDato(criterio, this.gasto());
  }

  /** «Sugerido ✓ · Leído S/ 85.50» junto a la casilla; vacío si la lectura no dice nada de ese criterio */
  protected sugerenciaDe(clave: string): string {
    const l = this.lecturaLista();
    return l?.estado === 'leido' ? textoSugerencia(l.sugerencias[clave]) : '';
  }

  protected marcar(clave: string, cumple: boolean): void {
    this.checklist.update(c => ({ ...c, [clave]: cumple }));
    this.seguirMotivo();
  }

  /** Marca las casillas sugeridas, sin tocar las demás; el conteo y el motivo siguen como con un clic */
  protected aplicarSugerencias(): void {
    const claves = this.sugeridas();
    this.checklist.update(c => ({ ...c, ...Object.fromEntries(claves.map(clave => [clave, true])) }));
    this.seguirMotivo();
  }

  /** Con el rechazo abierto, el motivo propuesto sigue a las casillas mientras la persona no lo edite */
  private seguirMotivo(): void {
    if (this.rechazando() && this.motivo === this.motivoPropuesto) {
      this.motivo = this.motivoPropuesto = motivoDesdeChecklist(this.criterios(), this.checklist());
    }
  }
```

- [ ] **Step 4: Mostrar la lectura en la plantilla**

En `revision-comprobante.component.html`, después de `<legend class="checklist__titulo">Checklist documental</legend>`:

```html
      <div class="checklist__lectura" aria-live="polite">
        @if (leyendo()) {
          <span aria-busy="true">Leyendo comprobante…</span>
        } @else if (avisoLectura()) {
          <span class="checklist__aviso"><app-icono nombre="info-circle" [tamano]="16" />{{ avisoLectura() }}</span>
        } @else if (sugeridas().length) {
          <button type="button" class="btn btn--secundario btn--sm" (click)="aplicarSugerencias()">
            <app-icono nombre="check" [tamano]="16" />Aplicar sugerencias
          </button>
        }
      </div>
```

Y dentro de la etiqueta de cada criterio, después de `<span>{{ textoDe(criterio) }}</span>`:

```html
              @if (sugerenciaDe(criterio.clave); as nota) {
                <span class="checklist__sugerencia" [class.checklist__sugerencia--si]="sugeridas().includes(criterio.clave)">{{ nota }}</span>
              }
```

- [ ] **Step 5: Estilos**

En `frontend/src/styles/gastos.css`, después de la regla `.checklist__conteo { … }`:

```css
/* Lectura automática del comprobante: discreta; sugiere, nunca alarma */
.checklist__criterio {
  flex-wrap: wrap;
  row-gap: 0;
}

.checklist__lectura {
  display: flex;
  align-items: center;
  margin-top: var(--e-1);
  color: var(--tinta-2);
  font-size: var(--t-13);
}

.checklist__aviso {
  display: inline-flex;
  align-items: center;
  gap: var(--e-1);
}

.checklist__sugerencia {
  color: var(--tinta-2);
  font-size: var(--t-13);
  white-space: nowrap;
}

.checklist__sugerencia--si {
  color: var(--sustentado-tinta);
  font-weight: 600;
}
```

- [ ] **Step 6: Correr las pruebas y ver que pasan**

Run: `cd frontend && npx ng test --watch=false`
Expected: PASS en todas (242 en el estado probado), incluidas las 11 del bloque «lectura automática del comprobante» (la de estados cuenta 3: `no_disponible`, `error`, `sin_texto`) y las de `gasto-detalle.component.spec.ts`, que montan la revisión con el `LecturaService` real sobre un `GastosApiService` falso sin `get` (la lectura falla, se muestra el aviso y nada más cambia).

- [ ] **Step 7: Checkpoint (sin commit)**

Run: `cd frontend && npx ng test --watch=false && npx ng build`
Expected: todas las pruebas en verde y build de producción sin errores ni avisos de presupuesto.

---

### Task 7: Imagen de producción, verificación en navegador con un comprobante real y cierre

**Files:**
- Modify: `gastos-service/Dockerfile` (etapa final)
- Fuera del repositorio: `$SCRATCH/preparar.sh`, `$SCRATCH/visual/ocr-navegador.mjs`, `$SCRATCH/visual/capturas/ocr-*.png`, `$SCRATCH/limpiar.sh`

**Interfaces:**
- Consumes: todo lo anterior; un PDF de `gastos-service/uploads/` que contenga el RUC 20481608750 (las guías de muestra de LIZAR lo tienen); `db.sql` para el esquema temporal; `playwright-core` instalado en `$SCRATCH/visual` (fuera del proyecto).
- Produces: evidencia de que, con un comprobante real, se ven las sugerencias, se aplican y se aprueba; la imagen Docker pide las herramientas.

- [ ] **Step 1: Instalar las herramientas en la imagen de producción**

En `gastos-service/Dockerfile`, reemplazar la línea `RUN adduser …` de la etapa final por:

```dockerfile
# Lectura automática de comprobantes: poppler (pdftotext, pdftoppm) y Tesseract con español.
# Sin ellos el servicio arranca igual y la lectura responde «no_disponible».
RUN apk add --no-cache poppler-utils tesseract-ocr tesseract-ocr-data-spa \
    && adduser -D -u 10001 lizar && mkdir -p /app/uploads && chown lizar /app/uploads
```

Run: `command -v docker && (cd gastos-service && docker build -t gastos-service-ocr . && docker run --rm --entrypoint tesseract gastos-service-ocr --list-langs) || echo "docker no está instalado: se omite"`
Expected: en esta máquina, `docker no está instalado: se omite` (anótalo para el informe). Con Docker, la lista termina en `spa`.

- [ ] **Step 2: Preparar un esquema temporal, un comprobante real y el servicio en el 8093**

`$SCRATCH/preparar.sh` (crearlo con `mkdir -p /private/tmp/lizar-ocr/visual/capturas /private/tmp/lizar-ocr/uploads`):

```bash
#!/bin/bash
# Prepara la verificación del OCR en navegador: esquema temporal con db.sql, un comprobante real de
# muestra y un gastos-service de prueba en el 8093. No toca las tablas reales ni los puertos 8000, 8081 ni 4200.
set -euo pipefail
SCRATCH=/private/tmp/lizar-ocr
RAIZ="/Users/helcorp/Downloads/Proyectos/Universidad/Proyecto Campstone"
ESQ=prueba_ocr_navegador
URL="$(sed -n 's/^DATABASE_URL=//p' "$RAIZ/gastos-service/.env" | tr -d "\"'")"
SEP='?'
case "$URL" in *\?*) SEP='&' ;; esac
mkdir -p "$SCRATCH/uploads" "$SCRATCH/visual/capturas"

# 1. Esquema temporal con las tablas y los datos de db.sql
psql "$URL" -v ON_ERROR_STOP=1 -q -c "DROP SCHEMA IF EXISTS $ESQ CASCADE" -c "CREATE SCHEMA $ESQ"
PGOPTIONS="-c search_path=$ESQ" psql "$URL" -v ON_ERROR_STOP=1 -q -f "$RAIZ/db.sql" > /dev/null

# 2. Un comprobante real: el primer PDF de uploads/ que trae el RUC de LIZAR
rm -f "$SCRATCH/uploads/muestra.pdf"
for f in "$RAIZ"/gastos-service/uploads/*.pdf; do
  if pdftotext -layout "$f" - 2>/dev/null | grep -q 20481608750; then
    cp "$f" "$SCRATCH/uploads/muestra.pdf"
    break
  fi
done
test -f "$SCRATCH/uploads/muestra.pdf" || { echo "no hay en uploads/ un PDF con el RUC 20481608750"; exit 1; }

# 3. gastos-service de prueba en el 8093 (aplica las migraciones dentro del esquema temporal)
if lsof -nP -iTCP:8093 -sTCP:LISTEN -t > /dev/null; then echo "el puerto 8093 está ocupado"; exit 1; fi
cd "$RAIZ/gastos-service"
go build -o "$SCRATCH/gs-ocr" .
DATABASE_URL="${URL}${SEP}search_path=$ESQ" PORT=8093 UPLOADS_DIR="$SCRATCH/uploads" nohup "$SCRATCH/gs-ocr" > "$SCRATCH/gs-ocr.log" 2>&1 &
for _ in $(seq 1 30); do
  curl -sf http://localhost:8093/health > /dev/null && break
  sleep 1
done
grep "lectura de comprobantes" "$SCRATCH/gs-ocr.log"

# 4. Un gasto Pagado de S/ 85.50 con el comprobante de muestra Pendiente
read -r GASTO COMPROBANTE < <(PGOPTIONS="-c search_path=$ESQ" psql "$URL" -v ON_ERROR_STOP=1 -q -tA -F ' ' <<'SQL'
WITH g AS (
  INSERT INTO gasto (viaje_id, cliente_id, monto, fecha, estado, creado_por)
  SELECT (SELECT min(id_viaje) FROM programacionviaje), (SELECT min(id_cliente) FROM cliente), 85.50, '2026-01-01', 'Pagado', 'Prueba OCR'
  RETURNING id
)
INSERT INTO comprobante (gasto_id, archivo_url, tipo_archivo)
SELECT id, '/uploads/muestra.pdf', 'pdf' FROM g
RETURNING gasto_id, id;
SQL
)

# 5. La fecha del gasto pasa a ser la leída, para que también se sugiera (las sugerencias se recalculan)
LECTURA="http://localhost:8093/comprobantes/$COMPROBANTE/lectura"
FECHA=$(curl -s "$LECTURA" | python3 -c 'import json, sys; print(json.load(sys.stdin)["datos"]["fecha"]["valor"])')
if [ -n "$FECHA" ]; then
  PGOPTIONS="-c search_path=$ESQ" psql "$URL" -v ON_ERROR_STOP=1 -q -c "UPDATE gasto SET fecha = '$FECHA' WHERE id = $GASTO"
fi
curl -s "$LECTURA" | python3 -c '
import json, sys
d = json.load(sys.stdin)
print("estado:", d["estado"])
for clave, s in sorted(d["sugerencias"].items()):
    print(" ", clave, json.dumps(s, ensure_ascii=False))'
echo "GASTO=$GASTO COMPROBANTE=$COMPROBANTE"
```

Run: `bash /private/tmp/lizar-ocr/preparar.sh`
Expected: el registro de arranque incluye `lectura de comprobantes: pdftotext sí, pdftoppm sí, tesseract (spa) no`; la lectura muestra `estado: leido` con `a_nombre_empresa` → `{"sugerido": true, "detalle": "RUC 20481608750"}`, `tipo_valido` sin sugerencia (es una guía) y, tras igualar la fecha del gasto, `fecha_coincide` sugerido; la última línea es `GASTO=<n> COMPROBANTE=<m>`.

- [ ] **Step 3: Escribir el recorrido en navegador**

`$SCRATCH/visual/ocr-navegador.mjs` (si falta `playwright-core`: `cd /private/tmp/lizar-ocr/visual && npm init -y >/dev/null && npm install --no-audit --no-fund playwright-core@1.48.2`; queda fuera del proyecto):

```js
// Verificación del OCR en navegador con un comprobante real, contra el gastos-service de prueba (8093).
// Uso: node ocr-navegador.mjs <gasto> <comprobante>
import { chromium } from 'playwright-core';
import { mkdirSync } from 'node:fs';

const [GASTO, COMPROBANTE] = process.argv.slice(2).map(Number);
if (!GASTO || !COMPROBANTE) throw new Error('Uso: node ocr-navegador.mjs <gasto> <comprobante>');
const FRONT = 'http://localhost:4300';
const GO = 'http://localhost:8093';
const CHROME = '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
const SALIDA = new URL('./capturas/', import.meta.url).pathname;
mkdirSync(SALIDA, { recursive: true });
const USUARIO = { id_usuario: 1, username: 'prueba', email: 'prueba@lizar.pe', nombre: 'Prueba OCR', rol: 'admin', estado: true };
const al8000 = [];

const verificar = (condicion, mensaje) => {
  if (!condicion) throw new Error(`Falló: ${mensaje}`);
  console.log('OK', mensaje);
};

async function esperarServidor() {
  for (let i = 0; i < 90; i++) {
    try {
      if ((await fetch(FRONT)).ok) return;
    } catch {
      /* todavía compilando */
    }
    await new Promise(r => setTimeout(r, 2000));
  }
  throw new Error(`ng serve no respondió en ${FRONT}`);
}

async function contexto(navegador, opciones) {
  const ctx = await navegador.newContext(opciones);
  await ctx.addInitScript(([u, go]) => {
    localStorage.setItem('lizar_token', 'prueba');
    localStorage.setItem('lizar_user', JSON.stringify(u));
    localStorage.setItem('lizar_gastos_api_base', go);
  }, [USUARIO, GO]);
  // La API principal (en localhost apunta al 8081) se desvía al servicio de prueba: el 8081 del usuario no se toca
  await ctx.route(/localhost:8081/, ruta => ruta.continue({ url: ruta.request().url().replace('http://localhost:8081', GO) }));
  await ctx.route(/localhost:8000|127\.0\.0\.1:8000/, ruta => {
    al8000.push(ruta.request().url());
    return ruta.abort();
  });
  return ctx;
}

/** Abre el gasto y espera la lectura (hasta 35 s: el servidor corta a los 30). Devuelve cuántas casillas se sugieren. */
async function abrirRevision(p) {
  await p.goto(`${FRONT}/app/gastos/${GASTO}`, { waitUntil: 'networkidle' });
  await p.getByText('Sugerido ✓ · RUC 20481608750').waitFor({ timeout: 35000 });
  return p.locator('.checklist__sugerencia--si').count();
}

await esperarServidor();
const navegador = await chromium.launch({ executablePath: CHROME, headless: true });

// 1. Celular: las sugerencias se leen y nada se desborda (sin aprobar)
{
  const ctx = await contexto(navegador, { viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true, deviceScaleFactor: 2 });
  const p = await ctx.newPage();
  await abrirRevision(p);
  const desborde = await p.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
  verificar(desborde === 0, 'sin desborde horizontal en el celular');
  await p.screenshot({ path: `${SALIDA}ocr-celular.png`, fullPage: true });
  await ctx.close();
}

// 2. Escritorio: ver, aplicar, completar a mano y aprobar
{
  const ctx = await contexto(navegador, { viewport: { width: 1440, height: 900 } });
  const p = await ctx.newPage();
  const sugeridas = await abrirRevision(p);
  verificar(true, 'la revisión muestra «Sugerido ✓ · RUC 20481608750»');
  verificar(await p.getByText(/^Sugerido ✓ · Leída \d{2}\/\d{2}\/\d{4}$/).isVisible(), 'la fecha leída se sugiere');
  await p.screenshot({ path: `${SALIDA}ocr-sugerencias-escritorio.png`, fullPage: true });
  await p.getByRole('button', { name: 'Aplicar sugerencias' }).click();
  await p.getByText(`${sugeridas} de 5 criterios`).waitFor();
  verificar(sugeridas >= 2, `«Aplicar sugerencias» marca ${sugeridas} de 5`);
  await p.screenshot({ path: `${SALIDA}ocr-aplicadas-escritorio.png`, fullPage: true });
  const casillas = p.locator('.checklist input[type="checkbox"]');
  for (let i = 0; i < (await casillas.count()); i++) {
    if (!(await casillas.nth(i).isChecked())) await casillas.nth(i).check();
  }
  await p.getByText('5 de 5 criterios').waitFor();
  await p.getByRole('button', { name: 'Aprobar', exact: true }).click();
  await p.getByText('Comprobante aprobado. El gasto quedó como "Sustentado".').waitFor();
  verificar(true, 'con las demás marcadas a mano se aprueba con el copy exacto');
  await ctx.close();
}

const revisado = await fetch(`${GO}/comprobantes/${COMPROBANTE}/lectura`);
verificar(revisado.status === 409, 'la lectura del comprobante ya aprobado responde 409');
verificar(al8000.length === 0, `ninguna petición salió al 8000 y las del 8081 se desviaron al 8093${al8000.length ? ': ' + al8000[0] : ''}`);
await navegador.close();
console.log('Lectura verificada en navegador.');
```

- [ ] **Step 4: Levantar el frontend en el 4300 y ejecutar el recorrido**

En segundo plano, desde `frontend/`: `npx ng serve --port 4300` (no tocar el 4200).

Run: `cd /private/tmp/lizar-ocr/visual && node ocr-navegador.mjs <GASTO> <COMPROBANTE>` (los números de la última línea del Step 2)
Expected: líneas `OK …` para: la revisión muestra «Sugerido ✓ · RUC 20481608750»; la fecha leída se sugiere; sin desborde horizontal en el celular; «Aplicar sugerencias» marca las sugeridas (`2 de 5` con las guías de muestra: empresa y fecha); con las demás marcadas a mano se aprueba con el copy exacto; la lectura del comprobante ya aprobado responde 409; ninguna petición salió al 8000 y las del 8081 se desviaron al 8093; y `Lectura verificada en navegador.`

- [ ] **Step 5: Revisar las capturas**

Abre `ocr-celular.png`, `ocr-sugerencias-escritorio.png` y `ocr-aplicadas-escritorio.png` en `/private/tmp/lizar-ocr/visual/capturas/`. Comprueba: «Aplicar sugerencias» queda bajo la leyenda, discreto (botón secundario pequeño); cada «Sugerido ✓ · …» se lee en verde de guía y lo que no coincide en gris; en el celular la nota baja a la línea siguiente sin desbordar. Corrige lo que encuentres en un solo lote, vuelve a correr las pruebas del frontend y el recorrido una vez.

- [ ] **Step 6: Limpiar el entorno de prueba**

`$SCRATCH/limpiar.sh`:

```bash
#!/bin/bash
# Deja todo como estaba: detiene el servicio de prueba y borra el esquema temporal.
set -uo pipefail
SCRATCH=/private/tmp/lizar-ocr
RAIZ="/Users/helcorp/Downloads/Proyectos/Universidad/Proyecto Campstone"
URL="$(sed -n 's/^DATABASE_URL=//p' "$RAIZ/gastos-service/.env" | tr -d "\"'")"
PIDS=$(lsof -nP -iTCP:8093 -sTCP:LISTEN -t)
if [ -n "$PIDS" ]; then kill $PIDS; fi
psql "$URL" -c "DROP SCHEMA IF EXISTS prueba_ocr_navegador CASCADE"
psql "$URL" -tAc "SELECT count(*) FROM information_schema.schemata WHERE schema_name LIKE 'prueba_%'"
rm -rf "$SCRATCH/uploads" "$SCRATCH/gs-ocr"
```

Run: `bash /private/tmp/lizar-ocr/limpiar.sh`
Expected: `DROP SCHEMA`, y `0` esquemas `prueba_%` sobrantes. Detén también tu `ng serve` del 4300.

- [ ] **Step 7: Verificación final (sin commit)**

Run:
```bash
cd "/Users/helcorp/Downloads/Proyectos/Universidad/Proyecto Campstone/gastos-service" && gofmt -l . && go vet ./... && TEST_DATABASE_URL="$(sed -n 's/^DATABASE_URL=//p' .env | tr -d "\"'")" go test ./... -count=1
cd ../frontend && npx ng test --watch=false && npx ng build
cd .. && git diff --stat -- frontend/src/app/core/models/gasto-estados.ts frontend/src/app/core/utils/error-messages.ts gastos-service/go.mod gastos-service/go.sum frontend/package.json && git log --oneline | head -1
```
Expected: Go sin salida de gofmt, vet limpio y todo `ok`; frontend en verde y build sin avisos; `git diff --stat` de esos archivos vacío (ni copy RNF-1 ni dependencias cambiaron); `git log` sigue en `1b81dbe feat: checklist documental configurable por cliente (REQ-15, REQ-16)`.

Informa al usuario: qué se agregó; que debe **reiniciar gastos-service del 8081** para que aplique la migración 0005 y exponga la ruta nueva; que en esta máquina falta Tesseract (`brew install tesseract tesseract-lang`) y, sin él, las fotos y los PDF escaneados quedan «No se pudo leer el comprobante automáticamente» mientras los PDF con texto sí se leen; que el Dockerfile ya instala las herramientas pero no se pudo construir aquí; y que las muestras actuales son guías de LIZAR, así que la precisión con fotos de celular reales queda por probar (spec §11).

---

## Self-review

- **Cobertura de la spec:** §1 y §6 (solo ✓, nunca `legible`) → Tarea 2 y `TestSugerenciasCuandoTodoCoincide`; §3 (herramientas, PATH al iniciar, 20 s / 30 s, sin shell) → Tarea 3 y Step 6 de la Tarea 4; §4 (ruta dentro de `UPLOADS_DIR`, umbral de 20, `pdftoppm -r 200 -png -f 1 -l 2`, `tesseract -l spa --psm 6`, normalización) → Tareas 1 y 3; §5 (monto, fecha, RUC, tipo, confianza) → Tarea 1; §7 (tabla, no guardar `no_disponible`/`error`, recalcular sugerencias) → Tarea 4; §8 (respuesta y los 5 casos, sin rol) → Tarea 4; §9 (servicio con 30 s, «Leyendo comprobante…», «Sugerido ✓ · …», «Aplicar sugerencias», avisos, `mensajeError`, no bloquea) → Tareas 5 y 6; §10 (unitarias, integración, herramientas reales, Vitest, navegador) → Tareas 1–7; §11 (requisito del servidor de producción) → Step 1 de la Tarea 7.
- **Marcadores:** ningún paso remite a «similar a» ni deja código por escribir; los números de gasto y comprobante del Step 4 de la Tarea 7 los imprime el Step 2.
- **Tipos y nombres:** `Lectura{Estado, Texto}`, `Lector.Leer(ctx, ruta, tipoArchivo)`, `Sugerir(d, texto, monto, fecha)`, `Handler.Lector`, `LecturaService.lectura(id)`, `clavesSugeridas(criterios, lectura)`, `textoSugerencia(s)` y `MSG_LECTURA_FALLIDA` se usan igual en todas las tareas.
- **Review Focus:** las cinco líneas tienen su prueba en la tarea dueña del código (Tareas 1, 2, 3, 4, 5 y 6).
