package bootstrap

import (
	"context"
	"errors"
	"net/http"

	firestore "cloud.google.com/go/firestore"
	authadapter "github.com/963sup/linebot-platform/internal/adapter/outbound/google/api/oauth"
	fsadapter "github.com/963sup/linebot-platform/internal/adapter/outbound/google/cloud/firestore"
	"github.com/963sup/linebot-platform/internal/application/usecases/workspace"
	oauthinfra "github.com/963sup/linebot-platform/internal/infrastructure/google/api/oauth"
)

// WorkspaceLinkRuntime is deliberately NOT wired into public LINE webhook.
// Identity verification and protected credentials must be provided by callers.
type WorkspaceLinkRuntime struct {
	Links *workspace.LinkService
	Store *fsadapter.OAuthStore
	Flow  *oauthinfra.Flow
}

func NewWorkspaceLinkRuntime(ctx context.Context, client *firestore.Client, keys *oauthinfra.Keyring,
	settings oauthinfra.Settings, transport *http.Client) (*WorkspaceLinkRuntime, error) {
	if ctx == nil || client == nil || keys == nil || transport == nil {
		return nil, errors.New("Google OAuth runtime requires authorized dependencies")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	flow, err := oauthinfra.NewFlow(settings)
	if err != nil {
		return nil, err
	}
	store, err := fsadapter.NewOAuthStore(client, keys)
	if err != nil {
		return nil, err
	}
	provider, err := authadapter.NewProvider(flow, transport)
	if err != nil {
		return nil, err
	}
	service, err := workspace.NewLinkService(provider, store, store)
	if err != nil {
		return nil, err
	}
	return &WorkspaceLinkRuntime{Links: service, Store: store, Flow: flow}, nil
}

// NewWorkspaceOperationsForAccount refuses unlinked/disabled Google accounts.
// Refresh uses a Firestore transaction/version guard across Cloud Run replicas.
func NewWorkspaceOperationsForAccount(ctx context.Context, rt *WorkspaceLinkRuntime, owner, subject string) (*workspace.Operations, error) {
	if rt == nil || rt.Store == nil || rt.Flow == nil {
		return nil, errors.New("Google workspace runtime missing")
	}
	if ctx == nil {
		return nil, errors.New("Google workspace request context required")
	}
	if _, err := rt.Store.GetConnection(ctx, owner, subject); err != nil {
		return nil, errors.New("Google account not linked or disabled")
	}
	source, err := authadapter.NewPersistedTokenSource(ctx, rt.Flow, rt.Store, owner, subject)
	if err != nil {
		return nil, err
	}
	return NewWorkspaceOperations(ctx, source)
}
