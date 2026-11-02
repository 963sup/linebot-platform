// Package firestore implements transactionally stored LINE inbox/outbox events.
package firestore

import (
	"context"
	"errors"
	"time"

	sdk "cloud.google.com/go/firestore"
	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	inboxCollection  = "line_inbox_v1"
	outboxCollection = "line_outbox_v1"
	leaseDuration    = 90 * time.Second
	inboxRetention   = 7 * 24 * time.Hour
	outboxRetention  = time.Hour
)

type storedJob struct {
	ReplyToken string           `firestore:"reply_token"`
	Message    outbound.Message `firestore:"message"`
	ReplyUntil time.Time        `firestore:"reply_until"`
	CreatedAt  time.Time        `firestore:"created_at"`
	UpdatedAt  time.Time        `firestore:"updated_at"`
	ExpiresAt  time.Time        `firestore:"expires_at"`
	State      string           `firestore:"state"`
	LeaseID    string           `firestore:"lease_id"`
	LeaseUntil time.Time        `firestore:"lease_until"`
}

type Store struct{ client *sdk.Client }

func NewStore(client *sdk.Client) (*Store, error) {
	if client == nil {
		return nil, errors.New("Firestore client is required")
	}
	return &Store{client: client}, nil
}

func (s *Store) outbox(key string) *sdk.DocumentRef {
	return s.client.Collection(outboxCollection).Doc(key)
}

// Record creates both Firestore documents atomically; only the first receipt
// gets to set a reply token. Never store the original user message or user ID.
func (s *Store) Record(ctx context.Context, job outbound.EventJob) (bool, error) {
	if !validKey(job.Key) || job.ReplyToken == "" || job.ReplyUntil.IsZero() {
		return false, errors.New("invalid event job")
	}
	inbox := s.client.Collection(inboxCollection).Doc(job.Key)
	outbox := s.outbox(job.Key)
	var created bool
	err := s.client.RunTransaction(ctx, func(ctx context.Context, tx *sdk.Transaction) error {
		created = false // reset on Firestore's automatic transaction retries
		_, err := tx.Get(inbox)
		if err == nil {
			return nil
		}
		if status.Code(err) != codes.NotFound {
			return err
		}
		if err := tx.Create(inbox, map[string]interface{}{
			"created_at": job.CreatedAt,
			"expires_at": job.CreatedAt.Add(inboxRetention),
		}); err != nil {
			return err
		}
		if err := tx.Create(outbox, storedJob{
			ReplyToken: job.ReplyToken,
			Message:    job.Message,
			ReplyUntil: job.ReplyUntil,
			CreatedAt:  job.CreatedAt,
			UpdatedAt:  job.CreatedAt,
			ExpiresAt:  job.CreatedAt.Add(outboxRetention),
			State:      "pending",
		}); err != nil {
			return err
		}
		created = true
		return nil
	})
	return created, err
}

// Claim commits "sending" before any network send. The lease is for
// observability/recovery, not permission to send a second time.
func (s *Store) Claim(ctx context.Context, key, leaseID string, now time.Time) (outbound.EventJob, bool, error) {
	if !validKey(key) || leaseID == "" {
		return outbound.EventJob{}, false, errors.New("invalid claim")
	}
	var job outbound.EventJob
	var claimed bool
	err := s.client.RunTransaction(ctx, func(ctx context.Context, tx *sdk.Transaction) error {
		claimed = false
		job = outbound.EventJob{}
		ref := s.outbox(key)
		snapshot, err := tx.Get(ref)
		if status.Code(err) == codes.NotFound {
			return nil // inbox might outlive an outbox TTL document
		}
		if err != nil {
			return err
		}
		var saved storedJob
		if err := snapshot.DataTo(&saved); err != nil {
			return err
		}
		switch saved.State {
		case "pending":
			if !now.Before(saved.ReplyUntil) {
				return tx.Update(ref, []sdk.Update{
					{Path: "state", Value: "expired"},
					{Path: "reply_token", Value: ""},
					{Path: "updated_at", Value: now},
				})
			}
			if err := tx.Update(ref, []sdk.Update{
				{Path: "state", Value: "sending"},
				{Path: "lease_id", Value: leaseID},
				{Path: "lease_until", Value: now.Add(leaseDuration)},
				{Path: "updated_at", Value: now},
			}); err != nil {
				return err
			}
			job = outbound.EventJob{
				Key: key, ReplyToken: saved.ReplyToken, Message: saved.Message,
				ReplyUntil: saved.ReplyUntil, CreatedAt: saved.CreatedAt,
			}
			claimed = true
		case "sending":
			if !now.Before(saved.LeaseUntil) {
				return tx.Update(ref, []sdk.Update{
					{Path: "state", Value: "unknown"},
					{Path: "reply_token", Value: ""},
					{Path: "updated_at", Value: now},
				})
			}
		}
		return nil
	})
	return job, claimed, err
}

func (s *Store) Complete(ctx context.Context, key, leaseID string, outcome outbound.DeliveryState) error {
	if !validKey(key) || leaseID == "" {
		return errors.New("invalid completion")
	}
	if outcome != outbound.DeliverySent && outcome != outbound.DeliveryExpired && outcome != outbound.DeliveryUnknown {
		return errors.New("invalid completion state")
	}
	return s.client.RunTransaction(ctx, func(ctx context.Context, tx *sdk.Transaction) error {
		ref := s.outbox(key)
		snapshot, err := tx.Get(ref)
		if err != nil {
			return err
		}
		var saved storedJob
		if err := snapshot.DataTo(&saved); err != nil {
			return err
		}
		if saved.State != "sending" || saved.LeaseID != leaseID {
			return errors.New("event lease is no longer active")
		}
		return tx.Update(ref, []sdk.Update{
			{Path: "state", Value: string(outcome)},
			{Path: "reply_token", Value: ""},
			{Path: "lease_id", Value: ""},
			{Path: "updated_at", Value: time.Now().UTC()},
		})
	})
}

// Recover queries pending events for re-enqueue and marks abandoned sending
// leases unknown. It intentionally never turns an old sending state to pending.
func (s *Store) Recover(ctx context.Context, now time.Time, limit int) ([]string, error) {
	if limit < 1 || limit > 100 {
		return nil, errors.New("recovery batch size must be 1–100")
	}
	// Query each state separately. These bounded index queries require only
	// the default single-field state index.
	query := s.client.Collection(outboxCollection).Where("state", "==", "pending").Limit(limit)
	iter := query.Documents(ctx)
	defer iter.Stop()
	var pending []string
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var saved storedJob
		if err := doc.DataTo(&saved); err != nil {
			return nil, err
		}
		if !now.Before(saved.ReplyUntil) {
			if err := s.expirePending(ctx, doc.Ref, now); err != nil {
				return nil, err
			}
			continue
		}
		if validKey(doc.Ref.ID) {
			pending = append(pending, doc.Ref.ID)
		}
	}
	// An expired lease means the prior worker might have called LINE. Mark
	// unknown; no second sending attempt is automatically permitted.
	sending := s.client.Collection(outboxCollection).Where("state", "==", "sending").Limit(limit)
	inProgress := sending.Documents(ctx)
	defer inProgress.Stop()
	for {
		doc, err := inProgress.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var saved storedJob
		if err := doc.DataTo(&saved); err != nil {
			return nil, err
		}
		if !now.Before(saved.LeaseUntil) {
			if err := s.unknownExpired(ctx, doc.Ref, now); err != nil {
				return nil, err
			}
		}
	}
	return pending, nil
}

func (s *Store) expirePending(ctx context.Context, ref *sdk.DocumentRef, now time.Time) error {
	return s.transition(ctx, ref, "pending", now, func(s storedJob) bool {
		return !now.Before(s.ReplyUntil)
	}, "expired")
}

func (s *Store) unknownExpired(ctx context.Context, ref *sdk.DocumentRef, now time.Time) error {
	return s.transition(ctx, ref, "sending", now, func(s storedJob) bool {
		return !now.Before(s.LeaseUntil)
	}, "unknown")
}

func (s *Store) transition(ctx context.Context, ref *sdk.DocumentRef, state string, now time.Time, due func(storedJob) bool, outcome string) error {
	return s.client.RunTransaction(ctx, func(ctx context.Context, tx *sdk.Transaction) error {
		doc, err := tx.Get(ref)
		if status.Code(err) == codes.NotFound {
			return nil
		}
		if err != nil {
			return err
		}
		var saved storedJob
		if err := doc.DataTo(&saved); err != nil {
			return err
		}
		if saved.State != state || !due(saved) {
			return nil
		}
		return tx.Update(ref, []sdk.Update{
			{Path: "state", Value: outcome},
			{Path: "reply_token", Value: ""},
			{Path: "lease_id", Value: ""},
			{Path: "updated_at", Value: now},
		})
	})
}

func validKey(key string) bool {
	if len(key) != 64 {
		return false
	}
	for _, r := range key {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}
