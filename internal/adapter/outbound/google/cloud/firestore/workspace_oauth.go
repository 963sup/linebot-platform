// Package firestore provides Google account-link state and encrypted
// credentials using Firestore transactions. It is independent of LINE inbox.
package firestore

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	sdk "cloud.google.com/go/firestore"
	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
	"github.com/963sup/linebot-platform/internal/infrastructure/google/api/oauth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	oauthStateCollection = "google_oauth_states_v1"
	oauthLinksCollection = "google_oauth_connections_v1"
)

var ErrConnectionConflict = errors.New("Google connection was updated concurrently")

// OAuthStore accepts a caller-owned Firestore client and an authenticated
// AES-256 keyring whose secret bytes come from an authorized secret source.
type OAuthStore struct {
	client *sdk.Client
	keys *oauth.Keyring
	now func() time.Time
}

func NewOAuthStore(client *sdk.Client, keys *oauth.Keyring) (*OAuthStore, error) {
	if client == nil || keys == nil {
		return nil, errors.New("Firestore client and OAuth keyring are required")
	}
	return &OAuthStore{client: client, keys: keys, now: time.Now}, nil
}

func opaqueStateID(state string) (string, error) {
	if len(state) != 43 { return "", errors.New("invalid OAuth state format") }
	raw, err := base64.RawURLEncoding.DecodeString(state)
	if err != nil || len(raw) != 32 { return "", errors.New("invalid OAuth state") }
	h := sha256.Sum256([]byte("google-oauth-state-v1:" + state))
	return hex.EncodeToString(h[:]), nil
}

func connectionID(owner, subject string) (string, error) {
	if owner == "" || subject == "" || strings.TrimSpace(owner) != owner || strings.TrimSpace(subject) != subject {
		return "", errors.New("invalid Google connection identity")
	}
	h := sha256.Sum256([]byte("google-oauth-link-v1:" + owner + "\x00" + subject))
	return hex.EncodeToString(h[:]), nil
}

type storedState struct {
	Ciphertext []byte `firestore:"ciphertext"`
	KeyID string `firestore:"key_id"`
	ExpiresAt time.Time `firestore:"expires_at"`
}

type storedConnection struct {
	Ciphertext []byte `firestore:"ciphertext"`
	KeyID string `firestore:"key_id"`
	Active bool `firestore:"active"`
	Version int64 `firestore:"version"`
	UpdatedAt time.Time `firestore:"updated_at"`
	ExpiresAt time.Time `firestore:"expires_at,omitempty"`
}

func (s *OAuthStore) CreateSession(ctx context.Context, state string, session outbound.OAuthSession) error {
	if ctx == nil { return errors.New("context required") }
	if err := ctx.Err(); err != nil { return err }
	id, err := opaqueStateID(state)
	if err != nil { return err }
	if session.OwnerID == "" || len(session.Verifier) < 43 || len(session.Nonce) < 32 ||
		len(session.Scopes) == 0 || !session.ExpiresAt.After(s.now()) ||
		session.ExpiresAt.After(s.now().Add(11*time.Minute)) {
		return errors.New("invalid OAuth session")
	}
	plain, err := json.Marshal(session)
	if err != nil { return errors.New("cannot encode OAuth session") }
	keyID, sealed, err := s.keys.Seal("state:"+id, plain)
	if err != nil { return err }
	ref := s.client.Collection(oauthStateCollection).Doc(id)
	_, err = ref.Create(ctx, storedState{Ciphertext:sealed, KeyID:keyID, ExpiresAt:session.ExpiresAt})
	if err != nil { return errors.New("unable to create one-time OAuth session") }
	return nil
}

func (s *OAuthStore) ConsumeSession(ctx context.Context, state string) (outbound.OAuthSession, error) {
	if ctx == nil { return outbound.OAuthSession{}, errors.New("context required") }
	if err := ctx.Err(); err != nil { return outbound.OAuthSession{}, err }
	id, err := opaqueStateID(state)
	if err != nil { return outbound.OAuthSession{}, err }
	ref := s.client.Collection(oauthStateCollection).Doc(id)
	var saved storedState
	err = s.client.RunTransaction(ctx, func(ctx context.Context, tx *sdk.Transaction) error {
		saved = storedState{} // Firestore may restart this transaction.
		snap, err := tx.Get(ref)
		if err != nil { return err }
		if err := snap.DataTo(&saved); err != nil { return err }
		if !s.now().Before(saved.ExpiresAt) {
			return errors.New("expired OAuth state")
		}
		return tx.Delete(ref)
	})
	if err != nil { return outbound.OAuthSession{}, errors.New("OAuth session expired, consumed or unavailable") }
	plain, err := s.keys.Open("state:"+id, saved.KeyID, saved.Ciphertext)
	if err != nil { return outbound.OAuthSession{}, err }
	var session outbound.OAuthSession
	if err := json.Unmarshal(plain, &session); err != nil { return outbound.OAuthSession{}, errors.New("invalid persisted OAuth session") }
	if !s.now().Before(session.ExpiresAt) || session.OwnerID == "" || len(session.Verifier) < 43 {
		return outbound.OAuthSession{}, errors.New("invalid persisted OAuth state")
	}
	return session, nil
}

func (s *OAuthStore) linkDoc(owner, subject string) (*sdk.DocumentRef, string, error) {
	id, err := connectionID(owner, subject)
	if err != nil { return nil, "", err }
	return s.client.Collection(oauthLinksCollection).Doc(id), id, nil
}

func (s *OAuthStore) decodeConnection(id string, saved storedConnection) (outbound.OAuthConnection, error) {
	if !saved.Active { return outbound.OAuthConnection{}, errors.New("Google connection disabled") }
	plain, err := s.keys.Open("link:"+id, saved.KeyID, saved.Ciphertext)
	if err != nil { return outbound.OAuthConnection{}, err }
	var conn outbound.OAuthConnection
	if err := json.Unmarshal(plain, &conn); err != nil {
		return outbound.OAuthConnection{}, errors.New("invalid encrypted Google connection")
	}
	conn.Version = saved.Version
	conn.Active = saved.Active
	return conn, nil
}

func (s *OAuthStore) encodeConnection(id string, conn outbound.OAuthConnection) (string, []byte, error) {
	plain, err := json.Marshal(conn)
	if err != nil { return "", nil, errors.New("invalid Google connection payload") }
	return s.keys.Seal("link:"+id, plain)
}

func (s *OAuthStore) SaveConnection(ctx context.Context, conn outbound.OAuthConnection) error {
	if ctx == nil { return errors.New("context required") }
	if err := ctx.Err(); err != nil { return err }
	ref, id, err := s.linkDoc(conn.OwnerID, conn.GoogleSubject)
	if err != nil { return err }
	if !conn.Active || conn.Credentials.AccessToken == "" || len(conn.Scopes) == 0 {
		return errors.New("invalid Google OAuth connection")
	}
	err = s.client.RunTransaction(ctx, func(ctx context.Context, tx *sdk.Transaction) error {
		snap, err := tx.Get(ref)
		if status.Code(err) == codes.NotFound {
			if conn.Credentials.RefreshToken == "" { return errors.New("new connection requires offline consent") }
			newLink := conn
			newLink.Version = 1
			key, sealed, err := s.encodeConnection(id, newLink)
			if err != nil { return err }
			return tx.Create(ref, storedConnection{Ciphertext:sealed,KeyID:key,Active:true,Version:1,UpdatedAt:s.now().UTC()})
		}
		if err != nil { return err }
		var old storedConnection
		if err := snap.DataTo(&old); err != nil { return err }
		next := conn
		if conn.Credentials.RefreshToken == "" && old.Active {
			previous, err := s.decodeConnection(id, old)
			if err != nil { return err }
			if previous.OwnerID != conn.OwnerID || previous.GoogleSubject != conn.GoogleSubject { return errors.New("connection identity conflict") }
			next.Credentials.RefreshToken = previous.Credentials.RefreshToken
		}
		if next.Credentials.RefreshToken == "" { return errors.New("offline consent required") }
		next.Version = old.Version + 1
		key, sealed, err := s.encodeConnection(id, next)
		if err != nil { return err }
		return tx.Set(ref, storedConnection{Ciphertext:sealed,KeyID:key,Active:true,Version:next.Version,UpdatedAt:s.now().UTC()})
	})
	if err != nil { return errors.New("unable to save Google connection") }
	return nil
}

func (s *OAuthStore) GetConnection(ctx context.Context, owner, subject string) (outbound.OAuthConnection, error) {
	if ctx == nil { return outbound.OAuthConnection{}, errors.New("context required") }
	if err := ctx.Err(); err != nil { return outbound.OAuthConnection{}, err }
	ref, id, err := s.linkDoc(owner, subject)
	if err != nil { return outbound.OAuthConnection{}, err }
	snap, err := ref.Get(ctx)
	if err != nil { return outbound.OAuthConnection{}, errors.New("Google connection unavailable") }
	var saved storedConnection
	if err := snap.DataTo(&saved); err != nil { return outbound.OAuthConnection{}, errors.New("invalid Google connection record") }
	conn, err := s.decodeConnection(id, saved)
	if err != nil { return outbound.OAuthConnection{}, err }
	if conn.OwnerID != owner || conn.GoogleSubject != subject { return outbound.OAuthConnection{}, errors.New("Google connection identity mismatch") }
	return conn, nil
}

func (s *OAuthStore) UpdateCredentials(ctx context.Context, conn outbound.OAuthConnection, expectedVersion int64) error {
	if ctx == nil { return errors.New("context required") }
	if err := ctx.Err(); err != nil { return err }
	ref, id, err := s.linkDoc(conn.OwnerID, conn.GoogleSubject)
	if err != nil { return err }
	if expectedVersion <= 0 || conn.Credentials.AccessToken == "" { return errors.New("invalid OAuth refresh") }
	err = s.client.RunTransaction(ctx, func(ctx context.Context, tx *sdk.Transaction) error {
		snap, err := tx.Get(ref)
		if err != nil { return err }
		var saved storedConnection
		if err := snap.DataTo(&saved); err != nil { return err }
		if !saved.Active || saved.Version != expectedVersion { return ErrConnectionConflict }
		old, err := s.decodeConnection(id, saved)
		if err != nil { return err }
		if old.OwnerID != conn.OwnerID || old.GoogleSubject != conn.GoogleSubject {
			return errors.New("connection identity changed")
		}
		// Refresh is never an opportunity to silently expand authorized scopes.
		if len(conn.Scopes) != len(old.Scopes) { return ErrConnectionConflict }
		for i, scope := range old.Scopes {
			if conn.Scopes[i] != scope { return ErrConnectionConflict }
		}
		next := conn
		if next.Credentials.RefreshToken == "" { next.Credentials.RefreshToken = old.Credentials.RefreshToken }
		next.Version = expectedVersion + 1
		next.Active = true
		key, sealed, err := s.encodeConnection(id, next)
		if err != nil { return err }
		return tx.Update(ref, []sdk.Update{
			{Path:"ciphertext",Value:sealed},
			{Path:"key_id",Value:key},
			{Path:"version",Value:next.Version},
			{Path:"updated_at",Value:s.now().UTC()},
		})
	})
	if errors.Is(err, ErrConnectionConflict) { return ErrConnectionConflict }
	if err != nil { return errors.New("unable to persist refreshed Google token") }
	return nil
}

func (s *OAuthStore) DeactivateConnection(ctx context.Context, owner, subject string) error {
	if ctx == nil { return errors.New("context required") }
	if err := ctx.Err(); err != nil { return err }
	ref, _, err := s.linkDoc(owner, subject)
	if err != nil { return err }
	err = s.client.RunTransaction(ctx, func(ctx context.Context, tx *sdk.Transaction) error {
		snap, err := tx.Get(ref)
		if err != nil { return err }
		var saved storedConnection
		if err := snap.DataTo(&saved); err != nil { return err }
		if !saved.Active { return errors.New("Google account already disconnected") }
		now := s.now().UTC()
		return tx.Update(ref, []sdk.Update{
			{Path:"active",Value:false},
			{Path:"ciphertext",Value:[]byte{}},
			{Path:"key_id",Value:""},
			{Path:"version",Value:saved.Version+1},
			{Path:"updated_at",Value:now},
			{Path:"expires_at",Value:now.Add(30*24*time.Hour)},
		})
	})
	if err != nil { return errors.New("cannot deactivate Google connection") }
	return nil
}
