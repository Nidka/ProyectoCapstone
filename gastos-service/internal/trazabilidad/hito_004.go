package trazabilidad

// Hito004 conserva la relación entre el cambio 4 y la fuente entregada.
var Hito004 = Cambio{
	Numero:      4,
	Fecha:       "24/09/2026",
	Descripcion: "Relaciones SQL para asignar cliente (HU-03)",
	Motivo:      "Vincular cada gasto con su cliente",
	Responsable: "Mantilla Arqueros, Frank Piero",
	Rutas:       []string{"db.sql", "gastos-service/migrations"},
	Fuente:      "Daily 2",
}
