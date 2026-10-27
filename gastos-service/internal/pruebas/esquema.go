// Package pruebas prepara una base de datos aislada para las pruebas de integración.
// Solo lo importan archivos _test.go.
package pruebas

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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
	return nuevoEsquema(t, tablasMinimas)
}

// NuevoEsquemaCompleto es como NuevoEsquema pero con el esquema y los datos reales de db.sql
// (todas las tablas del sistema, no solo las mínimas), para las pruebas de la API de maestros.
func NuevoEsquemaCompleto(t testing.TB) *pgxpool.Pool {
	t.Helper()
	sql, err := os.ReadFile(filepath.Join(raizModulo(t), "..", "db.sql"))
	if err != nil {
		t.Fatal(err)
	}
	return nuevoEsquema(t, string(sql))
}

func nuevoEsquema(t testing.TB, ddl string) *pgxpool.Pool {
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

	if _, err := pool.Exec(ctx, ddl); err != nil {
		t.Fatal(err)
	}
	if err := db.RunMigrations(pool, directorioMigraciones(t)); err != nil {
		t.Fatal(err)
	}
	return pool
}

// directorioMigraciones ubica gastos-service/migrations.
func directorioMigraciones(t testing.TB) string {
	return filepath.Join(raizModulo(t), "migrations")
}

// raizModulo sube desde el directorio de la prueba hasta encontrar go.mod; así funciona desde
// cualquier paquete y también con go test -trimpath.
func raizModulo(t testing.TB) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		padre := filepath.Dir(dir)
		if padre == dir {
			t.Fatal("no se encontró go.mod subiendo desde el directorio de la prueba")
		}
		dir = padre
	}
}

// DirectorioMigraciones es el directorio de migraciones del servicio, para las pruebas que las
// reaplican (el servicio las reaplica todas en cada arranque).
func DirectorioMigraciones(t testing.TB) string { return directorioMigraciones(t) }
