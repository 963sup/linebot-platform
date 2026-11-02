package main

import (
	"context"
	"crypto/rand"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/963sup/linebot-platform/internal/adapter/outbound/line"
	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
	"github.com/963sup/linebot-platform/internal/application/usecases/messaging"
	lineinfra "github.com/963sup/linebot-platform/internal/infrastructure/line"
)

func uuidv4() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
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
	}
	client, err := lineinfra.NewMessagingClient(getenv("LINE_CHANNEL_ACCESS_TOKEN"))
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
