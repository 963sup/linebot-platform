package messagingdelivery

import (
	"strings"
	"testing"
)

func TestReplyPolicyAttendanceEntryAndCompatibility(t *testing.T) {
	policy, err := NewReplyPolicy("https://liff.line.me/2000000000-attendance")
	if err != nil {
		t.Fatal(err)
	}
	attendance := policy.ResponseForText("  打卡  ")
	if attendance.Kind != KindFlex || attendance.Flex.Action == nil || attendance.Flex.Action.Label != "開啟打卡" {
		t.Fatalf("attendance reply = %#v", attendance)
	}
	if got := policy.ResponseForText("卡片"); got.Kind != KindFlex || got.Flex.Action != nil {
		t.Fatalf("demo Flex reply changed: %#v", got)
	}
	for _, input := range []string{"签到", "签退", "hello", ""} {
		if got := policy.ResponseForText(input); got.Kind != KindText {
			t.Fatalf("input %q returned %#v", input, got)
		}
	}
	disabled, err := NewReplyPolicy("")
	if err != nil || disabled.ResponseForText("打卡").Kind != KindText {
		t.Fatalf("disabled policy = %#v, %v", disabled, err)
	}
}

func TestURIActionValidation(t *testing.T) {
	valid := URIAction{Label: "開啟打卡", URI: "https://liff.line.me/2000000000-attendance"}
	if err := valid.Validate(); err != nil {
		t.Fatal(err)
	}
	invalid := []URIAction{
		{Label: "", URI: valid.URI},
		{Label: strings.Repeat("字", 21), URI: valid.URI},
		{Label: " open", URI: valid.URI},
		{Label: valid.Label, URI: "javascript:alert(1)"},
		{Label: valid.Label, URI: "https://user@example.test/path"},
		{Label: valid.Label, URI: "https://example.test/path?q=1"},
		{Label: valid.Label, URI: "https://example.test/path#fragment"},
		{Label: valid.Label, URI: "https://example.test/" + strings.Repeat("x", 1000)},
	}
	for _, action := range invalid {
		if err := action.Validate(); err == nil {
			t.Fatalf("invalid action accepted: %#v", action)
		}
	}
}
