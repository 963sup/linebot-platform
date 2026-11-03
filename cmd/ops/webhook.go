package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"time"

	"github.com/963sup/linebot-platform/internal/application/usecases/channel"
	"github.com/963sup/linebot-platform/internal/bootstrap"
)

// Mutating webhook configuration is only permitted through a manually
// confirmed operator CLI. No such handler is exposed on the public webhook.
func runWebhookAdmin(args []string, getenv func(string) string, output io.Writer) error {
	flags := flag.NewFlagSet("webhook-admin", flag.ContinueOnError)
	action := flags.String("action", "", "set|test")
	endpoint := flags.String("endpoint", "", "HTTPS destination")
	apply := flags.Bool("apply", false, "explicitly authorize account configuration action")
	confirm := flags.String("confirm", "", "must match the action")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if len(flags.Args()) != 0 || !*apply || *confirm != *action || (*action != "set" && *action != "test") {
		return errors.New("webhook-admin requires --apply --confirm ACTION")
	}
	if getenv == nil || output == nil {
		return errors.New("CLI configuration required")
	}
	if err := channel.ValidateWebhookURL(*endpoint); err != nil {
		return err
	}
	rt, err := bootstrap.NewLINEOpsRuntime(getenv(bootstrap.KeyChannelToken))
	if err != nil {
		return err
	}
	defer rt.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	if *action == "set" {
		return rt.WebhookAdmin.Set(ctx, *endpoint)
	}
	result, err := rt.WebhookAdmin.Test(ctx, *endpoint)
	if err != nil {
		return err
	}
	return json.NewEncoder(output).Encode(result)
}
