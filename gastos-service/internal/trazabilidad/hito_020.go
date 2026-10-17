package trazabilidad

// Hito020 conserva la relación entre el cambio 20 y la fuente entregada.
var Hito020 = Cambio{
	Numero:      20,
	Fecha:       "17/10/2026",
	Descripcion: "Reportes estadísticos con dos PDF, filtro de fechas y Chart.js",
	Motivo:      "Entregar dashboard gerencial y descargas",
	Responsable: "No indicado",
	Rutas:       []string{"gastos-service/internal/reportes", "frontend/src/app/features/reportes"},
	Fuente:      "Especificación 2026-10-17",
}
