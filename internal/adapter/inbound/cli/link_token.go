package cli

import (
	"context"
	"errors"
	"flag"
	"time"
)

// Link tokens are credentials. Never print them to terminal output.
func runLinkToken(args []string, getenv func(string) string) error {
	flags := flag.NewFlagSet("link-token", flag.ContinueOnError)
	user := flags.String("user", "", "LINE user ID")
	out := flags.String("out", "", "new file for sensitive link token (0600)")
	apply := flags.Bool("apply", false, "authorize token issuance")
	confirm := flags.String("confirm", "", "must equal issue-link-token")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if len(flags.Args()) != 0 || !*apply || *confirm != "issue-link-token" || *user == "" || *out == "" || getenv == nil {
		return errors.New("link-token requires --user --out --apply --confirm issue-link-token")
	}
	return writePrivateOperation(*out, func() ([]byte, error) {
		if linkTokenRuntimeFactory == nil {
			return nil, errors.New("link-token runtime is not configured")
		}
		r, err := linkTokenRuntimeFactory(getenv(channelTokenEnv))
		if err != nil {
			return nil, err
		}
		defer r.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		defer cancel()
		return r.IssueLinkToken(ctx, *user)
	})
}
