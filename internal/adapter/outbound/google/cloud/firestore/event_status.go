package firestore

import (
	"context"
	"errors"
	"fmt"
	"time"

	sdk "cloud.google.com/go/firestore"
	"cloud.google.com/go/firestore/apiv1/firestorepb"
	messagingdelivery "github.com/963sup/linebot-platform/internal/application/usecases/messaging-delivery"
	"google.golang.org/api/iterator"
)

var _ messagingdelivery.EventStatusReader = (*Store)(nil)

// Inspect queries server-side counts and bounded metadata samples only.
// It never returns reply tokens, user IDs or message payloads.
func (s *Store) Inspect(ctx context.Context, state messagingdelivery.EventState, limit int, now time.Time) (messagingdelivery.EventStatusReport, error) {
	if ctx == nil {
		return messagingdelivery.EventStatusReport{}, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return messagingdelivery.EventStatusReport{}, err
	}
	if !state.Valid() || limit < 0 || limit > 100 || now.IsZero() {
		return messagingdelivery.EventStatusReport{}, errors.New("invalid outbox status query")
	}
	report, err := s.inspect(ctx, state, limit, now)
	return report, classifyMessagingStoreError("inspect", err)
}

func (s *Store) inspect(ctx context.Context, state messagingdelivery.EventState, limit int, now time.Time) (messagingdelivery.EventStatusReport, error) {
	now = now.UTC()
	coll := s.client.Collection(outboxCollection)
	report := messagingdelivery.EventStatusReport{
		GeneratedAt: now,
		Counts:      make(map[messagingdelivery.EventState]int64),
	}
	for _, current := range []messagingdelivery.EventState{
		messagingdelivery.EventPending, messagingdelivery.EventSending, messagingdelivery.EventSent,
		messagingdelivery.EventExpired, messagingdelivery.EventUnknown,
	} {
		q := coll.Where("state", "==", string(current))
		result, err := q.NewAggregationQuery().WithCount("total").Get(ctx)
		if err != nil {
			return messagingdelivery.EventStatusReport{}, fmt.Errorf("count outbox state %s: %w", current, err)
		}
		switch value := result["total"].(type) {
		case int64:
			report.Counts[current] = value
		case *firestorepb.Value: // Emulator returns a protobuf value.
			report.Counts[current] = value.GetIntegerValue()
		default:
			return messagingdelivery.EventStatusReport{}, fmt.Errorf("unexpected Firestore count result for %s", current)
		}
	}

	oldest := coll.Where("state", "==", "pending").
		OrderBy("created_at", sdk.Asc).Limit(1).Documents(ctx)
	defer oldest.Stop()
	doc, err := oldest.Next()
	if err != nil && err != iterator.Done {
		return messagingdelivery.EventStatusReport{}, err
	}
	if err == nil {
		var saved storedJob
		if err := doc.DataTo(&saved); err != nil {
			return messagingdelivery.EventStatusReport{}, err
		}
		if !saved.CreatedAt.IsZero() && now.After(saved.CreatedAt) {
			report.OldestPendingAgeSeconds = int64(now.Sub(saved.CreatedAt).Seconds())
		}
	}

	if limit == 0 {
		return report, nil // Monitoring never prints a correlatable event key.
	}
	samples := coll.Where("state", "==", string(state)).
		OrderBy("updated_at", sdk.Asc).Limit(limit).Documents(ctx)
	defer samples.Stop()
	for {
		doc, err := samples.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return messagingdelivery.EventStatusReport{}, err
		}
		if !validKey(doc.Ref.ID) {
			continue
		}
		var saved storedJob
		if err := doc.DataTo(&saved); err != nil {
			return messagingdelivery.EventStatusReport{}, err
		}
		report.Samples = append(report.Samples, messagingdelivery.EventStatusSample{
			KeyHint:   doc.Ref.ID[:12],
			State:     state,
			UpdatedAt: saved.UpdatedAt,
		})
	}
	return report, nil
}
