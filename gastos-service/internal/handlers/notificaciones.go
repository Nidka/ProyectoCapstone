package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"gastos-service/internal/bitacora"

	"github.com/jackc/pgx/v5"
)

// limiteNotificaciones acota la lista; no_leidas siempre cuenta todas las pendientes.
const limiteNotificaciones = 100

// notificacionOut es un aviso tal como lo ve el Encargado de Logística (REQ-20, REQ-21).
type notificacionOut struct {
	ID       int        `json:"id"`
	Tipo     string     `json:"tipo"`
	IDGasto  int        `json:"id_gasto"`
	Mensaje  string     `json:"mensaje"`
	CreadaEn time.Time  `json:"creada_en"`
	Leida    bool       `json:"leida"`
	LeidaEn  *time.Time `json:"leida_en"`
}

const columnasNotificacion = `id, tipo, id_gasto, mensaje, creada_en, leida, leida_en`

func (n *notificacionOut) escanear(fila pgx.Row) error {
	return fila.Scan(&n.ID, &n.Tipo, &n.IDGasto, &n.Mensaje, &n.CreadaEn, &n.Leida, &n.LeidaEn)
}

// listNotificaciones devuelve los avisos con los no leídos primero (los más nuevos arriba) y el
// total de no leídos, para el número de la campana. Solo lo ve el Encargado de Logística.
func (h *Handler) listNotificaciones(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireLogistica(r); !ok {
		writeError(w, http.StatusForbidden, "solo el Encargado de Logística puede ver las notificaciones")
		return
	}
	ctx := r.Context()
	filas, err := h.DB.Query(ctx, `SELECT `+columnasNotificacion+` FROM notificacion ORDER BY leida, creada_en DESC, id DESC LIMIT $1`, limiteNotificaciones)
	if err != nil {
		writeFallo(w, r, err, "no se pudo listar las notificaciones")
		return
	}
	defer filas.Close()
	avisos := []notificacionOut{}
	for filas.Next() {
		var n notificacionOut
		if err := n.escanear(filas); err != nil {
			writeFallo(w, r, err, "error leyendo las notificaciones")
			return
		}
		avisos = append(avisos, n)
	}
	if err := filas.Err(); err != nil {
		writeFallo(w, r, err, "error leyendo las notificaciones")
		return
	}
	filas.Close()

	var noLeidas int
	if err := h.DB.QueryRow(ctx, `SELECT count(*) FROM notificacion WHERE NOT leida`).Scan(&noLeidas); err != nil {
		writeFallo(w, r, err, "no se pudo contar las notificaciones")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"no_leidas": noLeidas, "notificaciones": avisos})
}

// marcarNotificacionLeida marca un aviso como leído. Repetirlo no cambia nada (ni la fecha de
// lectura ni la bitácora); solo la primera vez deja su entrada, dentro de la misma transacción.
func (h *Handler) marcarNotificacionLeida(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireLogistica(r); !ok {
		writeError(w, http.StatusForbidden, "solo el Encargado de Logística puede marcar notificaciones")
		return
	}
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "id de notificación inválido")
		return
	}
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		writeFallo(w, r, err, "no se pudo iniciar la operación")
		return
	}
	defer tx.Rollback(ctx)

	var n notificacionOut
	if err := n.escanear(tx.QueryRow(ctx, `SELECT `+columnasNotificacion+` FROM notificacion WHERE id = $1 FOR UPDATE`, id)); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "notificación no encontrada")
			return
		}
		writeFallo(w, r, err, "no se pudo leer la notificación")
		return
	}
	if !n.Leida {
		if err := n.escanear(tx.QueryRow(ctx, `UPDATE notificacion SET leida = TRUE, leida_en = now() WHERE id = $1 RETURNING `+columnasNotificacion, id)); err != nil {
			writeFallo(w, r, err, "no se pudo marcar la notificación")
			return
		}
		err := bitacora.Registrar(ctx, tx, bitacora.Entrada{
			Actor: actorDe(r), Accion: "notificacion.leer", Entidad: "notificacion", IDEntidad: int64(id),
			Detalle: map[string]any{
				"id_gasto": n.IDGasto, "tipo": n.Tipo,
				"antes": map[string]any{"leida": false}, "despues": map[string]any{"leida": true},
			},
		})
		if err != nil {
			writeFallo(w, r, err, "no se pudo registrar el cambio")
			return
		}
	}
	if err := tx.Commit(ctx); err != nil {
		writeFallo(w, r, err, "no se pudo confirmar la operación")
		return
	}
	writeJSON(w, http.StatusOK, n)
}
