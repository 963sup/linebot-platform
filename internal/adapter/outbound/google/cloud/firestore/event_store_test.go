package firestore

import (
	"context"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	sdk "cloud.google.com/go/firestore"
	messagingdelivery "github.com/963sup/linebot-platform/internal/application/usecases/messaging-delivery"
	"github.com/963sup/linebot-platform/internal/domain/messaging-delivery/aggregates"
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

func TestMessagingStoreErrorClassificationIsSafe(t *testing.T) {
	t.Parallel()
	provider := status.Error(codes.Unavailable, "sensitive project and endpoint")
	err := classifyMessagingStoreError("claim", provider)
	if !errors.Is(err, ErrRepositoryUnavailable) || strings.Contains(err.Error(), "sensitive") {
		t.Fatalf("unsafe provider classification: %v", err)
	}
	if err := classifyMessagingStoreError("claim", messagingdelivery.ErrDeliveryInFlight); !errors.Is(err, messagingdelivery.ErrDeliveryInFlight) {
		t.Fatalf("in-flight sentinel lost: %v", err)
	}
	for _, sentinel := range []error{aggregates.ErrNotClaimable, aggregates.ErrClaimMismatch, aggregates.ErrTerminal} {
		if err := classifyMessagingStoreError("complete", sentinel); !errors.Is(err, sentinel) {
			t.Fatalf("domain sentinel %v lost: %v", sentinel, err)
		}
	}
	if err := classifyMessagingStoreError("claim", status.Error(codes.DeadlineExceeded, "sensitive")); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("deadline sentinel lost: %v", err)
	}
}

func TestMessagingStoreValidationIsNotClassifiedAsCorruption(t *testing.T) {
	t.Parallel()
	store := &Store{}
	if _, err := store.Recover(context.Background(), time.Time{}, 0); err == nil || errors.Is(err, ErrRepositoryCorrupt) || errors.Is(err, ErrRepositoryUnavailable) {
		t.Fatalf("Recover validation misclassified: %v", err)
	}
	if _, err := store.Inspect(context.Background(), messagingdelivery.EventState("invalid"), -1, time.Now()); err == nil || errors.Is(err, ErrRepositoryCorrupt) || errors.Is(err, ErrRepositoryUnavailable) {
		t.Fatalf("Inspect validation misclassified: %v", err)
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
	_, err := restoreDelivery(key, storedJob{SchemaVersion: 4, State: string(valueobjects.StatusPending), ReplyUntil: time.Now().Add(time.Minute), Version: 1})
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

func TestPersistedSchemaVersionDoesNotPromoteLegacyDocument(t *testing.T) {
	for input, want := range map[int]int{0: 2, 2: 2, 3: 3} {
		if got := persistedSchemaVersion(input); got != want {
			t.Fatalf("persistedSchemaVersion(%d)=%d want %d", input, got, want)
		}
	}
}
