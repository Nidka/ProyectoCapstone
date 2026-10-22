package trazabilidad

// Hito030 conserva la relación entre el cambio 30 y la fuente entregada.
var Hito030 = Cambio{
	Numero:      30,
	Fecha:       "22/10/2026",
	Descripcion: "Log de errores, comprobantes en solo lectura, respaldo y espera OCR",
	Motivo:      "Cumplir RNF-14, RNF-5 y RNF-4 y evitar fallos en lecturas lentas",
	Responsable: "No indicado",
	Rutas:       []string{"gastos-service/scripts/respaldo.sh", "gastos-service/internal/config"},
	Fuente:      "RESULTADO bloque 5",
}
