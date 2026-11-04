# DDD + Hexagonal Architecture migration contract

This branch establishes the **target domain first**, not a directory-only refactor.

## Naming contract
- Every bounded context stores value objects in `value-objects/`, never `valueobjects/` or `value_objects/`. The Go package identifier is `valueobjects`; the hyphen exists only in the import path.
- The architecture test walks all domain subdirectories to enforce this rule, not just the root scaffold.

## Dependency rules
- Identity domain owns account-link lifecycle, validated IDs, granted scopes and status decisions.
- Domain must never import LINE, Google, Firestore, application DTOs or infrastructure packages.
- OAuth PKCE/state/nonce verification, secrets, token refresh and storage are technical responsibilities.
- Application coordinates business use cases through small outbound ports.
- Adapters map SDK responses into plain application/domain models; bootstrap wires implementations.
- Identity is keyed by (verified LINE user, verified Google subject); multiple Google links per LINE owner remain supported pending a deliberate product rule.

## Required invariants
1. Invalid identity and scopes cannot produce valid AccountLink.
2. A disconnected account cannot authorize Google API operations, even when a token remains.
3. Caller ownership must be verified independently; a domain factory receiving strings is not proof of authentication.
4. OAuth state consumption stays one-time, expiring, bound to PKCE and nonce.
5. Credentials remain encrypted in a separate technical store, never in domain entities or logs.
6. Firestore writers must preserve uniqueness and concurrency control with atomic/versioned writes.
7. Preserve deployed Firestore/Cloud Tasks contracts through backward-compatible migration.

## Migration slices
1. **Completed:** introduce actual Identity value objects, aggregate, policy, factory, repository contract and isolated tests.
2. **Completed:** adapt Firestore account-link storage through a private legacy JSON codec while preserving encryption, transaction and version contracts.
3. **Completed:** refactor OAuth LinkService and TokenSource around Domain `AccountLink` while retaining user-scoped OAuth client isolation.
4. Add business-shaped Calendar/Tasks/Forms ports as real business use cases emerge; keep audited raw operations isolated for administrative use.
5. Expand Conversation/Audience bounded contexts only after identifying invariants. Add architecture checks and retire legacy DTOs once all callers migrate.

## Validation
Run `go test ./internal/domain/...`, `go test ./tests/architecture/...`, `go test ./...`, `go vet ./...` and GitHub Actions using the repository toolchain. No Cloud Run deployment or data migration is authorized by this slice.

## Slice 2: Identity integration (compatibility)
- `LinkService.Begin` validates identity and requested scopes with domain value objects before creating OAuth state.
- `LinkService.Complete` constructs the AccountLink aggregate after OAuth verification, and maps the aggregate back to the existing technical persistence contract.
- `Disconnect` validates the stored link and delegates state transition to AccountLink before the existing atomic Firestore deactivation.
- **Historical intermediate step, superseded by Slice 4:** Firestore previously validated application-level OAuthConnection DTOs through a separate mapper; the current implementation uses a private legacy codec and a Domain aggregate.
- No token or OAuth secret is stored in AccountLink; no data migration or Cloud Run release is part of this slice.

## Slice 3: Explicit domain ownership and consolidation
- Identity source of truth is `AccountLink` (aggregate root), containing one `GoogleConnection` entity; technical OAuth tokens and persistence version remain outside the domain.
- Value objects are separate `value-objects/line_user_id.go`, `google_account_id.go`, `connection_id.go`, `oauth_scope.go`, and `connection_status.go`.
- Consent scope matching moved from the OAuth use case to `policies.ValidateGrantedScopes`; unlink ownership/status validation moved to `policies.ValidateUnlink`.
- Operational connection eligibility moved to `specifications.CanUseGoogleAPI`.
- `factories.NewVerifiedLink` is for first verified linking; `factories.RestoreLink` is for database rehydration and does **not** create AccountLinked events.
- The Firestore adapter implements `repositories.ConnectionReader` using existing encrypted state; Bootstrap checks domain eligibility before issuing a user-scoped token source.
- **Superseded by Slice 4:** the application-level `outbound.OAuthConnection` was removed. Only the private Firestore `legacyOAuthConnection` record retains the old encrypted JSON keys for backwards-compatible reads and writes.
- Domain events are staged in memory but deliberately **not published** before transaction commit. Integrating a durable event publisher must be a separately tested slice.

## Slice 4: retire duplicate application OAuthConnection (PR #28)
- Deleted the duplicated `outbound.OAuthConnection` DTO and replaced it with `OAuthAuthorization{Link aggregates.AccountLink, Credentials, Version}`.
- The only owner/subject/scopes/active status in application is the Domain Aggregate; use cases and OAuth TokenSource no longer read parallel primitive business fields.
- Deleted `account_link_mapper.go` and its generic mapper test. The sole persistence codec now exists inside `workspace_oauth.go`, using a private `legacyOAuthConnection` wire struct to read/write the **identical** previously encrypted JSON keys.
- Added legacy JSON and encrypted payload contract tests; verified inactive records stay inaccessible and token refresh still uses transactional version checks.
- Existing `SaveConnection`, `GetConnection`, `UpdateCredentials`, `DeactivateConnection` method names and Firestore collection/field names are deliberately preserved so existing callers and stored data continue to work.
- Next step is to narrow the technical credential-store interface into separate read/write concerns without adding a second business model. This is distinct from removing the already retired DTO.

- The previous four-method global authorization repository interface is removed. The linking and token-refresh consumers now declare their own minimal 3-method and 2-method interfaces, respectively.

## CI acceptance for legacy code removal
- In addition to ordinary race tests, `make check` compiles `//go:build integration` tests (`-run '^$'`) so the Firestore migration contract cannot silently rot behind build tags. Actual Firestore emulator transaction validation remains an explicit separate environment gate.
- The Identity architecture test rejects reintroduction of the old application DTO, broad repository interface, or deleted duplicate mapper files.
