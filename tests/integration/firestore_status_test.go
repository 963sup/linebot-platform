//go:build integration

package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

func TestFirestoreReadOnlyOutboxInspection(t *testing.T) {
	_, store := emulatorStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	now := time.Now().UTC().Truncate(time.Millisecond)
	job := eventJob(30001, now)
	if _, err := store.Record(ctx, job); err != nil {
		t.Fatal(err)
	}
	report, err := store.Inspect(ctx, outbound.EventPending, 10, now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if report.Counts[outbound.EventPending] == 0 || len(report.Samples) == 0 {
		t.Fatal("read-only inspection omitted pending work")
	}
	for _, sample := range report.Samples {
		if len(sample.KeyHint) != 12 || sample.State != outbound.EventPending {
			t.Fatal("unsafe or malformed sample metadata")
		}
	}
	_, claimed, err := store.Claim(ctx, job.Key, "operator-test-lease", now.Add(time.Second))
	if err != nil || !claimed {
		t.Fatalf("claim failed: claimed=%t err=%v", claimed, err)
	}
	if err := store.Complete(ctx, job.Key, "operator-test-lease", outbound.DeliveryUnknown); err != nil {
		t.Fatal(err)
	}
	report, err = store.Inspect(ctx, outbound.EventUnknown, 0, now.Add(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if report.Counts[outbound.EventUnknown] == 0 || len(report.Samples) != 0 {
		t.Fatal("monitoring must report unknown count without event identifiers")
	}
	if _, err := store.Inspect(ctx, outbound.EventState("invalid"), 1, now); err == nil {
		t.Fatal("invalid status filter accepted")
	}
}
