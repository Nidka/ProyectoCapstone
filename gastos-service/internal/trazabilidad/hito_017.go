package trazabilidad

// Hito017 conserva la relación entre el cambio 17 y la fuente entregada.
var Hito017 = Cambio{
	Numero:      17,
	Fecha:       "14/10/2026",
	Descripcion: "Índice compuesto por fecha y estado y consulta de reportes optimizada",
	Motivo:      "Reducir el tiempo de 420 ms a 180 ms según INC-01",
	Responsable: "Valera Lizarraga, Francisco Kadin; medición de Rodriguez Cardozo, Evelyn Victoria",
	Rutas:       []string{"gastos-service/migrations", "gastos-service/internal/reportes"},
	Fuente:      "Daily 13, Daily 14, INC-01",
}
