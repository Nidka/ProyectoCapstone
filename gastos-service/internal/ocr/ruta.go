package ocr

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrRutaFuera: el archivo del comprobante no está dentro de UPLOADS_DIR.
var ErrRutaFuera = errors.New("el archivo del comprobante está fuera de la carpeta de comprobantes")

// RutaSegura convierte el archivo_url de un comprobante («/uploads/<nombre>») en la ruta absoluta del
// archivo, y la rechaza si sale de uploadsDir (con «..» o con un enlace simbólico) o si no es un
// archivo regular.
func RutaSegura(uploadsDir, archivoURL string) (string, error) {
	nombre, ok := strings.CutPrefix(archivoURL, "/uploads/")
	if !ok || nombre == "" {
		return "", ErrRutaFuera
	}
	base, err := filepath.Abs(uploadsDir)
	if err != nil {
		return "", err
	}
	if real, err := filepath.EvalSymlinks(base); err == nil {
		base = real
	}
	ruta := filepath.Join(base, filepath.FromSlash(nombre))
	if !dentro(base, ruta) {
		return "", ErrRutaFuera
	}
	real, err := filepath.EvalSymlinks(ruta)
	if err != nil {
		return "", fmt.Errorf("archivo del comprobante: %w", err)
	}
	if !dentro(base, real) {
		return "", ErrRutaFuera
	}
	info, err := os.Stat(real)
	if err != nil {
		return "", fmt.Errorf("archivo del comprobante: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("archivo del comprobante: %s no es un archivo", nombre)
	}
	return real, nil
}

func dentro(base, ruta string) bool {
	rel, err := filepath.Rel(base, ruta)
	return err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
