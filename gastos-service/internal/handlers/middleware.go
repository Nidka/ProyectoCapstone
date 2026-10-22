package handlers

import "net/http"

// CORS habilita el acceso desde el frontend Angular (servido en otro origen).
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-User-Role, X-User-Name")
		// El frontend lee el nombre del PDF descargado
		w.Header().Set("Access-Control-Expose-Headers", "Content-Disposition")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// requireLogistica simula el control de rol "Encargado de Logística" (RF-7)
// mientras no exista el módulo de Gestión de Usuarios (RF-19 a RF-21).
// El rol y el nombre del usuario actual viajan en headers desde el frontend.
func requireLogistica(r *http.Request) (nombre string, autorizado bool) {
	role := r.Header.Get("X-User-Role")
	nombre = headerOrDefault(r, "X-User-Name", "Encargado de Logística")
	return nombre, role == "logistica"
}
