package firestore

import "testing"

func TestTenantRefRejectsInvalidBoundary(t *testing.T) {
	for _, value := range []string{"", "a", "../other", "tenant/other", " Tenant", "UPPER"} {
		if err := ValidateTenantID(value); err == nil {
			t.Fatalf("ValidateTenantID(%q) accepted", value)
		}
	}
}
