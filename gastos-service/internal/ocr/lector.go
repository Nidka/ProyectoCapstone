package ocr

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Estados de una lectura. Solo leido y sin_texto se guardan; no_disponible y error se reintentan.
const (
	Leido        = "leido"
	SinTexto     = "sin_texto"
	NoDisponible = "no_disponible"
	Error        = "error"
)

// Lectura es el resultado de leer un archivo: el estado y el texto ya normalizado.
type Lectura struct {
	Estado string
	Texto  string
}

// Lector lee el texto de un comprobante. tipoArchivo es «pdf» o «imagen» (comprobante.tipo_archivo).
type Lector interface {
	Leer(ctx context.Context, ruta, tipoArchivo string) Lectura
}

// Herramientas lee con pdftotext, pdftoppm y tesseract. Una ruta vacía es una herramienta que no está
// instalada.
type Herramientas struct {
	Pdftotext, Pdftoppm, Tesseract string
	// PorEjecucion limita cada programa; Total, la lectura completa de un comprobante.
	PorEjecucion, Total time.Duration
}

// NuevoLector busca las herramientas en el PATH (tesseract solo cuenta si tiene el idioma spa). El
// servicio arranca igual si falta alguna.
func NuevoLector() *Herramientas {
	h := &Herramientas{Pdftotext: buscar("pdftotext"), Pdftoppm: buscar("pdftoppm"), PorEjecucion: 20 * time.Second, Total: 30 * time.Second}
	if t := buscar("tesseract"); t != "" && h.tieneEspanol(t) {
		h.Tesseract = t
	}
	return h
}

func buscar(nombre string) string {
	ruta, err := exec.LookPath(nombre)
	if err != nil {
		return ""
	}
	return ruta
}

func (h *Herramientas) tieneEspanol(tesseract string) bool {
	salida, err := h.ejecutar(context.Background(), tesseract, true, "--list-langs")
	if err != nil {
		return false
	}
	for _, l := range strings.Split(salida, "\n") {
		if strings.TrimSpace(l) == "spa" {
			return true
		}
	}
	return false
}

// Resumen dice qué herramientas encontró, para el registro de arranque.
func (h *Herramientas) Resumen() string {
	si := func(ruta string) string {
		if ruta == "" {
			return "no"
		}
		return "sí"
	}
	return "pdftotext " + si(h.Pdftotext) + ", pdftoppm " + si(h.Pdftoppm) + ", tesseract (spa) " + si(h.Tesseract)
}

// ejecutar corre un programa con argumentos separados (nunca por una shell) y con tiempo límite.
// conErrores suma la salida de error (tesseract --list-langs escribe ahí en algunas versiones).
func (h *Herramientas) ejecutar(ctx context.Context, programa string, conErrores bool, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, h.PorEjecucion)
	defer cancel()
	cmd := exec.CommandContext(ctx, programa, args...)
	var salida bytes.Buffer
	cmd.Stdout = &salida
	if conErrores {
		cmd.Stderr = &salida
	}
	// Si un hijo del programa hereda la salida, no se espera más que esto tras el tiempo límite
	cmd.WaitDelay = time.Second
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return salida.String(), nil
}

// Leer lee un PDF (texto con pdftotext; si casi no tiene texto, como escaneado con pdftoppm y
// tesseract) o una imagen (tesseract), en no más de h.Total.
func (h *Herramientas) Leer(ctx context.Context, ruta, tipoArchivo string) Lectura {
	ctx, cancel := context.WithTimeout(ctx, h.Total)
	defer cancel()
	if tipoArchivo == "pdf" {
		return h.leerPDF(ctx, ruta)
	}
	if h.Tesseract == "" {
		return Lectura{Estado: NoDisponible}
	}
	texto, err := h.reconocer(ctx, ruta)
	if err != nil {
		return Lectura{Estado: Error}
	}
	return conTexto(texto)
}

func (h *Herramientas) leerPDF(ctx context.Context, ruta string) Lectura {
	texto := ""
	if h.Pdftotext != "" {
		var err error
		if texto, err = h.ejecutar(ctx, h.Pdftotext, false, "-layout", ruta, "-"); err != nil {
			return Lectura{Estado: Error}
		}
		if Alfanumericos(texto) >= MinAlfanumericos {
			return conTexto(texto)
		}
	}
	// Escaneado (o sin pdftotext): se pasa a imagen y se lee con tesseract
	switch {
	case h.Pdftoppm == "" && h.Pdftotext == "":
		return Lectura{Estado: NoDisponible}
	case h.Pdftoppm == "":
		return Lectura{Estado: SinTexto, Texto: Normalizar(texto)}
	case h.Tesseract == "":
		return Lectura{Estado: NoDisponible}
	}
	dir, err := os.MkdirTemp("", "lectura-comprobante-")
	if err != nil {
		return Lectura{Estado: Error}
	}
	defer os.RemoveAll(dir)
	if _, err := h.ejecutar(ctx, h.Pdftoppm, false, "-r", "200", "-png", "-f", "1", "-l", "2", ruta, filepath.Join(dir, "pagina")); err != nil {
		return Lectura{Estado: Error}
	}
	paginas, _ := filepath.Glob(filepath.Join(dir, "pagina*.png"))
	sort.Strings(paginas)
	partes := []string{}
	for _, p := range paginas {
		t, err := h.reconocer(ctx, p)
		if err != nil {
			return Lectura{Estado: Error}
		}
		partes = append(partes, t)
	}
	return conTexto(strings.Join(partes, "\n"))
}

func (h *Herramientas) reconocer(ctx context.Context, imagen string) (string, error) {
	return h.ejecutar(ctx, h.Tesseract, false, imagen, "stdout", "-l", "spa", "--psm", "6")
}

func conTexto(texto string) Lectura {
	texto = Normalizar(texto)
	if Alfanumericos(texto) < MinAlfanumericos {
		return Lectura{Estado: SinTexto, Texto: texto}
	}
	return Lectura{Estado: Leido, Texto: texto}
}
