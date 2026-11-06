package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/963sup/linebot-platform/internal/bootstrap"
)

func main() {
	cfg, err := bootstrap.LoadMiniAppConfig(os.Getenv)
	if err != nil {
		fmt.Fprintln(os.Stderr, "invalid configuration")
		os.Exit(1)
	}
	logger := bootstrap.NewLogger(cfg.Log, os.Stdout)
	runtime, err := bootstrap.NewMiniAppRuntime(cfg, logger)
	if err != nil {
		logger.Error("failed to configure Mini App runtime")
		os.Exit(1)
	}
	defer func() {
		if closeErr := runtime.Close(); closeErr != nil {
			logger.Error("Mini App runtime close failed", "error", closeErr)
		}
	}()
	server := &http.Server{
		Addr: ":" + cfg.Port, Handler: runtime.Handler,
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second,
		WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second,
	}
	ctx, stop := bootstrap.SignalContext()
	defer stop()
	logger.Info("Mini App service starting", "port", cfg.Port, "profile", string(cfg.Environment))
	if err := bootstrap.Serve(ctx, server, logger); err != nil {
		logger.Error("Mini App service stopped unexpectedly")
		if closeErr := runtime.Close(); closeErr != nil {
			logger.Error("Mini App runtime close failed", "error", closeErr)
		}
		os.Exit(1)
	}
}
