package notificaciones

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// ahoraPrueba es un jueves a las 10:00 en Lima (15:00 UTC).
var ahoraPrueba = time.Date(2026, 10, 22, 15, 0, 0, 0, time.UTC)

func TestLimiteSinComprobanteSonTresDiasExactos(t *testing.T) {
	quiero := time.Date(2026, 10, 19, 15, 0, 0, 0, time.UTC)
	if got := LimiteSinComprobante(ahoraPrueba); !got.Equal(quiero) {
		t.Errorf("LimiteSinComprobante = %v, quiero %v", got, quiero)
	}
}

func TestLimiteCobroEsHoyMasDosDiasEnLima(t *testing.T) {
	// 22/10 a las 10:00 en Lima: el límite es el 24/10.
	if got := LimiteCobro(ahoraPrueba).Format("2006-01-02"); got != "2026-10-24" {
		t.Errorf("LimiteCobro = %s, quiero 2026-10-24", got)
	}
	// Las 23:30 en Lima ya son el día siguiente en UTC: «hoy» sigue siendo el de Lima.
	tarde := time.Date(2026, 10, 23, 4, 30, 0, 0, time.UTC) // 22/10 23:30 en Lima
	if got := LimiteCobro(tarde).Format("2006-01-02"); got != "2026-10-24" {
		t.Errorf("LimiteCobro de noche = %s, quiero 2026-10-24", got)
	}
	// Pasada la medianoche de Lima cambia el día.
	madrugada := time.Date(2026, 10, 23, 5, 0, 0, 0, time.UTC) // 23/10 00:00 en Lima
	if got := LimiteCobro(madrugada).Format("2006-01-02"); got != "2026-10-25" {
		t.Errorf("LimiteCobro de madrugada = %s, quiero 2026-10-25", got)
	}
}

func TestMensajeSinComprobante(t *testing.T) {
	got := MensajeSinComprobante(12, 85.5)
	quiero := "El gasto N° 12 (S/ 85.50) lleva más de 3 días sin comprobante de pago."
	if got != quiero {
		t.Errorf("mensaje = %q", got)
	}
}

func TestMensajeCobroSegunLosDiasQueFaltan(t *testing.T) {
	hoy := time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC)
	dia := func(d int) time.Time { return hoy.AddDate(0, 0, d) }
	casos := []struct {
		fecha time.Time
		texto string
	}{
		{dia(2), "Faltan 2 días para el cobro del gasto N° 7 (24/10/2026) y su comprobante aún no está aprobado."},
		{dia(1), "Mañana es el cobro del gasto N° 7 (23/10/2026) y su comprobante aún no está aprobado."},
		{dia(0), "Hoy es el cobro del gasto N° 7 (22/10/2026) y su comprobante aún no está aprobado."},
		{dia(-1), "El cobro del gasto N° 7 (21/10/2026) venció hace 1 día y su comprobante aún no está aprobado."},
		{dia(-5), "El cobro del gasto N° 7 (17/10/2026) venció hace 5 días y su comprobante aún no está aprobado."},
	}
	for _, c := range casos {
		if got := MensajeCobro(7, c.fecha, hoy); got != c.texto {
			t.Errorf("fecha %s: %q, quiero %q", c.fecha.Format("2006-01-02"), got, c.texto)
		}
	}
}

func TestEjecutarHaceUnaPasadaAlArrancarYOtraEnCadaTic(t *testing.T) {
	var pasadas atomic.Int32
	hecho := make(chan struct{})
	generar := func(context.Context) (Resultado, error) {
		if pasadas.Add(1) == 3 {
			close(hecho)
		}
		return Resultado{}, nil
	}
	ctx, cancelar := context.WithCancel(context.Background())
	termino := make(chan struct{})
	go func() {
		Ejecutar(ctx, 10*time.Millisecond, generar, func(string, ...any) {})
		close(termino)
	}()
	select {
	case <-hecho:
	case <-time.After(2 * time.Second):
		t.Fatalf("solo hubo %d pasadas", pasadas.Load())
	}
	cancelar()
	select {
	case <-termino:
	case <-time.After(2 * time.Second):
		t.Fatal("Ejecutar no terminó al cancelar el contexto")
	}
}

func TestEjecutarHaceLaPrimeraPasadaSinEsperarElIntervalo(t *testing.T) {
	primera := make(chan struct{}, 1)
	generar := func(context.Context) (Resultado, error) {
		select {
		case primera <- struct{}{}:
		default:
		}
		return Resultado{}, nil
	}
	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()
	go Ejecutar(ctx, time.Hour, generar, func(string, ...any) {})
	select {
	case <-primera:
	case <-time.After(2 * time.Second):
		t.Fatal("no hubo pasada inmediata con un intervalo de una hora")
	}
}

func TestEjecutarSigueDespuesDeUnError(t *testing.T) {
	var pasadas atomic.Int32
	avisos := make(chan string, 10)
	hecho := make(chan struct{})
	generar := func(context.Context) (Resultado, error) {
		if pasadas.Add(1) == 1 {
			return Resultado{}, errors.New("base caída")
		}
		if pasadas.Load() == 2 {
			close(hecho)
		}
		return Resultado{}, nil
	}
	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()
	go Ejecutar(ctx, 10*time.Millisecond, generar, func(f string, a ...any) { avisos <- f })
	select {
	case <-hecho:
	case <-time.After(2 * time.Second):
		t.Fatal("tras el error no hubo otra pasada")
	}
	if len(avisos) == 0 {
		t.Error("el error no se avisó")
	}
}
