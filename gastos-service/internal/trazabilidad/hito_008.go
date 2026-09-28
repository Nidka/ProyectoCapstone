package trazabilidad

// Hito008 conserva la relación entre el cambio 8 y la fuente entregada.
var Hito008 = Cambio{
	Numero:      8,
	Fecha:       "28/09/2026",
	Descripcion: "QA del almacenamiento de adjuntos y pruebas unitarias de HU-05",
	Motivo:      "Validar los flujos antes del despliegue",
	Responsable: "Mantilla Arqueros, Frank Piero / Rodriguez Cardozo, Evelyn Victoria",
	Rutas:       []string{"gastos-service/internal/handlers", "frontend/src/app/features/gastos"},
	Fuente:      "Daily 5",
}
