package config

import (
	"strings"
	"testing"
)

func TestLogArchivoTieneValorPorDefecto(t *testing.T) {
	t.Setenv("LOG_ARCHIVO", "")
	if got := Load().LogArchivo; got != "logs/errores.log" {
		t.Fatalf("LogArchivo = %q", got)
	}
}

func TestLogArchivoSeLeeDelEntorno(t *testing.T) {
	t.Setenv("LOG_ARCHIVO", "/var/log/lizar/errores.log")
	if got := Load().LogArchivo; got != "/var/log/lizar/errores.log" {
		t.Fatalf("LogArchivo = %q", got)
	}
}

func TestSinJWTSecretNoHayClaveYNoSeUsaUnaPublica(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	cfg := Load()
	if cfg.JWTSecret != "" {
		t.Fatalf("sin JWT_SECRET no debe haber clave por defecto, hay %d caracteres", len(cfg.JWTSecret))
	}
	if err := cfg.Validar(); err == nil {
		t.Fatal("el servicio no debe arrancar sin JWT_SECRET")
	}
}

func TestValidarRechazaLaClavePublicaDeLaVersionAnterior(t *testing.T) {
	// Está en el código fuente de versiones anteriores: cualquiera puede firmar tokens con ella.
	cfg := Config{JWTSecret: "lizar-srl-fleet-manager-secret-2025"}
	if err := cfg.Validar(); err == nil {
		t.Fatal("la clave pública de la versión anterior no debe aceptarse")
	}
}

func TestValidarRechazaClavesCortas(t *testing.T) {
	if err := (Config{JWTSecret: "corta-pero-propia"}).Validar(); err == nil {
		t.Fatal("una clave de menos de 32 caracteres no debe aceptarse")
	}
}

func TestValidarAceptaUnaClaveLargaYPropia(t *testing.T) {
	if err := (Config{JWTSecret: "0123456789abcdef0123456789abcdef0123"}).Validar(); err != nil {
		t.Fatalf("Validar = %v", err)
	}
}

func TestElErrorDeValidarNoRepiteLaClave(t *testing.T) {
	clave := "corta-secreta"
	err := (Config{JWTSecret: clave}).Validar()
	if err == nil || strings.Contains(err.Error(), clave) {
		t.Fatalf("el mensaje no debe incluir la clave: %v", err)
	}
}
