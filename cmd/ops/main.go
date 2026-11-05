package main

import (
	"fmt"
	"os"

	"github.com/963sup/linebot-platform/internal/bootstrap"
)

func main() {
	ctx, stop := bootstrap.SignalContext()
	defer stop()
	if err := bootstrap.RunOps(ctx, os.Args[1:], os.Getenv); err != nil {
		fmt.Fprintln(os.Stderr, "ops:", err)
		os.Exit(1)
	}
}
