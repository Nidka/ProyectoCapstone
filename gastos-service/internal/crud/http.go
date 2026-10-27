package crud

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
)

// Registrar expone los recursos bajo el prefijo (p. ej. /api/v1): listar, obtener, crear,
// actualizar, borrar y la lista avanzada de cada uno.
func Registrar(mux *http.ServeMux, m *Motor, prefijo string, recursos []Recurso) {
	for i := range recursos {
		r := &recursos[i]
		base := prefijo + "/" + r.Ruta
		mux.HandleFunc("GET "+base, func(w http.ResponseWriter, req *http.Request) { m.listar(w, req, r) })
		mux.HandleFunc("GET "+base+"/{id}", func(w http.ResponseWriter, req *http.Request) { m.obtener(w, req, r) })
		mux.HandleFunc("POST "+base, func(w http.ResponseWriter, req *http.Request) { m.crear(w, req, r) })
		mux.HandleFunc("PUT "+base+"/{id}", func(w http.ResponseWriter, req *http.Request) { m.actualizar(w, req, r) })
		mux.HandleFunc("DELETE "+base+"/{id}", func(w http.ResponseWriter, req *http.Request) { m.borrar(w, req, r) })
		mux.HandleFunc("GET "+prefijo+"/advanced/"+r.Ruta, func(w http.ResponseWriter, req *http.Request) { m.avanzada(w, req, r) })
	}
}

// EscribirJSON responde con JSON sin escapar <, > ni &, como FastAPI.
func EscribirJSON(w http.ResponseWriter, estado int, v any) {
	cuerpo, err := jsonSinEscape(v)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(estado)
	_, _ = w.Write(append(cuerpo, '\n'))
}

// Detalle responde {"detail": "<mensaje>"}, el formato de error de /api/v1.
func Detalle(w http.ResponseWriter, estado int, mensaje string) {
	EscribirJSON(w, estado, map[string]string{"detail": mensaje})
}

func validacion(w http.ResponseWriter, errs []ErrorCampo) {
	EscribirJSON(w, http.StatusUnprocessableEntity, map[string]any{"detail": errs})
}

// falloInterno registra la causa y responde como un error no controlado de FastAPI.
func falloInterno(w http.ResponseWriter, err error) {
	log.Printf("crud: %v", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

// entero lee un parámetro entero con límites opcionales (min/max con ok=false si no aplican).
func entero(origen, nombre, txt string, def int, min, max *int) (int, *ErrorCampo) {
	if txt == "" {
		return def, nil
	}
	n, err := strconv.Atoi(txt)
	if err != nil {
		return 0, &ErrorCampo{Tipo: "int_parsing", Loc: []any{origen, nombre}, Msg: "Input should be a valid integer, unable to parse string as an integer", Input: txt}
	}
	if min != nil && n < *min {
		return 0, &ErrorCampo{Tipo: "greater_than_equal", Loc: []any{origen, nombre}, Msg: "Input should be greater than or equal to " + strconv.Itoa(*min), Input: txt, Ctx: map[string]any{"ge": *min}}
	}
	if max != nil && n > *max {
		return 0, &ErrorCampo{Tipo: "less_than_equal", Loc: []any{origen, nombre}, Msg: "Input should be less than or equal to " + strconv.Itoa(*max), Input: txt, Ctx: map[string]any{"le": *max}}
	}
	return n, nil
}

func paginacion(req *http.Request) (limit, offset int, errs []ErrorCampo) {
	uno, cero, doscientos := 1, 0, 200
	q := req.URL.Query()
	limit, e1 := entero("query", "limit", q.Get("limit"), 50, &uno, &doscientos)
	offset, e2 := entero("query", "offset", q.Get("offset"), 0, &cero, nil)
	for _, e := range []*ErrorCampo{e1, e2} {
		if e != nil {
			errs = append(errs, *e)
		}
	}
	return limit, offset, errs
}

// claveRuta lee {id}; nombre es el nombre del parámetro en el error (la clave en GET, item_id en PUT/DELETE).
func claveRuta(req *http.Request, nombre string) (int64, *ErrorCampo) {
	txt := req.PathValue("id")
	n, err := strconv.ParseInt(txt, 10, 64)
	if err != nil {
		return 0, &ErrorCampo{Tipo: "int_parsing", Loc: []any{"path", nombre}, Msg: "Input should be a valid integer, unable to parse string as an integer", Input: txt}
	}
	return n, nil
}

func (m *Motor) listar(w http.ResponseWriter, req *http.Request, r *Recurso) {
	limit, offset, errs := paginacion(req)
	if len(errs) > 0 {
		validacion(w, errs)
		return
	}
	filas, err := m.Listar(req.Context(), r, limit, offset)
	if err != nil {
		falloInterno(w, err)
		return
	}
	EscribirJSON(w, http.StatusOK, filas)
}

func (m *Motor) obtener(w http.ResponseWriter, req *http.Request, r *Recurso) {
	id, e := claveRuta(req, r.PK)
	if e != nil {
		validacion(w, []ErrorCampo{*e})
		return
	}
	fila, ok, err := m.Obtener(req.Context(), r, id)
	if err != nil {
		falloInterno(w, err)
		return
	}
	if !ok {
		Detalle(w, http.StatusNotFound, r.NoEncontrado)
		return
	}
	EscribirJSON(w, http.StatusOK, fila)
}

// leerObjeto lee el cuerpo como un objeto JSON; si no lo es, responde 422 y devuelve false.
func leerObjeto(w http.ResponseWriter, req *http.Request) (map[string]any, bool) {
	crudo, err := io.ReadAll(io.LimitReader(req.Body, 1<<20))
	if err != nil {
		falloInterno(w, err)
		return nil, false
	}
	var v any
	if err := json.Unmarshal(crudo, &v); err != nil {
		validacion(w, []ErrorCampo{{Tipo: "json_invalid", Loc: []any{"body", 0}, Msg: "JSON decode error", Input: map[string]any{}}})
		return nil, false
	}
	obj, ok := v.(map[string]any)
	if !ok {
		validacion(w, []ErrorCampo{{Tipo: "model_attributes_type", Loc: []any{"body"}, Msg: "Input should be a valid dictionary or object to extract fields from", Input: v}})
		return nil, false
	}
	return obj, true
}

func (m *Motor) crear(w http.ResponseWriter, req *http.Request, r *Recurso) {
	cuerpo, ok := leerObjeto(w, req)
	if !ok {
		return
	}
	valores, errs := ValidarCreacion(r, cuerpo)
	if len(errs) > 0 {
		validacion(w, errs)
		return
	}
	if r.Unico != nil {
		existe, err := m.ExisteUnico(req.Context(), r, valores[r.Unico.Campo])
		if err != nil {
			falloInterno(w, err)
			return
		}
		if existe {
			Detalle(w, http.StatusConflict, r.Unico.Mensaje)
			return
		}
	}
	fila, err := m.Crear(req.Context(), r, valores)
	if err != nil {
		falloInterno(w, err)
		return
	}
	EscribirJSON(w, http.StatusCreated, fila)
}

func (m *Motor) actualizar(w http.ResponseWriter, req *http.Request, r *Recurso) {
	id, e := claveRuta(req, "item_id")
	cuerpo, ok := leerObjeto(w, req)
	if !ok {
		return
	}
	var errs []ErrorCampo
	if e != nil {
		errs = append(errs, *e)
	}
	valores, errsCuerpo := ValidarCambios(r, cuerpo)
	errs = append(errs, errsCuerpo...)
	if len(errs) > 0 {
		validacion(w, errs)
		return
	}
	fila, existe, err := m.Actualizar(req.Context(), r, id, valores)
	if err != nil {
		falloInterno(w, err)
		return
	}
	if !existe {
		Detalle(w, http.StatusNotFound, "Recurso no encontrado")
		return
	}
	EscribirJSON(w, http.StatusOK, fila)
}

func (m *Motor) borrar(w http.ResponseWriter, req *http.Request, r *Recurso) {
	id, e := claveRuta(req, "item_id")
	if e != nil {
		validacion(w, []ErrorCampo{*e})
		return
	}
	existia, err := m.Borrar(req.Context(), r, id)
	if err != nil {
		falloInterno(w, err)
		return
	}
	if !existia {
		Detalle(w, http.StatusNotFound, "Recurso no encontrado")
		return
	}
	EscribirJSON(w, http.StatusOK, borrado{r: r, id: id})
}

// borrado es la respuesta de DELETE: {"deleted": true, "<clave>": id}.
type borrado struct {
	r  *Recurso
	id int64
}

func (b borrado) MarshalJSON() ([]byte, error) {
	clave, _ := jsonSinEscape(b.r.PK)
	return []byte(`{"deleted":true,` + string(clave) + `:` + strconv.FormatInt(b.id, 10) + `}`), nil
}

func (m *Motor) avanzada(w http.ResponseWriter, req *http.Request, r *Recurso) {
	limit, offset, errs := paginacion(req)
	q := req.URL.Query()
	dir := q.Get("order_dir")
	if dir == "" {
		dir = "asc"
	}
	if dir != "asc" && dir != "desc" {
		errs = append(errs, ErrorCampo{Tipo: "string_pattern_mismatch", Loc: []any{"query", "order_dir"}, Msg: "String should match pattern '^(asc|desc)$'", Input: dir, Ctx: map[string]any{"pattern": "^(asc|desc)$"}})
	}
	if len(errs) > 0 {
		validacion(w, errs)
		return
	}
	filas, err := m.Avanzada(req.Context(), r, ParamsAvanzada{
		Limit: limit, Offset: offset, Search: q.Get("search"), SearchFields: q.Get("search_fields"),
		OrderBy: q.Get("order_by"), OrderDir: dir, Filters: q.Get("filters"),
	})
	if err != nil {
		falloInterno(w, err)
		return
	}
	EscribirJSON(w, http.StatusOK, filas)
}
