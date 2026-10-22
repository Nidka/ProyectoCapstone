package trazabilidad

// Hito028 conserva la relación entre el cambio 28 y la fuente entregada.
var Hito028 = Cambio{
	Numero:      28,
	Fecha:       "22/10/2026",
	Descripcion: "Bitácora inmutable e historial por gasto",
	Motivo:      "Cumplir REQ-33 a REQ-36 y RNF-9 a RNF-12",
	Responsable: "Valera Lizarraga, Francisco Kadin",
	Rutas:       []string{"gastos-service/internal/bitacora", "gastos-service/migrations/0008_bitacora.sql"},
	Fuente:      "RESULTADO bloque 3, Acta 008",
}
