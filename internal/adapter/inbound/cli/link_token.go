package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"time"

	accountlinktoken "github.com/963sup/linebot-platform/internal/application/usecases/account-link-token"
)

// Link tokens are credentials. Never print them to terminal output.
func runLinkToken(args []string, getenv func(string) string, dependencies ...Dependencies) error {
	deps := dependency(dependencies)
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
		if deps.LinkTokenRuntime == nil {
			return nil, errors.New("link-token runtime is not configured")
		}
		r, err := deps.LinkTokenRuntime(getenv(channelTokenEnv))
		if err != nil {
			return nil, err
		}
		defer r.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		defer cancel()
		result, err := r.Issue(ctx, accountlinktoken.Subject{ID: *user})
		if err != nil {
			return nil, err
		}
		return json.Marshal(struct {
			LinkToken string `json:"linkToken"`
		}{LinkToken: result.Value})
	})
}
