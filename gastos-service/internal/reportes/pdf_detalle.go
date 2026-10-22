package reportes

import (
	"strconv"
	"time"

	"github.com/signintech/gopdf"
)

// PDFDetalle genera el listado de gastos del período (A4 horizontal) con los totales por estado.
func PDFDetalle(r *Resumen, ahora time.Time) ([]byte, error) {
	d, err := nuevoDocumento(gopdf.PageSizeA4Landscape, "Detalle de gastos", r, ahora)
	if err != nil {
		return nil, err
	}
	if len(r.Detalle) == 0 {
		d.parrafoVacio()
		return d.finalizar()
	}
	columnas := []columna{
		{"N°", 40, gopdf.Left}, {"Fecha", 62, gopdf.Left}, {"Cliente", 175, gopdf.Left}, {"Viaje", 45, gopdf.Right},
		{"Unidad", 70, gopdf.Left}, {"Conductor", 165, gopdf.Left}, {"Estado", 80, gopdf.Left},
		{"Monto", d.anchoUtil() - 637, gopdf.Right},
	}
	d.encabezadoTabla(columnas)
	d.alNuevaPagina = func() { d.encabezadoTabla(columnas) }
	for n, g := range r.Detalle {
		d.asegurar(altoFila)
		if n%2 == 1 {
			d.rect(margen, d.y, d.anchoUtil(), altoFila, colorPanel2)
		}
		d.fila(columnas, []string{
			strconv.Itoa(g.ID), FormatoFecha(g.Fecha), g.Cliente, strconv.Itoa(g.ViajeID),
			g.Unidad, g.Conductor, g.Estado, FormatoMonto(g.Monto),
		}, false)
	}
	d.alNuevaPagina = nil

	d.asegurar(altoFila*float64(len(r.PorEstado)+1) + 10)
	d.y += 6
	d.linea(margen, d.y, d.ancho-margen, d.y, colorLinea)
	totales := []columna{{"", d.anchoUtil() - 160, gopdf.Right}, {"", 160, gopdf.Right}}
	for _, e := range r.PorEstado {
		d.fila(totales, []string{"Total " + e.Estado + " (" + strconv.Itoa(e.Cantidad) + " gastos)", FormatoMonto(e.Monto)}, false)
	}
	d.fila(totales, []string{"Total general (" + strconv.Itoa(r.Indicadores.Cantidad) + " gastos)", FormatoMonto(r.Indicadores.Total)}, true)
	return d.finalizar()
}
