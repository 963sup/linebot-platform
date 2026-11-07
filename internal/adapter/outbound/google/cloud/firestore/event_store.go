// Package firestore implements transactionally stored LINE inbox/outbox events.
package firestore

import (
	"context"
	"errors"
	"math"
	"time"

	sdk "cloud.google.com/go/firestore"
	messagingdelivery "github.com/963sup/linebot-platform/internal/application/usecases/messaging-delivery"
	"github.com/963sup/linebot-platform/internal/domain/messaging-delivery/aggregates"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/messaging-delivery/value-objects"
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
	SchemaVersion  int                               `firestore:"schema_version"`
	Context        messagingdelivery.DeliveryContext `firestore:"delivery_context,omitempty"`
	ReplyToken     string                            `firestore:"reply_token"`
	Message        messagingdelivery.Message         `firestore:"message"`
	ReplyUntil     time.Time                         `firestore:"reply_until"`
	NextDispatchAt time.Time                         `firestore:"next_dispatch_at"`
	OccurredAt     time.Time                         `firestore:"occurred_at"`
	CreatedAt      time.Time                         `firestore:"created_at"`
	UpdatedAt      time.Time                         `firestore:"updated_at"`
	ExpiresAt      time.Time                         `firestore:"expires_at"`
	State          string                            `firestore:"state"`
	LeaseID        string                            `firestore:"lease_id"`
	LeaseUntil     time.Time                         `firestore:"lease_until"`
	Version        int64                             `firestore:"version"`
}

type Store struct{ client *sdk.Client }

var _ messagingdelivery.DeliveryWorkflowStore = (*Store)(nil)

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
func (s *Store) Record(ctx context.Context, job messagingdelivery.DeliveryWorkItem) (bool, error) {
	if ctx == nil {
		return false, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if !validKey(job.Key) || job.Context.Validate() != nil || job.ReplyToken == "" || job.ReplyUntil.IsZero() {
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
			if err := validateRedeliveryContext(saved, job.Context); err != nil {
				return err
			}
			delivery, restoreErr := restoreDelivery(job.Key, saved)
			if restoreErr != nil {
				return restoreErr
			}
			if job.ReplyUntil.After(saved.ReplyUntil) && delivery.RefreshReplyWindow(job.ReplyUntil) == nil {
				version, err := storedVersion(delivery.Version())
				if err != nil {
					return err
				}
				return tx.Update(outbox, []sdk.Update{
					{Path: "state", Value: string(delivery.Status())},
					{Path: "reply_until", Value: job.ReplyUntil},
					{Path: "next_dispatch_at", Value: job.CreatedAt},
					{Path: "reply_token", Value: job.ReplyToken},
					{Path: "lease_id", Value: ""},
					{Path: "lease_until", Value: time.Time{}},
					{Path: "updated_at", Value: job.CreatedAt},
					{Path: "schema_version", Value: persistedSchemaVersion(saved.SchemaVersion)},
					{Path: "version", Value: version},
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
			SchemaVersion:  3,
			Context:        job.Context,
			ReplyToken:     job.ReplyToken,
			Message:        job.Message,
			ReplyUntil:     job.ReplyUntil,
			NextDispatchAt: job.CreatedAt,
			OccurredAt:     job.OccurredAt,
			CreatedAt:      job.CreatedAt,
			UpdatedAt:      job.CreatedAt,
			ExpiresAt:      job.CreatedAt.Add(outboxRetention),
			State:          "pending",
			Version:        1,
		}); err != nil {
			return err
		}
		created = true
		return nil
	})
	return created, classifyMessagingStoreError("record", err)
}

// Claim commits "sending" before any network send. The lease is for
// observability/recovery, not permission to send a second time.
func (s *Store) Claim(ctx context.Context, key, leaseID string, now time.Time) (messagingdelivery.DeliveryWorkItem, bool, error) {
	if ctx == nil {
		return messagingdelivery.DeliveryWorkItem{}, false, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return messagingdelivery.DeliveryWorkItem{}, false, err
	}
	if !validKey(key) || leaseID == "" || now.IsZero() {
		return messagingdelivery.DeliveryWorkItem{}, false, errors.New("invalid claim")
	}
	var job messagingdelivery.DeliveryWorkItem
	var claimed bool
	err := s.client.RunTransaction(ctx, func(ctx context.Context, tx *sdk.Transaction) error {
		claimed = false
		job = messagingdelivery.DeliveryWorkItem{}
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
			delivery, restoreErr := restoreDelivery(key, saved)
			if restoreErr != nil {
				return restoreErr
			}
			if claimErr := delivery.Claim(leaseID, now, now.Add(leaseDuration)); claimErr != nil {
				if delivery.Status() == valueobjects.StatusExpired {
					version, versionErr := storedVersion(delivery.Version())
					if versionErr != nil {
						return versionErr
					}
					return tx.Update(ref, []sdk.Update{{Path: "state", Value: string(delivery.Status())}, {Path: "reply_token", Value: ""}, {Path: "updated_at", Value: now}, {Path: "schema_version", Value: persistedSchemaVersion(saved.SchemaVersion)}, {Path: "version", Value: version}})
				}
				return nil
			}
			version, err := storedVersion(delivery.Version())
			if err != nil {
				return err
			}
			if err := tx.Update(ref, []sdk.Update{
				{Path: "state", Value: string(delivery.Status())},
				{Path: "lease_id", Value: delivery.ClaimID()},
				{Path: "lease_until", Value: delivery.ClaimUntil()},
				{Path: "updated_at", Value: now},
				{Path: "schema_version", Value: persistedSchemaVersion(saved.SchemaVersion)},
				{Path: "version", Value: version},
			}); err != nil {
				return err
			}
			job = messagingdelivery.DeliveryWorkItem{
				Key: key, Context: saved.Context, ReplyToken: saved.ReplyToken, Message: saved.Message,
				ReplyUntil: saved.ReplyUntil, CreatedAt: saved.CreatedAt,
			}
			claimed = true
		case "sending":
			delivery, restoreErr := restoreDelivery(key, saved)
			if restoreErr != nil {
				return restoreErr
			}
			if delivery.Reconcile(now) {
				version, versionErr := storedVersion(delivery.Version())
				if versionErr != nil {
					return versionErr
				}
				return tx.Update(ref, []sdk.Update{{Path: "state", Value: string(delivery.Status())}, {Path: "reply_token", Value: ""}, {Path: "lease_id", Value: ""}, {Path: "lease_until", Value: time.Time{}}, {Path: "updated_at", Value: now}, {Path: "schema_version", Value: persistedSchemaVersion(saved.SchemaVersion)}, {Path: "version", Value: version}})
			}
			return messagingdelivery.ErrDeliveryInFlight
		}
		return nil
	})
	return job, claimed, classifyMessagingStoreError("claim", err)
}

func (s *Store) Complete(ctx context.Context, key, leaseID string, outcome messagingdelivery.DeliveryState, completedAt time.Time) error {
	if ctx == nil {
		return errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if !validKey(key) || leaseID == "" || completedAt.IsZero() {
		return errors.New("invalid completion")
	}
	if outcome != messagingdelivery.DeliverySent && outcome != messagingdelivery.DeliveryExpired && outcome != messagingdelivery.DeliveryUnknown {
		return errors.New("invalid completion state")
	}
	completedAt = completedAt.UTC()
	err := s.client.RunTransaction(ctx, func(ctx context.Context, tx *sdk.Transaction) error {
		ref := s.outbox(key)
		snapshot, err := tx.Get(ref)
		if err != nil {
			return err
		}
		var saved storedJob
		if err := snapshot.DataTo(&saved); err != nil {
			return err
		}
		delivery, restoreErr := restoreDelivery(key, saved)
		if restoreErr != nil {
			return restoreErr
		}
		if err := delivery.Complete(leaseID, outcome, completedAt); err != nil {
			return err
		}
		version, err := storedVersion(delivery.Version())
		if err != nil {
			return err
		}
		return tx.Update(ref, []sdk.Update{
			{Path: "state", Value: string(delivery.Status())},
			{Path: "reply_token", Value: ""},
			{Path: "lease_id", Value: ""},
			{Path: "lease_until", Value: time.Time{}},
			{Path: "updated_at", Value: completedAt},
			{Path: "schema_version", Value: persistedSchemaVersion(saved.SchemaVersion)},
			{Path: "version", Value: version},
		})
	})
	return classifyMessagingStoreError("complete", err)
}

// Recover makes bounded progress across *due* records, rather than repeatedly
// selecting the same 100 pending documents. A durable next_dispatch_at is
// reserved before enqueueing. If the process crashes, a later scan retries
// after dispatchCooldown; Claim remains the one-use LINE send gate.
//
// Required composite indexes: state+reply_until, state+lease_until,
// state+next_dispatch_at (all ascending). See deploy/firestore/indexes.json.
func (s *Store) Recover(ctx context.Context, now time.Time, limit int) ([]string, error) {
	if ctx == nil {
		return nil, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if now.IsZero() || limit < 1 || limit > 100 {
		return nil, errors.New("valid recovery time and batch size 1–100 are required")
	}
	keys, err := s.recover(ctx, now, limit)
	return keys, classifyMessagingStoreError("recover", err)
}

func (s *Store) recover(ctx context.Context, now time.Time, limit int) ([]string, error) {
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
		delivery, restoreErr := restoreDelivery(ref.ID, saved)
		if restoreErr != nil {
			return restoreErr
		}
		if delivery.Reconcile(now) {
			version, versionErr := storedVersion(delivery.Version())
			if versionErr != nil {
				return versionErr
			}
			return tx.Update(ref, []sdk.Update{
				{Path: "state", Value: string(delivery.Status())},
				{Path: "reply_token", Value: ""},
				{Path: "updated_at", Value: now},
				{Path: "schema_version", Value: persistedSchemaVersion(saved.SchemaVersion)},
				{Path: "version", Value: version},
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
	return s.reconcile(ctx, ref, now)
}

func (s *Store) unknownExpired(ctx context.Context, ref *sdk.DocumentRef, now time.Time) error {
	return s.reconcile(ctx, ref, now)
}

func (s *Store) reconcile(ctx context.Context, ref *sdk.DocumentRef, now time.Time) error {
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
		delivery, restoreErr := restoreDelivery(ref.ID, saved)
		if restoreErr != nil {
			return restoreErr
		}
		if !delivery.Reconcile(now) {
			return nil
		}
		version, err := storedVersion(delivery.Version())
		if err != nil {
			return err
		}
		return tx.Update(ref, []sdk.Update{
			{Path: "state", Value: string(delivery.Status())},
			{Path: "reply_token", Value: ""},
			{Path: "lease_id", Value: ""},
			{Path: "lease_until", Value: time.Time{}},
			{Path: "updated_at", Value: now},
			{Path: "schema_version", Value: persistedSchemaVersion(saved.SchemaVersion)},
			{Path: "version", Value: version},
		})
	})
}

// Legacy jobs have no scope to compare. Never manufacture one from a retry.
func validateRedeliveryContext(saved storedJob, incoming messagingdelivery.DeliveryContext) error {
	if saved.SchemaVersion == 3 && saved.Context != incoming {
		return errors.New("redelivery scope does not match stored event")
	}
	return nil
}

func restoreDelivery(key string, saved storedJob) (aggregates.Delivery, error) {
	if saved.SchemaVersion != 0 && saved.SchemaVersion != 2 && saved.SchemaVersion != 3 {
		return aggregates.Delivery{}, errors.New("unsupported delivery schema version")
	}
	id, err := valueobjects.ParseDeliveryID(key)
	if err != nil {
		return aggregates.Delivery{}, err
	}
	var version uint64
	switch {
	case saved.SchemaVersion == 0 && saved.Version == 0:
		version = 1 // legacy documents predate aggregate versioning
	case saved.Version <= 0:
		return aggregates.Delivery{}, errors.New("invalid delivery version")
	default:
		version = uint64(saved.Version)
	}
	claimID, claimUntil := saved.LeaseID, saved.LeaseUntil
	if saved.SchemaVersion == 0 && valueobjects.Status(saved.State) != valueobjects.StatusSending {
		// Legacy completion cleared lease_id but retained lease_until. It is
		// safe to normalize only terminal/pending legacy snapshots because
		// their state cannot authorize a network send.
		claimID, claimUntil = "", time.Time{}
	}
	return aggregates.Restore(id, valueobjects.Status(saved.State), saved.ReplyUntil, claimID, claimUntil, version)
}

func persistedSchemaVersion(saved int) int {
	if saved == 3 {
		return 3
	}
	return 2
}

func storedVersion(version uint64) (int64, error) {
	if version > math.MaxInt64 {
		return 0, errors.New("delivery version exceeds Firestore integer range")
	}
	return int64(version), nil
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
