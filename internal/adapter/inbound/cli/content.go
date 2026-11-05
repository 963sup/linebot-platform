package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	mediatransfer "github.com/963sup/linebot-platform/internal/application/usecases/media-transfer"
)

type mediaTransferRuntime interface {
	MessageContent(context.Context, string, mediatransfer.ContentVariant) (mediatransfer.BinaryContent, error)
	MessageContentStatus(context.Context, string) (mediatransfer.PreparationStatus, error)
}

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
	if contentRuntimeFactory == nil {
		return errors.New("content runtime is not configured")
	}
	r, err := contentRuntimeFactory(getenv(channelTokenEnv))
	if err != nil {
		return err
	}
	defer r.Close()
	media, ok := any(r).(mediaTransferRuntime)
	if !ok {
		return errors.New("content runtime does not implement media-transfer")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	if *kind == "transcoding" {
		status, inspectErr := media.MessageContentStatus(ctx, *id)
		if inspectErr != nil {
			return inspectErr
		}
		payload, encodeErr := json.Marshal(status)
		if encodeErr != nil {
			return fmt.Errorf("encode transcoding result: %w", encodeErr)
		}
		_, err = fmt.Fprintln(output, string(payload))
		return err
	}
	variant := mediatransfer.Original
	if *kind == "preview" {
		variant = mediatransfer.Preview
	} else if *kind != "content" {
		return errors.New("unsupported message content kind")
	}
	b, err := media.MessageContent(ctx, *id, variant)
	if err != nil {
		return err
	}
	return writePrivateFile(*out, b.Data)
}
