package handlers

import (
	"context"
	"net/http"
	"strings"

	"gastos-service/internal/auth"
	"gastos-service/internal/bitacora"
)

// CORS habilita el acceso desde el frontend Angular (servido en otro origen).
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		// El frontend lee el nombre del PDF descargado
		w.Header().Set("Access-Control-Expose-Headers", "Content-Disposition")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// requireLogistica dice si quien llama es el Encargado de Logística (RF-7) y devuelve su nombre,
// que queda como autor de lo que registra o aprueba. La identidad sale del token y de la base
// (auth.Servicio.Autenticar), nunca de cabeceras del cliente.
func requireLogistica(r *http.Request) (nombre string, autorizado bool) {
	id, ok := auth.IdentidadDe(r.Context())
	return id.Nombre, ok && id.Rol == auth.RolLogistica
}

// actorDe es el usuario autenticado de la petición, como autor de las entradas de la bitácora.
func actorDe(r *http.Request) bitacora.Actor {
	id, _ := auth.IdentidadDe(r.Context())
	return id.Actor()
}

// autorDeCambios es quien hace una operación sobre los maestros, tomado de la identidad del contexto.
// Sin identidad (algo que Servidor no permite) no hay autor y el motor se niega a cambiar nada.
func autorDeCambios(ctx context.Context) (bitacora.Actor, bool) {
	id, ok := auth.IdentidadDe(ctx)
	return id.Actor(), ok
}

// esPublica lista lo único que se atiende sin token: la salud del servicio, el inicio de sesión y
// los comprobantes subidos. Estos últimos los carga el navegador con <img> y <a href>, que no
// pueden enviar la cabecera Authorization. Es una excepción consciente: quien conozca la URL exacta
// de un archivo puede verlo. Por eso /uploads no lista carpetas (soloArchivos) y los nombres nuevos
// llevan una parte aleatoria de 128 bits (comprobantes.go); los archivos subidos antes de ese
// cambio conservan su nombre anterior (gasto-<id>-<hora>).
func esPublica(r *http.Request) bool {
	lectura := r.Method == http.MethodGet || r.Method == http.MethodHead
	switch {
	case lectura && r.URL.Path == "/health":
		return true
	case r.Method == http.MethodPost && r.URL.Path == "/api/v1/auth/login":
		return true
	case lectura && strings.HasPrefix(r.URL.Path, "/uploads/"):
		return true
	}
	return false
}
