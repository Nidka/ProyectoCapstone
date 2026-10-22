package trazabilidad

// Hito025 conserva la relación entre el cambio 25 y la fuente entregada.
var Hito025 = Cambio{
	Numero:      25,
	Fecha:       "22/10/2026",
	Descripcion: "Lectura OCR por comprobante con caché y endpoint de consulta",
	Motivo:      "Apoyar el checklist sin permitir que el OCR apruebe",
	Responsable: "Valera Lizarraga, Francisco Kadin",
	Rutas:       []string{"gastos-service/internal/ocr", "gastos-service/migrations/0005_lectura_comprobante.sql"},
	Fuente:      "Especificación OCR 2026-10-22",
}
