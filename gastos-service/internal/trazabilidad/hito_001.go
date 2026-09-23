package trazabilidad

// Hito001 conserva la relación entre el cambio 1 y la fuente entregada.
var Hito001 = Cambio{
	Numero:      1,
	Fecha:       "23/09/2026",
	Descripcion: "Endpoint REST en Go para registrar un gasto (HU-01)",
	Motivo:      "Primer módulo del plan 3.1.1; REQ-1 a REQ-5",
	Responsable: "Valera Lizarraga, Francisco Kadin",
	Rutas:       []string{"gastos-service/internal/handlers", "gastos-service/internal/registro"},
	Fuente:      "Daily 1",
}
