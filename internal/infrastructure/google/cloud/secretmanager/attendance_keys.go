package secretmanager

import (
	"context"
	"errors"
	"path"
	"sort"
	"strconv"

	sdk "cloud.google.com/go/secretmanager/apiv1"
	pb "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/963sup/linebot-platform/internal/infrastructure/attendancecrypto"
	"google.golang.org/api/iterator"
)

// LoadManagedAttendanceKeyring discovers enabled versions of the dedicated
// Attendance key secret. New ciphertext uses the highest numeric version;
// numeric and legacy vN identifiers remain readable after rotation.
func LoadManagedAttendanceKeyring(ctx context.Context, client *sdk.Client, parent string) (*attendancecrypto.Keyring, error) {
	if ctx == nil || client == nil || parent == "" {
		return nil, errors.New("managed attendance keyring requires a secret")
	}
	type version struct {
		id       string
		resource string
		number   uint64
	}
	var versions []version
	it := client.ListSecretVersions(ctx, &pb.ListSecretVersionsRequest{Parent: parent, Filter: "state:ENABLED"})
	for {
		item, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, errors.New("list attendance encryption key versions")
		}
		id := path.Base(item.Name)
		number, err := strconv.ParseUint(id, 10, 64)
		if err != nil || number == 0 {
			return nil, errors.New("attendance encryption key has an invalid version")
		}
		versions = append(versions, version{id: id, resource: item.Name, number: number})
	}
	if len(versions) == 0 {
		return nil, errors.New("attendance encryption key has no enabled version")
	}
	sort.Slice(versions, func(i, j int) bool { return versions[i].number < versions[j].number })
	keys := make(map[string][]byte, len(versions)*2)
	defer func() {
		for _, key := range keys {
			for i := range key {
				key[i] = 0
			}
		}
	}()
	for _, version := range versions {
		reply, err := client.AccessSecretVersion(ctx, &pb.AccessSecretVersionRequest{Name: version.resource})
		if err != nil || reply == nil || reply.Payload == nil || len(reply.Payload.Data) != 32 {
			return nil, errors.New("attendance encryption secret is missing, inaccessible or invalid")
		}
		keys[version.id] = append([]byte(nil), reply.Payload.Data...)
		keys["v"+version.id] = append([]byte(nil), reply.Payload.Data...)
	}
	return attendancecrypto.NewKeyring(versions[len(versions)-1].id, keys)
}

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
