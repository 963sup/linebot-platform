package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/963sup/linebot-platform/internal/bootstrap"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := bootstrap.LoadConfig(os.Getenv)
	if err != nil {
		logger.Error("invalid configuration")
		os.Exit(1)
	}
	router, err := bootstrap.NewWebhookRouter(cfg, logger)
	if err != nil {
		logger.Error("failed to configure HTTP router")
		os.Exit(1)
	}
	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	ctx, stop := bootstrap.SignalContext()
	defer stop()
	logger.Info("HTTP service starting", "port", cfg.Port, "line_configured", cfg.LiveLINE())
	if err := bootstrap.Serve(ctx, server, logger); err != nil {
		logger.Error("HTTP service stopped unexpectedly")
		os.Exit(1)
	}
}
