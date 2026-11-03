package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
	"github.com/963sup/linebot-platform/internal/bootstrap"
	"github.com/google/uuid"
)

func runCampaign(args []string, getenv func(string) string) error {
	flags := flag.NewFlagSet("campaign", flag.ContinueOnError)
	kind := flags.String("kind", "", "broadcast|multicast")
	recipients := flags.String("to", "", "comma-separated multicast user IDs")
	message := flags.String("text", "", "text")
	key := flags.String("retry-key", "", "stable UUID v4")
	send := flags.Bool("send", false, "explicitly enable a potentially billable send")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if len(flags.Args()) != 0 || !*send || getenv == nil {
		return errors.New("campaign requires --send and no extra arguments")
	}
	if (*kind != "broadcast" && *kind != "multicast") || strings.TrimSpace(*message) == "" {
		return errors.New("valid kind and text required")
	}
	var users []string
	if *kind == "broadcast" {
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
	rt, err := bootstrap.NewLINEOpsRuntime(getenv(bootstrap.KeyChannelToken))
	if err != nil {
		return err
	}
	defer rt.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	msgs := []outbound.Message{{Kind: outbound.KindText, Text: *message}}
	fmt.Fprintln(os.Stderr, "LINE campaign retry key:", *key)
	if *kind == "broadcast" {
		return rt.BulkMessaging.Broadcast(ctx, msgs, *key)
	}
	return rt.BulkMessaging.Multicast(ctx, users, msgs, *key)
}
