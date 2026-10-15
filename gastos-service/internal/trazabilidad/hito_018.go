package trazabilidad

// Hito018 conserva la relación entre el cambio 18 y la fuente entregada.
var Hito018 = Cambio{
	Numero:      18,
	Fecha:       "15/10/2026",
	Descripcion: "Indicador ISC y filtros por fecha y estado (HU-12)",
	Motivo:      "Medir gastos sustentados frente al total",
	Responsable: "Valera Lizarraga, Francisco Kadin / Mantilla Arqueros, Frank Piero",
	Rutas:       []string{"gastos-service/internal/reportes", "gastos-service/internal/handlers", "frontend"},
	Fuente:      "Daily 15",
}
