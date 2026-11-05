package bootstrap

import (
	"context"
	"errors"
	"reflect"
	"testing"

	workspaceautomation "github.com/963sup/linebot-platform/internal/application/usecases/workspace-automation"
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

func TestWorkspaceExecutorPreservesContract(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var request workspaceautomation.OperationCommand
	var approval workspaceautomation.OperationApproval
	payload := workspaceautomation.OperationResult{Payload: []byte(`{"ok":true}`)}
	sentinel := errors.New("upstream error")
	for _, wantErr := range []error{nil, sentinel} {
		calls := 0
		execute := workspaceExecutor(func(gotCtx context.Context, api workspaceautomation.API, gotRequest workspaceautomation.OperationCommand, gotApproval workspaceautomation.OperationApproval) (workspaceautomation.OperationResult, error) {
			calls++
			if gotCtx != ctx || api != workspaceautomation.APICalendar || !reflect.DeepEqual(gotRequest, request) || !reflect.DeepEqual(gotApproval, approval) {
				t.Fatal("workspace input or context changed")
			}
			return payload, wantErr
		})
		got, err := execute.Execute(ctx, workspaceautomation.APICalendar, request, approval)
		want := payload
		if wantErr != nil {
			want = workspaceautomation.OperationResult{}
		}
		if calls != 1 || !reflect.DeepEqual(got, want) || !errors.Is(err, wantErr) {
			t.Fatalf("result changed: calls=%d payload=%q error=%v", calls, got, err)
		}
	}
}
