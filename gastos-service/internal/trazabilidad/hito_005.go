package trazabilidad

// Hito005 conserva la relación entre el cambio 5 y la fuente entregada.
var Hito005 = Cambio{
	Numero:      5,
	Fecha:       "24/09/2026",
	Descripcion: "Configuración del almacenamiento de archivos",
	Motivo:      "Preparar la carga de comprobantes mediante UPLOADS_DIR",
	Responsable: "Valera Lizarraga, Francisco Kadin",
	Rutas:       []string{"gastos-service/internal/config", "gastos-service/uploads"},
	Fuente:      "Daily 2",
}
