# Reviewer Profile: security

## Activate when

Webhook verification, OAuth, tokens, credentials, IAM, OIDC, Secret Manager, identity, logs, or external input change.

## Required checks

- LINE raw-body HMAC verification precedes event processing.
- OAuth state, nonce, PKCE, redirect binding, expiry, and replay handling are enforced.
- Codes and tokens never enter URLs, logs, errors, fixtures, or client code.
- User identity binding and per-user isolation are explicit.
- Cloud Tasks OIDC, service identity, IAM least privilege, and secret references are explicit.
- Inputs have size, timeout, SSRF, replay, and redaction controls.

## Commands

Run relevant security/OAuth/Webhook tests, `go vet ./...`, and `go run ./cmd/codexcheck`.

## Evidence and restrictions

Separate code controls from deployed IAM, secret rotation, OAuth consent, and network policy. Never read secrets, create credentials, login OAuth, grant IAM, weaken TLS, or deploy. External controls not directly observed are `Not Verified`.
