package cli

import (
	"bytes"
	"testing"
)

func TestMenusFailClosed(t *testing.T) {
	var output bytes.Buffer
	getenv := func(string) string { return "" }
	for _, args := range [][]string{
		{"--kind", "list"},
		{"--read", "--kind", "missing"},
		{"--read", "--kind", "menu"},
		{"--read", "--kind", "list", "--id", "extra"},
	} {
		if err := runMenus(testContext(), args, getenv, &output); err == nil {
			t.Fatalf("invalid menus query accepted: %v", args)
		}
	}
	if output.Len() != 0 {
		t.Fatal("rejected queries must not print data")
	}
}
