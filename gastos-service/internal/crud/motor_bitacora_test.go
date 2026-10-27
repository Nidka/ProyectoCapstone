package crud_test

import (
	"context"
	"errors"
	"testing"

	"gastos-service/internal/bitacora"
	"gastos-service/internal/crud"
	"gastos-service/internal/pruebas"
)

func rubros() *crud.Recurso {
	for i := range crud.Recursos {
		if crud.Recursos[i].Ruta == "rubros" {
			return &crud.Recursos[i]
		}
	}
	return nil
}

func contar(t *testing.T, m *crud.Motor, tabla string) int {
	t.Helper()
	var n int
	if err := m.DB.QueryRow(context.Background(), "SELECT count(*) FROM "+tabla).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// Un motor que audita pero no sabe quién hace el cambio no cambia nada.
func TestMotorSinAutorNoCambiaNada(t *testing.T) {
	pool := pruebas.NuevoEsquemaCompleto(t)
	m := &crud.Motor{DB: pool, Actor: func(context.Context) (bitacora.Actor, bool) { return bitacora.Actor{}, false }}
	antes := contar(t, m, "rubro")
	_, err := m.Crear(context.Background(), rubros(), map[string]any{"nombre": "Sin autor", "descripcion": nil, "estado_rubro": true})
	if !errors.Is(err, crud.ErrSinAutor) {
		t.Fatalf("err = %v, quiero ErrSinAutor", err)
	}
	if _, err := m.Borrar(context.Background(), rubros(), int64(1)); !errors.Is(err, crud.ErrSinAutor) {
		t.Fatalf("Borrar: err = %v, quiero ErrSinAutor", err)
	}
	if _, _, err := m.Actualizar(context.Background(), rubros(), int64(1), map[string]any{"nombre": "x"}); !errors.Is(err, crud.ErrSinAutor) {
		t.Fatalf("Actualizar: err = %v, quiero ErrSinAutor", err)
	}
	if contar(t, m, "rubro") != antes || contar(t, m, "bitacora") != 0 {
		t.Error("el motor cambió algo sin autor")
	}
}

// Sin Actor el motor sigue siendo un CRUD simple (así lo usan las pruebas que no auditan).
func TestMotorSinActorNoRegistraBitacora(t *testing.T) {
	pool := pruebas.NuevoEsquemaCompleto(t)
	m := &crud.Motor{DB: pool}
	if _, err := m.Crear(context.Background(), rubros(), map[string]any{"nombre": "Sin bitácora", "descripcion": nil, "estado_rubro": true}); err != nil {
		t.Fatal(err)
	}
	if contar(t, m, "bitacora") != 0 {
		t.Error("sin Actor no debía registrarse nada")
	}
}
