package line

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRemainingMessagingAPIEndpointsAreWired(t *testing.T) {
	var got string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Method + " " + r.URL.Path
		if r.Header.Get("Authorization") != "Bearer private-token" {
			t.Error("missing bearer token")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{}"))
	}))
	defer server.Close()
	s := contractSender(t, server)
	ctx := context.Background()
	request := json.RawMessage("{}")
	cases := []struct {
		name string
		want string
		call func() error
	}{
		{"leave-group", "POST /v2/bot/group/group-1/leave", func() error { return s.LeaveGroup(ctx, "group-1") }},
		{"leave-room", "POST /v2/bot/room/room-1/leave", func() error { return s.LeaveRoom(ctx, "room-1") }},
		{"loading", "POST /v2/bot/chat/loading/start", func() error { return s.ShowLoading(ctx, request) }},
		{"mark-read", "POST /v2/bot/message/markAsRead", func() error { return s.MarkMessagesRead(ctx, request) }},
		{"mark-read-token", "POST /v2/bot/chat/markAsRead", func() error { return s.MarkMessagesReadByToken(ctx, request) }},
		{"issue-link-token", "POST /v2/bot/user/U123/linkToken", func() error { _, e := s.IssueLinkToken(ctx, "U123"); return e }},
		{"push-phone", "POST /bot/pnp/push", func() error { return s.PushMessagesByPhone(ctx, request) }},
		{"push-template-phone", "POST /v2/bot/message/pnp/templated/push", func() error { return s.PushTemplatedMessagesByPhone(ctx, request) }},
		{"pnp-stats", "GET /v2/bot/message/delivery/pnp", func() error { _, e := s.PNPMessageStatistics(ctx, "20261101"); return e }},
		{"pnp-template-stats", "GET /v2/bot/message/delivery/pnp/templated", func() error { _, e := s.PNPTemplatedMessageStatistics(ctx, "20261101"); return e }},
		{"create-coupon", "POST /v2/bot/coupon", func() error { _, e := s.CreateCoupon(ctx, request); return e }},
		{"close-coupon", "PUT /v2/bot/coupon/abc/close", func() error { return s.CloseCoupon(ctx, "abc") }},
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
