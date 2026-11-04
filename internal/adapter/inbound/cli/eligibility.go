package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
	"github.com/963sup/linebot-platform/internal/application/usecases/engagement"
)

// Eligibility-gated APIs must never be accessible from the public webhook.
func runEligibility(args []string, getenv func(string) string, output io.Writer) error {
	flags := flag.NewFlagSet("eligibility", flag.ContinueOnError)
	action := flags.String("action", "", "push-phone|push-template-phone|pnp-stats|pnp-template-stats|create-coupon|close-coupon")
	id := flags.String("id", "", "YYYYMMDD for statistics, coupon ID for close")
	request := flags.String("request", "", "JSON payload file; never pass phone numbers as CLI flags")
	apply := flags.Bool("apply", false, "authorize remote mutation")
	read := flags.Bool("read", false, "authorize query")
	confirm := flags.String("confirm", "", "must equal the remote write action")
	reviewed := flags.Bool("eligibility-reviewed", false, "acknowledge channel/region entitlement review")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if len(flags.Args()) != 0 || !*reviewed || getenv == nil || output == nil {
		return errors.New("eligibility requires --eligibility-reviewed and no extra arguments")
	}
	isRead := *action == "pnp-stats" || *action == "pnp-template-stats"
	if isRead && (!*read || *apply || *confirm != "") {
		return errors.New("statistics require --read only")
	}
	if !isRead && (!*apply || *read || *confirm != *action) {
		return errors.New("mutations require --apply --confirm ACTION")
	}
	cmd := engagement.EligibilityCommand{Action: *action, ID: *id}
	if *request != "" {
		payload, err := readPrivateBounded(*request, 256<<10)
		if err != nil {
			return err
		}
		value, err := decodeLINEJSON(payload)
		if err != nil {
			return errors.New("request must be valid JSON")
		}
		cmd.Payload = outbound.LINERequest{Value: value}
	}
	if err := engagement.ValidateEligibilityCommand(cmd); err != nil {
		return err
	}
	if eligibilityRuntimeFactory == nil {
		return errors.New("eligibility runtime is not configured")
	}
	r, err := eligibilityRuntimeFactory(getenv(channelTokenEnv))
	if err != nil {
		return err
	}
	defer r.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	data, err := r.Execute(ctx, cmd)
	if err != nil {
		return err
	}
	if len(data) > 0 {
		_, err = fmt.Fprintln(output, string(data))
	}
	return err
}
