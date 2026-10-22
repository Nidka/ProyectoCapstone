package reportes

import (
	"fmt"
	"time"

	"github.com/xuri/excelize/v2"
)

var encabezadosExcel = []string{"N°", "Fecha", "Cliente", "Viaje", "Unidad", "Conductor", "Estado", "Monto"}

// ExcelGastos arma el libro de Excel del detalle de gastos (HU-16): la hoja «Gastos» con una fila
// por gasto y su total con fórmula SUM, y la hoja «Resumen» con los filtros aplicados y el total
// por estado. Los textos van como texto (un cliente que empiece con «=» no se ejecuta) y los
// montos y fechas como números y fechas reales, para poder ordenarlos y sumarlos.
func ExcelGastos(filas []FilaDetalle, filtros string, ahora time.Time) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()
	if err := f.SetSheetName("Sheet1", "Gastos"); err != nil {
		return nil, err
	}
	estilos, err := estilosExcel(f)
	if err != nil {
		return nil, err
	}

	for i, titulo := range encabezadosExcel {
		celda, _ := excelize.CoordinatesToCellName(i+1, 1)
		if err := f.SetCellValue("Gastos", celda, titulo); err != nil {
			return nil, err
		}
	}
	_ = f.SetCellStyle("Gastos", "A1", "H1", estilos.encabezado)

	for i, g := range filas {
		fila := i + 2
		valores := []any{g.ID, g.Fecha, g.Cliente, g.ViajeID, g.Unidad, g.Conductor, g.Estado, g.Monto}
		for c, v := range valores {
			celda, _ := excelize.CoordinatesToCellName(c+1, fila)
			if err := f.SetCellValue("Gastos", celda, v); err != nil {
				return nil, err
			}
		}
	}
	ultima := len(filas) + 1
	if ultima > 1 {
		_ = f.SetCellStyle("Gastos", "B2", fmt.Sprintf("B%d", ultima), estilos.fecha)
		_ = f.SetCellStyle("Gastos", "H2", fmt.Sprintf("H%d", ultima), estilos.monto)
	}
	total := ultima + 1
	_ = f.SetCellValue("Gastos", fmt.Sprintf("G%d", total), "Total")
	if ultima > 1 {
		_ = f.SetCellFormula("Gastos", fmt.Sprintf("H%d", total), fmt.Sprintf("SUM(H2:H%d)", ultima))
	} else {
		// Sin gastos no hay rango que sumar (la fórmula se referiría a sí misma)
		_ = f.SetCellValue("Gastos", fmt.Sprintf("H%d", total), 0)
	}
	_ = f.SetCellStyle("Gastos", fmt.Sprintf("G%d", total), fmt.Sprintf("G%d", total), estilos.total)
	_ = f.SetCellStyle("Gastos", fmt.Sprintf("H%d", total), fmt.Sprintf("H%d", total), estilos.totalMonto)

	anchos := map[string]float64{"A": 8, "B": 12, "C": 34, "D": 8, "E": 14, "F": 28, "G": 14, "H": 16}
	for col, ancho := range anchos {
		_ = f.SetColWidth("Gastos", col, col, ancho)
	}
	_ = f.SetPanes("Gastos", &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"})
	if ultima > 1 {
		_ = f.AutoFilter("Gastos", fmt.Sprintf("A1:H%d", ultima), nil)
	}

	if err := hojaResumen(f, estilos, filas, filtros, ahora); err != nil {
		return nil, err
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type estilosHoja struct{ encabezado, fecha, monto, total, totalMonto, titulo int }

func estilosExcel(f *excelize.File) (estilosHoja, error) {
	var e estilosHoja
	var err error
	formatoMonto := `"S/" #,##0.00`
	fecha := "dd/mm/yyyy"
	crear := func(destino *int, estilo *excelize.Style) {
		if err == nil {
			*destino, err = f.NewStyle(estilo)
		}
	}
	crear(&e.encabezado, &excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"E11D48"}},
		Alignment: &excelize.Alignment{Vertical: "center"},
	})
	crear(&e.fecha, &excelize.Style{CustomNumFmt: &fecha, Alignment: &excelize.Alignment{Horizontal: "left"}})
	crear(&e.monto, &excelize.Style{CustomNumFmt: &formatoMonto})
	crear(&e.total, &excelize.Style{Font: &excelize.Font{Bold: true}, Alignment: &excelize.Alignment{Horizontal: "right"}})
	crear(&e.totalMonto, &excelize.Style{Font: &excelize.Font{Bold: true}, CustomNumFmt: &formatoMonto, Border: []excelize.Border{{Type: "top", Color: "1E2124", Style: 1}}})
	crear(&e.titulo, &excelize.Style{Font: &excelize.Font{Bold: true, Size: 14}})
	return e, err
}

func hojaResumen(f *excelize.File, estilos estilosHoja, filas []FilaDetalle, filtros string, ahora time.Time) error {
	if _, err := f.NewSheet("Resumen"); err != nil {
		return err
	}
	_ = f.SetCellValue("Resumen", "A1", "LIZAR SRL · Detalle de gastos")
	_ = f.SetCellStyle("Resumen", "A1", "A1", estilos.titulo)
	_ = f.SetCellValue("Resumen", "A2", "Emitido")
	_ = f.SetCellValue("Resumen", "B2", FormatoEmision(ahora))
	_ = f.SetCellValue("Resumen", "A3", "Filtros")
	_ = f.SetCellValue("Resumen", "B3", filtros)

	_ = f.SetCellValue("Resumen", "A5", "Estado")
	_ = f.SetCellValue("Resumen", "B5", "Gastos")
	_ = f.SetCellValue("Resumen", "C5", "Monto")
	_ = f.SetCellValue("Resumen", "D5", "%")
	_ = f.SetCellStyle("Resumen", "A5", "D5", estilos.encabezado)

	cantidad := map[string]int{}
	monto := map[string]float64{}
	totalMonto := 0.0
	for _, g := range filas {
		cantidad[g.Estado]++
		monto[g.Estado] += g.Monto
		totalMonto += g.Monto
	}
	porcentaje := estilosPorcentaje(f)
	for i, estado := range ordenEstados {
		fila := 6 + i
		pct := 0.0
		if totalMonto > 0 {
			pct = monto[estado] / totalMonto
		}
		_ = f.SetCellValue("Resumen", fmt.Sprintf("A%d", fila), estado)
		_ = f.SetCellValue("Resumen", fmt.Sprintf("B%d", fila), cantidad[estado])
		_ = f.SetCellValue("Resumen", fmt.Sprintf("C%d", fila), monto[estado])
		_ = f.SetCellValue("Resumen", fmt.Sprintf("D%d", fila), pct)
		_ = f.SetCellStyle("Resumen", fmt.Sprintf("C%d", fila), fmt.Sprintf("C%d", fila), estilos.monto)
		_ = f.SetCellStyle("Resumen", fmt.Sprintf("D%d", fila), fmt.Sprintf("D%d", fila), porcentaje)
	}
	_ = f.SetCellValue("Resumen", "A9", "Total")
	_ = f.SetCellFormula("Resumen", "B9", "SUM(B6:B8)")
	_ = f.SetCellFormula("Resumen", "C9", "SUM(C6:C8)")
	_ = f.SetCellStyle("Resumen", "A9", "B9", estilos.total)
	_ = f.SetCellStyle("Resumen", "C9", "C9", estilos.totalMonto)
	_ = f.SetColWidth("Resumen", "A", "A", 16)
	_ = f.SetColWidth("Resumen", "B", "B", 40)
	_ = f.SetColWidth("Resumen", "C", "D", 16)
	return nil
}

func estilosPorcentaje(f *excelize.File) int {
	id, _ := f.NewStyle(&excelize.Style{NumFmt: 10}) // 0.00 %
	return id
}
