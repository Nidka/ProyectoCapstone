package handlers_test

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"gastos-service/internal/ocr"
)

// capturarLog desvía el log estándar (el que registro.Configurar manda al archivo de errores).
func capturarLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var salida bytes.Buffer
	log.SetOutput(&salida)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	return &salida
}

// RNF-14: todo 500 deja su causa técnica en el log, no solo los de maestros, usuarios y reportes.
func TestLosErrores500QuedanEnElLogConSuCausa(t *testing.T) {
	casos := []struct {
		nombre string
		// romper deja la base lista para fallar y devuelve la petición que debe dar 500.
		romper func(t *testing.T, e *entorno) *httptest.ResponseRecorder
		ruta   string // lo que debe aparecer en la línea del log (método y ruta)
	}{
		{"registrar gasto", func(t *testing.T, e *entorno) *httptest.ResponseRecorder {
			g, _ := e.comprobantePendiente(t)
			cliente := e.clienteDe(t, g)
			e.quitarTabla(t, "gasto")
			return e.enviar(t, "POST", "/gastos", "logistica", map[string]any{"viaje_id": 1, "cliente_id": cliente, "monto": 10})
		}, "POST /gastos"},
		{"listar gastos", func(t *testing.T, e *entorno) *httptest.ResponseRecorder {
			e.quitarTabla(t, "gasto")
			return e.enviar(t, "GET", "/gastos", "logistica", nil)
		}, "GET /gastos"},
		{"adjuntar comprobante", func(t *testing.T, e *entorno) *httptest.ResponseRecorder {
			g, _ := e.comprobantePendiente(t)
			e.quitarTabla(t, "comprobante")
			return e.subirComprobante(t, g, "boleta.png", pngDe(2048))
		}, "POST /gastos/"},
		{"aprobar comprobante", func(t *testing.T, e *entorno) *httptest.ResponseRecorder {
			_, c := e.comprobantePendiente(t)
			e.quitarTabla(t, "comprobante")
			return e.enviar(t, "POST", fmt.Sprintf("/comprobantes/%d/aprobar", c), "logistica", map[string]any{"checklist": completo()})
		}, "POST /comprobantes/"},
		{"rechazar comprobante", func(t *testing.T, e *entorno) *httptest.ResponseRecorder {
			_, c := e.comprobantePendiente(t)
			e.quitarTabla(t, "comprobante")
			return e.enviar(t, "POST", fmt.Sprintf("/comprobantes/%d/rechazar", c), "logistica", map[string]any{"motivo_rechazo": "borroso"})
		}, "POST /comprobantes/"},
		{"configurar checklist del cliente", func(t *testing.T, e *entorno) *httptest.ResponseRecorder {
			g, _ := e.comprobantePendiente(t)
			cliente := e.clienteDe(t, g)
			e.quitarTabla(t, "criterio_checklist")
			cuerpo := map[string]any{"criterios": []criterioJSON{{Clave: "legible", Texto: "Se lee completo"}}}
			return e.enviar(t, "PUT", fmt.Sprintf("/clientes/%d/checklist", cliente), "logistica", cuerpo)
		}, "PUT /clientes/"},
		{"leer checklist del cliente", func(t *testing.T, e *entorno) *httptest.ResponseRecorder {
			g, _ := e.comprobantePendiente(t)
			cliente := e.clienteDe(t, g)
			e.quitarTabla(t, "criterio_checklist")
			return e.enviar(t, "GET", fmt.Sprintf("/clientes/%d/checklist", cliente), "logistica", nil)
		}, "GET /clientes/"},
		{"listar notificaciones", func(t *testing.T, e *entorno) *httptest.ResponseRecorder {
			e.quitarTabla(t, "notificacion")
			return e.enviar(t, "GET", "/notificaciones", "logistica", nil)
		}, "GET /notificaciones"},
		{"marcar notificación leída", func(t *testing.T, e *entorno) *httptest.ResponseRecorder {
			e.quitarTabla(t, "notificacion")
			return e.enviar(t, "POST", "/notificaciones/1/leida", "logistica", nil)
		}, "POST /notificaciones/1/leida"},
		{"historial del gasto", func(t *testing.T, e *entorno) *httptest.ResponseRecorder {
			g, _ := e.comprobantePendiente(t)
			e.quitarTabla(t, "bitacora")
			return e.enviar(t, "GET", fmt.Sprintf("/gastos/%d/historial", g), "logistica", nil)
		}, "GET /gastos/"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			e := nuevoEntorno(t)
			salida := capturarLog(t)
			rec := c.romper(t, e)
			if rec.Code != http.StatusInternalServerError {
				t.Fatalf("código %d, quiero 500: %s", rec.Code, rec.Body)
			}
			if strings.Contains(rec.Body.String(), "SQLSTATE") {
				t.Errorf("el cliente no debe ver la causa técnica: %s", rec.Body)
			}
			linea := salida.String()
			if !strings.Contains(linea, c.ruta) || !strings.Contains(linea, "SQLSTATE") {
				t.Fatalf("el log debía traer la ruta %q y la causa de la base, trae: %q", c.ruta, linea)
			}
		})
	}
}

func TestElFalloDeLaLecturaOCRQuedaEnElLog(t *testing.T) {
	e, _ := nuevoEntornoConLector(t, ocr.Lectura{Estado: ocr.Leido, Texto: textoBoleta})
	_, id := e.comprobantePendiente(t)
	e.quitarTabla(t, "lectura_comprobante")
	salida := capturarLog(t)
	codigo, _, cuerpo := e.pedirLectura(t, id)
	if codigo != http.StatusInternalServerError {
		t.Fatalf("código %d, quiero 500: %s", codigo, cuerpo)
	}
	if linea := salida.String(); !strings.Contains(linea, "/lectura") || !strings.Contains(linea, "SQLSTATE") {
		t.Fatalf("el log debía traer la ruta y la causa: %q", linea)
	}
}

func TestElFalloDeAutenticacionQuedaEnElLog(t *testing.T) {
	e := nuevoEntorno(t)
	salida := capturarLog(t)
	e.quitarTabla(t, "usuario")
	// El middleware no puede validar al usuario del token.
	if rec := e.enviar(t, "GET", "/gastos", "logistica", nil); rec.Code != http.StatusInternalServerError {
		t.Fatalf("middleware: código %d, quiero 500", rec.Code)
	}
	if linea := salida.String(); !strings.Contains(linea, "SQLSTATE") {
		t.Fatalf("el fallo del middleware no quedó en el log: %q", linea)
	}
	salida.Reset()
	// El inicio de sesión no puede buscar al usuario.
	if _, rec := e.iniciarSesion("ana", "cualquiera"); rec.Code != http.StatusInternalServerError {
		t.Fatalf("login: código %d, quiero 500", rec.Code)
	}
	if linea := salida.String(); !strings.Contains(linea, "SQLSTATE") {
		t.Fatalf("el fallo del login no quedó en el log: %q", linea)
	}
}

// quitarTabla borra una tabla del esquema temporal para provocar un fallo de la base.
func (e *entorno) quitarTabla(t *testing.T, tabla string) {
	t.Helper()
	if _, err := e.pool.Exec(context.Background(), `DROP TABLE `+tabla+` CASCADE`); err != nil {
		t.Fatal(err)
	}
}
