// Package registro deja los errores técnicos del servicio en un archivo, además de la consola (RNF-14).
package registro

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

// Configurar dirige el log estándar a la consola y al archivo indicado. Crea el directorio si hace
// falta y agrega al final de lo que ya hubiera. Con ruta vacía solo usa la consola. Devuelve la
// función que cierra el archivo.
func Configurar(ruta string, consola io.Writer) (func() error, error) {
	if ruta == "" {
		log.SetOutput(consola)
		return func() error { return nil }, nil
	}
	if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
		return nil, fmt.Errorf("no se pudo crear el directorio del registro de errores: %w", err)
	}
	archivo, err := os.OpenFile(ruta, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("no se pudo abrir el registro de errores: %w", err)
	}
	log.SetOutput(io.MultiWriter(consola, archivo))
	return archivo.Close, nil
}
