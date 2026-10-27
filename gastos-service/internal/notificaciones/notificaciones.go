// Package notificaciones genera los avisos para el Encargado de Logística (REQ-20, REQ-21, RNF-8):
// gastos que llevan más de 3 días sin comprobante y gastos con el cobro cerca (o vencido) cuyo
// comprobante aún no está aprobado. Una pasada del generador mira la base y crea los avisos que
// faltan, sin repetir los que ya existen; main.go la repite cada hora, de modo que un aviso
// aparece siempre dentro de las 24 horas siguientes a que se cumple la condición.
package notificaciones

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gastos-service/internal/bitacora"
	"gastos-service/internal/reportes"

	"github.com/jackc/pgx/v5"
)

const (
	TipoSinComprobante = "sin_comprobante"
	TipoCobroProximo   = "cobro_proximo"

	// DiasSinComprobante: un gasto avisa cuando pasan MÁS de estos días desde su registro sin
	// comprobante; justo a los 3 días todavía no.
	DiasSinComprobante = 3
	// DiasAvisoCobro: un gasto avisa cuando su fecha de cobro está a ESTOS días o menos de hoy
	// (hoy en Lima), incluida una fecha ya vencida.
	DiasAvisoCobro = 2
)

// autor firma en la bitácora los avisos que crea el generador: no los pidió ningún usuario.
var autor = bitacora.Actor{Nombre: "Sistema"}

// LimiteSinComprobante es el instante antes del cual un gasto sin comprobante ya cuenta como atrasado.
func LimiteSinComprobante(ahora time.Time) time.Time {
	return ahora.Add(-DiasSinComprobante * 24 * time.Hour)
}

// LimiteCobro es la última fecha de cobro que avisa hoy: hoy (en Lima) más DiasAvisoCobro días.
// Se devuelve a medianoche UTC porque solo importa el día.
func LimiteCobro(ahora time.Time) time.Time {
	return hoyEnLima(ahora).AddDate(0, 0, DiasAvisoCobro)
}

// hoyEnLima es la fecha de hoy en Lima, a medianoche UTC.
func hoyEnLima(ahora time.Time) time.Time {
	a, m, d := ahora.In(reportes.Lima).Date()
	return time.Date(a, m, d, 0, 0, 0, 0, time.UTC)
}

// MensajeSinComprobante es el texto del aviso REQ-20.
func MensajeSinComprobante(gastoID int, monto float64) string {
	return fmt.Sprintf("El gasto N° %d (S/ %.2f) lleva más de %d días sin comprobante de pago.", gastoID, monto, DiasSinComprobante)
}

// MensajeCobro es el texto del aviso REQ-21 según los días que faltan para el cobro (hoy y
// fechaCobro son solo fechas).
func MensajeCobro(gastoID int, fechaCobro, hoy time.Time) string {
	dias := int(fechaCobro.Sub(hoy).Hours() / 24)
	fecha := fechaCobro.Format("02/01/2006")
	const cola = "y su comprobante aún no está aprobado."
	switch {
	case dias > 1:
		return fmt.Sprintf("Faltan %d días para el cobro del gasto N° %d (%s) %s", dias, gastoID, fecha, cola)
	case dias == 1:
		return fmt.Sprintf("Mañana es el cobro del gasto N° %d (%s) %s", gastoID, fecha, cola)
	case dias == 0:
		return fmt.Sprintf("Hoy es el cobro del gasto N° %d (%s) %s", gastoID, fecha, cola)
	case dias == -1:
		return fmt.Sprintf("El cobro del gasto N° %d (%s) venció hace 1 día %s", gastoID, fecha, cola)
	}
	return fmt.Sprintf("El cobro del gasto N° %d (%s) venció hace %d días %s", gastoID, fecha, -dias, cola)
}

// Base es lo que necesita el generador: lo cumple el pool.
type Base interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// Generador crea los avisos pendientes. Ahora es el reloj: las pruebas lo fijan.
type Generador struct {
	DB    Base
	Ahora func() time.Time
}

// Resultado cuenta los avisos NUEVOS de una pasada, por tipo.
type Resultado struct {
	SinComprobante int
	CobroProximo   int
}

type candidato struct {
	gastoID    int
	monto      float64
	fechaCobro *time.Time
}

// Generar hace una pasada. Es todo o nada: si algo falla (incluida la bitácora) no queda ningún
// aviso de la pasada y la siguiente lo reintenta.
//
// REQ-20: gasto sin ningún comprobante Pendiente ni Aprobado, registrado hace más de 3 días. Un
// comprobante rechazado no cuenta como adjunto: el gasto vuelve a Registrado y sigue faltando.
// REQ-21: gasto con fecha de cobro a 2 días o menos (o vencida) que todavía no está Sustentado, es
// decir, sin comprobante aprobado: sin comprobante, o con uno pendiente de revisión o rechazado.
// Un aviso ya creado no se repite ni se borra aunque la condición deje de cumplirse.
func (g *Generador) Generar(ctx context.Context) (Resultado, error) {
	var res Resultado
	ahora := g.Ahora()
	tx, err := g.DB.Begin(ctx)
	if err != nil {
		return res, err
	}
	defer tx.Rollback(ctx)

	atrasados, err := candidatos(ctx, tx, `
		SELECT g.id, g.monto, g.fecha_cobro
		FROM gasto g
		WHERE g.creado_en::timestamptz < $1
		  AND NOT EXISTS (SELECT 1 FROM comprobante c WHERE c.gasto_id = g.id AND c.estado IN ('Pendiente', 'Aprobado'))
		  AND NOT EXISTS (SELECT 1 FROM notificacion n WHERE n.tipo = $2 AND n.id_gasto = g.id)
		ORDER BY g.id`, LimiteSinComprobante(ahora), TipoSinComprobante)
	if err != nil {
		return res, err
	}
	for _, c := range atrasados {
		nueva, err := crear(ctx, tx, TipoSinComprobante, c.gastoID, MensajeSinComprobante(c.gastoID, c.monto), ahora)
		if err != nil {
			return res, err
		}
		if nueva {
			res.SinComprobante++
		}
	}

	hoy := hoyEnLima(ahora)
	porCobrar, err := candidatos(ctx, tx, `
		SELECT g.id, g.monto, g.fecha_cobro
		FROM gasto g
		WHERE g.fecha_cobro IS NOT NULL AND g.fecha_cobro <= $1::date
		  AND g.estado <> 'Sustentado'
		  AND NOT EXISTS (SELECT 1 FROM notificacion n WHERE n.tipo = $2 AND n.id_gasto = g.id)
		ORDER BY g.id`, LimiteCobro(ahora), TipoCobroProximo)
	if err != nil {
		return res, err
	}
	for _, c := range porCobrar {
		nueva, err := crear(ctx, tx, TipoCobroProximo, c.gastoID, MensajeCobro(c.gastoID, *c.fechaCobro, hoy), ahora)
		if err != nil {
			return res, err
		}
		if nueva {
			res.CobroProximo++
		}
	}

	return res, tx.Commit(ctx)
}

func candidatos(ctx context.Context, tx pgx.Tx, sql string, args ...any) ([]candidato, error) {
	filas, err := tx.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer filas.Close()
	var out []candidato
	for filas.Next() {
		var c candidato
		if err := filas.Scan(&c.gastoID, &c.monto, &c.fechaCobro); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, filas.Err()
}

// crear inserta el aviso y su entrada de bitácora. Si ya existía (otra pasada se adelantó) no hace
// nada y devuelve false.
func crear(ctx context.Context, tx pgx.Tx, tipo string, gastoID int, mensaje string, ahora time.Time) (bool, error) {
	var id int
	err := tx.QueryRow(ctx, `
		INSERT INTO notificacion (tipo, id_gasto, mensaje, creada_en)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (tipo, id_gasto) DO NOTHING
		RETURNING id`, tipo, gastoID, mensaje, ahora).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	err = bitacora.Registrar(ctx, tx, bitacora.Entrada{
		Actor: autor, Accion: "notificacion.generar", Entidad: "notificacion", IDEntidad: int64(id),
		Detalle: map[string]any{"despues": map[string]any{"tipo": tipo, "id_gasto": gastoID, "mensaje": mensaje}},
	})
	return err == nil, err
}

// Ejecutar hace una pasada al arrancar y otra cada vez que pasa el intervalo, hasta que se cancele
// el contexto. Un error de una pasada se avisa y no detiene las siguientes.
func Ejecutar(ctx context.Context, cada time.Duration, generar func(context.Context) (Resultado, error), avisar func(formato string, args ...any)) {
	pasada := func() {
		res, err := generar(ctx)
		switch {
		case ctx.Err() != nil:
		case err != nil:
			avisar("notificaciones: no se pudo generar: %v", err)
		case res.SinComprobante+res.CobroProximo > 0:
			avisar("notificaciones: %d sin comprobante y %d por cobrar", res.SinComprobante, res.CobroProximo)
		}
	}
	pasada()
	reloj := time.NewTicker(cada)
	defer reloj.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-reloj.C:
			pasada()
		}
	}
}
