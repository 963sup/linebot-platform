package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/963sup/linebot-platform/internal/bootstrap"
)

func main() {
	cfg, err := bootstrap.LoadWorkerConfig(os.Getenv)
	if err != nil {
		fmt.Fprintln(os.Stderr, "worker configuration invalid") // no secret values
		os.Exit(1)
	}
	logger := bootstrap.NewLogger(cfg.Log, os.Stdout)
	startup, cancel := bootstrap.WorkerStartupContext()
	rt, err := bootstrap.NewWorkerRuntime(startup, cfg, logger)
	cancel()
	if err != nil {
		logger.Error("worker initialization failed")
		os.Exit(1)
	}
	defer func() {
		if closeErr := rt.Close(); closeErr != nil {
			logger.Error("worker runtime close failed", "error", closeErr)
		}
	}()
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
	logger.Info("private worker starting", "port", cfg.Port, "profile", string(cfg.Environment))
	if err := bootstrap.Serve(ctx, server, logger); err != nil {
		logger.Error("worker stopped unexpectedly")
		if closeErr := rt.Close(); closeErr != nil {
			logger.Error("worker runtime close failed", "error", closeErr)
		}
		os.Exit(1)
	}
}
