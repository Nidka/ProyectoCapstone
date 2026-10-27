package handlers_test

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

const cincoMB = 5 << 20

// pngDe devuelve un PNG válido en su cabecera, rellenado hasta el tamaño pedido.
func pngDe(tamano int) []byte {
	cabecera := []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}
	return append(cabecera, make([]byte, tamano-len(cabecera))...)
}

func (e *entorno) subirComprobante(t *testing.T, gastoID int, nombre string, contenido []byte) *httptest.ResponseRecorder {
	t.Helper()
	var cuerpo bytes.Buffer
	m := multipart.NewWriter(&cuerpo)
	parte, err := m.CreateFormFile("archivo", nombre)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = parte.Write(contenido)
	_ = m.Close()
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/gastos/%d/comprobantes", gastoID), &cuerpo)
	req.Header.Set("Content-Type", m.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+e.tokenLogistica)
	rec := httptest.NewRecorder()
	e.mux.ServeHTTP(rec, req)
	return rec
}

func (e *entorno) gastoRegistrado(t *testing.T) int {
	t.Helper()
	var id int
	ctx := context.Background()
	_, err := e.pool.Exec(ctx, `INSERT INTO programacionviaje DEFAULT VALUES; INSERT INTO cliente DEFAULT VALUES;`)
	if err != nil {
		t.Fatal(err)
	}
	err = e.pool.QueryRow(ctx, `INSERT INTO gasto (viaje_id, cliente_id, monto, fecha, estado, creado_por)
		VALUES ((SELECT max(id_viaje) FROM programacionviaje), (SELECT max(id_cliente) FROM cliente), 10, '2026-10-01', 'Registrado', 'x') RETURNING id`).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestComprobanteDeHasta5MBSeAcepta(t *testing.T) {
	e := nuevoEntorno(t)
	id := e.gastoRegistrado(t)
	rec := e.subirComprobante(t, id, "boleta.png", pngDe(cincoMB-1024)) // con el resto del formulario cabe en 5 MB
	if rec.Code != http.StatusCreated {
		t.Fatalf("código %d: %s", rec.Code, rec.Body)
	}
}

func TestComprobanteDeMasDe5MBSeRechaza(t *testing.T) {
	e := nuevoEntorno(t)
	id := e.gastoRegistrado(t)
	rec := e.subirComprobante(t, id, "boleta.png", pngDe(cincoMB+1))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("código %d: %s", rec.Code, rec.Body)
	}
	var estado string
	_ = e.pool.QueryRow(context.Background(), `SELECT estado FROM gasto WHERE id = $1`, id).Scan(&estado)
	if estado != "Registrado" {
		t.Fatalf("un archivo rechazado no debe cambiar el gasto: %s", estado)
	}
}
