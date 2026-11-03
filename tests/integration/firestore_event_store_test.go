//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	sdk "cloud.google.com/go/firestore"
	fsadapter "github.com/963sup/linebot-platform/internal/adapter/outbound/google/cloud/firestore"
	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

func emulatorStore(t *testing.T) (*sdk.Client, *fsadapter.Store) {
	t.Helper()
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Fatal("FIRESTORE_EMULATOR_HOST is required for integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := sdk.NewClient(ctx, "linebot-emulator-ci")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	store, err := fsadapter.NewStore(client)
	if err != nil {
		t.Fatal(err)
	}
	return client, store
}

func eventJob(id int, received time.Time) outbound.EventJob {
	return outbound.EventJob{
		Key: fmt.Sprintf("%064x", id),
		ReplyToken: "emulator-token",
		Message: outbound.Message{Kind: outbound.KindText, Text: "fixed reply"},
		OccurredAt: received,
		CreatedAt: received,
		ReplyUntil: received.Add(45 * time.Second),
	}
}

func TestFirestoreTransactionAndConcurrentClaim(t *testing.T) {
	_, store := emulatorStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	now := time.Now().UTC()
	job := eventJob(10001, now)
	created, err := store.Record(ctx, job)
	if err != nil || !created {
		t.Fatalf("initial Inbox/Outbox transaction: created=%t err=%v", created, err)
	}
	job.CreatedAt = now.Add(10 * time.Second)
	job.ReplyUntil = now.Add(55 * time.Second)
	created, err = store.Record(ctx, job)
	if err != nil || created {
		t.Fatalf("redelivery must refresh, not create a duplicate: created=%t err=%v", created, err)
	}

	var count atomic.Int32
	var winningLease string
	var guard sync.Mutex
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			lease := fmt.Sprintf("worker-%d", i)
			claimedJob, claimed, err := store.Claim(ctx, job.Key, lease, now.Add(11*time.Second))
			if err != nil {
				t.Errorf("concurrent Claim: %v", err)
				return
			}
			if claimed {
				if claimedJob.ReplyUntil.Before(job.ReplyUntil) {
					t.Error("redelivery deadline was not refreshed")
				}
				count.Add(1)
				guard.Lock()
				winningLease = lease
				guard.Unlock()
			}
		}(i)
	}
	wg.Wait()
	if count.Load() != 1 {
		t.Fatalf("expected exactly one durable claim, got %d", count.Load())
	}
	if err := store.Complete(ctx, job.Key, winningLease, outbound.DeliverySent); err != nil {
		t.Fatal(err)
	}
	_, err = store.Record(ctx, outbound.EventJob{
		Key: job.Key, ReplyToken: job.ReplyToken, Message: job.Message,
		OccurredAt: now, CreatedAt: now.Add(20*time.Second),
		ReplyUntil: now.Add(time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, claimed, err := store.Claim(ctx, job.Key, "late-worker", now.Add(21*time.Second)); err != nil || claimed {
		t.Fatalf("sent record reopened: claimed=%t err=%v", claimed, err)
	}
}

func TestRecoverPaginatesByPersistedDueTime(t *testing.T) {
	_, store := emulatorStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	now := time.Now().UTC().Truncate(time.Second)
	for i := 1; i <= 115; i++ {
		job := eventJob(i, now.Add(-time.Second))
		job.ReplyUntil = now.Add(10 * time.Minute) // adapter-level fairness, not LINE token eligibility
		if _, err := store.Record(ctx, job); err != nil {
			t.Fatalf("create pending %d: %v", i, err)
		}
	}
	first, err := store.Recover(ctx, now, 100)
	if err != nil || len(first) != 100 {
		t.Fatalf("first due batch: len=%d err=%v", len(first), err)
	}
	second, err := store.Recover(ctx, now, 100)
	if err != nil || len(second) != 15 {
		t.Fatalf("second due batch: len=%d err=%v", len(second), err)
	}
	third, err := store.Recover(ctx, now, 100)
	if err != nil || len(third) != 0 {
		t.Fatalf("same-time scan must not starve future work: len=%d err=%v", len(third), err)
	}
	seen := map[string]bool{}
	for _, key := range append(first, second...) {
		if seen[key] {
			t.Fatalf("repeat in due batches: %s", key)
		}
		seen[key] = true
	}
}

func TestRecoverExpiresPendingAndMarksAmbiguousSendingUnknown(t *testing.T) {
	client, store := emulatorStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	now := time.Now().UTC()
	expired := eventJob(20001, now.Add(-2*time.Minute))
	if _, err := store.Record(ctx, expired); err != nil {
		t.Fatal(err)
	}
	inProgress := eventJob(20002, now.Add(-2*time.Minute-time.Second))
	if _, err := store.Record(ctx, inProgress); err != nil {
		t.Fatal(err)
	}
	_, claimed, err := store.Claim(ctx, inProgress.Key, "old-lease", now.Add(-2*time.Minute))
	if err != nil || !claimed {
		t.Fatalf("claim preceding a crash: claimed=%t err=%v", claimed, err)
	}
	keys, err := store.Recover(ctx, now, 100)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range keys {
		if key == expired.Key || key == inProgress.Key {
			t.Fatal("expired or ambiguous work was re-enqueued for a second send")
		}
	}
	for _, tc := range []struct{ key, want string }{
		{expired.Key, "expired"}, {inProgress.Key, "unknown"},
	} {
		doc, err := client.Collection("line_outbox_v1").Doc(tc.key).Get(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if got, _ := doc.Data()["state"].(string); got != tc.want {
			t.Fatalf("%s: state=%q want=%q", tc.key, got, tc.want)
		}
		if got, _ := doc.Data()["reply_token"].(string); got != "" {
			t.Fatal("terminal outbox retained its reply token")
		}
	}
}
