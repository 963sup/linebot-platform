package secretmanager

import (
	"context"
	"testing"
)

func TestAttendanceKeyringRequiresPinnedSecretVersion(t *testing.T) {
	for _, resource := range []string{
		"", "projects/p/secrets/key/versions/latest", "projects/p/secrets/key/versions/v1",
		"projects/p/secrets/key", "projects/p/secrets/key/versions/1/extra",
	} {
		if isPinnedSecretVersion(resource) {
			t.Fatalf("unsafe resource accepted: %q", resource)
		}
	}
	if _, err := LoadAttendanceKeyring(context.Background(), nil, "v1", map[string]string{"v1": "projects/p/secrets/key/versions/1"}); err == nil {
		t.Fatal("nil Secret Manager client accepted")
	}
}
