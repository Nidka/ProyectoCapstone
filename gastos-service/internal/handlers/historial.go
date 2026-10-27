package handlers

import (
	"net/http"
	"strconv"

	"gastos-service/internal/bitacora"
)

// historialGasto implementa REQ-36: lo que se hizo con un gasto y con sus comprobantes, de lo más
// antiguo a lo más reciente. Lo puede leer cualquier usuario autenticado.
func (h *Handler) historialGasto(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "id de gasto inválido")
		return
	}
	ctx := r.Context()

	var existe bool
	if err := h.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM gasto WHERE id = $1)`, id).Scan(&existe); err != nil {
		writeFallo(w, r, err, "error validando el gasto")
		return
	}
	if !existe {
		writeError(w, http.StatusNotFound, "gasto no encontrado")
		return
	}

	historial, err := bitacora.HistorialDeGasto(ctx, h.DB, id)
	if err != nil {
		writeFallo(w, r, err, "no se pudo leer el historial")
		return
	}
	writeJSON(w, http.StatusOK, historial)
}
