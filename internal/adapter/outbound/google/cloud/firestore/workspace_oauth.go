// Package firestore provides Google account-link state and encrypted
// credentials using Firestore transactions. It is independent of LINE inbox.
package firestore

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"time"

	sdk "cloud.google.com/go/firestore"
	identity "github.com/963sup/linebot-platform/internal/application/usecases/identity"
	"github.com/963sup/linebot-platform/internal/domain/identity/aggregates"
	domainevents "github.com/963sup/linebot-platform/internal/domain/identity/events"
	"github.com/963sup/linebot-platform/internal/domain/identity/factories"
	"github.com/963sup/linebot-platform/internal/domain/identity/repositories"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/identity/value-objects"
	"github.com/963sup/linebot-platform/internal/infrastructure/google/api/oauth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	oauthStateCollection     = "google_oauth_states_v1"
	oauthLinksCollection     = "google_oauth_connections_v1"
	identityOutboxCollection = "identity_outbox_v1"
)

var ErrConnectionConflict = errors.New("Google connection was updated concurrently")

// OAuthStore accepts a caller-owned Firestore client and an authenticated
// AES-256 keyring whose secret bytes come from an authorized secret source.
type OAuthStore struct {
	client *sdk.Client
	keys   *oauth.Keyring
	now    func() time.Time
}

var (
	_ identity.OAuthSessionRepository = (*OAuthStore)(nil)
	_ identity.ConnectionStore        = (*OAuthStore)(nil)
	_ identity.IntegrationOutbox      = (*OAuthStore)(nil)
	_ repositories.ConnectionReader   = (*OAuthStore)(nil)
)

func NewOAuthStore(client *sdk.Client, keys *oauth.Keyring) (*OAuthStore, error) {
	if client == nil || keys == nil {
		return nil, errors.New("Firestore client and OAuth keyring are required")
	}
	return &OAuthStore{client: client, keys: keys, now: time.Now}, nil
}

func opaqueStateID(state string) (string, error) {
	if len(state) != 43 {
		return "", errors.New("invalid OAuth state format")
	}
	raw, err := base64.RawURLEncoding.DecodeString(state)
	if err != nil || len(raw) != 32 {
		return "", errors.New("invalid OAuth state")
	}
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
	Ciphertext []byte    `firestore:"ciphertext"`
	KeyID      string    `firestore:"key_id"`
	ExpiresAt  time.Time `firestore:"expires_at"`
}

type storedConnection struct {
	Ciphertext []byte     `firestore:"ciphertext"`
	KeyID      string     `firestore:"key_id"`
	Active     bool       `firestore:"active"`
	Version    int64      `firestore:"version"`
	UpdatedAt  time.Time  `firestore:"updated_at"`
	ExpiresAt  *time.Time `firestore:"expires_at,omitempty"`
}

// storedIdentityEvent is a credential-free integration-event envelope. It is
// written in the same Firestore transaction as the AccountLink state. A
// separate dispatcher may publish it only after that transaction commits.
type storedIdentityEvent struct {
	EventID          string    `firestore:"event_id"`
	EventType        string    `firestore:"event_type"`
	SchemaVersion    int       `firestore:"schema_version"`
	AggregateID      string    `firestore:"aggregate_id"`
	AggregateVersion int64     `firestore:"aggregate_version"`
	OccurredAt       time.Time `firestore:"occurred_at"`
	State            string    `firestore:"state"`
	CorrelationID    string    `firestore:"correlation_id"`
	IdempotencyKey   string    `firestore:"idempotency_key"`
	LeaseID          string    `firestore:"lease_id,omitempty"`
	LeaseUntil       time.Time `firestore:"lease_until,omitempty"`
	NextDispatchAt   time.Time `firestore:"next_dispatch_at"`
	UpdatedAt        time.Time `firestore:"updated_at"`
	Attempts         int64     `firestore:"attempts"`
	ExpiresAt        time.Time `firestore:"expires_at"`
}

type identityOutboxDocument struct {
	id     string
	record storedIdentityEvent
}

func identityOutboxDocuments(aggregateID string, version int64, occurredAt time.Time, pending []domainevents.IdentityEvent) ([]identityOutboxDocument, error) {
	if aggregateID == "" || version <= 0 || occurredAt.IsZero() {
		return nil, errors.New("invalid identity outbox metadata")
	}
	documents := make([]identityOutboxDocument, 0, len(pending))
	for index, event := range pending {
		var eventType string
		switch current := event.(type) {
		case domainevents.AccountLinked:
			eventType = "identity.account-linked"
			if current.ID.Owner().String() == "" || current.ID.GoogleAccount().String() == "" {
				return nil, errors.New("invalid identity domain event")
			}
		case domainevents.AccountUnlinked:
			eventType = "identity.account-unlinked"
			if current.ID.Owner().String() == "" || current.ID.GoogleAccount().String() == "" {
				return nil, errors.New("invalid identity domain event")
			}
		default:
			return nil, fmt.Errorf("unsupported identity domain event %T", event)
		}
		source := "identity-event-v1:" + aggregateID + "\x00" + strconv.FormatInt(version, 10) + "\x00" + strconv.Itoa(index) + "\x00" + eventType
		digest := sha256.Sum256([]byte(source))
		eventID := hex.EncodeToString(digest[:])
		correlationSource := sha256.Sum256([]byte("identity-correlation-v1:" + aggregateID))
		correlationID := hex.EncodeToString(correlationSource[:])
		documents = append(documents, identityOutboxDocument{
			id: eventID,
			record: storedIdentityEvent{
				EventID: eventID, EventType: eventType, SchemaVersion: 1,
				AggregateID: aggregateID, AggregateVersion: version,
				OccurredAt: occurredAt.UTC(), State: "pending",
				CorrelationID: correlationID, IdempotencyKey: eventID,
				NextDispatchAt: occurredAt.UTC(), UpdatedAt: occurredAt.UTC(),
				ExpiresAt: occurredAt.UTC().Add(30 * 24 * time.Hour),
			},
		})
	}
	return documents, nil
}

const identityOutboxLease = 2 * time.Minute

func randomIdentityLease() (string, error) {
	var value [16]byte
	if _, err := io.ReadFull(rand.Reader, value[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(value[:]), nil
}

func integrationEvent(record storedIdentityEvent) identity.IntegrationEvent {
	return identity.IntegrationEvent{
		EventID: record.EventID, EventType: record.EventType, SchemaVersion: record.SchemaVersion,
		OccurredAt: record.OccurredAt, AggregateID: record.AggregateID,
		AggregateVersion: record.AggregateVersion, CorrelationID: record.CorrelationID,
		IdempotencyKey: record.IdempotencyKey,
	}
}

func identityLeaseExpired(record storedIdentityEvent, now time.Time) bool {
	return record.State == "publishing" && !record.LeaseUntil.IsZero() && !record.LeaseUntil.After(now)
}

func identityEventDue(record storedIdentityEvent, now time.Time) bool {
	return record.State == "pending" && !record.NextDispatchAt.After(now)
}

func identityConnectionVersionMatches(saved, expected int64) bool {
	return expected > 0 && saved == expected
}

// ReserveIdentityEvents recovers expired ambiguous leases as unknown, then
// atomically reserves due pending events. Expired publishes are never retried.
func (s *OAuthStore) ReserveIdentityEvents(ctx context.Context, now time.Time, limit int) ([]identity.ReservedIntegrationEvent, error) {
	if ctx == nil || now.IsZero() || limit <= 0 || limit > 100 {
		return nil, errors.New("invalid identity outbox reservation")
	}
	coll := s.client.Collection(identityOutboxCollection)
	expired, err := coll.Where("state", "==", "publishing").Where("lease_until", "<=", now).Limit(limit).Documents(ctx).GetAll()
	if err != nil {
		return nil, errors.New("unable to inspect identity outbox")
	}
	for _, document := range expired {
		if err := s.client.RunTransaction(ctx, func(ctx context.Context, tx *sdk.Transaction) error {
			current, err := tx.Get(document.Ref)
			if err != nil {
				return err
			}
			var record storedIdentityEvent
			if err := current.DataTo(&record); err != nil {
				return err
			}
			// A concurrent resolver may already have published the event after
			// the query snapshot. Never downgrade that terminal state.
			if !identityLeaseExpired(record, now) {
				return nil
			}
			return tx.Update(document.Ref, []sdk.Update{
				{Path: "state", Value: "unknown"}, {Path: "updated_at", Value: now},
				{Path: "lease_id", Value: sdk.Delete}, {Path: "lease_until", Value: sdk.Delete},
			})
		}); err != nil {
			return nil, errors.New("unable to recover identity outbox")
		}
	}
	due, err := coll.Where("state", "==", "pending").Where("next_dispatch_at", "<=", now).
		OrderBy("next_dispatch_at", sdk.Asc).Limit(limit).Documents(ctx).GetAll()
	if err != nil {
		return nil, errors.New("unable to inspect identity outbox")
	}
	reserved := make([]identity.ReservedIntegrationEvent, 0, len(due))
	for _, candidate := range due {
		leaseID, err := randomIdentityLease()
		if err != nil {
			return nil, errors.New("unable to reserve identity outbox")
		}
		var record storedIdentityEvent
		claimed := false
		err = s.client.RunTransaction(ctx, func(ctx context.Context, tx *sdk.Transaction) error {
			// Firestore may retry this callback. Reset all captured outputs on
			// every attempt so an aborted attempt cannot leak a successful claim.
			claimed = false
			record = storedIdentityEvent{}
			current, err := tx.Get(candidate.Ref)
			if err != nil {
				return err
			}
			if err := current.DataTo(&record); err != nil {
				return err
			}
			if !identityEventDue(record, now) {
				return nil
			}
			claimed = true
			return tx.Update(candidate.Ref, []sdk.Update{
				{Path: "state", Value: "publishing"}, {Path: "lease_id", Value: leaseID},
				{Path: "lease_until", Value: now.Add(identityOutboxLease)},
				{Path: "updated_at", Value: now}, {Path: "attempts", Value: record.Attempts + 1},
			})
		})
		if err != nil {
			return nil, errors.New("unable to reserve identity outbox")
		}
		if claimed {
			reserved = append(reserved, identity.ReservedIntegrationEvent{Event: integrationEvent(record), LeaseID: leaseID})
		}
	}
	return reserved, nil
}

func (s *OAuthStore) ResolveIdentityEvent(ctx context.Context, eventID, leaseID string, resolution identity.DispatchResolution, now time.Time) error {
	if ctx == nil || eventID == "" || leaseID == "" || now.IsZero() {
		return errors.New("invalid identity outbox resolution")
	}
	ref := s.client.Collection(identityOutboxCollection).Doc(eventID)
	err := s.client.RunTransaction(ctx, func(ctx context.Context, tx *sdk.Transaction) error {
		snapshot, err := tx.Get(ref)
		if err != nil {
			return err
		}
		var record storedIdentityEvent
		if err := snapshot.DataTo(&record); err != nil {
			return err
		}
		if record.State != "publishing" || record.LeaseID != leaseID {
			return errors.New("identity outbox lease conflict")
		}
		state := "unknown"
		updates := []sdk.Update{{Path: "updated_at", Value: now}, {Path: "lease_id", Value: sdk.Delete}, {Path: "lease_until", Value: sdk.Delete}}
		switch resolution {
		case identity.DispatchPublished:
			state = "published"
		case identity.DispatchRetryable:
			state = "pending"
			updates = append(updates, sdk.Update{Path: "next_dispatch_at", Value: now.Add(time.Minute)})
		case identity.DispatchUnknown:
			state = "unknown"
		default:
			return errors.New("invalid identity outbox resolution")
		}
		updates = append(updates, sdk.Update{Path: "state", Value: state})
		return tx.Update(ref, updates)
	})
	if err != nil {
		return errors.New("unable to resolve identity outbox")
	}
	return nil
}

func (s *OAuthStore) createIdentityOutbox(tx *sdk.Transaction, documents []identityOutboxDocument) error {
	for _, document := range documents {
		ref := s.client.Collection(identityOutboxCollection).Doc(document.id)
		if err := tx.Create(ref, document.record); err != nil {
			return err
		}
	}
	return nil
}

func (s *OAuthStore) disconnectedConnection(id string, authorization identity.OAuthAuthorization, version int64, now time.Time) (storedConnection, error) {
	if id == "" || authorization.Link.IsActive() || version <= 0 || now.IsZero() {
		return storedConnection{}, errors.New("invalid disconnected Google connection")
	}
	next := authorization
	next.Credentials = identity.OAuthCredentials{}
	next.Version = version
	key, sealed, err := s.encodeConnection(id, next)
	if err != nil {
		return storedConnection{}, err
	}
	expiresAt := now.UTC().Add(30 * 24 * time.Hour)
	return storedConnection{
		Ciphertext: sealed, KeyID: key, Active: false, Version: version,
		UpdatedAt: now.UTC(), ExpiresAt: &expiresAt,
	}, nil
}

func (s *OAuthStore) CreateSession(ctx context.Context, state string, session identity.OAuthSession) error {
	if ctx == nil {
		return errors.New("context required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	id, err := opaqueStateID(state)
	if err != nil {
		return err
	}
	if session.OwnerID == "" || len(session.Verifier) < 43 || len(session.Nonce) < 32 ||
		len(session.Scopes) == 0 || !session.ExpiresAt.After(s.now()) ||
		session.ExpiresAt.After(s.now().Add(11*time.Minute)) {
		return errors.New("invalid OAuth session")
	}
	plain, err := json.Marshal(session)
	if err != nil {
		return errors.New("cannot encode OAuth session")
	}
	keyID, sealed, err := s.keys.Seal("state:"+id, plain)
	if err != nil {
		return err
	}
	ref := s.client.Collection(oauthStateCollection).Doc(id)
	_, err = ref.Create(ctx, storedState{Ciphertext: sealed, KeyID: keyID, ExpiresAt: session.ExpiresAt})
	if err != nil {
		return errors.New("unable to create one-time OAuth session")
	}
	return nil
}

func (s *OAuthStore) ConsumeSession(ctx context.Context, state string) (identity.OAuthSession, error) {
	if ctx == nil {
		return identity.OAuthSession{}, errors.New("context required")
	}
	if err := ctx.Err(); err != nil {
		return identity.OAuthSession{}, err
	}
	id, err := opaqueStateID(state)
	if err != nil {
		return identity.OAuthSession{}, err
	}
	ref := s.client.Collection(oauthStateCollection).Doc(id)
	var saved storedState
	err = s.client.RunTransaction(ctx, func(ctx context.Context, tx *sdk.Transaction) error {
		saved = storedState{} // Firestore may restart this transaction.
		snap, err := tx.Get(ref)
		if err != nil {
			return err
		}
		if err := snap.DataTo(&saved); err != nil {
			return err
		}
		if !s.now().Before(saved.ExpiresAt) {
			return identity.ErrOAuthSessionNotFound
		}
		return tx.Delete(ref)
	})
	if err != nil {
		return identity.OAuthSession{}, oauthSessionReadError(ctx, err)
	}
	plain, err := s.keys.Open("state:"+id, saved.KeyID, saved.Ciphertext)
	if err != nil {
		return identity.OAuthSession{}, err
	}
	var session identity.OAuthSession
	if err := json.Unmarshal(plain, &session); err != nil {
		return identity.OAuthSession{}, errors.New("invalid persisted OAuth session")
	}
	if !s.now().Before(session.ExpiresAt) || session.OwnerID == "" || len(session.Verifier) < 43 {
		return identity.OAuthSession{}, errors.New("invalid persisted OAuth state")
	}
	return session, nil
}

func (s *OAuthStore) linkDoc(owner, subject string) (*sdk.DocumentRef, string, error) {
	id, err := connectionID(owner, subject)
	if err != nil {
		return nil, "", err
	}
	return s.client.Collection(oauthLinksCollection).Doc(id), id, nil
}

// legacyOAuthConnection is the exact encrypted JSON shape that existing
// Firestore documents use. It is private to the persistence adapter.
type legacyOAuthConnection struct {
	OwnerID       string
	GoogleSubject string
	Scopes        []string
	Credentials   identity.OAuthCredentials
	Version       int64
	Active        bool
}

func authorizationFromLegacy(record legacyOAuthConnection) (identity.OAuthAuthorization, error) {
	status := valueobjects.ConnectionDisconnected
	if record.Active {
		status = valueobjects.ConnectionActive
	}
	link, err := factories.RestoreLink(record.OwnerID, record.GoogleSubject, record.Scopes, status)
	if err != nil {
		return identity.OAuthAuthorization{}, err
	}
	return identity.OAuthAuthorization{Link: link, Credentials: record.Credentials, Version: record.Version}, nil
}

func legacyFromAuthorization(auth identity.OAuthAuthorization) (legacyOAuthConnection, error) {
	id, err := auth.Link.ConnectionID()
	if err != nil {
		return legacyOAuthConnection{}, err
	}
	return legacyOAuthConnection{
		OwnerID:       id.Owner().String(),
		GoogleSubject: id.GoogleAccount().String(),
		Scopes:        auth.Link.Scopes(),
		Credentials:   auth.Credentials,
		Version:       auth.Version,
		Active:        auth.Link.IsActive(),
	}, nil
}

func (s *OAuthStore) decodeConnection(id string, saved storedConnection) (identity.OAuthAuthorization, error) {
	if !saved.Active {
		return identity.OAuthAuthorization{}, errors.New("Google connection disabled")
	}
	plain, err := s.keys.Open("link:"+id, saved.KeyID, saved.Ciphertext)
	if err != nil {
		return identity.OAuthAuthorization{}, err
	}
	var legacy legacyOAuthConnection
	if err := json.Unmarshal(plain, &legacy); err != nil {
		return identity.OAuthAuthorization{}, errors.New("invalid encrypted Google connection")
	}
	legacy.Version = saved.Version
	legacy.Active = saved.Active
	auth, err := authorizationFromLegacy(legacy)
	if err != nil {
		return identity.OAuthAuthorization{}, errors.New("invalid persisted Google connection")
	}
	return auth, nil
}

func (s *OAuthStore) encodeConnection(id string, auth identity.OAuthAuthorization) (string, []byte, error) {
	legacy, err := legacyFromAuthorization(auth)
	if err != nil {
		return "", nil, errors.New("invalid Google connection payload")
	}
	plain, err := json.Marshal(legacy)
	if err != nil {
		return "", nil, errors.New("invalid Google connection payload")
	}
	return s.keys.Seal("link:"+id, plain)
}

func (s *OAuthStore) SaveConnection(ctx context.Context, conn identity.OAuthAuthorization) error {
	if ctx == nil {
		return errors.New("context required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	legacy, err := legacyFromAuthorization(conn)
	if err != nil {
		return errors.New("invalid Google OAuth connection")
	}
	ref, id, err := s.linkDoc(legacy.OwnerID, legacy.GoogleSubject)
	if err != nil {
		return err
	}
	if !conn.Link.IsActive() || conn.Credentials.AccessToken == "" {
		return errors.New("invalid Google OAuth connection")
	}
	now := s.now().UTC()
	pending := conn.Link.PendingEvents()
	err = s.client.RunTransaction(ctx, func(ctx context.Context, tx *sdk.Transaction) error {
		snap, err := tx.Get(ref)
		if status.Code(err) == codes.NotFound {
			if conn.Credentials.RefreshToken == "" {
				return errors.New("new connection requires offline consent")
			}
			newLink := conn
			newLink.Version = 1
			key, sealed, err := s.encodeConnection(id, newLink)
			if err != nil {
				return err
			}
			outbox, err := identityOutboxDocuments(id, 1, now, pending)
			if err != nil {
				return err
			}
			if err := tx.Create(ref, storedConnection{Ciphertext: sealed, KeyID: key, Active: true, Version: 1, UpdatedAt: now}); err != nil {
				return err
			}
			return s.createIdentityOutbox(tx, outbox)
		}
		if err != nil {
			return err
		}
		var old storedConnection
		if err := snap.DataTo(&old); err != nil {
			return err
		}
		next := conn
		if conn.Credentials.RefreshToken == "" && old.Active {
			previous, err := s.decodeConnection(id, old)
			if err != nil {
				return err
			}
			if previous.Link.Owner() != conn.Link.Owner() || previous.Link.GoogleAccount() != conn.Link.GoogleAccount() {
				return errors.New("connection identity conflict")
			}
			next.Credentials.RefreshToken = previous.Credentials.RefreshToken
		}
		if next.Credentials.RefreshToken == "" {
			return errors.New("offline consent required")
		}
		next.Version = old.Version + 1
		key, sealed, err := s.encodeConnection(id, next)
		if err != nil {
			return err
		}
		outbox, err := identityOutboxDocuments(id, next.Version, now, pending)
		if err != nil {
			return err
		}
		if err := tx.Set(ref, storedConnection{Ciphertext: sealed, KeyID: key, Active: true, Version: next.Version, UpdatedAt: now}); err != nil {
			return err
		}
		return s.createIdentityOutbox(tx, outbox)
	})
	if err != nil {
		return errors.New("unable to save Google connection")
	}
	return nil
}

func (s *OAuthStore) GetConnection(ctx context.Context, owner, subject string) (identity.OAuthAuthorization, error) {
	if ctx == nil {
		return identity.OAuthAuthorization{}, errors.New("context required")
	}
	if err := ctx.Err(); err != nil {
		return identity.OAuthAuthorization{}, err
	}
	ref, id, err := s.linkDoc(owner, subject)
	if err != nil {
		return identity.OAuthAuthorization{}, err
	}
	snap, err := ref.Get(ctx)
	if err != nil {
		return identity.OAuthAuthorization{}, oauthConnectionReadError(ctx, err)
	}
	var saved storedConnection
	if err := snap.DataTo(&saved); err != nil {
		return identity.OAuthAuthorization{}, errors.New("invalid Google connection record")
	}
	conn, err := s.decodeConnection(id, saved)
	if err != nil {
		return identity.OAuthAuthorization{}, err
	}
	if conn.Link.Owner().String() != owner || conn.Link.GoogleAccount().String() != subject {
		return identity.OAuthAuthorization{}, errors.New("Google connection identity mismatch")
	}
	return conn, nil
}

func oauthSessionReadError(ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	if errors.Is(err, identity.ErrOAuthSessionNotFound) || status.Code(err) == codes.NotFound {
		return identity.ErrOAuthSessionNotFound
	}
	return errors.New("OAuth session unavailable")
}

func oauthConnectionReadError(ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	if status.Code(err) == codes.NotFound {
		return identity.ErrOAuthConnectionNotFound
	}
	return errors.New("Google connection unavailable")
}

func (s *OAuthStore) UpdateCredentials(ctx context.Context, conn identity.OAuthAuthorization, expectedVersion int64) error {
	if ctx == nil {
		return errors.New("context required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	legacy, err := legacyFromAuthorization(conn)
	if err != nil || !conn.Link.IsActive() {
		return errors.New("invalid Google OAuth connection")
	}
	ref, id, err := s.linkDoc(legacy.OwnerID, legacy.GoogleSubject)
	if err != nil {
		return err
	}
	if expectedVersion <= 0 || conn.Credentials.AccessToken == "" {
		return errors.New("invalid OAuth refresh")
	}
	err = s.client.RunTransaction(ctx, func(ctx context.Context, tx *sdk.Transaction) error {
		snap, err := tx.Get(ref)
		if err != nil {
			return err
		}
		var saved storedConnection
		if err := snap.DataTo(&saved); err != nil {
			return err
		}
		if !saved.Active || saved.Version != expectedVersion {
			return ErrConnectionConflict
		}
		old, err := s.decodeConnection(id, saved)
		if err != nil {
			return err
		}
		if old.Link.Owner() != conn.Link.Owner() || old.Link.GoogleAccount() != conn.Link.GoogleAccount() {
			return errors.New("connection identity changed")
		}
		// Refresh is never an opportunity to silently expand authorized scopes.
		oldScopes, newScopes := old.Link.Scopes(), conn.Link.Scopes()
		if len(oldScopes) != len(newScopes) {
			return ErrConnectionConflict
		}
		for i, scope := range oldScopes {
			if newScopes[i] != scope {
				return ErrConnectionConflict
			}
		}
		next := conn
		if next.Credentials.RefreshToken == "" {
			next.Credentials.RefreshToken = old.Credentials.RefreshToken
		}
		next.Version = expectedVersion + 1
		key, sealed, err := s.encodeConnection(id, next)
		if err != nil {
			return err
		}
		return tx.Update(ref, []sdk.Update{
			{Path: "ciphertext", Value: sealed},
			{Path: "key_id", Value: key},
			{Path: "version", Value: next.Version},
			{Path: "updated_at", Value: s.now().UTC()},
		})
	})
	if errors.Is(err, ErrConnectionConflict) {
		return ErrConnectionConflict
	}
	if err != nil {
		return errors.New("unable to persist refreshed Google token")
	}
	return nil
}

// FindConnection provides the domain-owned read contract without exposing credentials.
func (s *OAuthStore) FindConnection(ctx context.Context, id valueobjects.ConnectionID) (aggregates.AccountLink, error) {
	if !id.Valid() {
		return aggregates.AccountLink{}, errors.New("invalid connection ID")
	}
	auth, err := s.GetConnection(ctx, id.Owner().String(), id.GoogleAccount().String())
	if err != nil {
		return aggregates.AccountLink{}, err
	}
	return auth.Link, nil
}

var (
	_ repositories.ConnectionReader   = (*OAuthStore)(nil)
	_ identity.OAuthSessionRepository = (*OAuthStore)(nil)
	_ identity.ConnectionStore        = (*OAuthStore)(nil)
	_ identity.IntegrationOutbox      = (*OAuthStore)(nil)
)

func (s *OAuthStore) DeactivateConnection(ctx context.Context, authorization identity.OAuthAuthorization) error {
	if ctx == nil {
		return errors.New("context required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	id, err := authorization.Link.ConnectionID()
	if err != nil || authorization.Link.IsActive() {
		return errors.New("disconnected Google connection required")
	}
	ref, aggregateID, err := s.linkDoc(id.Owner().String(), id.GoogleAccount().String())
	if err != nil {
		return err
	}
	pending := authorization.Link.PendingEvents()
	if len(pending) == 0 {
		return errors.New("account unlink event required")
	}
	now := s.now().UTC()
	err = s.client.RunTransaction(ctx, func(ctx context.Context, tx *sdk.Transaction) error {
		snap, err := tx.Get(ref)
		if err != nil {
			return err
		}
		var saved storedConnection
		if err := snap.DataTo(&saved); err != nil {
			return err
		}
		if !saved.Active {
			return errors.New("Google account already disconnected")
		}
		if !identityConnectionVersionMatches(saved.Version, authorization.Version) {
			return ErrConnectionConflict
		}
		persisted, err := s.decodeConnection(aggregateID, saved)
		if err != nil {
			return err
		}
		if persisted.Link.Owner() != authorization.Link.Owner() ||
			persisted.Link.GoogleAccount() != authorization.Link.GoogleAccount() ||
			!slices.Equal(persisted.Link.Scopes(), authorization.Link.Scopes()) {
			return errors.New("connection identity conflict")
		}
		nextVersion := saved.Version + 1
		outbox, err := identityOutboxDocuments(aggregateID, nextVersion, now, pending)
		if err != nil {
			return err
		}
		disconnected, err := s.disconnectedConnection(aggregateID, authorization, nextVersion, now)
		if err != nil {
			return err
		}
		if err := tx.Set(ref, disconnected); err != nil {
			return err
		}
		return s.createIdentityOutbox(tx, outbox)
	})
	if err != nil {
		if errors.Is(err, ErrConnectionConflict) {
			return ErrConnectionConflict
		}
		return errors.New("cannot deactivate Google connection")
	}
	return nil
}
