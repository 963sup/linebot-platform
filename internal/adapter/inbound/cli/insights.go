package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/963sup/linebot-platform/internal/application/usecases/insights"
)

func runInsights(args []string, getenv func(string) string, output io.Writer) error {
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
	q := insights.Query{Kind: *kind, Date: *date, RequestID: *requestID, Unit: *unit, RichMenu: *menu, From: *from, To: *to}
	if err := insights.ValidateQuery(q); err != nil {
		return err
	}
	if insightsRuntimeFactory == nil {
		return errors.New("insights runtime is not configured")
	}
	rt, err := insightsRuntimeFactory(getenv(channelTokenEnv))
	if err != nil {
		return err
	}
	defer rt.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	result, err := rt.Inspect(ctx, q)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(output, string(result))
	return err
}
