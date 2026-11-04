package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/963sup/linebot-platform/internal/application/usecases/contacts"
)

func runContacts(args []string, getenv func(string) string, output io.Writer) error {
	flags := flag.NewFlagSet("contacts", flag.ContinueOnError)
	kind := flags.String("kind", "", "read-only LINE contacts/group query")
	conversationID := flags.String("conversation-id", "", "LINE group or room ID")
	userID := flags.String("user", "", "LINE user ID for group/room member profile")
	start := flags.String("start", "", "continuation token")
	limit := flags.Int("limit", 0, "follower page size 1-1000")
	read := flags.Bool("read", false, "authorize personal-data retrieval")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if !*read || len(flags.Args()) != 0 {
		return errors.New("contacts requires --read and no extra arguments")
	}
	if getenv == nil || output == nil {
		return errors.New("missing CLI configuration")
	}
	q := contacts.Inspection{Kind: *kind, ConversationID: *conversationID, UserID: *userID, Start: *start, Limit: *limit}
	if err := contacts.ValidateInspection(q); err != nil {
		return err
	}
	if contactsRuntimeFactory == nil {
		return errors.New("contacts runtime is not configured")
	}
	rt, err := contactsRuntimeFactory(getenv(channelTokenEnv))
	if err != nil {
		return err
	}
	defer rt.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	result, err := rt.Inspect(ctx, q)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(result.Value)
	if err != nil {
		return fmt.Errorf("encode contacts result: %w", err)
	}
	_, err = fmt.Fprintln(output, string(payload))
	return err
}
