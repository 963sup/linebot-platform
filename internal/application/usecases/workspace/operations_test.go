package workspace

import (
	"context"
	"testing"
)

type fakeCalendar struct{ called int }

func (f *fakeCalendar) ExecuteCalendar(context.Context, Command) (Result, error) {
	f.called++
	return Result{Value: map[string]any{"ok": true}}, nil
}

type fakeTasks struct{ called int }

func (f *fakeTasks) ExecuteTasks(context.Context, Command) (Result, error) {
	f.called++
	return Result{Value: map[string]any{"ok": true}}, nil
}

func TestWorkspaceOperationsRequireExplicitApproval(t *testing.T) {
	cal, task := &fakeCalendar{}, &fakeTasks{}
	s, err := NewOperations(cal, task)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		api, op  string
		approval Approval
		pass     bool
	}{
		{"calendar", "events.list", Approval{Read: true}, true},
		{"tasks", "tasks.list", Approval{Read: true}, true},
		{"calendar", "events.insert", Approval{Read: true}, false},
		{"calendar", "events.insert", Approval{Apply: true, Confirm: "events.insert"}, true},
		{"tasks", "tasks.delete", Approval{Apply: true, Confirm: "tasks.delete"}, false},
		{"tasks", "tasks.delete", Approval{Apply: true, Confirm: "tasks.delete", ConfirmDangerous: "tasks.delete"}, true},
		{"calendar", "calendars.clear", Approval{Apply: true, Confirm: "calendars.clear"}, false},
		{"calendar", "not-an-api", Approval{Read: true}, false},
	} {
		_, err := s.Execute(context.Background(), tc.api, Command{Operation: tc.op}, tc.approval)
		if (err == nil) != tc.pass {
			t.Fatalf("%s %s pass=%t err=%v", tc.api, tc.op, tc.pass, err)
		}
	}
	if cal.called != 2 || task.called != 2 {
		t.Fatalf("unexpected dispatch: calendar=%d tasks=%d", cal.called, task.called)
	}
	if _, err := NewOperations(nil, task); err == nil {
		t.Fatal("nil calendar port accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.Execute(ctx, "tasks", Command{Operation: "tasks.list"}, Approval{Read: true}); err == nil {
		t.Fatal("cancelled context accepted")
	}
}
