package trazabilidad

// Hito019 conserva la relación entre el cambio 19 y la fuente entregada.
var Hito019 = Cambio{
	Numero:      19,
	Fecha:       "16/10/2026",
	Descripcion: "Despliegue de reportes en pruebas",
	Motivo:      "Cerrar el módulo 3.3.1",
	Responsable: "Valera Lizarraga, Francisco Kadin / Mantilla Arqueros, Frank Piero",
	Rutas:       []string{"gastos-service", "frontend"},
	Fuente:      "Daily 16, Acta 005",
}
