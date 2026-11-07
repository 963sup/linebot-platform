package cli

import (
	"context"
	"testing"
)

func TestRetiredCommandsDoNotReadCredentials(t *testing.T) {
	for _, command := range []string{"workspace", "automation-requests", "audience-definitions", "interaction-surfaces", "promotions"} {
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
