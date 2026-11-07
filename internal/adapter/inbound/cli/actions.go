package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"time"

	conversationoperations "github.com/963sup/linebot-platform/internal/application/usecases/conversation-operations"
)

func runActions(parent context.Context, args []string, getenv func(string) string, dependencies ...Dependencies) (err error) {
	deps := dependency(dependencies)
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
	var payload map[string]any
	if *request != "" {
		data, err := readPrivateBounded(*request, 256<<10)
		if err != nil {
			return err
		}
		value, err := decodeLINEJSON(data)
		if err != nil {
			return errors.New("request must be valid JSON")
		}
		var ok bool
		payload, ok = value.(map[string]any)
		if !ok {
			return errors.New("request must be a JSON object")
		}
	}
	if deps.ActionsRuntime == nil {
		return errors.New("actions runtime is not configured")
	}
	r, err := deps.ActionsRuntime(getenv(messagingAPIChannelAccessTokenEnv))
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, r.Close()) }()
	ctx, cancel := context.WithTimeout(parent, 12*time.Second)
	defer cancel()
	switch *action {
	case "leave-group":
		return r.Leave(ctx, conversationoperations.Conversation{Kind: conversationoperations.Group, ID: *id})
	case "leave-room":
		return r.Leave(ctx, conversationoperations.Conversation{Kind: conversationoperations.Room, ID: *id})
	case "loading":
		chatID, ok := stringField(payload, "chatId")
		seconds, secondsOK := numberField(payload, "loadingSeconds")
		if !ok || !secondsOK {
			return errors.New("loading request requires chatId and loadingSeconds")
		}
		return r.ShowLoading(ctx, conversationoperations.LoadingIndicator{ChatID: chatID, Duration: time.Duration(seconds) * time.Second})
	case "mark-read":
		userID, ok := stringField(payload, "userId")
		if !ok {
			return errors.New("mark-read request requires userId")
		}
		return r.MarkRead(ctx, conversationoperations.ReadTarget{UserID: userID})
	case "mark-read-token":
		token, ok := stringField(payload, "markAsReadToken")
		if !ok {
			return errors.New("mark-read-token request requires markAsReadToken")
		}
		return r.MarkReadByToken(ctx, conversationoperations.ReadToken{Token: token})
	default:
		return fmt.Errorf("unsupported conversation action %q", *action)
	}
}

func stringField(payload map[string]any, name string) (string, bool) {
	value, ok := payload[name].(string)
	return value, ok && value != ""
}

func numberField(payload map[string]any, name string) (int64, bool) {
	switch value := payload[name].(type) {
	case json.Number:
		parsed, err := value.Int64()
		return parsed, err == nil
	case int64:
		return value, true
	case float64:
		return int64(value), value == float64(int64(value))
	default:
		return 0, false
	}
}
