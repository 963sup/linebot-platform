package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"strings"
	"time"

	"github.com/963sup/linebot-platform/internal/bootstrap"
)

// runProfile is intentionally CLI-only: no public HTTP route or automatic
// collection of profile data. It makes an external call only with --lookup.
func runProfile(args []string, getenv func(string) string, output io.Writer) error {
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
	runtime, err := bootstrap.NewLINEOpsRuntime(getenv(bootstrap.KeyChannelToken))
	if err != nil {
		return err
	}
	defer runtime.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	profile, err := runtime.Profiles.Get(ctx, *userID)
	if err != nil {
		return err
	}
	return json.NewEncoder(output).Encode(struct {
		UserID      string `json:"user_id"`
		DisplayName string `json:"display_name"`
		PictureURL  string `json:"picture_url,omitempty"`
	}{UserID: profile.ID, DisplayName: profile.DisplayName, PictureURL: profile.PictureURL})
}
