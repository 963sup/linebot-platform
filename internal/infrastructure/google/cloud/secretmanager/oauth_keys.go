package secretmanager

import (
	"context"
	"errors"
	"strings"

	sdk "cloud.google.com/go/secretmanager/apiv1"
	pb "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/963sup/linebot-platform/internal/infrastructure/google/api/oauth"
)

// LoadOAuthKeyring retrieves pinned, versioned 32-byte AES keys using the
// caller's least-privilege Google Cloud service identity. It never logs keys.
func LoadOAuthKeyring(ctx context.Context, client *sdk.Client, active string, versions map[string]string) (*oauth.Keyring, error) {
	if ctx == nil || client == nil || active == "" || len(versions) == 0 {
		return nil, errors.New("OAuth keyring requires Secret Manager client and versions")
	}
	if err := ctx.Err(); err != nil { return nil, err }
	keys := make(map[string][]byte, len(versions))
	for id, resource := range versions {
		if id == "" || !isPinnedSecretVersion(resource) {
			return nil, errors.New("OAuth encryption key must use a pinned secret version")
		}
		reply, err := client.AccessSecretVersion(ctx, &pb.AccessSecretVersionRequest{Name:resource})
		if err != nil || reply == nil || reply.Payload == nil || len(reply.Payload.Data) != 32 {
			return nil, errors.New("OAuth encryption secret is missing, inaccessible or invalid")
		}
		keys[id] = append([]byte(nil), reply.Payload.Data...)
	}
	ring, err := oauth.NewKeyring(active, keys)
	// Forget intermediate raw key slices when key schedule has been created.
	for _, key := range keys { for i := range key { key[i] = 0 } }
	return ring, err
}

func isPinnedSecretVersion(name string) bool {
	parts := strings.Split(name, "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "secrets" || parts[4] != "versions" {
		return false
	}
	for _, p := range []string{parts[1], parts[3], parts[5]} {
		if p == "" || p == "." || p == ".." || strings.ContainsAny(p, "\x00 	\r\n") {
			return false
		}
	}
	if parts[5] == "latest" { return false }
	for _, char := range parts[5] { if char < '0' || char > '9' { return false } }
	return true
}
