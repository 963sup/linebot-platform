# Google Workspace OAuth 核心（離線實作，尚未開啟）

本次新增 `internal/infrastructure/google/api/oauth/flow.go` 作為 **Google Authorization Code + PKCE + OIDC** 的無狀態 SDK 輔助元件；Calendar 38、Tasks 14、Forms 10 共用一套 OAuth 設定。僅使用既有直接依賴 `golang.org/x/oauth2 v0.37.0` 與 `google.golang.org/api v0.310.0`，不自行實作 Token HTTP 交換。

## 已完成的技術元件

- 使用 `google.Endpoint`，固定 HTTPS Callback URL。
- `RandomState` 產生密碼學安全的 state/nonce；`AuthorizationURL` 使用 S256 PKCE、offline access、OIDC nonce 與有限 Scope allowlist。
- `ExchangeAndVerify` 使用 `Config.Exchange(..., oauth2.VerifierOption(...))`；取得 ID Token 後使用 `idtoken.Validate` 驗證 Google 簽章／Audience，額外檢查 Issuer、Subject、Expiry 與原始 nonce，僅在 Email 經驗證時提供 Email。
- 在離線測試使用 Fake Google Token Endpoint 與 Fake ID Token Verifier；不使用任何 Google 真實帳號、Key、Token 或服務。
- Token 與 provider 原始錯誤不記錄、不轉成 CLI 文字；受控流程應直接處理 Result，而不是輸出到 stdout。

## 尚未完成的外部接線及驗收（不可對公提供登入）

- [使用者自助綁定規格](user-self-service-linking.md) 來自 PR #26，保留了**連結可轉傳時須在瀏覽器再次驗證 LINE MINI App／LIFF 身分**的安全限制。單靠 Callback 的 state Cookie 無法證明原始 LINE 身分。
- Firestore 一次性 State、AES-GCM 加密 Token Repository、金鑰輪替版本、刷新 CAS／停用及 OAuth Start／Callback Handler 的**離線程式碼已於 PR #27 實作**；但尚未在既有正式 LINE MINI App、Firestore IAM、Secret Manager 金鑰及真實 Google 帳號上驗收。
- OAuth Start／Callback 尚未掛載公開路由，可信的 Channel／LINE user／Google `sub` 隔離與瀏覽器再驗證、憑證輪替、權限擴充與解除連結後停止背景同步仍待整合測試。
- 使用者個別 Calendar／Tasks／Forms 同步及 Pub/Sub Watch 通知不是本輪已完成的服務；詳見 [Staging Acceptance](staging-acceptance.md)。

## 最小流程

1. 外層驗證 LINE user → 建立 nonce/state/verifier 與按需 scopes。
2. 原子儲存一次性 state，向使用者提供 Google 同意 URL。
3. Callback 驗證 state 並原子消耗，拒絕未知、過期或重複狀態。
4. 使用本核心 Exchange 授權碼並驗證 ID Token 的 Google `sub`。
5. **先**完成憑證加密保存與同一使用者關聯交易，才標示已連結。
6. 每次建立 Google Calendar／Tasks／Forms Client 時，從受授權的 TokenSource 構造；以最低 Scope 運作。

警告：`oauth2/v2` 的 Tokeninfo／Userinfo 不是 OAuth Token 交換、Refresh Token 或身分登入的主要機制；不為這三個 API 額外建立大規模抽象。

來源：
- https://pkg.go.dev/golang.org/x/oauth2
- https://pkg.go.dev/golang.org/x/oauth2/google
- https://pkg.go.dev/google.golang.org/api/idtoken
- https://developers.google.com/identity/protocols/oauth2/web-server


## P2 implementation status (PR #27, offline only)

- Google OAuth flow / PKCE / nonce / ID Token signature+issuer+audience validation and requested-vs-granted scope verification: implemented and covered by offline unit tests.
- `internal/application/usecases/identity/link_service.go`: orchestrates Begin / one-time Complete / Disconnect; requires a **trusted** already-authenticated LINE owner.
- `internal/adapter/outbound/google/cloud/firestore/workspace_oauth.go`: transactional single-use State, encrypted connection repository, versioned token updates and disconnected tombstones.
- `internal/infrastructure/google/api/oauth/keyring.go`: AES-256-GCM with document-associated data and key version support. `internal/infrastructure/google/cloud/secretmanager/oauth_keys.go` loads pinned key versions from an injected authorized Secret Manager client.
- `internal/adapter/outbound/google/api/oauth/provider.go`: Google OAuth provider adapter, fixed revocation endpoint and persist-before-use TokenSource. Across replicas, stale CAS updates are rejected and latest persisted credentials are reread; **this is not proof of zero token-refresh races during a real Google outage**.
- `internal/adapter/inbound/http/google/oauth/handler.go`: requires injected trusted LINE principal resolver, HTTPS Google authorization endpoint, secure host-only CSRF Cookie and one-time callback state; currently **not registered** on the public webhook.
- `internal/bootstrap/workspace_links.go`: explicit opt-in DI using injected Firestore client, keyring, Google OAuth client credentials and owned HTTP transport; no credentials automatically read from Git or an unsafe public endpoint.
- Local Go CI checks are separate from `tests/integration/google_oauth_firestore_test.go` which requires `FIRESTORE_EMULATOR_HOST` and additional runtime verification.

### Prerequisites remaining before public use

1. Verify the LINE principal with Messaging API Account Linking or LINE MINI App **and reverify the browser's LIFF identity at handoff and callback**; a forwarded authorization link or matching state cookie alone cannot bind a Google account.
2. Provision dedicated GCP staging project, restricted Firestore permissions, TTL policy and pinned Secret Manager encryption versions; validate revocation and key rotation.
3. Demonstrate Firestore Emulator transaction races, secret-manager IAM and real Google OAuth consent/refresh/revoke on a separate test Google account. Define a reauthorization fallback for concurrent or revoked refresh tokens.
4. Bind the HTTPS start/callback routes to a reviewed identity gate; enforce session and consent policy, use staging OAuth client IDs, and record redacted audit and recovery evidence.
5. Forms Watches require actual Pub/Sub topic permissions and renewal schedule. Calendar syncToken 410 invalidation and Tasks updatedMin full scan need independent durable checkpoints.
6. Do not mark P3/P4 or production readiness complete merely because offline CI succeeds.
