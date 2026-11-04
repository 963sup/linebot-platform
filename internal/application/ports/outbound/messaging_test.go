package outbound

import "testing"

func TestMessageValidation(t *testing.T) {
	samples := []struct {
		name string
		msg  Message
		ok   bool
	}{
		{"text", Message{Kind: KindText, Text: "hello"}, true},
		{"empty text", Message{Kind: KindText}, false},
		{"flex", Message{Kind: KindFlex, Flex: FlexCard{AltText: "alt", Title: "title", Body: "body"}}, true},
		{"flex no alt", Message{Kind: KindFlex, Flex: FlexCard{Title: "title", Body: "body"}}, false},
		{"unknown", Message{Kind: "image"}, false},
	}
	for _, tc := range samples {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.msg.Validate() == nil; got != tc.ok {
				t.Fatalf("valid = %t, want %t", got, tc.ok)
			}
		})
	}
}
