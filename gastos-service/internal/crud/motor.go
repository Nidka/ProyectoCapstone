package crud

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gastos-service/internal/bitacora"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Ejecutor es lo que el motor necesita de la base (lo cumple *pgxpool.Pool, y también una transacción).
type Ejecutor interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Begin(ctx context.Context) (pgx.Tx, error)
}

// Motor ejecuta las operaciones de los recursos. Los nombres de tabla y de columna salen siempre
// de las definiciones (constantes), nunca de lo que envía quien llama; los valores van como parámetros.
//
// Si Actor está definido, cada Crear, Actualizar y Borrar que cambia algo corre en una transacción
// junto con su entrada de bitácora (maestro.crear, maestro.editar, maestro.eliminar): si la bitácora
// falla, el cambio no se hace. Sin Actor el motor no registra nada (solo así se usa en pruebas
// del motor; el servicio siempre lo define).
type Motor struct {
	DB    Ejecutor
	Actor func(ctx context.Context) (bitacora.Actor, bool)
}

// ErrSinAutor: el motor audita pero la petición no trae un usuario autenticado.
var ErrSinAutor = errors.New("crud: no hay un usuario autenticado para registrar el cambio en la bitácora")

// auditar corre fn dentro de una transacción con la que fn también registra su entrada de bitácora.
// Sin Actor corre sin transacción, directamente sobre la base.
func (m *Motor) auditar(ctx context.Context, fn func(q Ejecutor, quien bitacora.Actor, auditado bool) error) error {
	if m.Actor == nil {
		return fn(m.DB, bitacora.Actor{}, false)
	}
	quien, ok := m.Actor(ctx)
	if !ok {
		return ErrSinAutor
	}
	tx, err := m.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(tx, quien, true); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// idEntidad es la clave primaria de una fila para la bitácora (todas son enteros).
func idEntidad(f Fila) int64 {
	switch n := f.Valor(f.recurso.PK).(type) {
	case int64:
		return n
	case int32:
		return int64(n)
	case int16:
		return int64(n)
	case int:
		return int64(n)
	}
	return 0
}

func registrarMaestro(ctx context.Context, q Ejecutor, quien bitacora.Actor, accion string, r *Recurso, id int64, detalle map[string]any) error {
	return bitacora.Registrar(ctx, q, bitacora.Entrada{Actor: quien, Accion: accion, Entidad: r.Tabla, IDEntidad: id, Detalle: detalle})
}

// ParamsAvanzada son los parámetros de GET /advanced/<recurso>.
type ParamsAvanzada struct {
	Limit, Offset int
	Search        string
	SearchFields  string
	OrderBy       string
	OrderDir      string
	Filters       string
}

func ident(s string) string { return `"` + s + `"` }

// escalaDecimal es la escala con que la API de Python entregaba los decimales: todas las columnas
// NUMERIC de los recursos eran Numeric(10, 2) ("8000.00", aunque la base guarde 8000).
const escalaDecimal = 2

// columnas arma la lista del SELECT; los decimales salen como texto con su escala para no perder el formato.
func columnas(r *Recurso) string {
	partes := make([]string, len(r.Campos))
	for i, c := range r.Campos {
		if c.Tipo == TDecimal {
			partes[i] = fmt.Sprintf("%s::numeric(38,%d)::text", ident(c.Nombre), escalaDecimal)
		} else {
			partes[i] = ident(c.Nombre)
		}
	}
	return strings.Join(partes, ", ")
}

// marcador devuelve el parámetro $n, con el cast que necesita el tipo.
func marcador(c Campo, n int) string {
	if c.Tipo == TDecimal {
		return fmt.Sprintf("$%d::numeric", n)
	}
	return fmt.Sprintf("$%d", n)
}

func leerFilas(r *Recurso, rows pgx.Rows) ([]Fila, error) {
	defer rows.Close()
	filas := []Fila{}
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			return nil, err
		}
		filas = append(filas, Fila{recurso: r, valores: vals})
	}
	return filas, rows.Err()
}

// Listar devuelve una página ordenada por la clave primaria.
func (m *Motor) Listar(ctx context.Context, r *Recurso, limit, offset int) ([]Fila, error) {
	sql := fmt.Sprintf("SELECT %s FROM %s ORDER BY %s OFFSET $1 LIMIT $2", columnas(r), ident(r.Tabla), ident(r.PK))
	rows, err := m.DB.Query(ctx, sql, offset, limit)
	if err != nil {
		return nil, err
	}
	return leerFilas(r, rows)
}

// Obtener devuelve un registro por su clave; el segundo valor es false si no existe.
func (m *Motor) Obtener(ctx context.Context, r *Recurso, id any) (Fila, bool, error) {
	return obtenerEn(ctx, m.DB, r, id, false)
}

// obtenerEn es Obtener sobre un ejecutor dado; con bloquear toma la fila FOR UPDATE.
func obtenerEn(ctx context.Context, q Ejecutor, r *Recurso, id any, bloquear bool) (Fila, bool, error) {
	sql := fmt.Sprintf("SELECT %s FROM %s WHERE %s = $1", columnas(r), ident(r.Tabla), ident(r.PK))
	if bloquear {
		sql += " FOR UPDATE"
	}
	rows, err := q.Query(ctx, sql, id)
	if err != nil {
		return Fila{}, false, err
	}
	filas, err := leerFilas(r, rows)
	if err != nil || len(filas) == 0 {
		return Fila{}, false, err
	}
	return filas[0], true, nil
}

// ExisteUnico dice si ya hay un registro con ese valor en el campo de unicidad del recurso.
func (m *Motor) ExisteUnico(ctx context.Context, r *Recurso, valor any) (bool, error) {
	c, _ := r.Campo(r.Unico.Campo)
	sql := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE %s = %s)", ident(r.Tabla), ident(c.Nombre), marcador(c, 1))
	var existe bool
	err := m.DB.QueryRow(ctx, sql, valor).Scan(&existe)
	return existe, err
}

// Crear inserta el registro con los valores ya validados y devuelve lo guardado.
func (m *Motor) Crear(ctx context.Context, r *Recurso, valores map[string]any) (Fila, error) {
	var fila Fila
	err := m.auditar(ctx, func(q Ejecutor, quien bitacora.Actor, auditado bool) error {
		f, err := crearEn(ctx, q, r, valores)
		if err != nil {
			return err
		}
		if f.recurso == nil {
			return errors.New("crud: el INSERT no devolvió el registro creado")
		}
		fila = f
		if !auditado {
			return nil
		}
		return registrarMaestro(ctx, q, quien, "maestro.crear", r, idEntidad(f), map[string]any{"despues": f})
	})
	if err != nil {
		return Fila{}, err
	}
	return fila, nil
}

func crearEn(ctx context.Context, q Ejecutor, r *Recurso, valores map[string]any) (Fila, error) {
	var cols, marcas []string
	var args []any
	for _, c := range r.Campos {
		if c.SoloLectura {
			continue
		}
		args = append(args, valores[c.Nombre])
		cols = append(cols, ident(c.Nombre))
		marcas = append(marcas, marcador(c, len(args)))
	}
	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) RETURNING %s", ident(r.Tabla), strings.Join(cols, ", "), strings.Join(marcas, ", "), columnas(r))
	rows, err := q.Query(ctx, sql, args...)
	if err != nil {
		return Fila{}, err
	}
	filas, err := leerFilas(r, rows)
	if err != nil || len(filas) == 0 {
		return Fila{}, err
	}
	return filas[0], nil
}

// Actualizar cambia las columnas indicadas; el segundo valor es false si el registro no existe.
func (m *Motor) Actualizar(ctx context.Context, r *Recurso, id any, valores map[string]any) (Fila, bool, error) {
	if len(valores) == 0 {
		return m.Obtener(ctx, r, id)
	}
	var fila Fila
	var existe bool
	err := m.auditar(ctx, func(q Ejecutor, quien bitacora.Actor, auditado bool) error {
		var antes Fila
		if auditado { // el «antes» se lee con la fila bloqueada para que sea el valor que de verdad se reemplaza
			var hay bool
			var err error
			if antes, hay, err = obtenerEn(ctx, q, r, id, true); err != nil || !hay {
				return err
			}
		}
		f, hay, err := actualizarEn(ctx, q, r, id, valores)
		fila, existe = f, hay
		if err != nil || !hay || !auditado {
			return err
		}
		return registrarMaestro(ctx, q, quien, "maestro.editar", r, idEntidad(f), map[string]any{"antes": antes, "despues": f})
	})
	if err != nil {
		return Fila{}, false, err
	}
	return fila, existe, nil
}

func actualizarEn(ctx context.Context, q Ejecutor, r *Recurso, id any, valores map[string]any) (Fila, bool, error) {
	var sets []string
	var args []any
	for _, c := range r.Campos {
		v, ok := valores[c.Nombre]
		if !ok {
			continue
		}
		args = append(args, v)
		sets = append(sets, fmt.Sprintf("%s = %s", ident(c.Nombre), marcador(c, len(args))))
	}
	args = append(args, id)
	sql := fmt.Sprintf("UPDATE %s SET %s WHERE %s = $%d RETURNING %s", ident(r.Tabla), strings.Join(sets, ", "), ident(r.PK), len(args), columnas(r))
	rows, err := q.Query(ctx, sql, args...)
	if err != nil {
		return Fila{}, false, err
	}
	filas, err := leerFilas(r, rows)
	if err != nil || len(filas) == 0 {
		return Fila{}, false, err
	}
	return filas[0], true, nil
}

// Borrar elimina el registro; devuelve false si no existía.
func (m *Motor) Borrar(ctx context.Context, r *Recurso, id any) (bool, error) {
	var existia bool
	err := m.auditar(ctx, func(q Ejecutor, quien bitacora.Actor, auditado bool) error {
		var antes Fila
		if auditado { // lo que se borra queda en la bitácora con todos sus valores
			var hay bool
			var err error
			if antes, hay, err = obtenerEn(ctx, q, r, id, true); err != nil || !hay {
				return err
			}
		}
		tag, err := q.Exec(ctx, fmt.Sprintf("DELETE FROM %s WHERE %s = $1", ident(r.Tabla), ident(r.PK)), id)
		if err != nil {
			return err
		}
		existia = tag.RowsAffected() > 0
		if !existia || !auditado {
			return nil
		}
		return registrarMaestro(ctx, q, quien, "maestro.eliminar", r, idEntidad(antes), map[string]any{"antes": antes})
	})
	if err != nil {
		return false, err
	}
	return existia, nil
}

// Avanzada es la lista con búsqueda parcial, filtros exactos y orden. Los nombres de campo que no
// son columnas del recurso se ignoran, como hacía la API de Python.
func (m *Motor) Avanzada(ctx context.Context, r *Recurso, p ParamsAvanzada) ([]Fila, error) {
	var donde []string
	var args []any

	if p.Search != "" {
		campos := r.Busqueda
		if p.SearchFields != "" {
			campos = nil
			for _, nombre := range strings.Split(p.SearchFields, ",") {
				nombre = strings.TrimSpace(nombre)
				if _, ok := r.Campo(nombre); ok && nombre != "" {
					campos = append(campos, nombre)
				}
			}
		}
		if len(campos) > 0 {
			args = append(args, "%"+p.Search+"%")
			var ors []string
			for _, nombre := range campos {
				ors = append(ors, fmt.Sprintf("CAST(%s AS TEXT) ILIKE $%d", ident(nombre), len(args)))
			}
			donde = append(donde, "("+strings.Join(ors, " OR ")+")")
		}
	}
	for _, par := range strings.Split(p.Filters, ",") {
		par = strings.TrimSpace(par)
		nombre, valor, ok := strings.Cut(par, ":")
		if !ok {
			continue
		}
		nombre = strings.TrimSpace(nombre)
		if _, existe := r.Campo(nombre); !existe {
			continue
		}
		args = append(args, strings.TrimSpace(valor))
		donde = append(donde, fmt.Sprintf("CAST(%s AS TEXT) = $%d", ident(nombre), len(args)))
	}

	orden := r.PK
	if _, ok := r.Campo(p.OrderBy); ok {
		orden = p.OrderBy
	}
	dir := "ASC"
	if p.OrderDir == "desc" {
		dir = "DESC"
	}
	sql := fmt.Sprintf("SELECT %s FROM %s", columnas(r), ident(r.Tabla))
	if len(donde) > 0 {
		sql += " WHERE " + strings.Join(donde, " AND ")
	}
	sql += fmt.Sprintf(" ORDER BY %s %s, %s OFFSET $%d LIMIT $%d", ident(orden), dir, ident(r.PK), len(args)+1, len(args)+2)
	args = append(args, p.Offset, p.Limit)

	rows, err := m.DB.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return leerFilas(r, rows)
}
