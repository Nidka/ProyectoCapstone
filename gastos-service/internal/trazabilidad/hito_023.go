package trazabilidad

// Hito023 conserva la relación entre el cambio 23 y la fuente entregada.
var Hito023 = Cambio{
	Numero:      23,
	Fecha:       "20/10/2026 al 22/10/2026",
	Descripcion: "Extractor de monto, fecha y RUC y corrección manual (OCR-02, OCR-03)",
	Motivo:      "Permitir que el encargado confirme los datos antes de aprobar",
	Responsable: "Valera Lizarraga, Francisco Kadin / Mantilla Arqueros, Frank Piero",
	Rutas:       []string{"gastos-service/internal/ocr", "frontend/src/app/features/gastos"},
	Fuente:      "Daily 20 a Daily 22",
}
