package main

import "testing"

func TestPushRequiresExplicitSend(t *testing.T) {
	err := run([]string{"push", "--to", "U123", "--text", "hello"}, func(string) string { return "" })
	if err == nil {
		t.Fatal("push must require --send")
	}
}

func TestUUIDv4(t *testing.T) {
	u, err := uuidv4()
	if err != nil || len(u) != 36 || u[14] != '4' {
		t.Fatalf("uuid = %q, err = %v", u, err)
	}
}
