package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/963sup/linebot-platform/internal/application/usecases/channel"
	"github.com/963sup/linebot-platform/internal/bootstrap"
)

func runInspect(args []string, getenv func(string) string, output io.Writer) error {
	flags := flag.NewFlagSet("inspect", flag.ContinueOnError)
	kind := flags.String("kind", "", "explicitly whitelisted LINE read-only report")
	date := flags.String("date", "", "delivery date YYYYMMDD")
	requestID := flags.String("request-id", "", "narrowcast progress request ID")
	start := flags.String("start", "", "aggregation page cursor")
	limit := flags.Int("limit", 0, "aggregation names page size 1-100 (0 = API default)")
	confirm := flags.Bool("read", false, "explicitly authorize a read-only LINE API call")
	if err := flags.Parse(args); err != nil { return err }
	if len(flags.Args()) != 0 || !*confirm {
		return errors.New("inspect requires --read and no extra arguments")
	}
	if getenv == nil || output == nil { return errors.New("missing CLI configuration") }
	query := channel.Inspection{Kind: *kind, Date: *date, RequestID: *requestID, Start: *start, Limit: *limit}
	// Validate kinds and inputs without making external connections.
	if err := channel.ValidateInspection(query); err != nil { return err }
	runtime, err := bootstrap.NewLINEOpsRuntime(getenv(bootstrap.KeyChannelToken))
	if err != nil { return err }
	defer runtime.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	data, err := runtime.Channel.Inspect(ctx, query)
	if err != nil { return err }
	if strings.TrimSpace(string(data)) == "" { return fmt.Errorf("LINE report is empty") }
	_, err = fmt.Fprintln(output, string(data))
	return err
}
