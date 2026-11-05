package providerconfiguration

import (
	"context"
	"testing"
)

type webhookStub struct{ sets, tests int }

func (s *webhookStub) SetEndpoint(context.Context, string) error { s.sets++; return nil }
func (s *webhookStub) TestEndpoint(context.Context, string) (WebhookTestResult, error) {
	s.tests++
	return WebhookTestResult{Reachable: true, StatusCode: 200}, nil
}

func TestServiceValidatesBeforeCallingGateway(t *testing.T) {
	stub := &webhookStub{}
	service, err := New(stub)
	if err != nil {
		t.Fatal(err)
	}
	for _, endpoint := range []string{"", "http://example.com/hook", " https://example.com/hook", "https://u:p@example.com/hook", "https://example.com/hook?q=secret", "https://example.com/hook#secret"} {
		if err := service.ConfigureWebhook(context.Background(), endpoint); err == nil {
			t.Fatalf("accepted unsafe endpoint %q", endpoint)
		}
	}
	if stub.sets != 0 {
		t.Fatal("invalid endpoint reached gateway")
	}
	const endpoint = "https://example.com/hook"
	if err := service.ConfigureWebhook(context.Background(), endpoint); err != nil {
		t.Fatal(err)
	}
	result, err := service.TestWebhook(context.Background(), endpoint)
	if err != nil || !result.Reachable || stub.sets != 1 || stub.tests != 1 {
		t.Fatalf("unexpected result=%+v err=%v calls=%d/%d", result, err, stub.sets, stub.tests)
	}
}

func TestServiceRejectsNilAndCanceledContext(t *testing.T) {
	service, _ := New(&webhookStub{})
	if err := service.ConfigureWebhook(nil, "https://example.com/hook"); err == nil {
		t.Fatal("nil context accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.TestWebhook(ctx, "https://example.com/hook"); err == nil {
		t.Fatal("canceled context accepted")
	}
}
