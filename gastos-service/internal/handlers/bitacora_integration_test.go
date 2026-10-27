package handlers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"gastos-service/internal/handlers"
	"gastos-service/internal/pruebas"
)

// asiento es una fila de la bitácora tal como la guarda la base.
type asiento struct {
	IDUsuario *int
	Usuario   string
	Accion    string
	Entidad   string
	IDEntidad *int64
	Detalle   map[string]any
}

// sub lee un valor anidado del detalle, por ejemplo sub("despues", "estado").
func (a asiento) sub(ruta ...string) any {
	var actual any = a.Detalle
	for _, k := range ruta {
		m, ok := actual.(map[string]any)
		if !ok {
			return nil
		}
		actual = m[k]
	}
	return actual
}

// entornoCompleto es el servicio sobre el esquema completo de db.sql (viaje 1 y cliente 1 existen).
func entornoCompleto(t *testing.T) *entorno {
	t.Helper()
	pool := pruebas.NuevoEsquemaCompleto(t)
	return armarEntorno(t, pool, handlers.New(pool, configDePrueba(t.TempDir())))
}

func (e *entorno) asientos(t *testing.T) []asiento {
	t.Helper()
	filas, err := e.pool.Query(context.Background(), `SELECT id_usuario, usuario, accion, entidad, id_entidad, detalle FROM bitacora ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	defer filas.Close()
	var out []asiento
	for filas.Next() {
		var a asiento
		var detalle []byte
		if err := filas.Scan(&a.IDUsuario, &a.Usuario, &a.Accion, &a.Entidad, &a.IDEntidad, &detalle); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(detalle, &a.Detalle); err != nil {
			t.Fatal(err)
		}
		out = append(out, a)
	}
	return out
}

// unaNueva exige que, desde antes, la bitácora haya ganado exactamente una entrada y la devuelve.
func (e *entorno) unaNueva(t *testing.T, antes int, accion string) asiento {
	t.Helper()
	todos := e.asientos(t)
	if len(todos) != antes+1 {
		acciones := []string{}
		for _, a := range todos {
			acciones = append(acciones, a.Accion)
		}
		t.Fatalf("la bitácora tiene %d entradas (%v), quiero %d", len(todos), acciones, antes+1)
	}
	a := todos[antes]
	if a.Accion != accion {
		t.Fatalf("acción = %q, quiero %q", a.Accion, accion)
	}
	return a
}

func (e *entorno) total(t *testing.T) int { return len(e.asientos(t)) }

func (e *entorno) idDeUsuario(t *testing.T, username string) int { return e.idDe(t, username) }

// registrarGasto crea un gasto por la API (viaje 1, cliente 1) y devuelve su id.
func (e *entorno) registrarGasto(t *testing.T) int {
	t.Helper()
	rec := pedir(e.mux, "POST", "/gastos", e.tokenLogistica, map[string]any{"viaje_id": 1, "cliente_id": 1, "monto": 85.5})
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /gastos = %d %s", rec.Code, rec.Body)
	}
	var g struct {
		ID int `json:"id"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &g)
	return g.ID
}

// adjuntar sube un comprobante por la API y devuelve su id.
func (e *entorno) adjuntar(t *testing.T, gastoID int) int {
	t.Helper()
	rec := e.subirComprobante(t, gastoID, "boleta.png", pngDe(2048))
	if rec.Code != http.StatusCreated {
		t.Fatalf("adjuntar = %d %s", rec.Code, rec.Body)
	}
	var c struct {
		ID int `json:"id"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &c)
	return c.ID
}

func TestRegistrarGastoGeneraSuEntrada(t *testing.T) {
	e := entornoCompleto(t)
	// Lo rechazado no deja rastro: ni el 400 de validación ni el 403 del admin.
	pedir(e.mux, "POST", "/gastos", e.tokenLogistica, map[string]any{"viaje_id": 1, "cliente_id": 1, "monto": 0})
	pedir(e.mux, "POST", "/gastos", e.tokenAdmin, map[string]any{"viaje_id": 1, "cliente_id": 1, "monto": 10})
	if n := e.total(t); n != 0 {
		t.Fatalf("las peticiones rechazadas dejaron %d entradas", n)
	}

	id := e.registrarGasto(t)
	a := e.unaNueva(t, 0, "gasto.registrar")
	if a.Usuario != "Ana Quispe" || a.IDUsuario == nil || *a.IDUsuario != e.idDeUsuario(t, "ana") {
		t.Errorf("autor = %q %v", a.Usuario, a.IDUsuario)
	}
	if a.Entidad != "gasto" || a.IDEntidad == nil || *a.IDEntidad != int64(id) {
		t.Errorf("entidad = %q %v, quiero gasto %d", a.Entidad, a.IDEntidad, id)
	}
	if a.sub("despues", "monto") != 85.5 || a.sub("despues", "estado") != "Registrado" || a.sub("despues", "cliente_id") != 1.0 || a.sub("despues", "viaje_id") != 1.0 {
		t.Errorf("detalle = %v", a.Detalle)
	}
}

func TestAdjuntarComprobanteGeneraSuEntrada(t *testing.T) {
	e := entornoCompleto(t)
	gasto := e.registrarGasto(t)
	antes := e.total(t)
	comprobante := e.adjuntar(t, gasto)
	a := e.unaNueva(t, antes, "comprobante.adjuntar")
	if a.Usuario != "Ana Quispe" || a.Entidad != "comprobante" || a.IDEntidad == nil || *a.IDEntidad != int64(comprobante) {
		t.Errorf("asiento = %+v", a)
	}
	if a.sub("gasto_id") != float64(gasto) || a.sub("antes", "gasto_estado") != "Registrado" ||
		a.sub("despues", "gasto_estado") != "Pagado" || a.sub("despues", "estado") != "Pendiente" || a.sub("despues", "tipo_archivo") != "imagen" {
		t.Errorf("detalle = %v", a.Detalle)
	}
	// Un archivo que no es imagen ni PDF se rechaza sin dejar rastro.
	antes = e.total(t)
	if rec := e.subirComprobante(t, gasto, "notas.txt", []byte("hola")); rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("txt = %d", rec.Code)
	}
	if e.total(t) != antes {
		t.Error("un archivo rechazado dejó una entrada")
	}
}

func TestAprobarGeneraSuEntradaConChecklist(t *testing.T) {
	e := entornoCompleto(t)
	gasto := e.registrarGasto(t)
	comprobante := e.adjuntar(t, gasto)

	// Incompleto: 422 y sin entrada.
	antes := e.total(t)
	if rec := pedir(e.mux, "POST", fmt.Sprintf("/comprobantes/%d/aprobar", comprobante), e.tokenLogistica, map[string]any{"checklist": map[string]bool{"legible": true}}); rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("aprobar incompleto = %d", rec.Code)
	}
	if e.total(t) != antes {
		t.Fatal("una aprobación rechazada dejó una entrada")
	}

	if rec := pedir(e.mux, "POST", fmt.Sprintf("/comprobantes/%d/aprobar", comprobante), e.tokenLogistica, map[string]any{"checklist": completo()}); rec.Code != http.StatusOK {
		t.Fatalf("aprobar = %d %s", rec.Code, rec.Body)
	}
	a := e.unaNueva(t, antes, "comprobante.aprobar")
	if a.Usuario != "Ana Quispe" || a.Entidad != "comprobante" || *a.IDEntidad != int64(comprobante) || a.sub("gasto_id") != float64(gasto) {
		t.Errorf("asiento = %+v", a)
	}
	if a.sub("antes", "estado") != "Pendiente" || a.sub("antes", "gasto_estado") != "Pagado" ||
		a.sub("despues", "estado") != "Aprobado" || a.sub("despues", "gasto_estado") != "Sustentado" {
		t.Errorf("antes/después = %v", a.Detalle)
	}
	items, _ := a.Detalle["checklist"].([]any)
	if len(items) != len(criteriosEstandar) {
		t.Fatalf("checklist del detalle = %v", a.Detalle["checklist"])
	}
	primero := items[0].(map[string]any)
	if primero["criterio"] != "legible" || primero["cumple"] != true || primero["texto"] == "" {
		t.Errorf("primer criterio = %v", primero)
	}

	// Aprobar de nuevo: 409 y sin entrada.
	antes = e.total(t)
	if rec := pedir(e.mux, "POST", fmt.Sprintf("/comprobantes/%d/aprobar", comprobante), e.tokenLogistica, map[string]any{"checklist": completo()}); rec.Code != http.StatusConflict {
		t.Fatalf("segunda aprobación = %d", rec.Code)
	}
	if e.total(t) != antes {
		t.Error("una aprobación repetida dejó una entrada")
	}
}

func TestRechazarGeneraSuEntradaConMotivoYChecklist(t *testing.T) {
	e := entornoCompleto(t)
	gasto := e.registrarGasto(t)
	comprobante := e.adjuntar(t, gasto)
	antes := e.total(t)

	checklist := completo()
	checklist["legible"] = false
	rec := pedir(e.mux, "POST", fmt.Sprintf("/comprobantes/%d/rechazar", comprobante), e.tokenLogistica, map[string]any{"motivo_rechazo": "Está borroso", "checklist": checklist})
	if rec.Code != http.StatusOK {
		t.Fatalf("rechazar = %d %s", rec.Code, rec.Body)
	}
	a := e.unaNueva(t, antes, "comprobante.rechazar")
	if a.Entidad != "comprobante" || *a.IDEntidad != int64(comprobante) || a.sub("gasto_id") != float64(gasto) {
		t.Errorf("asiento = %+v", a)
	}
	if a.sub("despues", "estado") != "Rechazado" || a.sub("despues", "motivo_rechazo") != "Está borroso" ||
		a.sub("antes", "gasto_estado") != "Pagado" || a.sub("despues", "gasto_estado") != "Registrado" {
		t.Errorf("antes/después = %v", a.Detalle)
	}
	items, _ := a.Detalle["checklist"].([]any)
	incumplidos := 0
	for _, it := range items {
		if it.(map[string]any)["cumple"] == false {
			incumplidos++
		}
	}
	if len(items) != len(criteriosEstandar) || incumplidos != 1 {
		t.Errorf("checklist = %v", a.Detalle["checklist"])
	}

	// Sin motivo: 400 y sin entrada.
	otro := e.adjuntar(t, gasto)
	antes = e.total(t)
	if rec := pedir(e.mux, "POST", fmt.Sprintf("/comprobantes/%d/rechazar", otro), e.tokenLogistica, map[string]any{"motivo_rechazo": " "}); rec.Code != http.StatusBadRequest {
		t.Fatalf("rechazar sin motivo = %d", rec.Code)
	}
	if e.total(t) != antes {
		t.Error("un rechazo inválido dejó una entrada")
	}
}

func TestConfigurarYRestablecerChecklistGeneranSuEntrada(t *testing.T) {
	e := entornoCompleto(t)

	antes := e.total(t)
	rec := pedir(e.mux, "PUT", "/clientes/1/checklist", e.tokenLogistica, map[string]any{"criterios": []criterioJSON{{Texto: "Trae el sello"}, {Texto: "Trae la firma"}}})
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT = %d %s", rec.Code, rec.Body)
	}
	a := e.unaNueva(t, antes, "checklist.configurar")
	if a.Entidad != "cliente" || *a.IDEntidad != 1 || a.Usuario != "Ana Quispe" {
		t.Errorf("asiento = %+v", a)
	}
	if a.sub("antes", "personalizado") != false || a.sub("despues", "personalizado") != true {
		t.Errorf("detalle = %v", a.Detalle)
	}
	if antes, _ := a.sub("antes", "criterios").([]any); len(antes) != len(criteriosEstandar) {
		t.Errorf("antes.criterios = %v (eran los estándar)", a.sub("antes", "criterios"))
	}
	if despues, _ := a.sub("despues", "criterios").([]any); len(despues) != 2 || despues[0].(map[string]any)["texto"] != "Trae el sello" {
		t.Errorf("despues.criterios = %v", a.sub("despues", "criterios"))
	}

	// Una configuración inválida (422) no deja rastro.
	n := e.total(t)
	if rec := pedir(e.mux, "PUT", "/clientes/1/checklist", e.tokenLogistica, map[string]any{"criterios": []criterioJSON{}}); rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("PUT vacío = %d", rec.Code)
	}
	if e.total(t) != n {
		t.Error("un checklist inválido dejó una entrada")
	}

	n = e.total(t)
	if rec := pedir(e.mux, "DELETE", "/clientes/1/checklist", e.tokenLogistica, nil); rec.Code != http.StatusOK {
		t.Fatalf("DELETE = %d %s", rec.Code, rec.Body)
	}
	a = e.unaNueva(t, n, "checklist.restablecer")
	if a.Entidad != "cliente" || *a.IDEntidad != 1 || a.sub("antes", "personalizado") != true || a.sub("despues", "personalizado") != false {
		t.Errorf("restablecer = %+v", a)
	}
	if antes, _ := a.sub("antes", "criterios").([]any); len(antes) != 2 {
		t.Errorf("antes.criterios = %v", a.sub("antes", "criterios"))
	}

	// Restablecer lo que ya es el estándar no cambia nada: no se registra.
	n = e.total(t)
	if rec := pedir(e.mux, "DELETE", "/clientes/1/checklist", e.tokenLogistica, nil); rec.Code != http.StatusOK {
		t.Fatalf("DELETE repetido = %d", rec.Code)
	}
	if e.total(t) != n {
		t.Error("restablecer lo que ya era estándar dejó una entrada")
	}
}

func TestGestionDeUsuariosGeneraSusEntradasSinSecretos(t *testing.T) {
	e := entornoCompleto(t)
	adminID := e.idDeUsuario(t, "admin_prueba")

	// Crear
	n := e.total(t)
	rec := pedir(e.mux, "POST", "/api/v1/usuarios", e.tokenAdmin, cuerpoUsuario("nuevo"))
	if rec.Code != http.StatusCreated {
		t.Fatalf("crear = %d %s", rec.Code, rec.Body)
	}
	nuevo := usuarioDe(t, rec)
	a := e.unaNueva(t, n, "usuario.crear")
	if a.Usuario != "Admin Prueba" || *a.IDUsuario != adminID || a.Entidad != "usuario" || *a.IDEntidad != int64(nuevo.ID) {
		t.Errorf("crear = %+v", a)
	}
	if a.sub("despues", "username") != "nuevo" || a.sub("despues", "rol") != "logistica" || a.sub("despues", "estado") != true || a.sub("despues", "debe_cambiar_clave") != true {
		t.Errorf("detalle de crear = %v", a.Detalle)
	}

	// Un duplicado (409) no deja rastro.
	n = e.total(t)
	if rec := pedir(e.mux, "POST", "/api/v1/usuarios", e.tokenAdmin, cuerpoUsuario("nuevo")); rec.Code != http.StatusConflict {
		t.Fatalf("duplicado = %d", rec.Code)
	}
	if e.total(t) != n {
		t.Error("un alta duplicada dejó una entrada")
	}

	// Editar
	rec = pedir(e.mux, "PUT", fmt.Sprintf("/api/v1/usuarios/%d", nuevo.ID), e.tokenAdmin, map[string]any{"nombre": "Nombre Nuevo", "rol": "admin"})
	if rec.Code != http.StatusOK {
		t.Fatalf("editar = %d %s", rec.Code, rec.Body)
	}
	a = e.unaNueva(t, n, "usuario.editar")
	if *a.IDEntidad != int64(nuevo.ID) || a.sub("antes", "nombre") != "Persona nuevo" || a.sub("despues", "nombre") != "Nombre Nuevo" ||
		a.sub("antes", "rol") != "logistica" || a.sub("despues", "rol") != "admin" {
		t.Errorf("editar = %v", a.Detalle)
	}
	// Restablecer la clave desde la edición se anota sin guardar la clave.
	n = e.total(t)
	if rec := pedir(e.mux, "PUT", fmt.Sprintf("/api/v1/usuarios/%d", nuevo.ID), e.tokenAdmin, map[string]any{"password": "OtraClave123"}); rec.Code != http.StatusOK {
		t.Fatalf("restablecer = %d", rec.Code)
	}
	a = e.unaNueva(t, n, "usuario.editar")
	if a.sub("clave_restablecida") != true {
		t.Errorf("detalle = %v", a.Detalle)
	}

	// Desactivar y activar (y repetirlos no cambia nada: no se registra).
	n = e.total(t)
	if rec := pedir(e.mux, "POST", fmt.Sprintf("/api/v1/usuarios/%d/desactivar", nuevo.ID), e.tokenAdmin, nil); rec.Code != http.StatusOK {
		t.Fatalf("desactivar = %d", rec.Code)
	}
	a = e.unaNueva(t, n, "usuario.desactivar")
	if *a.IDEntidad != int64(nuevo.ID) || a.sub("antes", "estado") != true || a.sub("despues", "estado") != false {
		t.Errorf("desactivar = %v", a.Detalle)
	}
	pedir(e.mux, "POST", fmt.Sprintf("/api/v1/usuarios/%d/desactivar", nuevo.ID), e.tokenAdmin, nil)
	if e.total(t) != n+1 {
		t.Error("desactivar a quien ya estaba inactivo dejó una entrada")
	}
	// Un 409 (el admin se desactiva a sí mismo) tampoco.
	if rec := pedir(e.mux, "POST", fmt.Sprintf("/api/v1/usuarios/%d/desactivar", adminID), e.tokenAdmin, nil); rec.Code != http.StatusConflict {
		t.Fatalf("autodesactivar = %d", rec.Code)
	}
	if e.total(t) != n+1 {
		t.Error("una desactivación rechazada dejó una entrada")
	}

	n = e.total(t)
	if rec := pedir(e.mux, "POST", fmt.Sprintf("/api/v1/usuarios/%d/activar", nuevo.ID), e.tokenAdmin, nil); rec.Code != http.StatusOK {
		t.Fatalf("activar = %d", rec.Code)
	}
	a = e.unaNueva(t, n, "usuario.activar")
	if a.sub("antes", "estado") != false || a.sub("despues", "estado") != true {
		t.Errorf("activar = %v", a.Detalle)
	}
	pedir(e.mux, "POST", fmt.Sprintf("/api/v1/usuarios/%d/activar", nuevo.ID), e.tokenAdmin, nil)
	if e.total(t) != n+1 {
		t.Error("activar a quien ya estaba activo dejó una entrada")
	}

	// Nada de la bitácora guarda contraseñas ni hashes.
	var secretos int
	if err := e.pool.QueryRow(context.Background(), `SELECT count(*) FROM bitacora WHERE detalle::text ~* '(hash|\$2[aby]\$|secreta123|OtraClave123)'`).Scan(&secretos); err != nil || secretos != 0 {
		t.Errorf("entradas con secretos: %d, %v", secretos, err)
	}
}

func TestRegistrarUsuarioPorAuthRegisterGeneraSuEntrada(t *testing.T) {
	e := entornoCompleto(t)
	n := e.total(t)
	rec := pedir(e.mux, "POST", "/api/v1/auth/register", e.tokenAdmin, cuerpoUsuario("viaregister"))
	if rec.Code != http.StatusCreated {
		t.Fatalf("register = %d %s", rec.Code, rec.Body)
	}
	a := e.unaNueva(t, n, "usuario.crear")
	if a.Usuario != "Admin Prueba" || a.sub("despues", "username") != "viaregister" {
		t.Errorf("asiento = %+v", a)
	}
}

func TestCambiarClaveGeneraSuEntrada(t *testing.T) {
	e := entornoCompleto(t)
	id := sembrarConClave(t, e, "maria", "María Paz", "logistica", "inicial123", true)
	token, rec := e.iniciarSesion("maria", "inicial123")
	if token == "" {
		t.Fatalf("login = %d %s", rec.Code, rec.Body)
	}
	n := e.total(t)

	// Una clave actual incorrecta (400) no deja rastro.
	if rec := pedir(e.mux, "POST", "/api/v1/auth/cambiar-clave", token, map[string]any{"clave_actual": "mala", "clave_nueva": "ClaveSegura9"}); rec.Code != http.StatusBadRequest {
		t.Fatalf("clave mala = %d", rec.Code)
	}
	if e.total(t) != n {
		t.Fatal("un cambio de clave rechazado dejó una entrada")
	}

	if rec := pedir(e.mux, "POST", "/api/v1/auth/cambiar-clave", token, map[string]any{"clave_actual": "inicial123", "clave_nueva": "ClaveSegura9"}); rec.Code != http.StatusOK {
		t.Fatalf("cambiar clave = %d %s", rec.Code, rec.Body)
	}
	a := e.unaNueva(t, n, "usuario.cambiar_clave")
	if a.Usuario != "María Paz" || *a.IDUsuario != id || a.Entidad != "usuario" || *a.IDEntidad != int64(id) {
		t.Errorf("asiento = %+v", a)
	}
	if a.sub("antes", "debe_cambiar_clave") != true || a.sub("despues", "debe_cambiar_clave") != false {
		t.Errorf("detalle = %v", a.Detalle)
	}
	var secretos int
	_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM bitacora WHERE detalle::text ~* '(hash|\$2[aby]\$|inicial123|ClaveSegura9)'`).Scan(&secretos)
	if secretos != 0 {
		t.Errorf("entradas con secretos: %d", secretos)
	}
}

func TestInicioYCierreDeSesion(t *testing.T) {
	e := entornoCompleto(t)
	id := sembrarConClave(t, e, "luis", "Luis Rojas", "logistica", "secreta123", false)

	// Una clave mala no abre sesión ni deja entrada (evita llenar una tabla inborrable).
	if _, rec := e.iniciarSesion("luis", "mala"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("login malo = %d", rec.Code)
	}
	if e.total(t) != 0 {
		t.Fatal("un login fallido dejó una entrada")
	}

	token, rec := e.iniciarSesion("luis", "secreta123")
	if token == "" {
		t.Fatalf("login = %d %s", rec.Code, rec.Body)
	}
	a := e.unaNueva(t, 0, "sesion.inicio")
	if a.Usuario != "Luis Rojas" || *a.IDUsuario != id || a.Entidad != "usuario" || *a.IDEntidad != int64(id) {
		t.Errorf("sesion.inicio = %+v", a)
	}

	// Sin token no se puede cerrar sesión.
	if rec := pedir(e.mux, "POST", "/api/v1/auth/logout", "", nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("logout sin token = %d", rec.Code)
	}
	if e.total(t) != 1 {
		t.Fatal("un logout sin token dejó una entrada")
	}

	rec = pedir(e.mux, "POST", "/api/v1/auth/logout", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("logout = %d %s", rec.Code, rec.Body)
	}
	cierre := e.unaNueva(t, 1, "sesion.cierre")
	if cierre.Usuario != "Luis Rojas" || *cierre.IDUsuario != id || cierre.Entidad != "usuario" || *cierre.IDEntidad != int64(id) {
		t.Errorf("sesion.cierre = %+v", cierre)
	}
}

func TestCerrarSesionSePermiteConLaClavePendienteDeCambio(t *testing.T) {
	e := entornoCompleto(t)
	sembrarConClave(t, e, "pablo", "Pablo Diaz", "logistica", "inicial123", true)
	token, _ := e.iniciarSesion("pablo", "inicial123")
	if rec := pedir(e.mux, "POST", "/api/v1/auth/logout", token, nil); rec.Code != http.StatusOK {
		t.Fatalf("logout con clave pendiente = %d %s", rec.Code, rec.Body)
	}
	if acciones := e.asientos(t); len(acciones) != 2 || acciones[1].Accion != "sesion.cierre" {
		t.Errorf("asientos = %+v", acciones)
	}
}

func TestMaestrosGeneranSusEntradas(t *testing.T) {
	e := entornoCompleto(t)

	n := e.total(t)
	rec := pedir(e.mux, "POST", "/api/v1/rubros", e.tokenLogistica, map[string]any{"nombre": "Rubro de prueba", "descripcion": "x"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("crear rubro = %d %s", rec.Code, rec.Body)
	}
	var rubro map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &rubro)
	id := int64(rubro["id_rubro"].(float64))
	a := e.unaNueva(t, n, "maestro.crear")
	if a.Usuario != "Ana Quispe" || a.Entidad != "rubro" || *a.IDEntidad != id || a.sub("despues", "nombre") != "Rubro de prueba" {
		t.Errorf("crear = %+v", a)
	}

	// Un duplicado (409) y un cuerpo inválido (422) no dejan rastro.
	n = e.total(t)
	pedir(e.mux, "POST", "/api/v1/rubros", e.tokenLogistica, map[string]any{"nombre": "Rubro de prueba"})
	pedir(e.mux, "POST", "/api/v1/rubros", e.tokenLogistica, map[string]any{})
	if e.total(t) != n {
		t.Fatal("altas rechazadas dejaron entradas")
	}

	rec = pedir(e.mux, "PUT", fmt.Sprintf("/api/v1/rubros/%d", id), e.tokenLogistica, map[string]any{"descripcion": "y"})
	if rec.Code != http.StatusOK {
		t.Fatalf("editar rubro = %d %s", rec.Code, rec.Body)
	}
	a = e.unaNueva(t, n, "maestro.editar")
	if a.Entidad != "rubro" || *a.IDEntidad != id || a.sub("antes", "descripcion") != "x" || a.sub("despues", "descripcion") != "y" {
		t.Errorf("editar = %v", a.Detalle)
	}

	// Editar algo que no existe (404) no deja rastro.
	n = e.total(t)
	if rec := pedir(e.mux, "PUT", "/api/v1/rubros/99999", e.tokenLogistica, map[string]any{"descripcion": "y"}); rec.Code != http.StatusNotFound {
		t.Fatalf("editar inexistente = %d", rec.Code)
	}
	if e.total(t) != n {
		t.Fatal("editar un inexistente dejó una entrada")
	}

	if rec := pedir(e.mux, "DELETE", fmt.Sprintf("/api/v1/rubros/%d", id), e.tokenLogistica, nil); rec.Code != http.StatusOK {
		t.Fatalf("borrar rubro = %d", rec.Code)
	}
	a = e.unaNueva(t, n, "maestro.eliminar")
	if a.Entidad != "rubro" || *a.IDEntidad != id || a.sub("antes", "nombre") != "Rubro de prueba" {
		t.Errorf("eliminar = %v", a.Detalle)
	}
	if a.sub("despues") != nil {
		t.Errorf("un borrado no tiene «despues»: %v", a.Detalle)
	}

	n = e.total(t)
	if rec := pedir(e.mux, "DELETE", fmt.Sprintf("/api/v1/rubros/%d", id), e.tokenLogistica, nil); rec.Code != http.StatusNotFound {
		t.Fatalf("borrar de nuevo = %d", rec.Code)
	}
	if e.total(t) != n {
		t.Error("borrar un inexistente dejó una entrada")
	}
}

// Si no se puede escribir en la bitácora, el cambio no se hace (regla de negocio de REQ-33).
func TestSiFallaLaBitacoraFallaElCambio(t *testing.T) {
	e := entornoCompleto(t)
	ctx := context.Background()
	cuenta := func(sql string, args ...any) int {
		t.Helper()
		var n int
		if err := e.pool.QueryRow(ctx, sql, args...).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	// Preparación mientras la bitácora todavía funciona.
	gastoParaAdjuntar := e.registrarGasto(t)
	gastoAprobar := e.registrarGasto(t)
	aprobar := e.adjuntar(t, gastoAprobar)
	gastoRechazar := e.registrarGasto(t)
	rechazar := e.adjuntar(t, gastoRechazar)
	if rec := pedir(e.mux, "PUT", "/clientes/2/checklist", e.tokenLogistica, map[string]any{"criterios": []criterioJSON{{Texto: "Criterio propio"}}}); rec.Code != http.StatusOK {
		t.Fatalf("preparar checklist = %d", rec.Code)
	}
	otro := pedir(e.mux, "POST", "/api/v1/usuarios", e.tokenAdmin, cuerpoUsuario("otro"))
	otroID := usuarioDe(t, otro).ID
	sembrarConClave(t, e, "maria", "María Paz", "logistica", "inicial123", true)
	tokenMaria, _ := e.iniciarSesion("maria", "inicial123")
	sembrarConClave(t, e, "juan", "Juan Gil", "logistica", "secreta123", false)
	var rubro map[string]any
	rec := pedir(e.mux, "POST", "/api/v1/rubros", e.tokenLogistica, map[string]any{"nombre": "Para borrar"})
	_ = json.Unmarshal(rec.Body.Bytes(), &rubro)
	rubroID := int(rubro["id_rubro"].(float64))
	var hashMaria string
	_ = e.pool.QueryRow(ctx, `SELECT hashed_password FROM usuario WHERE username = 'maria'`).Scan(&hashMaria)
	asientosAntes := e.total(t)

	// A partir de aquí, todo INSERT en la bitácora falla.
	if _, err := e.pool.Exec(ctx, `ALTER TABLE bitacora ADD CONSTRAINT bitacora_rota CHECK (false) NOT VALID`); err != nil {
		t.Fatal(err)
	}

	gastos := cuenta(`SELECT count(*) FROM gasto`)
	casos := []struct {
		nombre, metodo, ruta, token string
		cuerpo                      any
		intacto                     func() bool
	}{
		{"registrar gasto", "POST", "/gastos", e.tokenLogistica, map[string]any{"viaje_id": 1, "cliente_id": 1, "monto": 10},
			func() bool { return cuenta(`SELECT count(*) FROM gasto`) == gastos }},
		{"aprobar", "POST", fmt.Sprintf("/comprobantes/%d/aprobar", aprobar), e.tokenLogistica, map[string]any{"checklist": completo()},
			func() bool {
				return cuenta(`SELECT count(*) FROM comprobante WHERE id = $1 AND estado = 'Pendiente'`, aprobar) == 1 &&
					cuenta(`SELECT count(*) FROM gasto WHERE id = $1 AND estado = 'Pagado'`, gastoAprobar) == 1 &&
					cuenta(`SELECT count(*) FROM checklist_comprobante WHERE comprobante_id = $1`, aprobar) == 0
			}},
		{"rechazar", "POST", fmt.Sprintf("/comprobantes/%d/rechazar", rechazar), e.tokenLogistica, map[string]any{"motivo_rechazo": "borroso"},
			func() bool {
				return cuenta(`SELECT count(*) FROM comprobante WHERE id = $1 AND estado = 'Pendiente'`, rechazar) == 1 &&
					cuenta(`SELECT count(*) FROM gasto WHERE id = $1 AND estado = 'Pagado'`, gastoRechazar) == 1
			}},
		{"configurar checklist", "PUT", "/clientes/1/checklist", e.tokenLogistica, map[string]any{"criterios": []criterioJSON{{Texto: "Nuevo criterio"}}},
			func() bool { return cuenta(`SELECT count(*) FROM criterio_checklist WHERE cliente_id = 1`) == 0 }},
		{"restablecer checklist", "DELETE", "/clientes/2/checklist", e.tokenLogistica, nil,
			func() bool { return cuenta(`SELECT count(*) FROM criterio_checklist WHERE cliente_id = 2`) == 1 }},
		{"crear usuario", "POST", "/api/v1/usuarios", e.tokenAdmin, cuerpoUsuario("fallido"),
			func() bool { return cuenta(`SELECT count(*) FROM usuario WHERE username = 'fallido'`) == 0 }},
		{"registrar usuario", "POST", "/api/v1/auth/register", e.tokenAdmin, cuerpoUsuario("fallido2"),
			func() bool { return cuenta(`SELECT count(*) FROM usuario WHERE username = 'fallido2'`) == 0 }},
		{"editar usuario", "PUT", fmt.Sprintf("/api/v1/usuarios/%d", otroID), e.tokenAdmin, map[string]any{"nombre": "Cambiado"},
			func() bool {
				return cuenta(`SELECT count(*) FROM usuario WHERE id_usuario = $1 AND nombre = 'Persona otro'`, otroID) == 1
			}},
		{"desactivar usuario", "POST", fmt.Sprintf("/api/v1/usuarios/%d/desactivar", otroID), e.tokenAdmin, nil,
			func() bool {
				return cuenta(`SELECT count(*) FROM usuario WHERE id_usuario = $1 AND estado`, otroID) == 1
			}},
		{"cambiar clave", "POST", "/api/v1/auth/cambiar-clave", tokenMaria, map[string]any{"clave_actual": "inicial123", "clave_nueva": "ClaveSegura9"},
			func() bool {
				return cuenta(`SELECT count(*) FROM usuario WHERE username = 'maria' AND hashed_password = $1 AND debe_cambiar_clave`, hashMaria) == 1
			}},
		{"cerrar sesión", "POST", "/api/v1/auth/logout", e.tokenAdmin, nil, func() bool { return true }},
		{"crear maestro", "POST", "/api/v1/rubros", e.tokenLogistica, map[string]any{"nombre": "Fallido"},
			func() bool { return cuenta(`SELECT count(*) FROM rubro WHERE nombre = 'Fallido'`) == 0 }},
		{"editar maestro", "PUT", fmt.Sprintf("/api/v1/rubros/%d", rubroID), e.tokenLogistica, map[string]any{"nombre": "Renombrado"},
			func() bool {
				return cuenta(`SELECT count(*) FROM rubro WHERE id_rubro = $1 AND nombre = 'Para borrar'`, rubroID) == 1
			}},
		{"borrar maestro", "DELETE", fmt.Sprintf("/api/v1/rubros/%d", rubroID), e.tokenLogistica, nil,
			func() bool { return cuenta(`SELECT count(*) FROM rubro WHERE id_rubro = $1`, rubroID) == 1 }},
	}
	for _, c := range casos {
		rec := pedir(e.mux, c.metodo, c.ruta, c.token, c.cuerpo)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("%s: código %d (%s), quiero 500", c.nombre, rec.Code, rec.Body)
		}
		if !c.intacto() {
			t.Errorf("%s: el cambio se hizo aunque falló la bitácora", c.nombre)
		}
	}

	// Adjuntar y el inicio de sesión también.
	if rec := e.subirComprobante(t, gastoParaAdjuntar, "boleta.png", pngDe(2048)); rec.Code != http.StatusInternalServerError {
		t.Errorf("adjuntar: código %d, quiero 500", rec.Code)
	}
	if cuenta(`SELECT count(*) FROM comprobante WHERE gasto_id = $1`, gastoParaAdjuntar) != 0 ||
		cuenta(`SELECT count(*) FROM gasto WHERE id = $1 AND estado = 'Registrado'`, gastoParaAdjuntar) != 1 {
		t.Error("adjuntar: el cambio se hizo aunque falló la bitácora")
	}
	if token, rec := e.iniciarSesion("juan", "secreta123"); token != "" || rec.Code != http.StatusInternalServerError {
		t.Errorf("login con la bitácora rota: token=%q código %d, quiero 500 sin token", token, rec.Code)
	}
	if e.total(t) != asientosAntes {
		t.Errorf("la bitácora cambió de %d a %d entradas", asientosAntes, e.total(t))
	}
}

func TestHistorialDelGastoEnOrdenCronologico(t *testing.T) {
	e := entornoCompleto(t)
	gasto := e.registrarGasto(t)
	ajeno := e.registrarGasto(t)
	e.adjuntar(t, ajeno)
	primero := e.adjuntar(t, gasto)
	if rec := pedir(e.mux, "POST", fmt.Sprintf("/comprobantes/%d/rechazar", primero), e.tokenLogistica, map[string]any{"motivo_rechazo": "borroso"}); rec.Code != http.StatusOK {
		t.Fatalf("rechazar = %d", rec.Code)
	}
	segundo := e.adjuntar(t, gasto)
	if rec := pedir(e.mux, "POST", fmt.Sprintf("/comprobantes/%d/aprobar", segundo), e.tokenLogistica, map[string]any{"checklist": completo()}); rec.Code != http.StatusOK {
		t.Fatalf("aprobar = %d", rec.Code)
	}

	// Cualquier usuario autenticado lo lee, el admin también.
	for _, token := range []string{e.tokenLogistica, e.tokenAdmin} {
		rec := pedir(e.mux, "GET", fmt.Sprintf("/gastos/%d/historial", gasto), token, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("historial = %d %s", rec.Code, rec.Body)
		}
		var historial []map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &historial); err != nil {
			t.Fatalf("la respuesta debía ser una lista: %s", rec.Body)
		}
		acciones := []string{}
		for _, h := range historial {
			acciones = append(acciones, h["accion"].(string))
		}
		quiero := "gasto.registrar,comprobante.adjuntar,comprobante.rechazar,comprobante.adjuntar,comprobante.aprobar"
		if strings.Join(acciones, ",") != quiero {
			t.Fatalf("acciones = %v, quiero %s", acciones, quiero)
		}
		for i, h := range historial {
			for _, campo := range []string{"fecha", "usuario", "accion", "entidad", "id_entidad", "detalle"} {
				if _, ok := h[campo]; !ok {
					t.Errorf("la entrada %d no trae %q: %v", i, campo, h)
				}
			}
			if i > 0 && h["fecha"].(string) < historial[i-1]["fecha"].(string) {
				t.Errorf("fechas fuera de orden: %v antes de %v", historial[i-1]["fecha"], h["fecha"])
			}
			if h["usuario"] != "Ana Quispe" {
				t.Errorf("usuario = %v", h["usuario"])
			}
		}
		if historial[0]["entidad"] != "gasto" || historial[1]["entidad"] != "comprobante" {
			t.Errorf("entidades = %v / %v", historial[0]["entidad"], historial[1]["entidad"])
		}
		if _, ok := historial[2]["detalle"].(map[string]any); !ok {
			t.Errorf("detalle = %v", historial[2]["detalle"])
		}
	}
}

func TestHistorialDeUnGastoSinMovimientosYErrores(t *testing.T) {
	e := entornoCompleto(t)
	// Un gasto creado a mano (sin pasar por la API) existe y tiene el historial vacío: una lista, no null.
	var id int
	if err := e.pool.QueryRow(context.Background(), `INSERT INTO gasto (viaje_id, cliente_id, monto, fecha, creado_por) VALUES (1, 1, 5, '2026-10-01', 'x') RETURNING id`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	rec := pedir(e.mux, "GET", fmt.Sprintf("/gastos/%d/historial", id), e.tokenLogistica, nil)
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Errorf("sin movimientos: %d %q", rec.Code, rec.Body)
	}
	if rec := pedir(e.mux, "GET", "/gastos/99999/historial", e.tokenLogistica, nil); rec.Code != http.StatusNotFound {
		t.Errorf("gasto inexistente = %d, quiero 404", rec.Code)
	}
	if rec := pedir(e.mux, "GET", "/gastos/abc/historial", e.tokenLogistica, nil); rec.Code != http.StatusBadRequest {
		t.Errorf("id inválido = %d, quiero 400", rec.Code)
	}
	if rec := pedir(e.mux, "GET", fmt.Sprintf("/gastos/%d/historial", id), "", nil); rec.Code != http.StatusUnauthorized {
		t.Errorf("sin token = %d, quiero 401", rec.Code)
	}
}

// Con la bitácora recién creada (RNF-9/10), ni siquiera el usuario con el que se conecta el
// servicio puede alterarla.
func TestLaBitacoraDeLaApiNoSePuedeAlterar(t *testing.T) {
	e := entornoCompleto(t)
	e.registrarGasto(t)
	for _, sql := range []string{`UPDATE bitacora SET usuario = 'otro'`, `DELETE FROM bitacora`, `TRUNCATE bitacora`} {
		if _, err := e.pool.Exec(context.Background(), sql); err == nil {
			t.Errorf("%q debía rechazarse", sql)
		}
	}
	if e.total(t) != 1 {
		t.Error("la bitácora cambió")
	}
}
