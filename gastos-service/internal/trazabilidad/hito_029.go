package trazabilidad

// Hito029 conserva la relación entre el cambio 29 y la fuente entregada.
var Hito029 = Cambio{
	Numero:      29,
	Fecha:       "22/10/2026",
	Descripcion: "Notificaciones horarias, fecha de cobro y campana de avisos",
	Motivo:      "Cumplir REQ-20, REQ-21 y RNF-8",
	Responsable: "Valera Lizarraga, Francisco Kadin",
	Rutas:       []string{"gastos-service/internal/notificaciones", "gastos-service/migrations/0009_notificaciones.sql", "frontend/src/app/shell"},
	Fuente:      "RESULTADO bloque 4",
}
