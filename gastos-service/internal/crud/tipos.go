// Package crud es el motor genérico de los recursos de /api/v1 (antes en la API de Python):
// una definición por recurso, y listar, obtener, crear, actualizar, borrar y la lista
// avanzada sobre ella, con las mismas respuestas y errores que daba FastAPI.
package crud

import (
	"bytes"
	"encoding/json"
	"strconv"
	"time"
)

// Tipo es el tipo de una columna tal como lo expone la API.
type Tipo int

const (
	TEntero    Tipo = iota
	TTexto          // cadena
	TBool           // verdadero o falso
	TDecimal        // NUMERIC: se lee y se escribe como texto para no perder decimales ("12.50")
	TFechaHora      // TIMESTAMP sin zona: 2026-09-29T08:00:00
	TFecha          // DATE: 2026-09-29
)

// Campo es una columna de un recurso, en el orden en que sale en el JSON.
type Campo struct {
	Nombre      string
	Tipo        Tipo
	Obligatorio bool // obligatorio al crear
	Nulo        bool // admite null
	SoloLectura bool // solo sale en las respuestas (la clave generada por la base)
	Email       bool
	Defecto     any // valor al crear si no viene; nil = NULL
	Max, Min    int // largo del texto; 0 = sin límite
}

// Unico es la regla de unicidad que responde 409 al crear.
type Unico struct{ Campo, Mensaje string }

// Recurso describe una tabla expuesta en /api/v1/<Ruta>.
type Recurso struct {
	Ruta         string
	Tabla        string
	PK           string
	PKDado       bool // la clave la envía quien crea (ubigeo); si no, la genera la base
	NoEncontrado string
	Unico        *Unico
	Busqueda     []string
	Campos       []Campo
}

// Campo devuelve la columna con ese nombre.
func (r *Recurso) Campo(nombre string) (Campo, bool) {
	for _, c := range r.Campos {
		if c.Nombre == nombre {
			return c, true
		}
	}
	return Campo{}, false
}

// TipoPK es el tipo de la clave primaria.
func (r *Recurso) TipoPK() Tipo {
	c, _ := r.Campo(r.PK)
	return c.Tipo
}

// Fila es un registro leído, con sus valores en el orden de Recurso.Campos.
type Fila struct {
	recurso *Recurso
	valores []any
}

// MarshalJSON escribe el objeto con las claves en orden y sin escapar <, > ni &, como FastAPI.
func (f Fila) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte('{')
	for i, c := range f.recurso.Campos {
		if i > 0 {
			b.WriteByte(',')
		}
		nombre, _ := jsonSinEscape(c.Nombre)
		b.Write(nombre)
		b.WriteByte(':')
		v, err := jsonSinEscape(valorSalida(c.Tipo, f.valores[i]))
		if err != nil {
			return nil, err
		}
		b.Write(v)
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}

// Valor devuelve el valor de una columna, ya con el formato de salida.
func (f Fila) Valor(nombre string) any {
	for i, c := range f.recurso.Campos {
		if c.Nombre == nombre {
			return valorSalida(c.Tipo, f.valores[i])
		}
	}
	return nil
}

func jsonSinEscape(v any) ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(b.Bytes(), "\n"), nil
}

// valorSalida da a cada valor leído de la base el formato que tenía la API de Python.
func valorSalida(t Tipo, v any) any {
	if v == nil {
		return nil
	}
	switch t {
	case TFechaHora:
		if ts, ok := v.(time.Time); ok {
			return FormatoFechaHora(ts)
		}
	case TFecha:
		if ts, ok := v.(time.Time); ok {
			return ts.Format("2006-01-02")
		}
	case TEntero:
		switch n := v.(type) {
		case int16:
			return int64(n)
		case int32:
			return int64(n)
		}
	}
	return v
}

// FormatoFechaHora escribe un instante como datetime.isoformat() de Python: sin zona y con
// microsegundos solo si los hay.
func FormatoFechaHora(t time.Time) string {
	s := t.Format("2006-01-02T15:04:05")
	if us := t.Nanosecond() / 1000; us != 0 {
		s += "." + padIzquierda(strconv.Itoa(us), 6)
	}
	return s
}

func padIzquierda(s string, n int) string {
	for len(s) < n {
		s = "0" + s
	}
	return s
}
