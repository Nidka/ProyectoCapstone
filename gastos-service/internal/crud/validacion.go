package crud

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// ErrorCampo es un error de validación con la forma de los de FastAPI: type, loc, msg, input y ctx.
type ErrorCampo struct {
	Tipo  string         `json:"type"`
	Loc   []any          `json:"loc"`
	Msg   string         `json:"msg"`
	Input any            `json:"input"`
	Ctx   map[string]any `json:"ctx,omitempty"`
}

var (
	reEmail   = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
	reDecimal = regexp.MustCompile(`^[+-]?(\d+\.?\d*|\.\d+)$`)
)

// Coercionar convierte un valor JSON al tipo del campo con la misma tolerancia que Pydantic en
// modo normal (por ejemplo, "12" es un entero válido). Devuelve el valor listo para la base o el
// error. Los decimales viajan como texto y los instantes como time.Time.
func Coercionar(c Campo, v any) (any, *ErrorCampo) {
	if v == nil {
		if c.Nulo {
			return nil, nil
		}
		return nil, errTipo(c.Tipo, v)
	}
	switch c.Tipo {
	case TEntero:
		return coercionarEntero(v)
	case TTexto:
		s, ok := v.(string)
		if !ok {
			return nil, &ErrorCampo{Tipo: "string_type", Msg: "Input should be a valid string", Input: v}
		}
		n := utf8.RuneCountInString(s)
		if c.Max > 0 && n > c.Max {
			return nil, &ErrorCampo{Tipo: "string_too_long", Msg: fmt.Sprintf("String should have at most %d characters", c.Max), Input: v, Ctx: map[string]any{"max_length": c.Max}}
		}
		if c.Min > 0 && n < c.Min {
			return nil, &ErrorCampo{Tipo: "string_too_short", Msg: fmt.Sprintf("String should have at least %d characters", c.Min), Input: v, Ctx: map[string]any{"min_length": c.Min}}
		}
		if c.Email && !reEmail.MatchString(s) {
			return nil, &ErrorCampo{Tipo: "value_error", Msg: "value is not a valid email address", Input: v}
		}
		return s, nil
	case TBool:
		return coercionarBool(v)
	case TDecimal:
		return coercionarDecimal(v)
	case TFechaHora:
		return coercionarFechaHora(v)
	case TFecha:
		return coercionarFecha(v)
	}
	return nil, errTipo(c.Tipo, v)
}

func errTipo(t Tipo, v any) *ErrorCampo {
	switch t {
	case TEntero:
		return &ErrorCampo{Tipo: "int_type", Msg: "Input should be a valid integer", Input: v}
	case TTexto:
		return &ErrorCampo{Tipo: "string_type", Msg: "Input should be a valid string", Input: v}
	case TBool:
		return &ErrorCampo{Tipo: "bool_type", Msg: "Input should be a valid boolean", Input: v}
	case TDecimal:
		return &ErrorCampo{Tipo: "decimal_type", Msg: "Decimal input should be an integer, float, string or Decimal object", Input: v}
	case TFechaHora:
		return &ErrorCampo{Tipo: "datetime_type", Msg: "Input should be a valid datetime", Input: v}
	default:
		return &ErrorCampo{Tipo: "date_type", Msg: "Input should be a valid date", Input: v}
	}
}

func coercionarEntero(v any) (any, *ErrorCampo) {
	switch n := v.(type) {
	case float64:
		if n != math.Trunc(n) || math.Abs(n) > 9e15 {
			return nil, &ErrorCampo{Tipo: "int_from_float", Msg: "Input should be a valid integer, got a number with a fractional part", Input: v}
		}
		return int64(n), nil
	case string:
		i, err := strconv.ParseInt(strings.TrimSpace(n), 10, 64)
		if err != nil {
			return nil, &ErrorCampo{Tipo: "int_parsing", Msg: "Input should be a valid integer, unable to parse string as an integer", Input: v}
		}
		return i, nil
	}
	return nil, errTipo(TEntero, v)
}

func coercionarBool(v any) (any, *ErrorCampo) {
	switch b := v.(type) {
	case bool:
		return b, nil
	case float64:
		if b == 0 || b == 1 {
			return b == 1, nil
		}
	case string:
		switch strings.ToLower(strings.TrimSpace(b)) {
		case "true", "t", "yes", "y", "on", "1":
			return true, nil
		case "false", "f", "no", "n", "off", "0":
			return false, nil
		}
		return nil, &ErrorCampo{Tipo: "bool_parsing", Msg: "Input should be a valid boolean, unable to interpret input", Input: v}
	default:
		return nil, errTipo(TBool, v)
	}
	return nil, &ErrorCampo{Tipo: "bool_parsing", Msg: "Input should be a valid boolean, unable to interpret input", Input: v}
}

func coercionarDecimal(v any) (any, *ErrorCampo) {
	switch d := v.(type) {
	case float64:
		return strconv.FormatFloat(d, 'f', -1, 64), nil
	case string:
		s := strings.TrimSpace(d)
		if !reDecimal.MatchString(s) {
			return nil, &ErrorCampo{Tipo: "decimal_parsing", Msg: "Input should be a valid decimal", Input: v}
		}
		return s, nil
	}
	return nil, errTipo(TDecimal, v)
}

var formatosFechaHora = []string{
	"2006-01-02T15:04:05.999999999Z07:00", "2006-01-02T15:04:05.999999999", "2006-01-02 15:04:05.999999999Z07:00",
	"2006-01-02 15:04:05.999999999", "2006-01-02T15:04Z07:00", "2006-01-02T15:04", "2006-01-02",
}

func coercionarFechaHora(v any) (any, *ErrorCampo) {
	s, ok := v.(string)
	if !ok {
		return nil, errTipo(TFechaHora, v)
	}
	for _, f := range formatosFechaHora {
		if t, err := time.Parse(f, strings.TrimSpace(s)); err == nil {
			// La columna es TIMESTAMP sin zona: se guarda la hora tal como se escribió
			return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC), nil
		}
	}
	return nil, &ErrorCampo{Tipo: "datetime_from_date_parsing", Msg: "Input should be a valid datetime or date, invalid character in year", Input: v}
}

func coercionarFecha(v any) (any, *ErrorCampo) {
	s, ok := v.(string)
	if !ok {
		return nil, errTipo(TFecha, v)
	}
	for _, f := range []string{"2006-01-02", "2006-01-02T15:04:05"} {
		if t, err := time.Parse(f, strings.TrimSpace(s)); err == nil {
			return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), nil
		}
	}
	return nil, &ErrorCampo{Tipo: "date_from_datetime_parsing", Msg: "Input should be a valid date or datetime, invalid character in year", Input: v}
}

// ValidarCreacion valida el cuerpo de un POST contra el recurso. Devuelve los valores en el orden
// de las columnas de creación (con los defectos aplicados) o los errores, todos juntos.
func ValidarCreacion(r *Recurso, cuerpo map[string]any) (map[string]any, []ErrorCampo) {
	valores := map[string]any{}
	var errs []ErrorCampo
	for _, c := range r.Campos {
		if c.SoloLectura {
			continue
		}
		v, presente := cuerpo[c.Nombre]
		if !presente {
			if c.Obligatorio {
				errs = append(errs, ErrorCampo{Tipo: "missing", Loc: []any{"body", c.Nombre}, Msg: "Field required", Input: cuerpo})
				continue
			}
			valores[c.Nombre] = c.Defecto
			continue
		}
		val, e := Coercionar(c, v)
		if e != nil {
			e.Loc = []any{"body", c.Nombre}
			errs = append(errs, *e)
			continue
		}
		valores[c.Nombre] = val
	}
	return valores, errs
}

// ValidarCambios valida los campos de un PUT: solo se aplican las columnas conocidas (la clave
// primaria no se cambia) y las demás claves se ignoran, como hacía la API de Python.
func ValidarCambios(r *Recurso, cuerpo map[string]any) (map[string]any, []ErrorCampo) {
	valores := map[string]any{}
	var errs []ErrorCampo
	for _, c := range r.Campos {
		v, presente := cuerpo[c.Nombre]
		if !presente || c.Nombre == r.PK {
			continue
		}
		val, e := Coercionar(c, v)
		if e != nil {
			e.Loc = []any{"body", c.Nombre}
			errs = append(errs, *e)
			continue
		}
		valores[c.Nombre] = val
	}
	return valores, errs
}
