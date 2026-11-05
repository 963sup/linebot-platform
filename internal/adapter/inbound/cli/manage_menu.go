package cli

import (
	"context"
	"errors"
	"flag"
	"strings"
	"time"

	experiencepresentation "github.com/963sup/linebot-platform/internal/application/usecases/experience-presentation"
	mediatransfer "github.com/963sup/linebot-platform/internal/application/usecases/media-transfer"
)

func runManageMenu(parent context.Context, args []string, getenv func(string) string, dependencies ...Dependencies) error {
	deps := dependency(dependencies)
	flags := flag.NewFlagSet("manage-menu", flag.ContinueOnError)
	action := flags.String("action", "", "rich menu action")
	menu := flags.String("menu-id", "", "rich menu ID")
	user := flags.String("user", "", "LINE user ID")
	alias := flags.String("alias-id", "", "rich menu alias ID")
	requestFile := flags.String("request", "", "JSON request file")
	imageFile := flags.String("image-file", "", "local PNG or JPEG file")
	imageType := flags.String("image-type", "", "image/png or image/jpeg")
	apply := flags.Bool("apply", false, "enable remote write")
	confirm := flags.String("confirm", "", "must equal action")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if len(flags.Args()) != 0 || !*apply || *confirm != *action || getenv == nil {
		return errors.New("manage-menu requires --apply --confirm ACTION and no extra arguments")
	}
	var document experiencepresentation.ProviderDocument
	if *requestFile != "" {
		data, err := readPrivateBounded(*requestFile, 256<<10)
		if err != nil {
			return err
		}
		value, err := decodeLINEJSON(data)
		if err != nil {
			return errors.New("rich menu request must be valid JSON")
		}
		document.Value = value
	}
	ctx, cancel := context.WithTimeout(parent, 12*time.Second)
	defer cancel()
	if *action == "upload-image" {
		if *imageFile == "" || document.Value != nil || strings.TrimSpace(*menu) == "" || (*imageType != "image/png" && *imageType != "image/jpeg") {
			return errors.New("upload-image requires menu ID and PNG or JPEG image")
		}
		data, err := readPrivateBounded(*imageFile, 20<<20)
		if err != nil {
			return err
		}
		if deps.MenuImageUploadRuntime == nil {
			return errors.New("menu image upload runtime is not configured")
		}
		runtime, err := deps.MenuImageUploadRuntime(getenv(channelTokenEnv))
		if err != nil {
			return err
		}
		defer runtime.Close()
		return runtime.StorePresentationImage(ctx, *menu, mediatransfer.BinaryContent{Data: data, ContentType: *imageType})
	}
	if err := validateManageMenuInput(*action, *menu, *user, *alias, document.Value != nil); err != nil {
		return err
	}
	if deps.MenuManagementRuntime == nil {
		return errors.New("menu management runtime is not configured")
	}
	runtime, err := deps.MenuManagementRuntime(getenv(channelTokenEnv))
	if err != nil {
		return err
	}
	defer runtime.Close()
	switch *action {
	case "create":
		return runtime.CreateRichMenu(ctx, document)
	case "validate":
		return runtime.ValidateRichMenu(ctx, document)
	case "create-alias":
		return runtime.CreateRichMenuAlias(ctx, document)
	case "update-alias":
		return runtime.UpdateRichMenuAlias(ctx, *alias, document)
	case "bulk-link":
		return runtime.BulkLinkRichMenu(ctx, document)
	case "bulk-unlink":
		return runtime.BulkUnlinkRichMenu(ctx, document)
	case "batch":
		return runtime.RichMenuBatch(ctx, document)
	case "validate-batch":
		return runtime.ValidateRichMenuBatch(ctx, document)
	case "set-default":
		return runtime.SetDefaultRichMenu(ctx, *menu)
	case "cancel-default":
		return runtime.CancelDefaultRichMenu(ctx)
	case "link-user":
		return runtime.LinkRichMenuToUser(ctx, *user, *menu)
	case "unlink-user":
		return runtime.UnlinkRichMenuFromUser(ctx, *user)
	case "delete-menu":
		return runtime.DeleteRichMenu(ctx, *menu)
	case "delete-alias":
		return runtime.DeleteRichMenuAlias(ctx, *alias)
	default:
		return errors.New("unsupported rich menu action")
	}
}

func validateManageMenuInput(action, menuID, userID, aliasID string, hasDocument bool) error {
	validID := func(value string) bool { return value != "" && value == strings.TrimSpace(value) }
	noIDs := menuID == "" && userID == "" && aliasID == ""
	switch action {
	case "create", "validate", "create-alias", "bulk-link", "bulk-unlink", "batch", "validate-batch":
		if !hasDocument || !noIDs {
			return errors.New("rich menu action requires a JSON document and no IDs")
		}
	case "update-alias":
		if !validID(aliasID) || !hasDocument || menuID != "" || userID != "" {
			return errors.New("update-alias requires alias ID and a JSON document")
		}
	case "set-default", "delete-menu":
		if !validID(menuID) || userID != "" || aliasID != "" || hasDocument {
			return errors.New("rich menu action requires only a menu ID")
		}
	case "cancel-default":
		if !noIDs || hasDocument {
			return errors.New("cancel-default accepts no IDs or document")
		}
	case "link-user":
		if !validID(menuID) || !validID(userID) || aliasID != "" || hasDocument {
			return errors.New("link-user requires only menu and user IDs")
		}
	case "unlink-user":
		if !validID(userID) || menuID != "" || aliasID != "" || hasDocument {
			return errors.New("unlink-user requires only a user ID")
		}
	case "delete-alias":
		if !validID(aliasID) || menuID != "" || userID != "" || hasDocument {
			return errors.New("delete-alias requires only an alias ID")
		}
	default:
		return errors.New("unsupported rich menu action")
	}
	return nil
}
