package firestore

import (
	"errors"
	"regexp"
	"strings"

	sdk "cloud.google.com/go/firestore"
)

var tenantIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{1,62}$`)

// TenantRef returns the root document for a validated SaaS isolation boundary.
// Business repositories must build their collections below this document.
func TenantRef(client *sdk.Client, tenantID string) (*sdk.DocumentRef, error) {
	if client == nil {
		return nil, errors.New("Firestore client is required")
	}
	if err := ValidateTenantID(tenantID); err != nil {
		return nil, err
	}
	return client.Collection("tenants").Doc(tenantID), nil
}

func ValidateTenantID(tenantID string) error {
	if tenantID != strings.TrimSpace(tenantID) || !tenantIDPattern.MatchString(tenantID) {
		return errors.New("tenant ID is invalid")
	}
	return nil
}
