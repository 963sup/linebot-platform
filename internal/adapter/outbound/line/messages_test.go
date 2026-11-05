package line

import (
	"testing"

	messagingdelivery "github.com/963sup/linebot-platform/internal/application/usecases/messaging-delivery"
	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
)

func TestConvertMessages(t *testing.T) {
	result, err := convert([]messagingdelivery.Message{
		{Kind: messagingdelivery.KindText, Text: "hi"},
		{Kind: messagingdelivery.KindFlex, Flex: messagingdelivery.FlexCard{AltText: "alt", Title: "Title", Body: "Body"}},
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
	if _, ok := result[1].(*messaging_api.FlexMessage); !ok {
		t.Fatalf("got %T, want FlexMessage", result[1])
	}
}
