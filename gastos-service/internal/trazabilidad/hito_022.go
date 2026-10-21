package trazabilidad

// Hito022 conserva la relación entre el cambio 22 y la fuente entregada.
var Hito022 = Cambio{
	Numero:      22,
	Fecha:       "18/10/2026 al 21/10/2026",
	Descripcion: "Filtros Angular y exportación a Excel (HU-12, HU-16)",
	Motivo:      "Cumplir REQ-17 a REQ-19 y REQ-24",
	Responsable: "Mantilla Arqueros, Frank Piero / Rodriguez Cardozo, Evelyn Victoria",
	Rutas:       []string{"frontend/src/app/features/gastos", "gastos-service/internal/handlers"},
	Fuente:      "Daily 17 a Daily 21",
}
