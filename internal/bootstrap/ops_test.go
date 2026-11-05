package bootstrap

import "testing"

func TestLINEOpsRuntimeRequiresToken(t *testing.T) {
	if _, err := NewLINEOpsRuntime(""); err == nil {
		t.Fatal("empty channel token must fail")
	}
	runtime, err := NewLINEOpsRuntime("test-token")
	if err != nil {
		t.Fatal(err)
	}
	if runtime.Messaging == nil || runtime.BulkMessaging == nil || runtime.Profiles == nil ||
		runtime.Contacts == nil || runtime.Validation == nil || runtime.WebhookAdmin == nil ||
		runtime.Channel == nil || runtime.DeliveryReports == nil || runtime.PhoneDelivery == nil ||
		runtime.Audiences == nil || runtime.AudienceManager == nil || runtime.Content == nil ||
		runtime.Coupons == nil || runtime.CouponCreator == nil ||
		runtime.Actions == nil || runtime.AccountLink == nil {
		t.Fatal("CLI runtime must wire all contract-backed LINE application services")
	}
	if runtime.Messaging != runtime.BulkMessaging {
		t.Fatal("messaging delivery service should be shared by direct and bulk delivery")
	}
	if runtime.Content != runtime.RichMenuImages || runtime.Content != runtime.MenuImageReader {
		t.Fatal("media transfer service should be shared by all binary-content operations")
	}
	runtime.Close()
	runtime.Close()
}
