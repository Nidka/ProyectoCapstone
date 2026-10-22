package reportes

import "time"

// Resumen es lo que muestran el dashboard y los PDF para un rango de fechas.
type Resumen struct {
	Desde        string        `json:"desde"`
	Hasta        string        `json:"hasta"`
	Indicadores  Indicadores   `json:"indicadores"`
	PorMes       []Mes         `json:"por_mes"`
	PorEstado    []PorEstado   `json:"por_estado"`
	PorCliente   []FilaRanking `json:"por_cliente"`
	PorUnidad    []FilaRanking `json:"por_unidad"`
	PorConductor []FilaRanking `json:"por_conductor"`
	// Detalle solo se carga para el PDF de detalle.
	Detalle []FilaDetalle `json:"-"`
}

// Indicadores son las cifras principales del período.
type Indicadores struct {
	Total                float64 `json:"total"`
	Cantidad             int     `json:"cantidad"`
	Promedio             float64 `json:"promedio"`
	Sustentado           float64 `json:"sustentado"`
	Pendiente            float64 `json:"pendiente"`
	PorcentajeSustentado float64 `json:"porcentaje_sustentado"`
	// GastosSustentados e ISC (índice de sustentación de costos, HU-15) cuentan gastos, no montos:
	// ISC = gastos sustentados ÷ gastos totales × 100.
	GastosSustentados int     `json:"gastos_sustentados"`
	ISC               float64 `json:"isc"`
}

// Mes es el gasto de un mes (AAAA-MM) separado por estado.
type Mes struct {
	Mes        string  `json:"mes"`
	Registrado float64 `json:"registrado"`
	Pagado     float64 `json:"pagado"`
	Sustentado float64 `json:"sustentado"`
	Total      float64 `json:"total"`
}

// PorEstado es la participación de un estado en el período.
type PorEstado struct {
	Estado     string  `json:"estado"`
	Cantidad   int     `json:"cantidad"`
	Monto      float64 `json:"monto"`
	Porcentaje float64 `json:"porcentaje"`
}

// FilaRanking es un puesto de un ranking (cliente, unidad o conductor).
type FilaRanking struct {
	Puesto     int     `json:"puesto"`
	Etiqueta   string  `json:"etiqueta"`
	Cantidad   int     `json:"cantidad"`
	Monto      float64 `json:"monto"`
	Porcentaje float64 `json:"porcentaje"`
}

// FilaDetalle es un gasto del período con los datos de su viaje.
type FilaDetalle struct {
	ID        int
	Fecha     time.Time
	Cliente   string
	ViajeID   int
	Unidad    string
	Conductor string
	Estado    string
	Monto     float64
}
