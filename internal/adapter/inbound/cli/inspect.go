package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	insightsreporting "github.com/963sup/linebot-platform/internal/application/usecases/insights-reporting"
	providerconfiguration "github.com/963sup/linebot-platform/internal/application/usecases/provider-configuration"
)

func runInspect(parent context.Context, args []string, getenv func(string) string, output io.Writer, dependencies ...Dependencies) (err error) {
	deps := dependency(dependencies)
	flags := flag.NewFlagSet("inspect", flag.ContinueOnError)
	kind := flags.String("kind", "", "explicitly whitelisted LINE read-only report")
	date := flags.String("date", "", "delivery date YYYYMMDD")
	requestID := flags.String("request-id", "", "narrowcast progress request ID")
	start := flags.String("start", "", "aggregation page cursor")
	limit := flags.Int("limit", 0, "aggregation names page size 1-100 (0 = API default)")
	confirm := flags.Bool("read", false, "explicitly authorize a read-only LINE API call")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if len(flags.Args()) != 0 || !*confirm {
		return errors.New("inspect requires --read and no extra arguments")
	}
	if getenv == nil || output == nil {
		return errors.New("missing CLI configuration")
	}
	if deps.ChannelInspectionRuntime == nil {
		return errors.New("channel inspection runtime is not configured")
	}
	runtime, err := deps.ChannelInspectionRuntime(getenv(messagingAPIChannelAccessTokenEnv))
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, runtime.Close()) }()
	ctx, cancel := context.WithTimeout(parent, 12*time.Second)
	defer cancel()
	var data any
	switch *kind {
	case "bot-info":
		data, err = runtime.BotInfo(ctx)
	case "webhook-endpoint":
		data, err = runtime.WebhookEndpoint(ctx)
	case "message-quota":
		data, err = runtime.MessageQuota(ctx)
	case "message-quota-consumption":
		data, err = runtime.MessageQuotaConsumption(ctx)
	case "aggregation-usage":
		data, err = runtime.AggregationUnitUsage(ctx)
	case "aggregation-names":
		data, err = runtime.AggregationUnitNames(ctx, fmt.Sprint(*limit), *start)
	case "narrowcast-progress":
		data, err = runtime.NarrowcastProgress(ctx, *requestID)
	case "delivery-reply", "delivery-push", "delivery-multicast", "delivery-broadcast":
		day, parseErr := time.Parse("20060102", *date)
		if parseErr != nil {
			return errors.New("delivery date must be valid YYYYMMDD")
		}
		query := insightsreporting.DailyQuery{Day: day}
		switch *kind {
		case "delivery-reply":
			data, err = runtime.DeliveryReply(ctx, query)
		case "delivery-push":
			data, err = runtime.DeliveryPush(ctx, query)
		case "delivery-multicast":
			data, err = runtime.DeliveryMulticast(ctx, query)
		default:
			data, err = runtime.DeliveryBroadcast(ctx, query)
		}
	default:
		return errors.New("unsupported channel inspection kind")
	}
	if err != nil {
		return err
	}
	if snapshot, ok := data.(providerconfiguration.ChannelSnapshot); ok {
		data = snapshot.Value
	}
	payload, err := json.Marshal(data)
	if err != nil || string(payload) == "null" {
		return fmt.Errorf("LINE report is empty")
	}
	_, err = fmt.Fprintln(output, string(payload))
	return err
}
