package channel

import (
	"context"
	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
	"testing"
)

type fakeConfigurator struct {
	changed int
	tested  int
}

func (f *fakeConfigurator) SetWebhookEndpoint(context.Context, string) error { f.changed++; return nil }
func (f *fakeConfigurator) TestWebhookEndpoint(context.Context, string) (outbound.WebhookEndpointTest, error) {
	f.tested++
	return outbound.WebhookEndpointTest{Success: true}, nil
}
func TestWebhookConfigureRequiresHTTPSAndSafeInput(t *testing.T) {
	f := &fakeConfigurator{}
	s, err := NewWebhookConfigurator(f)
	if err != nil {
		t.Fatal(err)
	}
	for _, url := range []string{"", "http://example.com/x", "***example.com/x", "https://example.com/x?token=secret", " https://example.com/x", "https://example.com/x#secret"} {
		if err := s.Set(context.Background(), url); err == nil {
			t.Fatalf("unsafe webhook URL accepted: %q", url)
		}
	}
	if f.changed != 0 {
		t.Fatal("invalid URLs must never reach external API")
	}
	if err := s.Set(context.Background(), "https://example.com/webhook"); err != nil {
		t.Fatal(err)
	}
	if result, err := s.Test(context.Background(), "https://example.com/webhook"); err != nil || !result.Success {
		t.Fatalf("test result: %+v %v", result, err)
	}
	if f.changed != 1 || f.tested != 1 {
		t.Fatalf("unexpected call count %+v", f)
	}
}
