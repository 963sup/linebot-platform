package line

import (
	"testing"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
)

func TestConvertMessages(t *testing.T) {
	result, err := convert([]outbound.Message{
		{Kind: outbound.KindText, Text: "hi"},
		{Kind: outbound.KindFlex, Flex: outbound.FlexCard{AltText: "alt", Title: "Title", Body: "Body"}},
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
