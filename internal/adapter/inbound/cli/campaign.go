package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	messagingdelivery "github.com/963sup/linebot-platform/internal/application/usecases/messaging-delivery"
	"github.com/google/uuid"
)

func runCampaign(parent context.Context, args []string, getenv func(string) string, dependencies ...Dependencies) (err error) {
	deps := dependency(dependencies)
	flags := flag.NewFlagSet("campaign", flag.ContinueOnError)
	kind := flags.String("kind", "", "broadcast|multicast|narrowcast")
	recipients := flags.String("to", "", "comma-separated multicast user IDs")
	message := flags.String("text", "", "text")
	requestFile := flags.String("request", "", "Narrowcast request JSON file")
	key := flags.String("retry-key", "", "stable UUID v4")
	send := flags.Bool("send", false, "explicitly enable a potentially billable send")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if len(flags.Args()) != 0 || !*send || getenv == nil {
		return errors.New("campaign requires --send and no extra arguments")
	}
	if *kind != "broadcast" && *kind != "multicast" && *kind != "narrowcast" {
		return errors.New("valid campaign kind required")
	}
	var users []string
	var request messagingdelivery.NarrowcastRequest
	if *kind == "narrowcast" {
		if *recipients != "" || *message != "" || *requestFile == "" {
			return errors.New("narrowcast requires --request and does not accept --to or --text")
		}
		data, err := os.ReadFile(filepath.Clean(*requestFile))
		if err != nil {
			return fmt.Errorf("read narrowcast request: %w", err)
		}
		if !json.Valid(data) {
			return errors.New("narrowcast request must be valid JSON")
		}
		value, err := decodeLINEJSON(data)
		if err != nil {
			return errors.New("narrowcast request must be valid JSON")
		}
		request = messagingdelivery.NarrowcastRequest{Payload: value}
	} else if strings.TrimSpace(*message) == "" || *requestFile != "" {
		return errors.New("broadcast and multicast require --text and do not accept --request")
	} else if *kind == "broadcast" {
		if *recipients != "" {
			return errors.New("broadcast must not specify --to")
		}
	} else {
		if *recipients == "" {
			return errors.New("multicast requires --to")
		}
		users = strings.Split(*recipients, ",")
	}
	if *key == "" {
		value, err := uuidv4()
		if err != nil {
			return err
		}
		*key = value
	} else {
		id, err := uuid.Parse(*key)
		if err != nil || id.Version() != 4 {
			return errors.New("--retry-key must be UUID v4")
		}
		*key = id.String()
	}
	if deps.BulkMessagingRuntime == nil {
		return errors.New("bulk messaging runtime is not configured")
	}
	rt, err := deps.BulkMessagingRuntime(getenv(messagingAPIChannelAccessTokenEnv))
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, rt.Close()) }()
	ctx, cancel := context.WithTimeout(parent, 12*time.Second)
	defer cancel()
	msgs := []messagingdelivery.Message{{Kind: messagingdelivery.KindText, Text: *message}}
	fmt.Fprintln(os.Stderr, "LINE campaign retry key:", *key)
	if *kind == "narrowcast" {
		return rt.SendNarrowcast(ctx, request, *key)
	}
	if *kind == "broadcast" {
		return rt.Broadcast(ctx, msgs, *key)
	}
	return rt.Multicast(ctx, users, msgs, *key)
}
