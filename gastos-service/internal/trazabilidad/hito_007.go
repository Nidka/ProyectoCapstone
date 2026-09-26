package trazabilidad

// Hito007 conserva la relación entre el cambio 7 y la fuente entregada.
var Hito007 = Cambio{
	Numero:      7,
	Fecha:       "26/09/2026",
	Descripcion: "Cambio de estado del gasto en Angular y cierre del endpoint de carga (HU-05)",
	Motivo:      "Gestionar los estados Registrado, Pagado y Sustentado",
	Responsable: "Rodriguez Cardozo, Evelyn Victoria / Valera Lizarraga, Francisco Kadin",
	Rutas:       []string{"frontend/src/app/features/gastos", "gastos-service/internal/handlers"},
	Fuente:      "Daily 4",
}
