package bitacora_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"gastos-service/internal/bitacora"
	"gastos-service/internal/db"
	"gastos-service/internal/pruebas"
)

func TestRegistrarGuardaTodosLosCamposAlConfirmar(t *testing.T) {
	pool := pruebas.NuevoEsquema(t)
	ctx := context.Background()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	err = bitacora.Registrar(ctx, tx, bitacora.Entrada{
		Actor:     bitacora.Actor{ID: 7, Nombre: "Ana Quispe"},
		Accion:    "gasto.registrar",
		Entidad:   "gasto",
		IDEntidad: 42,
		Detalle:   map[string]any{"despues": map[string]any{"monto": 85.5}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	var idUsuario, idEntidad int
	var usuario, accion, entidad string
	var detalle []byte
	var conFecha bool
	err = pool.QueryRow(ctx, `SELECT id_usuario, usuario, accion, entidad, id_entidad, detalle, fecha IS NOT NULL FROM bitacora`).
		Scan(&idUsuario, &usuario, &accion, &entidad, &idEntidad, &detalle, &conFecha)
	if err != nil {
		t.Fatal(err)
	}
	if idUsuario != 7 || usuario != "Ana Quispe" || accion != "gasto.registrar" || entidad != "gasto" || idEntidad != 42 || !conFecha {
		t.Errorf("fila = %d %q %q %q %d %v", idUsuario, usuario, accion, entidad, idEntidad, conFecha)
	}
	var d map[string]map[string]float64
	if err := json.Unmarshal(detalle, &d); err != nil || d["despues"]["monto"] != 85.5 {
		t.Errorf("detalle = %s, %v", detalle, err)
	}
}

func TestSinUsuarioNiEntidadSeGuardanComoNulos(t *testing.T) {
	pool := pruebas.NuevoEsquema(t)
	ctx := context.Background()
	err := bitacora.Registrar(ctx, pool, bitacora.Entrada{Actor: bitacora.Actor{Nombre: "sistema"}, Accion: "usuario.crear", Entidad: "usuario"})
	if err != nil {
		t.Fatal(err)
	}
	var sinUsuario, sinEntidad bool
	var detalle string
	if err := pool.QueryRow(ctx, `SELECT id_usuario IS NULL, id_entidad IS NULL, detalle::text FROM bitacora`).Scan(&sinUsuario, &sinEntidad, &detalle); err != nil {
		t.Fatal(err)
	}
	if !sinUsuario || !sinEntidad || detalle != "{}" {
		t.Errorf("sinUsuario=%v sinEntidad=%v detalle=%s", sinUsuario, sinEntidad, detalle)
	}
}

func TestRegistrarRechazaEntradasIncompletas(t *testing.T) {
	pool := pruebas.NuevoEsquema(t)
	casos := map[string]bitacora.Entrada{
		"sin usuario": {Accion: "a.b", Entidad: "x"},
		"sin acción":  {Actor: bitacora.Actor{Nombre: "Ana"}, Entidad: "x"},
		"sin entidad": {Actor: bitacora.Actor{Nombre: "Ana"}, Accion: "a.b"},
	}
	for nombre, e := range casos {
		if err := bitacora.Registrar(context.Background(), pool, e); err == nil {
			t.Errorf("%s: debía fallar", nombre)
		}
	}
	var n int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM bitacora`).Scan(&n); err != nil || n != 0 {
		t.Errorf("filas = %d, %v", n, err)
	}
}

func TestRollbackRevierteLaBitacora(t *testing.T) {
	pool := pruebas.NuevoEsquema(t)
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if err := bitacora.Registrar(ctx, tx, bitacora.Entrada{Actor: bitacora.Actor{Nombre: "Ana"}, Accion: "gasto.registrar", Entidad: "gasto", IDEntidad: 1}); err != nil {
		t.Fatal(err)
	}
	if err := tx.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM bitacora`).Scan(&n); err != nil || n != 0 {
		t.Errorf("tras el rollback quedaron %d filas, %v", n, err)
	}
}

// RNF-9 y RNF-10: nadie modifica ni borra la bitácora, tampoco el dueño de la tabla (que es con
// quien se conecta el servicio).
func TestElTriggerRechazaUpdateDeleteYTruncate(t *testing.T) {
	pool := pruebas.NuevoEsquema(t)
	ctx := context.Background()
	if err := bitacora.Registrar(ctx, pool, bitacora.Entrada{Actor: bitacora.Actor{Nombre: "Ana"}, Accion: "gasto.registrar", Entidad: "gasto", IDEntidad: 1}); err != nil {
		t.Fatal(err)
	}
	sentencias := map[string]string{
		"UPDATE":   `UPDATE bitacora SET usuario = 'otro'`,
		"DELETE":   `DELETE FROM bitacora`,
		"TRUNCATE": `TRUNCATE bitacora`,
	}
	for nombre, sql := range sentencias {
		_, err := pool.Exec(ctx, sql)
		if err == nil || !strings.Contains(err.Error(), "bitácora") {
			t.Errorf("%s: err = %v, quiero el rechazo del trigger", nombre, err)
		}
	}
	var usuario string
	var n int
	if err := pool.QueryRow(ctx, `SELECT min(usuario), count(*) FROM bitacora`).Scan(&usuario, &n); err != nil || usuario != "Ana" || n != 1 {
		t.Errorf("la fila cambió: %q %d %v", usuario, n, err)
	}
}

// El servicio reaplica todas las migraciones en cada arranque: la 0008 no debe duplicar nada, ni
// perder filas, ni dejar la tabla sin protección.
func TestLaMigracionEsIdempotenteYConservaLaProteccion(t *testing.T) {
	pool := pruebas.NuevoEsquema(t)
	ctx := context.Background()
	if err := bitacora.Registrar(ctx, pool, bitacora.Entrada{Actor: bitacora.Actor{Nombre: "Ana"}, Accion: "gasto.registrar", Entidad: "gasto", IDEntidad: 1}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := db.RunMigrations(pool, pruebas.DirectorioMigraciones(t)); err != nil {
			t.Fatalf("reaplicar migraciones (vuelta %d): %v", i+1, err)
		}
	}
	var filas, triggers int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM bitacora`).Scan(&filas); err != nil || filas != 1 {
		t.Errorf("filas = %d, %v", filas, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM pg_trigger WHERE tgrelid = 'bitacora'::regclass AND NOT tgisinternal`).Scan(&triggers); err != nil || triggers != 2 {
		t.Errorf("triggers = %d, %v (quiero 2: filas y TRUNCATE)", triggers, err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM bitacora`); err == nil {
		t.Error("tras reaplicar, el DELETE debía seguir rechazado")
	}
}
