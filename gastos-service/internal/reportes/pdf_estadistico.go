package reportes

import (
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/signintech/gopdf"
)

// PDFEstadistico genera el reporte estadístico (A4 vertical): indicadores, gasto por mes,
// distribución por estado y los rankings por cliente, unidad y conductor.
func PDFEstadistico(r *Resumen, ahora time.Time) ([]byte, error) {
	d, err := nuevoDocumento(gopdf.PageSizeA4, "Reporte estadístico de gastos", r, ahora)
	if err != nil {
		return nil, err
	}
	d.indicadores(r.Indicadores)
	if r.Indicadores.Cantidad == 0 {
		d.parrafoVacio()
		return d.finalizar()
	}
	d.graficoPorMes(r.PorMes)
	d.distribucionEstados(r.PorEstado)
	d.ranking("Gasto por cliente", "Cliente", r.PorCliente)
	d.ranking("Gasto por unidad", "Unidad (placa)", r.PorUnidad)
	d.ranking("Gasto por conductor", "Conductor", r.PorConductor)
	return d.finalizar()
}

func (d *documento) indicadores(i Indicadores) {
	const separacion, altoRecuadro = 10.0, 68.0
	ancho := (d.anchoUtil() - 3*separacion) / 4
	recuadros := []struct{ etiqueta, valor, nota, nota2 string }{
		{"Total gastado", FormatoMonto(i.Total), "", ""},
		{"Gastos registrados", strconv.Itoa(i.Cantidad), "", ""},
		{"Gasto promedio", FormatoMonto(i.Promedio), "", ""},
		{"ISC · gastos sustentados", FormatoPorcentaje(i.ISC), strconv.Itoa(i.GastosSustentados) + " de " + strconv.Itoa(i.Cantidad) + " gastos", "Pendiente: " + FormatoMonto(i.Pendiente)},
	}
	for n, rc := range recuadros {
		x := margen + float64(n)*(ancho+separacion)
		d.rect(x, d.y, ancho, altoRecuadro, colorPanel2)
		d.texto(x+10, d.y+8, ancho-20, 10, rc.etiqueta, false, 8, colorTinta2, gopdf.Left)
		d.texto(x+10, d.y+21, ancho-20, 18, rc.valor, true, d.tamanoQueCabe(rc.valor, ancho-20, true, 14, 8), colorTinta, gopdf.Left)
		if rc.nota != "" {
			d.texto(x+10, d.y+41, ancho-20, 10, rc.nota, false, 7.5, colorTinta3, gopdf.Left)
		}
		if rc.nota2 != "" {
			d.texto(x+10, d.y+52, ancho-20, 10, rc.nota2, false, 7.5, colorTinta3, gopdf.Left)
		}
	}
	d.y += altoRecuadro + 22
}

// escalaAgradable redondea el máximo hacia arriba a 4 pasos de 1, 2, 2.5 o 5 por potencia de 10.
func escalaAgradable(maximo float64) (tope, paso float64) {
	if maximo <= 0 {
		return 4, 1
	}
	bruto := maximo / 4
	potencia := math.Pow(10, math.Floor(math.Log10(bruto)))
	for _, m := range []float64{1, 2, 2.5, 5, 10} {
		if m*potencia >= bruto {
			paso = m * potencia
			break
		}
	}
	return 4 * paso, paso
}

func montoEje(v float64) string { return strings.TrimSuffix(FormatoMonto(v), ".00") }

func (d *documento) graficoPorMes(meses []Mes) {
	const altoGrafico, anchoEje = 150.0, 58.0
	d.tituloSeccion("Gasto por mes", altoGrafico+50)
	tope, paso := escalaAgradable(maximoMensual(meses))
	x0, y0 := margen+anchoEje, d.y+altoGrafico
	anchoPlot := d.anchoUtil() - anchoEje
	for k := 0; k <= 4; k++ {
		y := y0 - float64(k)*paso/tope*altoGrafico
		d.linea(x0, y, x0+anchoPlot, y, colorLinea)
		d.texto(margen, y-5, anchoEje-6, 10, montoEje(float64(k)*paso), false, 7, colorTinta3, gopdf.Right)
	}
	if len(meses) > 0 {
		hueco := anchoPlot / float64(len(meses))
		anchoBarra := math.Min(hueco*0.6, 26)
		saltoEtiqueta := 1
		if len(meses) > 12 {
			saltoEtiqueta = 2
		}
		for n, m := range meses {
			x := x0 + float64(n)*hueco + (hueco-anchoBarra)/2
			y := y0
			for _, estado := range ordenEstados {
				alto := montoDeEstado(m, estado) / tope * altoGrafico
				d.rect(x, y-alto, anchoBarra, alto, coloresEstado[estado])
				y -= alto
			}
			if n%saltoEtiqueta == 0 {
				anchoEtiqueta := hueco * float64(saltoEtiqueta)
				centro := x0 + float64(n)*hueco + hueco/2
				d.texto(centro-anchoEtiqueta/2, y0+3, anchoEtiqueta, 10, MesCorto(m.Mes), false, 7, colorTinta2, gopdf.Center)
			}
		}
	}
	d.y = y0 + 18
	d.leyenda()
	d.y += 20
}

// maximoMensual es la columna más alta tal como se dibuja: la suma de sus segmentos.
func maximoMensual(meses []Mes) float64 {
	maximo := 0.0
	for _, m := range meses {
		maximo = math.Max(maximo, m.Registrado+m.Pagado+m.Sustentado)
	}
	return maximo
}

func montoDeEstado(m Mes, estado string) float64 {
	switch estado {
	case "Registrado":
		return m.Registrado
	case "Pagado":
		return m.Pagado
	default:
		return m.Sustentado
	}
}

func (d *documento) leyenda() {
	x := margen + 58
	for _, estado := range ordenEstados {
		d.rect(x, d.y+2, 8, 8, coloresEstado[estado])
		d.texto(x+12, d.y, 80, 12, estado, false, 8, colorTinta2, gopdf.Left)
		x += 90
	}
	d.y += 14
}

func (d *documento) distribucionEstados(estados []PorEstado) {
	d.tituloSeccion("Distribución por estado", altoFila*float64(len(estados)+1))
	columnas := []columna{{"Estado", 120, gopdf.Left}, {"Gastos", 70, gopdf.Right}, {"Monto", 110, gopdf.Right}, {"Participación", d.anchoUtil() - 300, gopdf.Left}}
	d.encabezadoTabla(columnas)
	for _, e := range estados {
		d.rect(margen+4, d.y+4, 8, 8, coloresEstado[e.Estado])
		d.texto(margen+16, d.y, 100, altoFila, e.Estado, false, 8.5, colorTinta, gopdf.Left)
		d.texto(margen+120, d.y, 66, altoFila, strconv.Itoa(e.Cantidad), false, 8.5, colorTinta, gopdf.Right)
		d.texto(margen+190, d.y, 106, altoFila, FormatoMonto(e.Monto), false, 8.5, colorTinta, gopdf.Right)
		d.barraPorcentaje(margen+300, columnas[3].ancho, e.Porcentaje, coloresEstado[e.Estado])
		d.y += altoFila
	}
	d.y += 18
}

// barraPorcentaje dibuja una barra proporcional y el porcentaje a su derecha.
func (d *documento) barraPorcentaje(x, ancho, porcentaje float64, c color) {
	anchoBarra := ancho - 56
	d.rect(x+4, d.y+5, anchoBarra, 6, colorPanel2)
	d.rect(x+4, d.y+5, anchoBarra*math.Min(porcentaje, 100)/100, 6, c)
	d.texto(x+anchoBarra+8, d.y, 44, altoFila, FormatoPorcentaje(porcentaje), false, 8, colorTinta2, gopdf.Right)
}

func (d *documento) ranking(titulo, nombreColumna string, filas []FilaRanking) {
	d.tituloSeccion(titulo, altoFila*float64(len(filas)+2))
	columnas := []columna{{"Puesto", 44, gopdf.Left}, {nombreColumna, 190, gopdf.Left}, {"Gastos", 54, gopdf.Right}, {"Monto", 90, gopdf.Right}, {"%", d.anchoUtil() - 378, gopdf.Left}}
	d.encabezadoTabla(columnas)
	for _, f := range filas {
		d.fila(columnas[:4], []string{strconv.Itoa(f.Puesto), f.Etiqueta, strconv.Itoa(f.Cantidad), FormatoMonto(f.Monto)}, false)
		d.y -= altoFila
		d.barraPorcentaje(margen+378, columnas[4].ancho, f.Porcentaje, colorTinta2)
		d.y += altoFila
	}
	d.y += 18
}
