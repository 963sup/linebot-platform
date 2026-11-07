//go:build integration

package integration_test

import (
	sdk "cloud.google.com/go/firestore"
	"context"
	"crypto/sha256"
	"encoding/hex"
	fsadapter "github.com/963sup/linebot-platform/internal/adapter/outbound/google/cloud/firestore"
	messagingdelivery "github.com/963sup/linebot-platform/internal/application/usecases/messaging-delivery"
	"testing"
	"time"
)

func TestFirestoreSharedBotCutoverPreservesOldEvents(t *testing.T) {
	client, store := emulatorStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	const destination = "U0123456789abcdef0123456789abcdef"
	namespace, err := fsadapter.EventKeyNamespace(ctx, client, destination)
	if err != nil || namespace != destination {
		t.Fatalf("new installation: %q %v", namespace, err)
	}
	hash := sha256.Sum256([]byte(destination))
	_, err = client.Collection("line_channel_connections_v1").Doc(hex.EncodeToString(hash[:])).Create(ctx, map[string]any{
		"tenant_id": "legacy-tenant", "connection_id": "line-main", "messaging_destination": destination,
	})
	if err != nil {
		t.Fatal(err)
	}
	namespace, err = fsadapter.EventKeyNamespace(ctx, client, destination)
	if err != nil || namespace != "legacy-tenant\x00line-main" {
		t.Fatalf("legacy namespace: %q %v", namespace, err)
	}
	key, _ := messagingdelivery.NewEventKey(namespace, "old-event")
	now := time.Now().UTC().Truncate(time.Millisecond)
	job := eventJob(1, now)
	job.Key = key.String()
	if _, err := store.Record(ctx, job); err != nil {
		t.Fatal(err)
	}
	outbox := client.Collection("line_outbox_v1").Doc(job.Key)
	// Seed an actual v3 Firestore payload, including fields removed from Go.
	_, err = outbox.Update(ctx, []sdk.Update{
		{Path: "schema_version", Value: 3},
		{Path: "delivery_context.TenantID", Value: "legacy-tenant"},
		{Path: "delivery_context.ConnectionID", Value: "line-main"},
	})
	if err != nil {
		t.Fatal(err)
	}
	job.ReplyUntil = now.Add(time.Minute)
	if created, err := store.Record(ctx, job); err != nil || created {
		t.Fatalf("legacy replay created=%v err=%v", created, err)
	}
	changed := job
	changed.Context.ConversationKind = messagingdelivery.ConversationGroup
	if _, err := store.Record(ctx, changed); err == nil {
		t.Fatal("legacy replay changed conversation")
	}
	if _, claimed, err := store.Claim(ctx, job.Key, "claim", now); err != nil || !claimed {
		t.Fatalf("claim=%v err=%v", claimed, err)
	}
	if err := store.Complete(ctx, job.Key, "claim", messagingdelivery.DeliverySent, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if created, err := store.Record(ctx, job); err != nil || created {
		t.Fatalf("sent replay created=%v err=%v", created, err)
	}
	if _, claimed, err := store.Claim(ctx, job.Key, "duplicate", now.Add(time.Second)); err != nil || claimed {
		t.Fatalf("sent event reclaimed=%v err=%v", claimed, err)
	}
	doc, err := outbox.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Data()["schema_version"] != int64(3) {
		t.Fatal("legacy schema silently promoted")
	}
}

func TestFirestoreNewSchemaAndLegacyRoomDrain(t *testing.T) {
	client, store := emulatorStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	now := time.Now().UTC().Truncate(time.Millisecond)
	job := eventJob(2, now)
	if _, err := store.Record(ctx, job); err != nil {
		t.Fatal(err)
	}
	outbox := client.Collection("line_outbox_v1").Doc(job.Key)
	doc, err := outbox.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Data()["schema_version"] != int64(4) {
		t.Fatal("new delivery must use schema v4")
	}
	scope, ok := doc.Data()["delivery_context"].(map[string]any)
	if !ok {
		t.Fatal("missing conversation scope")
	}
	if _, found := scope["TenantID"]; found {
		t.Fatal("new delivery stored TenantID")
	}
	if _, found := scope["ConnectionID"]; found {
		t.Fatal("new delivery stored ConnectionID")
	}
	_, err = outbox.Update(ctx, []sdk.Update{{Path: "schema_version", Value: 3}, {Path: "delivery_context.ConversationKind", Value: "room"}})
	if err != nil {
		t.Fatal(err)
	}
	legacy, claimed, err := store.Claim(ctx, job.Key, "drain", now)
	if err != nil || !claimed || legacy.Context.ConversationKind != "room" {
		t.Fatalf("legacy Room drain claimed=%v err=%v", claimed, err)
	}
}
