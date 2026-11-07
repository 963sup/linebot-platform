package messagingdelivery

import (
	"strings"
	"testing"
)

func TestReplyPolicyMiniAppEntry(t *testing.T) {
	policy, err := NewReplyPolicy("https://miniapp.line.me/2000000000-example")
	if err != nil {
		t.Fatal(err)
	}
	reply := policy.ResponseForText("MINI App")
	if reply.Kind != KindFlex || reply.Flex.Action == nil || reply.Flex.Action.URI != "https://miniapp.line.me/2000000000-example" {
		t.Fatalf("reply=%#v", reply)
	}
	if help := policy.ResponseForText("功能"); help.Kind != KindFlex || !strings.Contains(help.Flex.Body, "MINI App") {
		t.Fatalf("help=%#v", help)
	}
}

func TestReplyPolicyWithoutMiniAppEntry(t *testing.T) {
	policy, err := NewReplyPolicy("")
	if err != nil {
		t.Fatal(err)
	}
	reply := policy.ResponseForText("miniapp")
	if reply.Kind != KindText || !strings.Contains(reply.Text, "尚未設定") {
		t.Fatalf("reply=%#v", reply)
	}
}

func TestReplyPolicyRejectsInvalidMiniAppEntry(t *testing.T) {
	if _, err := NewReplyPolicy("not a URL"); err == nil {
		t.Fatal("invalid URL accepted")
	}
}
