package contactdirectory

import (
	"context"
	"errors"
	"strings"
	"time"
)

const (
	DefaultPageSize = 20
	MaxPageSize     = 1000
)

type PageRequest struct {
	Cursor string
	Limit  int
}

type RecipientPage struct {
	Recipients []RecipientReference
	NextCursor string
}

type ConversationSummary struct {
	Conversation RecipientReference
	Name         string
	PictureURL   string
	MemberCount  int64
	ObservedAt   time.Time
}

type ConversationReader interface {
	ListFollowers(context.Context, PageRequest) (RecipientPage, error)
	GetConversationSummary(context.Context, RecipientReference) (ConversationSummary, error)
	ListMembers(context.Context, RecipientReference, PageRequest) (RecipientPage, error)
	GetMemberProfile(context.Context, RecipientReference, RecipientID) (ProfileSnapshot, error)
}

type RecipientID string

type ConversationQueries struct{ reader ConversationReader }

func NewConversationQueries(reader ConversationReader) (*ConversationQueries, error) {
	if reader == nil {
		return nil, errors.New("conversation reader is required")
	}
	return &ConversationQueries{reader: reader}, nil
}

func (q *ConversationQueries) ListFollowers(ctx context.Context, page PageRequest) (RecipientPage, error) {
	if err := validateContext(ctx); err != nil {
		return RecipientPage{}, err
	}
	page, err := normalizePage(page)
	if err != nil {
		return RecipientPage{}, err
	}
	return q.reader.ListFollowers(ctx, page)
}

func (q *ConversationQueries) GetSummary(ctx context.Context, conversation RecipientReference) (ConversationSummary, error) {
	if err := validateContext(ctx); err != nil {
		return ConversationSummary{}, err
	}
	if err := validateConversation(conversation); err != nil {
		return ConversationSummary{}, err
	}
	return q.reader.GetConversationSummary(ctx, conversation)
}

func (q *ConversationQueries) ListMembers(ctx context.Context, conversation RecipientReference, page PageRequest) (RecipientPage, error) {
	if err := validateContext(ctx); err != nil {
		return RecipientPage{}, err
	}
	if err := validateConversation(conversation); err != nil {
		return RecipientPage{}, err
	}
	page, err := normalizePage(page)
	if err != nil {
		return RecipientPage{}, err
	}
	return q.reader.ListMembers(ctx, conversation, page)
}

func (q *ConversationQueries) GetMemberProfile(ctx context.Context, conversation RecipientReference, member RecipientID) (ProfileSnapshot, error) {
	if err := validateContext(ctx); err != nil {
		return ProfileSnapshot{}, err
	}
	if err := validateConversation(conversation); err != nil {
		return ProfileSnapshot{}, err
	}
	if !validContactID(string(member)) {
		return ProfileSnapshot{}, errors.New("member recipient ID is required")
	}
	return q.reader.GetMemberProfile(ctx, conversation, member)
}

func validateConversation(value RecipientReference) error {
	if err := validateRecipient(value); err != nil {
		return err
	}
	if value.Kind != RecipientGroup {
		return errors.New("conversation must be a group")
	}
	return nil
}

func normalizePage(page PageRequest) (PageRequest, error) {
	if page.Cursor != strings.TrimSpace(page.Cursor) || page.Limit < 0 || page.Limit > MaxPageSize {
		return PageRequest{}, errors.New("invalid contact pagination")
	}
	if page.Limit == 0 {
		page.Limit = DefaultPageSize
	}
	return page, nil
}

func validContactID(value string) bool { return value != "" && value == strings.TrimSpace(value) }
