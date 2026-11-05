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

type providerInspectionRuntime interface {
	providerconfiguration.ChannelReader
	insightsreporting.DeliveryReader
}

func runInspect(args []string, getenv func(string) string, output io.Writer, dependencies ...Dependencies) error {
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
	runtime, err := deps.ChannelInspectionRuntime(getenv(channelTokenEnv))
	if err != nil {
		return err
	}
	defer runtime.Close()
	reader, ok := any(runtime).(providerInspectionRuntime)
	if !ok {
		return errors.New("channel inspection runtime does not implement provider-configuration and insights-reporting")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	var data any
	switch *kind {
	case "bot-info":
		data, err = reader.BotInfo(ctx)
	case "webhook-endpoint":
		data, err = reader.WebhookEndpoint(ctx)
	case "message-quota":
		data, err = reader.MessageQuota(ctx)
	case "message-quota-consumption":
		data, err = reader.MessageQuotaConsumption(ctx)
	case "aggregation-usage":
		data, err = reader.AggregationUnitUsage(ctx)
	case "aggregation-names":
		data, err = reader.AggregationUnitNames(ctx, fmt.Sprint(*limit), *start)
	case "narrowcast-progress":
		data, err = reader.NarrowcastProgress(ctx, *requestID)
	case "delivery-reply", "delivery-push", "delivery-multicast", "delivery-broadcast":
		day, parseErr := time.Parse("20060102", *date)
		if parseErr != nil {
			return errors.New("delivery date must be valid YYYYMMDD")
		}
		query := insightsreporting.DailyQuery{Day: day}
		switch *kind {
		case "delivery-reply":
			data, err = reader.DeliveryReply(ctx, query)
		case "delivery-push":
			data, err = reader.DeliveryPush(ctx, query)
		case "delivery-multicast":
			data, err = reader.DeliveryMulticast(ctx, query)
		default:
			data, err = reader.DeliveryBroadcast(ctx, query)
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
