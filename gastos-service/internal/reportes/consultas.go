package reportes

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Iniciador abre transacciones (lo cumple *pgxpool.Pool).
type Iniciador interface {
	BeginTx(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error)
}

// Uniones de un gasto con su cliente, su viaje, la unidad (tracto) y el conductor.
// Todas son LEFT JOIN: lo que no se pueda unir se muestra como «Sin asignar».
const unionesGastos = `
	FROM gasto g
	LEFT JOIN cliente c ON c.id_cliente = g.cliente_id
	LEFT JOIN programacionviaje v ON v.id_viaje = g.viaje_id
	LEFT JOIN unidad u ON u.id_unidad = v.id_tracto
	LEFT JOIN conductorunidad cu ON cu.id_conductor_unidad = v.id_conductor_unidad
	LEFT JOIN conductor d ON d.id_conductor = cu.id_conductor`

const desdeGastos = unionesGastos + `
	WHERE g.fecha BETWEEN $1::date AND $2::date`

const (
	etiquetaCliente   = `COALESCE(NULLIF(trim(c.razon_social), ''), 'Sin asignar')`
	etiquetaUnidad    = `COALESCE(NULLIF(trim(u.placa), ''), 'Sin asignar')`
	etiquetaConductor = `COALESCE(NULLIF(regexp_replace(trim(concat_ws(' ', d.nombre1, d.apellido1, d.apellido2)), '\s+', ' ', 'g'), ''), 'Sin asignar')`
)

const sqlIndicadores = `
	SELECT count(*)::int,
	       COALESCE(round(sum(monto), 2), 0)::float8,
	       COALESCE(round(avg(monto), 2), 0)::float8,
	       COALESCE(round(sum(monto) FILTER (WHERE estado = 'Sustentado'), 2), 0)::float8,
	       COALESCE(round(sum(monto) FILTER (WHERE estado IN ('Registrado', 'Pagado')), 2), 0)::float8,
	       COALESCE(round(100 * sum(monto) FILTER (WHERE estado = 'Sustentado') / NULLIF(sum(monto), 0), 1), 0)::float8,
	       (count(*) FILTER (WHERE estado = 'Sustentado'))::int,
	       COALESCE(round(100.0 * count(*) FILTER (WHERE estado = 'Sustentado') / NULLIF(count(*), 0), 1), 0)::float8
	FROM gasto
	WHERE fecha BETWEEN $1::date AND $2::date`

const sqlPorMes = `
	WITH meses AS (
		SELECT generate_series(date_trunc('month', $1::date), date_trunc('month', $2::date), interval '1 month')::date AS mes
	),
	gastos_mes AS (
		SELECT date_trunc('month', fecha)::date AS mes, estado, monto
		FROM gasto
		WHERE fecha BETWEEN $1::date AND $2::date
	)
	SELECT to_char(m.mes, 'YYYY-MM'),
	       COALESCE(round(sum(g.monto) FILTER (WHERE g.estado = 'Registrado'), 2), 0)::float8,
	       COALESCE(round(sum(g.monto) FILTER (WHERE g.estado = 'Pagado'), 2), 0)::float8,
	       COALESCE(round(sum(g.monto) FILTER (WHERE g.estado = 'Sustentado'), 2), 0)::float8,
	       COALESCE(round(sum(g.monto), 2), 0)::float8
	FROM meses m
	LEFT JOIN gastos_mes g ON g.mes = m.mes
	GROUP BY m.mes
	ORDER BY m.mes`

const sqlPorEstado = `
	WITH estados (estado, orden) AS (
		VALUES ('Registrado', 1), ('Pagado', 2), ('Sustentado', 3)
	),
	por_estado AS (
		SELECT e.estado, e.orden, count(g.id) AS cantidad, COALESCE(sum(g.monto), 0) AS monto
		FROM estados e
		LEFT JOIN gasto g ON g.estado = e.estado AND g.fecha BETWEEN $1::date AND $2::date
		GROUP BY e.estado, e.orden
	)
	SELECT estado,
	       cantidad::int,
	       round(monto, 2)::float8,
	       COALESCE(round(100 * monto / NULLIF(sum(monto) OVER (), 0), 1), 0)::float8
	FROM por_estado
	ORDER BY orden`

// sqlRanking agrupa por el id de una dimensión (dos homónimos son dos filas), muestra su
// etiqueta y numera con RANK(). Los %s son las constantes clave* y etiqueta*, nunca datos del usuario.
const sqlRanking = `
	WITH base AS (
		SELECT g.monto, %s AS clave, %s AS etiqueta` + desdeGastos + `
	),
	agrupado AS (
		SELECT clave, etiqueta, count(*) AS cantidad, sum(monto) AS monto
		FROM base
		GROUP BY clave, etiqueta
	),
	ranqueado AS (
		SELECT etiqueta, cantidad, monto,
		       rank() OVER (ORDER BY monto DESC) AS puesto,
		       100 * monto / NULLIF(sum(monto) OVER (), 0) AS porcentaje
		FROM agrupado
	)
	SELECT puesto::int, etiqueta, cantidad::int, round(monto, 2)::float8, COALESCE(round(porcentaje, 1), 0)::float8
	FROM ranqueado
	ORDER BY monto DESC, etiqueta ASC
	LIMIT 10`

var sqlDetalle = `
	SELECT g.id, g.fecha, ` + etiquetaCliente + `, g.viaje_id, ` + etiquetaUnidad + `, ` + etiquetaConductor + `, g.estado, round(g.monto, 2)::float8` +
	desdeGastos + `
	ORDER BY g.fecha, g.id`

// Consultar calcula el resumen del filtro en una sola transacción de solo lectura
// (REPEATABLE READ): todas las cifras salen de la misma foto de los datos.
func Consultar(ctx context.Context, db Iniciador, f Filtro, conDetalle bool) (*Resumen, error) {
	tx, err := db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	r := &Resumen{Desde: f.DesdeISO(), Hasta: f.HastaISO()}
	args := []any{f.Desde, f.Hasta}

	i := &r.Indicadores
	if err := tx.QueryRow(ctx, sqlIndicadores, args...).Scan(&i.Cantidad, &i.Total, &i.Promedio, &i.Sustentado, &i.Pendiente, &i.PorcentajeSustentado, &i.GastosSustentados, &i.ISC); err != nil {
		return nil, fmt.Errorf("indicadores: %w", err)
	}
	if r.PorMes, err = consultarFilas(ctx, tx, sqlPorMes, args, func(row pgx.CollectableRow) (Mes, error) {
		var m Mes
		err := row.Scan(&m.Mes, &m.Registrado, &m.Pagado, &m.Sustentado, &m.Total)
		return m, err
	}); err != nil {
		return nil, fmt.Errorf("por mes: %w", err)
	}
	if r.PorEstado, err = consultarFilas(ctx, tx, sqlPorEstado, args, func(row pgx.CollectableRow) (PorEstado, error) {
		var e PorEstado
		err := row.Scan(&e.Estado, &e.Cantidad, &e.Monto, &e.Porcentaje)
		return e, err
	}); err != nil {
		return nil, fmt.Errorf("por estado: %w", err)
	}
	rankings := []struct {
		destino         *[]FilaRanking
		clave, etiqueta string
	}{
		{&r.PorCliente, "g.cliente_id", etiquetaCliente},
		{&r.PorUnidad, "u.id_unidad", etiquetaUnidad},
		{&r.PorConductor, "d.id_conductor", etiquetaConductor},
	}
	for _, rk := range rankings {
		if *rk.destino, err = consultarFilas(ctx, tx, fmt.Sprintf(sqlRanking, rk.clave, rk.etiqueta), args, func(row pgx.CollectableRow) (FilaRanking, error) {
			var fr FilaRanking
			err := row.Scan(&fr.Puesto, &fr.Etiqueta, &fr.Cantidad, &fr.Monto, &fr.Porcentaje)
			return fr, err
		}); err != nil {
			return nil, fmt.Errorf("ranking: %w", err)
		}
	}
	if conDetalle {
		if r.Detalle, err = consultarFilas(ctx, tx, sqlDetalle, args, func(row pgx.CollectableRow) (FilaDetalle, error) {
			var d FilaDetalle
			err := row.Scan(&d.ID, &d.Fecha, &d.Cliente, &d.ViajeID, &d.Unidad, &d.Conductor, &d.Estado, &d.Monto)
			return d, err
		}); err != nil {
			return nil, fmt.Errorf("detalle: %w", err)
		}
	}
	return r, tx.Commit(ctx)
}

func consultarFilas[T any](ctx context.Context, tx pgx.Tx, sql string, args []any, leer pgx.RowToFunc[T]) ([]T, error) {
	rows, err := tx.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	filas, err := pgx.CollectRows(rows, leer)
	if err != nil {
		return nil, err
	}
	if filas == nil {
		filas = []T{}
	}
	return filas, nil
}

// FiltroDetalle son los filtros del listado de gastos exportable: todos opcionales (cero, vacío o
// nil significa «sin filtro»).
type FiltroDetalle struct {
	Desde, Hasta *time.Time
	ViajeID      int
	ClienteID    int
	Estado       string
}

// ConsultarDetalle devuelve los gastos que cumplen los filtros, con los datos de su viaje, ordenados
// por fecha y número. Corre en una transacción de solo lectura, como el resto del módulo.
func ConsultarDetalle(ctx context.Context, db Iniciador, f FiltroDetalle) ([]FilaDetalle, error) {
	var donde []string
	var args []any
	agregar := func(condicion string, valor any) {
		args = append(args, valor)
		donde = append(donde, fmt.Sprintf(condicion, len(args)))
	}
	if f.Desde != nil {
		agregar("g.fecha >= $%d::date", *f.Desde)
	}
	if f.Hasta != nil {
		agregar("g.fecha <= $%d::date", *f.Hasta)
	}
	if f.ViajeID > 0 {
		agregar("g.viaje_id = $%d", f.ViajeID)
	}
	if f.ClienteID > 0 {
		agregar("g.cliente_id = $%d", f.ClienteID)
	}
	if f.Estado != "" {
		agregar("g.estado = $%d", f.Estado)
	}
	consulta := `SELECT g.id, g.fecha, ` + etiquetaCliente + `, g.viaje_id, ` + etiquetaUnidad + `, ` + etiquetaConductor + `, g.estado, round(g.monto, 2)::float8` + unionesGastos
	if len(donde) > 0 {
		consulta += " WHERE " + strings.Join(donde, " AND ")
	}
	consulta += " ORDER BY g.fecha, g.id"

	tx, err := db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	filas, err := consultarFilas(ctx, tx, consulta, args, func(row pgx.CollectableRow) (FilaDetalle, error) {
		var d FilaDetalle
		err := row.Scan(&d.ID, &d.Fecha, &d.Cliente, &d.ViajeID, &d.Unidad, &d.Conductor, &d.Estado, &d.Monto)
		return d, err
	})
	if err != nil {
		return nil, err
	}
	return filas, tx.Commit(ctx)
}
