package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/963sup/linebot-platform/internal/application/usecases/engagement"
	"github.com/963sup/linebot-platform/internal/bootstrap"
)

func runAudienceAdmin(args []string, getenv func(string) string, output io.Writer) error {
	flags := flag.NewFlagSet("audience-admin", flag.ContinueOnError)
	action := flags.String("action", "", "create|append|create-click|create-impression|update-description|delete|create-file|append-file")
	id := flags.Int64("audience-id", 0, "existing audience group ID for edit/delete/file append")
	request := flags.String("request", "", "JSON request file, used for JSON operations")
	file := flags.String("file", "", "file of user IDs to upload, maximum 20 MiB")
	description := flags.String("description", "", "audience description for create-file")
	uploadDescription := flags.String("upload-description", "", "description of this file upload")
	ifa := flags.Bool("ifa", false, "explicit IFA audience choice")
	apply := flags.Bool("apply", false, "authorize mutation")
	confirm := flags.String("confirm", "", "must match action")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if len(flags.Args()) != 0 || getenv == nil || output == nil ||
		!*apply || *confirm == "" || *confirm != *action {
		return errors.New("audience-admin requires --apply --confirm ACTION")
	}
	cmd := engagement.AudienceCommand{
		Action: *action, ID: *id, Description: *description,
		UploadDescription: *uploadDescription, IFAAudience: *ifa,
	}
	if *request != "" {
		b, err := readPrivateBounded(*request, 256<<10)
		if err != nil {
			return err
		}
		cmd.Request = json.RawMessage(b)
	}
	if *file != "" {
		b, err := readPrivateBounded(*file, 20<<20)
		if err != nil {
			return err
		}
		cmd.File = b
	}
	if err := engagement.ValidateAudienceCommand(cmd); err != nil {
		return err
	}
	r, err := bootstrap.NewLINEOpsRuntime(getenv(bootstrap.KeyChannelToken))
	if err != nil {
		return err
	}
	defer r.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	result, err := r.AudienceManager.Execute(ctx, cmd)
	if err != nil {
		return err
	}
	if len(result) > 0 {
		_, err = fmt.Fprintln(output, string(result))
	}
	return err
}
