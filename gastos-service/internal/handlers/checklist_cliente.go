package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"gastos-service/internal/checklist"

	"github.com/jackc/pgx/v5"
)

// consultador es lo que necesita leer criterios: lo cumplen el pool y una transacción.
type consultador interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type checklistCliente struct {
	ClienteID     int                  `json:"cliente_id"`
	Personalizado bool                 `json:"personalizado"`
	Criterios     []checklist.Criterio `json:"criterios"`
}

func leerCriterios(ctx context.Context, q consultador, clienteID any) ([]checklist.Criterio, error) {
	sql := `SELECT clave, texto FROM criterio_checklist WHERE cliente_id = $1 ORDER BY orden, id`
	if clienteID == nil {
		sql = `SELECT clave, texto FROM criterio_checklist WHERE cliente_id IS NULL AND $1::int IS NULL ORDER BY orden, id`
	}
	rows, err := q.Query(ctx, sql, clienteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []checklist.Criterio{}
	for rows.Next() {
		var c checklist.Criterio
		if err := rows.Scan(&c.Clave, &c.Texto); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// criteriosVigentes devuelve los criterios que se exigen a un cliente: los suyos si configuró
// alguno, o los estándar.
func criteriosVigentes(ctx context.Context, q consultador, clienteID int) (criterios []checklist.Criterio, personalizado bool, err error) {
	if criterios, err = leerCriterios(ctx, q, clienteID); err != nil || len(criterios) > 0 {
		return criterios, len(criterios) > 0, err
	}
	criterios, err = leerCriterios(ctx, q, nil)
	return criterios, false, err
}

func (h *Handler) clienteExiste(ctx context.Context, id int) (bool, error) {
	var existe bool
	err := h.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM cliente WHERE id_cliente = $1)`, id).Scan(&existe)
	return existe, err
}

func (h *Handler) clienteDeRuta(w http.ResponseWriter, r *http.Request) (int, bool) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "id de cliente inválido")
		return 0, false
	}
	existe, err := h.clienteExiste(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "error validando el cliente")
		return 0, false
	}
	if !existe {
		writeError(w, http.StatusNotFound, "el cliente indicado no existe")
		return 0, false
	}
	return id, true
}

func (h *Handler) responderChecklist(w http.ResponseWriter, ctx context.Context, clienteID int) {
	criterios, personalizado, err := criteriosVigentes(ctx, h.DB, clienteID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo leer el checklist")
		return
	}
	writeJSON(w, http.StatusOK, checklistCliente{ClienteID: clienteID, Personalizado: personalizado, Criterios: criterios})
}

// getChecklistCliente devuelve los criterios que se revisan en los comprobantes de un cliente.
func (h *Handler) getChecklistCliente(w http.ResponseWriter, r *http.Request) {
	id, ok := h.clienteDeRuta(w, r)
	if !ok {
		return
	}
	h.responderChecklist(w, r.Context(), id)
}

// putChecklistCliente reemplaza el checklist de un cliente (REQ-15): de 1 a 15 criterios; los que
// traen clave la conservan y los nuevos reciben una.
func (h *Handler) putChecklistCliente(w http.ResponseWriter, r *http.Request) {
	if _, autorizado := requireLogistica(r); !autorizado {
		writeError(w, http.StatusForbidden, "solo el Encargado de Logística puede configurar el checklist")
		return
	}
	id, ok := h.clienteDeRuta(w, r)
	if !ok {
		return
	}
	var req struct {
		Criterios []checklist.Entrada `json:"criterios"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	if err := checklist.ValidarEntradas(req.Criterios); err != nil {
		writeError(w, http.StatusUnprocessableEntity, checklist.Mensaje(err))
		return
	}
	criterios := checklist.AsignarClaves(req.Criterios)

	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo iniciar la operación")
		return
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM criterio_checklist WHERE cliente_id = $1`, id); err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo guardar el checklist")
		return
	}
	for orden, c := range criterios {
		if _, err := tx.Exec(ctx, `INSERT INTO criterio_checklist (cliente_id, clave, texto, orden) VALUES ($1, $2, $3, $4)`, id, c.Clave, c.Texto, orden+1); err != nil {
			writeError(w, http.StatusInternalServerError, "no se pudo guardar el checklist")
			return
		}
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo confirmar la operación")
		return
	}
	h.responderChecklist(w, ctx, id)
}

// deleteChecklistCliente devuelve al cliente al checklist estándar.
func (h *Handler) deleteChecklistCliente(w http.ResponseWriter, r *http.Request) {
	if _, autorizado := requireLogistica(r); !autorizado {
		writeError(w, http.StatusForbidden, "solo el Encargado de Logística puede configurar el checklist")
		return
	}
	id, ok := h.clienteDeRuta(w, r)
	if !ok {
		return
	}
	if _, err := h.DB.Exec(r.Context(), `DELETE FROM criterio_checklist WHERE cliente_id = $1`, id); err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo restablecer el checklist")
		return
	}
	h.responderChecklist(w, r.Context(), id)
}
