package firestore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"regexp"
	"strings"

	sdk "cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// EventKeyNamespace reads retired routing metadata once at startup, solely to
// preserve deduplication keys across the shared-Bot cutover. Never delete or
// change this metadata while its namespace is in use. New installations need
// no registration and use the configured, verified Bot destination instead.
func EventKeyNamespace(ctx context.Context, client *sdk.Client, destination string) (string, error) {
	if ctx == nil || client == nil || strings.TrimSpace(destination) == "" {
		return "", errors.New("context, Firestore client and Bot destination are required")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	hash := sha256.Sum256([]byte(destination))
	doc, err := client.Collection("line_channel_connections_v1").Doc(hex.EncodeToString(hash[:])).Get(ctx)
	if err != nil {
		return eventKeyNamespace(destination, legacyEventNamespace{}, err)
	}
	var saved legacyEventNamespace
	if err := doc.DataTo(&saved); err != nil {
		return "", ErrRepositoryCorrupt
	}
	return eventKeyNamespace(destination, saved, nil)
}

// This is a persistence compatibility record, not a tenant business model.
type legacyEventNamespace struct {
	TenantID     string `firestore:"tenant_id"`
	ConnectionID string `firestore:"connection_id"`
	Destination  string `firestore:"messaging_destination"`
}

func eventKeyNamespace(destination string, saved legacyEventNamespace, readErr error) (string, error) {
	if status.Code(readErr) == codes.NotFound {
		return destination, nil
	}
	if readErr != nil {
		return "", classifyMessagingStoreError("read event namespace", readErr)
	}
	if saved.Destination != destination || !validLegacyNamespacePart(saved.TenantID) || !validLegacyNamespacePart(saved.ConnectionID) {
		return "", ErrRepositoryCorrupt
	}
	return saved.TenantID + "\x00" + saved.ConnectionID, nil
}

var legacyNamespacePart = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{1,62}$`)

func validLegacyNamespacePart(value string) bool { return legacyNamespacePart.MatchString(value) }
