package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/963sup/linebot-platform/internal/application/usecases/engagement"
)

// All Rich Menu mutations demand an explicit action-specific confirmation.
// This command is local/operator-only and never exposed on a webhook route.
func runManageMenu(args []string, getenv func(string) string) error {
	flags := flag.NewFlagSet("manage-menu", flag.ContinueOnError)
	action := flags.String("action", "", "create|validate|create-alias|update-alias|bulk-link|bulk-unlink|batch|validate-batch|set-default|cancel-default|link-user|unlink-user|delete-menu|delete-alias")
	menu := flags.String("menu-id", "", "rich menu ID")
	user := flags.String("user", "", "LINE user ID")
	alias := flags.String("alias-id", "", "rich menu alias ID")
	requestFile := flags.String("request", "", "JSON request file for create, validation, alias, bulk, or batch actions")
	imageFile := flags.String("image-file", "", "local PNG or JPEG file for upload-image")
	imageType := flags.String("image-type", "", "image/png or image/jpeg")
	apply := flags.Bool("apply", false, "explicitly enable a remote write")
	confirm := flags.String("confirm", "", "must equal the action to avoid accidental changes")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if len(flags.Args()) != 0 || !*apply || *confirm != *action {
		return errors.New("manage-menu requires --apply --confirm ACTION and no extra arguments")
	}
	if getenv == nil {
		return errors.New("missing environment reader")
	}
	var payload json.RawMessage
	if *requestFile != "" {
		data, err := os.ReadFile(filepath.Clean(*requestFile))
		if err != nil {
			return fmt.Errorf("read rich menu request: %w", err)
		}
		if !json.Valid(data) {
			return errors.New("rich menu request must be valid JSON")
		}
		payload = data
	}
	if *imageFile != "" {
		if *action != "upload-image" || *requestFile != "" || strings.TrimSpace(*menu) == "" {
			return errors.New("image-file is only valid for upload-image with --menu-id")
		}
		if *imageType != "image/png" && *imageType != "image/jpeg" {
			return errors.New("image-type must be image/png or image/jpeg")
		}
		_, err := os.Stat(filepath.Clean(*imageFile))
		if err != nil {
			return errors.New("image upload file is not accessible")
		}
	}
	cmd := engagement.MenuCommand{Action: *action, RichMenuID: *menu, UserID: *user, AliasID: *alias, Payload: payload}
	if *action == "upload-image" && *imageFile != "" {
		data, err := readPrivateBounded(*imageFile, 20<<20)
		if err != nil {
			return err
		}
		cmd.Image = data
		cmd.ContentType = *imageType
	}
	if err := engagement.ValidateMenuCommand(cmd); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	if *action == "upload-image" {
		if menuImageUploadRuntimeFactory == nil {
			return errors.New("menu image upload runtime is not configured")
		}
		runtime, err := menuImageUploadRuntimeFactory(getenv(channelTokenEnv))
		if err != nil {
			return err
		}
		defer runtime.Close()
		return runtime.Upload(ctx, cmd.RichMenuID, cmd.ContentType, cmd.Image)
	}
	if menuManagementRuntimeFactory == nil {
		return errors.New("menu management runtime is not configured")
	}
	runtime, err := menuManagementRuntimeFactory(getenv(channelTokenEnv))
	if err != nil {
		return err
	}
	defer runtime.Close()
	return runtime.Execute(ctx, cmd)
}
