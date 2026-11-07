package line

import (
	"testing"

	messagingdelivery "github.com/963sup/linebot-platform/internal/application/usecases/messaging-delivery"
	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
)

func TestConvertMessages(t *testing.T) {
	result, err := convert([]messagingdelivery.Message{
		{Kind: messagingdelivery.KindText, Text: "hi"},
		{Kind: messagingdelivery.KindFlex, Flex: messagingdelivery.FlexCard{
			AltText: "alt", Title: "Title", Body: "Body",
			Action: &messagingdelivery.URIAction{Label: "開啟 MINI App", URI: "https://miniapp.line.me/2000000000-example"},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 {
		t.Fatalf("messages = %d", len(result))
	}
	if _, ok := result[0].(messaging_api.TextMessage); !ok {
		t.Fatalf("got %T, want TextMessage", result[0])
	}
	flex, ok := result[1].(*messaging_api.FlexMessage)
	if !ok {
		t.Fatalf("got %T, want FlexMessage", result[1])
	}
	bubble, ok := flex.Contents.(messaging_api.FlexBubble)
	if !ok || bubble.Footer == nil || len(bubble.Footer.Contents) != 1 {
		t.Fatalf("Flex contents = %#v", flex.Contents)
	}
	button, ok := bubble.Footer.Contents[0].(*messaging_api.FlexButton)
	if !ok {
		t.Fatalf("footer component = %T", bubble.Footer.Contents[0])
	}
	action, ok := button.Action.(*messaging_api.UriAction)
	if !ok || action.Label != "開啟 MINI App" || action.Uri != "https://miniapp.line.me/2000000000-example" {
		t.Fatalf("button action = %#v", button.Action)
	}
}
