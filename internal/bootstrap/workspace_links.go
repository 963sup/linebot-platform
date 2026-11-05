package bootstrap

import (
	"context"
	"errors"
	"net/http"

	firestore "cloud.google.com/go/firestore"
	authadapter "github.com/963sup/linebot-platform/internal/adapter/outbound/google/api/oauth"
	fsadapter "github.com/963sup/linebot-platform/internal/adapter/outbound/google/cloud/firestore"
	"github.com/963sup/linebot-platform/internal/application/usecases/identity"
	workspace "github.com/963sup/linebot-platform/internal/application/usecases/workspace-automation"
	oauthinfra "github.com/963sup/linebot-platform/internal/infrastructure/google/api/oauth"
)

// WorkspaceLinkRuntime is deliberately NOT wired into public LINE webhook.
// Identity verification and protected credentials must be provided by callers.
type WorkspaceLinkRuntime struct {
	Links *identity.LinkService
	store *fsadapter.OAuthStore
	flow  *oauthinfra.Flow
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
	service, err := identity.NewLinkService(provider, store, store)
	if err != nil {
		return nil, err
	}
	return &WorkspaceLinkRuntime{Links: service, store: store, flow: flow}, nil
}

// NewWorkspaceOperationsForAccount refuses unlinked/disabled Google accounts.
// Refresh uses a Firestore transaction/version guard across Cloud Run replicas.
func NewWorkspaceOperationsForAccount(ctx context.Context, rt *WorkspaceLinkRuntime, owner, subject string) (*workspace.Operations, error) {
	if rt == nil || rt.store == nil || rt.flow == nil {
		return nil, errors.New("Google workspace runtime missing")
	}
	if ctx == nil {
		return nil, errors.New("Google workspace request context required")
	}
	id, err := identity.AuthorizeLinkedGoogleAccount(ctx, rt.store, owner, subject, nil)
	if err != nil {
		return nil, errors.New("Google account not linked or disabled")
	}
	source, err := authadapter.NewPersistedTokenSource(ctx, rt.flow, rt.store, id.Owner().String(), id.GoogleAccount().String())
	if err != nil {
		return nil, err
	}
	return NewWorkspaceOperations(ctx, source)
}
