// Package bitacora es el registro de auditoría (REQ-33 a REQ-36): quién hizo qué y cuándo, con los
// valores de antes y de después. Se escribe siempre dentro de la misma transacción que el cambio
// que describe, de modo que o quedan los dos o no queda ninguno. La tabla no admite UPDATE,
// DELETE ni TRUNCATE (lo impone un trigger de la migración 0008).
package bitacora

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Actor es quien hace el cambio. ID cero significa «sin usuario de la tabla usuario» (por ejemplo,
// el sistema al sembrar el administrador); el nombre es obligatorio y queda copiado en la fila
// porque el usuario puede cambiar de nombre o desactivarse después.
type Actor struct {
	ID     int
	Nombre string
}

// Entrada es un hecho por registrar. Accion es un texto corto «entidad.verbo» (gasto.registrar,
// comprobante.aprobar, sesion.inicio…). IDEntidad cero significa «sin id». Detalle se guarda como
// JSON; por convención lleva «antes» y «despues». Nunca debe llevar contraseñas ni hashes.
type Entrada struct {
	Actor     Actor
	Accion    string
	Entidad   string
	IDEntidad int64
	Detalle   any
}

// Ejecutor es lo que Registrar necesita: lo cumplen una transacción (pgx.Tx) y el pool. Para que el
// registro sea parte del cambio hay que pasarle la transacción del cambio.
type Ejecutor interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// Registrar inserta la entrada. Si falla, quien llama debe deshacer también su cambio: la regla es
// que no haya cambios sin registro.
func Registrar(ctx context.Context, q Ejecutor, e Entrada) error {
	if e.Actor.Nombre == "" {
		return errors.New("bitácora: falta el usuario que hace el cambio")
	}
	if e.Accion == "" || e.Entidad == "" {
		return errors.New("bitácora: faltan la acción o la entidad")
	}
	detalle := []byte("{}")
	if e.Detalle != nil {
		b, err := json.Marshal(e.Detalle)
		if err != nil {
			return err
		}
		detalle = b
	}
	var idUsuario, idEntidad any
	if e.Actor.ID != 0 {
		idUsuario = e.Actor.ID
	}
	if e.IDEntidad != 0 {
		idEntidad = e.IDEntidad
	}
	_, err := q.Exec(ctx, `
		INSERT INTO bitacora (id_usuario, usuario, accion, entidad, id_entidad, detalle)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		idUsuario, e.Actor.Nombre, e.Accion, e.Entidad, idEntidad, detalle)
	return err
}

// Registro es una fila de la bitácora tal como se muestra en el historial.
type Registro struct {
	Fecha     time.Time       `json:"fecha"`
	Usuario   string          `json:"usuario"`
	Accion    string          `json:"accion"`
	Entidad   string          `json:"entidad"`
	IDEntidad *int64          `json:"id_entidad"`
	Detalle   json.RawMessage `json:"detalle"`
}

// Consultador es lo que necesita el historial: lo cumple el pool.
type Consultador interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// HistorialDeGasto devuelve, de la más antigua a la más reciente, lo que se registró sobre un gasto
// y sobre sus comprobantes (REQ-36).
func HistorialDeGasto(ctx context.Context, q Consultador, gastoID int) ([]Registro, error) {
	filas, err := q.Query(ctx, `
		SELECT fecha, usuario, accion, entidad, id_entidad, detalle
		FROM bitacora
		WHERE (entidad = 'gasto' AND id_entidad = $1)
		   OR (entidad = 'comprobante' AND id_entidad IN (SELECT id FROM comprobante WHERE gasto_id = $1))
		ORDER BY fecha, id`, gastoID)
	if err != nil {
		return nil, err
	}
	defer filas.Close()
	historial := []Registro{}
	for filas.Next() {
		var r Registro
		if err := filas.Scan(&r.Fecha, &r.Usuario, &r.Accion, &r.Entidad, &r.IDEntidad, &r.Detalle); err != nil {
			return nil, err
		}
		historial = append(historial, r)
	}
	return historial, filas.Err()
}
