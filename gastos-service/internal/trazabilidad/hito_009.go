package trazabilidad

// Hito009 conserva la relación entre el cambio 9 y la fuente entregada.
var Hito009 = Cambio{
	Numero:      9,
	Fecha:       "29/09/2026",
	Descripcion: "Endpoint de aprobación, consultas SQL y boceto del modal de rechazo (HU-07, HU-08)",
	Motivo:      "Cumplir REQ-10 a REQ-14",
	Responsable: "Valera Lizarraga, Francisco Kadin / Mantilla Arqueros, Frank Piero",
	Rutas:       []string{"gastos-service/internal/handlers", "frontend/src/app/features/gastos"},
	Fuente:      "Daily 6",
}
