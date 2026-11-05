package line

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/963sup/linebot-platform/internal/application/usecases/membership"
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
		{"list", func() error { _, e := s.ListMemberships(ctx, membership.PageRequest{}); return e }},
		{"subscription", func() error { _, e := s.GetSubscription(ctx, membership.RecipientID("U123")); return e }},
		{"users", func() error {
			_, e := s.ListMembershipMembers(ctx, membership.MembershipID("123"), membership.PageRequest{Limit: 10})
			return e
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.fn(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
