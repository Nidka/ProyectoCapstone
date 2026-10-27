package notificaciones

import (
	"context"
	"testing"
	"time"

	"gastos-service/internal/db"
	"gastos-service/internal/pruebas"

	"github.com/jackc/pgx/v5/pgxpool"
)

type entorno struct {
	pool *pgxpool.Pool
	gen  *Generador
}

func nuevoEntorno(t *testing.T) *entorno {
	t.Helper()
	pool := pruebas.NuevoEsquema(t)
	return &entorno{pool: pool, gen: &Generador{DB: pool, Ahora: func() time.Time { return ahoraPrueba }}}
}

// gasto inserta un gasto registrado en `creado` (un instante) con la fecha de cobro dada (vacía = sin
// fecha) y, si se pide, un comprobante en ese estado.
func (e *entorno) gasto(t *testing.T, creado time.Time, estado, fechaCobro, comprobante string) int {
	t.Helper()
	ctx := context.Background()
	var viaje, cliente, id int
	if err := e.pool.QueryRow(ctx, `INSERT INTO programacionviaje DEFAULT VALUES RETURNING id_viaje`).Scan(&viaje); err != nil {
		t.Fatal(err)
	}
	if err := e.pool.QueryRow(ctx, `INSERT INTO cliente DEFAULT VALUES RETURNING id_cliente`).Scan(&cliente); err != nil {
		t.Fatal(err)
	}
	var cobro any
	if fechaCobro != "" {
		cobro = fechaCobro
	}
	// creado_en es TIMESTAMP sin zona: se guarda la hora local de la sesión, igual que now().
	if err := e.pool.QueryRow(ctx, `
		INSERT INTO gasto (viaje_id, cliente_id, monto, fecha, estado, creado_por, creado_en, fecha_cobro)
		VALUES ($1, $2, 85.50, '2026-09-01', $3, 'Ana Quispe', $4::timestamptz::timestamp, $5::date) RETURNING id`,
		viaje, cliente, estado, creado, cobro).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if comprobante != "" {
		if _, err := e.pool.Exec(ctx, `INSERT INTO comprobante (gasto_id, archivo_url, tipo_archivo, estado) VALUES ($1, '/uploads/x.jpg', 'imagen', $2)`, id, comprobante); err != nil {
			t.Fatal(err)
		}
	}
	return id
}

func (e *entorno) generar(t *testing.T) Resultado {
	t.Helper()
	res, err := e.gen.Generar(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func (e *entorno) avisos(t *testing.T, tipo string) []int {
	t.Helper()
	filas, err := e.pool.Query(context.Background(), `SELECT id_gasto FROM notificacion WHERE tipo = $1 ORDER BY id_gasto`, tipo)
	if err != nil {
		t.Fatal(err)
	}
	defer filas.Close()
	var out []int
	for filas.Next() {
		var id int
		if err := filas.Scan(&id); err != nil {
			t.Fatal(err)
		}
		out = append(out, id)
	}
	return out
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

func TestSinComprobanteAvisaSoloDespuesDeTresDias(t *testing.T) {
	e := nuevoEntorno(t)
	e.gasto(t, ahoraPrueba.Add(-72*time.Hour), "Registrado", "", "") // justo 3 días: todavía no
	pasado := e.gasto(t, ahoraPrueba.Add(-72*time.Hour-time.Second), "Registrado", "", "")
	e.gasto(t, ahoraPrueba.Add(-time.Hour), "Registrado", "", "")
	res := e.generar(t)
	if res.SinComprobante != 1 || res.CobroProximo != 0 {
		t.Errorf("resultado = %+v", res)
	}
	if got := e.avisos(t, TipoSinComprobante); !igual(got, []int{pasado}) {
		t.Errorf("avisos = %v, quiero solo el gasto %d (los de 3 días exactos aún no)", got, pasado)
	}
	var mensaje string
	var leida bool
	if err := e.pool.QueryRow(context.Background(), `SELECT mensaje, leida FROM notificacion WHERE id_gasto = $1`, pasado).Scan(&mensaje, &leida); err != nil {
		t.Fatal(err)
	}
	if mensaje != MensajeSinComprobante(pasado, 85.5) || leida {
		t.Errorf("mensaje = %q, leida = %v", mensaje, leida)
	}
}

func TestSinComprobanteSegunElEstadoDelComprobante(t *testing.T) {
	e := nuevoEntorno(t)
	viejo := ahoraPrueba.Add(-10 * 24 * time.Hour)
	e.gasto(t, viejo, "Pagado", "", "Pendiente")
	e.gasto(t, viejo, "Sustentado", "", "Aprobado")
	rechazado := e.gasto(t, viejo, "Registrado", "", "Rechazado")
	sin := e.gasto(t, viejo, "Registrado", "", "")
	e.generar(t)
	if got := e.avisos(t, TipoSinComprobante); !igual(got, []int{rechazado, sin}) {
		t.Errorf("avisos = %v, quiero [%d %d]: el rechazado no cuenta como adjunto", got, rechazado, sin)
	}
}

func TestCobroProximoRespetaLosLimitesDeDosDias(t *testing.T) {
	e := nuevoEntorno(t)
	reciente := ahoraPrueba.Add(-time.Hour)
	// Hoy en Lima es el 22/10/2026.
	vencido := e.gasto(t, reciente, "Registrado", "2026-10-21", "")
	hoy := e.gasto(t, reciente, "Pagado", "2026-10-22", "Pendiente")
	dos := e.gasto(t, reciente, "Pagado", "2026-10-24", "Pendiente")
	e.gasto(t, reciente, "Pagado", "2026-10-25", "Pendiente")    // a 3 días: todavía no
	e.gasto(t, reciente, "Pagado", "", "Pendiente")              // sin fecha de cobro
	e.gasto(t, reciente, "Sustentado", "2026-10-22", "Aprobado") // ya aprobado
	res := e.generar(t)
	if res.CobroProximo != 3 || res.SinComprobante != 0 {
		t.Errorf("resultado = %+v", res)
	}
	if got := e.avisos(t, TipoCobroProximo); !igual(got, []int{vencido, hoy, dos}) {
		t.Errorf("avisos = %v, quiero [%d %d %d]: a 2 días sí, a 3 no, vencido sí, Sustentado no", got, vencido, hoy, dos)
	}
}

func TestCobroProximoUsaElDiaDeLimaYNoElDeUTC(t *testing.T) {
	e := nuevoEntorno(t)
	// 23:30 del 22/10 en Lima es 04:30 UTC del 23/10: el límite debe seguir siendo el 24/10.
	e.gen.Ahora = func() time.Time { return time.Date(2026, 10, 23, 4, 30, 0, 0, time.UTC) }
	reciente := e.gen.Ahora().Add(-time.Hour)
	dos := e.gasto(t, reciente, "Pagado", "2026-10-24", "Pendiente")
	e.gasto(t, reciente, "Pagado", "2026-10-25", "Pendiente")
	e.generar(t)
	if got := e.avisos(t, TipoCobroProximo); !igual(got, []int{dos}) {
		t.Errorf("avisos = %v, quiero [%d]", got, dos)
	}
}

func TestNoDuplicaNiBorraAvisos(t *testing.T) {
	e := nuevoEntorno(t)
	id := e.gasto(t, ahoraPrueba.Add(-5*24*time.Hour), "Registrado", "2026-10-23", "")
	// Un gasto puede tener un aviso de cada tipo, pero solo uno por tipo.
	if res := e.generar(t); res.SinComprobante != 1 || res.CobroProximo != 1 {
		t.Fatalf("primera pasada = %+v", res)
	}
	if res := e.generar(t); res != (Resultado{}) {
		t.Errorf("segunda pasada = %+v, quiero nada nuevo", res)
	}
	// Marcado como leído, tampoco vuelve a crearse.
	if _, err := e.pool.Exec(context.Background(), `UPDATE notificacion SET leida = TRUE, leida_en = now()`); err != nil {
		t.Fatal(err)
	}
	if res := e.generar(t); res != (Resultado{}) {
		t.Errorf("pasada con avisos leídos = %+v", res)
	}
	// Si la condición deja de cumplirse (se adjunta y aprueba el comprobante), los avisos se quedan.
	ctx := context.Background()
	if _, err := e.pool.Exec(ctx, `INSERT INTO comprobante (gasto_id, archivo_url, tipo_archivo, estado) VALUES ($1, '/uploads/x.jpg', 'imagen', 'Aprobado')`, id); err != nil {
		t.Fatal(err)
	}
	if _, err := e.pool.Exec(ctx, `UPDATE gasto SET estado = 'Sustentado' WHERE id = $1`, id); err != nil {
		t.Fatal(err)
	}
	e.generar(t)
	var n, noLeidas int
	_ = e.pool.QueryRow(ctx, `SELECT count(*), count(*) FILTER (WHERE NOT leida) FROM notificacion`).Scan(&n, &noLeidas)
	if n != 2 || noLeidas != 0 {
		t.Errorf("avisos = %d (sin leer %d), quiero 2 intactos", n, noLeidas)
	}
}

func TestSeAvisaAlCumplirseLaCondicionEnUnaPasadaPosterior(t *testing.T) {
	e := nuevoEntorno(t)
	id := e.gasto(t, ahoraPrueba.Add(-2*24*time.Hour), "Registrado", "", "")
	if res := e.generar(t); res != (Resultado{}) {
		t.Fatalf("a los 2 días no hay aviso: %+v", res)
	}
	// El reloj avanza 25 horas: el gasto ya tiene más de 3 días y la siguiente pasada avisa.
	e.gen.Ahora = func() time.Time { return ahoraPrueba.Add(25 * time.Hour) }
	if res := e.generar(t); res.SinComprobante != 1 {
		t.Fatalf("pasada posterior = %+v", res)
	}
	if got := e.avisos(t, TipoSinComprobante); !igual(got, []int{id}) {
		t.Errorf("avisos = %v", got)
	}
}

func TestCadaAvisoDejaSuEntradaEnLaBitacora(t *testing.T) {
	e := nuevoEntorno(t)
	id := e.gasto(t, ahoraPrueba.Add(-5*24*time.Hour), "Registrado", "", "")
	e.generar(t)
	e.generar(t)
	ctx := context.Background()
	var avisoID int64
	if err := e.pool.QueryRow(ctx, `SELECT id FROM notificacion WHERE id_gasto = $1`, id).Scan(&avisoID); err != nil {
		t.Fatal(err)
	}
	var n int
	var usuario, accion, entidad string
	var idUsuario *int
	var idEntidad int64
	var idGasto float64
	if err := e.pool.QueryRow(ctx, `
		SELECT count(*) OVER (), id_usuario, usuario, accion, entidad, id_entidad, (detalle->'despues'->>'id_gasto')::float
		FROM bitacora`).Scan(&n, &idUsuario, &usuario, &accion, &entidad, &idEntidad, &idGasto); err != nil {
		t.Fatal(err)
	}
	if n != 1 || idUsuario != nil || usuario != "Sistema" || accion != "notificacion.generar" || entidad != "notificacion" || idEntidad != avisoID || int(idGasto) != id {
		t.Errorf("bitácora: n=%d usuario=%v/%q accion=%q entidad=%q id=%d (aviso %d) gasto=%v", n, idUsuario, usuario, accion, entidad, idEntidad, avisoID, idGasto)
	}
}

// Sin bitácora no hay aviso: la pasada entera se deshace y la siguiente lo reintenta.
func TestSiFallaLaBitacoraNoQuedaNingunAviso(t *testing.T) {
	e := nuevoEntorno(t)
	e.gasto(t, ahoraPrueba.Add(-5*24*time.Hour), "Registrado", "", "")
	if _, err := e.pool.Exec(context.Background(), `DROP TABLE bitacora`); err != nil {
		t.Fatal(err)
	}
	if _, err := e.gen.Generar(context.Background()); err == nil {
		t.Fatal("esperaba un error")
	}
	var n int
	_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM notificacion`).Scan(&n)
	if n != 0 {
		t.Errorf("avisos = %d, quiero 0", n)
	}
}

// El runner reaplica todas las migraciones en cada arranque: la 0009 no debe tocar lo ya guardado.
func TestLaMigracionSePuedeReaplicar(t *testing.T) {
	e := nuevoEntorno(t)
	id := e.gasto(t, ahoraPrueba.Add(-5*24*time.Hour), "Registrado", "2026-10-23", "")
	e.generar(t)
	for i := 0; i < 3; i++ {
		if err := db.RunMigrations(e.pool, pruebas.DirectorioMigraciones(t)); err != nil {
			t.Fatal(err)
		}
	}
	var n int
	var cobro *time.Time
	ctx := context.Background()
	_ = e.pool.QueryRow(ctx, `SELECT count(*) FROM notificacion WHERE id_gasto = $1`, id).Scan(&n)
	_ = e.pool.QueryRow(ctx, `SELECT fecha_cobro FROM gasto WHERE id = $1`, id).Scan(&cobro)
	if n != 2 || cobro == nil || cobro.Format("2006-01-02") != "2026-10-23" {
		t.Errorf("tras reaplicar: %d avisos, fecha_cobro %v", n, cobro)
	}
}

func TestLaFechaDeCobroNoPuedeSerAnteriorAlGasto(t *testing.T) {
	e := nuevoEntorno(t)
	ctx := context.Background()
	var viaje, cliente int
	_ = e.pool.QueryRow(ctx, `INSERT INTO programacionviaje DEFAULT VALUES RETURNING id_viaje`).Scan(&viaje)
	_ = e.pool.QueryRow(ctx, `INSERT INTO cliente DEFAULT VALUES RETURNING id_cliente`).Scan(&cliente)
	insertar := func(cobro string) error {
		_, err := e.pool.Exec(ctx, `
			INSERT INTO gasto (viaje_id, cliente_id, monto, fecha, estado, creado_por, fecha_cobro)
			VALUES ($1, $2, 10, '2026-09-10', 'Registrado', 'x', $3::date)`, viaje, cliente, cobro)
		return err
	}
	if err := insertar("2026-09-10"); err != nil {
		t.Errorf("el mismo día debe valer: %v", err)
	}
	if err := insertar("2026-09-09"); err == nil {
		t.Error("la base aceptó una fecha de cobro anterior a la fecha del gasto")
	}
}
