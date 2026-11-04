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
1. **Current commit:** introduce actual identity value objects, aggregate, policy, factory, repository contract and isolated tests. Do not rewire production OAuth prematurely.
2. Adapt existing Firestore account-link store through mappers while keeping its encryption, transactions and version checks.
3. Refactor OAuth LinkService to delegate link lifecycle to the aggregate; retain user-scoped OAuth client isolation.
4. Add business-shaped Calendar/Tasks/Forms ports as real business use cases emerge; keep audited raw operations isolated for administrative use.
5. Expand Conversation/Audience bounded contexts only after identifying invariants. Add architecture checks and retire legacy DTOs once all callers migrate.

## Validation
Run `go test ./internal/domain/...`, `go test ./tests/architecture/...`, `go test ./...`, `go vet ./...` and GitHub Actions using the repository toolchain. No Cloud Run deployment or data migration is authorized by this slice.
