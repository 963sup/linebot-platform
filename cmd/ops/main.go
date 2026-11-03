package main

import (
	"fmt"
	"os"

	"github.com/963sup/linebot-platform/internal/adapter/inbound/cli"
)

func main() {
	if err := cli.Run(os.Args[1:], os.Getenv); err != nil {
		fmt.Fprintln(os.Stderr, "ops:", err)
		os.Exit(1)
	}
}
