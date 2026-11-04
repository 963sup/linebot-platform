package outbound

import "context"

// BinaryContent is returned only to an explicit local operator command. It
// must never be logged or exposed by the public webhook.
type BinaryContent struct {
	Data        []byte
	ContentType string
}

type MessageContentReader interface {
	MessageContent(context.Context, string) (BinaryContent, error)
	MessageContentPreview(context.Context, string) (BinaryContent, error)
	MessageContentTranscoding(context.Context, string) (LINEResult, error)
}

type RichMenuImageManager interface {
	SetRichMenuImage(context.Context, string, BinaryContent) error
}

type RichMenuImageReader interface {
	RichMenuImage(context.Context, string) (BinaryContent, error)
}
