package demo

import (
	"context"
	neturl "net/url"
	"os"
	"os/exec"
	"strings"
	"testing"

	"gastos-service/internal/pruebas"

	"github.com/jackc/pgx/v5/pgxpool"
)

// sinSecretos quita de un texto la URL de conexión y su contraseña, para que un fallo de psql no
// los deje en el registro de la prueba.
func sinSecretos(texto, url string) string {
	if u, err := neturl.Parse(url); err == nil && u.User != nil {
		if clave, ok := u.User.Password(); ok && clave != "" {
			texto = strings.ReplaceAll(texto, clave, "<clave>")
		}
	}
	return strings.ReplaceAll(texto, url, "<TEST_DATABASE_URL>")
}

// psql ejecuta un script de esta carpeta contra el esquema temporal de la prueba.
func psql(t *testing.T, pool *pgxpool.Pool, script string) {
	t.Helper()
	ruta, err := exec.LookPath("psql")
	if err != nil {
		t.Skip("psql no está instalado: no se puede ejecutar el script de demostración")
	}
	var esquema string
	if err := pool.QueryRow(context.Background(), `SELECT current_schema()`).Scan(&esquema); err != nil {
		t.Fatal(err)
	}
	url := os.Getenv("TEST_DATABASE_URL")
	cmd := exec.Command(ruta, "-X", "-q", "-v", "ON_ERROR_STOP=1", "-f", script, url)
	cmd.Env = append(os.Environ(), "PGOPTIONS=-c search_path="+esquema)
	if salida, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s: %v\n%s", script, err, sinSecretos(string(salida), url))
	}
}

func contar(t *testing.T, pool *pgxpool.Pool, consulta string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), consulta).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// Los avisos de notificacion apuntan a gasto y son permanentes: el script de limpieza debe borrarlos
// antes que los gastos DEMO, o la demo ya no se puede limpiar ni repetir.
func TestLaDemoSePuedeLimpiarYRepetirConAvisosGenerados(t *testing.T) {
	pool := pruebas.NuevoEsquemaCompleto(t)
	psql(t, pool, "reportes_demo.sql")
	if contar(t, pool, `SELECT count(*) FROM gasto WHERE creado_por = 'DEMO'`) == 0 {
		t.Fatal("la demo no cargó gastos")
	}
	// Lo que haría el generador de avisos con gastos DEMO sin comprobante.
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO notificacion (tipo, id_gasto, mensaje)
		SELECT 'sin_comprobante', id, 'Aviso de demostración' FROM gasto WHERE creado_por = 'DEMO'`); err != nil {
		t.Fatal(err)
	}

	psql(t, pool, "reportes_demo.sql") // repetir la carga también pasa por la limpieza
	psql(t, pool, "reportes_demo_limpiar.sql")

	if n := contar(t, pool, `SELECT count(*) FROM gasto WHERE creado_por = 'DEMO'`); n != 0 {
		t.Errorf("quedaron %d gastos DEMO", n)
	}
	if n := contar(t, pool, `SELECT count(*) FROM notificacion`); n != 0 {
		t.Errorf("quedaron %d avisos de gastos DEMO", n)
	}
}
