// Package reportes calcula las estadísticas de gastos de un rango de fechas
// y genera los PDF que se descargan desde el dashboard gerencial.
package reportes

import "time"

// Lima es la zona horaria de LIZAR (Perú no tiene horario de verano).
var Lima = time.FixedZone("America/Lima", -5*60*60)

const formatoISO = "2006-01-02"

// rangoMaximoDias es el rango más largo que se acepta, contando ambos extremos.
const rangoMaximoDias = 731

// Filtro es el rango de fechas del gasto, con ambos extremos incluidos.
type Filtro struct {
	Desde time.Time
	Hasta time.Time
}

// ErrFiltro es un error del usuario al elegir el rango: se responde con 400.
type ErrFiltro struct{ Mensaje string }

func (e ErrFiltro) Error() string { return e.Mensaje }

// ParseFiltro resuelve desde y hasta (AAAA-MM-DD). Sin hasta, es hoy en Lima;
// sin desde, el primer día del mes 11 meses antes del mes de hasta.
func ParseFiltro(desdeTxt, hastaTxt string, ahora time.Time) (Filtro, error) {
	hoy := ahora.In(Lima)
	hasta := time.Date(hoy.Year(), hoy.Month(), hoy.Day(), 0, 0, 0, 0, time.UTC)
	if hastaTxt != "" {
		h, err := parseFecha(hastaTxt)
		if err != nil {
			return Filtro{}, err
		}
		hasta = h
	}
	desde := time.Date(hasta.Year(), hasta.Month()-11, 1, 0, 0, 0, 0, time.UTC)
	if desdeTxt != "" {
		d, err := parseFecha(desdeTxt)
		if err != nil {
			return Filtro{}, err
		}
		desde = d
	}
	if desde.After(hasta) {
		return Filtro{}, ErrFiltro{"la fecha inicial no puede ser posterior a la final"}
	}
	if int(hasta.Sub(desde).Hours()/24)+1 > rangoMaximoDias {
		return Filtro{}, ErrFiltro{"el rango no puede superar 24 meses"}
	}
	return Filtro{Desde: desde, Hasta: hasta}, nil
}

func parseFecha(txt string) (time.Time, error) {
	t, err := time.Parse(formatoISO, txt)
	if err != nil {
		return time.Time{}, ErrFiltro{"fecha inválida: use AAAA-MM-DD"}
	}
	return t, nil
}

// DesdeISO devuelve el inicio del rango como AAAA-MM-DD.
func (f Filtro) DesdeISO() string { return f.Desde.Format(formatoISO) }

// HastaISO devuelve el fin del rango como AAAA-MM-DD.
func (f Filtro) HastaISO() string { return f.Hasta.Format(formatoISO) }
