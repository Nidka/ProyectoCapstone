package trazabilidad

// Hito015 conserva la relación entre el cambio 15 y la fuente entregada.
var Hito015 = Cambio{
	Numero:      15,
	Fecha:       "10/10/2026",
	Descripcion: "Checklist documental de cinco criterios y bloqueo de aprobación incompleta",
	Motivo:      "Impedir gastos Sustentados sin checklist completo",
	Responsable: "Valera Lizarraga, Francisco Kadin / Mantilla Arqueros, Frank Piero / Rodriguez Cardozo, Evelyn Victoria",
	Rutas:       []string{"gastos-service/internal/checklist", "gastos-service/migrations", "frontend"},
	Fuente:      "Especificación 2026-10-10, Daily 10, Acta 004",
}
