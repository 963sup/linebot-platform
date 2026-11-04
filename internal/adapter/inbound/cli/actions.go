package cli

import (
	"context"
	"errors"
	"flag"
	"time"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
	"github.com/963sup/linebot-platform/internal/application/usecases/messaging"
)

func runActions(args []string, getenv func(string) string) error {
	flags := flag.NewFlagSet("actions", flag.ContinueOnError)
	action := flags.String("action", "", "leave-group|leave-room|loading|mark-read|mark-read-token")
	id := flags.String("id", "", "group or room ID for leave actions")
	request := flags.String("request", "", "JSON file containing the request; no secrets in CLI arguments")
	apply := flags.Bool("apply", false, "authorize a remote write")
	confirm := flags.String("confirm", "", "must exactly match --action")
	eligibilityReviewed := flags.Bool("eligibility-reviewed", false, "required for restricted mark-read operation")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if len(flags.Args()) != 0 || !*apply || *confirm == "" || *confirm != *action || getenv == nil {
		return errors.New("actions requires --apply --confirm ACTION and no extra arguments")
	}
	if *action == "mark-read" && !*eligibilityReviewed {
		return errors.New("mark-read requires --eligibility-reviewed")
	}
	cmd := messaging.ConversationActionCommand{Action: *action, ID: *id}
	if *request != "" {
		payload, err := readPrivateBounded(*request, 256<<10)
		if err != nil {
			return err
		}
		value, err := decodeLINEJSON(payload)
		if err != nil {
			return errors.New("request must be valid JSON")
		}
		cmd.Request = outbound.LINERequest{Value: value}
	}
	if err := messaging.ValidateConversationAction(cmd); err != nil {
		return err
	}
	if actionsRuntimeFactory == nil {
		return errors.New("actions runtime is not configured")
	}
	r, err := actionsRuntimeFactory(getenv(channelTokenEnv))
	if err != nil {
		return err
	}
	defer r.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	return r.Execute(ctx, cmd)
}
