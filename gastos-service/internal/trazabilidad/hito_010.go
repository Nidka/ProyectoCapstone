package trazabilidad

// Hito010 conserva la relación entre el cambio 10 y la fuente entregada.
var Hito010 = Cambio{
	Numero:      10,
	Fecha:       "30/09/2026",
	Descripcion: "Modal Angular de rechazo y retorno del gasto a Registrado (HU-08, HU-09)",
	Motivo:      "Exigir motivo y devolver el gasto rechazado a Registrado",
	Responsable: "Rodriguez Cardozo, Evelyn Victoria / Valera Lizarraga, Francisco Kadin",
	Rutas:       []string{"frontend/src/app/features/gastos", "gastos-service/internal/handlers"},
	Fuente:      "Daily 7",
}
