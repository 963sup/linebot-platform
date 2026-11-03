package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/963sup/linebot-platform/internal/application/usecases/contacts"
	"github.com/963sup/linebot-platform/internal/bootstrap"
)

func runContent(args []string, getenv func(string) string, output io.Writer) error {
	f := flag.NewFlagSet("content", flag.ContinueOnError)
	kind := f.String("kind", "", "content|preview|transcoding")
	id := f.String("message-id", "", "LINE message ID")
	out := f.String("out", "", "output file for binary content")
	read := f.Bool("read", false, "authorize content retrieval")
	if err := f.Parse(args); err != nil {
		return err
	}
	if !*read || len(f.Args()) != 0 || getenv == nil || output == nil {
		return errors.New("content requires --read and no extra arguments")
	}
	if (*kind == "content" || *kind == "preview") && *out == "" {
		return errors.New("binary content requires --out")
	}
	r, err := bootstrap.NewLINEOpsRuntime(getenv(bootstrap.KeyChannelToken))
	if err != nil {
		return err
	}
	defer r.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	b, j, err := r.Content.Inspect(ctx, contacts.ContentQuery{Kind: *kind, MessageID: *id})
	if err != nil {
		return err
	}
	if *kind == "transcoding" {
		_, err = fmt.Fprintln(output, string(j))
		return err
	}
	return os.WriteFile(filepath.Clean(*out), b.Data, 0o600)
}
