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
	cfg, err := bootstrap.LoadWorkerConfig(os.Getenv)
	if err != nil {
		logger.Error("worker configuration invalid")
		os.Exit(1)
	}
	startup, cancel := bootstrap.WorkerStartupContext()
	rt, err := bootstrap.NewWorkerRuntime(startup, cfg, logger)
	cancel()
	if err != nil {
		logger.Error("worker initialization failed")
		os.Exit(1)
	}
	defer rt.Close()
	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           rt.Handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	ctx, stop := bootstrap.SignalContext()
	defer stop()
	logger.Info("private worker starting", "port", cfg.Port)
	if err := bootstrap.Serve(ctx, server, logger); err != nil {
		logger.Error("worker stopped unexpectedly")
		rt.Close()
		os.Exit(1)
	}
}
