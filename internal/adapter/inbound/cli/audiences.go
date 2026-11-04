package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/963sup/linebot-platform/internal/application/usecases/engagement"
)

func runAudiences(args []string, getenv func(string) string, output io.Writer) error {
	flags := flag.NewFlagSet("audiences", flag.ContinueOnError)
	kind := flags.String("kind", "", "group|shared-group|list|shared-list")
	id := flags.Int64("audience-id", 0, "audience group ID")
	page := flags.Int64("page", 1, "list page")
	size := flags.Int64("size", 20, "list size")
	read := flags.Bool("read", false, "authorize a read-only audience query")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if len(flags.Args()) != 0 || !*read {
		return errors.New("audiences requires --read and no extra arguments")
	}
	if getenv == nil || output == nil {
		return errors.New("missing CLI configuration")
	}
	q := engagement.AudienceQuery{Kind: *kind, ID: *id, Page: *page, Size: *size}
	if err := engagement.ValidateAudienceQuery(q); err != nil {
		return err
	}
	if audienceRuntimeFactory == nil {
		return errors.New("audience runtime is not configured")
	}
	rt, err := audienceRuntimeFactory(getenv(channelTokenEnv))
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
