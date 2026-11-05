package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	contactdirectory "github.com/963sup/linebot-platform/internal/application/usecases/contact-directory"
)

type contactDirectoryRuntime interface {
	ListFollowers(context.Context, contactdirectory.PageRequest) (contactdirectory.RecipientPage, error)
	GetSummary(context.Context, contactdirectory.RecipientReference) (contactdirectory.ConversationSummary, error)
	ListMembers(context.Context, contactdirectory.RecipientReference, contactdirectory.PageRequest) (contactdirectory.RecipientPage, error)
	GetMemberProfile(context.Context, contactdirectory.RecipientReference, contactdirectory.RecipientID) (contactdirectory.ProfileSnapshot, error)
}

func runContacts(parent context.Context, args []string, getenv func(string) string, output io.Writer, dependencies ...Dependencies) error {
	deps := dependency(dependencies)
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
	if deps.ContactsRuntime == nil {
		return errors.New("contacts runtime is not configured")
	}
	rt, err := deps.ContactsRuntime(getenv(channelTokenEnv))
	if err != nil {
		return err
	}
	defer rt.Close()
	reader, ok := any(rt).(contactDirectoryRuntime)
	if !ok {
		return errors.New("contacts runtime does not implement contact-directory")
	}
	ctx, cancel := context.WithTimeout(parent, 12*time.Second)
	defer cancel()
	page := contactdirectory.PageRequest{Cursor: *start, Limit: *limit}
	conversation := contactdirectory.RecipientReference{ID: *conversationID}
	var result any
	switch *kind {
	case "followers":
		result, err = reader.ListFollowers(ctx, page)
	case "group-summary", "group-member-count":
		conversation.Kind = contactdirectory.RecipientGroup
		result, err = reader.GetSummary(ctx, conversation)
	case "room-member-count":
		conversation.Kind = contactdirectory.RecipientRoom
		result, err = reader.GetSummary(ctx, conversation)
	case "group-members":
		conversation.Kind = contactdirectory.RecipientGroup
		result, err = reader.ListMembers(ctx, conversation, page)
	case "room-members":
		conversation.Kind = contactdirectory.RecipientRoom
		result, err = reader.ListMembers(ctx, conversation, page)
	case "group-member-profile":
		conversation.Kind = contactdirectory.RecipientGroup
		result, err = reader.GetMemberProfile(ctx, conversation, contactdirectory.RecipientID(*userID))
	case "room-member-profile":
		conversation.Kind = contactdirectory.RecipientRoom
		result, err = reader.GetMemberProfile(ctx, conversation, contactdirectory.RecipientID(*userID))
	default:
		return errors.New("unsupported contacts inspection kind")
	}
	if err != nil {
		return err
	}
	payload, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("encode contacts result: %w", err)
	}
	_, err = fmt.Fprintln(output, string(payload))
	return err
}
