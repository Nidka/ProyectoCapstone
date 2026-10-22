package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gastos-service/internal/reportes"
)

var estadosGasto = map[string]bool{"Registrado": true, "Pagado": true, "Sustentado": true}

// filtroGastos son los filtros de la consulta de gastos (HU-12): por viaje, por cliente y por
// estado. Un valor en cero o vacío significa «sin filtro».
type filtroGastos struct {
	ViajeID   int
	ClienteID int
	Estado    string
}

// leerFiltroGastos lee ?viaje_id=&cliente_id=&estado=; si alguno no es válido devuelve el mensaje.
func leerFiltroGastos(r *http.Request) (filtroGastos, string) {
	var f filtroGastos
	q := r.URL.Query()
	for _, campo := range []struct {
		nombre  string
		destino *int
	}{{"viaje_id", &f.ViajeID}, {"cliente_id", &f.ClienteID}} {
		txt := q.Get(campo.nombre)
		if txt == "" {
			continue
		}
		n, err := strconv.Atoi(txt)
		if err != nil || n <= 0 {
			return f, campo.nombre + " debe ser un número entero positivo"
		}
		*campo.destino = n
	}
	if estado := q.Get("estado"); estado != "" {
		if !estadosGasto[estado] {
			return f, "estado debe ser Registrado, Pagado o Sustentado"
		}
		f.Estado = estado
	}
	return f, ""
}

// condiciones devuelve las condiciones SQL del filtro sobre la tabla gasto (alias g) y sus
// parámetros, numerados a partir de desde+1.
func (f filtroGastos) condiciones(desde int) ([]string, []any) {
	var cond []string
	var args []any
	agregar := func(col string, v any) {
		args = append(args, v)
		cond = append(cond, fmt.Sprintf("g.%s = $%d", col, desde+len(args)))
	}
	if f.ViajeID > 0 {
		agregar("viaje_id", f.ViajeID)
	}
	if f.ClienteID > 0 {
		agregar("cliente_id", f.ClienteID)
	}
	if f.Estado != "" {
		agregar("estado", f.Estado)
	}
	return cond, args
}

// descripcion resume los filtros activos para mostrarlos en un documento.
func (f filtroGastos) descripcion() string {
	var partes []string
	if f.ViajeID > 0 {
		partes = append(partes, "Viaje N° "+strconv.Itoa(f.ViajeID))
	}
	if f.ClienteID > 0 {
		partes = append(partes, "Cliente N° "+strconv.Itoa(f.ClienteID))
	}
	if f.Estado != "" {
		partes = append(partes, "Estado: "+f.Estado)
	}
	if len(partes) == 0 {
		return "Sin filtros"
	}
	return strings.Join(partes, " · ")
}

// leerRango lee ?desde=&hasta= (AAAA-MM-DD), ambos opcionales; si no sirven devuelve el mensaje.
func leerRango(r *http.Request) (desde, hasta *time.Time, mensaje string) {
	leer := func(nombre string) (*time.Time, bool) {
		txt := r.URL.Query().Get(nombre)
		if txt == "" {
			return nil, true
		}
		t, err := time.Parse("2006-01-02", txt)
		if err != nil {
			return nil, false
		}
		return &t, true
	}
	var ok bool
	if desde, ok = leer("desde"); !ok {
		return nil, nil, "desde debe tener el formato AAAA-MM-DD"
	}
	if hasta, ok = leer("hasta"); !ok {
		return nil, nil, "hasta debe tener el formato AAAA-MM-DD"
	}
	if desde != nil && hasta != nil && desde.After(*hasta) {
		return nil, nil, "la fecha inicial no puede ser posterior a la final"
	}
	return desde, hasta, ""
}

// exportarGastos descarga en Excel (HU-16) los gastos que cumplen los mismos filtros de la consulta
// de gastos, más un rango de fechas opcional.
func (h *Handler) exportarGastos(w http.ResponseWriter, r *http.Request) {
	filtro, mensaje := leerFiltroGastos(r)
	if mensaje != "" {
		writeError(w, http.StatusBadRequest, mensaje)
		return
	}
	desde, hasta, mensaje := leerRango(r)
	if mensaje != "" {
		writeError(w, http.StatusBadRequest, mensaje)
		return
	}
	filas, err := reportes.ConsultarDetalle(r.Context(), h.DB, reportes.FiltroDetalle{
		Desde: desde, Hasta: hasta, ViajeID: filtro.ViajeID, ClienteID: filtro.ClienteID, Estado: filtro.Estado,
	})
	if err != nil {
		errorDeReporte(w, err)
		return
	}
	descripcion := filtro.descripcion()
	if desde != nil || hasta != nil {
		rango := "Período: "
		if desde != nil {
			rango += desde.Format("02/01/2006")
		}
		rango += " – "
		if hasta != nil {
			rango += hasta.Format("02/01/2006")
		}
		if descripcion == "Sin filtros" {
			descripcion = rango
		} else {
			descripcion += " · " + rango
		}
	}
	ahora := time.Now()
	libro, err := reportes.ExcelGastos(filas, descripcion, ahora)
	if err != nil {
		errorDeReporte(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="gastos-`+ahora.In(reportes.Lima).Format("2006-01-02")+`.xlsx"`)
	w.Header().Set("Content-Length", strconv.Itoa(len(libro)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(libro)
}
