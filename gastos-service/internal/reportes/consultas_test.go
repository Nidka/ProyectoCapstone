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
	want := reportes.Indicadores{Total: 400, Cantidad: 4, Promedio: 100, Sustentado: 149.75, Pendiente: 250.25, PorcentajeSustentado: 37.4, GastosSustentados: 2, ISC: 50}
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

func TestRankingSeparaHomonimos(t *testing.T) {
	pool := pruebas.NuevoEsquema(t)
	_, err := pool.Exec(context.Background(), `
		INSERT INTO cliente (id_cliente, razon_social) VALUES (1, 'Agro SAC'), (2, 'Agro SAC');
		INSERT INTO conductor (id_conductor, nombre1, apellido1) VALUES (1, 'Juan', 'Pérez'), (2, 'Juan', 'Pérez');
		INSERT INTO conductorunidad (id_conductor_unidad, id_conductor) VALUES (1, 1), (2, 2);
		INSERT INTO programacionviaje (id_viaje, id_conductor_unidad) VALUES (1, 1), (2, 2);
		INSERT INTO gasto (viaje_id, cliente_id, monto, fecha, estado, creado_por) VALUES
			(1, 1, 100, '2026-05-10', 'Registrado', 'x'),
			(2, 2,  50, '2026-05-11', 'Registrado', 'x');`)
	if err != nil {
		t.Fatal(err)
	}
	r := consultar(t, pool, "2026-05-01", "2026-05-31")
	for nombre, filas := range map[string][]reportes.FilaRanking{"cliente": r.PorCliente, "conductor": r.PorConductor} {
		if len(filas) != 2 || filas[0].Monto != 100 || filas[1].Monto != 50 {
			t.Errorf("por %s: dos homónimos debían ser dos filas: %+v", nombre, filas)
		}
	}
}

// El ISC cuenta gastos, no montos: 1 sustentado de 3 es 33.3 % aunque ese gasto valga casi todo.
func TestISCEsElPorcentajeDeGastosSustentados(t *testing.T) {
	pool := pruebas.NuevoEsquema(t)
	_, err := pool.Exec(context.Background(), `
		INSERT INTO cliente (id_cliente, razon_social) VALUES (1, 'Agro SAC');
		INSERT INTO programacionviaje (id_viaje) VALUES (1);
		INSERT INTO gasto (viaje_id, cliente_id, monto, fecha, estado, creado_por) VALUES
			(1, 1, 900, '2026-05-10', 'Sustentado', 'x'),
			(1, 1, 50,  '2026-05-11', 'Pagado', 'x'),
			(1, 1, 50,  '2026-05-12', 'Registrado', 'x');`)
	if err != nil {
		t.Fatal(err)
	}
	i := consultar(t, pool, "2026-05-01", "2026-05-31").Indicadores
	if i.GastosSustentados != 1 || i.ISC != 33.3 || i.PorcentajeSustentado != 90 {
		t.Fatalf("indicadores = %+v", i)
	}
}
