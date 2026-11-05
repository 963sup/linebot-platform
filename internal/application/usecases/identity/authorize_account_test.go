package identity

import (
	"context"
	"errors"
	"testing"

	"github.com/963sup/linebot-platform/internal/domain/identity/aggregates"
	"github.com/963sup/linebot-platform/internal/domain/identity/factories"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/identity/value-objects"
)

type connectionReaderFake struct {
	link aggregates.AccountLink
	err  error
}

func (f connectionReaderFake) FindConnection(context.Context, valueobjects.ConnectionID) (aggregates.AccountLink, error) {
	return f.link, f.err
}

func TestAuthorizeLinkedGoogleAccount(t *testing.T) {
	t.Parallel()
	scope := "https://www.googleapis.com/auth/tasks"
	link, err := factories.NewVerifiedLink("line-owner", "google-subject", []string{scope})
	if err != nil {
		t.Fatal(err)
	}
	id, err := AuthorizeLinkedGoogleAccount(context.Background(), connectionReaderFake{link: link}, "line-owner", "google-subject", []string{scope})
	if err != nil || !id.Valid() {
		t.Fatalf("AuthorizeLinkedGoogleAccount() = (%v, %v)", id, err)
	}
	if _, err := AuthorizeLinkedGoogleAccount(context.Background(), connectionReaderFake{link: link}, "line-owner", "google-subject", []string{"missing"}); err == nil {
		t.Fatal("missing scope accepted")
	}
	if _, err := AuthorizeLinkedGoogleAccount(context.Background(), connectionReaderFake{err: errors.New("read failed")}, "line-owner", "google-subject", []string{scope}); err == nil {
		t.Fatal("repository failure accepted")
	}
}
