package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/963sup/linebot-platform/internal/application/usecases/engagement"
	"github.com/963sup/linebot-platform/internal/bootstrap"
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
	cmd := engagement.MenuCommand{Action: *action, RichMenuID: *menu, UserID: *user, AliasID: *alias, Payload: payload}
	if err := engagement.ValidateMenuCommand(cmd); err != nil {
		return err
	}
	runtime, err := bootstrap.NewLINEOpsRuntime(getenv(bootstrap.KeyChannelToken))
	if err != nil {
		return err
	}
	defer runtime.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	return runtime.MenuManager.Execute(ctx, cmd)
}
