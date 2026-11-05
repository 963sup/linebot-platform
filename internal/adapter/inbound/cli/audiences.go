package cli

import (
	"context"
	"errors"
	"flag"
	"io"
	"time"

	"github.com/963sup/linebot-platform/internal/application/usecases/audience"
)

func runAudiences(parent context.Context, args []string, getenv func(string) string, output io.Writer, dependencies ...Dependencies) error {
	deps := dependency(dependencies)
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
	if deps.AudienceRuntime == nil {
		return errors.New("audience runtime is not configured")
	}
	rt, err := deps.AudienceRuntime(getenv(channelTokenEnv))
	if err != nil {
		return err
	}
	defer rt.Close()
	ctx, cancel := context.WithTimeout(parent, 12*time.Second)
	defer cancel()
	var result any
	switch *kind {
	case "group", "shared-group":
		if *id <= 0 {
			return errors.New("audience ID must be positive")
		}
		result, err = rt.Get(ctx, audience.ProviderAudienceID(*id), *kind == "shared-group")
	case "list", "shared-list":
		if *id != 0 || *page < 1 || *size < 1 || *size > audience.MaxPageSize {
			return errors.New("audience page must be positive and size must be 1-100")
		}
		result, err = rt.List(ctx, audience.PageRequest{Number: *page, Size: *size}, *kind == "shared-list")
	default:
		return errors.New("unsupported audience query kind")
	}
	if err != nil {
		return err
	}
	return writeJSON(output, result)
}
