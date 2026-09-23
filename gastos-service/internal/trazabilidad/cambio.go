package trazabilidad

// Cambio relaciona una modificación del producto con su evidencia documental.
type Cambio struct {
	Numero      int
	Fecha       string
	Descripcion string
	Motivo      string
	Responsable string
	Rutas       []string
	Fuente      string
}
