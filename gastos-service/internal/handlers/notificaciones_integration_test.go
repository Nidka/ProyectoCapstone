package handlers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gastos-service/internal/notificaciones"
)

type avisoJSON struct {
	ID       int        `json:"id"`
	Tipo     string     `json:"tipo"`
	IDGasto  int        `json:"id_gasto"`
	Mensaje  string     `json:"mensaje"`
	CreadaEn time.Time  `json:"creada_en"`
	Leida    bool       `json:"leida"`
	LeidaEn  *time.Time `json:"leida_en"`
}

type listaAvisosJSON struct {
	NoLeidas       int         `json:"no_leidas"`
	Notificaciones []avisoJSON `json:"notificaciones"`
}

// aviso inserta una notificación del gasto dado, creada en el instante dado.
func (e *entorno) aviso(t *testing.T, tipo string, gasto int, creada time.Time, leida bool) int {
	t.Helper()
	var id int
	err := e.pool.QueryRow(context.Background(), `
		INSERT INTO notificacion (tipo, id_gasto, mensaje, creada_en, leida, leida_en)
		VALUES ($1, $2, $3, $4::timestamptz, $5::boolean, CASE WHEN $5::boolean THEN $4::timestamptz END) RETURNING id`,
		tipo, gasto, fmt.Sprintf("Aviso %s del gasto %d", tipo, gasto), creada, leida).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func (e *entorno) listarAvisos(t *testing.T) listaAvisosJSON {
	t.Helper()
	rec := pedir(e.mux, "GET", "/notificaciones", e.tokenLogistica, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /notificaciones = %d %s", rec.Code, rec.Body)
	}
	var l listaAvisosJSON
	if err := json.Unmarshal(rec.Body.Bytes(), &l); err != nil {
		t.Fatal(err)
	}
	return l
}

func (e *entorno) marcarLeida(id any) *httptest.ResponseRecorder {
	return pedir(e.mux, "POST", fmt.Sprintf("/notificaciones/%v/leida", id), e.tokenLogistica, nil)
}

func TestNotificacionesExigenTokenYSonSoloDeLogistica(t *testing.T) {
	e := nuevoEntorno(t)
	gasto, _ := e.comprobantePendiente(t)
	id := e.aviso(t, notificaciones.TipoSinComprobante, gasto, time.Now(), false)
	for _, r := range []struct{ metodo, ruta string }{
		{"GET", "/notificaciones"}, {"POST", fmt.Sprintf("/notificaciones/%d/leida", id)},
	} {
		if rec := pedir(e.mux, r.metodo, r.ruta, "", nil); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s sin token = %d, quiero 401", r.metodo, r.ruta, rec.Code)
		}
		rec := pedir(e.mux, r.metodo, r.ruta, e.tokenAdmin, nil)
		if rec.Code != http.StatusForbidden {
			t.Errorf("admin %s %s = %d, quiero 403 (%s)", r.metodo, r.ruta, rec.Code, rec.Body)
		}
	}
	var leida bool
	_ = e.pool.QueryRow(context.Background(), `SELECT leida FROM notificacion WHERE id = $1`, id).Scan(&leida)
	if leida {
		t.Error("el 403 no debía marcar nada")
	}
}

func TestListarNotificacionesSinNingunaDaListaVacia(t *testing.T) {
	e := nuevoEntorno(t)
	rec := pedir(e.mux, "GET", "/notificaciones", e.tokenLogistica, nil)
	if rec.Code != http.StatusOK || rec.Body.String() != "{\"no_leidas\":0,\"notificaciones\":[]}\n" {
		t.Errorf("sin avisos: %d %q", rec.Code, rec.Body)
	}
}

func TestListarNotificacionesNoLeidasPrimeroYConElTotal(t *testing.T) {
	e := nuevoEntorno(t)
	g1, _ := e.comprobantePendiente(t)
	g2, _ := e.comprobantePendiente(t)
	g3, _ := e.comprobantePendiente(t)
	base := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	leidaNueva := e.aviso(t, notificaciones.TipoCobroProximo, g1, base.Add(5*time.Hour), true)
	pendienteVieja := e.aviso(t, notificaciones.TipoSinComprobante, g2, base, false)
	pendienteNueva := e.aviso(t, notificaciones.TipoSinComprobante, g3, base.Add(time.Hour), false)
	l := e.listarAvisos(t)
	var ids []int
	for _, a := range l.Notificaciones {
		ids = append(ids, a.ID)
	}
	if !igual(ids, []int{pendienteNueva, pendienteVieja, leidaNueva}) || l.NoLeidas != 2 {
		t.Errorf("ids = %v, no_leidas = %d; quiero [%d %d %d] y 2", ids, l.NoLeidas, pendienteNueva, pendienteVieja, leidaNueva)
	}
	a := l.Notificaciones[0]
	if a.Tipo != notificaciones.TipoSinComprobante || a.IDGasto != g3 || a.Mensaje == "" || a.Leida || a.LeidaEn != nil || !a.CreadaEn.Equal(base.Add(time.Hour)) {
		t.Errorf("primer aviso = %+v", a)
	}
	if última := l.Notificaciones[2]; !última.Leida || última.LeidaEn == nil {
		t.Errorf("el aviso leído debe traer leida_en: %+v", última)
	}
}

func igual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestMarcarNotificacionComoLeida(t *testing.T) {
	e := entornoCompleto(t)
	gasto := e.registrarGasto(t)
	id := e.aviso(t, notificaciones.TipoSinComprobante, gasto, time.Now(), false)
	antes := e.total(t)

	rec := e.marcarLeida(id)
	var a avisoJSON
	if err := json.Unmarshal(rec.Body.Bytes(), &a); err != nil || rec.Code != http.StatusOK || !a.Leida || a.LeidaEn == nil || a.ID != id {
		t.Fatalf("marcar leída: %d %s", rec.Code, rec.Body)
	}
	if l := e.listarAvisos(t); l.NoLeidas != 0 || !l.Notificaciones[0].Leida {
		t.Errorf("después de marcar: %+v", l)
	}
	// Queda en la bitácora, con el nombre de quien la leyó.
	entrada := e.unaNueva(t, antes, "notificacion.leer")
	if entrada.Usuario != "Ana Quispe" || entrada.Entidad != "notificacion" || entrada.IDEntidad == nil || *entrada.IDEntidad != int64(id) ||
		entrada.sub("antes", "leida") != false || entrada.sub("despues", "leida") != true || entrada.sub("id_gasto") != float64(gasto) {
		t.Errorf("entrada = %+v", entrada)
	}

	// Es idempotente: repetir no cambia la fecha de lectura ni agrega entradas.
	rec2 := e.marcarLeida(id)
	var b avisoJSON
	if err := json.Unmarshal(rec2.Body.Bytes(), &b); err != nil || rec2.Code != http.StatusOK || !b.Leida || b.LeidaEn == nil || !b.LeidaEn.Equal(*a.LeidaEn) {
		t.Errorf("segunda vez: %d %s", rec2.Code, rec2.Body)
	}
	if n := e.total(t); n != antes+1 {
		t.Errorf("la segunda vez agregó entradas: %d → %d", antes+1, n)
	}
}

func TestMarcarNotificacionInexistenteODeIdMalo(t *testing.T) {
	e := nuevoEntorno(t)
	if rec := e.marcarLeida(999); rec.Code != http.StatusNotFound {
		t.Errorf("inexistente = %d, quiero 404 (%s)", rec.Code, rec.Body)
	}
	for _, id := range []string{"abc", "0", "-3"} {
		if rec := e.marcarLeida(id); rec.Code != http.StatusBadRequest {
			t.Errorf("id %q = %d, quiero 400", id, rec.Code)
		}
	}
}

// Si la bitácora falla, marcar como leída falla entero y el aviso queda sin leer.
func TestSiFallaLaBitacoraNoSeMarcaLaNotificacion(t *testing.T) {
	e := nuevoEntorno(t)
	gasto, _ := e.comprobantePendiente(t)
	id := e.aviso(t, notificaciones.TipoSinComprobante, gasto, time.Now(), false)
	if _, err := e.pool.Exec(context.Background(), `ALTER TABLE bitacora RENAME TO bitacora_rota`); err != nil {
		t.Fatal(err)
	}
	if rec := e.marcarLeida(id); rec.Code != http.StatusInternalServerError {
		t.Errorf("sin bitácora = %d, quiero 500", rec.Code)
	}
	var leida bool
	_ = e.pool.QueryRow(context.Background(), `SELECT leida FROM notificacion WHERE id = $1`, id).Scan(&leida)
	if leida {
		t.Error("el aviso quedó leído aunque falló la bitácora")
	}
}

// Del gasto antiguo al aviso que ve la persona de Logística, pasando por el generador.
func TestUnGastoAtrasadoLlegaALaCampana(t *testing.T) {
	e := entornoCompleto(t)
	gasto := e.registrarGasto(t)
	if _, err := e.pool.Exec(context.Background(), `UPDATE gasto SET creado_en = now() - interval '4 days' WHERE id = $1`, gasto); err != nil {
		t.Fatal(err)
	}
	gen := &notificaciones.Generador{DB: e.pool, Ahora: time.Now}
	if res, err := gen.Generar(context.Background()); err != nil || res.SinComprobante != 1 {
		t.Fatalf("generar: %+v %v", res, err)
	}
	l := e.listarAvisos(t)
	if l.NoLeidas != 1 || len(l.Notificaciones) != 1 || l.Notificaciones[0].IDGasto != gasto || l.Notificaciones[0].Tipo != notificaciones.TipoSinComprobante {
		t.Errorf("lista = %+v", l)
	}
}

func registrarConCobro(e *entorno, cobro any, fecha any) *httptest.ResponseRecorder {
	cuerpo := map[string]any{"viaje_id": 1, "cliente_id": 1, "monto": 50}
	if cobro != nil {
		cuerpo["fecha_cobro"] = cobro
	}
	if fecha != nil {
		cuerpo["fecha"] = fecha
	}
	return pedir(e.mux, "POST", "/gastos", e.tokenLogistica, cuerpo)
}

func TestRegistrarGastoConFechaDeCobro(t *testing.T) {
	e := entornoCompleto(t)
	rec := registrarConCobro(e, "2026-12-15", "2026-12-01")
	var g struct {
		ID         int     `json:"id"`
		FechaCobro *string `json:"fecha_cobro"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &g); err != nil || rec.Code != http.StatusCreated || g.FechaCobro == nil || (*g.FechaCobro)[:10] != "2026-12-15" {
		t.Fatalf("POST con fecha_cobro: %d %s", rec.Code, rec.Body)
	}
	// Se ve igual en el detalle y en el listado; queda en la bitácora.
	for _, ruta := range []string{fmt.Sprintf("/gastos/%d", g.ID), "/gastos"} {
		if cuerpo := pedir(e.mux, "GET", ruta, e.tokenLogistica, nil).Body.String(); !strings.Contains(cuerpo, `"fecha_cobro":"2026-12-15`) {
			t.Errorf("GET %s no trae la fecha de cobro: %s", ruta, cuerpo)
		}
	}
	if a := e.unaNueva(t, 0, "gasto.registrar"); a.sub("despues", "fecha_cobro") != "2026-12-15" {
		t.Errorf("bitácora = %v", a.Detalle)
	}
}

func TestRegistrarGastoSinFechaDeCobroLaDevuelveNula(t *testing.T) {
	e := entornoCompleto(t)
	for _, cobro := range []any{nil, ""} {
		rec := registrarConCobro(e, cobro, nil)
		if rec.Code != http.StatusCreated || !strings.Contains(rec.Body.String(), `"fecha_cobro":null`) {
			t.Errorf("fecha_cobro %v: %d %s", cobro, rec.Code, rec.Body)
		}
	}
	if a := e.asientos(t)[0]; a.sub("despues", "fecha_cobro") != nil {
		t.Errorf("sin fecha de cobro la bitácora no debe llevar el campo: %v", a.Detalle)
	}
}

func TestFechaDeCobroInvalida(t *testing.T) {
	e := entornoCompleto(t)
	casos := []struct {
		nombre      string
		cobro, fech any
	}{
		{"formato", "15/12/2026", nil},
		{"no es texto", 20261215, nil},
		{"día inexistente", "2026-02-30", nil},
		{"anterior al gasto", "2026-11-30", "2026-12-01"},
		{"anterior a hoy cuando el gasto es de hoy", time.Now().AddDate(0, 0, -1).Format("2006-01-02"), nil},
	}
	for _, c := range casos {
		rec := registrarConCobro(e, c.cobro, c.fech)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: %d, quiero 400 (%s)", c.nombre, rec.Code, rec.Body)
		}
	}
	if n := e.total(t); n != 0 {
		t.Errorf("los rechazos dejaron %d entradas", n)
	}
	var n int
	_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM gasto`).Scan(&n)
	if n != 0 {
		t.Errorf("se guardaron %d gastos inválidos", n)
	}
	// El mismo día que el gasto sí vale.
	if rec := registrarConCobro(e, "2026-12-01", "2026-12-01"); rec.Code != http.StatusCreated {
		t.Errorf("mismo día = %d (%s)", rec.Code, rec.Body)
	}
}
