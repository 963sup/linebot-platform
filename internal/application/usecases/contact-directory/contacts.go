package contactdirectory

import (
	"context"
	"errors"
	"strings"
	"time"
)

type RecipientKind string

const (
	RecipientUser  RecipientKind = "user"
	RecipientGroup RecipientKind = "group"
	RecipientRoom  RecipientKind = "room"
)

type RecipientReference struct {
	ID   string
	Kind RecipientKind
}

type ResolveRecipientQuery struct {
	ID   string
	Kind RecipientKind
}

type ProfileQuery struct {
	Recipient RecipientReference
}

type ProfileSnapshot struct {
	Recipient     RecipientReference
	DisplayName   string
	PictureURL    string
	StatusMessage string
	ObservedAt    time.Time
}

// Reader is a consumer-owned query port. Provider SDK types must be mapped by
// the implementing adapter before crossing this boundary.
type Reader interface {
	ResolveRecipient(context.Context, ResolveRecipientQuery) (RecipientReference, error)
	GetProfile(context.Context, ProfileQuery) (ProfileSnapshot, error)
}

type Queries struct {
	reader Reader
}

func NewQueries(reader Reader) (*Queries, error) {
	if reader == nil {
		return nil, errors.New("contact reader is required")
	}
	return &Queries{reader: reader}, nil
}

func (q *Queries) ResolveRecipient(ctx context.Context, query ResolveRecipientQuery) (RecipientReference, error) {
	if err := validateContext(ctx); err != nil {
		return RecipientReference{}, err
	}
	if err := validateRecipient(RecipientReference{ID: query.ID, Kind: query.Kind}); err != nil {
		return RecipientReference{}, err
	}
	return q.reader.ResolveRecipient(ctx, query)
}

func (q *Queries) GetProfile(ctx context.Context, query ProfileQuery) (ProfileSnapshot, error) {
	if err := validateContext(ctx); err != nil {
		return ProfileSnapshot{}, err
	}
	if err := validateRecipient(query.Recipient); err != nil {
		return ProfileSnapshot{}, err
	}
	return q.reader.GetProfile(ctx, query)
}

func validateContext(ctx context.Context) error {
	if ctx == nil {
		return errors.New("context is required")
	}
	return ctx.Err()
}

func validateRecipient(recipient RecipientReference) error {
	if recipient.ID == "" || strings.TrimSpace(recipient.ID) != recipient.ID {
		return errors.New("recipient ID is required")
	}
	switch recipient.Kind {
	case RecipientUser, RecipientGroup, RecipientRoom:
		return nil
	default:
		return errors.New("unsupported recipient kind")
	}
}
