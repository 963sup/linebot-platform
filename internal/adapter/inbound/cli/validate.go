package cli

import (
	"context"
	"errors"
	"flag"
	"time"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

func runValidate(args []string, getenv func(string) string) error {
	flags := flag.NewFlagSet("validate", flag.ContinueOnError)
	kind := flags.String("kind", "", "reply|push|multicast|narrowcast|broadcast")
	text := flags.String("text", "", "text message to validate without sending")
	confirm := flags.Bool("validate", false, "explicitly call LINE message validation")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if len(flags.Args()) != 0 || !*confirm || getenv == nil {
		return errors.New("validate requires --validate and no extra arguments")
	}
	if *text == "" {
		return errors.New("text is required")
	}
	if *kind != "reply" && *kind != "push" && *kind != "multicast" && *kind != "narrowcast" && *kind != "broadcast" {
		return errors.New("unsupported validation kind")
	}
	if validationRuntimeFactory == nil {
		return errors.New("validation runtime is not configured")
	}
	rt, err := validationRuntimeFactory(getenv(channelTokenEnv))
	if err != nil {
		return err
	}
	defer rt.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	return rt.Validate(ctx, *kind, []outbound.Message{{Kind: outbound.KindText, Text: *text}})
}
