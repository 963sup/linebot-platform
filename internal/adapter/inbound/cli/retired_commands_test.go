package cli

import (
	"bytes"
	"context"
	"testing"
)

func TestRetiredCommandsDoNotReadCredentials(t *testing.T) {
	for _, command := range []string{"channel-connections", "workspace", "automation-requests", "audience-definitions", "interaction-surfaces", "promotions"} {
		t.Run(command, func(t *testing.T) {
			err := runWithContext(context.Background(), []string{command}, func(string) string {
				t.Fatal("retired command requested credentials")
				return ""
			})
			if err == nil {
				t.Fatal("retired command accepted")
			}
		})
	}
}

func TestRoomCommandsDoNotReadCredentials(t *testing.T) {
	getenv := func(string) string { t.Fatal("retired Room command requested credentials"); return "" }
	if err := runActions(context.Background(), []string{"--action", "leave-room", "--apply", "--confirm", "leave-room"}, getenv); err == nil {
		t.Fatal("Room mutation accepted")
	}
	for _, kind := range []string{"room-member-count", "room-members", "room-member-profile"} {
		if err := runContacts(context.Background(), []string{"--kind", kind, "--read"}, getenv, &bytes.Buffer{}); err == nil {
			t.Fatalf("Room query %s accepted", kind)
		}
	}
}
