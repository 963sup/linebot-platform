# Google Workspace P3 Staging Acceptance (not executed)

**Status:** not run. External Google account consent, Google Cloud resources, Pub/Sub, OAuth verification/production approval were not granted by this PR. The following are **acceptance gates**, not deployment commands.

## Source of user-self-service security contract

The requirements from [PR #26](https://github.com/963sup/linebot-platform/pull/26) are preserved in [User Self-Service Linking](user-self-service-linking.md). All work in PR #27 must comply with forwarded-link defense, independent LINE Login/LIFF verification, channel/user/Google subject isolation, and disabling sync on account unlink. Existing LINE or Firestore deployment does not prove these OAuth-specific boundaries.

## Authorized setup

1. Designate an isolated Google Cloud project, OAuth web application Client ID and approved HTTPS redirect URI. Enable Calendar API, Tasks API, Forms API and OAuth consent screen as required. Review Google verification/publishing requirements for sensitive scopes.
2. Securely provision OAuth Client Secret in Secret Manager, a **pinned, randomly generated 32-byte AES-256 key** in a separate versioned Secret, and a restricted Firestore database.
3. Grant least-privilege IAM only to the private OAuth service runtime. Do not give LINE's public webhook token-access authority; do not allow unverified LINE users to start OAuth.
4. Enable TTL on `google_oauth_states_v1.expires_at` and `google_oauth_connections_v1.expires_at`. Firestore TTL is asynchronous; application-level expiry and disconnect checks remain mandatory.
5. Wire a **verified LINE principal resolver** to the OAuth Start/Callback handler after review of the LINE Account Linking or LINE Login trust chain. The current code intentionally does not register HTTP routes.

## Acceptance matrix

| Gate | Expected staging evidence |
|---|---|
| OAuth identity | State binds to verified LINE Channel/user; spoofed LINE ID and unauthorized HTTP start rejected; browser LINE Login/LIFF reverified at handoff/callback |
| CSRF/PKCE | Missing/modified/replayed/forwarded authorization link or cookie rejected; expired session rejected; code verifier and nonce checked |
| Google identity | Wrong Google Client ID, issuer, nonce, expiry, or subject rejected; no email-as-identity shortcut |
| Scopes | Only approved Calendar / Tasks / Forms scopes are issued; missing granted API scope is rejected |
| Secure credentials | Token ciphertext contains no plaintext; changing document ID causes AES-GCM authentication error; no raw tokens in logs |
| Firestore transactions | At least 12 concurrent attempts to consume a state yield at most one success; stale credential update rejected |
| Rotation/revocation | Key version migration permits old reads/new writes; disconnect disables local access before remote revoke |
| OAuth refresh | Expired tokens refresh, persisted version advances, Refresh Token survives omitted new Refresh Token; ambiguous refresh is recoverable |
| Calendar / Tasks | Read and intentional write using consented staging user; quota/403/429/error classifications tested |
| Forms | All 10 methods offline; staging user verifies create/get/batchUpdate/publish/responses with explicit authorization |
| Negative paths | Inactive account or missing Secret Manager permissions fails closed; no redirected token leakage |

Record UTC time, isolated project ID (not secrets), approved OAuth client alias, GitHub image digest, Cloud Run IAM result, CI link, and supporting sanitized logs for each verified gate. Keep production disabled until independently approved.

## P4 follow-up: opt-in synchronization

- Calendar: checkpoint nextSyncToken only after all pages and durable application of changes; on HTTP 410 wipe/replace projection and perform full sync.
- Tasks: periodic paginated reconciliation using updatedMin and showDeleted/showHidden as appropriate; no push API or syncToken exists.
- Forms: Watches target Pub/Sub, not a direct callback URL; handle at-least-once notifications, throttling and seven-day renewals. Read changed form/responses via API after receiving the notification.
- Resource creation and background scheduling require separate authorization and staging validation.
