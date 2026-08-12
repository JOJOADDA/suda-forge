package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"suda-forge/adapters/runtimes/lxc"
	"suda-forge/internal/config"
	"suda-forge/internal/events"
	"suda-forge/internal/httpapi"
	"suda-forge/internal/lifecycle"
	"suda-forge/internal/postgres"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("database pool creation failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	if err := db.Ping(ctx); err != nil {
		logger.Error("database ping failed", "error", err)
		os.Exit(1)
	}

	runtimeProvider := lxc.New()
	if cfg.LXCBinary != "" && cfg.LXCBinary != "lxc" {
		runtimeProvider.CreateBinary = cfg.LXCBinary
	}
	projects := postgres.Projects{DB: db}
	lifecycleService := lifecycle.Service{Projects: projects, Runtime: runtimeProvider, Now: time.Now}
	api := httpapi.Server{Projects: projects, Lifecycle: lifecycleService, Events: events.NewBus()}

	server := &http.Server{Addr: cfg.HTTPAddr, Handler: api.Handler(), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		logger.Info("server_started", "addr", cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server_failed", "error", err)
			stop()
		}
	}()
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
	logger.Info("server_stopped")
}
