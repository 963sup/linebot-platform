package line

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	accountlinktoken "github.com/963sup/linebot-platform/internal/application/usecases/account-link-token"
	conversationoperations "github.com/963sup/linebot-platform/internal/application/usecases/conversation-operations"
	insightsreporting "github.com/963sup/linebot-platform/internal/application/usecases/insights-reporting"
	messagingdelivery "github.com/963sup/linebot-platform/internal/application/usecases/messaging-delivery"
	"github.com/963sup/linebot-platform/internal/application/usecases/promotion"
)

func TestRemainingMessagingAPIEndpointsAreWired(t *testing.T) {
	var got string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Method + " " + r.URL.Path
		if r.Header.Get("Authorization") != "Bearer private-token" {
			t.Error("missing bearer token")
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/v2/bot/user/U123/linkToken" {
			_, _ = w.Write([]byte(`{"linkToken":"secret-link-token"}`))
			return
		}
		_, _ = w.Write([]byte("{}"))
	}))
	defer server.Close()
	s := contractSender(t, server)
	ctx := context.Background()
	request := messagingdelivery.PhoneDeliveryRequest{Payload: map[string]any{}}
	day := insightsreporting.DailyQuery{Day: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)}
	cases := []struct {
		name string
		want string
		call func() error
	}{
		{"leave-group", "POST /v2/bot/group/group-1/leave", func() error {
			return s.Leave(ctx, conversationoperations.Conversation{Kind: conversationoperations.Group, ID: "group-1"})
		}},
		{"leave-room", "POST /v2/bot/room/room-1/leave", func() error {
			return s.Leave(ctx, conversationoperations.Conversation{Kind: conversationoperations.Room, ID: "room-1"})
		}},
		{"loading", "POST /v2/bot/chat/loading/start", func() error {
			return s.ShowLoading(ctx, conversationoperations.LoadingIndicator{ChatID: "U123", Duration: 5 * time.Second})
		}},
		{"mark-read", "POST /v2/bot/message/markAsRead", func() error { return s.MarkRead(ctx, conversationoperations.ReadTarget{UserID: "U123"}) }},
		{"mark-read-token", "POST /v2/bot/chat/markAsRead", func() error { return s.MarkReadByToken(ctx, conversationoperations.ReadToken{Token: "token"}) }},
		{"issue-link-token", "POST /v2/bot/user/U123/linkToken", func() error { _, e := s.Issue(ctx, accountlinktoken.Subject{ID: "U123"}); return e }},
		{"push-phone", "POST /bot/pnp/push", func() error { return s.PushMessagesByPhone(ctx, request) }},
		{"push-template-phone", "POST /v2/bot/message/pnp/templated/push", func() error { return s.PushTemplatedMessagesByPhone(ctx, request) }},
		{"pnp-stats", "GET /v2/bot/message/delivery/pnp", func() error { _, e := s.PNPMessageStatistics(ctx, day); return e }},
		{"pnp-template-stats", "GET /v2/bot/message/delivery/pnp/templated", func() error { _, e := s.PNPTemplatedMessageStatistics(ctx, day); return e }},
		{"create-coupon", "POST /v2/bot/coupon", func() error {
			_, e := s.CreateCouponDocument(ctx, promotion.ProviderCouponDocument{Value: map[string]any{}})
			return e
		}},
		{"close-coupon", "PUT /v2/bot/coupon/abc/close", func() error { return s.CloseCoupon(ctx, promotion.CouponID("abc")) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got = ""
			if err := tc.call(); err != nil {
				t.Fatalf("call: %v", err)
			}
			if got != tc.want {
				t.Fatalf("endpoint = %q, want %q", got, tc.want)
			}
		})
	}
}
