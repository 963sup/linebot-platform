package bootstrap

import (
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
