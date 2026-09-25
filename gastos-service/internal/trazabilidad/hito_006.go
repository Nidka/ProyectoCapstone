package trazabilidad

// Hito006 conserva la relación entre el cambio 6 y la fuente entregada.
var Hito006 = Cambio{
	Numero:      6,
	Fecha:       "25/09/2026",
	Descripcion: "Endpoint de subida de comprobantes y pruebas de PDF corruptos (HU-04, HU-06)",
	Motivo:      "Cumplir REQ-6 a REQ-9",
	Responsable: "Valera Lizarraga, Francisco Kadin / Mantilla Arqueros, Frank Piero",
	Rutas:       []string{"gastos-service/internal/handlers"},
	Fuente:      "Daily 3, Acta 001",
}
