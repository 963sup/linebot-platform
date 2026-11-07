package messagingdelivery

import (
	"strings"
	"testing"
)

func TestReplyPolicyIntentFamilies(t *testing.T) {
	policy, err := NewReplyPolicy("https://liff.line.me/2000000000-attendance")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name         string
		input        string
		wantKind     MessageKind
		wantAction   bool
		wantText     string
		dontWantText string
	}{
		{name: "attendance trimmed", input: "  打卡  ", wantKind: KindFlex, wantAction: true, wantText: "上班或下班"},
		{name: "attendance clock in", input: "上班", wantKind: KindFlex, wantAction: true, wantText: "上班或下班"},
		{name: "attendance clock out", input: "下班", wantKind: KindFlex, wantAction: true, wantText: "上班或下班"},
		{name: "attendance generic", input: "出勤", wantKind: KindFlex, wantAction: true, wantText: "上班或下班"},
		{name: "attendance traditional sign in", input: "簽到", wantKind: KindFlex, wantAction: true, wantText: "上班或下班"},
		{name: "attendance simplified sign in", input: "签到", wantKind: KindFlex, wantAction: true, wantText: "上班或下班"},
		{name: "attendance traditional sign out", input: "簽退", wantKind: KindFlex, wantAction: true, wantText: "上班或下班"},
		{name: "attendance simplified sign out", input: "签退", wantKind: KindFlex, wantAction: true, wantText: "上班或下班"},
		{name: "help chinese", input: "功能", wantKind: KindFlex, wantAction: true, wantText: "今天班表"},
		{name: "help menu", input: "選單", wantKind: KindFlex, wantAction: true, wantText: "今天班表"},
		{name: "help english menu", input: "menu", wantKind: KindFlex, wantAction: true, wantText: "今天班表"},
		{name: "help english", input: "help", wantKind: KindFlex, wantAction: true, wantText: "今天班表"},
		{name: "help question", input: "？", wantKind: KindFlex, wantAction: true, wantText: "今天班表"},
		{name: "schedule today", input: "今天班表", wantKind: KindFlex, wantAction: true, wantText: "尚未連接"},
		{name: "schedule mine", input: "我的班表", wantKind: KindFlex, wantAction: true, wantText: "尚未連接"},
		{name: "status attendance", input: "打卡狀態", wantKind: KindFlex, wantAction: true, wantText: "尚未連接"},
		{name: "status mine", input: "我的狀態", wantKind: KindFlex, wantAction: true, wantText: "尚未連接"},
		{name: "greeting", input: "hello", wantKind: KindText, wantText: "打卡"},
		{name: "flex compatibility", input: "卡片", wantKind: KindFlex, wantText: "基礎架構"},
		{name: "unknown", input: "隨便問", wantKind: KindText, wantText: "目前支援", dontWantText: "訊息已收到"},
		{name: "empty", input: "", wantKind: KindText, wantText: "目前支援", dontWantText: "訊息已收到"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := policy.ResponseForText(tt.input)
			if err := got.Validate(); err != nil {
				t.Fatalf("input=%q got=%#v want=valid message: %v", tt.input, got, err)
			}
			if got.Kind != tt.wantKind {
				t.Fatalf("input=%q got.kind=%q got=%#v want.kind=%q", tt.input, got.Kind, got, tt.wantKind)
			}
			if (got.Flex.Action != nil) != tt.wantAction {
				t.Fatalf("input=%q got.action=%#v got=%#v want.action=%t", tt.input, got.Flex.Action, got, tt.wantAction)
			}
			content := got.Text
			if got.Kind == KindFlex {
				content = got.Flex.Body
			}
			if !strings.Contains(content, tt.wantText) {
				t.Fatalf("input=%q got.content=%q got=%#v want substring=%q", tt.input, content, got, tt.wantText)
			}
			if tt.dontWantText != "" && strings.Contains(content, tt.dontWantText) {
				t.Fatalf("input=%q got.content=%q got=%#v want without=%q", tt.input, content, got, tt.dontWantText)
			}
		})
	}
}

func TestReplyPolicyDisabledAttendanceEntry(t *testing.T) {
	disabled, err := NewReplyPolicy("")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name, input, wantText string
	}{
		{name: "attendance", input: "打卡", wantText: "尚未設定"},
		{name: "help", input: "功能", wantText: "尚未設定"},
		{name: "status", input: "今天班表", wantText: "聯絡管理員"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := disabled.ResponseForText(tt.input)
			if err := got.Validate(); err != nil {
				t.Fatalf("input=%q got=%#v want=valid message: %v", tt.input, got, err)
			}
			if got.Kind != KindText || !strings.Contains(got.Text, tt.wantText) {
				t.Fatalf("input=%q got=%#v want.kind=%q want substring=%q", tt.input, got, KindText, tt.wantText)
			}
		})
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
