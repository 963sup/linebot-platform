package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"strings"
	"time"

	contactdirectory "github.com/963sup/linebot-platform/internal/application/usecases/contact-directory"
)

// runProfile is intentionally CLI-only: no public HTTP route or automatic
// collection of profile data. It makes an external call only with --lookup.
func runProfile(parent context.Context, args []string, getenv func(string) string, output io.Writer, dependencies ...Dependencies) (err error) {
	deps := dependency(dependencies)
	flags := flag.NewFlagSet("profile", flag.ContinueOnError)
	userID := flags.String("user", "", "LINE user ID")
	lookup := flags.Bool("lookup", false, "explicitly fetch the profile from LINE")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if len(flags.Args()) != 0 {
		return errors.New("unexpected profile arguments")
	}
	if !*lookup {
		return errors.New("profile lookup requires --lookup")
	}
	if *userID == "" || *userID != strings.TrimSpace(*userID) {
		return errors.New("--user must be a nonempty LINE user ID without surrounding whitespace")
	}
	if getenv == nil || output == nil {
		return errors.New("profile runtime is not configured")
	}
	if getenv(channelTokenEnv) == "" {
		return errors.New("access token is required")
	}
	if deps.ProfileRuntime == nil {
		return errors.New("profile runtime is not configured")
	}
	runtime, err := deps.ProfileRuntime(getenv(channelTokenEnv))
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, runtime.Close()) }()
	ctx, cancel := context.WithTimeout(parent, 12*time.Second)
	defer cancel()
	profile, err := runtime.GetProfile(ctx, contactdirectory.ProfileQuery{Recipient: contactdirectory.RecipientReference{
		ID: *userID, Kind: contactdirectory.RecipientUser,
	}})
	if err != nil {
		return err
	}
	return json.NewEncoder(output).Encode(struct {
		UserID      string `json:"user_id"`
		DisplayName string `json:"display_name"`
		PictureURL  string `json:"picture_url,omitempty"`
	}{UserID: profile.Recipient.ID, DisplayName: profile.DisplayName, PictureURL: profile.PictureURL})
}
