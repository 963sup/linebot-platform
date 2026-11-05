package membership

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

type MembershipID string
type RecipientID string

type PageRequest struct {
	Token string
	Limit int
}

type MembershipView struct {
	ID         MembershipID
	Name       string
	Status     string
	ObservedAt time.Time
}

type MembershipPage struct {
	Memberships []MembershipView
	NextToken   string
}

type SubscriptionView struct {
	RecipientID RecipientID
	Membership  MembershipView
	Status      string
	ValidUntil  time.Time
	ObservedAt  time.Time
}

type MemberView struct {
	RecipientID RecipientID
	Status      string
	ObservedAt  time.Time
}

type MemberPage struct {
	Members   []MemberView
	NextToken string
}

// Reader is owned by this use case and implemented by an outbound adapter.
type Reader interface {
	ListMemberships(context.Context, PageRequest) (MembershipPage, error)
	GetSubscription(context.Context, RecipientID) (SubscriptionView, error)
	ListMembershipMembers(context.Context, MembershipID, PageRequest) (MemberPage, error)
}

type Queries struct {
	reader Reader
}

func NewQueries(reader Reader) (*Queries, error) {
	if reader == nil {
		return nil, errors.New("membership reader is required")
	}
	return &Queries{reader: reader}, nil
}

func (q *Queries) ListMemberships(ctx context.Context, page PageRequest) (MembershipPage, error) {
	if err := validateContext(ctx); err != nil {
		return MembershipPage{}, err
	}
	if err := validatePage(page); err != nil {
		return MembershipPage{}, err
	}
	page = normalizePage(page)
	return q.reader.ListMemberships(ctx, page)
}

func (q *Queries) GetSubscription(ctx context.Context, recipientID RecipientID) (SubscriptionView, error) {
	if err := validateContext(ctx); err != nil {
		return SubscriptionView{}, err
	}
	if !validIdentifier(string(recipientID)) {
		return SubscriptionView{}, errors.New("recipient ID is required")
	}
	return q.reader.GetSubscription(ctx, recipientID)
}

func (q *Queries) ListMembers(ctx context.Context, membershipID MembershipID, page PageRequest) (MemberPage, error) {
	if err := validateContext(ctx); err != nil {
		return MemberPage{}, err
	}
	if !validIdentifier(string(membershipID)) {
		return MemberPage{}, errors.New("membership ID is required")
	}
	if err := validatePage(page); err != nil {
		return MemberPage{}, err
	}
	page = normalizePage(page)
	return q.reader.ListMembershipMembers(ctx, membershipID, page)
}

func validateContext(ctx context.Context) error {
	if ctx == nil {
		return errors.New("context is required")
	}
	return ctx.Err()
}

func validatePage(page PageRequest) error {
	if page.Limit < 0 || page.Limit > MaxPageSize || strings.TrimSpace(page.Token) != page.Token {
		return errors.New("invalid membership pagination")
	}
	return nil
}

func normalizePage(page PageRequest) PageRequest {
	if page.Limit == 0 {
		page.Limit = DefaultPageSize
	}
	return page
}

func validIdentifier(value string) bool {
	return value != "" && strings.TrimSpace(value) == value
}
