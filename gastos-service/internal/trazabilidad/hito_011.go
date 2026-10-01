package trazabilidad

// Hito011 conserva la relación entre el cambio 11 y la fuente entregada.
var Hito011 = Cambio{
	Numero:      11,
	Fecha:       "01/10/2026",
	Descripcion: "Despliegue de la API de comprobantes y validación frontend de rechazos",
	Motivo:      "Cerrar el módulo 3.1.2",
	Responsable: "Valera Lizarraga, Francisco Kadin / Mantilla Arqueros, Frank Piero",
	Rutas:       []string{"gastos-service", "frontend"},
	Fuente:      "Daily 8",
}
