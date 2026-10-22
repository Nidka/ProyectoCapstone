package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gastos-service/internal/checklist"
	"gastos-service/internal/models"

	"github.com/jackc/pgx/v5"
)

const maxUploadSize = 5 << 20 // 5 MB (RNF-5)

// uploadComprobante implementa RF-4 (adjuntar comprobante a un gasto),
// RF-6 (rechazar formatos que no sean imagen o PDF) y RF-5 (el gasto pasa
// a Pagado automáticamente al adjuntar un comprobante).
func (h *Handler) uploadComprobante(w http.ResponseWriter, r *http.Request) {
	gastoID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || gastoID <= 0 {
		writeError(w, http.StatusBadRequest, "id de gasto inválido")
		return
	}

	ctx := context.Background()

	var gastoExiste bool
	if err := h.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM gasto WHERE id = $1)`, gastoID).Scan(&gastoExiste); err != nil {
		writeError(w, http.StatusInternalServerError, "error validando el gasto")
		return
	}
	if !gastoExiste {
		writeError(w, http.StatusNotFound, "el gasto indicado no existe")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		writeError(w, http.StatusBadRequest, "el archivo es demasiado grande o el formulario es inválido")
		return
	}

	file, header, err := r.FormFile("archivo")
	if err != nil {
		writeError(w, http.StatusBadRequest, "adjunta un archivo en el campo 'archivo'")
		return
	}
	defer file.Close()

	// RF-6: se valida por contenido real del archivo, no solo por su extensión.
	buf := make([]byte, 512)
	n, _ := file.Read(buf)
	contentType := http.DetectContentType(buf[:n])
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo leer el archivo")
		return
	}

	tipoArchivo, ok := clasificarTipoArchivo(contentType, header.Filename)
	if !ok {
		writeError(w, http.StatusUnprocessableEntity, "solo se permiten archivos de imagen o PDF")
		return
	}

	ext := filepath.Ext(header.Filename)
	filename := fmt.Sprintf("gasto-%d-%d%s", gastoID, time.Now().UnixNano(), ext)
	destPath := filepath.Join(h.Cfg.UploadsDir, filename)

	dest, err := os.Create(destPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo guardar el archivo")
		return
	}
	defer dest.Close()

	if _, err := io.Copy(dest, file); err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo guardar el archivo")
		return
	}

	// RNF-3: el comprobante no se edita una vez guardado; cada subida crea un
	// registro nuevo con su propio archivo.
	archivoURL := "/uploads/" + filename

	tx, err := h.DB.Begin(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo iniciar la operación")
		return
	}
	defer tx.Rollback(ctx)

	var c models.Comprobante
	err = tx.QueryRow(ctx, `
		INSERT INTO comprobante (gasto_id, archivo_url, tipo_archivo, estado, creado_en)
		VALUES ($1, $2, $3, 'Pendiente', now())
		RETURNING id, gasto_id, archivo_url, tipo_archivo, estado, motivo_rechazo, aprobado_por, aprobado_en, creado_en
	`, gastoID, archivoURL, tipoArchivo).Scan(
		&c.ID, &c.GastoID, &c.ArchivoURL, &c.TipoArchivo, &c.Estado,
		&c.MotivoRechazo, &c.AprobadoPor, &c.AprobadoEn, &c.CreadoEn,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo registrar el comprobante")
		return
	}

	// RF-5: adjuntar un comprobante cambia el gasto a "Pagado".
	if _, err := tx.Exec(ctx, `UPDATE gasto SET estado = 'Pagado' WHERE id = $1`, gastoID); err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo actualizar el estado del gasto")
		return
	}

	if err := tx.Commit(ctx); err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo confirmar la operación")
		return
	}

	c.Checklist = []models.ItemChecklist{}
	writeJSON(w, http.StatusCreated, c)
}

func clasificarTipoArchivo(contentType, filename string) (string, bool) {
	lower := strings.ToLower(filename)
	switch {
	case strings.HasPrefix(contentType, "image/"):
		return "imagen", true
	case contentType == "application/pdf", strings.HasSuffix(lower, ".pdf"):
		return "pdf", true
	case strings.HasSuffix(lower, ".jpg"), strings.HasSuffix(lower, ".jpeg"),
		strings.HasSuffix(lower, ".png"), strings.HasSuffix(lower, ".webp"):
		return "imagen", true
	default:
		return "", false
	}
}

// listComprobantes soporta la pantalla de aprobación (filtrando por
// ?estado=Pendiente) y el historial; cada comprobante trae su checklist.
func (h *Handler) listComprobantes(w http.ResponseWriter, r *http.Request) {
	estado := r.URL.Query().Get("estado")
	ctx := context.Background()

	query := `
		SELECT id, gasto_id, archivo_url, tipo_archivo, estado, motivo_rechazo, aprobado_por, aprobado_en, creado_en
		FROM comprobante`
	args := []any{}
	if estado != "" {
		query += ` WHERE estado = $1`
		args = append(args, estado)
	}
	query += ` ORDER BY creado_en DESC LIMIT 200`

	rows, err := h.DB.Query(ctx, query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo listar los comprobantes")
		return
	}
	defer rows.Close()

	out := []models.Comprobante{}
	ids := []int{}
	for rows.Next() {
		var c models.Comprobante
		if err := rows.Scan(&c.ID, &c.GastoID, &c.ArchivoURL, &c.TipoArchivo, &c.Estado, &c.MotivoRechazo, &c.AprobadoPor, &c.AprobadoEn, &c.CreadoEn); err != nil {
			writeError(w, http.StatusInternalServerError, "error leyendo los comprobantes")
			return
		}
		c.Checklist = []models.ItemChecklist{}
		out = append(out, c)
		ids = append(ids, c.ID)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "error leyendo los comprobantes")
		return
	}
	rows.Close()

	if err := h.adjuntarChecklists(ctx, out, ids); err != nil {
		writeError(w, http.StatusInternalServerError, "error leyendo los checklists")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// adjuntarChecklists carga en una sola consulta el checklist de los comprobantes listados.
func (h *Handler) adjuntarChecklists(ctx context.Context, comprobantes []models.Comprobante, ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	rows, err := h.DB.Query(ctx, `SELECT comprobante_id, criterio, cumple FROM checklist_comprobante WHERE comprobante_id = ANY($1)`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()

	porComprobante := map[int][]models.ItemChecklist{}
	for rows.Next() {
		var id int
		var it models.ItemChecklist
		if err := rows.Scan(&id, &it.Criterio, &it.Cumple); err != nil {
			return err
		}
		porComprobante[id] = append(porComprobante[id], it)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for i := range comprobantes {
		if items, ok := porComprobante[comprobantes[i].ID]; ok {
			comprobantes[i].Checklist = checklist.Ordenar(items)
		}
	}
	return nil
}

var (
	errComprobanteNoEncontrado = errors.New("comprobante no encontrado")
	errYaRevisado              = errors.New("el comprobante ya fue revisado")
)

// bloquearPendiente toma el comprobante con FOR UPDATE y exige que siga Pendiente;
// devuelve el gasto al que pertenece.
func bloquearPendiente(ctx context.Context, tx pgx.Tx, id int) (int, error) {
	var gastoID int
	var estado string
	err := tx.QueryRow(ctx, `SELECT gasto_id, estado FROM comprobante WHERE id = $1 FOR UPDATE`, id).Scan(&gastoID, &estado)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, errComprobanteNoEncontrado
	}
	if err != nil {
		return 0, err
	}
	if estado != "Pendiente" {
		return 0, errYaRevisado
	}
	return gastoID, nil
}

func guardarChecklist(ctx context.Context, tx pgx.Tx, id int, items []models.ItemChecklist) error {
	for _, it := range items {
		if _, err := tx.Exec(ctx, `INSERT INTO checklist_comprobante (comprobante_id, criterio, cumple) VALUES ($1, $2, $3)`, id, it.Criterio, it.Cumple); err != nil {
			return err
		}
	}
	return nil
}

func escribirErrorRevision(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errComprobanteNoEncontrado):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, errYaRevisado):
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "no se pudo completar la revisión")
	}
}

type aprobarRequest struct {
	AprobadoPor string          `json:"aprobado_por"`
	Checklist   map[string]bool `json:"checklist"`
}

// aprobarComprobante implementa RF-7 (aprobar) para el rol Encargado de
// Logística: exige el checklist documental completo y hace avanzar el gasto
// a "Sustentado".
func (h *Handler) aprobarComprobante(w http.ResponseWriter, r *http.Request) {
	nombre, autorizado := requireLogistica(r)
	if !autorizado {
		writeError(w, http.StatusForbidden, "solo el Encargado de Logística puede aprobar comprobantes")
		return
	}

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id inválido")
		return
	}

	var req aprobarRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	if err := checklist.ValidarAprobacion(req.Checklist); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if req.AprobadoPor != "" {
		nombre = req.AprobadoPor
	}

	ctx := context.Background()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo iniciar la operación")
		return
	}
	defer tx.Rollback(ctx)

	gastoID, err := bloquearPendiente(ctx, tx, id)
	if err != nil {
		escribirErrorRevision(w, err)
		return
	}
	if err := guardarChecklist(ctx, tx, id, checklist.Items(req.Checklist)); err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo guardar el checklist")
		return
	}
	if _, err := tx.Exec(ctx, `UPDATE comprobante SET estado = 'Aprobado', aprobado_por = $1, aprobado_en = now() WHERE id = $2`, nombre, id); err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo aprobar el comprobante")
		return
	}
	if _, err := tx.Exec(ctx, `UPDATE gasto SET estado = 'Sustentado' WHERE id = $1`, gastoID); err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo actualizar el gasto")
		return
	}

	if err := tx.Commit(ctx); err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo confirmar la operación")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "aprobado"})
}

type rechazarRequest struct {
	MotivoRechazo string          `json:"motivo_rechazo"`
	AprobadoPor   string          `json:"aprobado_por"`
	Checklist     map[string]bool `json:"checklist"`
}

// rechazarComprobante implementa RF-8 (motivo obligatorio) y RF-9 (el gasto
// vuelve a "Registrado"); guarda el checklist si viene.
func (h *Handler) rechazarComprobante(w http.ResponseWriter, r *http.Request) {
	nombre, autorizado := requireLogistica(r)
	if !autorizado {
		writeError(w, http.StatusForbidden, "solo el Encargado de Logística puede rechazar comprobantes")
		return
	}

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id inválido")
		return
	}

	var req rechazarRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	if strings.TrimSpace(req.MotivoRechazo) == "" {
		writeError(w, http.StatusBadRequest, "motivo_rechazo es obligatorio")
		return
	}
	if err := checklist.ValidarRechazo(req.Checklist); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if req.AprobadoPor != "" {
		nombre = req.AprobadoPor
	}

	ctx := context.Background()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo iniciar la operación")
		return
	}
	defer tx.Rollback(ctx)

	gastoID, err := bloquearPendiente(ctx, tx, id)
	if err != nil {
		escribirErrorRevision(w, err)
		return
	}
	if err := guardarChecklist(ctx, tx, id, checklist.Items(req.Checklist)); err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo guardar el checklist")
		return
	}
	if _, err := tx.Exec(ctx, `UPDATE comprobante SET estado = 'Rechazado', motivo_rechazo = $1, aprobado_por = $2, aprobado_en = now() WHERE id = $3`, req.MotivoRechazo, nombre, id); err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo rechazar el comprobante")
		return
	}
	if _, err := tx.Exec(ctx, `UPDATE gasto SET estado = 'Registrado' WHERE id = $1`, gastoID); err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo actualizar el gasto")
		return
	}

	if err := tx.Commit(ctx); err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo confirmar la operación")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "rechazado"})
}
