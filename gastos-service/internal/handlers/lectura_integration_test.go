package handlers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

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
	h := handlers.New(pool, configDePrueba(uploads))
	h.Lector = lector
	return armarEntorno(t, pool, h), lector
}

type lecturaJSON struct {
	Estado      string                    `json:"estado"`
	Datos       ocr.Datos                 `json:"datos"`
	Sugerencias map[string]ocr.Sugerencia `json:"sugerencias"`
}

// pedirLectura hace el GET con el token del administrador: la lectura no exige ser de Logística.
func (e *entorno) pedirLectura(t *testing.T, id any) (int, lecturaJSON, string) {
	t.Helper()
	rec := pedir(e.mux, http.MethodGet, fmt.Sprintf("/comprobantes/%v/lectura", id), e.tokenAdmin, nil)
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
