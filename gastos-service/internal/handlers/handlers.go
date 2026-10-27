package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"gastos-service/internal/auth"
	"gastos-service/internal/config"
	"gastos-service/internal/crud"
	"gastos-service/internal/ocr"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Handler agrupa las dependencias compartidas por todos los endpoints.
type Handler struct {
	DB   *pgxpool.Pool
	Cfg  config.Config
	Auth *auth.Servicio
	// Lector lee los comprobantes para sugerir el checklist; las pruebas lo reemplazan por uno falso.
	Lector ocr.Lector
}

func New(pool *pgxpool.Pool, cfg config.Config) *Handler {
	return &Handler{DB: pool, Cfg: cfg, Auth: &auth.Servicio{DB: pool, Clave: []byte(cfg.JWTSecret)}, Lector: ocr.NuevoLector()}
}

// Servidor es el servicio completo tal como lo atiende main.go: las rutas de Register detrás del
// middleware de autenticación y de CORS. Las pruebas de integración usan este mismo armado.
func (h *Handler) Servidor() http.Handler {
	mux := http.NewServeMux()
	h.Register(mux)
	return CORS(h.Auth.Autenticar(mux, esPublica))
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
	mux.HandleFunc("GET /gastos/{id}/historial", h.historialGasto)
	// Corregir monto o fecha de un gasto aún no sustentado, con motivo y rastro en la bitácora
	mux.HandleFunc("POST /gastos/{id}/correccion", h.corregirGasto)

	// Registro de Comprobante de Pago (RF-4, RF-5, RF-6)
	mux.HandleFunc("POST /gastos/{id}/comprobantes", h.uploadComprobante)

	// Aprobación de Comprobante de Pago (RF-7, RF-8, RF-9)
	mux.HandleFunc("GET /comprobantes", h.listComprobantes)
	mux.HandleFunc("POST /comprobantes/{id}/aprobar", h.aprobarComprobante)
	mux.HandleFunc("POST /comprobantes/{id}/rechazar", h.rechazarComprobante)

	// Lectura automática del comprobante para sugerir el checklist (OCR)
	mux.HandleFunc("GET /comprobantes/{id}/lectura", h.lecturaComprobante)

	// Checklist documental configurable por cliente (REQ-15, REQ-16)
	mux.HandleFunc("GET /clientes/{id}/checklist", h.getChecklistCliente)
	mux.HandleFunc("PUT /clientes/{id}/checklist", h.putChecklistCliente)
	mux.HandleFunc("DELETE /clientes/{id}/checklist", h.deleteChecklistCliente)

	// Notificaciones de gastos sin comprobante y de cobros cercanos (REQ-20, REQ-21)
	mux.HandleFunc("GET /notificaciones", h.listNotificaciones)
	mux.HandleFunc("POST /notificaciones/{id}/leida", h.marcarNotificacionLeida)

	// Reportes estadísticos (dashboard gerencial y PDF)
	mux.HandleFunc("GET /reportes/resumen", h.resumenReporte)
	mux.HandleFunc("GET /reportes/estadistico.pdf", h.pdfEstadistico)
	mux.HandleFunc("GET /reportes/detalle.pdf", h.pdfDetalle)

	// API de maestros y operación (antes en la API de Python): 19 recursos CRUD
	crud.Registrar(mux, &crud.Motor{DB: h.DB, Actor: autorDeCambios}, "/api/v1", crud.Recursos)
	h.Auth.Registrar(mux, "/api/v1")

	// Archivos de comprobantes servidos estáticamente
	mux.Handle("GET /uploads/", http.StripPrefix("/uploads/", http.FileServer(soloArchivos{http.Dir(h.Cfg.UploadsDir)})))
}

// soloArchivos es un sistema de archivos que no muestra carpetas: pedir un directorio da 404 en
// vez del listado que arma http.FileServer, que enlazaría todos los comprobantes a quien lo pida.
type soloArchivos struct{ http.FileSystem }

func (s soloArchivos) Open(nombre string) (http.File, error) {
	f, err := s.FileSystem.Open(nombre)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil || info.IsDir() {
		f.Close()
		return nil, os.ErrNotExist
	}
	return f, nil
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "env": h.Cfg.AppEnv})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// instante devuelve la expresión SQL que lee una columna TIMESTAMP sin zona (creado_en,
// aprobado_en) como instante real. Esas columnas guardan la hora de pared del servidor de
// base de datos (now()); sin esto pgx las marcaría como UTC y el frontend, que muestra en
// hora de Lima, las adelantaría 5 horas hacia atrás. El alias conserva el nombre de la columna.
func instante(columna string) string {
	nombre := columna
	if i := strings.LastIndex(columna, "."); i >= 0 {
		nombre = columna[i+1:]
	}
	return columna + " AT TIME ZONE current_setting('TimeZone') AS " + nombre
}

// writeFallo responde un 500 con el mensaje genérico y deja la causa técnica en el log estándar
// (que registro.Configurar manda también al archivo de errores, RNF-14). El cliente nunca ve err.
func writeFallo(w http.ResponseWriter, r *http.Request, err error, msg string) {
	log.Printf("%s %s: %s: %v", r.Method, r.URL.Path, msg, err)
	writeError(w, http.StatusInternalServerError, msg)
}
