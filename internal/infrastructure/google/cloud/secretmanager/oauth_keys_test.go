package secretmanager

import "testing"

func TestOAuthKeyringRequiresPinnedSecretVersion(t *testing.T) {
	for _, tc := range []struct {
		name  string
		valid bool
	}{
		{"projects/staging/secrets/google-oauth-aes/versions/3", true},
		{"projects/123456/secrets/google-oauth-aes/versions/1", true},
		{"projects/staging/secrets/google-oauth-aes/versions/latest", false},
		{"projects/staging/secrets/google-oauth-aes/versions/../x", false},
		{"projects/staging/secrets/google-oauth-aes/versions/-1", false},
		{"projects//secrets/google-oauth-aes/versions/1", false},
	} {
		if got := isPinnedSecretVersion(tc.name); got != tc.valid {
			t.Fatalf("%s valid=%t expected=%t", tc.name, got, tc.valid)
		}
	}
}
