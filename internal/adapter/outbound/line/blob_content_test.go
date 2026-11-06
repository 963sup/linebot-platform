package line

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	mediatransfer "github.com/963sup/linebot-platform/internal/application/usecases/media-transfer"
	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
)

func TestBlobSenderUsesOfficialLINEBlobEndpoints(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer private-token" {
			t.Error("missing authorization")
		}
		paths = append(paths, r.Method+" "+r.URL.Path)
		if r.URL.Path == "/v2/bot/message/1/content/transcoding" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"succeeded"}`))
			return
		}
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("blob"))
	}))
	defer server.Close()
	client, err := messaging_api.NewMessagingApiBlobAPI("private-token", messaging_api.WithBlobEndpoint(server.URL), messaging_api.WithBlobHTTPClient(server.Client()))
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewBlobSender(client)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, call := range []func() error{
		func() error { _, e := s.FetchMessageContent(ctx, "1", mediatransfer.Original); return e },
		func() error { _, e := s.FetchMessageContent(ctx, "1", mediatransfer.Preview); return e },
		func() error { _, e := s.MessageContentStatus(ctx, "1"); return e },
		func() error { _, e := s.FetchPresentationImage(ctx, "richmenu-1"); return e },
		func() error {
			return s.StorePresentationImage(ctx, "richmenu-1", mediatransfer.BinaryContent{Data: []byte("png"), ContentType: "image/png"})
		},
	} {
		if err := call(); err != nil {
			t.Fatal(err)
		}
	}
	if len(paths) != 5 {
		t.Fatalf("paths = %v", paths)
	}
}

func TestReadBlobRejectsEmptyResponse(t *testing.T) {
	if _, err := readBlob(nil, nil); err == nil {
		t.Fatal("nil response accepted")
	}
	if _, err := readBlob(&http.Response{}, nil); err == nil {
		t.Fatal("nil response body accepted")
	}
}

func TestMessageContentStatusRejectsUnknownProviderStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"unknown"}`))
	}))
	defer server.Close()
	client, err := messaging_api.NewMessagingApiBlobAPI("private-token", messaging_api.WithBlobEndpoint(server.URL), messaging_api.WithBlobHTTPClient(server.Client()))
	if err != nil {
		t.Fatal(err)
	}
	sender, err := NewBlobSender(client)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sender.MessageContentStatus(context.Background(), "1"); err == nil {
		t.Fatal("unknown transcoding status accepted")
	}
}
