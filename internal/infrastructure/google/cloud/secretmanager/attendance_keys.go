package secretmanager

import (
	"context"
	"errors"

	sdk "cloud.google.com/go/secretmanager/apiv1"
	pb "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/963sup/linebot-platform/internal/infrastructure/attendancecrypto"
)

// LoadAttendanceKeyring loads pinned 32-byte AES keys for attendance location
// evidence. It never accepts the mutable "latest" alias or logs key material.
func LoadAttendanceKeyring(ctx context.Context, client *sdk.Client, active string, versions map[string]string) (*attendancecrypto.Keyring, error) {
	if ctx == nil || client == nil || active == "" || len(versions) == 0 {
		return nil, errors.New("attendance keyring requires Secret Manager client and versions")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	keys := make(map[string][]byte, len(versions))
	defer func() {
		for _, key := range keys {
			for i := range key {
				key[i] = 0
			}
		}
	}()
	for id, resource := range versions {
		if id == "" || !isPinnedSecretVersion(resource) {
			return nil, errors.New("attendance encryption key must use a pinned secret version")
		}
		reply, err := client.AccessSecretVersion(ctx, &pb.AccessSecretVersionRequest{Name: resource})
		if err != nil || reply == nil || reply.Payload == nil || len(reply.Payload.Data) != 32 {
			return nil, errors.New("attendance encryption secret is missing, inaccessible or invalid")
		}
		keys[id] = append([]byte(nil), reply.Payload.Data...)
	}
	keyring, err := attendancecrypto.NewKeyring(active, keys)
	return keyring, err
}
