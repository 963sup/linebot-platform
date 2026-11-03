package firestore

import (
	"context"
	"errors"
	"fmt"
	"time"

	sdk "cloud.google.com/go/firestore"
	"cloud.google.com/go/firestore/apiv1/firestorepb"
	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
	"google.golang.org/api/iterator"
)

var _ outbound.EventStatusReader = (*Store)(nil)

// Inspect queries server-side counts and bounded metadata samples only.
// It never returns reply tokens, user IDs or message payloads.
func (s *Store) Inspect(ctx context.Context, state outbound.EventState, limit int, now time.Time) (outbound.EventStatusReport, error) {
	if ctx == nil {
		return outbound.EventStatusReport{}, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return outbound.EventStatusReport{}, err
	}
	if !state.Valid() || limit < 0 || limit > 100 {
		return outbound.EventStatusReport{}, errors.New("invalid outbox status query")
	}
	now = now.UTC()
	coll := s.client.Collection(outboxCollection)
	report := outbound.EventStatusReport{
		GeneratedAt: now,
		Counts: make(map[outbound.EventState]int64),
	}
	for _, current := range []outbound.EventState{
		outbound.EventPending, outbound.EventSending, outbound.EventSent,
		outbound.EventExpired, outbound.EventUnknown,
	} {
		q := coll.Where("state", "==", string(current))
		result, err := q.NewAggregationQuery().WithCount("total").Get(ctx)
		if err != nil {
			return outbound.EventStatusReport{}, fmt.Errorf("count outbox state %s: %w", current, err)
		}
		switch value := result["total"].(type) {
		case int64:
			report.Counts[current] = value
		case *firestorepb.Value: // Emulator returns a protobuf value.
			report.Counts[current] = value.GetIntegerValue()
		default:
			return outbound.EventStatusReport{}, fmt.Errorf("unexpected Firestore count result for %s", current)
		}
	}

	oldest := coll.Where("state", "==", "pending").
		OrderBy("created_at", sdk.Asc).Limit(1).Documents(ctx)
	defer oldest.Stop()
	doc, err := oldest.Next()
	if err != nil && err != iterator.Done {
		return outbound.EventStatusReport{}, err
	}
	if err == nil {
		var saved storedJob
		if err := doc.DataTo(&saved); err != nil {
			return outbound.EventStatusReport{}, err
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
			return outbound.EventStatusReport{}, err
		}
		if !validKey(doc.Ref.ID) {
			continue
		}
		var saved storedJob
		if err := doc.DataTo(&saved); err != nil {
			return outbound.EventStatusReport{}, err
		}
		report.Samples = append(report.Samples, outbound.EventStatusSample{
			KeyHint: doc.Ref.ID[:12],
			State: state,
			UpdatedAt: saved.UpdatedAt,
		})
	}
	return report, nil
}
