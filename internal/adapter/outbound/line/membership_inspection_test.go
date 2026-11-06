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

func TestSubscriptionPreservesAllProviderPlansAndLegacyProjection(t *testing.T) {
	const subscription = `{"membership":{"membershipId":42,"title":"Gold","description":"Benefits plan","benefits":["first","second"],"price":12.5,"currency":"JPY"},"user":{"membershipNo":7,"joinedTime":1700000000,"nextBillingDate":"2026-12-01","totalSubscriptionMonths":3}}`
	for _, tc := range []struct {
		name, payload string
		count         int
	}{
		{"empty", `{"subscriptions":[]}`, 0},
		{"one", `{"subscriptions":[` + subscription + `]}`, 1},
		{"multiple and optional children", `{"subscriptions":[` + subscription + `,{"membership":null,"user":null}]}`, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("method=%s", r.Method)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tc.payload))
			}))
			defer server.Close()
			result, err := contractSender(t, server).GetSubscription(context.Background(), membership.RecipientID("U123"))
			if err != nil {
				t.Fatal(err)
			}
			if result.RecipientID != "U123" || result.Membership != (membership.MembershipView{}) || result.Status != "" || !result.ValidUntil.IsZero() || result.ObservedAt.IsZero() || len(result.Subscriptions) != tc.count {
				t.Fatalf("subscription=%#v", result)
			}
			if tc.count == 0 {
				if result.Subscriptions == nil {
					t.Fatal("explicit empty subscriptions lost")
				}
				return
			}
			plan, user := result.Subscriptions[0].Membership, result.Subscriptions[0].User
			if plan == nil || plan.ID != "42" || plan.Name != "Gold" || plan.Description != "Benefits plan" || len(plan.Benefits) != 2 || plan.Benefits[1] != "second" || plan.Price != 12.5 || plan.Currency != "JPY" {
				t.Fatalf("plan=%#v", plan)
			}
			if user == nil || user.MembershipNo != 7 || user.JoinedAt.Unix() != 1700000000 || user.NextBillingDate != "2026-12-01" || user.TotalSubscriptionMonths != 3 {
				t.Fatalf("user=%#v", user)
			}
			if tc.count == 2 && (result.Subscriptions[1].Membership != nil || result.Subscriptions[1].User != nil) {
				t.Fatal("optional children invented")
			}
		})
	}
}

func TestSubscriptionPropagatesProviderFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"message":"unavailable"}`))
	}))
	defer server.Close()
	if _, err := contractSender(t, server).GetSubscription(context.Background(), membership.RecipientID("U123")); err == nil {
		t.Fatal("provider error accepted as subscription")
	}
}
