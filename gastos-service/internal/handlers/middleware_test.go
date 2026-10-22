package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gastos-service/internal/handlers"
)

// El navegador solo envía PUT y DELETE (checklist por cliente) si la respuesta al preflight los permite.
func TestCORSPermiteLosMetodosDeLaAPI(t *testing.T) {
	h := handlers.CORS(http.NotFoundHandler())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodOptions, "/clientes/1/checklist", nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("preflight = %d, quiero 204", rec.Code)
	}
	permitidos := rec.Header().Get("Access-Control-Allow-Methods")
	for _, m := range []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"} {
		if !strings.Contains(permitidos, m) {
			t.Errorf("Access-Control-Allow-Methods = %q, falta %s", permitidos, m)
		}
	}
}
