package line

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	contactdirectory "github.com/963sup/linebot-platform/internal/application/usecases/contact-directory"
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
		{"followers", func() error { _, e := s.ListFollowers(ctx, contactdirectory.PageRequest{Limit: 10}); return e }},
		{"group", func() error {
			_, e := s.GetConversationSummary(ctx, contactdirectory.RecipientReference{ID: "C123", Kind: contactdirectory.RecipientGroup})
			return e
		}},
		{"groupids", func() error {
			_, e := s.ListMembers(ctx, contactdirectory.RecipientReference{ID: "C123", Kind: contactdirectory.RecipientGroup}, contactdirectory.PageRequest{})
			return e
		}},
		{"groupmember", func() error {
			_, e := s.GetMemberProfile(ctx, contactdirectory.RecipientReference{ID: "C123", Kind: contactdirectory.RecipientGroup}, "U123")
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
