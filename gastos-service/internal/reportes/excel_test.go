package reportes

import (
	"bytes"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"
)

func abrirExcel(t *testing.T, datos []byte) *excelize.File {
	t.Helper()
	f, err := excelize.OpenReader(bytes.NewReader(datos))
	if err != nil {
		t.Fatalf("no es un .xlsx válido: %v", err)
	}
	t.Cleanup(func() { _ = f.Close() })
	return f
}

func filasDeEjemplo() []FilaDetalle {
	return []FilaDetalle{
		{ID: 7, Fecha: time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC), Cliente: "Agro Norte SAC", ViajeID: 14, Unidad: "T1A-123", Conductor: "José Ñique", Estado: "Sustentado", Monto: 85.5},
		{ID: 8, Fecha: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), Cliente: "Minera Sur", ViajeID: 15, Unidad: "Sin asignar", Conductor: "Sin asignar", Estado: "Pagado", Monto: 14.5},
		{ID: 9, Fecha: time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC), Cliente: "Minera Sur", ViajeID: 15, Unidad: "T9", Conductor: "Ana", Estado: "Sustentado", Monto: 100},
	}
}

func TestExcelDeGastosTieneDetalleTotalesYResumen(t *testing.T) {
	datos, err := ExcelGastos(filasDeEjemplo(), "Cliente N° 2 · Estado: Pagado", ahoraPrueba)
	if err != nil {
		t.Fatal(err)
	}
	f := abrirExcel(t, datos)
	if hojas := f.GetSheetList(); len(hojas) != 2 || hojas[0] != "Gastos" || hojas[1] != "Resumen" {
		t.Fatalf("hojas = %v", hojas)
	}
	encabezado, _ := f.GetRows("Gastos")
	if got := encabezado[0]; len(got) != 8 || got[0] != "N°" || got[2] != "Cliente" || got[7] != "Monto" {
		t.Fatalf("encabezado = %v", got)
	}
	if v, _ := f.GetCellValue("Gastos", "C2"); v != "Agro Norte SAC" {
		t.Errorf("C2 = %q", v)
	}
	if v, _ := f.GetCellValue("Gastos", "D2"); v != "14" {
		t.Errorf("el viaje es un número: %q", v)
	}
	// El monto es un número (no texto): se puede sumar y ordenar en Excel
	if tipo, _ := f.GetCellType("Gastos", "H2"); tipo != excelize.CellTypeNumber && tipo != excelize.CellTypeUnset {
		t.Errorf("tipo de H2 = %v", tipo)
	}
	if formula, _ := f.GetCellFormula("Gastos", "H5"); formula != "SUM(H2:H4)" {
		t.Errorf("fórmula del total = %q", formula)
	}
	if total, err := f.CalcCellValue("Gastos", "H5"); err != nil || total != "S/ 200.00" {
		t.Errorf("total = %q (%v)", total, err)
	}
	if v, _ := f.GetCellValue("Gastos", "G5"); v != "Total" {
		t.Errorf("rótulo del total = %q", v)
	}
	filtros, _ := f.GetCellValue("Resumen", "B3")
	if filtros != "Cliente N° 2 · Estado: Pagado" {
		t.Errorf("filtros en el resumen = %q", filtros)
	}
	resumen, _ := f.GetRows("Resumen")
	var porEstado []string
	for _, fila := range resumen {
		if len(fila) > 0 && (fila[0] == "Sustentado" || fila[0] == "Pagado" || fila[0] == "Registrado") {
			porEstado = append(porEstado, fila[0]+"|"+fila[1])
		}
	}
	if len(porEstado) != 3 || porEstado[0] != "Registrado|0" || porEstado[1] != "Pagado|1" || porEstado[2] != "Sustentado|2" {
		t.Errorf("resumen por estado = %v", porEstado)
	}
}

func TestExcelSinGastosSigueSiendoValido(t *testing.T) {
	datos, err := ExcelGastos(nil, "Sin filtros", ahoraPrueba)
	if err != nil {
		t.Fatal(err)
	}
	f := abrirExcel(t, datos)
	filas, _ := f.GetRows("Gastos")
	if len(filas) < 1 || filas[0][0] != "N°" {
		t.Fatalf("filas = %v", filas)
	}
	if v, _ := f.GetCellValue("Gastos", "G2"); v != "Total" {
		t.Errorf("rótulo del total sin gastos = %q", v)
	}
	if formula, _ := f.GetCellFormula("Gastos", "H2"); formula != "" {
		t.Errorf("sin gastos el total no debe ser una fórmula (sería circular): %q", formula)
	}
}

func TestExcelNoInterpretaTextoComoFormula(t *testing.T) {
	// Un cliente llamado «=HYPERLINK(...)» (o que empiece con +, - o @) debe quedar como texto
	peligrosos := map[string]string{"C2": "=HYPERLINK(\"http://x\")", "E2": "+cmd", "F2": "@x"}
	filas := []FilaDetalle{{ID: 1, Fecha: time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC), Cliente: peligrosos["C2"], Unidad: peligrosos["E2"], Conductor: peligrosos["F2"], Estado: "Pagado", Monto: 1}}
	datos, err := ExcelGastos(filas, "Sin filtros", ahoraPrueba)
	if err != nil {
		t.Fatal(err)
	}
	f := abrirExcel(t, datos)
	for celda, texto := range peligrosos {
		if formula, _ := f.GetCellFormula("Gastos", celda); formula != "" {
			t.Errorf("%s se guardó como fórmula: %q", celda, formula)
		}
		if v, _ := f.GetCellValue("Gastos", celda); v != texto {
			t.Errorf("%s = %q, se esperaba el texto %q", celda, v, texto)
		}
	}
}
