package francisco

// Aporte017 identifica el trabajo documentado en 22/10/2026.
var Aporte017 = Aporte{
	Numero:      17,
	Fecha:       "22/10/2026",
	Descripcion: "Agregar lectura OCR por comprobante con caché",
	Rutas:       []string{"gastos-service/internal/ocr", "gastos-service/migrations/0005_lectura_comprobante.sql"},
	Fuente:      "Especificación OCR 2026-10-22",
}
