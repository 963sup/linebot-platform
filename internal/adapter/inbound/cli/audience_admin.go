package cli

import (
	"context"
	"errors"
	"flag"
	"io"
	"time"

	"github.com/963sup/linebot-platform/internal/application/usecases/audience"
)

func runAudienceAdmin(args []string, getenv func(string) string, output io.Writer) error {
	flags := flag.NewFlagSet("audience-admin", flag.ContinueOnError)
	action := flags.String("action", "", "create|append|create-click|create-impression|update-description|delete|create-file|append-file")
	id := flags.Int64("audience-id", 0, "existing audience group ID")
	request := flags.String("request", "", "JSON request file")
	file := flags.String("file", "", "file of user IDs to upload, maximum 20 MiB")
	description := flags.String("description", "", "audience description")
	uploadDescription := flags.String("upload-description", "", "description of this file upload")
	ifa := flags.Bool("ifa", false, "explicit IFA audience choice")
	apply := flags.Bool("apply", false, "authorize mutation")
	confirm := flags.String("confirm", "", "must match action")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if len(flags.Args()) != 0 || getenv == nil || output == nil || !*apply || *confirm != *action {
		return errors.New("audience-admin requires --apply --confirm ACTION")
	}
	var document map[string]any
	if *request != "" {
		data, err := readPrivateBounded(*request, 256<<10)
		if err != nil {
			return err
		}
		value, err := decodeLINEJSON(data)
		if err != nil {
			return errors.New("request must be valid JSON")
		}
		var ok bool
		document, ok = value.(map[string]any)
		if !ok {
			return errors.New("request must be a JSON object")
		}
	}
	var data []byte
	if *file != "" {
		var err error
		data, err = readPrivateBounded(*file, audience.MaxAudienceFileSize)
		if err != nil {
			return err
		}
	}
	if audienceManagementRuntimeFactory == nil {
		return errors.New("audience management runtime is not configured")
	}
	runtime, err := audienceManagementRuntimeFactory(getenv(channelTokenEnv))
	if err != nil {
		return err
	}
	defer runtime.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	providerID := audience.ProviderAudienceID(*id)
	var result any
	switch *action {
	case "create":
		result, err = runtime.CreateGroup(ctx, audience.CreateGroupCommand{Description: textField(document, "description"), Recipients: audienceRecipients(document)})
	case "append":
		err = runtime.AppendRecipients(ctx, audience.AppendRecipientsCommand{ID: audience.ProviderAudienceID(audienceNumber(document, "audienceGroupId")), Recipients: audienceRecipients(document)})
	case "create-click", "create-impression":
		result, err = runtime.CreateInteractionGroup(ctx, audience.CreateInteractionGroupCommand{Description: textField(document, "description"), SourceRequestID: textField(document, "requestId")}, *action == "create-impression")
	case "update-description":
		err = runtime.UpdateDescription(ctx, providerID, textField(document, "description"))
	case "delete":
		err = runtime.Delete(ctx, providerID)
	case "create-file":
		result, err = runtime.CreateFromFile(ctx, audience.AudienceFile{Data: data, Description: *description, UploadDescription: *uploadDescription, IFAAudience: *ifa})
	case "append-file":
		err = runtime.AppendFromFile(ctx, providerID, audience.AudienceFile{Data: data, UploadDescription: *uploadDescription})
	default:
		return errors.New("unsupported audience action")
	}
	if err != nil {
		return err
	}
	if result != nil {
		return writeJSON(output, result)
	}
	return nil
}

func textField(object map[string]any, name string) string {
	value, _ := object[name].(string)
	return value
}
func audienceNumber(object map[string]any, name string) int64 {
	value, _ := numberField(object, name)
	return value
}
func audienceRecipients(object map[string]any) []string {
	raw, _ := object["audiences"].([]any)
	result := make([]string, 0, len(raw))
	for _, item := range raw {
		if entry, ok := item.(map[string]any); ok {
			if id := textField(entry, "id"); id != "" {
				result = append(result, id)
			}
		}
	}
	return result
}
