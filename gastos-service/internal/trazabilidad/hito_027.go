package trazabilidad

// Hito027 conserva la relación entre el cambio 27 y la fuente entregada.
var Hito027 = Cambio{
	Numero:      27,
	Fecha:       "22/10/2026",
	Descripcion: "Gestión de usuarios y cambio obligatorio de clave",
	Motivo:      "Cumplir REQ-28 a REQ-32",
	Responsable: "Mantilla Arqueros, Frank Piero",
	Rutas:       []string{"gastos-service/internal/auth", "gastos-service/migrations/0007_usuario_gestion.sql", "frontend/src/app/features/usuarios"},
	Fuente:      "RESULTADO bloque 2, Acta 007",
}
