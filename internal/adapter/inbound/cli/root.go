package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	messagingdelivery "github.com/963sup/linebot-platform/internal/application/usecases/messaging-delivery"
)

const (
	messagingAPIChannelAccessTokenEnv = "LINE_MESSAGING_API_CHANNEL_ACCESS_TOKEN"
	pushCommandTimeout                = 12 * time.Second
)

func uuidv4() (string, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("generate UUID v4: %w", err)
	}
	return id.String(), nil
}

func runWithContext(ctx context.Context, args []string, getenv func(string) string, dependencies ...Dependencies) (err error) {
	if ctx == nil {
		return errors.New("CLI context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	deps := dependency(dependencies)
	if len(args) > 0 && args[0] == "channel-tokens" {
		return runChannelTokens(ctx, args[1:], deps)
	}
	if len(args) > 0 && args[0] == "audience-definitions" {
		return runAudienceDefinitions(ctx, args[1:], getenv, os.Stdout, deps)
	}
	if len(args) > 0 && args[0] == "interaction-surfaces" {
		return runInteractionSurfaces(ctx, args[1:], getenv, os.Stdout, deps)
	}
	if len(args) > 0 && args[0] == "promotions" {
		return runPromotions(ctx, args[1:], getenv, os.Stdout, deps)
	}
	if len(args) > 0 && args[0] == "channel-connections" {
		return runChannelConnections(ctx, args[1:], os.Stdout, deps)
	}
	if len(args) > 0 && args[0] == "actions" {
		return runActions(ctx, args[1:], getenv, deps)
	}
	if len(args) > 0 && args[0] == "link-token" {
		return runLinkToken(ctx, args[1:], getenv, deps)
	}
	if len(args) > 0 && args[0] == "eligibility" {
		return runEligibility(ctx, args[1:], getenv, os.Stdout, deps)
	}
	if len(args) > 0 && args[0] == "menu-image" {
		return runMenuImage(ctx, args[1:], getenv, deps)
	}
	if len(args) > 0 && args[0] == "events" {
		return runEvents(ctx, args[1:], getenv, deps)
	}
	if len(args) > 0 && args[0] == "profile" {
		return runProfile(ctx, args[1:], getenv, os.Stdout, deps)
	}
	if len(args) > 0 && args[0] == "inspect" {
		return runInspect(ctx, args[1:], getenv, os.Stdout, deps)
	}
	if len(args) > 0 && args[0] == "menus" {
		return runMenus(ctx, args[1:], getenv, os.Stdout, deps)
	}
	if len(args) > 0 && args[0] == "contacts" {
		return runContacts(ctx, args[1:], getenv, os.Stdout, deps)
	}
	if len(args) > 0 && args[0] == "content" {
		return runContent(ctx, args[1:], getenv, os.Stdout, deps)
	}
	if len(args) > 0 && args[0] == "insights" {
		return runInsights(ctx, args[1:], getenv, os.Stdout, deps)
	}
	if len(args) > 0 && args[0] == "coupons" {
		return runCoupons(ctx, args[1:], getenv, os.Stdout, deps)
	}
	if len(args) > 0 && args[0] == "memberships" {
		return runMemberships(ctx, args[1:], getenv, os.Stdout, deps)
	}
	if len(args) > 0 && args[0] == "audiences" {
		return runAudiences(ctx, args[1:], getenv, os.Stdout, deps)
	}
	if len(args) > 0 && args[0] == "manage-menu" {
		return runManageMenu(ctx, args[1:], getenv, deps)
	}
	if len(args) > 0 && args[0] == "campaign" {
		return runCampaign(ctx, args[1:], getenv, deps)
	}
	if len(args) > 0 && args[0] == "validate" {
		return runValidate(ctx, args[1:], getenv, deps)
	}
	if len(args) > 0 && args[0] == "webhook-admin" {
		return runWebhookAdmin(ctx, args[1:], getenv, os.Stdout, deps)
	}
	if len(args) == 0 || args[0] != "push" {
		return errors.New("usage: ops push --to ID --text MESSAGE [--send] [--retry-key UUID] | ops events --inspect --project PROJECT | ops profile --user ID --lookup")
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
	var message messagingdelivery.Message
	if *title != "" || *body != "" {
		if *text != "" {
			return errors.New("choose either --text or --flex-*")
		}
		message = messagingdelivery.Message{Kind: messagingdelivery.KindFlex, Flex: messagingdelivery.FlexCard{
			AltText: *alt, Title: *title, Body: *body,
		}}
	} else {
		message = messagingdelivery.Message{Kind: messagingdelivery.KindText, Text: *text}
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
	if deps.PushRuntime == nil {
		return errors.New("push runtime is not configured")
	}
	runtime, err := deps.PushRuntime(getenv(messagingAPIChannelAccessTokenEnv))
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, runtime.Close()) }()
	// Printed before sending so an uncertain result can be retried with the same key.
	fmt.Fprintln(os.Stderr, "LINE push retry key:", *key)
	commandCtx, cancel := context.WithTimeout(ctx, pushCommandTimeout)
	defer cancel()
	return runtime.Push(commandCtx, *to, []messagingdelivery.Message{message}, *key)
}

// Run dispatches operator commands without leaking CLI logic into cmd/ops.
func Run(ctx context.Context, args []string, getenv func(string) string, deps Dependencies) error {
	return runWithContext(ctx, args, getenv, deps)
}
