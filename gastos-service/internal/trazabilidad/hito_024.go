package trazabilidad

// Hito024 conserva la relación entre el cambio 24 y la fuente entregada.
var Hito024 = Cambio{
	Numero:      24,
	Fecha:       "22/10/2026",
	Descripcion: "Migración de Python y FastAPI a un único servicio Go",
	Motivo:      "Centralizar 118 operaciones y el login en un solo servicio",
	Responsable: "No indicado",
	Rutas:       []string{"gastos-service/internal/crud", "gastos-service/internal/auth", "gastos-service/Dockerfile", "docker-compose.demo.yml"},
	Fuente:      "Especificación 2026-10-22, README",
}
