package models

import "time"

// Gasto representa un gasto adicional de viaje.
// Flujo de estado: Registrado -> Pagado (RF-5) -> Sustentado (RF-7) | -> Registrado (RF-9)
type Gasto struct {
	ID        int       `json:"id"`
	ViajeID   int       `json:"viaje_id"`
	ClienteID int       `json:"cliente_id"`
	Monto     float64   `json:"monto"`
	Fecha     time.Time `json:"fecha"`
	Estado    string    `json:"estado"`
	CreadoPor string    `json:"creado_por"`
	CreadoEn  time.Time `json:"creado_en"`
}

// Comprobante representa un comprobante de pago adjuntado a un gasto.
type Comprobante struct {
	ID            int             `json:"id"`
	GastoID       int             `json:"gasto_id"`
	ArchivoURL    string          `json:"archivo_url"`
	TipoArchivo   string          `json:"tipo_archivo"`
	Estado        string          `json:"estado"`
	MotivoRechazo *string         `json:"motivo_rechazo,omitempty"`
	AprobadoPor   *string         `json:"aprobado_por,omitempty"`
	AprobadoEn    *time.Time      `json:"aprobado_en,omitempty"`
	CreadoEn      time.Time       `json:"creado_en"`
	Checklist     []ItemChecklist `json:"checklist"`
}

// ItemChecklist es un criterio del checklist documental con su resultado.
type ItemChecklist struct {
	Criterio string `json:"criterio"`
	Cumple   bool   `json:"cumple"`
}
