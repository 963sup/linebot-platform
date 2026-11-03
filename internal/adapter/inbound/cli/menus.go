package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/963sup/linebot-platform/internal/application/usecases/engagement"
	"github.com/963sup/linebot-platform/internal/bootstrap"
)

func runMenus(args []string, getenv func(string) string, output io.Writer) error {
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
	q := engagement.RichMenuQuery{Kind: *kind, ID: *id}
	if err := engagement.ValidateRichMenuQuery(q); err != nil {
		return err
	}
	rt, err := bootstrap.NewLINEOpsRuntime(getenv(bootstrap.KeyChannelToken))
	if err != nil {
		return err
	}
	defer rt.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	result, err := rt.Menus.Inspect(ctx, q)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(output, string(result))
	return err
}
