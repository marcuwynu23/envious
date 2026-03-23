package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"envious-web/internal/api"
	"envious-web/internal/auth"
	"envious-web/internal/config"
	"envious-web/internal/storage"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()

	// Structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{}))
	slog.SetDefault(logger)

	store, err := storage.Open(cfg)
	if err != nil {
		logger.Error("db_open_failed", "error", err.Error())
		os.Exit(1)
	}
	defer store.Close()

	ctx := context.Background()
	if key, err := auth.InitAdminKey(ctx, store); err != nil {
		logger.Error("auth_init_failed", "error", err.Error())
		os.Exit(1)
	} else if key != "" {
		// Print once to stdout so the admin can capture it securely.
		log.Printf("Envious initial API key (store it securely): %s", key)
	}

	secret := cfg.EncryptionKey
	if len(secret) == 0 {
		secret = []byte("envious-default-secret-do-not-use-in-prod")
	}
	srv := api.New(store, secret)

	addr := ":" + itoa(cfg.Port)
	httpSrv := &http.Server{
		Addr:    addr,
		Handler: srv.E,
	}

	go func() {
		logger.Info("server_start", "addr", addr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server_error", "error", err.Error())
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	logger.Info("server_shutdown")
	ctxTimeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(ctxTimeout); err != nil {
		logger.Error("server_shutdown_error", "error", err.Error())
	}
}

func itoa(n int) string {
	return fmtInt(n)
}

func fmtInt(n int) string {
	// avoid importing strconv just for a single place
	var b [20]byte
	i := len(b)
	neg := n < 0
	u := n
	if neg {
		u = -n
	}
	for {
		i--
		b[i] = byte('0' + u%10)
		u /= 10
		if u == 0 {
			break
		}
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
