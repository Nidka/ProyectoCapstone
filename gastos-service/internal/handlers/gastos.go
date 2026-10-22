package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"gastos-service/internal/models"
)

type createGastoRequest struct {
	ViajeID   int     `json:"viaje_id"`
	ClienteID int     `json:"cliente_id"`
	Monto     float64 `json:"monto"`
	Fecha     string  `json:"fecha"` // YYYY-MM-DD, opcional (hoy por defecto)
	CreadoPor string  `json:"creado_por"`
}

// createGasto implementa RF-1 (registrar gasto asociado a un viaje),
// RF-2 (rechazar si el viaje no existe) y RF-3 (cliente responsable de pago).
func (h *Handler) createGasto(w http.ResponseWriter, r *http.Request) {
	var req createGastoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}

	if req.ViajeID <= 0 {
		writeError(w, http.StatusBadRequest, "viaje_id es obligatorio")
		return
	}
	if req.ClienteID <= 0 {
		writeError(w, http.StatusBadRequest, "cliente_id (cliente responsable de pago) es obligatorio")
		return
	}
	if req.Monto <= 0 {
		writeError(w, http.StatusBadRequest, "monto debe ser mayor a 0")
		return
	}

	fecha := time.Now()
	if req.Fecha != "" {
		parsed, err := time.Parse("2006-01-02", req.Fecha)
		if err != nil {
			writeError(w, http.StatusBadRequest, "fecha inválida, use el formato YYYY-MM-DD")
			return
		}
		fecha = parsed
	}

	if req.CreadoPor == "" {
		req.CreadoPor = headerOrDefault(r, "X-User-Name", "Operador Demo")
	}

	ctx := context.Background()

	// RF-2: el viaje debe existir antes de guardar el gasto.
	var viajeExiste bool
	if err := h.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM programacionviaje WHERE id_viaje = $1)`, req.ViajeID).Scan(&viajeExiste); err != nil {
		writeError(w, http.StatusInternalServerError, "error validando el viaje")
		return
	}
	if !viajeExiste {
		writeError(w, http.StatusUnprocessableEntity, "el viaje indicado no existe")
		return
	}

	var clienteExiste bool
	if err := h.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM cliente WHERE id_cliente = $1)`, req.ClienteID).Scan(&clienteExiste); err != nil {
		writeError(w, http.StatusInternalServerError, "error validando el cliente")
		return
	}
	if !clienteExiste {
		writeError(w, http.StatusUnprocessableEntity, "el cliente indicado no existe")
		return
	}

	var g models.Gasto
	err := h.DB.QueryRow(ctx, `
		INSERT INTO gasto (viaje_id, cliente_id, monto, fecha, estado, creado_por, creado_en)
		VALUES ($1, $2, $3, $4, 'Registrado', $5, now())
		RETURNING id, viaje_id, cliente_id, monto, fecha, estado, creado_por, creado_en
	`, req.ViajeID, req.ClienteID, req.Monto, fecha, req.CreadoPor).Scan(
		&g.ID, &g.ViajeID, &g.ClienteID, &g.Monto, &g.Fecha, &g.Estado, &g.CreadoPor, &g.CreadoEn,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo registrar el gasto")
		return
	}

	writeJSON(w, http.StatusCreated, g)
}

// listGastos es un soporte mínimo para que las pantallas de comprobantes y
// aprobación puedan elegir un gasto; la consulta de gastos completa (con
// filtros y búsqueda) es una iteración posterior.
func (h *Handler) listGastos(w http.ResponseWriter, r *http.Request) {
	filtro, mensaje := leerFiltroGastos(r)
	if mensaje != "" {
		writeError(w, http.StatusBadRequest, mensaje)
		return
	}
	cond, args := filtro.condiciones(0)
	consulta := `SELECT g.id, g.viaje_id, g.cliente_id, g.monto, g.fecha, g.estado, g.creado_por, g.creado_en FROM gasto g`
	if len(cond) > 0 {
		consulta += " WHERE " + strings.Join(cond, " AND ")
	}
	consulta += " ORDER BY g.creado_en DESC, g.id DESC LIMIT 200"
	rows, err := h.DB.Query(context.Background(), consulta, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo listar los gastos")
		return
	}
	defer rows.Close()

	out := []models.Gasto{}
	for rows.Next() {
		var g models.Gasto
		if err := rows.Scan(&g.ID, &g.ViajeID, &g.ClienteID, &g.Monto, &g.Fecha, &g.Estado, &g.CreadoPor, &g.CreadoEn); err != nil {
			writeError(w, http.StatusInternalServerError, "error leyendo los gastos")
			return
		}
		out = append(out, g)
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) getGasto(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var g models.Gasto
	err := h.DB.QueryRow(context.Background(), `
		SELECT id, viaje_id, cliente_id, monto, fecha, estado, creado_por, creado_en
		FROM gasto WHERE id = $1
	`, id).Scan(&g.ID, &g.ViajeID, &g.ClienteID, &g.Monto, &g.Fecha, &g.Estado, &g.CreadoPor, &g.CreadoEn)
	if err != nil {
		writeError(w, http.StatusNotFound, "gasto no encontrado")
		return
	}
	writeJSON(w, http.StatusOK, g)
}
