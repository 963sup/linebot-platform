package line

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestContactsReadOnlySDKEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.Header.Get("Authorization") != "Bearer private-token" {
			t.Errorf("invalid contacts request")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{}"))
	}))
	defer server.Close()
	s := contractSender(t, server)
	ctx := context.Background()
	for _, tc := range []struct {
		name string
		fn   func() error
	}{
		{"followers", func() error { _, e := s.FollowerIDs(ctx, "", 10); return e }},
		{"group", func() error { _, e := s.GroupSummary(ctx, "C123"); return e }},
		{"groupcount", func() error { _, e := s.GroupMemberCount(ctx, "C123"); return e }},
		{"roomcount", func() error { _, e := s.RoomMemberCount(ctx, "R123"); return e }},
		{"groupids", func() error { _, e := s.GroupMemberIDs(ctx, "C123", ""); return e }},
		{"roomids", func() error { _, e := s.RoomMemberIDs(ctx, "R123", ""); return e }},
		{"groupmember", func() error { _, e := s.GroupMemberProfile(ctx, "C123", "U123"); return e }},
		{"roommember", func() error { _, e := s.RoomMemberProfile(ctx, "R123", "U123"); return e }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.fn(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
