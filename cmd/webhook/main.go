package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/963sup/linebot-platform/internal/bootstrap"
)

func main() {
	cfg, err := bootstrap.LoadConfig(os.Getenv)
	if err != nil {
		fmt.Fprintln(os.Stderr, "invalid configuration") // no secret values
		os.Exit(1)
	}
	logger := bootstrap.NewLogger(cfg.Log, os.Stdout)
	runtime, err := bootstrap.NewWebhookRuntime(cfg, logger)
	if err != nil {
		logger.Error("failed to configure HTTP runtime")
		os.Exit(1)
	}
	defer runtime.Close()
	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           runtime.Handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	ctx, stop := bootstrap.SignalContext()
	defer stop()
	logger.Info("HTTP service starting", "port", cfg.Port, "profile", string(cfg.Environment), "mode", cfg.EventMode, "line_configured", cfg.LiveLINE())
	if err := bootstrap.Serve(ctx, server, logger); err != nil {
		logger.Error("HTTP service stopped unexpectedly")
		runtime.Close() // os.Exit does not run deferred cleanup.
		os.Exit(1)
	}
}
