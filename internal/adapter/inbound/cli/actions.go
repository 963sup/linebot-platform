package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"time"

	"github.com/963sup/linebot-platform/internal/application/usecases/messaging"
	"github.com/963sup/linebot-platform/internal/bootstrap"
)

func runActions(args []string, getenv func(string) string) error {
	flags := flag.NewFlagSet("actions", flag.ContinueOnError)
	action := flags.String("action", "", "leave-group|leave-room|loading|mark-read|mark-read-token")
	id := flags.String("id", "", "group or room ID for leave actions")
	request := flags.String("request", "", "JSON file containing the request; no secrets in CLI arguments")
	apply := flags.Bool("apply", false, "authorize a remote write")
	confirm := flags.String("confirm", "", "must exactly match --action")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if len(flags.Args()) != 0 || !*apply || *confirm == "" || *confirm != *action || getenv == nil {
		return errors.New("actions requires --apply --confirm ACTION and no extra arguments")
	}
	cmd := messaging.ConversationActionCommand{Action: *action, ID: *id}
	if *request != "" {
		payload, err := readPrivateBounded(*request, 256<<10)
		if err != nil {
			return err
		}
		cmd.Payload = json.RawMessage(payload)
	}
	if err := messaging.ValidateConversationAction(cmd); err != nil {
		return err
	}
	r, err := bootstrap.NewLINEOpsRuntime(getenv(bootstrap.KeyChannelToken))
	if err != nil {
		return err
	}
	defer r.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	return r.Actions.Execute(ctx, cmd)
}
