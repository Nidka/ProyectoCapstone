package config

import (
	"bufio"
	"errors"
	"os"
	"strings"
)

// claveDeVersionAnterior es la clave que el servicio traía por defecto. Quedó en el código fuente y
// en el historial del repositorio, así que cualquiera puede firmar tokens con ella: se rechaza.
const claveDeVersionAnterior = "lizar-srl-fleet-manager-secret-2025"

// largoMinimoClave es el mínimo de caracteres de JWT_SECRET (32 bytes de HS256).
const largoMinimoClave = 32

// Config agrupa los parámetros de arranque del servicio, leídos de variables
// de entorno (o de un archivo .env local si existe).
type Config struct {
	Port        string
	DatabaseURL string
	UploadsDir  string
	// LogArchivo es el archivo donde se guardan los errores técnicos (RNF-14), además de la consola.
	LogArchivo string
	// JWTSecret firma los tokens de /api/v1/auth. No tiene valor por defecto: el servicio no arranca
	// sin una clave propia y larga (ver Validar). Quien conozca la clave puede hacerse pasar por
	// cualquier usuario.
	JWTSecret string
	AppEnv    string
}

// Load lee un .env si existe (sin sobreescribir variables ya definidas en el
// entorno) y arma la configuración con valores por defecto razonables.
func Load() Config {
	loadDotEnv(".env")

	return Config{
		Port:        getEnv("PORT", "8081"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"),
		UploadsDir:  getEnv("UPLOADS_DIR", "./uploads"),
		LogArchivo:  getEnv("LOG_ARCHIVO", "logs/errores.log"),
		JWTSecret:   getEnv("JWT_SECRET", ""),
		AppEnv:      getEnv("APP_ENV", "dev"),
	}
}

// Validar dice si la configuración permite arrancar. Hoy solo exige una JWT_SECRET propia: el rol
// y la identidad salen de un token firmado, y un secreto público o corto permite fabricarlos. El
// error nunca incluye la clave.
func (c Config) Validar() error {
	switch {
	case c.JWTSecret == "":
		return errors.New("falta JWT_SECRET: define una clave secreta de al menos 32 caracteres (por ejemplo, con `openssl rand -hex 32`)")
	case c.JWTSecret == claveDeVersionAnterior:
		return errors.New("JWT_SECRET usa la clave pública de una versión anterior: define una clave propia de al menos 32 caracteres")
	case len(c.JWTSecret) < largoMinimoClave:
		return errors.New("JWT_SECRET es demasiado corta: usa al menos 32 caracteres")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, value)
		}
	}
}
