package handlers

import (
	"encoding/json"
	"net/http"

	"gastos-service/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Handler agrupa las dependencias compartidas por todos los endpoints.
type Handler struct {
	DB  *pgxpool.Pool
	Cfg config.Config
}

func New(pool *pgxpool.Pool, cfg config.Config) *Handler {
	return &Handler{DB: pool, Cfg: cfg}
}

// Register expone las rutas del Módulo de Registro de Gastos y Comprobantes
// y arranca lo indispensable del Módulo de Aprobación de Comprobantes.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", h.health)

	// Registro de Gasto Adicional (RF-1, RF-2, RF-3)
	mux.HandleFunc("POST /gastos", h.createGasto)
	mux.HandleFunc("GET /gastos", h.listGastos)
	mux.HandleFunc("GET /gastos/exportar.xlsx", h.exportarGastos)
	mux.HandleFunc("GET /gastos/{id}", h.getGasto)

	// Registro de Comprobante de Pago (RF-4, RF-5, RF-6)
	mux.HandleFunc("POST /gastos/{id}/comprobantes", h.uploadComprobante)

	// Aprobación de Comprobante de Pago (RF-7, RF-8, RF-9)
	mux.HandleFunc("GET /comprobantes", h.listComprobantes)
	mux.HandleFunc("POST /comprobantes/{id}/aprobar", h.aprobarComprobante)
	mux.HandleFunc("POST /comprobantes/{id}/rechazar", h.rechazarComprobante)

	// Reportes estadísticos (dashboard gerencial y PDF)
	mux.HandleFunc("GET /reportes/resumen", h.resumenReporte)
	mux.HandleFunc("GET /reportes/estadistico.pdf", h.pdfEstadistico)
	mux.HandleFunc("GET /reportes/detalle.pdf", h.pdfDetalle)

	// Archivos de comprobantes servidos estáticamente
	mux.Handle("GET /uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir(h.Cfg.UploadsDir))))
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "gastos-service"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func headerOrDefault(r *http.Request, key, fallback string) string {
	if v := r.Header.Get(key); v != "" {
		return v
	}
	return fallback
}
