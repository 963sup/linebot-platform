package line

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMembershipReadOnlySDKEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("unexpected membership method %s", r.Method)
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
		{"list", func() error { _, e := s.MembershipList(ctx); return e }},
		{"subscription", func() error { _, e := s.MembershipSubscription(ctx, "U123"); return e }},
		{"users", func() error { _, e := s.MembershipUsers(ctx, 123, "", 10); return e }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.fn(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
