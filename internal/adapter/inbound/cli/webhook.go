package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"time"

	providerconfiguration "github.com/963sup/linebot-platform/internal/application/usecases/provider-configuration"
)

// Mutating webhook configuration is only permitted through a manually
// confirmed operator CLI. No such handler is exposed on the public webhook.
func runWebhookAdmin(args []string, getenv func(string) string, output io.Writer, dependencies ...Dependencies) error {
	deps := dependency(dependencies)
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
	if err := providerconfiguration.ValidateWebhookEndpoint(*endpoint); err != nil {
		return err
	}
	if deps.WebhookRuntime == nil {
		return errors.New("webhook runtime is not configured")
	}
	rt, err := deps.WebhookRuntime(getenv(channelTokenEnv))
	if err != nil {
		return err
	}
	defer rt.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	if *action == "set" {
		return rt.ConfigureWebhook(ctx, *endpoint)
	}
	result, err := rt.TestWebhook(ctx, *endpoint)
	if err != nil {
		return err
	}
	return json.NewEncoder(output).Encode(result)
}
