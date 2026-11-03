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
	dispatchCooldown = 2 * time.Minute
	inboxRetention   = 7 * 24 * time.Hour
	outboxRetention  = time.Hour
)

type storedJob struct {
	ReplyToken     string           `firestore:"reply_token"`
	Message        outbound.Message `firestore:"message"`
	ReplyUntil     time.Time        `firestore:"reply_until"`
	NextDispatchAt time.Time        `firestore:"next_dispatch_at"`
	OccurredAt     time.Time        `firestore:"occurred_at"`
	CreatedAt      time.Time        `firestore:"created_at"`
	UpdatedAt      time.Time        `firestore:"updated_at"`
	ExpiresAt      time.Time        `firestore:"expires_at"`
	State          string           `firestore:"state"`
	LeaseID        string           `firestore:"lease_id"`
	LeaseUntil     time.Time        `firestore:"lease_until"`
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
			// Only never-sent pending or expired work can be refreshed.
			// Once a sending claim exists, no duplicate LINE call is allowed.
			doc, err := tx.Get(outbox)
			if status.Code(err) == codes.NotFound {
				return nil // outbox already removed by TTL
			}
			if err != nil {
				return err
			}
			var saved storedJob
			if err := doc.DataTo(&saved); err != nil {
				return err
			}
			if (saved.State == "pending" || saved.State == "expired") &&
				job.ReplyUntil.After(saved.ReplyUntil) {
				return tx.Update(outbox, []sdk.Update{
					{Path: "state", Value: "pending"},
					{Path: "reply_until", Value: job.ReplyUntil},
					{Path: "next_dispatch_at", Value: job.CreatedAt},
					{Path: "reply_token", Value: job.ReplyToken},
					{Path: "updated_at", Value: job.CreatedAt},
				})
			}
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
			NextDispatchAt: job.CreatedAt,
			OccurredAt: job.OccurredAt,
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

// Recover makes bounded progress across *due* records, rather than repeatedly
// selecting the same 100 pending documents. A durable next_dispatch_at is
// reserved before enqueueing. If the process crashes, a later scan retries
// after dispatchCooldown; Claim remains the one-use LINE send gate.
//
// Required composite indexes: state+reply_until, state+lease_until,
// state+next_dispatch_at (all ascending). See deploy/firestore/indexes.json.
func (s *Store) Recover(ctx context.Context, now time.Time, limit int) ([]string, error) {
	if limit < 1 || limit > 100 {
		return nil, errors.New("recovery batch size must be 1–100")
	}
	coll := s.client.Collection(outboxCollection)

	// Expire only actually overdue replies; a large set of future replies
	// must not block expiry processing. Conditional updates guard races.
	expired := coll.Where("state", "==", "pending").
		Where("reply_until", "<=", now).OrderBy("reply_until", sdk.Asc).Limit(limit).Documents(ctx)
	defer expired.Stop()
	for {
		doc, err := expired.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		if err := s.expirePending(ctx, doc.Ref, now); err != nil {
			return nil, err
		}
	}

	// A timed-out lease means an earlier worker might have called LINE.
	// Never reclaim it for an automatic second send.
	abandoned := coll.Where("state", "==", "sending").
		Where("lease_until", "<=", now).OrderBy("lease_until", sdk.Asc).Limit(limit).Documents(ctx)
	defer abandoned.Stop()
	for {
		doc, err := abandoned.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		if err := s.unknownExpired(ctx, doc.Ref, now); err != nil {
			return nil, err
		}
	}

	// Process the earliest eligible next_dispatch_at, moving each selected
	// item forward so repeated schedule runs do not starve later pending jobs.
	due := coll.Where("state", "==", "pending").
		Where("next_dispatch_at", "<=", now).OrderBy("next_dispatch_at", sdk.Asc).Limit(limit).Documents(ctx)
	defer due.Stop()
	keys := make([]string, 0, limit)
	for {
		doc, err := due.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		if !validKey(doc.Ref.ID) {
			continue
		}
		reserved, err := s.reserveDispatch(ctx, doc.Ref, now)
		if err != nil {
			return nil, err
		}
		if reserved {
			keys = append(keys, doc.Ref.ID)
		}
	}
	return keys, nil
}

// reserveDispatch conditionally advances the due time in a Firestore
// transaction. Lost task publication can be retried after the cooldown.
func (s *Store) reserveDispatch(ctx context.Context, ref *sdk.DocumentRef, now time.Time) (bool, error) {
	var reserved bool
	err := s.client.RunTransaction(ctx, func(ctx context.Context, tx *sdk.Transaction) error {
		reserved = false
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
		if saved.State != "pending" || now.Before(saved.NextDispatchAt) {
			return nil
		}
		if !now.Before(saved.ReplyUntil) {
			return tx.Update(ref, []sdk.Update{
				{Path: "state", Value: "expired"},
				{Path: "reply_token", Value: ""},
				{Path: "updated_at", Value: now},
			})
		}
		if err := tx.Update(ref, []sdk.Update{
			{Path: "next_dispatch_at", Value: now.Add(dispatchCooldown)},
			{Path: "updated_at", Value: now},
		}); err != nil {
			return err
		}
		reserved = true
		return nil
	})
	return reserved, err
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
