package contacts

import (
	"context"
	"testing"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

type contactsReaderStub struct{ operation string }

func (s *contactsReaderStub) result(operation string) (outbound.LINEResult, error) {
	s.operation = operation
	return outbound.LINEResult{Value: outbound.LINEObject{"operation": operation}}, nil
}
func (s *contactsReaderStub) FollowerIDs(context.Context, string, int32) (outbound.LINEResult, error) {
	return s.result("followers")
}
func (s *contactsReaderStub) GroupSummary(context.Context, string) (outbound.LINEResult, error) {
	return s.result("group-summary")
}
func (s *contactsReaderStub) GroupMemberCount(context.Context, string) (outbound.LINEResult, error) {
	return s.result("group-member-count")
}
func (s *contactsReaderStub) RoomMemberCount(context.Context, string) (outbound.LINEResult, error) {
	return s.result("room-member-count")
}
func (s *contactsReaderStub) GroupMemberIDs(context.Context, string, string) (outbound.LINEResult, error) {
	return s.result("group-members")
}
func (s *contactsReaderStub) RoomMemberIDs(context.Context, string, string) (outbound.LINEResult, error) {
	return s.result("room-members")
}
func (s *contactsReaderStub) GroupMemberProfile(context.Context, string, string) (outbound.LINEResult, error) {
	return s.result("group-member-profile")
}
func (s *contactsReaderStub) RoomMemberProfile(context.Context, string, string) (outbound.LINEResult, error) {
	return s.result("room-member-profile")
}

func TestInspectorReturnsTypedResult(t *testing.T) {
	reader := &contactsReaderStub{}
	inspector, err := NewInspector(reader)
	if err != nil {
		t.Fatal(err)
	}
	result, err := inspector.Inspect(context.Background(), Inspection{Kind: "followers", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if reader.operation != "followers" || result.Value == nil {
		t.Fatalf("unexpected typed result: operation=%q result=%#v", reader.operation, result)
	}
}

type contentReaderStub struct{}

func (contentReaderStub) MessageContent(context.Context, string) (outbound.BinaryContent, error) {
	return outbound.BinaryContent{Data: []byte("content")}, nil
}
func (contentReaderStub) MessageContentPreview(context.Context, string) (outbound.BinaryContent, error) {
	return outbound.BinaryContent{Data: []byte("preview")}, nil
}
func (contentReaderStub) MessageContentTranscoding(context.Context, string) (outbound.LINEResult, error) {
	return outbound.LINEResult{Value: outbound.LINEObject{"status": "processing"}}, nil
}

func TestContentInspectorSeparatesBinaryAndTypedResults(t *testing.T) {
	inspector, err := NewContentInspector(contentReaderStub{})
	if err != nil {
		t.Fatal(err)
	}
	binary, result, err := inspector.Inspect(context.Background(), ContentQuery{Kind: "transcoding", MessageID: "m1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(binary.Data) != 0 || result.Value == nil {
		t.Fatalf("unexpected transcoding result: binary=%#v result=%#v", binary, result)
	}
}

type accountLinkStub struct{}

func (accountLinkStub) IssueLinkTokenResult(context.Context, string) (outbound.LINEResult, error) {
	return outbound.LINEResult{Value: outbound.LINEObject{"linkToken": "secret"}}, nil
}

func TestAccountLinkerReturnsTypedResult(t *testing.T) {
	linker, err := NewAccountLinker(accountLinkStub{})
	if err != nil {
		t.Fatal(err)
	}
	result, err := linker.IssueLinkToken(context.Background(), "U1")
	if err != nil || result.Value == nil {
		t.Fatalf("unexpected link token result: %#v, %v", result, err)
	}
}
