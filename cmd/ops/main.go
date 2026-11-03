package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"

	"github.com/963sup/linebot-platform/internal/adapter/outbound/line"
	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
	"github.com/963sup/linebot-platform/internal/application/usecases/messaging"
	"github.com/963sup/linebot-platform/internal/infrastructure/httpclient"
	lineinfra "github.com/963sup/linebot-platform/internal/infrastructure/line"
)

func uuidv4() (string, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("generate UUID v4: %w", err)
	}
	return id.String(), nil
}

func run(args []string, getenv func(string) string) error {
	if len(args) == 0 || args[0] != "push" {
		return errors.New("usage: ops push --to ID --text MESSAGE [--send] [--retry-key UUID]")
	}
	fs := flag.NewFlagSet("push", flag.ContinueOnError)
	to := fs.String("to", "", "LINE recipient ID")
	text := fs.String("text", "", "text message")
	title := fs.String("flex-title", "", "Flex title (instead of text)")
	body := fs.String("flex-body", "", "Flex body")
	alt := fs.String("alt", "LINE 通知", "Flex alternative text")
	send := fs.Bool("send", false, "explicit permission to send to LINE")
	key := fs.String("retry-key", "", "stable UUID, reuse for uncertain retries")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if !*send {
		return errors.New("dry run only: specify --send to perform an external push")
	}
	if *to == "" {
		return errors.New("--to is required")
	}
	var message outbound.Message
	if *title != "" || *body != "" {
		if *text != "" {
			return errors.New("choose either --text or --flex-*")
		}
		message = outbound.Message{Kind: outbound.KindFlex, Flex: outbound.FlexCard{
			AltText: *alt, Title: *title, Body: *body,
		}}
	} else {
		message = outbound.Message{Kind: outbound.KindText, Text: *text}
	}
	if err := message.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(*key) == "" {
		var err error
		*key, err = uuidv4()
		if err != nil {
			return err
		}
	} else {
		parsed, err := uuid.Parse(*key)
		if err != nil || parsed.Version() != 4 {
			return errors.New("--retry-key must be a valid UUID v4")
		}
		*key = parsed.String()
	}
	managed, err := httpclient.New(httpclient.Config{Timeout: httpclient.DefaultTimeout}, nil)
	if err != nil {
		return err
	}
	defer managed.Close()
	client, err := lineinfra.NewMessagingClientWithHTTPClient(getenv("LINE_CHANNEL_ACCESS_TOKEN"), managed.Client)
	if err != nil {
		return err
	}
	sender, err := line.NewSender(client)
	if err != nil {
		return err
	}
	service, err := messaging.NewService(sender)
	if err != nil {
		return err
	}
	// Printed before sending so an uncertain result can be retried with the same key.
	fmt.Fprintln(os.Stderr, "LINE push retry key:", *key)
	return service.Push(context.Background(), *to, []outbound.Message{message}, *key)
}

func main() {
	if err := run(os.Args[1:], os.Getenv); err != nil {
		fmt.Fprintln(os.Stderr, "ops:", err)
		os.Exit(1)
	}
}
