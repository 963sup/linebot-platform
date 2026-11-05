package cli

import (
	"context"
	"errors"
	"flag"
	"io"
	"time"

	insightsreporting "github.com/963sup/linebot-platform/internal/application/usecases/insights-reporting"
	messagingdelivery "github.com/963sup/linebot-platform/internal/application/usecases/messaging-delivery"
	"github.com/963sup/linebot-platform/internal/application/usecases/promotion"
)

func runEligibility(parent context.Context, args []string, getenv func(string) string, output io.Writer, dependencies ...Dependencies) error {
	deps := dependency(dependencies)
	flags := flag.NewFlagSet("eligibility", flag.ContinueOnError)
	action := flags.String("action", "", "push-phone|push-template-phone|pnp-stats|pnp-template-stats|create-coupon|close-coupon")
	id := flags.String("id", "", "YYYYMMDD for statistics, coupon ID for close")
	request := flags.String("request", "", "JSON payload file")
	apply := flags.Bool("apply", false, "authorize remote mutation")
	read := flags.Bool("read", false, "authorize query")
	confirm := flags.String("confirm", "", "must equal action")
	reviewed := flags.Bool("eligibility-reviewed", false, "acknowledge entitlement review")
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
	var document any
	if *request != "" {
		data, err := readPrivateBounded(*request, 256<<10)
		if err != nil {
			return err
		}
		document, err = decodeLINEJSON(data)
		if err != nil {
			return errors.New("request must be valid JSON")
		}
	}
	if deps.EligibilityRuntime == nil {
		return errors.New("eligibility runtime is not configured")
	}
	runtime, err := deps.EligibilityRuntime(getenv(channelTokenEnv))
	if err != nil {
		return err
	}
	defer runtime.Close()
	ctx, cancel := context.WithTimeout(parent, 12*time.Second)
	defer cancel()
	var result any
	switch *action {
	case "push-phone":
		err = runtime.PushMessagesByPhone(ctx, messagingdelivery.PhoneDeliveryRequest{Payload: document})
	case "push-template-phone":
		err = runtime.PushTemplatedMessagesByPhone(ctx, messagingdelivery.PhoneDeliveryRequest{Payload: document})
	case "pnp-stats", "pnp-template-stats":
		day, parseErr := time.Parse("20060102", *id)
		if parseErr != nil {
			return errors.New("statistics ID must be YYYYMMDD")
		}
		if *action == "pnp-stats" {
			result, err = runtime.PNPMessageStatistics(ctx, insightsreporting.DailyQuery{Day: day})
		} else {
			result, err = runtime.PNPTemplatedMessageStatistics(ctx, insightsreporting.DailyQuery{Day: day})
		}
	case "create-coupon":
		result, err = runtime.CreateCouponDocument(ctx, promotion.ProviderCouponDocument{Value: document})
	case "close-coupon":
		err = runtime.CloseCoupon(ctx, promotion.CouponID(*id))
	default:
		return errors.New("unsupported eligibility action")
	}
	if err != nil {
		return err
	}
	if result != nil {
		return writeJSON(output, result)
	}
	return nil
}
