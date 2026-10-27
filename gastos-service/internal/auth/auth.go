// Package auth reemplaza la autenticación de la API de Python: contraseñas con bcrypt y
// tokens JWT HS256 que llevan sub (usuario), rol, id y exp (8 horas).
package auth

import (
	"context"
	"errors"
	"time"

	"gastos-service/internal/bitacora"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

const vigenciaToken = 8 * time.Hour

// Usuario es la fila de la tabla usuario.
type Usuario struct {
	ID       int
	Username string
	Email    string
	Hash     string
	Nombre   string
	Rol      string
	Estado   bool
	// DebeCambiarClave es verdadera hasta que el usuario cambia la contraseña que le dieron; mientras
	// lo sea, el middleware solo le deja ver su perfil y cambiarla (ver Autenticar).
	DebeCambiarClave bool
}

// Servicio agrupa la clave de firma y el acceso a la tabla usuario.
type Servicio struct {
	DB    Consultas
	Clave []byte
	Ahora func() time.Time
	// prefijo es donde Registrar monta las rutas /auth (p. ej. /api/v1); el middleware lo usa para
	// reconocer las pocas rutas que se atienden mientras la contraseña está pendiente de cambio.
	prefijo string
}

// Consultas es lo que necesita de la base (lo cumple *pgxpool.Pool).
type Consultas interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Begin(ctx context.Context) (pgx.Tx, error)
}

// Hash genera el hash bcrypt (costo 12, como la API de Python). bcrypt solo mira los primeros 72
// bytes: se recorta igual que la versión de Python que se usaba, en vez de rechazar la contraseña.
func Hash(contrasena string) (string, error) {
	h, err := bcrypt.GenerateFromPassword(recortar(contrasena), 12)
	return string(h), err
}

// Verificar compara una contraseña con su hash (los hashes $2b$ de Python son válidos).
func Verificar(contrasena, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), recortar(contrasena)) == nil
}

func recortar(s string) []byte {
	b := []byte(s)
	if len(b) > 72 {
		b = b[:72]
	}
	return b
}

// CrearToken emite un token con la vigencia estándar.
func (s *Servicio) CrearToken(u Usuario) (string, error) {
	claims := jwt.MapClaims{"sub": u.Username, "rol": u.Rol, "id": u.ID, "exp": s.ahora().Add(vigenciaToken).Unix()}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.Clave)
}

// UsuarioDeToken devuelve el nombre de usuario del token, o error si no es válido, venció o no trae
// vencimiento (un token sin exp valdría para siempre).
func (s *Servicio) UsuarioDeToken(token string) (string, error) {
	t, err := jwt.Parse(token, func(*jwt.Token) (any, error) { return s.Clave, nil },
		jwt.WithValidMethods([]string{"HS256"}), jwt.WithTimeFunc(s.ahora), jwt.WithExpirationRequired())
	if err != nil || !t.Valid {
		return "", errors.New("token inválido")
	}
	sub, _ := t.Claims.(jwt.MapClaims)["sub"].(string)
	if sub == "" {
		return "", errors.New("token inválido")
	}
	return sub, nil
}

func (s *Servicio) ahora() time.Time {
	if s.Ahora != nil {
		return s.Ahora()
	}
	return time.Now()
}

const columnasUsuario = "id_usuario, username, email, hashed_password, nombre, rol, estado, debe_cambiar_clave"

func leer(row pgx.Row) (*Usuario, error) {
	var u Usuario
	err := row.Scan(&u.ID, &u.Username, &u.Email, &u.Hash, &u.Nombre, &u.Rol, &u.Estado, &u.DebeCambiarClave)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &u, err
}

// PorUsername busca un usuario; devuelve nil si no existe.
func (s *Servicio) PorUsername(ctx context.Context, username string) (*Usuario, error) {
	return leer(s.DB.QueryRow(ctx, "SELECT "+columnasUsuario+" FROM usuario WHERE username = $1", username))
}

// PorEmail busca un usuario por correo; devuelve nil si no existe.
func (s *Servicio) PorEmail(ctx context.Context, email string) (*Usuario, error) {
	return leer(s.DB.QueryRow(ctx, "SELECT "+columnasUsuario+" FROM usuario WHERE email = $1", email))
}

// Crear guarda un usuario nuevo con la contraseña ya hasheada. quien es el autor del alta: queda en
// la bitácora (usuario.crear) dentro de la misma transacción, y si no se puede registrar no se crea.
func (s *Servicio) Crear(ctx context.Context, quien bitacora.Actor, u Usuario, contrasena string) (*Usuario, error) {
	hash, err := Hash(contrasena) // bcrypt tarda: antes de abrir la transacción
	if err != nil {
		return nil, err
	}
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	creado, err := leer(tx.QueryRow(ctx, `
		INSERT INTO usuario (username, email, hashed_password, nombre, rol, estado, created_at)
		VALUES ($1, $2, $3, $4, $5, TRUE, $6)
		RETURNING `+columnasUsuario, u.Username, u.Email, hash, u.Nombre, u.Rol, s.ahora().UTC()))
	if err != nil {
		return nil, err
	}
	err = bitacora.Registrar(ctx, tx, bitacora.Entrada{
		Actor: quien, Accion: "usuario.crear", Entidad: "usuario", IDEntidad: int64(creado.ID),
		Detalle: map[string]any{"despues": instantanea(creado)},
	})
	if err != nil {
		return nil, err
	}
	return creado, tx.Commit(ctx)
}

// instantanea es lo que la bitácora guarda de un usuario: nunca la contraseña ni su hash.
func instantanea(u *Usuario) map[string]any {
	return map[string]any{
		"username": u.Username, "email": u.Email, "nombre": u.Nombre, "rol": u.Rol,
		"estado": u.Estado, "debe_cambiar_clave": u.DebeCambiarClave,
	}
}

// registrar deja una sola entrada de bitácora (sesión). Si falla, quien llama debe fallar también.
func (s *Servicio) registrar(ctx context.Context, e bitacora.Entrada) error {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := bitacora.Registrar(ctx, tx, e); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// SembrarAdmin crea el usuario admin/admin123 si todavía no hay ningún usuario. Como la contraseña
// es conocida, nace con debe_cambiar_clave: el primer ingreso exige cambiarla.
func (s *Servicio) SembrarAdmin(ctx context.Context) error {
	var n int
	if err := s.DB.QueryRow(ctx, "SELECT count(*) FROM usuario").Scan(&n); err != nil || n > 0 {
		return err
	}
	_, err := s.Crear(ctx, bitacora.Actor{Nombre: "Sistema"}, Usuario{Username: "admin", Email: "admin@lizar.pe", Nombre: "Administrador", Rol: RolAdmin}, "admin123")
	return err
}
