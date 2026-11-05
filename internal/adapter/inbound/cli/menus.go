package cli

import (
	"context"
	"errors"
	"flag"
	"io"
	"time"

	experiencepresentation "github.com/963sup/linebot-platform/internal/application/usecases/experience-presentation"
)

func runMenus(args []string, getenv func(string) string, output io.Writer, dependencies ...Dependencies) error {
	deps := dependency(dependencies)
	flags := flag.NewFlagSet("menus", flag.ContinueOnError)
	kind := flags.String("kind", "", "list|menu|default|alias-list|alias|user|batch-progress")
	id := flags.String("id", "", "requested menu ID, alias ID, user ID, or request ID")
	read := flags.Bool("read", false, "authorize an outbound read-only LINE request")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if len(flags.Args()) != 0 || !*read {
		return errors.New("menus requires --read with no extra arguments")
	}
	if getenv == nil || output == nil {
		return errors.New("missing CLI configuration")
	}
	if deps.MenusRuntime == nil {
		return errors.New("menus runtime is not configured")
	}
	rt, err := deps.MenusRuntime(getenv(channelTokenEnv))
	if err != nil {
		return err
	}
	defer rt.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	var result any
	switch *kind {
	case "list":
		if *id != "" {
			return errors.New("rich menu ID not accepted for this query")
		}
		result, err = rt.List(ctx)
	case "menu":
		result, err = rt.Get(ctx, experiencepresentation.ProviderSurfaceID(*id))
	case "default":
		if *id != "" {
			return errors.New("rich menu ID not accepted for this query")
		}
		result, err = rt.Default(ctx)
	case "alias-list":
		if *id != "" {
			return errors.New("rich menu ID not accepted for this query")
		}
		result, err = rt.ListAliases(ctx)
	case "alias":
		result, err = rt.GetAlias(ctx, experiencepresentation.AliasID(*id))
	case "user":
		result, err = rt.ForRecipient(ctx, experiencepresentation.RecipientID(*id))
	case "batch-progress":
		result, err = rt.BatchProgress(ctx, *id)
	default:
		return errors.New("unsupported rich menu query kind")
	}
	if err != nil {
		return err
	}
	return writeJSON(output, result)
}
