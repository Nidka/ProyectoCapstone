package trazabilidad

// Hito026 conserva la relación entre el cambio 26 y la fuente entregada.
var Hito026 = Cambio{
	Numero:      26,
	Fecha:       "22/10/2026",
	Descripcion: "Login JWT, roles admin y logística y eliminación de X-User-Role",
	Motivo:      "Evitar confiar en una cabecera controlada por el frontend",
	Responsable: "Mantilla Arqueros, Frank Piero",
	Rutas:       []string{"gastos-service/internal/auth", "gastos-service/migrations/0006_roles.sql", "frontend"},
	Fuente:      "RESULTADO_PENDIENTES bloque 1",
}
