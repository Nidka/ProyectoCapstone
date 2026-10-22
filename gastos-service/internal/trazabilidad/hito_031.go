package trazabilidad

// Hito031 conserva la relación entre el cambio 31 y la fuente entregada.
var Hito031 = Cambio{
	Numero:      31,
	Fecha:       "22/10/2026",
	Descripcion: "Corrección de seis hallazgos de la revisión final",
	Motivo:      "Corregir seguridad JWT, uploads, datos demo, logs, campana y desfase horario",
	Responsable: "No indicado",
	Rutas:       []string{"gastos-service/internal/config", "gastos-service/internal/handlers", "frontend/src/styles", "gastos-service/demo/reportes_demo_limpiar.sql"},
	Fuente:      "RESULTADO sección 8",
}
