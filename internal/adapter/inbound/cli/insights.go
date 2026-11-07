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
)

func runInsights(parent context.Context, args []string, getenv func(string) string, output io.Writer, dependencies ...Dependencies) (err error) {
	deps := dependency(dependencies)
	flags := flag.NewFlagSet("insights", flag.ContinueOnError)
	kind := flags.String("kind", "", "read-only insights report kind")
	date := flags.String("date", "", "YYYYMMDD")
	requestID := flags.String("request-id", "", "LINE request ID")
	unit := flags.String("unit", "", "aggregation unit")
	menu := flags.String("rich-menu", "", "rich menu ID")
	from := flags.String("from", "", "inclusive YYYYMMDD")
	to := flags.String("to", "", "inclusive YYYYMMDD")
	read := flags.Bool("read", false, "authorize an outbound insights API query")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if !*read || len(flags.Args()) != 0 {
		return errors.New("insights requires --read and no extra arguments")
	}
	if getenv == nil || output == nil {
		return errors.New("missing CLI configuration")
	}
	if deps.InsightsRuntime == nil {
		return errors.New("insights runtime is not configured")
	}
	rt, err := deps.InsightsRuntime(getenv(messagingAPIChannelAccessTokenEnv))
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, rt.Close()) }()
	ctx, cancel := context.WithTimeout(parent, 12*time.Second)
	defer cancel()
	parseDay := func(value string) (time.Time, error) { return time.Parse("20060102", value) }
	var result any
	switch *kind {
	case "demographics":
		result, err = rt.Demographics(ctx)
	case "deliveries", "followers":
		day, parseErr := parseDay(*date)
		if parseErr != nil {
			return errors.New("insights date must be valid YYYYMMDD")
		}
		if *kind == "deliveries" {
			result, err = rt.Deliveries(ctx, insightsreporting.DailyQuery{Day: day})
		} else {
			result, err = rt.Followers(ctx, insightsreporting.DailyQuery{Day: day})
		}
	case "message-event":
		result, err = rt.MessageEvent(ctx, insightsreporting.MessageEventQuery{RequestID: *requestID})
	case "statistics-per-unit", "menu-summary", "menu-daily":
		fromDay, fromErr := parseDay(*from)
		toDay, toErr := parseDay(*to)
		if fromErr != nil || toErr != nil {
			return errors.New("insights date range must be valid YYYYMMDD and ordered")
		}
		if *kind == "statistics-per-unit" {
			result, err = rt.StatisticsPerUnit(ctx, insightsreporting.UnitStatisticsQuery{Unit: *unit, RangeQuery: insightsreporting.RangeQuery{From: fromDay, To: toDay}})
		} else {
			query := insightsreporting.PresentationStatisticsQuery{PresentationID: *menu, RangeQuery: insightsreporting.RangeQuery{From: fromDay, To: toDay}}
			if *kind == "menu-summary" {
				result, err = rt.PresentationSummary(ctx, query)
			} else {
				result, err = rt.PresentationDaily(ctx, query)
			}
		}
	default:
		return errors.New("unsupported insights inspection kind")
	}
	if err != nil {
		return err
	}
	payload, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("encode insights result: %w", err)
	}
	_, err = fmt.Fprintln(output, string(payload))
	return err
}
