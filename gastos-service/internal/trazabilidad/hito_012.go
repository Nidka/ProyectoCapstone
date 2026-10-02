package trazabilidad

// Hito012 conserva la relación entre el cambio 12 y la fuente entregada.
var Hito012 = Cambio{
	Numero:      12,
	Fecha:       "02/10/2026",
	Descripcion: "Cierre del módulo de comprobantes e inicio del modelo SQL de aprobación",
	Motivo:      "Completar comprobantes e iniciar HU-07",
	Responsable: "Valera Lizarraga, Francisco Kadin / Mantilla Arqueros, Frank Piero",
	Rutas:       []string{"gastos-service/internal/handlers", "gastos-service/migrations"},
	Fuente:      "Daily 9, Acta 002",
}
