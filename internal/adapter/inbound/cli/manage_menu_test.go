package cli

import "testing"

func TestMenuMutationsNeedExplicitConfirmation(t *testing.T) {
	getenv := func(string) string { return "" }
	for _, args := range [][]string{
		{"--action", "cancel-default"},
		{"--action", "cancel-default", "--apply"},
		{"--action", "delete-menu", "--apply", "--confirm", "cancel-default", "--menu-id", "richmenu-1"},
		{"--action", "link-user", "--apply", "--confirm", "link-user", "--user", "U123"},
	} {
		if err := runManageMenu(testContext(), args, getenv); err == nil {
			t.Fatalf("unconfirmed or invalid menu operation accepted: %v", args)
		}
	}
}
