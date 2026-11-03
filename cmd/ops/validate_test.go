package main

import "testing"

func TestMessageValidationFailClosed(t *testing.T) {
	env := func(string) string { return "" }
	for _, args := range [][]string{
		{"--kind", "reply", "--text", "hello"},
		{"--kind", "bad", "--text", "hello", "--validate"},
		{"--kind", "push", "--validate"},
	} {
		if err := runValidate(args, env); err == nil {
			t.Fatalf("invalid validation accepted: %v", args)
		}
	}
}
