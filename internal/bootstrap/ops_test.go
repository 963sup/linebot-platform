package bootstrap

import "testing"

func TestLINEOpsRuntimeRequiresToken(t *testing.T) {
	if _, err := NewLINEOpsRuntime(""); err == nil {
		t.Fatal("empty channel token must fail")
	}
	runtime, err := NewLINEOpsRuntime("test-token")
	if err != nil {
		t.Fatal(err)
	}
	if runtime.Messaging == nil || runtime.Profiles == nil {
		t.Fatal("CLI runtime must wire messaging and profiles")
	}
	runtime.Close()
	runtime.Close()
}
