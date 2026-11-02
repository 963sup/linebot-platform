//go:build integration

package integration

import (
	"context"
	"fmt"
	"net"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	sdk "cloud.google.com/go/firestore"
	storeout "github.com/963sup/linebot-platform/internal/adapter/outbound/google/cloud/firestore"
	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
	"github.com/963sup/linebot-platform/internal/application/usecases/ingestion"
)

// These tests only communicate with an explicitly configured LOCAL emulator.
// The default Go CI has no emulator and never contacts real Google Cloud.
func emulator(t *testing.T) (*sdk.Client, context.Context) {
	t.Helper()
	host := os.Getenv("FIRESTORE_EMULATOR_HOST")
	if host == "" {
		t.Skip("FIRESTORE_EMULATOR_HOST is not set; no external Firestore access is permitted")
	}
	addr, _, err := net.SplitHostPort(host)
	if err != nil || (addr != "localhost" && addr != "127.0.0.1" && addr != "::1") {
		t.Fatalf("refusing non-loopback Firestore emulator endpoint")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)
	client, err := sdk.NewClient(ctx, "demo-linebot-platform-acceptance")
	if err != nil {
		t.Fatalf("local emulator client: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client, ctx
}

func newRecord(t *testing.T, client *sdk.Client, ctx context.Context, id string, deadline time.Time) (outbound.EventJob, *storeout.Store) {
	t.Helper()
	key, err := ingestion.EventKey("emulator-line-channel", fmt.Sprintf("%s-%s-%d", t.Name(), id, time.Now().UnixNano()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = client.Collection("line_outbox_v1").Doc(key).Delete(c)
		_, _ = client.Collection("line_inbox_v1").Doc(key).Delete(c)
	})
	store, err := storeout.NewStore(client)
	if err != nil {
		t.Fatal(err)
	}
	createdAt := deadline.Add(-30 * time.Second).UTC()
	job := outbound.EventJob{
		Key: key, ReplyToken: "fake-emulator-reply", ReplyUntil: deadline.UTC(),
		CreatedAt: createdAt,
		Message:   outbound.Message{Kind: outbound.KindText, Text: "fixed demo response"},
	}
	created, err := store.Record(ctx, job)
	if err != nil || !created {
		t.Fatalf("atomic inbox/outbox record: created=%v err=%v", created, err)
	}
	return job, store
}

func TestEmulatorAtomicRecordAndSingleUseClaim(t *testing.T) {
	client, ctx := emulator(t)
	now := time.Now().UTC()
	job, store := newRecord(t, client, ctx, "dedup", now.Add(25*time.Second))

	duplicate := job
	duplicate.ReplyToken = "second-token-must-not-overwrite"
	created, err := store.Record(ctx, duplicate)
	if err != nil || created {
		t.Fatalf("duplicate event changed durable record: %v %v", created, err)
	}
	doc, err := client.Collection("line_outbox_v1").Doc(job.Key).Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	fields := doc.Data()
	if fields["reply_token"] != job.ReplyToken || fields["state"] != "pending" {
		t.Fatalf("unexpected first durable record state")
	}
	if _, exists := fields["raw_message"]; exists {
		t.Fatal("inbox/outbox must not include raw messages")
	}
	expiry, ok := fields["expires_at"].(time.Time)
	if !ok || !expiry.Equal(job.CreatedAt.Add(time.Hour)) {
		t.Fatalf("outbox TTL field is missing or inconsistent")
	}

	type claimResult struct {
		lease string
		ok    bool
		err   error
	}
	result := make(chan claimResult, 4)
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			lease := fmt.Sprintf("test-lease-%d", i)
			_, ok, err := store.Claim(ctx, job.Key, lease, now)
			result <- claimResult{lease: lease, ok: ok, err: err}
		}(i)
	}
	wg.Wait()
	close(result)

	var winner string
	for r := range result {
		if r.err != nil {
			t.Fatalf("emulator transaction claim: %v", r.err)
		}
		if r.ok {
			if winner != "" {
				t.Fatal("two workers claimed one durable event")
			}
			winner = r.lease
		}
	}
	if winner == "" {
		t.Fatal("no worker claimed the pending event")
	}
	if err := store.Complete(ctx, job.Key, winner, outbound.DeliverySent); err != nil {
		t.Fatal(err)
	}
	_, claimed, err := store.Claim(ctx, job.Key, "retry-lease", now)
	if err != nil || claimed {
		t.Fatalf("terminal event must not be claimed again: %v %v", claimed, err)
	}
	sent, err := client.Collection("line_outbox_v1").Doc(job.Key).Get(ctx)
	if err != nil || sent.Data()["state"] != "sent" || sent.Data()["reply_token"] != "" {
		t.Fatal("terminal result must clear the LINE reply token")
	}
}

func TestEmulatorReconcileAndNoUnsafeResend(t *testing.T) {
	client, ctx := emulator(t)
	now := time.Now().UTC()
	pending, store := newRecord(t, client, ctx, "pending", now.Add(30*time.Second))
	sending, _ := newRecord(t, client, ctx, "sending", now.Add(30*time.Second))
	expired, _ := newRecord(t, client, ctx, "expired", now.Add(-time.Second))

	_, claimed, err := store.Claim(ctx, sending.Key, "lease-before-crash", now)
	if err != nil || !claimed {
		t.Fatalf("failed to simulate worker crash after Claim: %v", err)
	}

	keys, err := store.Recover(ctx, now, 100)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(keys, pending.Key) || slices.Contains(keys, sending.Key) || slices.Contains(keys, expired.Key) {
		t.Fatalf("unexpected pending recovery keys")
	}
	_, err = store.Recover(ctx, now.Add(2*time.Minute), 100)
	if err != nil {
		t.Fatal(err)
	}
	for _, testCase := range []struct {
		key   string
		state string
	}{
		{pending.Key, "expired"},
		{sending.Key, "unknown"},
		{expired.Key, "expired"},
	} {
		doc, err := client.Collection("line_outbox_v1").Doc(testCase.key).Get(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if doc.Data()["state"] != testCase.state || strings.TrimSpace(fmt.Sprint(doc.Data()["reply_token"])) != "" {
			t.Fatalf("unexpected recovery outcome for %s", testCase.state)
		}
		_, claim, err := store.Claim(ctx, testCase.key, "new-worker", now.Add(3*time.Minute))
		if err != nil || claim {
			t.Fatal("recovery must never resend a terminal or uncertain LINE reply")
		}
	}
}
