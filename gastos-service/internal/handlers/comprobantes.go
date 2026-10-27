package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gastos-service/internal/bitacora"
	"gastos-service/internal/checklist"
	"gastos-service/internal/models"

	"github.com/jackc/pgx/v5"
)

const maxUploadSize = 5 << 20 // 5 MB (RNF-5)

// sufijoAleatorio son 128 bits aleatorios en hexadecimal: hace que la URL de un comprobante no se
// pueda deducir del id del gasto ni de la hora de subida.
func sufijoAleatorio() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// uploadComprobante implementa RF-4 (adjuntar comprobante a un gasto),
// RF-6 (rechazar formatos que no sean imagen o PDF) y RF-5 (el gasto pasa
// a Pagado automáticamente al adjuntar un comprobante).
func (h *Handler) uploadComprobante(w http.ResponseWriter, r *http.Request) {
	if _, autorizado := requireLogistica(r); !autorizado {
		writeError(w, http.StatusForbidden, "solo el Encargado de Logística puede adjuntar comprobantes")
		return
	}

	gastoID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || gastoID <= 0 {
		writeError(w, http.StatusBadRequest, "id de gasto inválido")
		return
	}

	ctx := context.Background()

	var gastoExiste bool
	if err := h.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM gasto WHERE id = $1)`, gastoID).Scan(&gastoExiste); err != nil {
		writeFallo(w, r, err, "error validando el gasto")
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
		writeFallo(w, r, err, "no se pudo leer el archivo")
		return
	}

	tipoArchivo, ok := clasificarTipoArchivo(contentType, header.Filename)
	if !ok {
		writeError(w, http.StatusUnprocessableEntity, "solo se permiten archivos de imagen o PDF")
		return
	}

	ext := filepath.Ext(header.Filename)
	sufijo, err := sufijoAleatorio()
	if err != nil {
		writeFallo(w, r, err, "no se pudo guardar el archivo")
		return
	}
	filename := fmt.Sprintf("gasto-%d-%s%s", gastoID, sufijo, ext)
	destPath := filepath.Join(h.Cfg.UploadsDir, filename)

	dest, err := os.Create(destPath)
	if err != nil {
		writeFallo(w, r, err, "no se pudo guardar el archivo")
		return
	}
	defer dest.Close()
	// Si la operación no llega a confirmarse (incluida la bitácora), no queda un archivo huérfano.
	confirmado := false
	defer func() {
		if !confirmado {
			_ = os.Remove(destPath)
		}
	}()

	if _, err := io.Copy(dest, file); err != nil {
		writeFallo(w, r, err, "no se pudo guardar el archivo")
		return
	}
	// RNF-5: el comprobante guardado queda en solo lectura. Borrarlo (si la operación falla)
	// sigue funcionando, porque eso depende de los permisos del directorio y no del archivo.
	if err := dest.Close(); err != nil {
		writeFallo(w, r, err, "no se pudo guardar el archivo")
		return
	}
	if err := os.Chmod(destPath, 0o444); err != nil {
		writeFallo(w, r, err, "no se pudo guardar el archivo")
		return
	}

	// RNF-3: el comprobante no se edita una vez guardado; cada subida crea un
	// registro nuevo con su propio archivo.
	archivoURL := "/uploads/" + filename

	tx, err := h.DB.Begin(ctx)
	if err != nil {
		writeFallo(w, r, err, "no se pudo iniciar la operación")
		return
	}
	defer tx.Rollback(ctx)

	estadoAntes, err := estadoDelGasto(ctx, tx, gastoID)
	if err != nil {
		writeFallo(w, r, err, "no se pudo leer el estado del gasto")
		return
	}

	var c models.Comprobante
	err = tx.QueryRow(ctx, `
		INSERT INTO comprobante (gasto_id, archivo_url, tipo_archivo, estado, creado_en)
		VALUES ($1, $2, $3, 'Pendiente', now())
		RETURNING id, gasto_id, archivo_url, tipo_archivo, estado, motivo_rechazo, aprobado_por, `+instante("aprobado_en")+`, `+instante("creado_en")+`
	`, gastoID, archivoURL, tipoArchivo).Scan(
		&c.ID, &c.GastoID, &c.ArchivoURL, &c.TipoArchivo, &c.Estado,
		&c.MotivoRechazo, &c.AprobadoPor, &c.AprobadoEn, &c.CreadoEn,
	)
	if err != nil {
		writeFallo(w, r, err, "no se pudo registrar el comprobante")
		return
	}

	// RF-5: adjuntar un comprobante cambia el gasto a "Pagado".
	if _, err := tx.Exec(ctx, `UPDATE gasto SET estado = 'Pagado' WHERE id = $1`, gastoID); err != nil {
		writeFallo(w, r, err, "no se pudo actualizar el estado del gasto")
		return
	}
	err = bitacora.Registrar(ctx, tx, bitacora.Entrada{
		Actor: actorDe(r), Accion: "comprobante.adjuntar", Entidad: "comprobante", IDEntidad: int64(c.ID),
		Detalle: map[string]any{
			"gasto_id": gastoID,
			"antes":    map[string]any{"gasto_estado": estadoAntes},
			"despues": map[string]any{
				"estado": c.Estado, "gasto_estado": "Pagado", "archivo_url": c.ArchivoURL, "tipo_archivo": c.TipoArchivo,
			},
		},
	})
	if err != nil {
		writeFallo(w, r, err, "no se pudo registrar el comprobante")
		return
	}

	if err := tx.Commit(ctx); err != nil {
		writeFallo(w, r, err, "no se pudo confirmar la operación")
		return
	}

	confirmado = true
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
		SELECT id, gasto_id, archivo_url, tipo_archivo, estado, motivo_rechazo, aprobado_por, ` + instante("aprobado_en") + `, ` + instante("creado_en") + `
		FROM comprobante`
	args := []any{}
	if estado != "" {
		query += ` WHERE estado = $1`
		args = append(args, estado)
	}
	query += ` ORDER BY creado_en DESC LIMIT 200`

	rows, err := h.DB.Query(ctx, query, args...)
	if err != nil {
		writeFallo(w, r, err, "no se pudo listar los comprobantes")
		return
	}
	defer rows.Close()

	out := []models.Comprobante{}
	ids := []int{}
	for rows.Next() {
		var c models.Comprobante
		if err := rows.Scan(&c.ID, &c.GastoID, &c.ArchivoURL, &c.TipoArchivo, &c.Estado, &c.MotivoRechazo, &c.AprobadoPor, &c.AprobadoEn, &c.CreadoEn); err != nil {
			writeFallo(w, r, err, "error leyendo los comprobantes")
			return
		}
		c.Checklist = []models.ItemChecklist{}
		out = append(out, c)
		ids = append(ids, c.ID)
	}
	if err := rows.Err(); err != nil {
		writeFallo(w, r, err, "error leyendo los comprobantes")
		return
	}
	rows.Close()

	if err := h.adjuntarChecklists(ctx, out, ids); err != nil {
		writeFallo(w, r, err, "error leyendo los checklists")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// adjuntarChecklists carga en una sola consulta el checklist de los comprobantes listados.
func (h *Handler) adjuntarChecklists(ctx context.Context, comprobantes []models.Comprobante, ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	rows, err := h.DB.Query(ctx, `SELECT comprobante_id, criterio, texto, cumple FROM checklist_comprobante WHERE comprobante_id = ANY($1) ORDER BY id`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()

	porComprobante := map[int][]models.ItemChecklist{}
	for rows.Next() {
		var id int
		var it models.ItemChecklist
		if err := rows.Scan(&id, &it.Criterio, &it.Texto, &it.Cumple); err != nil {
			return err
		}
		porComprobante[id] = append(porComprobante[id], it)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for i := range comprobantes {
		if items, ok := porComprobante[comprobantes[i].ID]; ok {
			comprobantes[i].Checklist = items
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

// criteriosDelComprobante bloquea el comprobante pendiente y devuelve su gasto y los criterios
// vigentes del cliente de ese gasto (los suyos o los estándar).
func criteriosDelComprobante(ctx context.Context, tx pgx.Tx, id int) (int, []checklist.Criterio, error) {
	gastoID, err := bloquearPendiente(ctx, tx, id)
	if err != nil {
		return 0, nil, err
	}
	var clienteID int
	if err := tx.QueryRow(ctx, `SELECT cliente_id FROM gasto WHERE id = $1`, gastoID).Scan(&clienteID); err != nil {
		return 0, nil, err
	}
	vigentes, _, err := criteriosVigentes(ctx, tx, clienteID)
	return gastoID, vigentes, err
}

// estadoDelGasto lee el estado actual del gasto y lo bloquea hasta el final de la transacción: es
// el «antes» que queda en la bitácora, y nadie más puede cambiarlo mientras tanto.
func estadoDelGasto(ctx context.Context, tx pgx.Tx, gastoID int) (string, error) {
	var estado string
	err := tx.QueryRow(ctx, `SELECT estado FROM gasto WHERE id = $1 FOR UPDATE`, gastoID).Scan(&estado)
	return estado, err
}

// registrarRevision deja en la bitácora la aprobación o el rechazo de un comprobante (REQ-33):
// el estado antes y después del comprobante y del gasto, el checklist revisado y, si lo hay, el motivo.
func registrarRevision(ctx context.Context, tx pgx.Tx, actor bitacora.Actor, accion string, comprobanteID, gastoID int, gastoAntes, estado, gastoDespues, motivo string, items []models.ItemChecklist) error {
	despues := map[string]any{"estado": estado, "gasto_estado": gastoDespues}
	if motivo != "" {
		despues["motivo_rechazo"] = motivo
	}
	return bitacora.Registrar(ctx, tx, bitacora.Entrada{
		Actor: actor, Accion: accion, Entidad: "comprobante", IDEntidad: int64(comprobanteID),
		Detalle: map[string]any{
			"gasto_id":  gastoID,
			"antes":     map[string]any{"estado": "Pendiente", "gasto_estado": gastoAntes},
			"despues":   despues,
			"checklist": items,
		},
	})
}

func guardarChecklist(ctx context.Context, tx pgx.Tx, id int, items []models.ItemChecklist) error {
	for _, it := range items {
		if _, err := tx.Exec(ctx, `INSERT INTO checklist_comprobante (comprobante_id, criterio, texto, cumple) VALUES ($1, $2, $3, $4)`, id, it.Criterio, it.Texto, it.Cumple); err != nil {
			return err
		}
	}
	return nil
}

func escribirErrorRevision(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, errComprobanteNoEncontrado):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, errYaRevisado):
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeFallo(w, r, err, "no se pudo completar la revisión")
	}
}

type aprobarRequest struct {
	Checklist map[string]bool `json:"checklist"`
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

	ctx := context.Background()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		writeFallo(w, r, err, "no se pudo iniciar la operación")
		return
	}
	defer tx.Rollback(ctx)

	gastoID, vigentes, err := criteriosDelComprobante(ctx, tx, id)
	if err != nil {
		escribirErrorRevision(w, r, err)
		return
	}
	if err := checklist.ValidarAprobacion(vigentes, req.Checklist); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	items := checklist.Items(vigentes, req.Checklist)
	if err := guardarChecklist(ctx, tx, id, items); err != nil {
		writeFallo(w, r, err, "no se pudo guardar el checklist")
		return
	}
	gastoAntes, err := estadoDelGasto(ctx, tx, gastoID)
	if err != nil {
		writeFallo(w, r, err, "no se pudo leer el estado del gasto")
		return
	}
	if _, err := tx.Exec(ctx, `UPDATE comprobante SET estado = 'Aprobado', aprobado_por = $1, aprobado_en = now() WHERE id = $2`, nombre, id); err != nil {
		writeFallo(w, r, err, "no se pudo aprobar el comprobante")
		return
	}
	if _, err := tx.Exec(ctx, `UPDATE gasto SET estado = 'Sustentado' WHERE id = $1`, gastoID); err != nil {
		writeFallo(w, r, err, "no se pudo actualizar el gasto")
		return
	}
	if err := registrarRevision(ctx, tx, actorDe(r), "comprobante.aprobar", id, gastoID, gastoAntes, "Aprobado", "Sustentado", "", items); err != nil {
		writeFallo(w, r, err, "no se pudo registrar la revisión")
		return
	}

	if err := tx.Commit(ctx); err != nil {
		writeFallo(w, r, err, "no se pudo confirmar la operación")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "aprobado"})
}

type rechazarRequest struct {
	MotivoRechazo string          `json:"motivo_rechazo"`
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

	ctx := context.Background()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		writeFallo(w, r, err, "no se pudo iniciar la operación")
		return
	}
	defer tx.Rollback(ctx)

	gastoID, vigentes, err := criteriosDelComprobante(ctx, tx, id)
	if err != nil {
		escribirErrorRevision(w, r, err)
		return
	}
	if err := checklist.ValidarRechazo(vigentes, req.Checklist); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	items := checklist.Items(vigentes, req.Checklist)
	if err := guardarChecklist(ctx, tx, id, items); err != nil {
		writeFallo(w, r, err, "no se pudo guardar el checklist")
		return
	}
	gastoAntes, err := estadoDelGasto(ctx, tx, gastoID)
	if err != nil {
		writeFallo(w, r, err, "no se pudo leer el estado del gasto")
		return
	}
	if _, err := tx.Exec(ctx, `UPDATE comprobante SET estado = 'Rechazado', motivo_rechazo = $1, aprobado_por = $2, aprobado_en = now() WHERE id = $3`, req.MotivoRechazo, nombre, id); err != nil {
		writeFallo(w, r, err, "no se pudo rechazar el comprobante")
		return
	}
	if _, err := tx.Exec(ctx, `UPDATE gasto SET estado = 'Registrado' WHERE id = $1`, gastoID); err != nil {
		writeFallo(w, r, err, "no se pudo actualizar el gasto")
		return
	}
	if err := registrarRevision(ctx, tx, actorDe(r), "comprobante.rechazar", id, gastoID, gastoAntes, "Rechazado", "Registrado", req.MotivoRechazo, items); err != nil {
		writeFallo(w, r, err, "no se pudo registrar la revisión")
		return
	}

	if err := tx.Commit(ctx); err != nil {
		writeFallo(w, r, err, "no se pudo confirmar la operación")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "rechazado"})
}
