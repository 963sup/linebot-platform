package main

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/963sup/linebot-platform/internal/application/usecases/workspace"
)

func TestWorkspaceExecutorPreservesContract(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var request workspace.Command
	var approval workspace.Approval
	payload := workspace.Result{Value: map[string]any{"ok": true}}
	wantPayload := []byte(`{"ok":true}`)
	sentinel := errors.New("upstream error")
	for _, wantErr := range []error{nil, sentinel} {
		calls := 0
		execute := workspaceExecutor(func(gotCtx context.Context, action string, gotRequest workspace.Command, gotApproval workspace.Approval) (workspace.Result, error) {
			calls++
			if gotCtx != ctx || action != "calendar.list" || !reflect.DeepEqual(gotRequest, request) || !reflect.DeepEqual(gotApproval, approval) {
				t.Fatal("workspace input or context changed")
			}
			return payload, wantErr
		})
		got, err := execute.Execute(ctx, "calendar.list", request, approval)
		want := wantPayload
		if wantErr != nil {
			want = nil
		}
		if calls != 1 || !bytes.Equal(got, want) || !errors.Is(err, wantErr) {
			t.Fatalf("result changed: calls=%d payload=%q error=%v", calls, got, err)
		}
	}
}
