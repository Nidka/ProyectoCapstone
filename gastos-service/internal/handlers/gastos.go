package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gastos-service/internal/bitacora"
	"gastos-service/internal/models"

	"github.com/jackc/pgx/v5"
)

type createGastoRequest struct {
	ViajeID   int     `json:"viaje_id"`
	ClienteID int     `json:"cliente_id"`
	Monto     float64 `json:"monto"`
	Fecha     string  `json:"fecha"` // YYYY-MM-DD, opcional (hoy por defecto)
	// FechaCobro es la fecha esperada de cobro al cliente (YYYY-MM-DD), opcional (REQ-21).
	FechaCobro string `json:"fecha_cobro"`
}

// createGasto implementa RF-1 (registrar gasto asociado a un viaje),
// RF-2 (rechazar si el viaje no existe) y RF-3 (cliente responsable de pago).
func (h *Handler) createGasto(w http.ResponseWriter, r *http.Request) {
	nombre, autorizado := requireLogistica(r)
	if !autorizado {
		writeError(w, http.StatusForbidden, "solo el Encargado de Logística puede registrar gastos")
		return
	}

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
	var fechaCobro *time.Time
	if req.FechaCobro != "" {
		parsed, err := time.Parse("2006-01-02", req.FechaCobro)
		if err != nil {
			writeError(w, http.StatusBadRequest, "fecha de cobro inválida, use el formato YYYY-MM-DD")
			return
		}
		anio, mes, dia := fecha.Date()
		if parsed.Before(time.Date(anio, mes, dia, 0, 0, 0, 0, time.UTC)) {
			writeError(w, http.StatusBadRequest, "la fecha de cobro no puede ser anterior a la fecha del gasto")
			return
		}
		fechaCobro = &parsed
	}

	ctx := context.Background()

	// RF-2: el viaje debe existir antes de guardar el gasto.
	var viajeExiste bool
	if err := h.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM programacionviaje WHERE id_viaje = $1)`, req.ViajeID).Scan(&viajeExiste); err != nil {
		writeFallo(w, r, err, "error validando el viaje")
		return
	}
	if !viajeExiste {
		writeError(w, http.StatusUnprocessableEntity, "el viaje indicado no existe")
		return
	}

	var clienteExiste bool
	if err := h.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM cliente WHERE id_cliente = $1)`, req.ClienteID).Scan(&clienteExiste); err != nil {
		writeFallo(w, r, err, "error validando el cliente")
		return
	}
	if !clienteExiste {
		writeError(w, http.StatusUnprocessableEntity, "el cliente indicado no existe")
		return
	}

	// El gasto y su entrada de bitácora se guardan juntos (REQ-33): si falla uno, no queda ninguno.
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		writeFallo(w, r, err, "no se pudo iniciar la operación")
		return
	}
	defer tx.Rollback(ctx)

	var g models.Gasto
	err = tx.QueryRow(ctx, `
		INSERT INTO gasto (viaje_id, cliente_id, monto, fecha, estado, creado_por, creado_en, fecha_cobro)
		VALUES ($1, $2, $3, $4, 'Registrado', $5, now(), $6)
		RETURNING id, viaje_id, cliente_id, monto, fecha, estado, creado_por, `+instante("creado_en")+`, fecha_cobro
	`, req.ViajeID, req.ClienteID, req.Monto, fecha, nombre, fechaCobro).Scan(
		&g.ID, &g.ViajeID, &g.ClienteID, &g.Monto, &g.Fecha, &g.Estado, &g.CreadoPor, &g.CreadoEn, &g.FechaCobro,
	)
	if err != nil {
		writeFallo(w, r, err, "no se pudo registrar el gasto")
		return
	}
	despues := map[string]any{
		"viaje_id": g.ViajeID, "cliente_id": g.ClienteID, "monto": g.Monto,
		"fecha": g.Fecha.Format("2006-01-02"), "estado": g.Estado, "creado_por": g.CreadoPor,
	}
	if g.FechaCobro != nil {
		despues["fecha_cobro"] = g.FechaCobro.Format("2006-01-02")
	}
	err = bitacora.Registrar(ctx, tx, bitacora.Entrada{
		Actor: actorDe(r), Accion: "gasto.registrar", Entidad: "gasto", IDEntidad: int64(g.ID),
		Detalle: map[string]any{"despues": despues},
	})
	if err != nil {
		writeFallo(w, r, err, "no se pudo registrar el gasto")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		writeFallo(w, r, err, "no se pudo confirmar la operación")
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
	consulta := `SELECT g.id, g.viaje_id, g.cliente_id, g.monto, g.fecha, g.estado, g.creado_por, ` + instante("g.creado_en") + `, g.fecha_cobro FROM gasto g`
	if len(cond) > 0 {
		consulta += " WHERE " + strings.Join(cond, " AND ")
	}
	consulta += " ORDER BY g.creado_en DESC, g.id DESC LIMIT 200"
	rows, err := h.DB.Query(context.Background(), consulta, args...)
	if err != nil {
		writeFallo(w, r, err, "no se pudo listar los gastos")
		return
	}
	defer rows.Close()

	out := []models.Gasto{}
	for rows.Next() {
		var g models.Gasto
		if err := rows.Scan(&g.ID, &g.ViajeID, &g.ClienteID, &g.Monto, &g.Fecha, &g.Estado, &g.CreadoPor, &g.CreadoEn, &g.FechaCobro); err != nil {
			writeFallo(w, r, err, "error leyendo los gastos")
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
		SELECT id, viaje_id, cliente_id, monto, fecha, estado, creado_por, `+instante("creado_en")+`, fecha_cobro
		FROM gasto WHERE id = $1
	`, id).Scan(&g.ID, &g.ViajeID, &g.ClienteID, &g.Monto, &g.Fecha, &g.Estado, &g.CreadoPor, &g.CreadoEn, &g.FechaCobro)
	if err != nil {
		writeError(w, http.StatusNotFound, "gasto no encontrado")
		return
	}
	writeJSON(w, http.StatusOK, g)
}

type corregirGastoRequest struct {
	// Monto y Fecha son opcionales: solo se cambia lo que viene. Al menos uno es obligatorio.
	Monto  *float64 `json:"monto"`
	Fecha  string   `json:"fecha"` // YYYY-MM-DD
	Motivo string   `json:"motivo"`
}

const maxMotivoCorreccion = 200

// corregirGasto cambia el monto y/o la fecha de un gasto que todavía no está sustentado, por
// ejemplo cuando el comprobante dice otra cosa que lo digitado. Exige un motivo y deja en la
// bitácora el valor anterior, el nuevo y quién lo hizo, dentro de la misma transacción. Un gasto
// sustentado ya tiene su comprobante aprobado y queda cerrado.
func (h *Handler) corregirGasto(w http.ResponseWriter, r *http.Request) {
	if _, autorizado := requireLogistica(r); !autorizado {
		writeError(w, http.StatusForbidden, "solo el Encargado de Logística puede corregir gastos")
		return
	}
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "id de gasto inválido")
		return
	}
	var req corregirGastoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	motivo := strings.TrimSpace(req.Motivo)
	if motivo == "" {
		writeError(w, http.StatusBadRequest, "debes indicar el motivo de la corrección")
		return
	}
	if len([]rune(motivo)) > maxMotivoCorreccion {
		writeError(w, http.StatusBadRequest, "el motivo no puede pasar de 200 caracteres")
		return
	}
	if req.Monto == nil && req.Fecha == "" {
		writeError(w, http.StatusBadRequest, "indica el monto o la fecha que se corrige")
		return
	}
	if req.Monto != nil && *req.Monto <= 0 {
		writeError(w, http.StatusBadRequest, "monto debe ser mayor a 0")
		return
	}
	var fecha *time.Time
	if req.Fecha != "" {
		parsed, err := time.Parse("2006-01-02", req.Fecha)
		if err != nil {
			writeError(w, http.StatusBadRequest, "fecha inválida, use el formato YYYY-MM-DD")
			return
		}
		fecha = &parsed
	}

	ctx := context.Background()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		writeFallo(w, r, err, "no se pudo iniciar la operación")
		return
	}
	defer tx.Rollback(ctx)

	// El gasto se bloquea mientras se corrige: dos correcciones a la vez no se pisan.
	var antes models.Gasto
	err = tx.QueryRow(ctx, `
		SELECT id, viaje_id, cliente_id, monto, fecha, estado, creado_por, `+instante("creado_en")+`, fecha_cobro
		FROM gasto WHERE id = $1 FOR UPDATE
	`, id).Scan(&antes.ID, &antes.ViajeID, &antes.ClienteID, &antes.Monto, &antes.Fecha, &antes.Estado, &antes.CreadoPor, &antes.CreadoEn, &antes.FechaCobro)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "gasto no encontrado")
		return
	}
	if err != nil {
		writeFallo(w, r, err, "no se pudo leer el gasto")
		return
	}
	if antes.Estado == "Sustentado" {
		writeError(w, http.StatusConflict, "el gasto ya está sustentado y no se puede corregir")
		return
	}

	nuevoMonto, nuevaFecha := antes.Monto, antes.Fecha
	if req.Monto != nil {
		nuevoMonto = *req.Monto
	}
	if fecha != nil {
		nuevaFecha = *fecha
	}
	if nuevoMonto == antes.Monto && nuevaFecha.Format("2006-01-02") == antes.Fecha.Format("2006-01-02") {
		writeError(w, http.StatusBadRequest, "no hay cambios: el monto y la fecha ya son esos")
		return
	}
	if antes.FechaCobro != nil && antes.FechaCobro.Before(time.Date(nuevaFecha.Year(), nuevaFecha.Month(), nuevaFecha.Day(), 0, 0, 0, 0, time.UTC)) {
		writeError(w, http.StatusUnprocessableEntity, "la fecha del gasto no puede ser posterior a la fecha de cobro")
		return
	}

	var g models.Gasto
	err = tx.QueryRow(ctx, `
		UPDATE gasto SET monto = $2, fecha = $3 WHERE id = $1
		RETURNING id, viaje_id, cliente_id, monto, fecha, estado, creado_por, `+instante("creado_en")+`, fecha_cobro
	`, id, nuevoMonto, nuevaFecha).Scan(&g.ID, &g.ViajeID, &g.ClienteID, &g.Monto, &g.Fecha, &g.Estado, &g.CreadoPor, &g.CreadoEn, &g.FechaCobro)
	if err != nil {
		writeFallo(w, r, err, "no se pudo corregir el gasto")
		return
	}
	err = bitacora.Registrar(ctx, tx, bitacora.Entrada{
		Actor: actorDe(r), Accion: "gasto.corregir", Entidad: "gasto", IDEntidad: int64(id),
		Detalle: map[string]any{
			"antes":   map[string]any{"monto": antes.Monto, "fecha": antes.Fecha.Format("2006-01-02")},
			"despues": map[string]any{"monto": g.Monto, "fecha": g.Fecha.Format("2006-01-02")},
			"motivo":  motivo,
		},
	})
	if err != nil {
		writeFallo(w, r, err, "no se pudo registrar la corrección")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		writeFallo(w, r, err, "no se pudo confirmar la operación")
		return
	}
	writeJSON(w, http.StatusOK, g)
}
