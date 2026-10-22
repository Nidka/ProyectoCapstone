package handlers

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"gastos-service/internal/reportes"
)

// filtroReporte lee ?desde=&hasta=; si no sirven, ya respondió (400 o 500).
func filtroReporte(w http.ResponseWriter, r *http.Request) (reportes.Filtro, bool) {
	f, err := reportes.ParseFiltro(r.URL.Query().Get("desde"), r.URL.Query().Get("hasta"), time.Now())
	var ef reportes.ErrFiltro
	switch {
	case errors.As(err, &ef):
		writeError(w, http.StatusBadRequest, ef.Mensaje)
		return f, false
	case err != nil:
		errorDeReporte(w, err)
		return f, false
	}
	return f, true
}

// errorDeReporte registra la causa en el log y responde el 500 genérico.
func errorDeReporte(w http.ResponseWriter, err error) {
	log.Printf("reportes: %v", err)
	writeError(w, http.StatusInternalServerError, "no se pudo generar el reporte")
}

// resumenReporte devuelve las estadísticas del dashboard gerencial.
func (h *Handler) resumenReporte(w http.ResponseWriter, r *http.Request) {
	f, ok := filtroReporte(w, r)
	if !ok {
		return
	}
	resumen, err := reportes.Consultar(r.Context(), h.DB, f, false)
	if err != nil {
		errorDeReporte(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resumen)
}

// pdfEstadistico descarga el reporte estadístico del rango.
func (h *Handler) pdfEstadistico(w http.ResponseWriter, r *http.Request) {
	h.enviarPDF(w, r, "reporte-estadistico", false, reportes.PDFEstadistico)
}

// pdfDetalle descarga el listado de gastos del rango.
func (h *Handler) pdfDetalle(w http.ResponseWriter, r *http.Request) {
	h.enviarPDF(w, r, "detalle-gastos", true, reportes.PDFDetalle)
}

func (h *Handler) enviarPDF(w http.ResponseWriter, r *http.Request, nombre string, conDetalle bool, generar func(*reportes.Resumen, time.Time) ([]byte, error)) {
	f, ok := filtroReporte(w, r)
	if !ok {
		return
	}
	resumen, err := reportes.Consultar(r.Context(), h.DB, f, conDetalle)
	if err != nil {
		errorDeReporte(w, err)
		return
	}
	pdf, err := generar(resumen, time.Now())
	if err != nil {
		errorDeReporte(w, err)
		return
	}
	archivo := nombre + "-" + f.DesdeISO() + "_" + f.HastaISO() + ".pdf"
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="`+archivo+`"`)
	w.Header().Set("Content-Length", strconv.Itoa(len(pdf)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdf)
}
