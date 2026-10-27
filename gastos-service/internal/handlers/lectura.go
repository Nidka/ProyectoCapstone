package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"gastos-service/internal/ocr"

	"github.com/jackc/pgx/v5"
)

type respuestaLectura struct {
	Estado      string                    `json:"estado"`
	Datos       ocr.Datos                 `json:"datos"`
	Sugerencias map[string]ocr.Sugerencia `json:"sugerencias"`
}

// lecturaComprobante lee un comprobante Pendiente (o reusa su lectura guardada) y sugiere qué
// criterios del checklist cumple, comparando con el monto y la fecha actuales del gasto. Es de
// lectura: no exige el rol de Logística.
func (h *Handler) lecturaComprobante(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "id inválido")
		return
	}
	ctx := r.Context()

	var archivoURL, tipoArchivo, estado string
	var monto float64
	var fecha time.Time
	err = h.DB.QueryRow(ctx, `
		SELECT c.archivo_url, c.tipo_archivo, c.estado, g.monto::float8, g.fecha
		FROM comprobante c JOIN gasto g ON g.id = c.gasto_id
		WHERE c.id = $1`, id).Scan(&archivoURL, &tipoArchivo, &estado, &monto, &fecha)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, errComprobanteNoEncontrado.Error())
		return
	}
	if err != nil {
		writeFallo(w, r, err, "no se pudo leer el comprobante")
		return
	}
	if estado != "Pendiente" {
		writeError(w, http.StatusConflict, errYaRevisado.Error())
		return
	}

	lectura, datos, err := h.lecturaDe(ctx, id, archivoURL, tipoArchivo)
	if err != nil {
		writeFallo(w, r, err, "no se pudo guardar la lectura del comprobante")
		return
	}
	resp := respuestaLectura{Estado: lectura.Estado, Datos: datos, Sugerencias: map[string]ocr.Sugerencia{}}
	if lectura.Estado == ocr.Leido {
		resp.Sugerencias = ocr.Sugerir(datos, lectura.Texto, monto, fecha)
	}
	writeJSON(w, http.StatusOK, resp)
}

// lecturaDe devuelve la lectura guardada del comprobante o, si no hay, lo lee. Guarda solo las
// lecturas definitivas (leido, sin_texto): no_disponible y error se reintentan en la próxima consulta.
func (h *Handler) lecturaDe(ctx context.Context, id int, archivoURL, tipoArchivo string) (ocr.Lectura, ocr.Datos, error) {
	var l ocr.Lectura
	var crudo []byte
	err := h.DB.QueryRow(ctx, `SELECT estado, texto, datos FROM lectura_comprobante WHERE comprobante_id = $1`, id).Scan(&l.Estado, &l.Texto, &crudo)
	if err == nil {
		datos := ocr.Extraer("")
		if err := json.Unmarshal(crudo, &datos); err != nil {
			return l, datos, err
		}
		return l, datos, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return l, ocr.Datos{}, err
	}

	ruta, err := ocr.RutaSegura(h.Cfg.UploadsDir, archivoURL)
	if err != nil {
		return ocr.Lectura{Estado: ocr.Error}, ocr.Extraer(""), nil
	}
	l = h.Lector.Leer(ctx, ruta, tipoArchivo)
	datos := ocr.Extraer(l.Texto)
	if l.Estado != ocr.Leido && l.Estado != ocr.SinTexto {
		return l, datos, nil
	}
	crudo, err = json.Marshal(datos)
	if err != nil {
		return l, datos, err
	}
	_, err = h.DB.Exec(ctx, `
		INSERT INTO lectura_comprobante (comprobante_id, estado, texto, datos) VALUES ($1, $2, $3, $4)
		ON CONFLICT (comprobante_id) DO NOTHING`, id, l.Estado, l.Texto, crudo)
	return l, datos, err
}
