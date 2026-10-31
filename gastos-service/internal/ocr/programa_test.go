package ocr

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// Las herramientas simuladas son copias del binario de pruebas: también funcionan
// en Windows, donde exec.Command no puede ejecutar un script con #!/bin/sh.
func programa(t *testing.T, nombre, modo string) string {
	t.Helper()
	origen, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(t.TempDir(), "herramienta-ocr-prueba-"+nombre+".exe")
	// No usar enlaces duros: Windows mantiene bloqueado el archivo original
	// mientras el proceso de pruebas está vivo e impediría limpiar TempDir.
	contenido, err := os.ReadFile(origen)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, contenido, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta+".modo", []byte(modo), 0o600); err != nil {
		t.Fatal(err)
	}
	return ruta
}

func TestMain(m *testing.M) {
	ruta, err := os.Executable()
	if err != nil {
		panic(err)
	}
	if strings.HasPrefix(filepath.Base(ruta), "herramienta-ocr-prueba-") {
		modo, err := os.ReadFile(ruta + ".modo")
		if err != nil {
			panic(err)
		}
		os.Exit(simularHerramienta(string(modo), os.Args[1:]))
	}
	os.Exit(m.Run())
}

func simularHerramienta(modo string, args []string) int {
	switch modo {
	case "factura", "argumento", "blanco":
		if len(args) != 3 || args[0] != "-layout" || args[2] != "-" {
			return 9
		}
		switch modo {
		case "factura":
			fmt.Println("FACTURA ELECTRÓNICA F001-77\nR.U.C.   20481608750\nFECHA: 30/09/2026\nTOTAL S/ 85.5O")
		case "argumento":
			fmt.Println("ARCHIVO RECIBIDO: " + args[1])
		case "blanco":
			fmt.Print("  \f\n")
		}
	case "paginas":
		if len(args) != 9 || !slices.Equal(args[:7], []string{"-r", "200", "-png", "-f", "1", "-l", "2"}) {
			return 9
		}
		for _, sufijo := range []string{"-1.png", "-2.png"} {
			if err := os.WriteFile(args[8]+sufijo, nil, 0o600); err != nil {
				return 1
			}
		}
	case "pagina", "pagina-lenta", "boleta", "ruido":
		if len(args) != 6 || !slices.Equal(args[1:], []string{"stdout", "-l", "spa", "--psm", "6"}) {
			return 9
		}
		switch modo {
		case "pagina":
			fmt.Printf("PAGINA %s FACTURA ELECTRONICA TOTAL S/ 85.50\n", filepath.Base(args[0]))
		case "pagina-lenta":
			time.Sleep(200 * time.Millisecond)
			fmt.Println("FACTURA ELECTRONICA TOTAL S/ 85.50")
		case "boleta":
			fmt.Println("BOLETA DE VENTA\nIMPORTE TOTAL S/ 12.00\nGracias")
		case "ruido":
			fmt.Println("|| ~~")
		}
	case "error":
		fmt.Fprintln(os.Stderr, "Syntax Error: Couldn't find trailer dictionary")
		return 1
	case "lento":
		time.Sleep(5 * time.Second)
	case "idiomas-spa", "idiomas-eng":
		if !slices.Equal(args, []string{"--list-langs"}) {
			return 9
		}
		if modo == "idiomas-spa" {
			fmt.Fprintln(os.Stderr, "List of available languages (2):\neng\nspa")
		} else {
			fmt.Println("List of available languages (1):\neng")
		}
	case "vacio":
	default:
		return 9
	}
	return 0
}
