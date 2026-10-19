package trazabilidad

// Hito021 conserva la relación entre el cambio 21 y la fuente entregada.
var Hito021 = Cambio{
	Numero:      21,
	Fecha:       "17/10/2026 al 19/10/2026",
	Descripcion: "Integración de Tesseract y lote de veinte comprobantes (HU-17, OCR-01)",
	Motivo:      "Leer comprobantes con precisión mínima del 85 por ciento",
	Responsable: "Valera Lizarraga, Francisco Kadin",
	Rutas:       []string{"gastos-service/internal/ocr"},
	Fuente:      "Daily 17 a Daily 19",
}
