package architecture

import (
	"os"
	"path/filepath"
	"testing"
)

// TestIdentityContract prevents reverting to a directory-only domain or
// silently reintroducing the duplicated policy and factory files.
func TestIdentityContract(t *testing.T) {
	root := filepath.Join("..", "..", "internal", "domain", "identity")
	required := []string{
		"entities/google_connection.go",
		"aggregates/account_link.go",
		"value-objects/line_user_id.go",
		"value-objects/google_account_id.go",
		"value-objects/connection_id.go",
		"value-objects/oauth_scope.go",
		"value-objects/connection_status.go",
		"policies/linking_policy.go",
		"policies/unlinking_policy.go",
		"specifications/connection_eligibility.go",
		"factories/connection_factory.go",
		"repositories/connection_repository.go",
		"events/account_linked.go",
		"events/account_unlinked.go",
	}
	for _, p := range required {
		if info, err := os.Stat(filepath.Join(root, p)); err != nil || info.IsDir() {
			t.Errorf("missing concrete identity domain component %s: %v", p, err)
		}
	}
	for _, p := range []string{
		"value-objects/identifiers.go",
		"value-objects/oauth_scopes.go",
		"policies/access.go",
		"factories/account_link.go",
		"repositories/account_link.go",
	} {
		if _, err := os.Stat(filepath.Join(root, p)); !os.IsNotExist(err) {
			t.Errorf("obsolete duplicated implementation still present: %s", p)
		}
	}

	for _, path := range []string{
		filepath.Join("..", "..", "internal", "adapter", "outbound", "google", "cloud", "firestore", "account_link_mapper.go"),
		filepath.Join("..", "..", "internal", "adapter", "outbound", "google", "cloud", "firestore", "account_link_mapper_test.go"),
	} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("obsolete identity mapper still exists: %s", path)
		}
	}
}
