package trazabilidad

// Hito002 conserva la relación entre el cambio 2 y la fuente entregada.
var Hito002 = Cambio{
	Numero:      2,
	Fecha:       "23/09/2026",
	Descripcion: "Script SQL y base PostgreSQL 17 local desplegada (HU-02)",
	Motivo:      "Dejar la base lista para validar viaje y cliente",
	Responsable: "Mantilla Arqueros, Frank Piero",
	Rutas:       []string{"db.sql", "gastos-service/migrations"},
	Fuente:      "Daily 1",
}
