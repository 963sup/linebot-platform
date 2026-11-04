package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestDownloadChecksExactCommit(t *testing.T) {
	s := fixtureSource()
	calls := 0
	_, err := downloadSkills(s, func(name string, args ...string) ([]byte, error) {
		calls++
		joined := strings.Join(args, " ")
		if name != "git" {
			t.Fatalf("unexpected executable: %s", name)
		}
		if strings.Contains(joined, "rev-parse FETCH_HEAD") {
			return []byte(strings.Repeat("b", 40)), nil
		}
		if strings.Contains(joined, " archive ") {
			t.Fatal("archive used before commit verification")
		}
		return nil, nil
	})
	if err == nil || !strings.Contains(err.Error(), "differs") || calls != 3 {
		t.Fatalf("mismatched commit not rejected: %v calls=%d", err, calls)
	}
}

func TestDownloadUsesPinnedArchiveWithoutCheckout(t *testing.T) {
	s := fixtureSource()
	want := archive(t, "skills/example/SKILL.md")
	var commands []string
	got, err := downloadSkills(s, func(name string, args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		commands = append(commands, joined)
		if strings.Contains(joined, "rev-parse FETCH_HEAD") {
			return []byte(s.Commit + "\n"), nil
		}
		if strings.Contains(joined, " archive ") {
			if !strings.Contains(joined, s.Commit+" -- skills LICENSE") {
				t.Fatalf("unapproved archive command: %s", joined)
			}
			return want, nil
		}
		return nil, nil
	})
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("download failed: %v", err)
	}
	for _, command := range commands {
		if strings.Contains(command, "checkout") || strings.Contains(command, "install") {
			t.Fatalf("unexpected executable workflow: %s", command)
		}
	}
}

func TestMarketplaceRegistrationIsIdempotent(t *testing.T) {
	s := fixtureSource()
	listing := []byte(`[{"name":"google-plugins","source":"google/skills","ref":"` + s.Commit + `"}]`)
	calls := 0
	state, err := registerGoogle(s, func(_ string, args ...string) ([]byte, error) {
		calls++
		joined := strings.Join(args, " ")
		switch joined {
		case "plugin marketplace add --help":
			return []byte("--ref REF"), nil
		case "plugin marketplace list --json":
			return listing, nil
		default:
			t.Fatalf("existing marketplace should not be changed: %s", joined)
			return nil, nil
		}
	})
	if err != nil || state != "registered-pinned" || calls != 2 {
		t.Fatalf("unexpected state=%s err=%v calls=%d", state, err, calls)
	}
}

func TestMarketplaceStopsOnUnsupportedOrUnknownPin(t *testing.T) {
	for _, tc := range []struct{ name, help, listing, want string }{
		{"unsupported", "no pin flag", "[]", "unsupported"},
		{"different-pin", "--ref REF", `[{"name":"google-plugins","ref":"different"}]`, "blocked"},
		{"invalid-json", "--ref REF", `not json`, "unverified"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state, err := registerGoogle(fixtureSource(), func(_ string, args ...string) ([]byte, error) {
				joined := strings.Join(args, " ")
				if strings.HasSuffix(joined, "--help") {
					return []byte(tc.help), nil
				}
				if strings.Contains(joined, "list --json") {
					return []byte(tc.listing), nil
				}
				t.Fatal("attempted mutation for unsupported or conflicting marketplace")
				return nil, nil
			})
			if err == nil || state != tc.want {
				t.Fatalf("state=%s err=%v", state, err)
			}
		})
	}
}

func TestMarketplaceFailureDoesNotRetryOrFallback(t *testing.T) {
	calls := 0
	state, err := registerGoogle(fixtureSource(), func(_ string, args ...string) ([]byte, error) {
		calls++
		joined := strings.Join(args, " ")
		if strings.HasSuffix(joined, "--help") {
			return []byte("--ref REF"), nil
		}
		if strings.Contains(joined, "list --json") {
			return []byte("[]"), nil
		}
		return nil, os.ErrPermission
	})
	if err == nil || state != "failed" || calls != 3 {
		t.Fatalf("unexpected retry or auth bypass: %s %v calls=%d", state, err, calls)
	}
}

func TestOutputLimitCannotBeBypassed(t *testing.T) {
	var b limitedBuffer
	block := make([]byte, 1<<20)
	for i := 0; i < 64; i++ {
		if _, err := b.Write(block); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := b.Write([]byte{1}); err == nil {
		t.Fatal("output limit not enforced")
	}
}

func TestMarketplaceSourceCannotBeSpoofed(t *testing.T) {
	for _, record := range []string{
		`[{"name":"google-plugins","source":"https://evil.example/repo","ref":"approved-pin"}]`,
		`[{"name":"google-plugins","source":"google/skills","description":"approved-pin","ref":"other"}]`,
	} {
		found, pinned := marketplaceRecord([]byte(record), "google-plugins", "approved-pin")
		if !found || pinned {
			t.Fatal("unapproved source or unrelated field accepted")
		}
	}
	for _, value := range []string{`{}`, `{"unknown":[]}`, `[{"unexpected":"name"}]`, `[{"name":"x"},{"name":"x"}]`} {
		if _, err := marketplaceRecords([]byte(value)); err == nil {
			t.Errorf("unknown host schema accepted: %s", value)
		}
	}
}
