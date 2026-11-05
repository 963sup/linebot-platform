package firestore

import (
	"context"
	"math"
	"reflect"
	"testing"
	"time"

	sdk "cloud.google.com/go/firestore"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/messaging-delivery/value-objects"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func TestStoredJobUsesFirestoreCompatibleIntegerTypes(t *testing.T) {
	t.Parallel()
	typ := reflect.TypeOf(storedJob{})
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		switch field.Type.Kind() {
		case reflect.Uint, reflect.Uint64, reflect.Uintptr:
			t.Fatalf("storedJob.%s uses unsupported Firestore integer type %s", field.Name, field.Type)
		}
	}
}

func TestStoredVersionChecksFirestoreIntegerRange(t *testing.T) {
	t.Parallel()
	if got, err := storedVersion(math.MaxInt64); err != nil || got != math.MaxInt64 {
		t.Fatalf("storedVersion(MaxInt64) = %d, %v", got, err)
	}
	if _, err := storedVersion(uint64(math.MaxInt64) + 1); err == nil {
		t.Fatal("storedVersion accepted a value outside Firestore's signed integer range")
	}
}

func TestCompleteRejectsMissingCompletionTime(t *testing.T) {
	t.Parallel()
	store := &Store{}
	key := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if err := store.Complete(context.Background(), key, "lease", valueobjects.StatusSent, time.Time{}); err == nil {
		t.Fatal("Complete accepted a missing completion time")
	}
}

func TestStoredJobUsesRealFirestoreEncoder(t *testing.T) {
	t.Parallel()
	client, err := sdk.NewClient(
		context.Background(),
		"offline-encoder-test",
		option.WithoutAuthentication(),
		option.WithEndpoint("127.0.0.1:1"),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })

	batch := client.Batch()
	batch.Create(client.Collection("encoder_probe").Doc("create"), storedJob{Version: 1})
	batch.Update(client.Collection("encoder_probe").Doc("update"), []sdk.Update{{Path: "version", Value: int64(2)}})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := batch.Commit(ctx); status.Code(err) != codes.Canceled {
		t.Fatalf("Firestore encoding failed before canceled RPC: %v", err)
	}
}

func TestRestoreDeliveryNormalizesLegacyTerminalLease(t *testing.T) {
	t.Parallel()
	key := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	delivery, err := restoreDelivery(key, storedJob{
		State:      string(valueobjects.StatusSent),
		ReplyUntil: time.Now().Add(time.Minute),
		LeaseUntil: time.Now().Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("restoreDelivery(legacy terminal) error = %v", err)
	}
	if delivery.Status() != valueobjects.StatusSent || delivery.ClaimID() != "" || !delivery.ClaimUntil().IsZero() || delivery.Version() != 1 {
		t.Fatalf("restored delivery = status %q claim %q/%v version %d", delivery.Status(), delivery.ClaimID(), delivery.ClaimUntil(), delivery.Version())
	}
}

func TestRestoreDeliveryRejectsUnknownSchema(t *testing.T) {
	t.Parallel()
	key := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	_, err := restoreDelivery(key, storedJob{SchemaVersion: 3, State: string(valueobjects.StatusPending), ReplyUntil: time.Now().Add(time.Minute), Version: 1})
	if err == nil {
		t.Fatal("restoreDelivery(unknown schema) error = nil")
	}
}

func TestRestoreDeliveryRejectsNegativeVersion(t *testing.T) {
	t.Parallel()
	key := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	_, err := restoreDelivery(key, storedJob{SchemaVersion: 2, Version: -1, State: string(valueobjects.StatusPending), ReplyUntil: time.Now().Add(time.Minute)})
	if err == nil {
		t.Fatal("restoreDelivery accepted a negative persistence version")
	}
}

func TestRestoreDeliveryRejectsMissingVersionInCurrentSchema(t *testing.T) {
	t.Parallel()
	key := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	_, err := restoreDelivery(key, storedJob{SchemaVersion: 2, State: string(valueobjects.StatusPending), ReplyUntil: time.Now().Add(time.Minute)})
	if err == nil {
		t.Fatal("restoreDelivery accepted a schema v2 document without a version")
	}
}
