package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestProfileLookupRequiresExplicitFlagAndCredentials(t *testing.T) {
	var output bytes.Buffer
	getenv := func(string) string { return "" }
	for _, args := range [][]string{
		{"--user", "U123"},
		{"--lookup"},
		{"--lookup", "--user", " U123"},
		{"--lookup", "--user", "U123", "extra"},
	} {
		if err := runProfile(args, getenv, &output); err == nil {
			t.Fatalf("invalid profile args unexpectedly accepted: %v", args)
		}
	}
	if output.Len() != 0 {
		t.Fatal("invalid lookup must not write profile data")
	}
	err := runProfile([]string{"--lookup", "--user", "U123"}, getenv, &output)
	if err == nil || !strings.Contains(err.Error(), "access token") {
		t.Fatalf("missing token must fail before network call: %v", err)
	}
	if output.Len() != 0 {
		t.Fatal("failed lookup must not write profile data")
	}
}
