package bootstrap

import "testing"

func TestLINETokenOpsRuntimeComposesCredentialLifecycle(t *testing.T) {
	runtime, err := NewLINETokenOpsRuntime()
	if err != nil {
		t.Fatalf("NewLINETokenOpsRuntime: %v", err)
	}
	defer runtime.Close()
	if runtime.Credentials == nil {
		t.Fatal("expected credential lifecycle service")
	}
}
