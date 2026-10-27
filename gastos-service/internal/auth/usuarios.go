package auth

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"unicode/utf8"

	"gastos-service/internal/bitacora"
	"gastos-service/internal/crud"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Límites de las columnas de la tabla usuario y mínimo de la contraseña (REQ-32).
const (
	maxUsername = 50
	maxEmail    = 100
	maxNombre   = 100
	minClave    = 8
)

// Errores de negocio de la gestión de usuarios; el handler los traduce a 404 o 409.
var (
	ErrUsuarioNoEncontrado = errors.New("usuario no encontrado")
	ErrUltimoAdmin         = errors.New("no puede quedar el sistema sin un administrador activo")
	ErrAutodesactivacion   = errors.New("un administrador no puede desactivarse a sí mismo")
	ErrCorreoRepetido      = errors.New("el correo ya está registrado")
	ErrUsuarioRepetido     = errors.New("el usuario ya existe")
)

// bloqueoAdmins serializa (con pg_advisory_xact_lock) todo cambio que pueda quitar un admin activo:
// sin él, dos admins que se desactivan a la vez pasarían cada uno la comprobación y dejarían al
// sistema sin administrador.
const bloqueoAdmins = 0x4C495A41

// --- Validación del cuerpo (mismos errores 422 que /auth/register) ---

func errorCampo(tipo, campo, msg string, entrada any) crud.ErrorCampo {
	return crud.ErrorCampo{Tipo: tipo, Loc: []any{"body", campo}, Msg: msg, Input: entrada}
}

// textoLimitado lee un texto recortado de entre 1 y max caracteres. Si falta (y obligatorio) o no
// es válido añade el error; ok indica que el valor se puede usar.
func textoLimitado(cuerpo map[string]any, campo string, max int, errs *[]crud.ErrorCampo) (string, bool) {
	crudo, ok := texto(cuerpo, campo, errs)
	if !ok {
		return "", false
	}
	valor := strings.TrimSpace(crudo)
	if valor == "" {
		*errs = append(*errs, crud.ErrorCampo{Tipo: "string_too_short", Loc: []any{"body", campo}, Msg: "String should have at least 1 character", Input: crudo, Ctx: map[string]any{"min_length": 1}})
		return "", false
	}
	if utf8.RuneCountInString(valor) > max {
		*errs = append(*errs, crud.ErrorCampo{Tipo: "string_too_long", Loc: []any{"body", campo}, Msg: "String should have at most " + strconv.Itoa(max) + " characters", Input: crudo, Ctx: map[string]any{"max_length": max}})
		return "", false
	}
	return valor, true
}

func leerUsername(cuerpo map[string]any, errs *[]crud.ErrorCampo) (string, bool) {
	v, ok := textoLimitado(cuerpo, "username", maxUsername, errs)
	if ok && strings.ContainsAny(v, " \t\r\n") {
		*errs = append(*errs, errorCampo("value_error", "username", "Value error, El usuario no puede llevar espacios", v))
		return "", false
	}
	return v, ok
}

func leerEmail(cuerpo map[string]any, errs *[]crud.ErrorCampo) (string, bool) {
	v, ok := textoLimitado(cuerpo, "email", maxEmail, errs)
	if ok && !reCorreo.MatchString(v) {
		*errs = append(*errs, errorCampo("value_error", "email", "value is not a valid email address", v))
		return "", false
	}
	return v, ok
}

func leerRol(cuerpo map[string]any, errs *[]crud.ErrorCampo) (string, bool) {
	r, ok := texto(cuerpo, "rol", errs)
	if ok && !RolValido(r) {
		*errs = append(*errs, errorCampo("value_error", "rol", "Value error, Rol debe ser 'admin' o 'logistica'", r))
		return "", false
	}
	return r, ok
}

// leerClave lee una contraseña de al menos minClave caracteres. Nunca devuelve la contraseña en el
// error (el campo input va vacío): la respuesta de un 422 no debe repetirla.
func leerClave(cuerpo map[string]any, campo string, errs *[]crud.ErrorCampo) (string, bool) {
	antes := len(*errs)
	clave, ok := texto(cuerpo, campo, errs)
	if !ok {
		for i := antes; i < len(*errs); i++ {
			(*errs)[i].Input = nil
		}
		return "", false
	}
	if utf8.RuneCountInString(clave) < minClave {
		*errs = append(*errs, crud.ErrorCampo{Tipo: "string_too_short", Loc: []any{"body", campo}, Msg: "String should have at least " + strconv.Itoa(minClave) + " characters", Input: nil, Ctx: map[string]any{"min_length": minClave}})
		return "", false
	}
	return clave, true
}

// datosUsuario es lo que pide el admin al crear un usuario.
type datosUsuario struct{ Username, Email, Nombre, Rol, Password string }

// validarCreacion valida el cuerpo de POST /usuarios: usuario, correo, nombre y contraseña son
// obligatorios; el rol es opcional y por defecto «logistica».
func validarCreacion(cuerpo map[string]any) (datosUsuario, []crud.ErrorCampo) {
	var errs []crud.ErrorCampo
	d := datosUsuario{Rol: RolLogistica}
	d.Username, _ = leerUsername(cuerpo, &errs)
	d.Email, _ = leerEmail(cuerpo, &errs)
	d.Password, _ = leerClave(cuerpo, "password", &errs)
	d.Nombre, _ = textoLimitado(cuerpo, "nombre", maxNombre, &errs)
	if _, presente := cuerpo["rol"]; presente {
		if r, ok := leerRol(cuerpo, &errs); ok {
			d.Rol = r
		}
	}
	return d, errs
}

// cambiosUsuario son los campos que PUT /usuarios/{id} puede modificar; nil significa «no tocar».
// El nombre de usuario no cambia nunca. Password restablece la contraseña (el usuario deberá
// cambiarla en su próximo ingreso).
type cambiosUsuario struct{ Nombre, Email, Rol, Password *string }

func validarEdicion(cuerpo map[string]any) (cambiosUsuario, []crud.ErrorCampo) {
	var errs []crud.ErrorCampo
	var c cambiosUsuario
	if _, presente := cuerpo["nombre"]; presente {
		if v, ok := textoLimitado(cuerpo, "nombre", maxNombre, &errs); ok {
			c.Nombre = &v
		}
	}
	if _, presente := cuerpo["email"]; presente {
		if v, ok := leerEmail(cuerpo, &errs); ok {
			c.Email = &v
		}
	}
	if _, presente := cuerpo["rol"]; presente {
		if v, ok := leerRol(cuerpo, &errs); ok {
			c.Rol = &v
		}
	}
	if _, presente := cuerpo["password"]; presente {
		if v, ok := leerClave(cuerpo, "password", &errs); ok {
			c.Password = &v
		}
	}
	return c, errs
}

// validarCambioClave valida el cuerpo de POST /auth/cambiar-clave.
func validarCambioClave(cuerpo map[string]any) (actual, nueva string, errs []crud.ErrorCampo) {
	actual, _ = textoSinEco(cuerpo, "clave_actual", &errs)
	nueva, _ = leerClave(cuerpo, "clave_nueva", &errs)
	return actual, nueva, errs
}

// textoSinEco es como texto pero sin repetir en el error lo que mandó el usuario (es una contraseña).
func textoSinEco(cuerpo map[string]any, campo string, errs *[]crud.ErrorCampo) (string, bool) {
	antes := len(*errs)
	v, ok := texto(cuerpo, campo, errs)
	for i := antes; i < len(*errs); i++ {
		(*errs)[i].Input = nil
	}
	return v, ok
}

// --- Acceso a la tabla ---

// PorID busca un usuario; devuelve nil si no existe.
func (s *Servicio) PorID(ctx context.Context, id int) (*Usuario, error) {
	return leer(s.DB.QueryRow(ctx, "SELECT "+columnasUsuario+" FROM usuario WHERE id_usuario = $1", id))
}

// Listar devuelve todos los usuarios, activos e inactivos, por orden de creación.
func (s *Servicio) Listar(ctx context.Context) ([]Usuario, error) {
	filas, err := s.DB.Query(ctx, "SELECT "+columnasUsuario+" FROM usuario ORDER BY id_usuario")
	if err != nil {
		return nil, err
	}
	defer filas.Close()
	lista := []Usuario{}
	for filas.Next() {
		var u Usuario
		if err := filas.Scan(&u.ID, &u.Username, &u.Email, &u.Hash, &u.Nombre, &u.Rol, &u.Estado, &u.DebeCambiarClave); err != nil {
			return nil, err
		}
		lista = append(lista, u)
	}
	return lista, filas.Err()
}

// enTransaccionDeAdmins ejecuta fn en una transacción que ya tiene el bloqueo de admins.
func (s *Servicio) enTransaccionDeAdmins(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", int64(bloqueoAdmins)); err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// otrosAdminsActivos cuenta los administradores activos distintos de id.
func otrosAdminsActivos(ctx context.Context, tx pgx.Tx, id int) (int, error) {
	var n int
	err := tx.QueryRow(ctx, "SELECT count(*) FROM usuario WHERE rol = $1 AND estado AND id_usuario <> $2", RolAdmin, id).Scan(&n)
	return n, err
}

// esDuplicado traduce la violación de unicidad de la base (carrera entre dos altas) al error de negocio.
func esDuplicado(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		if strings.Contains(pg.ConstraintName, "email") {
			return ErrCorreoRepetido
		}
		return ErrUsuarioRepetido
	}
	return err
}

// CrearUsuario crea un usuario activo que debe cambiar la contraseña en su primer ingreso; quien es
// el administrador que lo da de alta.
func (s *Servicio) CrearUsuario(ctx context.Context, quien bitacora.Actor, d datosUsuario) (*Usuario, error) {
	if existe, err := s.PorUsername(ctx, d.Username); err != nil {
		return nil, err
	} else if existe != nil {
		return nil, ErrUsuarioRepetido
	}
	if existe, err := s.PorEmail(ctx, d.Email); err != nil {
		return nil, err
	} else if existe != nil {
		return nil, ErrCorreoRepetido
	}
	u, err := s.Crear(ctx, quien, Usuario{Username: d.Username, Email: d.Email, Nombre: d.Nombre, Rol: d.Rol}, d.Password)
	return u, esDuplicado(err)
}

// Editar aplica los cambios al usuario id; quien es el administrador que los hace y queda en la
// bitácora. Quitarle el rol de admin al último administrador activo da ErrUltimoAdmin. Restablecer
// la contraseña vuelve a exigir que el usuario la cambie.
func (s *Servicio) Editar(ctx context.Context, quien bitacora.Actor, id int, c cambiosUsuario) (*Usuario, error) {
	var hash string
	if c.Password != nil { // bcrypt tarda: se hace antes de tomar el bloqueo
		h, err := Hash(*c.Password)
		if err != nil {
			return nil, err
		}
		hash = h
	}
	var resultado *Usuario
	err := s.enTransaccionDeAdmins(ctx, func(tx pgx.Tx) error {
		actual, err := leer(tx.QueryRow(ctx, "SELECT "+columnasUsuario+" FROM usuario WHERE id_usuario = $1 FOR UPDATE", id))
		if err != nil {
			return err
		}
		if actual == nil {
			return ErrUsuarioNoEncontrado
		}
		nuevo := *actual
		if c.Nombre != nil {
			nuevo.Nombre = *c.Nombre
		}
		if c.Email != nil {
			nuevo.Email = *c.Email
		}
		if c.Rol != nil {
			nuevo.Rol = *c.Rol
		}
		if c.Password != nil {
			nuevo.Hash, nuevo.DebeCambiarClave = hash, true
		}
		if nuevo.Email != actual.Email {
			var otro int
			err := tx.QueryRow(ctx, "SELECT id_usuario FROM usuario WHERE email = $1 AND id_usuario <> $2", nuevo.Email, id).Scan(&otro)
			if err == nil {
				return ErrCorreoRepetido
			} else if !errors.Is(err, pgx.ErrNoRows) {
				return err
			}
		}
		if actual.Rol == RolAdmin && nuevo.Rol != RolAdmin && actual.Estado {
			if n, err := otrosAdminsActivos(ctx, tx, id); err != nil {
				return err
			} else if n == 0 {
				return ErrUltimoAdmin
			}
		}
		resultado, err = leer(tx.QueryRow(ctx, `
			UPDATE usuario SET nombre = $2, email = $3, rol = $4, hashed_password = $5, debe_cambiar_clave = $6
			WHERE id_usuario = $1 RETURNING `+columnasUsuario, id, nuevo.Nombre, nuevo.Email, nuevo.Rol, nuevo.Hash, nuevo.DebeCambiarClave))
		if err != nil {
			return esDuplicado(err)
		}
		detalle := map[string]any{"antes": instantanea(actual), "despues": instantanea(resultado)}
		if c.Password != nil {
			detalle["clave_restablecida"] = true
		}
		return bitacora.Registrar(ctx, tx, bitacora.Entrada{Actor: quien, Accion: "usuario.editar", Entidad: "usuario", IDEntidad: int64(id), Detalle: detalle})
	})
	return resultado, err
}

// Desactivar marca al usuario id como inactivo (no se borra nada). quien es el administrador que lo
// pide: no puede desactivarse a sí mismo, ni dejar al sistema sin un administrador activo. Repetirlo
// sobre un usuario ya inactivo no es un error, pero tampoco cambia nada y no se registra.
func (s *Servicio) Desactivar(ctx context.Context, quien bitacora.Actor, id int) (*Usuario, error) {
	if quien.ID == id {
		return nil, ErrAutodesactivacion
	}
	var resultado *Usuario
	err := s.enTransaccionDeAdmins(ctx, func(tx pgx.Tx) error {
		actual, err := leer(tx.QueryRow(ctx, "SELECT "+columnasUsuario+" FROM usuario WHERE id_usuario = $1 FOR UPDATE", id))
		if err != nil {
			return err
		}
		if actual == nil {
			return ErrUsuarioNoEncontrado
		}
		if !actual.Estado {
			resultado = actual
			return nil
		}
		if actual.Rol == RolAdmin {
			if n, err := otrosAdminsActivos(ctx, tx, id); err != nil {
				return err
			} else if n == 0 {
				return ErrUltimoAdmin
			}
		}
		resultado, err = leer(tx.QueryRow(ctx, "UPDATE usuario SET estado = FALSE WHERE id_usuario = $1 RETURNING "+columnasUsuario, id))
		if err != nil {
			return err
		}
		return bitacora.Registrar(ctx, tx, bitacora.Entrada{
			Actor: quien, Accion: "usuario.desactivar", Entidad: "usuario", IDEntidad: int64(id),
			Detalle: map[string]any{"antes": instantanea(actual), "despues": instantanea(resultado)},
		})
	})
	return resultado, err
}

// Activar vuelve a habilitar al usuario id; quien es el administrador que lo pide. Repetirlo sobre un
// usuario activo no es un error, pero tampoco cambia nada y no se registra.
func (s *Servicio) Activar(ctx context.Context, quien bitacora.Actor, id int) (*Usuario, error) {
	return s.cambiarEstado(ctx, quien, id)
}

func (s *Servicio) cambiarEstado(ctx context.Context, quien bitacora.Actor, id int) (*Usuario, error) {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	actual, err := leer(tx.QueryRow(ctx, "SELECT "+columnasUsuario+" FROM usuario WHERE id_usuario = $1 FOR UPDATE", id))
	if err != nil {
		return nil, err
	}
	if actual == nil {
		return nil, ErrUsuarioNoEncontrado
	}
	if actual.Estado {
		return actual, nil
	}
	u, err := leer(tx.QueryRow(ctx, "UPDATE usuario SET estado = TRUE WHERE id_usuario = $1 RETURNING "+columnasUsuario, id))
	if err != nil {
		return nil, err
	}
	err = bitacora.Registrar(ctx, tx, bitacora.Entrada{
		Actor: quien, Accion: "usuario.activar", Entidad: "usuario", IDEntidad: int64(id),
		Detalle: map[string]any{"antes": instantanea(actual), "despues": instantanea(u)},
	})
	if err != nil {
		return nil, err
	}
	return u, tx.Commit(ctx)
}

// CambiarClave guarda la contraseña nueva (ya validada) y quita la obligación de cambiarla; quien
// es el propio usuario.
func (s *Servicio) CambiarClave(ctx context.Context, quien bitacora.Actor, id int, nueva string) (*Usuario, error) {
	hash, err := Hash(nueva)
	if err != nil {
		return nil, err
	}
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	actual, err := leer(tx.QueryRow(ctx, "SELECT "+columnasUsuario+" FROM usuario WHERE id_usuario = $1 FOR UPDATE", id))
	if err != nil {
		return nil, err
	}
	if actual == nil {
		return nil, ErrUsuarioNoEncontrado
	}
	u, err := leer(tx.QueryRow(ctx, "UPDATE usuario SET hashed_password = $2, debe_cambiar_clave = FALSE WHERE id_usuario = $1 RETURNING "+columnasUsuario, id, hash))
	if err != nil {
		return nil, err
	}
	// Solo se anota que la clave cambió y si era obligatorio; nunca la clave ni su hash.
	err = bitacora.Registrar(ctx, tx, bitacora.Entrada{
		Actor: quien, Accion: "usuario.cambiar_clave", Entidad: "usuario", IDEntidad: int64(id),
		Detalle: map[string]any{
			"antes":   map[string]any{"debe_cambiar_clave": actual.DebeCambiarClave},
			"despues": map[string]any{"debe_cambiar_clave": u.DebeCambiarClave},
		},
	})
	if err != nil {
		return nil, err
	}
	return u, tx.Commit(ctx)
}
