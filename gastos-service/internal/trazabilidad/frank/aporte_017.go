package frank

// Aporte017 identifica el trabajo documentado en 22/10/2026.
var Aporte017 = Aporte{
	Numero:      17,
	Fecha:       "22/10/2026",
	Descripcion: "Implementar gestión de usuarios y cambio obligatorio de clave",
	Rutas:       []string{"gastos-service/internal/auth", "gastos-service/migrations/0007_usuario_gestion.sql", "frontend/src/app/features/usuarios"},
	Fuente:      "RESULTADO bloque 2, Acta 007",
}
