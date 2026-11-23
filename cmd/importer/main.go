package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"

	"github.com/mundo365/erp-importer-scansource/internal/adapters/postgres"
	"github.com/mundo365/erp-importer-scansource/internal/adapters/rabbitmq"
	"github.com/mundo365/erp-importer-scansource/internal/adapters/s3"
	"github.com/mundo365/erp-importer-scansource/internal/adapters/spreadsheet"
	"github.com/mundo365/erp-importer-scansource/internal/app"
	"github.com/mundo365/erp-importer-scansource/internal/platform/config"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(logger); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("importer stopped", "err", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	_ = godotenv.Load()
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	store, err := s3.NewStore(cfg.S3Endpoint, cfg.S3Region, cfg.S3AccessKey, cfg.S3SecretKey)
	if err != nil {
		return err
	}

	bus, err := rabbitmq.NewBus(cfg.RabbitMQURL, logger)
	if err != nil {
		return err
	}
	defer bus.Close()

	service := &app.Service{
		Files:   store,
		Catalog: postgres.NewRepository(pool),
		Events:  bus,
		Read:    spreadsheet.Read,
		Log:     logger,
	}

	health := chi.NewRouter()
	health.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		if err := pool.Ping(ctx); err != nil {
			http.Error(w, "unhealthy", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	server := &http.Server{Addr: cfg.HealthAddr, Handler: health}
	go func() {
		logger.Info("health listening", "addr", cfg.HealthAddr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("health server", "err", err)
		}
	}()

	err = bus.Consume(ctx, service.HandleFileReceived)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
	return err
}
