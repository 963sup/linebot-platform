package bootstrap

import "testing"

func TestLINEOpsRuntimeFamiliesRequireToken(t *testing.T) {
	constructors := []struct {
		name string
		new  func(string) (*LINEOpsRuntime, error)
	}{
		{name: "messaging", new: newLINEMessagingOpsRuntime},
		{name: "insight", new: newLINEInsightOpsRuntime},
		{name: "blob", new: newLINEBlobOpsRuntime},
		{name: "audience", new: newLINEAudienceOpsRuntime},
	}
	for _, constructor := range constructors {
		t.Run(constructor.name, func(t *testing.T) {
			if _, err := constructor.new(""); err == nil {
				t.Fatal("empty channel token must fail")
			}
		})
	}
}

func TestLINEOpsRuntimeFamiliesWireOnlyTheirCapabilities(t *testing.T) {
	messaging, err := newLINEMessagingOpsRuntime("test-token")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = messaging.Close() })
	if messaging.Messaging == nil || messaging.BulkMessaging == nil || messaging.Profiles == nil ||
		messaging.Contacts == nil || messaging.Validation == nil || messaging.WebhookAdmin == nil ||
		messaging.Channel == nil || messaging.DeliveryReports == nil || messaging.PhoneDelivery == nil || messaging.PhoneReports == nil ||
		messaging.Coupons == nil || messaging.CouponCreator == nil || messaging.Actions == nil ||
		messaging.AccountLink == nil || messaging.Memberships == nil || messaging.Menus == nil || messaging.MenuCommands == nil ||
		messaging.Insights != nil || messaging.Content != nil ||
		messaging.Audiences != nil || messaging.AudienceManager != nil {
		t.Fatal("messaging runtime capability isolation is invalid")
	}
	if messaging.Messaging != messaging.BulkMessaging {
		t.Fatal("messaging delivery service should be shared by direct and bulk delivery")
	}

	insight, err := newLINEInsightOpsRuntime("test-token")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = insight.Close() })
	if insight.Insights == nil || insight.Messaging != nil || insight.Content != nil || insight.Audiences != nil {
		t.Fatal("insight runtime capability isolation is invalid")
	}

	blob, err := newLINEBlobOpsRuntime("test-token")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = blob.Close() })
	if blob.Content == nil || blob.Content != blob.RichMenuImages || blob.Content != blob.MenuImageReader ||
		blob.Messaging != nil || blob.Insights != nil || blob.Audiences != nil {
		t.Fatal("blob runtime capability isolation is invalid")
	}

	audience, err := newLINEAudienceOpsRuntime("test-token")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = audience.Close() })
	if audience.Audiences == nil || audience.AudienceManager == nil || audience.Messaging != nil ||
		audience.Insights != nil || audience.Content != nil {
		t.Fatal("audience runtime capability isolation is invalid")
	}
}
