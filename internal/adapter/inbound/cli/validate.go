package cli

import (
	"context"
	"errors"
	"flag"
	"time"

	messagingdelivery "github.com/963sup/linebot-platform/internal/application/usecases/messaging-delivery"
)

func runValidate(parent context.Context, args []string, getenv func(string) string, dependencies ...Dependencies) (err error) {
	deps := dependency(dependencies)
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
	if deps.ValidationRuntime == nil {
		return errors.New("validation runtime is not configured")
	}
	rt, err := deps.ValidationRuntime(getenv(channelTokenEnv))
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, rt.Close()) }()
	ctx, cancel := context.WithTimeout(parent, 12*time.Second)
	defer cancel()
	_, err = rt.Validate(ctx, messagingdelivery.ValidationRequest{
		Delivery: messagingdelivery.DeliveryKind(*kind),
		Messages: []messagingdelivery.Message{{Kind: messagingdelivery.KindText, Text: *text}},
	})
	return err
}
