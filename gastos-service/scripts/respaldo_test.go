package scripts

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// ejecutar corre respaldo.sh con las variables dadas y devuelve su salida combinada.
func ejecutar(t *testing.T, entorno ...string) (string, error) {
	t.Helper()
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash no está disponible")
	}
	cmd := exec.Command("bash", "respaldo.sh")
	cmd.Env = append(os.Environ(), entorno...)
	salida, err := cmd.CombinedOutput()
	return string(salida), err
}

func escribir(t *testing.T, ruta, contenido string, modo os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, []byte(contenido), modo); err != nil {
		t.Fatal(err)
	}
}

func TestRespaldoEsEjecutable(t *testing.T) {
	info, err := os.Stat("respaldo.sh")
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "windows" {
		t.Skip("Windows no expone el bit de ejecución POSIX; se comprueba en Linux")
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Fatalf("respaldo.sh no es ejecutable: %v", info.Mode())
	}
}

func TestRespaldoCopiaUploadsEnCarpetaConFecha(t *testing.T) {
	raiz := t.TempDir()
	uploads := filepath.Join(raiz, "uploads")
	respaldos := filepath.Join(raiz, "no", "existe", "respaldos")
	escribir(t, filepath.Join(uploads, "gasto-1-1.png"), "uno", 0o444)
	escribir(t, filepath.Join(uploads, "sub", "gasto-2-2.pdf"), "dos", 0o444)

	salida, err := ejecutar(t, "UPLOADS_DIR="+uploads, "RESPALDO_DIR="+respaldos, "FECHA_RESPALDO=20261022-030000")
	if err != nil {
		t.Fatalf("falló: %v\n%s", err, salida)
	}

	destino := filepath.Join(respaldos, "uploads-20261022-030000")
	for ruta, esperado := range map[string]string{"gasto-1-1.png": "uno", "sub/gasto-2-2.pdf": "dos"} {
		got, err := os.ReadFile(filepath.Join(destino, ruta))
		if err != nil || string(got) != esperado {
			t.Fatalf("%s: %q, %v", ruta, got, err)
		}
	}
	if !strings.Contains(salida, destino) {
		t.Fatalf("la salida debería decir dónde quedó el respaldo: %q", salida)
	}
	// El original no se toca.
	if _, err := os.Stat(filepath.Join(uploads, "gasto-1-1.png")); err != nil {
		t.Fatal(err)
	}
}

func TestRespaldoFallaSiNoExisteUploads(t *testing.T) {
	raiz := t.TempDir()
	salida, err := ejecutar(t, "UPLOADS_DIR="+filepath.Join(raiz, "nada"), "RESPALDO_DIR="+filepath.Join(raiz, "r"))
	if err == nil {
		t.Fatalf("se esperaba un error:\n%s", salida)
	}
	if !strings.Contains(salida, "UPLOADS_DIR") {
		t.Fatalf("el mensaje debería nombrar la variable: %q", salida)
	}
	if _, err := os.Stat(filepath.Join(raiz, "r")); err == nil {
		t.Fatal("no debería crear la carpeta de respaldo si falla")
	}
}

func TestRespaldoNoPisaUnRespaldoExistente(t *testing.T) {
	raiz := t.TempDir()
	uploads := filepath.Join(raiz, "uploads")
	respaldos := filepath.Join(raiz, "respaldos")
	escribir(t, filepath.Join(uploads, "a.png"), "a", 0o444)
	entorno := []string{"UPLOADS_DIR=" + uploads, "RESPALDO_DIR=" + respaldos, "FECHA_RESPALDO=20261022-030000"}

	if salida, err := ejecutar(t, entorno...); err != nil {
		t.Fatalf("primer respaldo: %v\n%s", err, salida)
	}
	if salida, err := ejecutar(t, entorno...); err == nil {
		t.Fatalf("el segundo respaldo igual debería fallar:\n%s", salida)
	}
}

func TestRespaldoSePuedeRepetirOtroDiaConArchivosDeSoloLectura(t *testing.T) {
	// Los comprobantes están en 0444 (RNF-5); copiarlos de nuevo otro día no debe chocar.
	raiz := t.TempDir()
	uploads := filepath.Join(raiz, "uploads")
	escribir(t, filepath.Join(uploads, "a.png"), "a", 0o444)
	for _, fecha := range []string{"20261022-030000", "20261023-030000"} {
		salida, err := ejecutar(t, "UPLOADS_DIR="+uploads, "RESPALDO_DIR="+filepath.Join(raiz, "r"), "FECHA_RESPALDO="+fecha)
		if err != nil {
			t.Fatalf("%s: %v\n%s", fecha, err, salida)
		}
	}
}
