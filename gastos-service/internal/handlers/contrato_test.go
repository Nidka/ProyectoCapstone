package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"gastos-service/internal/auth"
	"gastos-service/internal/handlers"
	"gastos-service/internal/pruebas"
)

type pasoContrato struct {
	Metodo    string            `json:"metodo"`
	Ruta      string            `json:"ruta"`
	Cuerpo    any               `json:"cuerpo"`
	Token     string            `json:"token"`
	Cabeceras map[string]string `json:"cabeceras"`
	Guarda    string            `json:"guarda"`
}

type respuestaContrato struct {
	Estado int `json:"estado"`
	Cuerpo any `json:"cuerpo"`
}

func leerJSON(t *testing.T, nombre string, destino any) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "crud", "contrato", nombre))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, destino); err != nil {
		t.Fatal(err)
	}
}

// Diferencias deliberadas respecto de la API de Python, que el contrato debe reflejar:
//   - Toda ruta salvo /health y /auth/login exige Bearer; los pasos que el escenario manda sin
//     token (salvo los de /auth/login y /auth/me, que prueban justo eso) van con el del admin.
//   - /auth/register ya no es público (solo admin) y el rol «operador» se llama «logistica».
//   - Los usuarios traen debe_cambiar_clave (Python no lo conoce): se quita de la respuesta de Go
//     antes de comparar. El admin sembrado ya cambió su clave para que el escenario pueda usar la API.
const (
	rolPython = "operador"
	rolGo     = "logistica"
)

// aRolGo cambia «operador» por «logistica» en los cuerpos del escenario y de las respuestas de Python.
func aRolGo(v any) any {
	switch x := v.(type) {
	case map[string]any:
		c := make(map[string]any, len(x))
		for k, e := range x {
			c[k] = aRolGo(e)
		}
		return c
	case []any:
		c := make([]any, len(x))
		for i, e := range x {
			c[i] = aRolGo(e)
		}
		return c
	case string:
		if x == rolPython {
			return rolGo
		}
	}
	return v
}

// TestContratoConLaApiDePython repite sobre Go el escenario capturado de la API de Python
// (internal/crud/contrato) y exige el mismo código y la misma forma de respuesta en cada paso,
// con las diferencias deliberadas de arriba.
func TestContratoConLaApiDePython(t *testing.T) {
	pool := pruebas.NuevoEsquemaCompleto(t)
	h := handlers.New(pool, configDePrueba(t.TempDir()))
	if err := h.Auth.SembrarAdmin(t.Context()); err != nil {
		t.Fatal(err)
	}
	servidor := h.Servidor()
	if _, err := pool.Exec(t.Context(), `UPDATE usuario SET debe_cambiar_clave = FALSE WHERE username = 'admin'`); err != nil {
		t.Fatal(err)
	}
	admin, err := h.Auth.PorUsername(t.Context(), "admin")
	if err != nil || admin == nil {
		t.Fatalf("admin sembrado: %v, %v", admin, err)
	}
	tokenAdmin := tokenDe(t, auth.Usuario{ID: admin.ID, Username: admin.Username, Rol: admin.Rol})

	var pasos []pasoContrato
	var esperadas []respuestaContrato
	leerJSON(t, "escenario.json", &pasos)
	leerJSON(t, "python.json", &esperadas)
	for i := range esperadas {
		esperadas[i].Cuerpo = aRolGo(esperadas[i].Cuerpo)
	}
	if len(pasos) != len(esperadas) {
		t.Fatalf("escenario (%d) y respuestas de Python (%d) no coinciden", len(pasos), len(esperadas))
	}

	ultimoID := map[string]any{}
	tokens := map[string]string{}
	fallos := 0
	for i, p := range pasos {
		ruta := p.Ruta
		if strings.Contains(ruta, "{id}") {
			id, _ := ultimoID[strings.Split(ruta, "/{id}")[0]].(float64)
			ruta = strings.Replace(ruta, "{id}", strconv.FormatFloat(id, 'f', -1, 64), 1)
		}
		var cuerpo *bytes.Reader
		if p.Cuerpo != nil {
			b, _ := json.Marshal(aRolGo(p.Cuerpo))
			cuerpo = bytes.NewReader(b)
		} else {
			cuerpo = bytes.NewReader(nil)
		}
		req := httptest.NewRequest(p.Metodo, ruta, cuerpo)
		if p.Cuerpo != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if p.Token != "" {
			tok := p.Token
			if v, ok := tokens[p.Token]; ok {
				tok = v
			}
			req.Header.Set("Authorization", "Bearer "+tok)
		} else if p.Cabeceras == nil && p.Ruta != "/api/v1/auth/login" && p.Ruta != "/api/v1/auth/me" {
			req.Header.Set("Authorization", "Bearer "+tokenAdmin)
		}
		for k, v := range p.Cabeceras {
			req.Header.Set(k, v)
		}
		rec := httptest.NewRecorder()
		servidor.ServeHTTP(rec, req)

		var obtenido any
		if err := json.Unmarshal(rec.Body.Bytes(), &obtenido); err != nil {
			obtenido = rec.Body.String()
		}
		if p.Guarda != "" && rec.Code == http.StatusCreated {
			ultimoID[p.Ruta] = obtenido.(map[string]any)[p.Guarda]
		} else if p.Guarda != "" {
			ultimoID[p.Ruta] = 0
		}
		if p.Ruta == "/api/v1/auth/login" && rec.Code == http.StatusOK {
			if m, ok := p.Cuerpo.(map[string]any); ok && m["username"] == "admin" {
				tokens["@login-admin"] = obtenido.(map[string]any)["access_token"].(string)
			}
		}

		if msg := comparar(esperadas[i], rec.Code, obtenido); msg != "" {
			fallos++
			if fallos <= 25 {
				t.Errorf("paso %d %s %s: %s", i, p.Metodo, ruta, msg)
			}
		}
	}
	if fallos > 25 {
		t.Errorf("... y %d pasos más con diferencias", fallos-25)
	}
}

// comparar devuelve la diferencia entre la respuesta de Python y la de Go, o "" si coinciden.
// Un 500 solo se compara por su código (el cuerpo de Python es una traza de depuración); en un
// 422 se comparan el tipo y la ubicación de cada error; el token cambia en cada emisión.
func comparar(esperada respuestaContrato, estado int, obtenido any) string {
	if esperada.Estado != estado {
		return fmt.Sprintf("código %d, Python daba %d (%.200v)", estado, esperada.Estado, obtenido)
	}
	switch estado {
	case http.StatusInternalServerError:
		return ""
	case http.StatusUnprocessableEntity:
		a, b := resumen422(esperada.Cuerpo), resumen422(obtenido)
		if !reflect.DeepEqual(a, b) {
			return fmt.Sprintf("errores 422 distintos:\n  Python: %v\n  Go:     %v", a, b)
		}
		return ""
	}
	esp, obt := normalizar(esperada.Cuerpo), normalizar(obtenido)
	if !reflect.DeepEqual(esp, obt) {
		return fmt.Sprintf("cuerpo distinto:\n  Python: %.300v\n  Go:     %.300v", esp, obt)
	}
	return ""
}

func normalizar(v any) any {
	switch x := v.(type) {
	case map[string]any:
		c := make(map[string]any, len(x))
		for k, e := range x {
			if k == "debe_cambiar_clave" {
				continue
			}
			c[k] = normalizar(e)
		}
		if _, tiene := c["access_token"]; tiene {
			c["access_token"] = "<token>"
		}
		return c
	case []any:
		c := make([]any, len(x))
		for i, e := range x {
			c[i] = normalizar(e)
		}
		return c
	}
	return v
}

func resumen422(v any) []string {
	m, _ := v.(map[string]any)
	detalle, _ := m["detail"].([]any)
	var out []string
	for _, d := range detalle {
		e, _ := d.(map[string]any)
		out = append(out, fmt.Sprintf("%v %v", e["type"], e["loc"]))
	}
	return out
}
