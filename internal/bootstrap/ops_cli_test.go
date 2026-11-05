package bootstrap

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestBuildOpsCLIProvidesEveryRuntimeFactory(t *testing.T) {
	dependencies := reflect.ValueOf(BuildOpsCLI())
	dependencyType := dependencies.Type()
	for i := 0; i < dependencies.NumField(); i++ {
		if dependencies.Field(i).IsNil() {
			t.Errorf("runtime factory %s is nil", dependencyType.Field(i).Name)
		}
	}
}

func TestRunOpsPropagatesCanceledRootContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := RunOps(ctx, []string{"push"}, func(string) string { return "" }); !errors.Is(err, context.Canceled) {
		t.Fatalf("RunOps canceled context error = %v", err)
	}
}
