package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gastos-service/internal/config"
	"gastos-service/internal/db"
	"gastos-service/internal/handlers"
	"gastos-service/internal/notificaciones"
	"gastos-service/internal/ocr"
	"gastos-service/internal/registro"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validar(); err != nil {
		log.Fatalf("configuración inválida: %v", err)
	}

	// RNF-14: los errores técnicos van también a un archivo. Si no se puede abrir, el servicio
	// sigue con la consola sola: no vale la pena dejar de atender por no poder anotar.
	cerrarRegistro, err := registro.Configurar(cfg.LogArchivo, os.Stderr)
	if err != nil {
		log.Printf("%v; los errores solo saldrán por consola", err)
		cerrarRegistro = func() error { return nil }
	}
	defer cerrarRegistro()

	pool, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("no se pudo conectar a la base de datos: %v", err)
	}
	defer pool.Close()

	if err := db.RunMigrations(pool, "migrations"); err != nil {
		log.Fatalf("error aplicando migraciones: %v", err)
	}

	handler := handlers.New(pool, cfg)
	if lector, ok := handler.Lector.(*ocr.Herramientas); ok {
		log.Printf("lectura de comprobantes: %s", lector.Resumen())
	}
	if err := handler.Auth.SembrarAdmin(context.Background()); err != nil {
		log.Fatalf("no se pudo sembrar el usuario admin: %v", err)
	}

	if err := os.MkdirAll(cfg.UploadsDir, 0o755); err != nil {
		log.Fatalf("no se pudo crear el directorio de comprobantes: %v", err)
	}

	// Ctrl+C o SIGTERM cancelan el contexto: detienen el generador de avisos y cierran el servidor.
	ctx, parar := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer parar()

	// REQ-20 y REQ-21: una pasada al arrancar y otra cada hora; así un aviso nace dentro de las
	// 24 horas de cumplirse la condición (RNF-8).
	generador := &notificaciones.Generador{DB: pool, Ahora: time.Now}
	go notificaciones.Ejecutar(ctx, time.Hour, generador.Generar, log.Printf)

	servidor := &http.Server{Addr: ":" + cfg.Port, Handler: handler.Servidor()}
	cerrado := make(chan struct{})
	go func() {
		defer close(cerrado)
		<-ctx.Done()
		limite, cancelar := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelar()
		_ = servidor.Shutdown(limite)
	}()

	log.Printf("gastos-service escuchando en :%s", cfg.Port)
	if err := servidor.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
	<-cerrado
}
