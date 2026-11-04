package workspace

import (
	"context"
	"errors"
	"testing"

	"github.com/963sup/linebot-platform/internal/domain/identity/aggregates"
	"github.com/963sup/linebot-platform/internal/domain/identity/factories"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/identity/value-objects"
)

type fakeConnectionReader struct {
	link aggregates.AccountLink
	err  error
}

func (f fakeConnectionReader) FindConnection(context.Context, valueobjects.ConnectionID) (aggregates.AccountLink, error) {
	return f.link, f.err
}

func TestAuthorizeLinkedGoogleAccount(t *testing.T) {
	active, err := factories.RestoreLink("line-owner", "google-subject",
		[]string{"https://www.googleapis.com/auth/calendar.events"}, valueobjects.ConnectionActive)
	if err != nil {
		t.Fatal(err)
	}
	inactive, err := factories.RestoreLink("line-owner", "google-subject",
		[]string{"https://www.googleapis.com/auth/calendar.events"}, valueobjects.ConnectionDisconnected)
	if err != nil {
		t.Fatal(err)
	}
	crossOwner, err := factories.RestoreLink("another-owner", "google-subject",
		[]string{"https://www.googleapis.com/auth/calendar.events"}, valueobjects.ConnectionActive)
	if err != nil {
		t.Fatal(err)
	}
	const calendarScope = "https://www.googleapis.com/auth/calendar.events"
	cases := []struct {
		name         string
		reader       fakeConnectionReader
		owner        string
		subject      string
		scopes       []string
		wantErr      bool
	}{
		{name: "authorized", reader: fakeConnectionReader{link: active}, owner: "line-owner", subject: "google-subject", scopes: []string{calendarScope}},
		{name: "valid link without scope requirement", reader: fakeConnectionReader{link: active}, owner: "line-owner", subject: "google-subject"},
		{name: "missing scope", reader: fakeConnectionReader{link: active}, owner: "line-owner", subject: "google-subject", scopes: []string{"forms.body"}, wantErr: true},
		{name: "disconnected", reader: fakeConnectionReader{link: inactive}, owner: "line-owner", subject: "google-subject", wantErr: true},
		{name: "repository mismatch", reader: fakeConnectionReader{link: crossOwner}, owner: "line-owner", subject: "google-subject", wantErr: true},
		{name: "repository failure", reader: fakeConnectionReader{err: errors.New("not found")}, owner: "line-owner", subject: "google-subject", wantErr: true},
		{name: "invalid owner", reader: fakeConnectionReader{link: active}, owner: " ", subject: "google-subject", wantErr: true},
		{name: "invalid subject", reader: fakeConnectionReader{link: active}, owner: "line-owner", subject: "", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			id, err := AuthorizeLinkedGoogleAccount(context.Background(), tc.reader, tc.owner, tc.subject, tc.scopes)
			if (err != nil) != tc.wantErr {
				t.Fatalf("AuthorizeLinkedGoogleAccount(%s, %s) error=%v, wantErr=%v", tc.owner, tc.subject, err, tc.wantErr)
			}
			if !tc.wantErr && (!id.Valid() || id.Owner().String() != tc.owner || id.GoogleAccount().String() != tc.subject) {
				t.Fatal("authorized identity was not preserved")
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := AuthorizeLinkedGoogleAccount(ctx, fakeConnectionReader{link: active}, "line-owner", "google-subject", nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled context was not propagated: %v", err)
	}
	if _, err := AuthorizeLinkedGoogleAccount(context.Background(), nil, "line-owner", "google-subject", nil); err == nil {
		t.Fatal("nil repository accepted")
	}
}
