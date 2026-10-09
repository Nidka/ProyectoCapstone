package trazabilidad

// Hito014 conserva la relación entre el cambio 14 y la fuente entregada.
var Hito014 = Cambio{
	Numero:      14,
	Fecha:       "03/10/2026 al 09/10/2026",
	Descripcion: "Módulo de aprobación completo (3.2.1)",
	Motivo:      "Completar el hito documentado en el Acta 003",
	Responsable: "Rodriguez Cardozo, Evelyn Victoria / Valera Lizarraga, Francisco Kadin / Mantilla Arqueros, Frank Piero",
	Rutas:       []string{"gastos-service/internal/handlers", "frontend/src/app/features/gastos"},
	Fuente:      "Cronograma, Acta 003",
}
